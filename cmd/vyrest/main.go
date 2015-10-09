package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/jsouthworth/vyrest"
	"os"
	"sort"
	"text/tabwriter"
)

var host, user, pass, sid string

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <command> <args> \n", os.Args[0])
		flag.PrintDefaults()
		usage()
	}
	flag.StringVar(&host, "host", "", "Hostname")
	flag.StringVar(&user, "user", "", "Username")
	flag.StringVar(&pass, "pass", "", "Password")
	flag.StringVar(&sid, "sid", "", "Session-id to which to connect")
}

func handleError(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func setupSession(c *vyrest.Client, args ...string) {
	session, err := c.SetupSession()
	handleError(err)
	fmt.Println(session.Id)
}

func listSessions(c *vyrest.Client, args ...string) {
	sessions, err := c.ListSessions()
	handleError(err)
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)
	fmt.Fprintf(w, "session-id\tusername\tdescription\n")
	fmt.Fprintf(w, "----------\t--------\t-----------\n")
	for _, session := range sessions {
		fmt.Fprintf(w, "%s\t%s\t%s\n", session.Id, session.Username, session.Description)
	}
	w.Flush()
}

func teardownSession(c *vyrest.Client, args ...string) {
	if len(args) == 0 {
		handleError(errors.New("must supply sid to teardown"))
	}
	sid := args[0]
	session, err := c.GetSession(sid)
	handleError(err)
	handleError(session.Teardown())
}

func teardownSessions(c *vyrest.Client, args ...string) {
	handleError(c.TeardownAllSessions())
}

func sessionExists(c *vyrest.Client, args ...string) {
	if sid == "" {
		handleError(errors.New("must supply sid"))
	}
	exists, err := c.SessionExists(sid)
	handleError(err)
	fmt.Println(exists)
}

func getSession(c *vyrest.Client) *vyrest.Session {
	session, err := c.GetSession(sid)
	handleError(err)
	return session
}

func set(c *vyrest.Client, args ...string) {
	handleError(getSession(c).Set(args))
}

func del(c *vyrest.Client, args ...string) {
	handleError(getSession(c).Delete(args))
}

func get(c *vyrest.Client, args ...string) {
	resp, err := getSession(c).Get(args)
	handleError(err)
	b, err := json.MarshalIndent(resp, "", "    ")
	handleError(err)
	fmt.Println(string(b))
}

func commit(c *vyrest.Client, args ...string) {
	handleError(getSession(c).Commit())
}

func save(c *vyrest.Client, args ...string) {
	handleError(getSession(c).Save())
}

func load(c *vyrest.Client, args ...string) {
	handleError(getSession(c).Load())
}

func discard(c *vyrest.Client, args ...string) {
	handleError(getSession(c).Discard())
}

func show(c *vyrest.Client, args ...string) {
	out, err := getSession(c).Show()
	handleError(err)
	fmt.Println(out)
}

func getOp(c *vyrest.Client, args ...string) {
	resp, err := c.GetOperational(args)
	handleError(err)
	fmt.Println(resp.Children)
}

func startCmd(c *vyrest.Client, args ...string) {
	cmd, err := c.StartProcess(args)
	handleError(err)
	fmt.Println(cmd.Pid())
}

func runCmd(c *vyrest.Client, args ...string) {
	cmd, err := c.StartProcess(args)
	handleError(err)
	err = cmd.StreamOutput(os.Stdout)
	handleError(err)
}

func getOutput(c *vyrest.Client, args ...string) {
	pid := args[0]
	proc, err := c.GetProcess(pid)
	handleError(err)
	out, err := proc.Output()
	handleError(err)
	fmt.Println(out)
}

func killProcess(c *vyrest.Client, args ...string) {
	pid := args[0]
	proc, err := c.GetProcess(pid)
	handleError(err)
	handleError(proc.Kill())
}

func killProcesses(c *vyrest.Client, args ...string) {
	handleError(c.KillProcesses())
}

func listProcesses(c *vyrest.Client, args ...string) {
	procs, err := c.ListProcesses()
	handleError(err)
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)
	fmt.Fprintf(w, "process-id\tusername\tcommand\n")
	fmt.Fprintf(w, "----------\t--------\t-------\n")
	for _, proc := range procs {
		fmt.Fprintf(w, "%s\t%s\t%s\n", proc.Id, proc.Username, proc.Command)
	}
	w.Flush()
}

type cmd struct {
	fn    func(*vyrest.Client, ...string)
	info  string
	nargs int
}

var cmds = map[string]*cmd{
	"setup-session":     &cmd{setupSession, "Setup a new session", 0},
	"list-sessions":     &cmd{listSessions, "List all sessions", 0},
	"teardown-session":  &cmd{teardownSession, "Teardown a session", 0},
	"teardown-sessions": &cmd{teardownSessions, "Teardown all sessions", 0},
	"session-exists":    &cmd{sessionExists, "Check if a session exists", 0},
	"set":               &cmd{set, "Create a path in the configuration hierarchy", -1},
	"delete":            &cmd{del, "Delete a path from the configuation hierarchy", -1},
	"get":               &cmd{get, "Get children of the path", -1},
	"commit":            &cmd{commit, "Commit", 0},
	"save":              &cmd{save, "Save to the bootup configuration", 0},
	"load":              &cmd{load, "Load configuration from bootup configuation", 0},
	"discard":           &cmd{discard, "Discard configuration changes", 0},
	"show":              &cmd{show, "Show candidate configuration", 0},
	"get-op":            &cmd{getOp, "Get children of operational path", -1},
	"start-cmd":         &cmd{startCmd, "Start an operational command", -1},
	"get-output":        &cmd{getOutput, "Get output from a previously started operational command", 1},
	"kill-process":      &cmd{killProcess, "Kill an operational command", 1},
	"kill-processes":    &cmd{killProcesses, "Kill all currently running operational commands for your user", 0},
	"list-processes":    &cmd{listProcesses, "List all running operational commands", 0},
	"run-cmd":           &cmd{runCmd, "Start and retrieve output from an operational mode command", -1},
}

func usage() {
	w := tabwriter.NewWriter(os.Stderr, 0, 8, 2, '\t', 0)
	fmt.Fprintln(w, "Availble commands:")
	cmdnames := make([]string, 0, len(cmds))
	for name, _ := range cmds {
		cmdnames = append(cmdnames, name)
	}
	sort.Sort(sort.StringSlice(cmdnames))
	for _, name := range cmdnames {
		fmt.Fprintf(w, "  %s\t%s\n", name, cmds[name].info)
	}
	w.Flush()
}

func main() {
	flag.Parse()
	args := flag.Args()
	c := vyrest.Dial(host, user, pass)
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Must supply command")
		flag.Usage()
		os.Exit(1)
	}
	cmdin := args[0]
	cmd, ok := cmds[cmdin]
	if !ok {
		fmt.Fprintln(os.Stderr, "Invalid command")
		flag.Usage()
		os.Exit(1)
	}
	if len(args)-1 < cmd.nargs {
		fmt.Fprintln(os.Stderr, "Invalid number of arguements to", cmdin, "needs", cmd.nargs)
		os.Exit(1)
	}
	cmd.fn(c, args[1:]...)
}
