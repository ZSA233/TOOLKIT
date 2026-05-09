package network

import (
	"mtu-tuner/internal/core"
	protos "mtu-tuner/internal/views/routes/api/_gen_protos"
	sharedroute "mtu-tuner/internal/views/routes/api/shared"
)

func interfaceInfoProto(info core.InterfaceInfo) *protos.InterfaceInfo {
	return &protos.InterfaceInfo{
		PlatformName: info.PlatformName,
		Name:         info.Name,
		Index:        info.Index,
		Mtu:          info.MTU,
		Gateway:      info.Gateway,
		LocalAddress: info.LocalAddress,
		Description:  info.Description,
	}
}

func interfaceInfoListProto(values []core.InterfaceInfo) []*protos.InterfaceInfo {
	if len(values) == 0 {
		return []*protos.InterfaceInfo{}
	}
	items := make([]*protos.InterfaceInfo, 0, len(values))
	for _, value := range values {
		items = append(items, interfaceInfoProto(value))
	}
	return items
}

// Mutation routes now send only a stable interface identity, so preserve just
// the fields the app layer uses to look the interface back up.
func clashTargetProto(target core.ClashTarget) *protos.ClashTarget {
	return &protos.ClashTarget{
		Group:      target.Group,
		Leaf:       target.Leaf,
		Server:     target.Server,
		Port:       target.Port,
		ResolvedIp: target.ResolvedIP,
		ConfigPath: target.ConfigPath,
		Source:     target.Source,
	}
}

func probeSelectionProto(selection core.ProbeSelection) *protos.ProbeSelection {
	result := &protos.ProbeSelection{
		ProbeIp: selection.ProbeIP,
		Warning: selection.Warning,
	}
	if selection.Target != nil {
		result.Target = clashTargetProto(*selection.Target)
	}
	return result
}

func interfaceCommandResultProto(result core.InterfaceCommandResult) *protos.InterfaceCommandResult {
	return &protos.InterfaceCommandResult{
		Interface:   interfaceInfoProto(result.Interface),
		Output:      result.Output,
		OriginalMtu: result.OriginalMTU,
	}
}

func interfaceRefCore(info *protos.InterfaceRef) core.InterfaceInfo {
	return sharedroute.InterfaceRefCore(info)
}
