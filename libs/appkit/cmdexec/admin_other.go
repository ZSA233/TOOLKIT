//go:build !windows

package cmdexec

func isWindowsAdmin() bool {
	return false
}
