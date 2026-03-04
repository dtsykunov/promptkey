package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds persistent user preferences.
type Config struct {
	CaptureContext bool `json:"captureContext"`
}

func defaultConfig() Config {
	return Config{CaptureContext: true}
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config dir: %w", err)
	}
	return filepath.Join(dir, "promptkey", "config.json"), nil
}

func loadConfig() Config {
	path, err := configPath()
	if err != nil {
		debugf("loadConfig: %v", err)
		return defaultConfig()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultConfig() // not found on first run — normal
	}
	cfg := defaultConfig()
	if err := json.Unmarshal(data, &cfg); err != nil {
		debugf("loadConfig: parse error: %v", err)
		return defaultConfig()
	}
	return cfg
}

func (a *App) saveConfig() {
	path, err := configPath()
	if err != nil {
		debugf("saveConfig: %v", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		debugf("saveConfig: mkdir: %v", err)
		return
	}
	data, err := json.MarshalIndent(a.cfg, "", "  ")
	if err != nil {
		debugf("saveConfig: marshal: %v", err)
		return
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		debugf("saveConfig: write: %v", err)
	}
}
