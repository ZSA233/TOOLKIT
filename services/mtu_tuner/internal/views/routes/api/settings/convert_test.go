package settings

import (
	"reflect"
	"testing"

	"mtu-tuner/internal/core"
)

func TestSavedSettingsProtoRoundTripIncludesTestTargets(t *testing.T) {
	t.Parallel()

	settings := core.SavedSettings{
		Version:       2,
		RouteProbe:    "1.1.1.1",
		FallbackProbe: "8.8.8.8",
		HTTPProxy:     "http://127.0.0.1:7890",
		ClashAPI:      "http://127.0.0.1:9097",
		ProxyGroup:    "PROXY",
		ConfigPath:    "/tmp/config.yaml",
		BrowserPath:   "/Applications/Chromium.app",
		TestProfile:   "chrome",
		TestTargets: []core.TestTarget{{
			Name:     "docs",
			URL:      "https://example.com/docs",
			Enabled:  true,
			Profiles: []string{"browser", "stress"},
			Order:    15,
		}},
		SweepMTUs: "1500,1400",
		TargetMTU: 1400,
	}

	got := savedSettingsCore(savedSettingsDTO(settings))
	if !reflect.DeepEqual(got, settings) {
		t.Fatalf("savedSettingsCore(savedSettingsDTO()) = %#v, want %#v", got, settings)
	}
}
