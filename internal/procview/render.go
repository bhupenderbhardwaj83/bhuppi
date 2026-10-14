package procview

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/bhupenderbhardwaj/bhuppi/internal/style"
)

// MaxPaths is how many open paths one page shows.
const MaxPaths = 40

// MaxBlocks is how many agent processes one name page shows.
const MaxBlocks = 8

// Page is one process snapshot. Files and Outbound are set for proc agents.
type Page struct {
	Heading  string
	Detail   Detail
	Program  string
	Cwd      string
	Chain    []Row
	Kids     []Row
	Conns    []string
	Files    []string
	Outbound []End
	Agents   bool
}

// Format renders one process page.
func Format(p Page, color bool) string {
	var b strings.Builder
	if p.Heading != "" {
		fmt.Fprintln(&b, p.Heading)
		b.WriteByte('\n')
	}
	writeSection(&b, "Process", color)
	if !p.Detail.OK {
		fmt.Fprintln(&b, "  the process is not visible")
	} else {
		writeKV(&b, "pid", p.Detail.PID, style.Number, color)
		writeKV(&b, "user", p.Detail.User, style.Accent, color)
		writeKV(&b, "state", p.Detail.State, style.Accent, color)
		writeKV(&b, "nice", p.Detail.Nice, style.Number, color)
		writeKV(&b, "priority", p.Detail.Priority, style.Number, color)
		writeKV(&b, "cpu", p.Detail.CPU, style.Number, color)
		writeKV(&b, "memory", p.Detail.Mem, style.Number, color)
		writeKV(&b, "elapsed", p.Detail.Elapsed, style.Number, color)
		writeKV(&b, "command", RedactCommand(p.Detail.Command), style.File, color)
	}
	b.WriteByte('\n')
	writeSection(&b, "Program", color)
	if p.Program == "" {
		fmt.Fprintln(&b, "  the program file is not visible")
	} else {
		fmt.Fprintf(&b, "  %s\n", markPath(p.Program, color))
	}
	b.WriteByte('\n')
	writeSection(&b, "Called by", color)
	b.WriteString(chain(p.Chain, color))
	b.WriteByte('\n')
	writeSection(&b, "Children", color)
	if len(p.Kids) == 0 {
		fmt.Fprintln(&b, "  none")
	} else {
		for _, k := range p.Kids {
			fmt.Fprintf(&b, "  %s  %s\n", paint(style.Number, k.PID, color), paint(style.File, RedactCommand(k.Command), color))
		}
	}
	b.WriteByte('\n')
	writeSection(&b, "Working directory", color)
	if p.Cwd == "" {
		fmt.Fprintln(&b, "  the working directory is not visible")
	} else {
		fmt.Fprintf(&b, "  %s\n", markPath(p.Cwd, color))
	}
	b.WriteByte('\n')
	writeSection(&b, "Connections", color)
	if len(p.Conns) == 0 {
		fmt.Fprintln(&b, "  none")
	} else {
		for _, c := range p.Conns {
			fmt.Fprintf(&b, "  %s\n", colorNet(c, color))
		}
	}
	if p.Agents {
		b.WriteByte('\n')
		writeSection(&b, "Files open", color)
		shown, omitted := CapPaths(p.Files, MaxPaths)
		if len(shown) == 0 {
			fmt.Fprintln(&b, "  none")
		} else {
			b.WriteString(groupPaths(shown, color))
			if omitted > 0 {
				fmt.Fprintf(&b, "  %d paths omitted\n", omitted)
			}
		}
		b.WriteByte('\n')
		writeSection(&b, "Outbound", color)
		if len(p.Outbound) == 0 {
			fmt.Fprintln(&b, "  none")
		} else {
			for _, e := range p.Outbound {
				fmt.Fprintf(&b, "  %s\n", formatEnd(e, color))
			}
		}
	}
	return b.String()
}

// FormatAgents is the short table for `proc agents` with no name.
func FormatAgents(list []Agent, color bool) string {
	if len(list) == 0 {
		return "no matching agents are running\nlooked for: " + strings.Join(AgentNames, ", ") + "\n"
	}
	var b strings.Builder
	fmt.Fprintln(&b, paint(style.Accent, "pid     user     name          program", color))
	for _, a := range list {
		fmt.Fprintf(&b, "%s  %s  %s  %s\n",
			paint(style.Number, fmt.Sprintf("%-7s", a.PID), color),
			paint(style.Accent, fmt.Sprintf("%-8s", a.User), color),
			fmt.Sprintf("%-13s", a.Name),
			paint(style.Path, a.Program, color),
		)
	}
	return b.String()
}

