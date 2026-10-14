package app

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bhupenderbhardwaj/bhuppi/internal/aireview"
	"github.com/bhupenderbhardwaj/bhuppi/internal/dispatch"
	"github.com/bhupenderbhardwaj/bhuppi/internal/execx"
	"github.com/bhupenderbhardwaj/bhuppi/internal/netname"
	"github.com/bhupenderbhardwaj/bhuppi/internal/procview"
	"github.com/bhupenderbhardwaj/bhuppi/internal/reputation"
)

func renderProc(plan dispatch.Plan, stdout, stderr io.Writer, color bool) int {
	switch plan.Draw {
	case "pview":
		return renderView(plan, stdout, stderr, color)
	case "pagents":
		return renderAgents(plan, stdout, stderr, color)
	case "procs":
		return renderProcList(plan, stdout, stderr, color)
	case "conns":
		return renderConns(plan, stdout, stderr, color)
	default:
		fmt.Fprintf(stderr, "unhandled process page\n")
		return 2
	}
}

func renderProcList(plan dispatch.Plan, stdout, stderr io.Writer, color bool) int {
	if len(plan.Execs) == 0 {
		fmt.Fprintf(stderr, "proc list: empty argv\n")
		return 2
	}
	text, _, err := capture(plan.Execs[0])
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	table := aireview.ParseSnaps(text)
	shown := table
	if plan.Query != "" {
		shown = nil
		for _, s := range table {
			if strings.Contains(s.Command, plan.Query) {
				shown = append(shown, s)
			}
		}
	}
	if len(shown) == 0 {
		fmt.Fprintf(stderr, "no matches\n")
		return 2
	}
	pids := make([]string, len(shown))
	for i, s := range shown {
		pids[i] = s.PID
	}
	emit(stdout, aireview.FormatList(table, shown, cwdFiles(pids), color))
	return 0
}

func cwdFiles(pids []string) map[string]aireview.Files {
	files := map[string]aireview.Files{}
	const batch = 40
	for i := 0; i < len(pids); i += batch {
		j := i + batch
		if j > len(pids) {
			j = len(pids)
		}
		argv := []string{"lsof", "-n", "-P", "-a", "-p", strings.Join(pids[i:j], ","), "-d", "cwd"}
		text, _ := runQuiet(argv)
		for _, o := range aireview.ParseOpen(text) {
			if o.FD != "cwd" {
				continue
			}
			f := files[o.PID]
			if f.Cwd != "" {
				continue
			}
			if p := procview.PathOf(o.Name); p != "" {
				f.Cwd = p
			} else {
				f.Cwd = strings.TrimSpace(o.Name)
			}
			files[o.PID] = f
		}
	}
	return files
}

func renderConns(plan dispatch.Plan, stdout, stderr io.Writer, color bool) int {
	if plan.Query == "reputation-check" {
		return renderReputation(plan, stdout, stderr, color)
	}
	if len(plan.Execs) == 0 {
		fmt.Fprintf(stderr, "net connections: empty argv\n")
		return 2
	}
	text, _, err := capture(plan.Execs[0])
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	names := netname.Resolve(reputation.Public(connHosts(text)))
	emit(stdout, aireview.FormatLsofNamed(text, names, color))
	return 0
}

func renderReputation(plan dispatch.Plan, stdout, stderr io.Writer, color bool) int {
	keys, err := reputation.Load()
	if err != nil {
		fmt.Fprintf(stderr, "reputation: %v\n", err)
		return 1
	}
	if !keys.HasVT() && !keys.HasAbuse() {
		fmt.Fprint(stderr, reputation.Missing())
		return 2
	}
	if len(plan.Execs) == 0 {
		fmt.Fprintf(stderr, "net connections: empty argv\n")
		return 2
	}
	text, _, err := capture(plan.Execs[0])
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	ips := reputation.Public(connHosts(text))
	names := netname.Resolve(ips)
	var b strings.Builder
	b.WriteString(reputation.Preface(keys))
	if len(ips) == 0 {
		b.WriteString("no public addresses\n")
		emit(stdout, b.String())
		return 0
	}
	b.WriteString(reputation.Format(reputation.Check(ips, names, keys), color))
	emit(stdout, b.String())
	return 0
}

func connHosts(text string) []string {
	var hosts []string
	for _, o := range aireview.ParseOpen(text) {
		s := aireview.ParseSock(o.Name)
		if s.LocalHost != "" {
			hosts = append(hosts, s.LocalHost)
		}
		if s.RemoteHost != "" {
			hosts = append(hosts, s.RemoteHost)
		}
	}
	return hosts
}

func renderView(plan dispatch.Plan, stdout, stderr io.Writer, color bool) int {
	if len(plan.Execs) == 0 {
		fmt.Fprintf(stderr, "proc view: empty argv\n")
		return 2
	}
	if filepath.Base(plan.Execs[0][0]) == "pgrep" {
		text, _, err := capture(plan.Execs[0])
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			return 1
		}
		rows := procview.ParsePgrep(text)
		if len(rows) == 0 {
			fmt.Fprintf(stderr, "no matches\n")
			return 2
		}
		if len(rows) > 1 {
			emit(stdout, procview.FormatChoice(plan.Query, rows, "run bhuppi proc view PID for one of them"))
			return 0
		}
		steps, err := dispatch.Snapshot("pview", []string{rows[0].PID}, runtime.GOOS)
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			return 2
		}
		return emitPage(steps, false, "pid "+rows[0].PID, stdout, stderr, color)
	}
	return emitPage(plan.Execs, false, "pid "+plan.Query, stdout, stderr, color)
}

