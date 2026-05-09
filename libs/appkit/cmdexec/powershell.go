package cmdexec

import (
	"context"
	"strings"
)

func PowerShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func RunPowerShell(ctx context.Context, runner Runner, script string) (Result, error) {
	prefix := "$ErrorActionPreference='Stop';[Console]::OutputEncoding=[System.Text.Encoding]::UTF8;$OutputEncoding=[System.Text.Encoding]::UTF8;"
	return runner.Run(
		ctx,
		[]string{"powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", prefix + script},
		Options{HiddenWindow: true},
	)
}
