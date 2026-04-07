//go:build darwin || linux || freebsd || openbsd

package session

import "syscall"

func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}
