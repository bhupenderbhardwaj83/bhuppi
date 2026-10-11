package execx

import (
	"path/filepath"
	"strings"
)

// interactive programs need the user's terminal. bhuppi execs them unchanged.
var interactive = map[string]struct{}{
	"ssh": {}, "top": {}, "less": {}, "vim": {}, "vi": {}, "nano": {},
	"htop": {}, "watch": {}, "more": {}, "tmux": {}, "screen": {},
	"emacs": {}, "pg": {}, "man": {},
}

// NeedsTTY reports that the child must keep the real terminal.
//
// Capture stdout (and page it) only when stdout is a terminal and the program
// is not one of these. Stdin is never replaced with a pipe: the child keeps
// the stdin it was given. When stdout is a pipe or file, the child is exec'd
// so that redirection stays intact. Unknown interactive names are treated as
// needing the terminal.
func NeedsTTY(argv0 string, stdoutIsTTY bool) bool {
	if !stdoutIsTTY {
		return true
	}
	base := strings.ToLower(filepath.Base(argv0))
	base = strings.TrimSuffix(base, ".exe")
	if _, ok := interactive[base]; ok {
		return true
	}
	return false
}
