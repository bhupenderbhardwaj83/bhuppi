package dispatch

import (
	"regexp"
	"strings"
)

var safeArg = regexp.MustCompile(`^[A-Za-z0-9@%+=:,./_{}!-]+$`)

// Quote renders one argv element the way a shell would show it.
func Quote(a string) string {
	if a == "" {
		return "''"
	}
	if safeArg.MatchString(a) {
		return a
	}
	return "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
}

// FormatArgv joins argv for --dry-run.
func FormatArgv(argv []string) string {
	parts := make([]string, len(argv))
	for i, a := range argv {
		parts[i] = Quote(a)
	}
	return strings.Join(parts, " ")
}

// FormatPipeline joins stages with a pipe for display.
func FormatPipeline(steps [][]string) string {
	parts := make([]string, len(steps))
	for i, st := range steps {
		parts[i] = FormatArgv(st)
	}
	return strings.Join(parts, " | ")
}
