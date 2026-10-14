package help

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/bhupenderbhardwaj/bhuppi/internal/catalog"
	"github.com/bhupenderbhardwaj/bhuppi/internal/style"
	"github.com/bhupenderbhardwaj/bhuppi/internal/warn"
)

// Menu is the one-line command index for `bhuppi` or a group with no further args.
// `bhuppi help` and `--help` stay on the long page.
func Menu(cat *catalog.Catalog, path []string, opt Options) (string, error) {
	var b strings.Builder
	if len(path) == 0 {
		nodes := make([]catalog.Node, 0, len(cat.Groups)+len(cat.MetaPages))
		for i := range cat.Groups {
			nodes = append(nodes, cat.Groups[i])
		}
		for i := range cat.MetaPages {
			nodes = append(nodes, cat.MetaPages[i])
		}
		writeIndex(&b, "", nodes, opt, true)
		return b.String(), nil
	}
	n, err := menuNode(cat, path)
	if err != nil {
		return "", err
	}
	if len(n.Children) == 0 {
		return "", fmt.Errorf("no commands under %s\nsee: bhuppi help %s", strings.Join(path, " "), strings.Join(path, " "))
	}
	writeIndex(&b, n.Name, n.Children, opt, false)
	return b.String(), nil
}

// Brief is the short description used in tab completion.
// The printed menu adds syntax and an example beside it.
func Brief(n *catalog.Node) string {
	if n == nil {
		return ""
	}
	return menuWhat(menuRow{names: []string{n.Name}, title: n.Title, node: *n})
}

// SectionOf is the menu group for one command name.
func SectionOf(parent, name string, top bool) string {
	if top {
		if s := topSection(name); s != "" {
			return s
		}
		return "setup"
	}
	return childSection(parent, name)
}

func menuNode(cat *catalog.Catalog, path []string) (*catalog.Node, error) {
	if m := cat.Meta(path[0]); m != nil {
		n := m
		for _, part := range path[1:] {
			ch := catalog.Child(n, part)
			if ch == nil {
				return nil, fmt.Errorf("no commands under %s\nsee: bhuppi help %s", strings.Join(path, " "), n.Name)
			}
			n = ch
		}
		return n, nil
	}
	n, _, err := walk(cat, path)
	if err != nil {
		return nil, err
	}
	return n, nil
}

type menuSection struct {
	title string
	rows  []menuRow
}

func menuSections(parent string, rows []menuRow, top bool) []menuSection {
	if top {
		return topSections(rows)
	}
	switch parent {
	case "files":
		return filesSections(rows)
	case "proc":
		return splitCareful(rows, "look")
	default:
		if rowsDanger(rows) {
			return splitCareful(rows, "commands")
		}
		return []menuSection{{title: "commands", rows: rows}}
	}
}

func topSections(rows []menuRow) []menuSection {
	var out []menuSection
	used := map[string]bool{}
	for _, sec := range topGroups {
		got := takeNames(rows, sec.names, used)
		if len(got) > 0 {
			out = append(out, menuSection{title: sec.title, rows: got})
		}
	}
	var rest []menuRow
	for _, row := range rows {
		if !rowUsed(row, used) {
			rest = append(rest, row)
		}
	}
	if len(rest) > 0 {
		out = append(out, menuSection{title: "setup", rows: rest})
	}
	return out
}

func filesSections(rows []menuRow) []menuSection {
	used := map[string]bool{}
	look := takeNames(rows, filesLook, used)
	change := takeNames(rows, filesChange, used)
	careful := takeNames(rows, filesCareful, used)
	for _, row := range rows {
		if rowUsed(row, used) {
			continue
		}
		if rowDanger(row) {
			careful = append(careful, row)
		} else {
			change = append(change, row)
		}
		markUsed(row, used)
	}
	return nonempty(
		menuSection{title: "look", rows: look},
		menuSection{title: "change", rows: change},
		menuSection{title: "careful", rows: careful},
	)
}

func splitCareful(rows []menuRow, ordinary string) []menuSection {
	var safe, care []menuRow
	for _, row := range rows {
		if rowDanger(row) {
			care = append(care, row)
		} else {
			safe = append(safe, row)
		}
	}
	return nonempty(
		menuSection{title: ordinary, rows: safe},
		menuSection{title: "careful", rows: care},
	)
}

