package settingsstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"mtu-tuner/internal/core"
)

type Store struct {
	path string
}

func New(path string) (*Store, error) {
	if path != "" {
		return &Store{path: path}, nil
	}
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user config dir: %w", err)
	}
	return &Store{
		path: resolveDefaultPath(userConfigDir),
	}, nil
}

func resolveDefaultPath(userConfigDir string) string {
	publicPath := filepath.Join(userConfigDir, core.ConfigDirName, "config.json")
	legacyPath := filepath.Join(userConfigDir, core.LegacyConfigDirName, "config.json")
	if _, err := os.Stat(publicPath); err == nil {
		return publicPath
	}
	if _, err := os.Stat(legacyPath); err == nil {
		return legacyPath
	}
	return publicPath
}

func (store *Store) Path() string {
	return store.path
}

func (store *Store) Load() (core.SavedSettings, error) {
	defaults := core.DefaultSavedSettings()
	if store == nil || store.path == "" {
		return defaults, nil
	}
	data, err := os.ReadFile(store.path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaults, nil
		}
		return defaults, fmt.Errorf("read settings file: %w", err)
	}

	var settings core.SavedSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return defaults, fmt.Errorf("decode settings file: %w", err)
	}
	return core.NormalizeSavedSettings(settings), nil
}

func (store *Store) Save(settings core.SavedSettings) (core.SavedSettings, error) {
	normalized := core.NormalizeSavedSettings(settings)
	if store == nil || store.path == "" {
		return normalized, nil
	}

	if err := os.MkdirAll(filepath.Dir(store.path), 0o755); err != nil {
		return normalized, fmt.Errorf("create settings dir: %w", err)
	}

	payload, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return normalized, fmt.Errorf("encode settings file: %w", err)
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(store.path, payload, 0o644); err != nil {
		return normalized, fmt.Errorf("write settings file: %w", err)
	}
	return normalized, nil
}
