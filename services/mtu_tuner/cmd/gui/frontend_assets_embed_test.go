//go:build mtu_tuner_embed_frontend

package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFrontendAssetFSEmbedDoesNotRequireRuntimeDistDirectory(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller could not resolve cmd/gui directory")
	}

	guiDir := filepath.Dir(currentFile)
	distDir := filepath.Join(guiDir, "frontend", "dist")
	indexPath := filepath.Join(distDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("frontend dist asset %s must exist before running embed test: %v", indexPath, err)
	}

	hiddenDistDir := filepath.Join(guiDir, "frontend", "dist.runtime-test-hidden")
	_ = os.RemoveAll(hiddenDistDir)
	if err := os.Rename(distDir, hiddenDistDir); err != nil {
		t.Fatalf("hide runtime dist directory: %v", err)
	}
	defer func() {
		if err := os.Rename(hiddenDistDir, distDir); err != nil {
			t.Fatalf("restore runtime dist directory: %v", err)
		}
	}()

	assetFS, err := frontendAssetFS()
	if err != nil {
		t.Fatalf("frontendAssetFS should succeed from embedded assets: %v", err)
	}

	indexHTML, err := fs.ReadFile(assetFS, "index.html")
	if err != nil {
		t.Fatalf("embedded frontend assets should expose index.html: %v", err)
	}
	if len(indexHTML) == 0 {
		t.Fatal("embedded index.html must not be empty")
	}
}
