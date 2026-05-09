//go:build windows

package cmdexec

import "golang.org/x/sys/windows"

func isWindowsAdmin() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}
