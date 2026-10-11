package help

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bhupenderbhardwaj/bhuppi/internal/catalog"
	"github.com/bhupenderbhardwaj/bhuppi/internal/style"
)

const shapeLegend = "Shape: UPPERCASE = you supply it · [brackets] = optional · a|b = pick one"

// Options controls help rendering. OS changes which example is emphasized.
// It does not change what --dry-run will execute.
type Options struct {
	Plain bool
	Color bool
	OS    string // linux, darwin, windows, or empty for the host
}

// ColorEnabled reports whether ANSI color should be used for this writer.
func ColorEnabled(w io.Writer, plain bool) bool {
	if !style.Allow(plain) {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}

// Render prints overview when path is empty, otherwise that page.
func Render(cat *catalog.Catalog, path []string, opt Options) (string, error) {
	if len(path) == 0 {
		return overview(cat, opt), nil
	}
	if path[0] == "recipe" {
		return renderRecipe(cat, path[1:], opt)
	}
	if m := cat.Meta(path[0]); m != nil && len(path) == 1 {
		return leaf(cat, m, "bhuppi "+m.Name, opt), nil
	}
	n, title, err := walk(cat, path)
	if err != nil {
		return "", err
	}
	if len(n.Children) > 0 {
		return groupPage(title, n, opt), nil
	}
	return leaf(cat, n, title, opt), nil
}

func walk(cat *catalog.Catalog, path []string) (*catalog.Node, string, error) {
	g := cat.Group(path[0])
	if g == nil {
		return nil, "", fmt.Errorf("no help page for %q\nsee: bhuppi help", strings.Join(path, " "))
	}
	n := g
	title := "bhuppi " + g.Name
	for _, part := range path[1:] {
		ch := catalog.Child(n, part)
		if ch == nil {
			return nil, "", fmt.Errorf("no help page for %q\nsee: bhuppi help %s", strings.Join(path, " "), n.Name)
		}
		n = ch
		title += " " + part
	}
	return n, title, nil
}

func overview(cat *catalog.Catalog, opt Options) string {
	var b strings.Builder
	id := cat.Identity
	if opt.Plain {
		fmt.Fprintf(&b, "%s\n%s\n", id.Wordmark, id.Tagline)
	} else {
		fmt.Fprintf(&b, "%s\n", identityArt(id.Wordmark, id.Tagline, opt.Color))
	}
	b.WriteByte('\n')
	fmt.Fprintf(&b, "%s\n\n", id.Rule)
	writeLabel(&b, "WHAT:", id.What, opt)
	writeLabel(&b, "WHEN:", id.When, opt)
	writeLabel(&b, "RULE:", id.Rule, opt)
	b.WriteByte('\n')
	b.WriteString(paint(style.Accent, "PICK ONE", opt, true) + "\n")
	for i := range cat.Groups {
		g := &cat.Groups[i]
		if !g.Sparkle {
			continue
		}
		fmt.Fprintf(&b, "  %-8s ✨ %s\n", g.Name, g.Pick)
	}
	b.WriteByte('\n')
	b.WriteString(shapeLegend)
	b.WriteString("\n\n")
	for i := range cat.Groups {
		g := &cat.Groups[i]
		writeSection(&b, sectionTitle(g), opt)
		for j := range g.Children {
			ch := &g.Children[j]
			writeRow(&b, ch.Name, ch.Sparkle, ch.Shape, ch.Example, ch.Desc, nil, opt)
		}
		b.WriteByte('\n')
	}
	writeSection(&b, "SETUP — this atlas", opt)
	for i := range cat.MetaPages {
		m := &cat.MetaPages[i]
		writeRow(&b, m.Name, false, m.Shape, m.Example, m.Desc, nil, opt)
	}
	b.WriteByte('\n')
	writeNotThis(&b, id.NotThis, opt)
	fmt.Fprintf(&b, "more: %s\n", id.More)
	b.WriteByte('\n')
	writeFooters(&b, opt.Plain, opt.Color)
	return b.String()
}

func groupPage(title string, n *catalog.Node, opt Options) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s — %s\n", title, n.Title)
	writeContract(&b, n, opt)
	b.WriteByte('\n')
	b.WriteString(shapeLegend)
	b.WriteString("\n\n")
	sections := map[string][]*catalog.Node{}
	var order []string
	for i := range n.Children {
		ch := &n.Children[i]
		sec := ch.Section
		if sec == "" {
			sec = "Commands"
		}
		if _, ok := sections[sec]; !ok {
			order = append(order, sec)
		}
		sections[sec] = append(sections[sec], ch)
	}
	for _, sec := range order {
		fmt.Fprintf(&b, "%s\n", paint(style.Accent, strings.ToUpper(sec), opt, true))
		for _, ch := range sections[sec] {
			extras := ch.Examples
			writeRow(&b, ch.Name, ch.Sparkle, ch.Shape, ch.Example, ch.Desc, extras, opt)
		}
		b.WriteByte('\n')
	}
	writeOSNotes(&b, n, opt.OS)
	writeNotThis(&b, n.NotThis, opt)
	fmt.Fprintf(&b, "TRY NEXT: %s\n", n.More)
	fmt.Fprintf(&b, "more: %s\n\n", n.More)
	writeFooters(&b, opt.Plain, opt.Color)
	return b.String()
}

