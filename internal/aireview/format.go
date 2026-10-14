package aireview

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bhupenderbhardwaj/bhuppi/internal/procview"
	"github.com/bhupenderbhardwaj/bhuppi/internal/style"
)

// Proc is one process in the review tree.
type Proc struct {
	PID, PPID, Program, Command string
}

// Persist is one launch agent or user unit. Sensitive marks the row, not a file body.
type Persist struct {
	Label, Program string
	RunAtLoad      bool
	Sensitive      bool
}

// Block is one matching process, shown with the proc view fields.
type Block struct {
	Snap            Snap
	Program, Cwd    string
	Chain, Children []Proc
}

// Report is one agent block.
type Report struct {
	Heading   string
	Blocks    []Block
	Inbound   []Sock
	Outbound  []Sock
	Sensitive []string
	Persist   []Persist
	Cron      []string
	Launch    []Job
}

// None is the line when no named agent is running.
func None() string {
	return "no matching agents are running\nlooked for: " + strings.Join(procview.AgentNames, ", ") + "\n"
}

// Format renders one review. Abnormal lines are red and start with !.
func Format(r Report, color bool) string {
	var b strings.Builder
	if r.Heading != "" {
		fmt.Fprintln(&b, paint(style.Accent, r.Heading, color))
		b.WriteByte('\n')
	}
	b.WriteString(summary(r, color))
	b.WriteByte('\n')
	rule(&b, "processes", color)
	if len(r.Blocks) == 0 {
		fmt.Fprintln(&b, "  none")
	} else {
		n := len(r.Blocks)
		for i, blk := range r.Blocks {
			if i > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(processBox(i+1, n, blk, color))
		}
	}
	b.WriteByte('\n')
	rule(&b, "connections", color)
	fmt.Fprintln(&b, paint(style.Accent, "  Inbound", color))
	if len(r.Inbound) == 0 {
		fmt.Fprintln(&b, "    none")
	} else {
		for _, s := range r.Inbound {
			fmt.Fprintf(&b, "    %s\n", inboundLine(s, nil, color))
		}
	}
	fmt.Fprintln(&b, paint(style.Accent, "  Outbound", color))
	b.WriteString(outboundTree(r.Outbound, color))
	b.WriteByte('\n')
	rule(&b, "sensitive paths", color)
	if len(r.Sensitive) == 0 {
		fmt.Fprintln(&b, "  none")
	} else {
		for _, p := range r.Sensitive {
			fmt.Fprintf(&b, "  %s\n", paint(style.Hidden, "! "+p+"  ! sensitive path", color))
		}
	}
	b.WriteByte('\n')
	rule(&b, "persistence", color)
	if len(r.Persist) == 0 {
		fmt.Fprintln(&b, "  none")
	} else {
		for _, p := range r.Persist {
			b.WriteString(persistBlock(p, color))
		}
	}
	b.WriteByte('\n')
	rule(&b, "scheduled tasks", color)
	if len(r.Cron) == 0 && len(r.Launch) == 0 {
		fmt.Fprintln(&b, "  none")
	} else {
		if len(r.Cron) == 0 {
			fmt.Fprintln(&b, "  crontab")
			fmt.Fprintln(&b, "    none")
		} else {
			fmt.Fprintln(&b, "  crontab")
			for _, line := range r.Cron {
				fmt.Fprintf(&b, "    %s\n", line)
			}
		}
		if len(r.Launch) > 0 {
			fmt.Fprintln(&b, "  launchctl")
			for _, j := range r.Launch {
				line := j.PID + "  " + j.Status + "  " + j.Label
				if badStatus(j.Status) {
					line = "! " + line
					fmt.Fprintf(&b, "    %s\n", paint(style.Hidden, line, color))
					continue
				}
				fmt.Fprintf(&b, "    %s  %s  %s\n", paint(style.Number, j.PID, color), j.Status, j.Label)
			}
		}
	}
	return b.String()
}

type sumRow struct {
	label string
	n     int
	warn  bool
}

