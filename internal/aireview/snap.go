package aireview

import (
	"os"
	"strconv"
	"strings"

	"github.com/bhupenderbhardwaj/bhuppi/internal/procview"
)

// Snap is one process from the wide ps used by the review.
type Snap struct {
	PID, PPID, User, State, Nice, Priority, CPU, Mem, Elapsed, Command string
}

// Files is the executable and working directory lsof reported for one pid.
type Files struct {
	Program, Cwd string
}

// Open is one lsof row: pid, fd, and NAME.
type Open struct {
	PID, FD, Name string
}

// ParseSnaps reads `ps -o pid=,ppid=,user=,ni=,pri=,stat=,%cpu=,%mem=,etime=,command=`.
func ParseSnaps(text string) []Snap {
	var out []Snap
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		f := strings.Fields(line)
		if len(f) < 9 {
			continue
		}
		if _, err := strconv.Atoi(f[0]); err != nil {
			continue
		}
		if _, err := strconv.Atoi(f[1]); err != nil {
			continue
		}
		out = append(out, Snap{
			PID: f[0], PPID: f[1], User: f[2], Nice: f[3], Priority: f[4], State: f[5],
			CPU: f[6], Mem: f[7], Elapsed: f[8], Command: strings.Join(f[9:], " "),
		})
	}
	return out
}

// MatchSnaps keeps command lines that name this agent.
func MatchSnaps(table []Snap, name string) []Snap {
	var out []Snap
	for _, s := range table {
		if procview.MatchesName(s.Command, name) {
			out = append(out, s)
		}
	}
	return out
}

// LsofArgv names every pid in as few lsof commands as the argument list allows.
func LsofArgv(pids []string, net bool) [][]string {
	if len(pids) == 0 {
		return nil
	}
	const batch = 40
	var out [][]string
	for i := 0; i < len(pids); i += batch {
		j := i + batch
		if j > len(pids) {
			j = len(pids)
		}
		arg := strings.Join(pids[i:j], ",")
		argv := []string{"lsof", "-n", "-P", "-p", arg}
		if net {
			argv = []string{"lsof", "-n", "-P", "-a", "-p", arg, "-i"}
		}
		out = append(out, argv)
	}
	return out
}

// ParseOpen reads lsof's PID, FD, and NAME columns.
func ParseOpen(text string) []Open {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return nil
	}
	pidAt := strings.Index(lines[0], "PID")
	fdAt := strings.Index(lines[0], "FD")
	nameAt := strings.Index(lines[0], "NAME")
	if pidAt < 0 || fdAt < 0 || nameAt < 0 {
		return nil
	}
	var out []Open
	for _, ln := range lines[1:] {
		if strings.TrimSpace(ln) == "" || len(ln) <= nameAt {
			continue
		}
		pid := fieldAt(ln, pidAt)
		fd := fieldAt(ln, fdAt)
		if _, err := strconv.Atoi(pid); err != nil {
			continue
		}
		out = append(out, Open{PID: pid, FD: fd, Name: strings.TrimSpace(ln[nameAt:])})
	}
	return out
}

func fieldAt(line string, at int) string {
	if at < 0 || at >= len(line) {
		return ""
	}
	// lsof sometimes starts a right-aligned value one column before the header word.
	for at > 0 && line[at] != ' ' && line[at-1] != ' ' {
		at--
	}
	f := strings.Fields(line[at:])
	if len(f) == 0 {
		return ""
	}
	return f[0]
}

// FamilyPIDs is pid plus every descendant.
func FamilyPIDs(table []Snap, pid string) []string {
	byPPID := map[string][]string{}
	for _, s := range table {
		byPPID[s.PPID] = append(byPPID[s.PPID], s.PID)
	}
	var out []string
	seen := map[string]bool{}
	var walk func(id string)
	walk = func(id string) {
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		out = append(out, id)
		for _, c := range byPPID[id] {
			if c != id {
				walk(c)
			}
		}
	}
	walk(pid)
	return out
}

// Blocks builds one view per matching process. table is the full ps list.
func Blocks(table, matched []Snap, files map[string]Files) []Block {
	var out []Block
	for _, s := range matched {
		f := files[s.PID]
		prog := chooseProgram(s.Command, f.Program)
		chain := chainOf(table, s.PID)
		for i := range chain {
			chain[i].Program = chooseProgram(chain[i].Command, files[chain[i].PID].Program)
		}
		kids := childrenOf(table, s.PID)
		for i := range kids {
			kids[i].Program = chooseProgram(kids[i].Command, files[kids[i].PID].Program)
		}
		out = append(out, Block{
			Snap:     s,
			Program:  prog,
			Cwd:      f.Cwd,
			Chain:    chain,
			Children: kids,
		})
	}
	return out
}

func chainOf(table []Snap, pid string) []Proc {
	by := map[string]Snap{}
	for _, s := range table {
		by[s.PID] = s
	}
	var up []Snap
	seen := map[string]bool{}
	cur := pid
	for cur != "" && !seen[cur] {
		s, ok := by[cur]
		if !ok {
			break
		}
		seen[cur] = true
		up = append(up, s)
		if s.PID == "1" || s.PPID == "" || s.PPID == "0" || s.PPID == s.PID {
			break
		}
		cur = s.PPID
	}
	procs := make([]Proc, len(up))
	for i := range up {
		s := up[len(up)-1-i]
		procs[i] = procOf(s)
	}
	return procs
}

func childrenOf(table []Snap, pid string) []Proc {
	var out []Proc
	for _, s := range table {
		if s.PPID == pid && s.PID != pid {
			out = append(out, procOf(s))
		}
	}
	return out
}

func procOf(s Snap) Proc {
	return Proc{PID: s.PID, PPID: s.PPID, Program: chooseProgram(s.Command, ""), Command: s.Command}
}

// chooseProgram keeps the on-disk executable. A longer path from the command
// line wins over a shorter lsof name that stopped at a space.
func chooseProgram(command, lsof string) string {
	p := procview.ProgramPath(command)
	if lsof != "" && exists(lsof) && len(lsof) > len(p) {
		return lsof
	}
	if exists(p) {
		return p
	}
	if lsof != "" {
		return lsof
	}
	return p
}

func exists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// ShortCommand is the program path plus a role. The long argument string stays off the page.
func ShortCommand(program, command string) string {
	command = procview.RedactCommand(command)
	if program == "" {
		program = procview.ProgramPath(command)
	}
	role := roleOf(command)
	if role == "" {
		role = roleOf(program)
	}
	switch {
	case program == "" && role == "":
		return command
	case program == "":
		return role
	case role == "":
		return program
	default:
		return program + " " + role
	}
}

func roleOf(s string) string {
	if v := typeFlag(s); v != "" && len(v) <= 32 && !strings.ContainsAny(v, "/\\") {
		return v
	}
	low := strings.ToLower(s)
	for _, role := range []string{"gpu-process", "renderer", "crashpad", "codex-cli", "network"} {
		if strings.Contains(low, role) {
			return role
		}
	}
	return ""
}

func typeFlag(s string) string {
	const eq = "--type="
	if i := strings.Index(s, eq); i >= 0 {
		rest := s[i+len(eq):]
		if f := strings.Fields(rest); len(f) > 0 {
			return f[0]
		}
	}
	f := strings.Fields(s)
	for i, a := range f {
		if a == "--type" && i+1 < len(f) {
			return f[i+1]
		}
	}
	return ""
}