func leaf(cat *catalog.Catalog, n *catalog.Node, title string, opt Options) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s — %s\n", title, n.Title)
	writeContract(&b, n, opt)
	b.WriteByte('\n')
	b.WriteString(shapeLegend)
	b.WriteString("\n\n")
	writeRow(&b, n.Name, n.Sparkle, n.Shape, n.Example, n.Desc, n.Examples, opt)
	if runs := runsFor(n, opt.OS); runs != "" {
		fmt.Fprintf(&b, "  runs: %s\n", runs)
	}
	writeOSNotes(&b, n, opt.OS)
	if n.Docs != "" {
		fmt.Fprintf(&b, "  docs: %s\n", n.Docs)
	}
	b.WriteByte('\n')
	writeNotThis(&b, n.NotThis, opt)
	fmt.Fprintf(&b, "TRY NEXT: %s\n", n.More)
	fmt.Fprintf(&b, "more: %s\n\n", n.More)
	writeFooters(&b, opt.Plain, opt.Color)
	_ = cat
	return b.String()
}

func renderRecipe(cat *catalog.Catalog, rest []string, opt Options) (string, error) {
	if len(rest) == 0 {
		page := &catalog.Node{
			Name:    "recipe",
			Title:   "safe pipelines",
			What:    "Named, copy-safe compositions. Each one prints its pipeline. --run executes it.",
			When:    "You want a known-safe pipeline and you want to read it before it runs.",
			Rule:    "Recipes do not assemble shell from your text. Placeholders are filled as single arguments.",
			Desc:    "Without --run, bhuppi prints the pipeline and exits. --dry-run prints the same line.",
			NotThis: "Not an exploit recipe. Not a hidden rm -rf. Not a scanner.",
			More:    "bhuppi recipe disk-hogs · bhuppi help recipe disk-hogs",
			Shape:   "bhuppi recipe NAME [--run]",
			Example: "bhuppi recipe disk-hogs",
		}
		var b strings.Builder
		fmt.Fprintf(&b, "bhuppi recipe — %s\n", page.Title)
		writeContract(&b, page, opt)
		b.WriteByte('\n')
		b.WriteString(shapeLegend)
		b.WriteString("\n\n")
		for i := range cat.Recipes {
			r := &cat.Recipes[i]
			writeRow(&b, r.Name, false, r.Shape, r.Example, r.Desc, nil, opt)
		}
		b.WriteByte('\n')
		writeNotThis(&b, page.NotThis, opt)
		fmt.Fprintf(&b, "TRY NEXT: %s\nmore: %s\n\n", page.More, page.More)
		writeFooters(&b, opt.Plain, opt.Color)
		return b.String(), nil
	}
	r := cat.Recipe(rest[0])
	if r == nil {
		return "", fmt.Errorf("no recipe %q\nsee: bhuppi help recipe", rest[0])
	}
	return leaf(cat, r, "bhuppi recipe "+r.Name, opt), nil
}

func writeContract(b *strings.Builder, n *catalog.Node, opt Options) {
	writeLabel(b, "WHAT:", n.What, opt)
	writeLabel(b, "WHEN:", n.When, opt)
	writeLabel(b, "RULE:", n.Rule, opt)
}

func writeLabel(b *strings.Builder, label, text string, opt Options) {
	width := 88 - len(label) - 1
	if width < 24 {
		width = 24
	}
	lines := wrap(text, width)
	shown := paint(style.Label, label, opt, true)
	if len(lines) == 0 {
		fmt.Fprintf(b, "%s\n", shown)
		return
	}
	fmt.Fprintf(b, "%s %s\n", shown, lines[0])
	pad := strings.Repeat(" ", len(label)+1)
	for _, ln := range lines[1:] {
		fmt.Fprintf(b, "%s%s\n", pad, ln)
	}
}

func writeNotThis(b *strings.Builder, text string, opt Options) {
	b.WriteString(paint(style.Hidden, "NOT THIS", opt, true) + "\n")
	for _, ln := range wrap(text, 84) {
		fmt.Fprintf(b, "  %s\n", ln)
	}
}

func writeOSNotes(b *strings.Builder, n *catalog.Node, osName string) {
	if osName != "" {
		fmt.Fprintf(b, "  rendered for %s\n", osName)
	}
	if n.AlsoLinux != "" {
		fmt.Fprintf(b, "  also linux: %s\n", n.AlsoLinux)
	}
	if n.AlsoMacos != "" {
		fmt.Fprintf(b, "  also macos: %s\n", n.AlsoMacos)
	}
}

