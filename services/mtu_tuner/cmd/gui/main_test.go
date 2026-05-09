package main

import (
	"testing"
	"testing/fstest"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestDefaultAppOptionsTerminatesAfterLastWindowClosedOnMac(t *testing.T) {
	options := defaultAppOptions(
		fstest.MapFS{
			"index.html": {Data: []byte("<!doctype html><html></html>")},
		},
		[]application.Service{},
	)

	if !options.Mac.ApplicationShouldTerminateAfterLastWindowClosed {
		t.Fatalf("expected mac app options to terminate after the last window closes")
	}
}
