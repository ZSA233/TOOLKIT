package system

import (
	"testing"

	"mtu-tuner/internal/core"
)

func TestSystemStatusProtoMapsAppStatusFields(t *testing.T) {
	t.Parallel()

	status := core.SystemStatus{
		PlatformName:          "windows",
		IsAdmin:               true,
		SupportsPersistentMTU: true,
		Busy:                  true,
		CurrentTaskKind:       "mtu_sweep",
		CurrentTaskStatus:     core.TaskStatusRunning,
	}

	got := systemStatusDTO(status)
	if got.PlatformName != status.PlatformName {
		t.Fatalf("PlatformName = %q, want %q", got.PlatformName, status.PlatformName)
	}
	if got.IsAdmin != status.IsAdmin {
		t.Fatalf("IsAdmin = %v, want %v", got.IsAdmin, status.IsAdmin)
	}
	if got.SupportsPersistentMtu != status.SupportsPersistentMTU {
		t.Fatalf("SupportsPersistentMtu = %v, want %v", got.SupportsPersistentMtu, status.SupportsPersistentMTU)
	}
	if got.Busy != status.Busy {
		t.Fatalf("Busy = %v, want %v", got.Busy, status.Busy)
	}
	if got.CurrentTaskKind != status.CurrentTaskKind {
		t.Fatalf("CurrentTaskKind = %q, want %q", got.CurrentTaskKind, status.CurrentTaskKind)
	}
	if got.CurrentTaskStatus != status.CurrentTaskStatus {
		t.Fatalf("CurrentTaskStatus = %q, want %q", got.CurrentTaskStatus, status.CurrentTaskStatus)
	}
}
