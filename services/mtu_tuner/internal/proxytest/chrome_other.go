//go:build !windows

package proxytest

import "os/exec"

func configureChromeCommand(cmd *exec.Cmd) {
}
