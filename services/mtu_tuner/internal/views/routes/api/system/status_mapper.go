package system

import (
	"mtu-tuner/internal/core"
	protos "mtu-tuner/internal/views/routes/api/_gen_protos"
)

func systemStatusProto(status core.SystemStatus) *protos.SystemStatus {
	return &protos.SystemStatus{
		PlatformName:          status.PlatformName,
		IsAdmin:               status.IsAdmin,
		SupportsPersistentMtu: status.SupportsPersistentMTU,
		Busy:                  status.Busy,
		CurrentTaskKind:       status.CurrentTaskKind,
		CurrentTaskStatus:     status.CurrentTaskStatus,
	}
}
