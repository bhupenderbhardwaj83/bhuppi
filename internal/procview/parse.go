package procview

import (
	"os"
	"strconv"
	"strings"
)

// Row is one process from ps.
type Row struct {
	PID, PPID, User, Command string
}

// Detail is the single-process ps line.
type Detail struct {
	PID, User, State, Nice, Priority, CPU, Mem, Elapsed, Command string
	OK                                                           bool
}

// End is one outbound remote.
type End struct {
	Host, Port string
}

// ParseDetail reads `ps -p PID -o pid,user,uid,ni,pri,stat,%cpu,%mem,etime,command`.
func ParseDetail(text string) Detail {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 9 {
			continue
		}
		if _, err := strconv.Atoi(f[0]); err != nil {
			continue
		}
		return Detail{
			PID: f[0], User: f[1], Nice: f[3], Priority: f[4], State: f[5],
			CPU: f[6], Mem: f[7], Elapsed: f[8],
			Command: strings.Join(f[9:], " "),
			OK:      true,
		}
	}
	return Detail{}
}

// ParseTable reads `ps -axo pid=,ppid=,user=,command=`.
func ParseTable(text string) []Row {
	var out []Row
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		if _, err := strconv.Atoi(f[0]); err != nil {
			continue
		}
		if _, err := strconv.Atoi(f[1]); err != nil {
			continue
		}
		out = append(out, Row{PID: f[0], PPID: f[1], User: f[2], Command: strings.Join(f[3:], " ")})
	}
	return out
}

// ParseList reads `ps -axo pid=,user=,command=`.
func ParseList(text string) []Row {
	var out []Row
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		if _, err := strconv.Atoi(f[0]); err != nil {
			continue
		}
		out = append(out, Row{PID: f[0], User: f[1], Command: strings.Join(f[2:], " ")})
	}
	return out
}

// ParsePgrep reads `pgrep -lf` lines: pid, then the command.
func ParsePgrep(text string) []Row {
	var out []Row
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		if _, err := strconv.Atoi(f[0]); err != nil {
			continue
		}
		out = append(out, Row{PID: f[0], Command: strings.Join(f[1:], " ")})
	}
	return out
}

// Ancestors walks from pid up to pid 1. The first row is the top of the chain.
func Ancestors(rows []Row, pid string) []Row {
	by := map[string]Row{}
	for _, r := range rows {
		by[r.PID] = r
	}
	var up []Row
	seen := map[string]bool{}
	cur := pid
	for cur != "" && !seen[cur] {
		r, ok := by[cur]
		if !ok {
			break
		}
		seen[cur] = true
		up = append(up, r)
		if r.PID == "1" || r.PPID == "" || r.PPID == "0" || r.PPID == r.PID {
			break
		}
		cur = r.PPID
	}
	for i, j := 0, len(up)-1; i < j; i, j = i+1, j-1 {
		up[i], up[j] = up[j], up[i]
	}
	return up
}

// Children returns direct children of pid.
func Children(rows []Row, pid string) []Row {
	var out []Row
	for _, r := range rows {
		if r.PPID == pid && r.PID != pid {
			out = append(out, r)
		}
	}
	return out
}

// LsofNames returns the NAME column. fd limits the rows to that FD (txt, cwd) when set.
func LsofNames(text, fd string) []string {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return nil
	}
	nameAt := strings.Index(lines[0], "NAME")
	fdAt := strings.Index(lines[0], "FD")
	var out []string
	for _, ln := range lines[1:] {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		if fd != "" {
			got := fdField(ln, fdAt)
			if got != fd {
				continue
			}
		}
		name := strings.TrimSpace(ln)
		if nameAt >= 0 && len(ln) > nameAt {
			name = strings.TrimSpace(ln[nameAt:])
		}
		if name != "" && name != "NAME" {
			out = append(out, name)
		}
	}
	return out
}

func fdField(line string, fdAt int) string {
	rest := line
	if fdAt >= 0 && len(line) > fdAt {
		rest = line[fdAt:]
	}
	f := strings.Fields(rest)
	if len(f) == 0 {
		return ""
	}
	return f[0]
}

// PathOf returns a filesystem path from an lsof NAME, without a trailing note.
func PathOf(name string) string {
	name = strings.TrimSpace(name)
	for _, suf := range []string{" (deleted)", " (TEXT)", " (DF)"} {
		name = strings.TrimSpace(strings.TrimSuffix(name, suf))
	}
	if strings.HasPrefix(name, "/") {
		return name
	}
	return ""
}

