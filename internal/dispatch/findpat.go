package dispatch

import (
	"fmt"
	"strings"

	"github.com/bhupenderbhardwaj/bhuppi/internal/catalog"
)

// nameFlags are the simplified name and text shapes. --regex is the raw pattern.
type nameFlags struct {
	contains, startswith, endswith string
	anyof, oneof, regex            string
	hasContains                    bool
	hasStart                       bool
	hasEnd                         bool
	sensitive                      bool
	insensitive                    bool
	word                           bool
	invert                         bool
	line                           bool
	set                            bool
}

func (f nameFlags) validate() error {
	simple := f.hasContains || f.hasStart || f.hasEnd || f.anyof != "" || f.oneof != ""
	if f.regex != "" && simple {
		return fmt.Errorf("--regex is the raw pattern; do not mix it with --contains, --startswith, --endswith, --anyof, or --oneof")
	}
	n := 0
	if f.hasContains {
		n++
	}
	if f.hasStart {
		n++
	}
	if f.hasEnd {
		n++
	}
	if n > 1 {
		return fmt.Errorf("use one of --contains, --startswith, --endswith")
	}
	if f.oneof != "" && f.anyof != "" {
		return fmt.Errorf("use --oneof or --anyof, not both")
	}
	if f.oneof != "" && (f.contains != "" || f.startswith != "" || f.endswith != "") {
		return fmt.Errorf("put the anchor on --oneof by passing the flag with no extra text, or use one pattern")
	}
	if f.sensitive && f.insensitive {
		return fmt.Errorf("use one of --sensitivecase or --insensitivecase")
	}
	return nil
}

func (f nameFlags) hasPattern() bool {
	return f.hasContains || f.hasStart || f.hasEnd || f.anyof != "" || f.oneof != "" || f.regex != ""
}

func (f nameFlags) namePrim() string {
	if f.insensitive {
		return "-iname"
	}
	return "-name"
}

func specialFind(_ *catalog.Node, args []string, opt Opt) ([]string, error) {
	f, rest, err := splitNameFlags(args, "find")
	if err != nil {
		return nil, err
	}
	if err := f.validate(); err != nil {
		return nil, err
	}
	if f.line {
		return nil, fmt.Errorf("--line is for text search")
	}
	if !f.hasPattern() {
		if len(rest) < 1 {
			return nil, fmt.Errorf("find needs a name")
		}
		pattern := rest[0]
		dir := "."
		var extra []string
		if len(rest) > 1 {
			dir = rest[1]
			extra = rest[2:]
		}
		// Case, word, and invert are find primaries. fd stays the plain-glob path.
		if !f.sensitive && !f.insensitive && !f.word && !f.invert && opt.present("fd") {
			return append([]string{"fd", "--glob", pattern, dir}, extra...), nil
		}
		return findNames(dir, f.namePrim(), []string{pattern}, f.invert, extra), nil
	}
	if len(rest) > 1 {
		return nil, fmt.Errorf("find takes one path")
	}
	dir := "."
	if len(rest) == 1 {
		dir = rest[0]
	}
	if f.regex != "" {
		// BSD find matches -regex against the whole path. The pattern is not
		// rewritten. -iregex is the case-insensitive primary on this Mac.
		// GNU's -regextype is a help example for Linux, not a rewrite.
		prim := "-regex"
		if f.insensitive {
			prim = "-iregex"
		}
		return findNames(dir, prim, []string{f.regex}, f.invert, nil), nil
	}
	globs, err := f.globs()
	if err != nil {
		return nil, err
	}
	return findNames(dir, f.namePrim(), globs, f.invert, nil), nil
}

