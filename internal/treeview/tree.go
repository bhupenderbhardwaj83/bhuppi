// Package treeview draws the directory and process trees bhuppi prints.
// Discovery commands only supply entries. They are not the picture.
package treeview

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/bhupenderbhardwaj/bhuppi/internal/style"
)

// Dir turns find's path list into a box-drawing tree.
// root is the path that was walked. depth is how many levels under the root
// were requested; entries deeper than that are omitted.
func Dir(root, listing string, depth int, color bool) string {
	root = filepath.Clean(root)
	top := &dnode{name: rootLabel(root), dir: true}
	for _, line := range strings.Split(listing, "\n") {
		line = strings.TrimRight(line, "\r")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = filepath.Clean(line)
		if line == root {
			continue
		}
		rel, err := filepath.Rel(root, line)
		if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
			continue
		}
		parts := strings.Split(rel, string(os.PathSeparator))
		if len(parts) > depth {
			continue
		}
		cur := top
		for i, part := range parts {
			if part == "" || part == "." {
				continue
			}
			ch := cur.take(part)
			if i < len(parts)-1 {
				ch.dir = true
			}
			cur = ch
		}
		if cur != top {
			if st, err := os.Stat(line); err == nil && st.IsDir() {
				cur.dir = true
			}
		}
	}
	rows := dirRows(top, "", true, 0, depth)
	return paintRows(rows, color)
}

func rootLabel(root string) string {
	if root == "." {
		return "./"
	}
	base := filepath.Base(root)
	if base == "" || base == string(os.PathSeparator) {
		base = root
	}
	return base + "/"
}

type dnode struct {
	name string
	dir  bool
	kids []*dnode
}

func (n *dnode) take(name string) *dnode {
	for _, k := range n.kids {
		if k.name == name {
			return k
		}
	}
	k := &dnode{name: name}
	n.kids = append(n.kids, k)
	return k
}

func (n *dnode) sort() {
	sort.Slice(n.kids, func(i, j int) bool { return n.kids[i].name < n.kids[j].name })
	for _, k := range n.kids {
		k.sort()
	}
}

type row struct {
	indent string
	label  string
	role   style.Role
	level  int
}

func dirRows(n *dnode, prefix string, root bool, level, max int) []row {
	n.sort()
	label := n.name
	role := style.File
	if n.dir && !strings.HasSuffix(label, "/") {
		label += "/"
	}
	if isHidden(n.name) {
		role = style.Hidden
	} else if n.dir {
		role = style.Folder
	}
	rows := []row{{indent: prefix, label: label, role: role, level: level}}
	if !root && level >= max {
		return rows
	}
	for i, k := range n.kids {
		last := i == len(n.kids)-1
		branch := "├── "
		next := prefix + "│   "
		if last {
			branch = "└── "
			next = prefix + "    "
		}
		childLevel := level + 1
		if root {
			childLevel = 1
		}
		sub := dirRows(k, "", false, childLevel, max)
		if len(sub) == 0 {
			continue
		}
		sub[0].indent = prefix + branch
		for i := 1; i < len(sub); i++ {
			sub[i].indent = next + sub[i].indent
		}
		rows = append(rows, sub...)
	}
	return rows
}

func isHidden(name string) bool {
	base := strings.TrimSuffix(name, "/")
	return base != "." && base != ".." && strings.HasPrefix(base, ".")
}

// Proc is one process row from ps.
type Proc struct {
	PID, PPID, User, Command string
}

// ParsePS reads `ps -axo pid=,ppid=,user=,ni=,stat=,command=`.
func ParsePS(text string) []Proc {
	var out []Proc
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		f := strings.Fields(line)
		if len(f) < 6 {
			continue
		}
		if _, err := strconv.Atoi(f[0]); err != nil {
			continue
		}
		out = append(out, Proc{
			PID: f[0], PPID: f[1], User: f[2],
			Command: strings.Join(f[5:], " "),
		})
	}
	return out
}

// ProcTree draws a process tree from a ps table. depth is levels under the roots.
func ProcTree(procs []Proc, depth int, color bool) string {
	by := map[string]*pnode{}
	var order []string
	for _, p := range procs {
		if _, ok := by[p.PID]; ok {
			continue
		}
		by[p.PID] = &pnode{p: p}
		order = append(order, p.PID)
	}
	var roots []*pnode
	for _, id := range order {
		n := by[id]
		parent := by[n.p.PPID]
		if parent == nil || n.p.PPID == "0" || n.p.PPID == n.p.PID {
			roots = append(roots, n)
			continue
		}
		parent.kids = append(parent.kids, n)
	}
	sort.Slice(roots, func(i, j int) bool { return lessPID(roots[i].p.PID, roots[j].p.PID) })
	var rows []row
	for _, r := range roots {
		rows = append(rows, procRows(r, "", true, 0, depth)...)
	}
	return paintProc(rows, color)
}

