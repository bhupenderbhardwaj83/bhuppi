// Package clip writes a command line to the system clipboard.
package clip

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
)

// ErrMissing means this machine has no clipboard tool bhuppi knows.
var ErrMissing = errors.New("clipboard tool missing")

// Copy writes text. sink, when set, receives the text instead of a system tool.
func Copy(text string, sink func(string) error) error {
	if sink != nil {
		return sink(text)
	}
	cmd, err := system()
	if err != nil {
		return err
	}
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return ErrMissing
	}
	return nil
}

func system() (*exec.Cmd, error) {
	switch runtime.GOOS {
	case "darwin":
		path, err := exec.LookPath("pbcopy")
		if err != nil {
			return nil, ErrMissing
		}
		return exec.Command(path), nil
	case "windows":
		path, err := exec.LookPath("clip")
		if err != nil {
			return nil, ErrMissing
		}
		return exec.Command(path), nil
	default:
		if path, err := exec.LookPath("wl-copy"); err == nil {
			return exec.Command(path), nil
		}
		if path, err := exec.LookPath("xclip"); err == nil {
			return exec.Command(path, "-selection", "clipboard"), nil
		}
		if path, err := exec.LookPath("xsel"); err == nil {
			return exec.Command(path, "--clipboard"), nil
		}
		return nil, ErrMissing
	}
}