// findNames builds find PATH [!] [ ( ] -name GLOB [ -o ... ] [ ) ].
// --word does not add a word-boundary test: find has none. An exact name is
// already -name TEXT. --contains stays '*TEXT*'.
func findNames(dir, prim string, globs []string, invert bool, extra []string) []string {
	argv := []string{"find", dir}
	if invert {
		argv = append(argv, "!")
	}
	if len(globs) == 1 {
		argv = append(argv, prim, globs[0])
		return append(argv, extra...)
	}
	argv = append(argv, "(")
	for i, g := range globs {
		if i > 0 {
			argv = append(argv, "-o")
		}
		argv = append(argv, prim, g)
	}
	argv = append(argv, ")")
	return append(argv, extra...)
}

func (f nameFlags) globs() ([]string, error) {
	mode := "exact"
	switch {
	case f.hasStart:
		mode = "start"
	case f.hasEnd:
		mode = "end"
	case f.hasContains || f.anyof != "":
		mode = "contains"
	}
	class := bracket(f.anyof)
	if f.oneof != "" {
		vals := splitComma(f.oneof)
		if len(vals) == 0 {
			return nil, fmt.Errorf("--oneof needs values")
		}
		var globs []string
		for _, v := range vals {
			globs = append(globs, globOf(v, "", mode))
		}
		return globs, nil
	}
	text := f.contains
	if f.hasStart {
		text = f.startswith
	}
	if f.hasEnd {
		text = f.endswith
	}
	if text == "" && class == "" {
		return nil, fmt.Errorf("find needs a pattern")
	}
	return []string{globOf(text, class, mode)}, nil
}

func globOf(text, class, mode string) string {
	switch mode {
	case "start":
		return text + class + "*"
	case "end":
		return "*" + class + text
	case "exact":
		return text + class
	default:
		return "*" + text + class + "*"
	}
}

func specialSearch(_ *catalog.Node, args []string, opt Opt) ([]string, error) {
	f, rest, err := splitNameFlags(args, "search")
	if err != nil {
		return nil, err
	}
	if err := f.validate(); err != nil {
		return nil, err
	}
	if !f.hasPattern() {
		if len(rest) < 1 {
			return nil, fmt.Errorf("search needs text")
		}
		return searchCmd(opt, rest[0], true, nil, rest[1:], f), nil
	}
	if f.regex != "" {
		return searchCmd(opt, f.regex, false, nil, rest, f), nil
	}
	if f.oneof != "" && !f.hasContains && !f.hasStart && !f.hasEnd && f.anyof == "" {
		var extra []string
		for _, v := range splitComma(f.oneof) {
			extra = append(extra, "-e", v)
		}
		if len(extra) == 0 {
			return nil, fmt.Errorf("--oneof needs values")
		}
		return searchCmd(opt, "", true, extra, rest, f), nil
	}
	pat, fixed, err := f.searchPattern()
	if err != nil {
		return nil, err
	}
	return searchCmd(opt, pat, fixed, nil, rest, f), nil
}

func (f nameFlags) searchPattern() (string, bool, error) {
	mode := "contains"
	text := f.contains
	switch {
	case f.hasStart:
		mode = "start"
		text = f.startswith
	case f.hasEnd:
		mode = "end"
		text = f.endswith
	}
	class := bracket(f.anyof)
	if f.oneof != "" {
		var bits []string
		for _, v := range splitComma(f.oneof) {
			bits = append(bits, placePat(escapeLiteral(v), "", mode))
		}
		if len(bits) == 0 {
			return "", false, fmt.Errorf("--oneof needs values")
		}
		return strings.Join(bits, "|"), false, nil
	}
	if mode == "contains" && class == "" {
		if text == "" {
			return "", false, fmt.Errorf("search needs text")
		}
		return text, true, nil
	}
	if text == "" && class == "" {
		return "", false, fmt.Errorf("search needs text")
	}
	return placePat(escapeLiteral(text), class, mode), false, nil
}

func placePat(text, class, mode string) string {
	body := text + class
	switch mode {
	case "start":
		return "^" + body
	case "end":
		return body + "$"
	default:
		return body
	}
}

