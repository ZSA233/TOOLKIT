//go:build !windows

package cmdexec

import "syscall"

func sysProcAttr(hidden bool) *syscall.SysProcAttr {
	return nil
}