func renderAgents(plan dispatch.Plan, stdout, stderr io.Writer, color bool) int {
	if isPID(plan.Query) {
		return emitPage(plan.Execs, true, "pid "+plan.Query, stdout, stderr, color)
	}
	if len(plan.Execs) == 0 {
		fmt.Fprintf(stderr, "proc agents: empty argv\n")
		return 2
	}
	text, _, err := capture(plan.Execs[0])
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	rows := procview.ParseList(text)
	if plan.Query == "" {
		emit(stdout, procview.FormatAgents(procview.MatchAgents(rows), color))
		return 0
	}
	matched := procview.MatchName(rows, plan.Query)
	if len(matched) == 0 {
		fmt.Fprintf(stderr, "no matches\n")
		return 2
	}
	if len(matched) > procview.MaxBlocks {
		emit(stdout, procview.FormatChoice(plan.Query, matched, "pass one pid: bhuppi proc agents PID"))
		return 0
	}
	var b strings.Builder
	for i, r := range matched {
		steps, err := dispatch.Snapshot("pagents", []string{r.PID}, runtime.GOOS)
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			return 2
		}
		if i > 0 {
			b.WriteByte('\n')
		}
		page, ok := collect(steps, true, "pid "+r.PID+" — "+plan.Query, stderr)
		if !ok {
			fmt.Fprintf(stderr, "no such process %s\n", r.PID)
		}
		b.WriteString(procview.Format(page, color))
	}
	emit(stdout, b.String())
	return 0
}

func emitPage(steps [][]string, agents bool, heading string, stdout, stderr io.Writer, color bool) int {
	page, ok := collect(steps, agents, heading, stderr)
	emit(stdout, procview.Format(page, color))
	if !ok {
		fmt.Fprintf(stderr, "no such process\n")
		return 2
	}
	return 0
}

func collect(steps [][]string, agents bool, heading string, stderr io.Writer) (procview.Page, bool) {
	out := map[string]string{}
	for _, st := range steps {
		argv, err := absolute(st)
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			continue
		}
		text, _, err := capture(argv)
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			continue
		}
		out[stepKey(st)] = text
	}
	page := pageFrom(out, agents)
	page.Heading = heading
	return page, page.Detail.OK
}

func pageFrom(out map[string]string, agents bool) procview.Page {
	d := procview.ParseDetail(out["detail"])
	table := procview.ParseTable(out["table"])
	program := firstPath(procview.LsofNames(out["txt"], "txt"))
	cwd := firstPath(procview.LsofNames(out["cwd"], "cwd"))
	var files []string
	for _, n := range procview.LsofNames(out["files"], "") {
		if p := procview.PathOf(n); p != "" {
			files = append(files, p)
		}
	}
	return procview.Page{
		Detail:   d,
		Program:  program,
		Cwd:      cwd,
		Chain:    procview.Ancestors(table, d.PID),
		Kids:     procview.Children(table, d.PID),
		Conns:    procview.LsofNames(out["net"], ""),
		Files:    files,
		Outbound: procview.Outbound(out["net"]),
		Agents:   agents,
	}
}

func firstPath(names []string) string {
	if len(names) == 0 {
		return ""
	}
	if p := procview.PathOf(names[0]); p != "" {
		return p
	}
	return strings.TrimSpace(names[0])
}

func stepKey(st []string) string {
	base := filepath.Base(st[0])
	joined := strings.Join(st, " ")
	switch base {
	case "pgrep":
		return "pgrep"
	case "ps":
		if strings.Contains(joined, "ppid=") {
			return "table"
		}
		if containsArg(st, "-p") {
			return "detail"
		}
		return "list"
	case "lsof":
		if containsArg(st, "txt") {
			return "txt"
		}
		if containsArg(st, "cwd") {
			return "cwd"
		}
		if containsArg(st, "-i") {
			return "net"
		}
		return "files"
	default:
		return base
	}
}

func containsArg(st []string, want string) bool {
	for _, a := range st {
		if a == want {
			return true
		}
	}
	return false
}

func absolute(argv []string) ([]string, error) {
	if len(argv) == 0 {
		return nil, fmt.Errorf("empty argv")
	}
	if filepath.IsAbs(argv[0]) {
		return argv, nil
	}
	p, err := exec.LookPath(argv[0])
	if err != nil {
		return nil, fmt.Errorf("%s: not found", argv[0])
	}
	out := append([]string{p}, argv[1:]...)
	return out, nil
}

func capture(argv []string) (string, int, error) {
	var buf bytes.Buffer
	code, err := execx.RunCaptured(argv, true, &buf, io.Discard)
	return buf.String(), code, err
}

func isPID(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
