package core

import (
	"reflect"
	"testing"
)

func TestDefaultSavedSettingsIncludeConfiguredTestTargets(t *testing.T) {
	t.Parallel()

	settings := DefaultSavedSettings()
	if got := len(settings.TestTargets); got != 10 {
		t.Fatalf("len(DefaultSavedSettings().TestTargets) = %d, want 10", got)
	}
	if settings.TestTargets[0].Name != "yt_page" {
		t.Fatalf("DefaultSavedSettings().TestTargets[0].Name = %q, want yt_page", settings.TestTargets[0].Name)
	}
	if !settings.TestTargets[0].Enabled {
		t.Fatal("DefaultSavedSettings().TestTargets[0].Enabled = false, want true")
	}
	if !reflect.DeepEqual(settings.TestTargets[0].Profiles, []string{"browser", "stress", "quick"}) {
		t.Fatalf("DefaultSavedSettings().TestTargets[0].Profiles = %#v, want browser/stress/quick", settings.TestTargets[0].Profiles)
	}
}

func TestNormalizeSavedSettingsFillsDefaultTargetsWhenMissing(t *testing.T) {
	t.Parallel()

	normalized := NormalizeSavedSettings(SavedSettings{
		RouteProbe: "8.8.8.8",
	})
	if got := len(normalized.TestTargets); got != len(DefaultSavedSettings().TestTargets) {
		t.Fatalf("len(NormalizeSavedSettings().TestTargets) = %d, want %d", got, len(DefaultSavedSettings().TestTargets))
	}

	normalized = NormalizeSavedSettings(SavedSettings{
		TestTargets: []TestTarget{},
	})
	if got := len(normalized.TestTargets); got != len(DefaultSavedSettings().TestTargets) {
		t.Fatalf("len(NormalizeSavedSettings(empty).TestTargets) = %d, want %d", got, len(DefaultSavedSettings().TestTargets))
	}
}

func TestNormalizeSavedSettingsPreservesConfiguredTargets(t *testing.T) {
	t.Parallel()

	normalized := NormalizeSavedSettings(SavedSettings{
		TestTargets: []TestTarget{{
			Name:     "  Docs  ",
			URL:      " https://example.com/docs ",
			Enabled:  true,
			Profiles: []string{" stress ", "browser", "stress", "invalid"},
			Order:    40,
		}},
	})

	want := []TestTarget{{
		Name:     "Docs",
		URL:      "https://example.com/docs",
		Enabled:  true,
		Profiles: []string{"stress", "browser"},
		Order:    40,
	}}
	if !reflect.DeepEqual(normalized.TestTargets, want) {
		t.Fatalf("NormalizeSavedSettings().TestTargets = %#v, want %#v", normalized.TestTargets, want)
	}
}

func TestParseMTUList(t *testing.T) {
	t.Parallel()

	got, err := ParseMTUList("1500, 1480 1460")
	if err != nil {
		t.Fatalf("ParseMTUList() error = %v", err)
	}

	want := []int{1500, 1480, 1460}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseMTUList() = %#v, want %#v", got, want)
	}
}

func TestParseMTUListRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	if _, err := ParseMTUList("1500,100"); err == nil {
		t.Fatal("ParseMTUList() expected error for MTU below minimum")
	}
	if _, err := ParseMTUList(""); err == nil {
		t.Fatal("ParseMTUList() expected error for empty input")
	}
}
