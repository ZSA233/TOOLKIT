package settings

import (
	"strings"
	"testing"
)

func TestSaveCurrentSettingsRejectsMissingSettingsPayload(t *testing.T) {
	t.Parallel()

	router := &Router{}
	if _, err := router.SaveCurrentSettings(nil, &REQ_SaveCurrentSettings{}); err == nil {
		t.Fatal("SaveCurrentSettings() error = nil, want payload error")
	} else if !strings.Contains(err.Error(), "settings payload is required") {
		t.Fatalf("SaveCurrentSettings() error = %q, want payload error", err.Error())
	}
}
