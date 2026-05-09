package settingsstore

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"mtu-tuner/internal/core"
)

func TestStoreLoadMissingReturnsDefaults(t *testing.T) {
	t.Parallel()

	store, err := New(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := core.DefaultSavedSettings()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}
}

func TestStoreSaveAndLoadRoundTrip(t *testing.T) {
	t.Parallel()

	store, err := New(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	saved, err := store.Save(core.SavedSettings{
		RouteProbe:    "8.8.8.8",
		FallbackProbe: "1.0.0.1",
		HTTPProxy:     "http://127.0.0.1:8899",
		ClashAPI:      "http://127.0.0.1:19097",
		ProxyGroup:    "PROXY",
		ConfigPath:    "/tmp/clash.yaml",
		BrowserPath:   "/Applications/Chromium.app",
		TestProfile:   "browser",
		TestTargets: []core.TestTarget{
			{
				Name:     "docs",
				URL:      "https://example.com/docs",
				Enabled:  true,
				Profiles: []string{"browser", "stress"},
				Order:    10,
			},
		},
		SweepMTUs: "1500,1400",
		TargetMTU: 1380,
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	reloaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !reflect.DeepEqual(reloaded, saved) {
		t.Fatalf("Load() = %#v, want %#v", reloaded, saved)
	}
}

func TestStoreLoadMigratesLegacyConfigWithoutTestTargets(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{
  "version": 1,
  "route_probe": "1.1.1.1",
  "fallback_probe": "8.8.8.8",
  "http_proxy": "http://127.0.0.1:7890",
  "clash_api": "http://127.0.0.1:9097",
  "proxy_group": "PROXY",
  "config_path": "",
  "browser_path": "",
  "test_profile": "chrome",
  "sweep_mtus": "1500,1400",
  "target_mtu": 1400
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store, err := New(configPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Version != core.ConfigVersion {
		t.Fatalf("Load().Version = %d, want %d", got.Version, core.ConfigVersion)
	}
	if !reflect.DeepEqual(got.TestTargets, core.DefaultSavedSettings().TestTargets) {
		t.Fatalf("Load().TestTargets = %#v, want defaults %#v", got.TestTargets, core.DefaultSavedSettings().TestTargets)
	}
}

func TestResolveDefaultPathPrefersLegacyConfigWhenPresent(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()
	legacyPath := filepath.Join(configRoot, "mtu-quick-tuner", "config.json")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(legacyPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got := resolveDefaultPath(configRoot)

	if got != legacyPath {
		t.Fatalf("resolveDefaultPath() = %q, want legacy path %q", got, legacyPath)
	}
}

func TestResolveDefaultPathUsesPublicToolDirectoryByDefault(t *testing.T) {
	t.Parallel()

	configRoot := t.TempDir()

	got := resolveDefaultPath(configRoot)
	want := filepath.Join(configRoot, "mtu-tuner", "config.json")
	if got != want {
		t.Fatalf("resolveDefaultPath() = %q, want %q", got, want)
	}
}
