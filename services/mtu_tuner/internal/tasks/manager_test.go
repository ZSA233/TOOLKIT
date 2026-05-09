package tasks

import (
	"context"
	"testing"
	"time"

	"mtu-tuner/internal/core"
)

func TestManagerRejectsBusyTask(t *testing.T) {
	t.Parallel()

	manager := NewManager()
	started := make(chan struct{})
	release := make(chan struct{})

	if _, err := manager.Start(core.TaskKindConnectivityTest, func(controller *Controller) error {
		close(started)
		<-release
		return nil
	}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	<-started

	if _, err := manager.Start(core.TaskKindMTUSweep, func(controller *Controller) error { return nil }); err == nil {
		t.Fatal("second Start() expected busy error")
	}

	close(release)
	waitForIdle(t, manager)
}

func TestManagerCancelTransitionsToIdle(t *testing.T) {
	t.Parallel()

	manager := NewManager()
	started := make(chan struct{})

	if _, err := manager.Start(core.TaskKindConnectivityTest, func(controller *Controller) error {
		close(started)
		<-controller.Context().Done()
		return controller.Context().Err()
	}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	<-started

	state := manager.Cancel()
	if state.Status != core.TaskStatusStopping {
		t.Fatalf("Cancel() status = %q, want %q", state.Status, core.TaskStatusStopping)
	}
	if !state.CancelRequested {
		t.Fatal("Cancel() expected CancelRequested=true")
	}

	waitForIdle(t, manager)
}

func waitForIdle(t *testing.T, manager *Manager) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if manager.Snapshot().Status == core.TaskStatusIdle {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("manager did not become idle; last state = %#v", manager.Snapshot())
}

func TestControllerCancelRequestedReflectsContext(t *testing.T) {
	t.Parallel()

	controller := &Controller{ctx: context.Background()}
	if controller.CancelRequested() {
		t.Fatal("CancelRequested() = true for active context")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	controller.ctx = ctx
	if !controller.CancelRequested() {
		t.Fatal("CancelRequested() = false for cancelled context")
	}
}
