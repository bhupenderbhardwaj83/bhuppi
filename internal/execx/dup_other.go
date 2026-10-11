//go:build !windows && !linux && !darwin

package execx

import "syscall"

func dupToStdin(fd int) error {
	return syscall.Dup2(fd, 0)
}
