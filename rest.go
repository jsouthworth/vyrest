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
	Id          string `json:"id"`
	Username    string `json:"username"`
	Started     string `json:"started"`
	Modified    string `json:"modified"`
	Updated     string `json:"updated"`
	Description string `json:"description"`
}

type Child struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

type ConfResp struct {
	Name        string   `json:"name"`
	State       string   `json:"state"`
	Type        []string `json:"type"`
	Enumeration []string `json:"enumeration"`
	End         string   `json:"end"`
	Mandatory   string   `json:"mandatory"`
	Multi       string   `json:"multi"`
	Default     string   `json:"default"`
	Help        string   `json:"help"`
	ValHelp     []string `json:"val_help"`
	CompHelp    string   `json:"comp_help"`
	Children    []*Child `json:"children"`
}

type OpResp struct {
	Children []string `json:"children"`
	Enum     []string `json:"enum"`
	Action   string   `json:"action"`
	Help     string   `json:"help"`
}

type Process struct {
	Username string `json:"username"`
	Started  string `json:"started"`
	Updated  string `json:"updated"`
	Id       string `json:"id"`
	Command  string `json:"command"`
}

type Command struct {
	pid    string
	client *Client
}

func (c *Command) Output() (string, error) {
	buf := new(bytes.Buffer)
	res, err := c.client.doRequest("GET", "/rest/op/"+c.pid)
	if err != nil {
		return "", err
	}
	io.Copy(buf, res.Body)
	res.Body.Close()
	for res.StatusCode != 410 {
		res, err = c.client.doRequest("GET", "/rest/op/"+c.pid)
		if err != nil {
			if res != nil && res.StatusCode == 410 {
				break
			}
			return "", err
		}
		if res.StatusCode == 410 {
			res.Body.Close()
			break
		}
		if _, err := io.Copy(buf, res.Body); err != nil {
			res.Body.Close()
			break
		}
		res.Body.Close()
	}
	return buf.String(), nil
}

func (c *Command) Pid() string {
	return c.pid
}

func (c *Command) Kill() error {
	//TODO: should use MessageErrorResp
	_, err := c.client.doRequest("DELETE", "/rest/op/"+c.pid)
	return err
}

type Client struct {
	*http.Client
	host string
	auth string
	user string
	sid  string
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

func (c *Client) SetupSession() (string, error) {
	res, err := c.doRequest("POST", "/rest/conf")
	if err != nil {
		return "", err
	}
	sid := path.Base(res.Header.Get("Location"))
	c.sid = sid
	return sid, nil
}

func (c *Client) TeardownSession() error {
	return c.TeardownSid(c.sid)
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

func (c *Client) ConnectSession(sid string) error {
	exists, err := c.SessionExists(sid)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("session does not exist")
	}
	c.sid = sid
	return nil
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
	return resp.Sessions, nil
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
		err := c.TeardownSid(session.Id)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) TeardownSid(sid string) error {
	//TODO, this should use MessageErrorResp
	_, err := c.doRequest("DELETE", "/rest/conf/"+sid)
	return err
}

func (c *Client) Set(path []string) error {
	_, err := c.doRequest("PUT", "/rest/conf/"+c.sid+"/set"+pathToString(path))
	return err
}

func (c *Client) Delete(path []string) error {
	_, err := c.doRequest("PUT", "/rest/conf/"+c.sid+"/delete"+pathToString(path))
	return err
}

func (c *Client) Get(path []string) (*ConfResp, error) {
	var resp *ConfResp
	res, err := c.doRequest("GET", "/rest/conf/"+c.sid+pathToString(path))
	if err != nil {
		return nil, err
	}
	err = c.decodeInto(res, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) Commit() error {
	_, err := c.doMsgRequest("POST", "/rest/conf/"+c.sid+"/commit")
	return err
}

func (c *Client) Save() error {
	_, err := c.doMsgRequest("POST", "/rest/conf/"+c.sid+"/save")
	return err
}

func (c *Client) Load() error {
	_, err := c.doMsgRequest("POST", "/rest/conf/"+c.sid+"/load")
	return err
}

func (c *Client) Discard() error {
	_, err := c.doMsgRequest("POST", "/rest/conf/"+c.sid+"/discard")
	return err
}

func (c *Client) Show() (string, error) {
	return c.doMsgRequest("POST", "/rest/conf/"+c.sid+"/show")
}

func (c *Client) GetOperational(path []string) (*OpResp, error) {
	var resp *OpResp
	res, err := c.doRequest("GET", "/rest/op/"+c.sid+pathToString(path))
	if err != nil {
		return nil, err
	}
	err = c.decodeInto(res, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) StartOperationalCmd(p []string) (*Command, error) {
	res, err := c.doRequest("POST", "/rest/op"+pathToString(p))
	if err != nil {
		return nil, err
	}
	return &Command{
		pid:    path.Base(res.Header.Get("Location")),
		client: c,
	}, nil
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
	return resp.Processes, nil
}

func (c *Client) ProcessToCommand(proc *Process) *Command {
	return &Command{
		pid:    proc.Id,
		client: c,
	}
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
		cmd := c.ProcessToCommand(proc)
		err := cmd.Kill()
		if err != nil {
			return err
		}
	}
	return nil
}
