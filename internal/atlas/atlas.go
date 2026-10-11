// Package atlas is the long command guide extracted from the owned epub.
// The JSON is embedded; bhuppi does not read the epub at runtime.
package atlas

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/bhupenderbhardwaj/bhuppi/internal/style"
)

//go:embed data.json
var raw []byte

type book struct {
	Commands []Command `json:"commands"`
}

// Command is one chapter.
type Command struct {
	Name        string `json:"name"`
	Syntax      string `json:"syntax"`
	Description string `json:"description"`
	Anatomy     []Item `json:"anatomy"`
	Origins     []Item `json:"origins"`
	Cases       []Case `json:"cases"`
}

// Item is an anatomy or origins bullet.
type Item struct {
	Label string `json:"label"`
	Text  string `json:"text"`
}

// Case is one numbered use case from the book.
type Case struct {
	Title   string   `json:"title"`
	Command string   `json:"command"`
	What    string   `json:"what"`
	Parts   []string `json:"parts"`
	Note    string   `json:"note"`
}

var loaded book

func init() {
	if err := json.Unmarshal(raw, &loaded); err != nil {
		panic(err)
	}
}

// Count is how many commands the book documents.
func Count() int { return len(loaded.Commands) }

// Has reports whether the book has this command name.
func Has(name string) bool {
	return find(name) != nil
}

func find(name string) *Command {
	q := strings.ToLower(name)
	for i := range loaded.Commands {
		if strings.ToLower(loaded.Commands[i].Name) == q {
			return &loaded.Commands[i]
		}
	}
	return nil
}

// RenderIndex is the searchable summary. q filters names when it is not empty
// and not an exact command.
func RenderIndex(color bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s — %d commands\n", paint(style.Accent, "bhuppi atlas", color, true), Count())
	b.WriteString("One line each. bhuppi atlas NAME opens the guide. bhuppi atlas NAME SECTION jumps.\n\n")
	for i := range loaded.Commands {
		c := &loaded.Commands[i]
		desc := c.Description
		if len(desc) > 72 {
			desc = desc[:72] + "…"
		}
		fmt.Fprintf(&b, "  %-14s %s\n", paint(style.Accent, c.Name, color, false), desc)
	}
	return b.String()
}

// Render writes one command, or one section when section is set.
// ok is false when the name is missing; the string is then the closest-names note.
func Render(name, section string, color bool) (string, bool) {
	c := find(name)
	if c == nil {
		return missing(name, color), false
	}
	if section != "" {
		if body, ok := renderSection(c, section, color); ok {
			return body, true
		}
	}
	return renderFull(c, color), true
}

