package network

import (
	"mtu-tuner/internal/core"
	apitypes "mtu-tuner/internal/views/routes/api/_gen_types"
	sharedroute "mtu-tuner/internal/views/routes/api/shared"
)

func interfaceInfoDTO(info core.InterfaceInfo) *apitypes.InterfaceInfo {
	return &apitypes.InterfaceInfo{
		PlatformName: info.PlatformName,
		Name:         info.Name,
		Index:        info.Index,
		Mtu:          info.MTU,
		Gateway:      info.Gateway,
		LocalAddress: info.LocalAddress,
		Description:  info.Description,
	}
}

func interfaceInfoListDTO(values []core.InterfaceInfo) []*apitypes.InterfaceInfo {
	if len(values) == 0 {
		return []*apitypes.InterfaceInfo{}
	}
	items := make([]*apitypes.InterfaceInfo, 0, len(values))
	for _, value := range values {
		items = append(items, interfaceInfoDTO(value))
	}
	return items
}

// Mutation routes now send only a stable interface identity, so preserve just
// the fields the app layer uses to look the interface back up.
func clashTargetDTO(target core.ClashTarget) *apitypes.ClashTarget {
	return &apitypes.ClashTarget{
		Group:      target.Group,
		Leaf:       target.Leaf,
		Server:     target.Server,
		Port:       target.Port,
		ResolvedIp: target.ResolvedIP,
		ConfigPath: target.ConfigPath,
		Source:     target.Source,
	}
}

func probeSelectionDTO(selection core.ProbeSelection) *apitypes.ProbeSelection {
	result := &apitypes.ProbeSelection{
		ProbeIp: selection.ProbeIP,
		Warning: selection.Warning,
	}
	if selection.Target != nil {
		result.Target = clashTargetDTO(*selection.Target)
	}
	return result
}

func interfaceCommandResultDTO(result core.InterfaceCommandResult) *apitypes.InterfaceCommandResult {
	return &apitypes.InterfaceCommandResult{
		Interface:   interfaceInfoDTO(result.Interface),
		Output:      result.Output,
		OriginalMtu: result.OriginalMTU,
	}
}

func interfaceRefCore(info *apitypes.InterfaceRef) core.InterfaceInfo {
	return sharedroute.InterfaceRefCore(info)
}