func summaryRows(r Report) []sumRow {
	_, outbound := Counts(GroupOutbound(r.Outbound))
	creds := 0
	for _, p := range r.Sensitive {
		if CredentialPath(p) {
			creds++
		}
	}
	return []sumRow{
		{"processes", len(r.Blocks), false},
		{"inbound", len(r.Inbound), false},
		{"outbound", outbound, abnormalOutbound(r.Outbound) > 0},
		{"sensitive files", len(r.Sensitive), len(r.Sensitive) > 0},
		{"persistence", len(r.Persist), false},
		{"scheduled tasks", len(r.Cron) + len(r.Launch), false},
		{"credentials", creds, creds > 0},
	}
}

func summary(r Report, color bool) string {
	rows := summaryRows(r)
	var b strings.Builder
	section(&b, "Summary", color)
	fmt.Fprintf(&b, "  %s\n", joinSummary(rows[:3], color))
	fmt.Fprintf(&b, "  %s\n", joinSummary(rows[3:], color))
	return b.String()
}

func joinSummary(rows []sumRow, color bool) string {
	parts := make([]string, len(rows))
	for i, row := range rows {
		role := style.Number
		if row.warn {
			role = style.Hidden
		}
		parts[i] = row.label + " " + paint(role, strconv.Itoa(row.n), color)
	}
	return strings.Join(parts, "   ")
}

func abnormalOutbound(socks []Sock) int {
	n := 0
	for _, s := range socks {
		if s.RemoteHost == "" {
			continue
		}
		if procview.Abnormal(s.RemoteHost, s.RemotePort) {
			n++
		}
	}
	return n
}

// FormatList is the proc list page: a count, then one box per process.
func FormatList(table, shown []Snap, files map[string]Files, color bool) string {
	if shown == nil {
		shown = table
	}
	blocks := Blocks(table, shown, files)
	var b strings.Builder
	section(&b, "Summary", color)
	fmt.Fprintf(&b, "  processes %s\n", paint(style.Number, strconv.Itoa(len(blocks)), color))
	if len(blocks) == 0 {
		fmt.Fprintln(&b, "  none")
		return b.String()
	}
	b.WriteByte('\n')
	for i, blk := range blocks {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(processBox(i+1, len(blocks), blk, color))
	}
	return b.String()
}

// FormatLsof renders one lsof -i table as inbound and outbound.
func FormatLsof(text string, color bool) string {
	return FormatLsofNamed(text, nil, color)
}

// FormatLsofNamed is FormatLsof with reverse-DNS names beside numeric remotes.
// A missing name leaves the address. The address is never replaced by a name.
func FormatLsofNamed(text string, names map[string]string, color bool) string {
	var socks []Sock
	for _, o := range ParseOpen(text) {
		socks = append(socks, ParseSock(o.Name))
	}
	in, out := SplitDirection(socks)
	return formatConns(in, out, names, color)
}

func formatConns(in, out []Sock, names map[string]string, color bool) string {
	_, outbound := Counts(GroupOutbound(out))
	rows := []sumRow{
		{"inbound", len(in), false},
		{"outbound", outbound, abnormalOutbound(out) > 0},
	}
	var b strings.Builder
	section(&b, "Summary", color)
	fmt.Fprintf(&b, "  %s\n\n", joinSummary(rows, color))
	rule(&b, "inbound", color)
	if len(in) == 0 {
		fmt.Fprintln(&b, "  none")
	} else {
		for _, s := range in {
			fmt.Fprintf(&b, "  %s\n", inboundLine(s, names, color))
		}
	}
	b.WriteByte('\n')
	rule(&b, "outbound", color)
	b.WriteString(connTree(out, names, color))
	return b.String()
}

