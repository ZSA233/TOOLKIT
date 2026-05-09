//go:build windows

package proxytest

import (
	"os/exec"
	"syscall"
)

func configureChromeCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
