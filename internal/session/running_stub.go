//go:build !(darwin || linux || freebsd || openbsd)

package session

func pidAlive(pid int) bool {
	// No portable zero-signal check; show all registry rows.
	return pid > 0
}
