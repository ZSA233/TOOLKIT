package app

import (
	"context"
	"fmt"
	"strings"

	"mtu-tuner/internal/core"
)

func (service *Service) DetectInterface(ctx context.Context, request core.DetectRequest) (core.DetectResult, error) {
	selection, err := service.selectProbe(ctx, request)
	if err != nil {
		return core.DetectResult{}, err
	}
	routeInterface, err := service.netiface.DetectInterface(ctx, selection.ProbeIP)
	if err != nil {
		return core.DetectResult{}, err
	}
	candidates, err := service.netiface.ListInterfaces(ctx)
	if err != nil {
		return core.DetectResult{}, err
	}

	// Route probes often land on TUN adapters, but MTU bottlenecks for proxy
	// connectivity usually live on the underlying egress NIC instead.
	info, warning := service.preferDetectedInterface(ctx, routeInterface, candidates)
	if warning != "" {
		selection.Warning = joinDetectWarnings(selection.Warning, warning)
	}

	originalMTU := service.recordOriginalMTU(info)
	return core.DetectResult{
		Selection:   selection,
		Interface:   info,
		OriginalMTU: originalMTU,
		Candidates:  mergeInterfaceCandidates(info, candidates),
	}, nil
}

func (service *Service) selectProbe(ctx context.Context, request core.DetectRequest) (core.ProbeSelection, error) {
	if request.ClashCurrent {
		target, err := service.clash.ResolveCurrentTarget(ctx, core.ResolveTargetRequest{
			Controller: request.Controller,
			Secret:     request.Secret,
			Group:      request.Group,
			ConfigPath: request.ConfigPath,
		})
		if err != nil {
			return core.ProbeSelection{}, err
		}
		return core.ProbeSelection{
			ProbeIP: target.ResolvedIP,
			Target:  &target,
		}, nil
	}
	return service.clash.SelectProbe(ctx, request, true)
}

func (service *Service) preferDetectedInterface(
	ctx context.Context,
	routeInterface core.InterfaceInfo,
	candidates []core.InterfaceInfo,
) (core.InterfaceInfo, string) {
	if !isLikelyVirtualInterface(routeInterface) {
		return routeInterface, ""
	}

	if preferred, ok := service.detectUnderlyingInterface(ctx, routeInterface, candidates); ok {
		if sameInterface(routeInterface, preferred) {
			return routeInterface, ""
		}
		return preferred, fmt.Sprintf(
			"Current route resolves to virtual interface %s. Selected underlying interface %s instead.",
			interfaceDisplayName(routeInterface),
			interfaceDisplayName(preferred),
		)
	}

	return routeInterface, ""
}

func (service *Service) detectUnderlyingInterface(
	ctx context.Context,
	routeInterface core.InterfaceInfo,
	candidates []core.InterfaceInfo,
) (core.InterfaceInfo, bool) {
	if preferred, err := service.netiface.DetectDefaultInterface(ctx); err == nil && !isLikelyVirtualInterface(preferred) {
		return mergeDetectedCandidate(preferred, candidates), true
	}
	return pickPreferredInterfaceCandidate(routeInterface, candidates)
}

func mergeDetectedCandidate(info core.InterfaceInfo, candidates []core.InterfaceInfo) core.InterfaceInfo {
	key := core.InterfaceKey(info)
	for _, candidate := range candidates {
		if core.InterfaceKey(candidate) == key {
			return candidate
		}
	}
	return info
}

func pickPreferredInterfaceCandidate(routeInterface core.InterfaceInfo, candidates []core.InterfaceInfo) (core.InterfaceInfo, bool) {
	routeKey := core.InterfaceKey(routeInterface)
	bestIndex := -1
	bestScore := -1
	for index, candidate := range candidates {
		if core.InterfaceKey(candidate) == routeKey {
			continue
		}
		score := preferredInterfaceScore(candidate)
		if score < 0 {
			continue
		}
		if score > bestScore {
			bestIndex = index
			bestScore = score
		}
	}
	if bestIndex < 0 {
		return core.InterfaceInfo{}, false
	}
	return candidates[bestIndex], true
}