func runsFor(n *catalog.Node, osName string) string {
	switch osName {
	case "linux":
		if n.RunsLinux != "" {
			return n.RunsLinux
		}
	case "darwin":
		if n.RunsDarwin != "" {
			return n.RunsDarwin
		}
	case "windows":
		if n.RunsWindows != "" {
			return n.RunsWindows
		}
	}
	if n.Runs != "" {
		return n.Runs
	}
	switch osName {
	case "linux":
		return n.RunsLinux
	case "darwin":
		return n.RunsDarwin
	case "windows":
		return n.RunsWindows
	}
	return ""
}

func writeRow(b *strings.Builder, name string, sparkle bool, shape, example, desc string, extras []string, opt Options) {
	mark := ""
	if sparkle {
		mark = "✨ "
	}
	left := fmt.Sprintf("  %-12s %s%s", name, mark, shape)
	pad := 58 - len(left)
	if pad < 1 {
		pad = 1
	}
	shown := left
	if opt.Color {
		shown = fmt.Sprintf("  %-12s %s%s", paint(style.Accent, name, opt, false), mark, shape)
	}
	fmt.Fprintf(b, "%s%s|  %s\n", shown, strings.Repeat(" ", pad), paint(style.Example, "e.g. "+example, opt, false))
	for _, ln := range wrap(desc, 70) {
		fmt.Fprintf(b, "                  %s\n", ln)
	}
	for _, ex := range extras {
		fmt.Fprintf(b, "    %s\n", paint(style.Example, ex, opt, false))
	}
}

func writeSection(b *strings.Builder, title string, opt Options) {
	fmt.Fprintf(b, "%s\n", paint(style.Accent, title, opt, true))
}

func sectionTitle(g *catalog.Node) string {
	name := strings.ToUpper(g.Name)
	if g.Sparkle {
		return "✨ " + name + " — " + g.Title
	}
	return name + " — " + g.Title
}

func writeFooters(b *strings.Builder, plain, color bool) {
	sections := []struct {
		title string
		lines []string
	}{
		{"PROGRESSIVE HELP", []string{
			"bhuppi help                      this overview (static; no model)",
			"bhuppi help GROUP                group contract + child commands",
			"bhuppi help GROUP CMD            leaf or language-mode",
			"bhuppi help GROUP CMD SUB        one feature page",
			"bhuppi GROUP --help              same page as bhuppi help GROUP",
			"bhuppi GROUP CMD --help          same page as bhuppi help GROUP CMD",
			"  e.g. bhuppi help files · bhuppi help text awk · bhuppi help text awk fields",
		}},
		{"ALIASES", []string{
			"h = help",
			"files go = folders go = cd, via the shell hook (a child cannot chdir you)",
			"no bp binary",
		}},
		{"GLOBAL OPTIONS", []string{
			"--dry-run           print argv, do not exec",
			"--os linux|darwin   render the other OS's examples (does not change exec)",
			"--plain             no color for this one command",
			"bhuppi color off|on  color in later shells (default on)",
			"-h / --help         the same renderer as bhuppi help …",
			"Enter               one more line, when the output is taller than the screen",
			"Space               next page",
			"q                   stop",
		}},
	}
	for i, sec := range sections {
		title := sec.title
		if color && !plain {
			title = style.Paint(style.Accent, sec.title, style.Auto, true)
		}
		if plain {
			fmt.Fprintf(b, "%s\n", title)
			for _, ln := range sec.lines {
				fmt.Fprintf(b, "  %s\n", ln)
			}
			continue
		}
		fmt.Fprintf(b, "┌ %s\n", title)
		for _, ln := range sec.lines {
			fmt.Fprintf(b, "│ %s\n", ln)
		}
		if i == len(sections)-1 {
			b.WriteString("└\n")
		}
	}
}

func identityArt(word, tag string, color bool) string {
	lines := []string{
		"╭──────────────╮",
		"│      │       │",
		"│    ┌─┴─┐     │",
		"│    │   │     │",
		"╰──────────╮   │",
		"           ╰───╯",
	}
	lines[2] = lines[2] + "     " + word
	lines[3] = lines[3] + "     " + tag
	art := strings.Join(lines, "\n")
	if !color {
		return art
	}
	// The notch stays the bhuppi mark, not a face.
	return style.Paint(style.Accent, art, style.Auto, false)
}

func paint(role style.Role, text string, opt Options, bold bool) string {
	if !opt.Color || text == "" {
		return text
	}
	return style.Paint(role, text, style.Auto, bold)
}

func wrap(text string, width int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if width < 20 {
		width = 20
	}
	var lines []string
	for _, para := range strings.Split(text, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			continue
		}
		cur := words[0]
		for _, w := range words[1:] {
			if len(cur)+1+len(w) > width {
				lines = append(lines, cur)
				cur = w
				continue
			}
			cur += " " + w
		}
		lines = append(lines, cur)
	}
	return lines
}
