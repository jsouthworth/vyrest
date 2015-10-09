package vyrest

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

type MessageResp struct {
	Message string `json:"message"`
}

type MessageErrorResp struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}

type GetProcessResp struct {
	Processes []*Process `json:"process"`
}

type GetSessionResp struct {
	Message  string     `json:"message"`
	Sessions []*Session `json:"session"`
}

type Session struct {
	client      *Client
	Id          string `json:"id"`
	Username    string `json:"username"`
	Started     string `json:"started"`
	Modified    string `json:"modified"`
	Updated     string `json:"updated"`
	Description string `json:"description"`
}

func (s *Session) getPath(path string) string {
	return "/rest/conf/" + s.Id + path
}

func (s *Session) Set(path []string) error {
	_, err := s.client.doRequest("PUT", s.getPath("/set"+pathToString(path)))
	return err
}

func (s *Session) Delete(path []string) error {
	_, err := s.client.doRequest("PUT", s.getPath("/delete"+pathToString(path)))
	return err
}

func (s *Session) Get(path []string) (*ConfResp, error) {
	var resp *ConfResp
	res, err := s.client.doRequest("GET", s.getPath(pathToString(path)))
	if err != nil {
		return nil, err
	}
	err = s.client.decodeInto(res, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Session) Commit() error {
	_, err := s.client.doMsgRequest("POST", s.getPath("/commit"))
	return err
}

func (s *Session) Save() error {
	_, err := s.client.doMsgRequest("POST", s.getPath("/save"))
	return err
}

func (s *Session) Load() error {
	_, err := s.client.doMsgRequest("POST", s.getPath("/load"))
	return err
}

func (s *Session) Discard() error {
	_, err := s.client.doMsgRequest("POST", s.getPath("/discard"))
	return err
}

func (s *Session) Show() (string, error) {
	return s.client.doMsgRequest("POST", s.getPath("/show"))
}

func (s *Session) Teardown() error {
	//TODO, this should use MessageErrorResp
	_, err := s.client.doRequest("DELETE", s.getPath(""))
	return err
}

type Child struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

type ValHelp struct {
	Type string `json:"type"`
	Vals string `json:"vals"`
	Help string `json:"help"`
}

type ConfResp struct {
	Name        string    `json:"name"`
	State       string    `json:"state"`
	Type        []string  `json:"type"`
	Enumeration []string  `json:"enumeration"`
	End         string    `json:"end"`
	Mandatory   string    `json:"mandatory"`
	Multi       string    `json:"multi"`
	Default     string    `json:"default"`
	Help        string    `json:"help"`
	ValHelp     []ValHelp `json:"val_help"`
	CompHelp    string    `json:"comp_help"`
	Children    []*Child  `json:"children"`
}

type OpResp struct {
	Children []string `json:"children"`
	Enum     []string `json:"enum"`
	Action   string   `json:"action"`
	Help     string   `json:"help"`
}

type Process struct {
	client   *Client
	Username string `json:"username"`
	Started  string `json:"started"`
	Updated  string `json:"updated"`
	Id       string `json:"id"`
	Command  string `json:"command"`
}

func (p *Process) getPath() string {
	return "/rest/op/" + p.Id
}

func (p *Process) getOutput(w io.Writer) error {
	res, err := p.client.doRequest("GET", p.getPath())
	if err != nil {
		return err
	}
	io.Copy(w, res.Body)
	res.Body.Close()
	for res.StatusCode != 410 {
		res, err = p.client.doRequest("GET", p.getPath())
		if err != nil {
			if res != nil && res.StatusCode == 410 {
				break
			}
			return err
		}
		if res.StatusCode == 410 {
			res.Body.Close()
			break
		}
		if _, err := io.Copy(w, res.Body); err != nil {
			res.Body.Close()
			break
		}
		res.Body.Close()
	}
	return nil
}

func (p *Process) StreamOutput(w io.Writer) error {
	return p.getOutput(w)
}

func (p *Process) Output() (string, error) {
	buf := new(bytes.Buffer)
	err := p.getOutput(buf)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (p *Process) Pid() string {
	return p.Id
}

func (p *Process) Kill() error {
	//TODO: should use MessageErrorResp
	_, err := p.client.doRequest("DELETE", p.getPath())
	return err
}

type Client struct {
	*http.Client
	host string
	auth string
	user string
}

func pathToString(path []string) string {
	var str string
	for _, v := range path {
		str += "/" + strings.Replace(url.QueryEscape(v), "+", "%20", -1)
	}
	return str
}

func Dial(host, user, pass string) *Client {
	c := new(Client)
	c.Client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	c.host = host
	c.auth = base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
	c.user = user
	return c
}

func (c *Client) genRequest(method, path string) *http.Request {
	var req *http.Request
	req, _ = http.NewRequest(strings.ToUpper(method), "https://"+c.host+path, nil)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Vyatta-Specification-Version", "0.1")
	req.Header.Add("Authorization", "Basic "+c.auth)
	return req
}

func (c *Client) doRequest(method, path string) (*http.Response, error) {
	resp, err := c.Do(c.genRequest(method, path))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode > 399 {
		return resp, errors.New(resp.Status)
	}
	return resp, nil
}

func (c *Client) doMsgRequest(method, path string) (string, error) {
	var msg MessageResp
	resp, err := c.Do(c.genRequest(method, path))
	if err != nil {
		return "", err
	}
	err = c.decodeInto(resp, &msg)
	if err != nil {
		return "", err
	}
	res := msg.Message
	if resp.StatusCode > 399 {
		if res == "" {
			return "", errors.New(resp.Status)
		}
		return "", errors.New(res)
	}
	return res, nil
}

func (c *Client) decodeInto(res *http.Response, v interface{}) error {
	dec := json.NewDecoder(res.Body)
	err := dec.Decode(v)
	res.Body.Close()
	if err != nil && err != io.EOF {
		return err
	}
	return nil
}

func (c *Client) SetupSession() (*Session, error) {
	res, err := c.doRequest("POST", "/rest/conf")
	if err != nil {
		return nil, err
	}
	sid := path.Base(res.Header.Get("Location"))
	return c.GetSession(sid)
}

func (c *Client) SessionExists(sid string) (bool, error) {
	sessions, err := c.ListSessions()
	if err != nil {
		return false, err
	}
	for _, session := range sessions {
		if session.Id == sid {
			return true, nil
		}
	}
	return false, nil
}

func (c *Client) ListSessions() ([]*Session, error) {
	var resp GetSessionResp
	res, err := c.doRequest("GET", "/rest/conf")
	if err != nil {
		return nil, err
	}
	err = c.decodeInto(res, &resp)
	if err != nil {
		return nil, err
	}
	for _, session := range resp.Sessions {
		session.client = c
	}
	return resp.Sessions, nil
}

func (c *Client) GetSession(sid string) (*Session, error) {
	sessions, err := c.ListSessions()
	if err != nil {
		return nil, err
	}
	for _, session := range sessions {
		if session.Id == sid {
			return session, nil
		}
	}
	return nil, errors.New("session does not exist")
}

func (c *Client) TeardownAllSessions() error {
	sessions, err := c.ListSessions()
	if err != nil {
		return err
	}
	for _, session := range sessions {
		if c.user != session.Username {
			continue
		}
		err := session.Teardown()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) GetOperational(path []string) (*OpResp, error) {
	var resp *OpResp
	res, err := c.doRequest("GET", "/rest/op/"+pathToString(path))
	if err != nil {
		return nil, err
	}
	err = c.decodeInto(res, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) StartProcess(p []string) (*Process, error) {
	res, err := c.doRequest("POST", "/rest/op"+pathToString(p))
	if err != nil {
		return nil, err
	}
	pid := path.Base(res.Header.Get("Location"))
	return c.GetProcess(pid)
}

func (c *Client) ListProcesses() ([]*Process, error) {
	var resp GetProcessResp
	res, err := c.doRequest("GET", "/rest/op")
	if err != nil {
		return nil, err
	}
	err = c.decodeInto(res, &resp)
	if err != nil {
		return nil, err
	}
	for _, process := range resp.Processes {
		process.client = c
	}
	return resp.Processes, nil
}

func (c *Client) GetProcess(pid string) (*Process, error) {
	processes, err := c.ListProcesses()
	if err != nil {
		return nil, err
	}
	for _, process := range processes {
		if process.Id == pid {
			return process, nil
		}
	}
	return nil, errors.New("process not found")
}

func (c *Client) KillProcesses() error {
	procs, err := c.ListProcesses()
	if err != nil {
		return err
	}
	for _, proc := range procs {
		if c.user != proc.Username {
			continue
		}
		err := proc.Kill()
		if err != nil {
			return err
		}
	}
	return nil
}
