package system

import (
	"mtu-tuner/internal/core"
	apitypes "mtu-tuner/internal/views/routes/api/_gen_types"
)

func systemStatusDTO(status core.SystemStatus) *apitypes.SystemStatus {
	return &apitypes.SystemStatus{
		PlatformName:          status.PlatformName,
		IsAdmin:               status.IsAdmin,
		SupportsPersistentMtu: status.SupportsPersistentMTU,
		Busy:                  status.Busy,
		CurrentTaskKind:       status.CurrentTaskKind,
		CurrentTaskStatus:     status.CurrentTaskStatus,
	}
}