func nonempty(secs ...menuSection) []menuSection {
	var out []menuSection
	for _, sec := range secs {
		if len(sec.rows) > 0 {
			out = append(out, sec)
		}
	}
	return out
}

type namedGroup struct {
	title string
	names []string
}

var topGroups = []namedGroup{
	{"everyday", []string{"files", "folders", "text"}},
	{"machine", []string{"proc", "disk", "net", "host", "svc", "logs", "pack", "auth"}},
	{"build", []string{"dev", "lang", "data", "docs"}},
	{"guide", []string{"help", "explain", "atlas", "recipe", "doctor", "ai-agent"}},
	{"setup", []string{"config", "color", "completion", "shell-init", "version"}},
}

var filesLook = []string{"list", "show", "open", "find", "map", "stat", "file", "where", "less", "more", "readlink", "realpath", "basename", "dirname"}
var filesChange = []string{"copy", "move", "make", "link", "install", "mktemp", "go", "rsync"}
var filesCareful = []string{"remove", "chmod", "chown", "chgrp"}

func topSection(name string) string {
	for _, sec := range topGroups {
		for _, n := range sec.names {
			if n == name {
				return sec.title
			}
		}
	}
	return ""
}

func childSection(parent, name string) string {
	switch parent {
	case "files":
		if listed(filesLook, name) {
			return "look"
		}
		if listed(filesChange, name) {
			return "change"
		}
		if listed(filesCareful, name) || nameDanger(name) {
			return "careful"
		}
		return "change"
	case "proc":
		if nameDanger(name) {
			return "careful"
		}
		return "look"
	default:
		if nameDanger(name) {
			return "careful"
		}
		return "commands"
	}
}

