package core

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

const (
	TaskStatusIdle      = "idle"
	TaskStatusRunning   = "running"
	TaskStatusStopping  = "stopping"
	TaskStatusCompleted = "completed"
	TaskStatusCancelled = "cancelled"
	TaskStatusFailed    = "failed"

	// Task kinds are serialized to frontend clients and must stay stable across
	// task state, progress, and log events.
	TaskKindConnectivityTest = "connectivity_test"
	TaskKindMTUSweep         = "mtu_sweep"
)

func DefaultSavedSettings() SavedSettings {
	return SavedSettings{
		Version:       ConfigVersion,
		RouteProbe:    DefaultProbe,
		FallbackProbe: DefaultFallbackProbe,
		HTTPProxy:     DefaultHTTPProxy,
		ClashAPI:      DefaultController,
		ProxyGroup:    DefaultClashGroup,
		ConfigPath:    "",
		BrowserPath:   "",
		TestProfile:   DefaultTestProfile,
		TestTargets:   DefaultTestTargets(),
		SweepMTUs:     DefaultMTUList,
		TargetMTU:     DefaultQuickMTU,
	}
}

func NormalizeSavedSettings(settings SavedSettings) SavedSettings {
	defaults := DefaultSavedSettings()
	settings.Version = ConfigVersion
	if strings.TrimSpace(settings.RouteProbe) == "" {
		settings.RouteProbe = defaults.RouteProbe
	}
	if strings.TrimSpace(settings.FallbackProbe) == "" {
		settings.FallbackProbe = defaults.FallbackProbe
	}
	if strings.TrimSpace(settings.HTTPProxy) == "" {
		settings.HTTPProxy = defaults.HTTPProxy
	}
	if strings.TrimSpace(settings.ClashAPI) == "" {
		settings.ClashAPI = defaults.ClashAPI
	}
	if strings.TrimSpace(settings.ProxyGroup) == "" {
		settings.ProxyGroup = defaults.ProxyGroup
	}
	if strings.TrimSpace(settings.TestProfile) == "" {
		settings.TestProfile = defaults.TestProfile
	}
	settings.TestTargets = normalizeSavedTestTargets(settings.TestTargets)
	if len(settings.TestTargets) == 0 {
		settings.TestTargets = defaults.TestTargets
	}
	if strings.TrimSpace(settings.SweepMTUs) == "" {
		settings.SweepMTUs = defaults.SweepMTUs
	}
	if settings.TargetMTU == 0 {
		settings.TargetMTU = defaults.TargetMTU
	}
	return settings
}

func ValidateMTU(mtu int) error {
	if mtu < 576 || mtu > 9000 {
		return fmt.Errorf("MTU must be between 576 and 9000")
	}
	return nil
}

func ParseMTUList(value string) ([]int, error) {
	parts := regexp.MustCompile(`[,\s]+`).Split(strings.TrimSpace(value), -1)
	mtus := make([]int, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		var mtu int
		if _, err := fmt.Sscanf(part, "%d", &mtu); err != nil {
			return nil, fmt.Errorf("invalid MTU value %q: %w", part, err)
		}
		if err := ValidateMTU(mtu); err != nil {
			return nil, err
		}
		mtus = append(mtus, mtu)
	}
	if len(mtus) == 0 {
		return nil, fmt.Errorf("no MTU values provided")
	}
	return mtus, nil
}

func InterfaceKey(info InterfaceInfo) string {
	if strings.TrimSpace(info.Index) != "" {
		return strings.ToLower(info.PlatformName) + "|" + strings.TrimSpace(info.Index)
	}
	return strings.ToLower(info.PlatformName) + "|" + strings.TrimSpace(info.Name)
}

func SupportsPersistentMTU(goos string) bool {
	return strings.EqualFold(goos, "windows")
}

func IsAutoProbe(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "auto", "auto-clash", "clash", "current", "current-clash":
		return true
	default:
		return false
	}
}

func IsAutoGroup(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "auto", "default", "main":
		return true
	default:
		return false
	}
}

func normalizeSavedTestTargets(targets []TestTarget) []TestTarget {
	if len(targets) == 0 {
		return nil
	}

	normalized := make([]TestTarget, 0, len(targets))
	for _, target := range targets {
		normalized = append(normalized, TestTarget{
			Name:     strings.TrimSpace(target.Name),
			URL:      strings.TrimSpace(target.URL),
			Enabled:  target.Enabled,
			Profiles: normalizeTestTargetProfiles(target.Profiles),
			Order:    target.Order,
		})
	}
	return normalized
}

func normalizeTestTargetProfiles(profiles []string) []string {
	if len(profiles) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		value := strings.ToLower(strings.TrimSpace(profile))
		if value == "" || !slices.Contains(TestProfiles, value) || slices.Contains(normalized, value) {
			continue
		}
		normalized = append(normalized, value)
	}
	return normalized
}
