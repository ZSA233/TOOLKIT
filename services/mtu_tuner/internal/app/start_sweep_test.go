package app

import (
	"strings"
	"testing"

	"mtu-tuner/internal/core"
	"mtu-tuner/internal/tasks"
)

func TestStartSweepRejectsWithoutAdminPrivileges(t *testing.T) {
	service := &Service{
		goos:      "windows",
		proxytest: nil,
		tasks:     tasks.NewManager(),
		isAdmin: func() bool {
			return false
		},
	}

	_, err := service.StartSweep(core.SweepRunRequest{})
	if err == nil {
		t.Fatal("StartSweep() error = nil, want admin privilege error")
	}
	if !strings.Contains(err.Error(), "admin/root privileges") {
		t.Fatalf("StartSweep() error = %q, want admin/root privileges message", err.Error())
	}
	if state := service.TaskState(nil); state.Status != core.TaskStatusIdle {
		t.Fatalf("TaskState().Status = %q, want %q", state.Status, core.TaskStatusIdle)
	}
}