func listed(names []string, name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

func takeNames(rows []menuRow, order []string, used map[string]bool) []menuRow {
	var got []menuRow
	for _, name := range order {
		for _, row := range rows {
			if rowUsed(row, used) || !rowHas(row, name) {
				continue
			}
			got = append(got, row)
			markUsed(row, used)
			break
		}
	}
	return got
}

func rowHas(row menuRow, name string) bool {
	for _, n := range row.names {
		if n == name {
			return true
		}
	}
	return false
}

func rowUsed(row menuRow, used map[string]bool) bool {
	for _, n := range row.names {
		if used[n] {
			return true
		}
	}
	return false
}

func markUsed(row menuRow, used map[string]bool) {
	for _, n := range row.names {
		used[n] = true
	}
}

func nameDanger(name string) bool {
	on, _ := warn.Name(name)
	return on
}

func rowDanger(row menuRow) bool {
	for _, name := range row.names {
		if nameDanger(name) {
			return true
		}
	}
	return false
}

func rowsDanger(rows []menuRow) bool {
	for _, row := range rows {
		if rowDanger(row) {
			return true
		}
	}
	return false
}

func writeSectionRows(b *strings.Builder, rows []menuRow, opt Options) {
	for _, line := range sectionDisplays(rows, opt) {
		fmt.Fprintln(b, line)
	}
}

// CompletionRow is one tab-completion match. Display is the same four columns
// as the menu. Section is only the header; it is not part of Display.
type CompletionRow struct {
	Section string
	Name    string
	Display string
}

// CompletionRows lists one insertable command per line, in menu section order.
func CompletionRows(parent string, nodes []catalog.Node, top bool, opt Options) []CompletionRow {
	rows := make([]menuRow, 0, len(nodes))
	for i := range nodes {
		n := nodes[i]
		rows = append(rows, menuRow{names: []string{n.Name}, title: n.Title, node: n, top: top})
	}
	var out []CompletionRow
	for _, sec := range menuSections(parent, rows, top) {
		lines := sectionDisplays(sec.rows, opt)
		for i, row := range sec.rows {
			name := ""
			if len(row.names) > 0 {
				name = row.names[0]
			}
			out = append(out, CompletionRow{Section: sec.title, Name: name, Display: lines[i]})
		}
	}
	return out
}

func sectionDisplays(rows []menuRow, opt Options) []string {
	if len(rows) == 0 {
		return nil
	}
	type col struct {
		label, what, syntax, example string
		danger                       bool
	}
	cols := make([]col, len(rows))
	nameW := 0
	descNatural := 0
	for i, row := range rows {
		label := strings.Join(row.names, "  ")
		danger := rowDanger(row)
		if danger {
			label = "! " + label
		}
		what := menuWhat(row)
		syntax, example := menuSyntax(row)
		cols[i] = col{label: label, what: what, syntax: syntax, example: example, danger: danger}
		if n := utf8.RuneCountInString(label); n > nameW {
			nameW = n
		}
		if n := utf8.RuneCountInString(what); n > descNatural {
			descNatural = n
		}
	}
	tailOf := func(syntax, example string) int {
		return utf8.RuneCountInString(syntax) + 2 + utf8.RuneCountInString("e.g. "+example)
	}
	maxTail := func() int {
		n := 0
		for _, c := range cols {
			if t := tailOf(c.syntax, c.example); t > n {
				n = t
			}
		}
		return n
	}
	// Align the syntax column inside the section. A 100-column terminal should
	// not wrap: drop optional flags, then long alternation lists, and only then
	// shorten a description. Examples stay whole.
	const limit = 100
	for nameW+2+descNatural+2+maxTail() > limit {
		progressed := false
		for i := range cols {
			if nameW+2+descNatural+2+tailOf(cols[i].syntax, cols[i].example) <= limit {
				continue
			}
			if next, ok := shortenSyntax(cols[i].syntax); ok {
				cols[i].syntax = next
				progressed = true
			}
		}
		if !progressed {
			break
		}
	}
	descW := descNatural
	if budget := limit - nameW - 2 - 2 - maxTail(); budget < descW {
		if budget < 1 {
			descW = 0
		} else {
			descW = budget
		}
	}
	lines := make([]string, len(cols))
	for i, c := range cols {
		what := shortenWords(c.what, descW)
		lines[i] = menuColumns(c.label, what, c.syntax, c.example, c.danger, nameW, descW, opt)
	}
	return lines
}

func menuColumns(label, what, syntax, example string, danger bool, nameW, descW int, opt Options) string {
	name := padWidth(label, nameW)
	desc := padWidth(what, descW)
	var b strings.Builder
	b.WriteString(paintName(name, danger, opt))
	b.WriteString("  ")
	b.WriteString(desc)
	b.WriteString("  ")
	b.WriteString(dimText(syntax, opt))
	b.WriteString("  ")
	b.WriteString(paintExample("e.g. "+example, opt))
	return b.String()
}

func paintName(text string, danger bool, opt Options) string {
	if !opt.Color || text == "" {
		return text
	}
	role := style.Accent
	if danger {
		role = style.Hidden
	}
	return style.Paint(role, text, style.Auto, true)
}

func paintExample(text string, opt Options) string {
	if !opt.Color || text == "" {
		return text
	}
	// Teal, and dim so the bold violet name stays the loudest thing on the row.
	return "\x1b[2m" + style.Paint(style.Example, text, style.Auto, false)
}

func dimText(text string, opt Options) string {
	if !opt.Color || text == "" {
		return text
	}
	return "\x1b[2m" + text + "\x1b[22m"
}

func padWidth(s string, width int) string {
	n := utf8.RuneCountInString(s)
	if n >= width {
		return s
	}
	return s + strings.Repeat(" ", width-n)
}

func shortenWords(s string, width int) string {
	if utf8.RuneCountInString(s) <= width {
		return s
	}
	if width < 1 {
		return ""
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return ""
	}
	prefix := trimWeakEdge(fitWords(words, width, false), false)
	suffix := trimWeakEdge(fitWords(words, width, true), true)
	if strings.Contains(suffix, ",") {
		suffix = ""
	}
	if utf8.RuneCountInString(suffix) > utf8.RuneCountInString(prefix) {
		return suffix
	}
	last := words[len(words)-1]
	if suffix != "" && utf8.RuneCountInString(suffix) == utf8.RuneCountInString(prefix) &&
		strings.Contains(suffix, last) && !strings.Contains(prefix, last) {
		return suffix
	}
	if prefix != "" {
		return prefix
	}
	if suffix != "" {
		return suffix
	}
	return words[0]
}

func fitWords(words []string, width int, fromEnd bool) string {
	cur := ""
	if fromEnd {
		for i := len(words) - 1; i >= 0; i-- {
			next := words[i]
			if cur != "" {
				next = words[i] + " " + cur
			}
			if utf8.RuneCountInString(next) > width {
				break
			}
			cur = next
		}
		return cur
	}
	for _, w := range words {
		next := w
		if cur != "" {
			next = cur + " " + w
		}
		if utf8.RuneCountInString(next) > width {
			break
		}
		cur = next
	}
	return cur
}

func trimWeakEdge(s string, leading bool) string {
	words := strings.Fields(s)
	for len(words) > 1 {
		edge := words[len(words)-1]
		if leading {
			edge = words[0]
		}
		if !weakWord(edge) {
			break
		}
		if leading {
			words = words[1:]
		} else {
			words = words[:len(words)-1]
		}
	}
	return strings.Join(words, " ")
}

func weakWord(w string) bool {
	switch strings.Trim(w, ",.") {
	case "a", "an", "the", "and", "or", "of", "to", "for", "with":
		return true
	default:
		return false
	}
}

func menuWhat(row menuRow) string {
	for _, name := range row.names {
		if on, reason := warn.Name(name); on && reason != "" {
			return reason
		}
	}
	if row.title != "" {
		return row.title
	}
	if row.node.Title != "" {
		return row.node.Title
	}
	if len(row.names) > 0 {
		return row.names[0]
	}
	return row.node.Name
}

func menuSyntax(row menuRow) (syntax, example string) {
	n := &row.node
	if row.top && groupMenu(n) {
		child := preferredChild(n)
		return "bhuppi " + n.Name, "bhuppi " + n.Name + " " + child
	}
	syntax = strings.TrimSpace(n.Shape)
	example = calmExample(n, firstExample(n))
	if syntax == "" {
		syntax = "bhuppi " + n.Name
	}
	if example == "" || unsafeExample(example) {
		example = deriveExample(syntax)
	}
	if example == "" {
		example = syntax
	}
	return syntax, example
}

// groupMenu is a top-level name whose job is to list subcommands.
// version, doctor, help, and color are commands, so they keep their own syntax.
func groupMenu(n *catalog.Node) bool {
	if n == nil || len(n.Children) == 0 {
		return false
	}
	if n.Delivery == "group" || n.Name == "ai-agent" {
		return preferredChild(n) != ""
	}
	return false
}

// preferredChild is the one subcommand a group row uses as its example.
func preferredChild(n *catalog.Node) string {
	if n == nil {
		return ""
	}
	switch n.Name {
	case "files", "proc":
		if catalog.Child(n, "list") != nil {
			return "list"
		}
	case "disk":
		if catalog.Child(n, "review") != nil {
			return "review"
		}
	case "net":
		if catalog.Child(n, "connections") != nil {
			return "connections"
		}
	case "ai-agent":
		if catalog.Child(n, "review") != nil {
			return "review"
		}
	}
	if n.Pick != "" {
		parts := n.Pick
		if i := strings.Index(parts, "—"); i >= 0 {
			parts = parts[i+len("—"):]
		}
		for _, part := range strings.Split(parts, ",") {
			name := strings.TrimSpace(part)
			if catalog.Child(n, name) != nil {
				return name
			}
		}
	}
	for i := range n.Children {
		if n.Children[i].Sparkle {
			return n.Children[i].Name
		}
	}
	if len(n.Children) > 0 {
		return n.Children[0].Name
	}
	return ""
}

func firstExample(n *catalog.Node) string {
	if n == nil {
		return ""
	}
	if s := strings.TrimSpace(n.Example); s != "" {
		return s
	}
	for _, ex := range n.Examples {
		if s := strings.TrimSpace(ex); s != "" {
			return s
		}
	}
	return ""
}

// calmExample keeps catalog examples, and swaps a dummy file operand on a
// delete for notes.txt. Directory removes stay on their catalog path.
func calmExample(n *catalog.Node, example string) string {
	if unsafeExample(example) {
		return ""
	}
	on, reason := warn.Name(n.Name)
	if !on || reason != "deletes files" || strings.Contains(example, "folders") {
		return example
	}
	fields := strings.Fields(example)
	if len(fields) == 0 {
		return example
	}
	switch fields[len(fields)-1] {
	case "a", "b", "f", "empty-dir", ".":
		fields[len(fields)-1] = "notes.txt"
		return strings.Join(fields, " ")
	}
	return example
}

func unsafeExample(s string) bool {
	low := strings.ToLower(s)
	for _, bad := range []string{
		"rm -rf /", "rm -fr /", "diskutil erase", "erasedisk", "mkfs", "wipefs",
	} {
		if strings.Contains(low, bad) {
			return true
		}
	}
	return false
}

func deriveExample(syntax string) string {
	if strings.TrimSpace(syntax) == "" {
		return ""
	}
	var out []string
	for _, f := range strings.Fields(syntax) {
		if strings.HasPrefix(f, "[") {
			continue
		}
		if v, ok := placeholderExample[f]; ok {
			out = append(out, v)
			continue
		}
		if strings.Contains(f, "|") && !strings.Contains(f, "/") && !strings.Contains(f, ".") {
			out = append(out, "notes.txt")
			continue
		}
		out = append(out, f)
	}
	return strings.Join(out, " ")
}

var placeholderExample = map[string]string{
	"PATH":    "notes.txt",
	"FILE":    "notes.txt",
	"SRC":     "notes.txt",
	"DST":     "notes.txt",
	"DIR":     "tmp",
	"MODE":    "644",
	"PID":     "123",
	"USER":    "user",
	"OWNER":   "user",
	"GROUP":   "staff",
	"NAME":    "notes",
	"URL":     "https://example.com",
	"TEXT":    "TODO",
	"PAT":     "TODO",
	"KEYWORD": "ssh",
	"DISK":    "disk0",
}

// shortenSyntax drops the quiet syntax toward a shorter form when a row would
// wrap. The example is left alone. One step per call so a later pass can stop
// once the section fits.
func shortenSyntax(s string) (string, bool) {
	if strings.Contains(s, "--") {
		if t := trimOptionalFlags(s); t != "" && t != s {
			return t, true
		}
	}
	if c := compactAlt(s); c != "" && c != s {
		return c, true
	}
	return s, false
}

// compactAlt turns a long a|b|c list, or "A | B | C", into first|….
// A short pair such as DIR|FILE stays, so the syntax column still shows both.
func compactAlt(s string) string {
	fields := strings.Fields(s)
	var out []string
	changed := false
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		if changed && (f == "…" || f == "...") {
			continue
		}
		if i+1 < len(fields) && fields[i+1] == "|" {
			out = append(out, f+"|…")
			return strings.Join(out, " ")
		}
		pipes := strings.Count(f, "|")
		if pipes > 0 && !strings.HasSuffix(f, "|…") && !strings.HasSuffix(f, "|...") &&
			(pipes >= 2 || utf8.RuneCountInString(f) > 12) {
			first, _, _ := strings.Cut(f, "|")
			if first != "" {
				if strings.HasSuffix(f, "]") {
					first += "|…]"
				} else {
					first += "|…"
				}
				out = append(out, first)
				changed = true
				continue
			}
		}
		out = append(out, f)
	}
	if !changed {
		return s
	}
	return strings.Join(out, " ")
}

func trimOptionalFlags(syntax string) string {
	var b strings.Builder
	for i := 0; i < len(syntax); {
		if syntax[i] != '[' {
			b.WriteByte(syntax[i])
			i++
			continue
		}
		j, ok := closeBracket(syntax, i)
		if !ok {
			b.WriteString(syntax[i:])
			break
		}
		group := syntax[i:j]
		if !strings.Contains(group, "--") {
			b.WriteString(group)
		}
		i = j
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func closeBracket(s string, i int) (int, bool) {
	depth := 0
	for j := i; j < len(s); j++ {
		switch s[j] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return j + 1, true
			}
		}
	}
	return 0, false
}
