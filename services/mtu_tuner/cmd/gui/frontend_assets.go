package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

func frontendAssetFS() (fs.FS, error) {
	assetDir, err := frontendAssetDir()
	if err != nil {
		return nil, err
	}
	return os.DirFS(assetDir), nil
}

func frontendAssetDir() (string, error) {
	executablePath, err := os.Executable()
	if err == nil {
		executableDir := filepath.Dir(executablePath)
		for _, candidate := range []string{
			filepath.Join(executableDir, "dist"),
			filepath.Join(executableDir, "frontend", "dist"),
		} {
			if frontendAssetIndexExists(candidate) {
				return candidate, nil
			}
		}
	}

	// Keep `go run ./cmd/gui` working inside the repo when the frontend was built in place.
	_, currentFile, _, ok := runtime.Caller(0)
	if ok {
		sourceDistDir := filepath.Join(filepath.Dir(currentFile), "frontend", "dist")
		if frontendAssetIndexExists(sourceDistDir) {
			return sourceDistDir, nil
		}
	}

	return "", fmt.Errorf("frontend dist assets not found; build cmd/gui/frontend first")
}

func frontendAssetIndexExists(assetDir string) bool {
	info, err := os.Stat(filepath.Join(assetDir, "index.html"))
	return err == nil && !info.IsDir()
}
