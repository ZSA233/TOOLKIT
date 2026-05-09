package app

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"mtu-tuner/internal/core"
	"mtu-tuner/internal/infra/settingsstore"
)

func TestPrepareTestRunRequestLoadsConfiguredTargetsFromSettings(t *testing.T) {
	t.Parallel()

	store, err := settingsstore.New(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	customTargets := []core.TestTarget{{
		Name:     "docs",
		URL:      "https://example.com/docs",
		Enabled:  true,
		Profiles: []string{"browser"},
		Order:    25,
	}}
	if _, err := store.Save(core.SavedSettings{
		TestTargets: customTargets,
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	service := &Service{settings: store}
	request, err := service.prepareTestRunRequest(context.Background(), core.TestRunRequest{
		TestProfile: "browser",
	})
	if err != nil {
		t.Fatalf("prepareTestRunRequest() error = %v", err)
	}
	if !reflect.DeepEqual(request.TestTargets, customTargets) {
		t.Fatalf("prepareTestRunRequest().TestTargets = %#v, want %#v", request.TestTargets, customTargets)
	}
}
