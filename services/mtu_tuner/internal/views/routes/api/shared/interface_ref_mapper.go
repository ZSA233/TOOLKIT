package shared

import (
	"mtu-tuner/internal/core"
	protos "mtu-tuner/internal/views/routes/api/_gen_protos"
)

// InterfaceRef only carries the stable identity fields needed to look an
// interface back up inside the app layer.
func InterfaceRefCore(info *protos.InterfaceRef) core.InterfaceInfo {
	if info == nil {
		return core.InterfaceInfo{}
	}
	return core.InterfaceInfo{
		PlatformName: info.PlatformName,
		Name:         info.Name,
		Index:        info.Index,
	}
}
