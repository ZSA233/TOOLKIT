//go:build windows

package cmdexec

import "syscall"

func sysProcAttr(hidden bool) *syscall.SysProcAttr {
	if !hidden {
		return nil
	}
	return &syscall.SysProcAttr{HideWindow: true}
}
