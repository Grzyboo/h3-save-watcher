package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const configFile = "h3savewatcher.json"

var (
	appDirOnce sync.Once
	appDir     string
)

// userConfigDir returns the user's config directory, falling back to the
// current working directory if it cannot be determined.
func userConfigDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return dir
}

// ensureAppDir returns the dedicated h3savewatcher directory. It is created
// inside the user's config directory if possible; if creation fails, the
// function falls back to the user config directory itself.
func ensureAppDir() string {
	appDirOnce.Do(func() {
		base := userConfigDir()
		candidate := filepath.Join(base, "h3savewatcher")
		if err := os.MkdirAll(candidate, 0755); err == nil {
			appDir = candidate
		} else {
			appDir = base
		}
	})
	return appDir
}

// legacyConfigPath returns the old config path directly in the user's config
// directory, used for one-time migration.
func legacyConfigPath() string {
	return filepath.Join(userConfigDir(), configFile)
}

// migrateLegacyConfigAndCache moves config and cache files from the old
// location (directly in the user's config directory) into the dedicated
// h3savewatcher directory. If a file already exists at the new location, the
// old file is left untouched so existing data is never overwritten.
func migrateLegacyConfigAndCache() {
	newConfig := configPath()
	newCache := cachePath()
	oldConfig := legacyConfigPath()
	oldCache := legacyCachePath()

	if _, err := os.Stat(newConfig); os.IsNotExist(err) {
		if _, err := os.Stat(oldConfig); err == nil {
			_ = os.Rename(oldConfig, newConfig)
		}
	}
	if _, err := os.Stat(newCache); os.IsNotExist(err) {
		if _, err := os.Stat(oldCache); err == nil {
			_ = os.Rename(oldCache, newCache)
		}
	}
}

// Config holds persisted application state.
type Config struct {
	WatchDir            string `json:"watch_dir"`
	Language            string `json:"language"`
	InstanceID          string `json:"instance_id"`
	UpdatedToVersion    string `json:"updated_to_version,omitempty"`
	InitialRunCompleted bool   `json:"initial_run_completed"`
}

// ensureInstanceID generates and persists a UUID v4 if one is not already set.
func ensureInstanceID(cfg *Config) {
	if cfg.InstanceID != "" {
		return
	}
	var b [16]byte
	_, _ = rand.Read(b[:])
	// Set version 4 and variant bits.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	cfg.InstanceID = fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
	saveConfig(*cfg)
}

func configPath() string {
	return filepath.Join(ensureAppDir(), configFile)
}

func loadConfig() Config {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return Config{}
	}
	var cfg Config
	_ = json.Unmarshal(data, &cfg)
	return cfg
}

func saveConfig(cfg Config) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return
	}
	_ = os.WriteFile(configPath(), data, 0644)
}

// setPendingUpdateVersion stores the version the application is about to be
// updated to, so the new process can show a one-time notification after restart.
func setPendingUpdateVersion(version string) {
	cfg := loadConfig()
	cfg.UpdatedToVersion = version
	saveConfig(cfg)
}

// clearPendingUpdateVersion removes the pending update version from the config.
func clearPendingUpdateVersion() {
	cfg := loadConfig()
	cfg.UpdatedToVersion = ""
	saveConfig(cfg)
}
