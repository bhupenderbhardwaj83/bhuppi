//go:build linux

package execx

import "syscall"

func dupToStdin(fd int) error {
	return syscall.Dup3(fd, 0, 0)
}