func connTree(socks []Sock, names map[string]string, color bool) string {
	groups := GroupOutbound(socks)
	if len(groups) == 0 {
		return "  none\n"
	}
	hosts, conns := Counts(groups)
	var b strings.Builder
	if Excessive(hosts, conns) {
		line := fmt.Sprintf("! excessive outbound  ! %d hosts, %d connections", hosts, conns)
		fmt.Fprintf(&b, "  %s\n", paint(style.Hidden, line, color))
	}
	shown := groups
	omitted := 0
	if len(shown) > MaxHosts {
		omitted = len(shown) - MaxHosts
		shown = shown[:MaxHosts]
	}
	for _, g := range shown {
		hostLine, hostRole := labeledHost(g.Host, names)
		fmt.Fprintf(&b, "  %s\n", paint(hostRole, hostLine, color))
		for i, p := range g.Ports {
			branch := "├── "
			if i == len(g.Ports)-1 {
				branch = "└── "
			}
			label := fmt.Sprintf(":%s  (%d)", p.Port, p.N)
			role := style.Number
			if portReason(p.Port) != "" {
				label = "! " + label + "  ! " + portReason(p.Port)
				role = style.Hidden
			}
			fmt.Fprintf(&b, "  %s%s\n", paint(style.Muted, branch, color), paint(role, label, color))
		}
	}
	if omitted > 0 {
		fmt.Fprintf(&b, "  %d hosts omitted\n", omitted)
	}
	return b.String()
}

func badStatus(s string) bool {
	if s == "" || s == "0" || s == "-" {
		return false
	}
	n, err := strconv.Atoi(s)
	return err != nil || n != 0
}

func inboundLine(s Sock, names map[string]string, color bool) string {
	local := endpoint(s.LocalHost, s.LocalPort)
	if s.Listen {
		return paint(style.Number, local, color) + " LISTEN"
	}
	remote := endpointNamed(s.RemoteHost, s.RemotePort, names)
	return paint(style.Number, remote, color) + " → " + paint(style.Number, local, color)
}

// labeledHost keeps the numeric address and appends a resolved name when one exists.
// A resolved name clears the bare-address mark. An unusual port is still marked on its own line.
func labeledHost(host string, names map[string]string) (string, style.Role) {
	label := host
	if n := names[host]; n != "" {
		label = host + "  " + n
	}
	reason := hostReason(host)
	if reason == "bare address" && names[host] != "" {
		reason = ""
	}
	if reason != "" {
		return "! " + label + "  ! " + reason, style.Hidden
	}
	return label, style.Number
}

func endpointNamed(host, port string, names map[string]string) string {
	if n := names[host]; n != "" {
		if port == "" {
			return host + "  " + n
		}
		return host + "  " + n + ":" + port
	}
	return endpoint(host, port)
}

func endpoint(host, port string) string {
	if host == "" {
		return port
	}
	if port == "" {
		return host
	}
	return host + ":" + port
}

func outboundTree(socks []Sock, color bool) string {
	groups := GroupOutbound(socks)
	hosts, conns := Counts(groups)
	var b strings.Builder
	if len(groups) == 0 {
		fmt.Fprintln(&b, "    none")
		return b.String()
	}
	if Excessive(hosts, conns) {
		line := fmt.Sprintf("! excessive outbound  ! %d hosts, %d connections", hosts, conns)
		fmt.Fprintf(&b, "    %s\n", paint(style.Hidden, line, color))
	}
	shown := groups
	omitted := 0
	if len(shown) > MaxHosts {
		omitted = len(shown) - MaxHosts
		shown = shown[:MaxHosts]
	}
	for _, g := range shown {
		hostLine := g.Host
		hostRole := style.Number
		if reason := hostReason(g.Host); reason != "" {
			hostLine = "! " + g.Host + "  ! " + reason
			hostRole = style.Hidden
		}
		fmt.Fprintf(&b, "    %s\n", paint(hostRole, hostLine, color))
		for i, p := range g.Ports {
			branch := "├── "
			if i == len(g.Ports)-1 {
				branch = "└── "
			}
			label := fmt.Sprintf(":%s  (%d)", p.Port, p.N)
			role := style.Number
			if portReason(p.Port) != "" {
				label = "! " + label + "  ! " + portReason(p.Port)
				role = style.Hidden
			}
			fmt.Fprintf(&b, "    %s%s\n", paint(style.Muted, branch, color), paint(role, label, color))
		}
	}
	if omitted > 0 {
		fmt.Fprintf(&b, "    %d hosts omitted\n", omitted)
	}
	return b.String()
}

