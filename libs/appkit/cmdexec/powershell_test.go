package cmdexec

import (
	"context"
	"testing"
)

type captureRunner struct {
	args    []string
	options Options
}

func (runner *captureRunner) Run(ctx context.Context, args []string, options Options) (Result, error) {
	runner.args = append([]string(nil), args...)
	runner.options = options
	return Result{}, nil
}

func TestPowerShellQuoteEscapesSingleQuotes(t *testing.T) {
	got := PowerShellQuote("it's ready")
	want := "'it''s ready'"
	if got != want {
		t.Fatalf("PowerShellQuote() = %q, want %q", got, want)
	}
}

func TestRunPowerShellUsesHiddenWindowAndPreamble(t *testing.T) {
	runner := &captureRunner{}

	if _, err := RunPowerShell(context.Background(), runner, "Write-Output 'ok'"); err != nil {
		t.Fatalf("RunPowerShell() error = %v", err)
	}

	if len(runner.args) != 6 {
		t.Fatalf("RunPowerShell() args len = %d, want 6", len(runner.args))
	}
	if runner.args[0] != "powershell.exe" {
		t.Fatalf("RunPowerShell() executable = %q, want powershell.exe", runner.args[0])
	}
	if runner.args[1] != "-NoProfile" || runner.args[2] != "-ExecutionPolicy" || runner.args[3] != "Bypass" || runner.args[4] != "-Command" {
		t.Fatalf("RunPowerShell() args = %#v", runner.args)
	}
	if !runner.options.HiddenWindow {
		t.Fatalf("RunPowerShell() HiddenWindow = false, want true")
	}
	if runner.args[5] == "" {
		t.Fatalf("RunPowerShell() command script is empty")
	}
}
