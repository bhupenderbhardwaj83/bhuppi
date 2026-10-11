//go:build windows

package execx

import (
	"os"
	"os/exec"
)

// Exec runs argv and waits. Windows has no syscall.Exec equivalent that keeps the console job.
func Exec(argv []string, stdinNull bool) error {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if stdinNull {
		cmd.Stdin = nil
	}
	return cmd.Run()
}
