//go:build !windows

package execx

import (
	"os"
	"syscall"
)

// Exec replaces this process with argv[0], which must be an absolute path.
func Exec(argv []string, stdinNull bool) error {
	if stdinNull {
		f, err := os.Open(os.DevNull)
		if err == nil {
			_ = dupToStdin(int(f.Fd()))
			_ = f.Close()
		}
	}
	return syscall.Exec(argv[0], argv, os.Environ())
}