// Outbound reads lsof -i names and keeps remotes after "->".
func Outbound(text string) []End {
	var out []End
	seen := map[string]bool{}
	for _, name := range LsofNames(text, "") {
		i := strings.Index(name, "->")
		if i < 0 {
			continue
		}
		host, port, ok := splitHostPort(name[i+2:])
		if !ok {
			continue
		}
		key := host + ":" + port
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, End{Host: host, Port: port})
	}
	return out
}

func splitHostPort(s string) (host, port string, ok bool) {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, ' '); i >= 0 {
		s = s[:i]
	}
	if strings.HasPrefix(s, "[") {
		end := strings.Index(s, "]")
		if end < 0 {
			return "", "", false
		}
		host = s[1:end]
		rest := s[end+1:]
		if !strings.HasPrefix(rest, ":") || len(rest) < 2 {
			return "", "", false
		}
		return host, rest[1:], true
	}
	i := strings.LastIndex(s, ":")
	if i <= 0 || i == len(s)-1 {
		return "", "", false
	}
	return s[:i], s[i+1:], true
}

// ProgramPath is the executable from a command line. A space inside the path is
// not an argument. The longest prefix that exists on disk wins, so a directory
// that is only the start of a longer path does not cut the name short.
func ProgramPath(command string) string {
	f := strings.Fields(command)
	if len(f) == 0 {
		return ""
	}
	if !strings.HasPrefix(f[0], "/") {
		return f[0]
	}
	best := ""
	path := ""
	for i := 0; i < len(f); i++ {
		if i > 0 && strings.HasPrefix(f[i], "-") {
			break
		}
		if path == "" {
			path = f[i]
		} else {
			path += " " + f[i]
		}
		if _, err := os.Stat(path); err == nil {
			best = path
		}
	}
	if best == "" {
		return f[0]
	}
	return best
}

// RedactCommand hides values that sit on a secret flag. It does not read files.
func RedactCommand(command string) string {
	f := strings.Fields(command)
	for i := 0; i < len(f); i++ {
		name, inline, ok := splitSecret(f[i])
		if !ok {
			continue
		}
		if inline {
			f[i] = name + "=***"
			continue
		}
		if i+1 < len(f) && !strings.HasPrefix(f[i+1], "-") {
			f[i+1] = "***"
			i++
		}
	}
	return strings.Join(f, " ")
}

func splitSecret(tok string) (name string, inline, ok bool) {
	raw := tok
	if i := strings.IndexByte(tok, '='); i >= 0 {
		raw = tok[:i]
		inline = true
	}
	if !secretFlag(raw) {
		return "", false, false
	}
	return raw, inline, true
}

func secretFlag(s string) bool {
	s = strings.ToLower(strings.TrimLeft(s, "-"))
	switch s {
	case "api-key", "apikey", "api_key", "token", "access-token", "auth-token", "password", "passwd", "secret":
		return true
	default:
		return false
	}
}

// MatchesName reports a command line for this agent. The macOS text caret
// (CursorUIViewService, com.apple.TextInputUI) is not the Cursor app.
func MatchesName(command, name string) bool {
	lowName := strings.ToLower(strings.TrimSpace(name))
	lowCmd := strings.ToLower(command)
	if lowName == "" || !strings.Contains(lowCmd, lowName) {
		return false
	}
	if lowName == "cursor" && (strings.Contains(lowCmd, "cursoruiviewservice") || strings.Contains(lowCmd, "com.apple.textinputui")) {
		return false
	}
	return true
}

// MatchName finds command lines that contain name, case-insensitively.
func MatchName(rows []Row, name string) []Row {
	var out []Row
	for _, r := range rows {
		if MatchesName(r.Command, name) {
			out = append(out, r)
		}
	}
	return out
}

// Agent is one row of the agents table.
type Agent struct {
	PID, User, Name, Program string
}

// MatchAgents keeps processes whose command contains one of AgentNames.
func MatchAgents(rows []Row) []Agent {
	var out []Agent
	for _, r := range rows {
		for _, name := range AgentNames {
			if MatchesName(r.Command, name) {
				out = append(out, Agent{PID: r.PID, User: r.User, Name: name, Program: ProgramPath(r.Command)})
				break
			}
		}
	}
	return out
}

// CapPaths keeps the first n unique paths and reports how many were left out.
func CapPaths(paths []string, n int) (shown []string, omitted int) {
	seen := map[string]bool{}
	for _, p := range paths {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		if len(shown) < n {
			shown = append(shown, p)
			continue
		}
		omitted++
	}
	return shown, omitted
}
