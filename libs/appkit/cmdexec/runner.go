package cmdexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"

	"toolkit/libs/utils/textutil"
)

type Options struct {
	HiddenWindow bool
}

type Result struct {
	Args        []string
	ExitCode    int
	Stdout      string
	Stderr      string
	StdoutBytes []byte
	StderrBytes []byte
}

type Runner interface {
	Run(ctx context.Context, args []string, options Options) (Result, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, args []string, options Options) (Result, error) {
	if len(args) == 0 {
		return Result{}, fmt.Errorf("command args are empty")
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.SysProcAttr = sysProcAttr(options.HiddenWindow)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	result := Result{
		Args:        append([]string(nil), args...),
		ExitCode:    exitCode(cmd.ProcessState, runErr),
		StdoutBytes: stdout.Bytes(),
		StderrBytes: stderr.Bytes(),
		Stdout:      textutil.NormalizeUTF8Lines(stdout.Bytes()),
		Stderr:      textutil.NormalizeUTF8Lines(stderr.Bytes()),
	}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(ctx.Err(), context.Canceled) {
		return result, ctx.Err()
	}

	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		return result, nil
	}
	if runErr != nil {
		return result, runErr
	}
	return result, nil
}