func preferredInterfaceScore(candidate core.InterfaceInfo) int {
	if isLikelyVirtualInterface(candidate) {
		return -1
	}

	score := 100
	if strings.TrimSpace(candidate.Gateway) != "" {
		score += 40
	}
	if looksLikePhysicalInterface(candidate) {
		score += 20
	}
	if candidate.MTU >= 1400 && candidate.MTU <= 2000 {
		score += 5
	}
	return score
}

func looksLikePhysicalInterface(info core.InterfaceInfo) bool {
	name := strings.ToLower(strings.TrimSpace(info.Name))
	description := strings.ToLower(strings.TrimSpace(info.Description))
	switch {
	case strings.Contains(name, "wi-fi"),
		strings.Contains(name, "wifi"),
		strings.Contains(name, "wlan"),
		strings.Contains(name, "ethernet"),
		strings.Contains(description, "wi-fi"),
		strings.Contains(description, "wifi"),
		strings.Contains(description, "wireless"),
		strings.Contains(description, "ethernet"),
		strings.HasPrefix(name, "en"),
		strings.HasPrefix(name, "eth"),
		strings.HasPrefix(name, "wlan"),
		strings.HasPrefix(name, "wwan"),
		strings.HasPrefix(name, "ppp"):
		return true
	default:
		return false
	}
}

func isLikelyVirtualInterface(info core.InterfaceInfo) bool {
	name := strings.ToLower(strings.TrimSpace(info.Name))
	description := strings.ToLower(strings.TrimSpace(info.Description))
	raw := name + " " + description
	normalized := normalizeInterfaceHint(raw)

	virtualHints := []string{
		"meta",
		"clash",
		"mihomo",
		"wintun",
		"wireguard",
		"tailscale",
		"zerotier",
		"openvpn",
		"anyconnect",
		"hyper v",
		"virtualbox",
		"vmware",
		"loopback",
		"npcap",
		"v ethernet",
	}
	for _, hint := range virtualHints {
		if strings.Contains(raw, hint) || strings.Contains(normalized, hint) {
			return true
		}
	}

	switch {
	case strings.HasPrefix(name, "utun"),
		strings.HasPrefix(name, "tun"),
		strings.HasPrefix(name, "tap"),
		strings.HasPrefix(name, "wg"),
		strings.HasPrefix(name, "zt"),
		strings.HasPrefix(name, "br-"),
		strings.HasPrefix(name, "virbr"),
		strings.HasPrefix(name, "docker"),
		strings.HasPrefix(name, "vboxnet"),
		strings.HasPrefix(name, "vmnet"),
		strings.HasPrefix(name, "veth"),
		strings.HasPrefix(name, "tailscale"),
		strings.HasPrefix(name, "wsl"):
		return true
	default:
		return false
	}
}

func normalizeInterfaceHint(value string) string {
	replacer := strings.NewReplacer(
		"-", " ",
		"_", " ",
		".", " ",
		"/", " ",
		"(", " ",
		")", " ",
	)
	return strings.Join(strings.Fields(replacer.Replace(value)), " ")
}

func sameInterface(left core.InterfaceInfo, right core.InterfaceInfo) bool {
	return core.InterfaceKey(left) == core.InterfaceKey(right)
}

func interfaceDisplayName(info core.InterfaceInfo) string {
	if strings.TrimSpace(info.Name) != "" {
		return info.Name
	}
	if strings.TrimSpace(info.Description) != "" {
		return info.Description
	}
	return "unknown"
}

func joinDetectWarnings(values ...string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		parts = append(parts, trimmed)
	}
	return strings.Join(parts, " ")
}

func mergeInterfaceCandidates(primary core.InterfaceInfo, candidates []core.InterfaceInfo) []core.InterfaceInfo {
	merged := make([]core.InterfaceInfo, 0, len(candidates)+1)
	seen := map[string]struct{}{}

	appendUnique := func(info core.InterfaceInfo) {
		key := core.InterfaceKey(info)
		if strings.TrimSpace(key) == "|" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		if strings.TrimSpace(info.Description) == "" {
			info.Description = info.Name
		}
		merged = append(merged, info)
	}

	appendUnique(primary)
	for _, candidate := range candidates {
		appendUnique(candidate)
	}
	return merged
}