type pnode struct {
	p    Proc
	kids []*pnode
}

func (n *pnode) sort() {
	sort.Slice(n.kids, func(i, j int) bool { return lessPID(n.kids[i].p.PID, n.kids[j].p.PID) })
	for _, k := range n.kids {
		k.sort()
	}
}

func lessPID(a, b string) bool {
	ai, ea := strconv.Atoi(a)
	bi, eb := strconv.Atoi(b)
	if ea == nil && eb == nil && ai != bi {
		return ai < bi
	}
	return a < b
}

func procRows(n *pnode, prefix string, root bool, level, max int) []row {
	n.sort()
	rows := []row{{indent: prefix, label: n.p.PID + "\t" + n.p.User + "\t" + trimCmd(n.p.Command), level: level}}
	if !root && level >= max {
		return rows
	}
	for i, k := range n.kids {
		last := i == len(n.kids)-1
		branch := "├── "
		next := prefix + "│   "
		if last {
			branch = "└── "
			next = prefix + "    "
		}
		childLevel := 1
		if !root {
			childLevel = level + 1
		}
		if childLevel > max {
			continue
		}
		sub := procRows(k, "", false, childLevel, max)
		if len(sub) == 0 {
			continue
		}
		sub[0].indent = prefix + branch
		for i := 1; i < len(sub); i++ {
			sub[i].indent = next + sub[i].indent
		}
		rows = append(rows, sub...)
	}
	return rows
}

func trimCmd(cmd string) string {
	const max = 64
	if utf8.RuneCountInString(cmd) <= max {
		return cmd
	}
	r := []rune(cmd)
	return string(r[:max-1]) + "…"
}

func paintRows(rows []row, color bool) string {
	width := 0
	for _, r := range rows {
		if n := utf8.RuneCountInString(r.indent + r.label); n > width {
			width = n
		}
	}
	var b strings.Builder
	for i, r := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		indent := paint(style.Muted, r.indent, color)
		label := paint(r.role, r.label, color)
		b.WriteString(indent)
		b.WriteString(label)
		if r.level > 0 {
			gap := width - utf8.RuneCountInString(r.indent+r.label) + 2
			if gap < 2 {
				gap = 2
			}
			b.WriteString(strings.Repeat(" ", gap))
			b.WriteString(paint(style.Number, "(Level "+strconv.Itoa(r.level)+")", color))
		}
	}
	if len(rows) > 0 {
		b.WriteByte('\n')
	}
	return b.String()
}

func paintProc(rows []row, color bool) string {
	// label is pid \t user \t command
	type vis struct {
		indent, pid, user, path, args string
		level                         int
	}
	var items []vis
	width := 0
	for _, r := range rows {
		pid, user, cmd, _ := splitProcLabel(r.label)
		path, args := splitCmd(cmd)
		plain := r.indent + pid + " " + user + " " + path
		if args != "" {
			plain += " " + args
		}
		if n := utf8.RuneCountInString(plain); n > width {
			width = n
		}
		items = append(items, vis{r.indent, pid, user, path, args, r.level})
	}
	var b strings.Builder
	for i, it := range items {
		if i > 0 {
			b.WriteByte('\n')
		}
		plain := it.indent + it.pid + " " + it.user + " " + it.path
		if it.args != "" {
			plain += " " + it.args
		}
		b.WriteString(paint(style.Muted, it.indent, color))
		b.WriteString(paint(style.Number, it.pid, color))
		b.WriteByte(' ')
		b.WriteString(paint(style.Accent, it.user, color))
		b.WriteByte(' ')
		b.WriteString(paint(style.File, it.path, color))
		if it.args != "" {
			b.WriteByte(' ')
			b.WriteString(it.args)
		}
		if it.level > 0 {
			gap := width - utf8.RuneCountInString(plain) + 2
			if gap < 2 {
				gap = 2
			}
			b.WriteString(strings.Repeat(" ", gap))
			b.WriteString(paint(style.Number, "(Level "+strconv.Itoa(it.level)+")", color))
		}
	}
	if len(items) > 0 {
		b.WriteByte('\n')
	}
	return b.String()
}

func splitProcLabel(label string) (pid, user, cmd string, ok bool) {
	parts := strings.SplitN(label, "\t", 3)
	if len(parts) != 3 {
		return "", "", label, false
	}
	return parts[0], parts[1], parts[2], true
}

func splitCmd(cmd string) (path, args string) {
	f := strings.Fields(cmd)
	if len(f) == 0 {
		return cmd, ""
	}
	return f[0], strings.Join(f[1:], " ")
}

func paint(role style.Role, text string, color bool) string {
	if !color || text == "" {
		return text
	}
	return style.Paint(role, text, style.Auto, false)
}