// FormatChoice lists pids when a keyword or a name matches too many processes.
func FormatChoice(what string, rows []Row, hint string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d processes match %s\n", len(rows), what)
	for _, r := range rows {
		cmd := RedactCommand(r.Command)
		if cmd == "" {
			cmd = r.PID
		}
		fmt.Fprintf(&b, "  %s  %s\n", r.PID, cmd)
	}
	fmt.Fprintln(&b, hint)
	return b.String()
}

func formatEnd(e End, color bool) string {
	label := e.Host + ":" + e.Port
	if reason := Reason(e.Host, e.Port); reason != "" {
		line := "! " + label + "  ! " + reason
		return paint(style.Hidden, line, color)
	}
	return paint(style.Number, label, color)
}

func markPath(p string, color bool) string {
	if Sensitive(p) {
		return paint(style.Hidden, "! "+p+"  ! sensitive path", color)
	}
	return paint(style.Path, p, color)
}

func groupPaths(paths []string, color bool) string {
	type grp struct {
		dir   string
		paths []string
	}
	var order []grp
	at := map[string]int{}
	for _, p := range paths {
		dir := filepath.Dir(p)
		i, ok := at[dir]
		if !ok {
			i = len(order)
			at[dir] = i
			order = append(order, grp{dir: dir})
		}
		order[i].paths = append(order[i].paths, p)
	}
	var b strings.Builder
	for _, g := range order {
		fmt.Fprintf(&b, "  %s\n", paint(style.Path, g.dir, color))
		for _, p := range g.paths {
			fmt.Fprintf(&b, "    %s\n", markPath(p, color))
		}
	}
	return b.String()
}

func chain(rows []Row, color bool) string {
	if len(rows) == 0 {
		return "  the caller chain is not visible\n"
	}
	var plain []string
	for i, r := range rows {
		cmd := RedactCommand(r.Command)
		if cmd == "" {
			cmd = r.PID
		}
		line := r.PID + " " + cmd
		if i > 0 {
			indent := strings.Repeat("    ", i-1)
			branch := "└── "
			if i > 1 {
				line = indent + branch + line
			} else {
				line = branch + line
			}
		}
		plain = append(plain, line)
	}
	width := 0
	for _, ln := range plain {
		if n := utf8.RuneCountInString(ln); n > width {
			width = n
		}
	}
	var b strings.Builder
	for i, r := range rows {
		ln := plain[i]
		fmt.Fprint(&b, "  ")
		b.WriteString(paintChain(ln, r, i, color))
		if i > 0 {
			gap := width - utf8.RuneCountInString(ln) + 2
			if gap < 2 {
				gap = 2
			}
			b.WriteString(strings.Repeat(" ", gap))
			b.WriteString(paint(style.Number, "(Level "+strconv.Itoa(i)+")", color))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func paintChain(line string, r Row, depth int, color bool) string {
	if !color {
		return line
	}
	// Recolor pid and command. The indent and branch stay muted.
	pid := r.PID
	cmd := RedactCommand(r.Command)
	if cmd == "" {
		cmd = r.PID
	}
	i := strings.LastIndex(line, pid+" "+cmd)
	if i < 0 {
		return line
	}
	return paint(style.Muted, line[:i], color) + paint(style.Number, pid, color) + " " + paint(style.File, cmd, color)
}

func colorNet(line string, color bool) string {
	if !color {
		return line
	}
	f := strings.Fields(line)
	for i, w := range f {
		if looksAddr(w) {
			f[i] = paint(style.Number, w, true)
		}
	}
	return strings.Join(f, " ")
}

func looksAddr(w string) bool {
	w = strings.Trim(w, "()")
	if w == "" {
		return false
	}
	if strings.Contains(w, ":") || strings.Count(w, ".") == 3 {
		return true
	}
	_, err := strconv.Atoi(w)
	return err == nil
}

func writeSection(b *strings.Builder, title string, color bool) {
	fmt.Fprintln(b, paint(style.Accent, title, color))
}

func writeKV(b *strings.Builder, key, val string, role style.Role, color bool) {
	fmt.Fprintf(b, "  %-10s %s\n", key, paint(role, val, color))
}

func paint(role style.Role, text string, color bool) string {
	if !color || text == "" {
		return text
	}
	return style.Paint(role, text, style.Auto, false)
}
