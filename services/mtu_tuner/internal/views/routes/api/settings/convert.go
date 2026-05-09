package settings

import (
	"mtu-tuner/internal/core"
	protos "mtu-tuner/internal/views/routes/api/_gen_protos"
)

func savedSettingsProto(settings core.SavedSettings) *protos.SavedSettings {
	targets := make([]*protos.TestTarget, 0, len(settings.TestTargets))
	for _, target := range settings.TestTargets {
		targets = append(targets, &protos.TestTarget{
			Name:     target.Name,
			Url:      target.URL,
			Enabled:  target.Enabled,
			Profiles: append([]string(nil), target.Profiles...),
			Order:    target.Order,
		})
	}
	return &protos.SavedSettings{
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

func savedSettingsCore(settings *protos.SavedSettings) core.SavedSettings {
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
