package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const configFile = "h3savewatcher.json"

// Config holds persisted application state.
type Config struct {
	WatchDir         string `json:"watch_dir"`
	Language         string `json:"language"`
	InstanceID       string `json:"instance_id"`
	UpdatedToVersion string `json:"updated_to_version,omitempty"`
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
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, configFile)
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