func hostReason(host string) string {
	if host == "" {
		return ""
	}
	if procview.Abnormal(host, "443") && procview.Reason(host, "443") == "bare address" {
		return "bare address"
	}
	return ""
}

func portReason(port string) string {
	if port != "80" && port != "443" {
		return "unusual port"
	}
	return ""
}

func persistBlock(p Persist, color bool) string {
	load := "false"
	if p.RunAtLoad {
		load = "true"
	}
	var b strings.Builder
	label := p.Label
	if label == "" {
		label = p.Program
	}
	if p.Sensitive {
		fmt.Fprintf(&b, "  %s\n", paint(style.Hidden, "! "+label, color))
	} else {
		fmt.Fprintf(&b, "  %s\n", label)
	}
	fmt.Fprintf(&b, "    program  %s\n", paint(style.Path, p.Program, color))
	fmt.Fprintf(&b, "    run at load  %s\n", load)
	return b.String()
}

func processBox(i, n int, blk Block, color bool) string {
	s := blk.Snap
	bar := paint(style.Muted, "│", color)
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s  pid %s\n", paint(style.Muted, "┌", color), paint(style.Number, fmt.Sprintf("%d/%d", i, n), color), paint(style.Number, s.PID, color))
	fmt.Fprintf(&b, "%s user %s   state %s   nice %s   priority %s\n", bar, paint(style.Accent, s.User, color), paint(style.Accent, s.State, color), paint(style.Number, s.Nice, color), paint(style.Number, s.Priority, color))
	fmt.Fprintf(&b, "%s cpu %s   memory %s   elapsed %s\n", bar, paint(style.Number, s.CPU, color), paint(style.Number, s.Mem, color), paint(style.Number, s.Elapsed, color))
	prog := blk.Program
	if prog == "" {
		prog = "the program file is not visible"
	}
	fmt.Fprintf(&b, "%s program  %s\n", bar, paint(style.Path, prog, color))
	fmt.Fprintf(&b, "%s command  %s\n", bar, paint(style.File, ShortCommand(blk.Program, s.Command), color))
	fmt.Fprintf(&b, "%s called by\n", bar)
	for _, line := range chainLines(blk.Chain, color) {
		fmt.Fprintf(&b, "%s   %s\n", bar, line)
	}
	if len(blk.Children) == 0 {
		fmt.Fprintf(&b, "%s children  none\n", bar)
	} else {
		fmt.Fprintf(&b, "%s children\n", bar)
		for _, k := range blk.Children {
			fmt.Fprintf(&b, "%s   %s  %s\n", bar, paint(style.Number, k.PID, color), paint(style.File, ShortCommand(k.Program, k.Command), color))
		}
	}
	cwd := blk.Cwd
	if cwd == "" {
		cwd = "—"
	}
	fmt.Fprintf(&b, "%s cwd  %s\n", bar, paint(style.Path, cwd, color))
	fmt.Fprintln(&b, paint(style.Muted, "└", color))
	return b.String()
}

func chainLines(rows []Proc, color bool) []string {
	if len(rows) == 0 {
		return []string{"the caller chain is not visible"}
	}
	out := make([]string, len(rows))
	for i, r := range rows {
		indent := ""
		if i > 0 {
			indent = strings.Repeat("    ", i-1) + "└── "
		}
		line := paint(style.Muted, indent, color) + paint(style.Number, r.PID, color) + "  " + paint(style.Path, ShortCommand(r.Program, r.Command), color)
		if i > 0 {
			line += "  " + paint(style.Number, "(Level "+strconv.Itoa(i)+")", color)
		}
		out[i] = line
	}
	return out
}

func rule(b *strings.Builder, title string, color bool) {
	fmt.Fprintln(b, paint(style.Accent, "── "+title+" ──", color))
}

func section(b *strings.Builder, title string, color bool) {
	fmt.Fprintln(b, paint(style.Accent, title, color))
}

func paint(role style.Role, text string, color bool) string {
	if !color || text == "" {
		return text
	}
	return style.Paint(role, text, style.Auto, false)
}
