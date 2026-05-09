//go:build windows

package elevate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"toolkit/libs/appkit/cmdexec"
)

func SupportsAdminRelaunch() bool {
	return true
}

// Windows question dialogs map most reliably to native Yes/No buttons.
func AdminRelaunchConfirmLabel() string {
	return "Yes"
}

func AdminRelaunchCancelLabel() string {
	return "No"
}

func RelaunchCurrentProcessAsAdmin() error {
	executablePath, err := os.Executable()
	if err != nil {
		return err
	}
	workingDir, err := os.Getwd()
	if err != nil {
		workingDir = filepath.Dir(executablePath)
	}

	argumentList := ""
	if len(os.Args) > 1 {
		quoted := make([]string, 0, len(os.Args)-1)
		for _, arg := range os.Args[1:] {
			quoted = append(quoted, cmdexec.PowerShellQuote(arg))
		}
		argumentList = " -ArgumentList @(" + strings.Join(quoted, ",") + ")"
	}

	script := fmt.Sprintf(
		"Start-Process -Verb RunAs -FilePath %s -WorkingDirectory %s%s",
		cmdexec.PowerShellQuote(executablePath),
		cmdexec.PowerShellQuote(workingDir),
		argumentList,
	)
	result, runErr := cmdexec.RunPowerShell(context.Background(), cmdexec.ExecRunner{}, script)
	if runErr != nil {
		return runErr
	}
	if result.ExitCode != 0 {
		if strings.TrimSpace(result.Stderr) != "" {
			return fmt.Errorf(result.Stderr)
		}
		return fmt.Errorf("administrator relaunch failed with exit code %d", result.ExitCode)
	}
	return nil
}
