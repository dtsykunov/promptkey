package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Provider holds connection details for one AI provider.
type Provider struct {
	Name         string `json:"name"`
	BaseURL      string `json:"baseURL"`
	APIKey       string `json:"apiKey"`
	Model        string `json:"model"`
	SystemPrompt string `json:"systemPrompt"`
}

// ContextConfig controls which context variables are captured on hotkey press.
type ContextConfig struct {
	Initialized bool `json:"initialized"` // sentinel: false = apply defaults on first load
	Enabled     bool `json:"enabled"`
	Clipboard   bool `json:"clipboard"`
	ActiveApp   bool `json:"activeApp"`
	DateTime    bool `json:"dateTime"`
	OSLocale    bool `json:"osLocale"`
}

func defaultContextConfig() ContextConfig {
	return ContextConfig{
		Initialized: true,
		Enabled:     true,
		Clipboard:   true,
		ActiveApp:   true,
		DateTime:    true,
		OSLocale:    true,
	}
}

// Config is the top-level app configuration.
type Config struct {
	Hotkey         string        `json:"hotkey"`
	Providers      []Provider    `json:"providers"`
	ActiveProvider string        `json:"activeProvider"`
	ResultW        int           `json:"resultWidth"`
	ResultH        int           `json:"resultHeight"`
	Context        ContextConfig `json:"context"`
}

const defaultResultW, defaultResultH = 480, 320
const defaultHotkey = "ctrl+alt+`"

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
		return Config{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}
	}
	return cfg
}

func (a *App) saveConfig() {
	path, err := configPath()
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	data, err := json.MarshalIndent(a.cfg, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

func (a *App) activeProvider() (Provider, error) {
	if a.cfg.ActiveProvider == "" || len(a.cfg.Providers) == 0 {
		return Provider{}, fmt.Errorf("no provider configured — edit %%APPDATA%%\\promptkey\\config.json")
	}
	for _, p := range a.cfg.Providers {
		if p.Name == a.cfg.ActiveProvider {
			return p, nil
		}
	}
	return Provider{}, fmt.Errorf("active provider %q not found in providers list", a.cfg.ActiveProvider)
}
