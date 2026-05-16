package settings

import (
	"mtu-tuner/internal/core"
	apitypes "mtu-tuner/internal/views/routes/api/_gen_types"
)

func savedSettingsDTO(settings core.SavedSettings) *apitypes.SavedSettings {
	targets := make([]*apitypes.TestTarget, 0, len(settings.TestTargets))
	for _, target := range settings.TestTargets {
		targets = append(targets, &apitypes.TestTarget{
			Name:     target.Name,
			Url:      target.URL,
			Enabled:  target.Enabled,
			Profiles: append([]string(nil), target.Profiles...),
			Order:    target.Order,
		})
	}
	return &apitypes.SavedSettings{
		Version:       settings.Version,
		RouteProbe:    settings.RouteProbe,
		FallbackProbe: settings.FallbackProbe,
		HttpProxy:     settings.HTTPProxy,
		ClashApi:      settings.ClashAPI,
		ProxyGroup:    settings.ProxyGroup,
		ConfigPath:    settings.ConfigPath,
		BrowserPath:   settings.BrowserPath,
		TestProfile:   settings.TestProfile,
		TestTargets:   targets,
		SweepMtus:     settings.SweepMTUs,
		TargetMtu:     settings.TargetMTU,
	}
}

func savedSettingsCore(settings *apitypes.SavedSettings) core.SavedSettings {
	if settings == nil {
		return core.DefaultSavedSettings()
	}
	targets := make([]core.TestTarget, 0, len(settings.TestTargets))
	for _, target := range settings.TestTargets {
		if target == nil {
			continue
		}
		targets = append(targets, core.TestTarget{
			Name:     target.Name,
			URL:      target.Url,
			Enabled:  target.Enabled,
			Profiles: append([]string(nil), target.Profiles...),
			Order:    target.Order,
		})
	}
	return core.SavedSettings{
		Version:       settings.Version,
		RouteProbe:    settings.RouteProbe,
		FallbackProbe: settings.FallbackProbe,
		HTTPProxy:     settings.HttpProxy,
		ClashAPI:      settings.ClashApi,
		ProxyGroup:    settings.ProxyGroup,
		ConfigPath:    settings.ConfigPath,
		BrowserPath:   settings.BrowserPath,
		TestProfile:   settings.TestProfile,
		TestTargets:   targets,
		SweepMTUs:     settings.SweepMtus,
		TargetMTU:     settings.TargetMtu,
	}
}