func renderFull(c *Command, color bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", paint(style.Accent, c.Name, color, true))
	if c.Syntax != "" {
		fmt.Fprintf(&b, "%s  %s\n", paint(style.Label, "Syntax", color, true), tintSyntax(c.Syntax, c.Name, color))
	}
	b.WriteByte('\n')
	fmt.Fprintf(&b, "%s\n", paint(style.Accent, "Description", color, true))
	b.WriteString(c.Description)
	b.WriteString("\n\n")
	if len(c.Anatomy) > 0 {
		fmt.Fprintf(&b, "%s\n", paint(style.Accent, "Anatomy", color, true))
		writeItems(&b, c.Anatomy, color)
		b.WriteByte('\n')
	}
	if len(c.Origins) > 0 {
		fmt.Fprintf(&b, "%s\n", paint(style.Accent, "Origins", color, true))
		writeItems(&b, c.Origins, color)
		b.WriteByte('\n')
	}
	for _, uc := range c.Cases {
		fmt.Fprintf(&b, "%s\n", paint(style.Accent, uc.Title, color, true))
		if uc.Command != "" {
			fmt.Fprintf(&b, "  %s\n", paint(style.Example, uc.Command, color, false))
		}
		if uc.What != "" {
			fmt.Fprintf(&b, "  %s\n", uc.What)
		}
		for _, p := range uc.Parts {
			fmt.Fprintf(&b, "  · %s\n", tintLoose(p, color))
		}
		if uc.Note != "" {
			fmt.Fprintf(&b, "  %s\n", paint(style.Hidden, uc.Note, color, false))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func renderSection(c *Command, q string, color bool) (string, bool) {
	needle := strings.ToLower(strings.TrimSpace(q))
	if needle == "" {
		return "", false
	}
	var b strings.Builder
	writeHead := func(title string) {
		fmt.Fprintf(&b, "%s — %s\n", paint(style.Accent, c.Name, color, true), paint(style.Accent, title, color, true))
	}
	if needle == "syntax" && c.Syntax != "" {
		writeHead("Syntax")
		fmt.Fprintf(&b, "%s\n", tintSyntax(c.Syntax, c.Name, color))
		fmt.Fprintf(&b, "more: bhuppi atlas %s\n", c.Name)
		return b.String(), true
	}
	if needle == "description" && c.Description != "" {
		writeHead("Description")
		b.WriteString(c.Description)
		b.WriteByte('\n')
		fmt.Fprintf(&b, "more: bhuppi atlas %s\n", c.Name)
		return b.String(), true
	}
	if item, ok := matchItem(c.Anatomy, needle); ok {
		writeHead(item.Label)
		fmt.Fprintf(&b, "%s\n", item.Text)
		fmt.Fprintf(&b, "more: bhuppi atlas %s\n", c.Name)
		return b.String(), true
	}
	if item, ok := matchItem(c.Origins, needle); ok {
		writeHead(item.Label)
		fmt.Fprintf(&b, "%s\n", item.Text)
		fmt.Fprintf(&b, "more: bhuppi atlas %s\n", c.Name)
		return b.String(), true
	}
	if needle == "anatomy" && len(c.Anatomy) > 0 {
		writeHead("Anatomy")
		writeItems(&b, c.Anatomy, color)
		fmt.Fprintf(&b, "more: bhuppi atlas %s\n", c.Name)
		return b.String(), true
	}
	if needle == "origins" && len(c.Origins) > 0 {
		writeHead("Origins")
		writeItems(&b, c.Origins, color)
		fmt.Fprintf(&b, "more: bhuppi atlas %s\n", c.Name)
		return b.String(), true
	}
	for _, uc := range c.Cases {
		title := strings.ToLower(uc.Title)
		if strings.Contains(title, needle) || slug(uc.Title) == slug(needle) {
			writeHead(uc.Title)
			if uc.Command != "" {
				fmt.Fprintf(&b, "  %s\n", paint(style.Example, uc.Command, color, false))
			}
			if uc.What != "" {
				fmt.Fprintf(&b, "  %s\n", uc.What)
			}
			for _, p := range uc.Parts {
				fmt.Fprintf(&b, "  · %s\n", p)
			}
			fmt.Fprintf(&b, "more: bhuppi atlas %s\n", c.Name)
			return b.String(), true
		}
	}
	return "", false
}

func matchItem(items []Item, needle string) (Item, bool) {
	for _, it := range items {
		l := strings.ToLower(it.Label)
		if l == needle || slug(l) == slug(needle) {
			return it, true
		}
	}
	for _, it := range items {
		if strings.Contains(strings.ToLower(it.Label), needle) {
			return it, true
		}
	}
	return Item{}, false
}

func writeItems(b *strings.Builder, items []Item, color bool) {
	for _, it := range items {
		label := it.Label
		if label == "" {
			fmt.Fprintf(b, "  · %s\n", it.Text)
			continue
		}
		fmt.Fprintf(b, "  %s  %s\n", paint(style.Label, label, color, true), it.Text)
	}
}

func missing(name string, color bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "no atlas page for %q\n", name)
	names := closest(name, 8)
	if len(names) > 0 {
		b.WriteString("closest:")
		for _, n := range names {
			fmt.Fprintf(&b, " %s", paint(style.Accent, n, color, false))
		}
		b.WriteByte('\n')
	}
	b.WriteString("see: bhuppi atlas\n")
	return b.String()
}

func closest(q string, n int) []string {
	q = strings.ToLower(q)
	type pair struct {
		name  string
		score int
	}
	var all []pair
	for i := range loaded.Commands {
		name := loaded.Commands[i].Name
		low := strings.ToLower(name)
		score := -dist(low, q)
		if strings.HasPrefix(low, q) {
			score += 80
		} else if strings.Contains(low, q) {
			score += 40
		}
		all = append(all, pair{name, score})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].score == all[j].score {
			return all[i].name < all[j].name
		}
		return all[i].score > all[j].score
	})
	if n > len(all) {
		n = len(all)
	}
	out := make([]string, 0, n)
	for _, p := range all[:n] {
		out = append(out, p.name)
	}
	return out
}

func dist(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(cur[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

func slug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func paint(role style.Role, text string, color, bold bool) string {
	if !color || text == "" {
		return text
	}
	return style.Paint(role, text, style.Auto, bold)
}

func tintSyntax(syntax, cmd string, color bool) string {
	if !color {
		return syntax
	}
	fields := strings.Fields(syntax)
	if len(fields) == 0 {
		return syntax
	}
	var b strings.Builder
	for i, f := range fields {
		if i > 0 {
			b.WriteByte(' ')
		}
		switch {
		case i == 0 || strings.EqualFold(f, cmd):
			b.WriteString(style.Paint(style.File, f, style.Auto, false))
		case strings.Contains(f, "/") || strings.HasPrefix(f, "~"):
			b.WriteString(style.Paint(style.Path, f, style.Auto, false))
		case style.LooksNumeric(f):
			b.WriteString(style.Paint(style.Number, f, style.Auto, false))
		default:
			b.WriteString(f)
		}
	}
	return b.String()
}

func tintLoose(s string, color bool) string {
	if !color {
		return s
	}
	fields := strings.Fields(s)
	// Keep the original spacing by coloring only obvious numbers and paths in a copy.
	out := s
	// Walk fields and replace the first occurrence. Safer to rebuild with Fields
	// when the line has no significant double spaces.
	var b strings.Builder
	for i, f := range fields {
		if i > 0 {
			b.WriteByte(' ')
		}
		switch {
		case strings.Contains(f, "/") || strings.HasPrefix(f, "~"):
			b.WriteString(style.Paint(style.Path, f, style.Auto, false))
		case style.LooksNumeric(strings.Trim(f, "`.,")):
			b.WriteString(style.Paint(style.Number, f, style.Auto, false))
		default:
			b.WriteString(f)
		}
	}
	if strings.Join(fields, " ") == strings.Join(strings.Fields(s), " ") {
		return b.String()
	}
	return out
}
