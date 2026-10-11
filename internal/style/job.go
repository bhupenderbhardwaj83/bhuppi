package style

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// ColorableJob is a command whose text output should be tinted after it runs.
func ColorableJob(path []string) bool {
	if len(path) < 2 {
		return false
	}
	switch path[0] {
	case "files", "folders":
		return path[1] == "list" || path[1] == "stat"
	case "proc":
		return path[1] == "stat"
	default:
		return false
	}
}

// ColorJob tints captured stdout. Binary output is returned unchanged.
func ColorJob(path []string, text string) string {
	if text == "" || strings.ContainsRune(text, 0) || !ColorableJob(path) {
		return text
	}
	switch {
	case len(path) >= 2 && path[1] == "list":
		return colorLines(text, func(ln string) string { return colorListLine(ln, Auto) })
	default:
		return colorLines(text, colorStatLine)
	}
}

func colorLines(text string, fn func(string) string) string {
	lines := strings.Split(text, "\n")
	for i, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		lines[i] = fn(ln)
	}
	return strings.Join(lines, "\n")
}

func colorListLine(ln string, ground Ground) string {
	if perm, rest, ok := splitLong(ln); ok {
		name := filenameAfterDate(rest)
		role := NameRole(filepath.Base(name), strings.HasPrefix(perm, "d"))
		head := strings.TrimSuffix(ln, name)
		if name != "" && strings.HasSuffix(ln, name) {
			return tintNumbers(head, ground) + Paint(role, name, ground, false)
		}
		return tintNumbers(ln, ground)
	}
	return colorPathLine(strings.TrimRight(ln, "\r"))
}

func colorPathLine(ln string) string {
	trim := strings.TrimSpace(ln)
	if trim == "" {
		return ln
	}
	base := filepath.Base(strings.TrimRight(trim, "/"))
	isDir := strings.HasSuffix(trim, "/")
	if !isDir {
		if st, err := os.Stat(trim); err == nil {
			isDir = st.IsDir()
		}
	}
	return Paint(NameRole(base, isDir), ln, Auto, false)
}

func colorStatLine(ln string) string {
	if q := quotedPath(ln); q != "" {
		i := strings.Index(ln, q)
		if i >= 0 {
			return tintNumbers(ln[:i], Auto) + Paint(Path, q, Auto, false) + tintNumbers(ln[i+len(q):], Auto)
		}
	}
	return tintTokens(ln, true, Auto)
}

func splitLong(ln string) (perm, rest string, ok bool) {
	fields := strings.Fields(ln)
	if len(fields) < 7 {
		return "", "", false
	}
	p := fields[0]
	if len(p) < 10 {
		return "", "", false
	}
	switch p[0] {
	case '-', 'd', 'l', 'c', 'b', 'p', 's':
	default:
		return "", "", false
	}
	for _, r := range p[1:] {
		switch r {
		case 'r', 'w', 'x', '-', 's', 'S', 't', 'T', '@', '+', '.':
		default:
			return "", "", false
		}
	}
	idx := strings.Index(ln, p)
	if idx < 0 {
		return "", "", false
	}
	return p, strings.TrimSpace(ln[idx+len(p):]), true
}

func filenameAfterDate(rest string) string {
	months := map[string]bool{
		"Jan": true, "Feb": true, "Mar": true, "Apr": true, "May": true, "Jun": true,
		"Jul": true, "Aug": true, "Sep": true, "Oct": true, "Nov": true, "Dec": true,
	}
	fields := strings.Fields(rest)
	for i, f := range fields {
		if months[f] && i+3 < len(fields) {
			// month, day, time-or-year, then the name (may contain spaces)
			day := fields[i+1]
			when := fields[i+2]
			mark := f + " " + day + " " + when
			j := strings.Index(rest, mark)
			if j < 0 {
				return fields[len(fields)-1]
			}
			return strings.TrimSpace(rest[j+len(mark):])
		}
	}
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}

func quotedPath(ln string) string {
	i := strings.Index(ln, `"`)
	if i < 0 || i+1 >= len(ln) {
		return ""
	}
	j := strings.Index(ln[i+1:], `"`)
	if j < 0 {
		return ""
	}
	return ln[i : i+1+j+1]
}

func tintNumbers(s string, ground Ground) string {
	return tintTokens(s, false, ground)
}

func tintTokens(s string, paths bool, ground Ground) string {
	var b strings.Builder
	start := 0
	flush := func(tok string) {
		if tok == "" {
			return
		}
		switch {
		case LooksNumeric(tok):
			b.WriteString(Paint(Number, tok, ground, false))
		case paths && (strings.Contains(tok, "/") || strings.HasPrefix(tok, "~")):
			b.WriteString(Paint(Path, tok, ground, false))
		default:
			b.WriteString(tok)
		}
	}
	for i, r := range s {
		if unicode.IsSpace(r) {
			flush(s[start:i])
			b.WriteRune(r)
			start = i + len(string(r))
		}
	}
	flush(s[start:])
	return b.String()
}

// ColorDoctor tints present/missing in the doctor table.
func ColorDoctor(text string) string {
	lines := strings.Split(text, "\n")
	for i, ln := range lines {
		fields := strings.Fields(ln)
		if len(fields) < 2 {
			if strings.HasPrefix(ln, "WHAT:") || strings.HasPrefix(ln, "WHEN:") || strings.HasPrefix(ln, "RULE:") {
				label, rest, _ := strings.Cut(ln, " ")
				lines[i] = Paint(Label, label, Auto, true) + " " + rest
			}
			continue
		}
		state := fields[1]
		if state != "present" && state != "missing" {
			continue
		}
		role := File
		if state == "missing" {
			role = Hidden
		}
		tool := fields[0]
		// Recolor the first two columns and leave the note.
		head := ln
		ti := strings.Index(ln, tool)
		si := strings.Index(ln, state)
		if ti < 0 || si < 0 || si < ti {
			continue
		}
		lines[i] = ln[:ti] + Paint(Accent, tool, Auto, false) + ln[ti+len(tool):si] + Paint(role, state, Auto, false) + ln[si+len(state):]
		_ = head
	}
	return strings.Join(lines, "\n")
}

// ColorMissing paints the first line of a missing-binary message red.
func ColorMissing(text string) string {
	line, rest, ok := strings.Cut(text, "\n")
	painted := Paint(Hidden, line, Auto, false)
	if !ok {
		return painted
	}
	return painted + "\n" + rest
}
