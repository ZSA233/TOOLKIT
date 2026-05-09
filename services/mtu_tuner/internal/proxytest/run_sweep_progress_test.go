package proxytest

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"mtu-tuner/internal/core"
)

type sweepProgressAdapterStub struct {
	currentMTU int
}

func (stub *sweepProgressAdapterStub) CurrentMTU(context.Context, core.InterfaceInfo) (int, error) {
	return stub.currentMTU, nil
}

func (stub *sweepProgressAdapterStub) SetMTU(_ context.Context, _ core.InterfaceInfo, mtu int, _ bool) (string, error) {
	stub.currentMTU = mtu
	return "", nil
}

func TestRunSweepEmitsCompletedProgress(t *testing.T) {
	tempDir := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir(%q) error = %v", tempDir, err)
	}
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	service := New("windows")
	adapter := &sweepProgressAdapterStub{currentMTU: 1500}
	lastDone := -1
	lastTotal := -1
	lastLabel := ""

	result, err := service.RunSweep(
		context.Background(),
		adapter,
		core.SweepRunRequest{
			Interface:   core.InterfaceInfo{Name: "Wi-Fi", MTU: 1500},
			HTTPProxy:   "bad-proxy",
			TestProfile: "quick",
			SweepMTUs:   "1400",
		},
		func(done int, total int, label string) {
			lastDone = done
			lastTotal = total
			lastLabel = label
		},
		nil,
	)
	if err != nil {
		t.Fatalf("RunSweep() error = %v", err)
	}
	if result.Cancelled {
		t.Fatal("RunSweep() cancelled = true, want false")
	}
	if lastDone != lastTotal {
		t.Fatalf("final progress = %d/%d, want complete", lastDone, lastTotal)
	}
	if lastLabel != "complete" {
		t.Fatalf("final progress label = %q, want complete", lastLabel)
	}
	if _, err := os.Stat(filepath.Join(tempDir, filepath.Base(result.OutputPath))); err != nil {
		t.Fatalf("sweep output %q missing: %v", result.OutputPath, err)
	}
}