func searchCmd(opt Opt, pattern string, fixed bool, extra, paths []string, f nameFlags) []string {
	if opt.present("rg") {
		argv := []string{"rg"}
		argv = append(argv, searchFlags(true, f)...)
		if fixed {
			argv = append(argv, "-F")
		}
		argv = append(argv, extra...)
		if pattern != "" {
			argv = append(argv, pattern)
		}
		return append(argv, paths...)
	}
	argv := []string{"grep", "-n"}
	argv = append(argv, searchFlags(false, f)...)
	if fixed {
		argv = append(argv, "-F")
	} else {
		argv = append(argv, "-E")
	}
	argv = append(argv, extra...)
	if pattern != "" {
		argv = append(argv, pattern)
	}
	if len(paths) == 0 {
		argv = append(argv, ".")
	} else {
		argv = append(argv, paths...)
	}
	return argv
}

// searchFlags are the match switches. rg -s forces case-sensitive and turns
// smart-case off. grep -s means something else, so grep simply omits -i.
func searchFlags(rg bool, f nameFlags) []string {
	var out []string
	switch {
	case f.insensitive:
		out = append(out, "-i")
	case f.sensitive && rg:
		out = append(out, "-s")
	}
	if f.invert {
		out = append(out, "-v")
	}
	if f.word {
		out = append(out, "-w")
	}
	if f.line {
		out = append(out, "-x")
	}
	return out
}

func splitNameFlags(args []string, noun string) (nameFlags, []string, error) {
	var f nameFlags
	var rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		key, val, inline := splitKV(a)
		switch key {
		case "--sensitivecase", "--insensitivecase", "--word", "--invert", "--line":
			if inline {
				return f, nil, fmt.Errorf("%s does not take a value", key)
			}
			switch key {
			case "--sensitivecase":
				f.sensitive = true
			case "--insensitivecase":
				f.insensitive = true
			case "--word":
				f.word = true
			case "--invert":
				f.invert = true
			case "--line":
				f.line = true
			}
			f.set = true
			continue
		case "--contains", "--startswith", "--endswith", "--anyof", "--oneof", "--regex":
			if !inline {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			switch key {
			case "--contains":
				f.hasContains = true
				f.contains = val
			case "--startswith":
				f.hasStart = true
				f.startswith = val
			case "--endswith":
				f.hasEnd = true
				f.endswith = val
			case "--anyof":
				if val == "" {
					return f, nil, fmt.Errorf("--anyof needs a class")
				}
				f.anyof = val
			case "--oneof":
				if val == "" {
					return f, nil, fmt.Errorf("--oneof needs values")
				}
				f.oneof = val
			case "--regex":
				if val == "" {
					return f, nil, fmt.Errorf("--regex needs a pattern")
				}
				f.regex = val
			}
			f.set = true
		default:
			if strings.HasPrefix(a, "--") {
				return f, nil, fmt.Errorf("%s does not take %s", noun, a)
			}
			rest = append(rest, a)
		}
	}
	if f.hasContains && f.contains == "" && f.anyof == "" && f.oneof == "" {
		return f, nil, fmt.Errorf("--contains needs text")
	}
	if f.hasStart && f.startswith == "" && f.anyof == "" && f.oneof == "" {
		return f, nil, fmt.Errorf("--startswith needs text")
	}
	if f.hasEnd && f.endswith == "" && f.anyof == "" && f.oneof == "" {
		return f, nil, fmt.Errorf("--endswith needs text")
	}
	return f, rest, nil
}

func splitKV(a string) (key, val string, inline bool) {
	if !strings.HasPrefix(a, "--") {
		return a, "", false
	}
	if i := strings.IndexByte(a, '='); i > 0 {
		return a[:i], a[i+1:], true
	}
	return a, "", false
}

func bracket(class string) string {
	if class == "" {
		return ""
	}
	if strings.HasPrefix(class, "[") && strings.HasSuffix(class, "]") {
		return class
	}
	return "[" + class + "]"
}

func splitComma(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func escapeLiteral(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\', '.', '^', '$', '*', '+', '?', '(', ')', '[', ']', '{', '}', '|':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
