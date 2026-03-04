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

// Config is the top-level app configuration.
type Config struct {
	Providers      []Provider `json:"providers"`
	ActiveProvider string     `json:"activeProvider"`
	ResultW        int        `json:"resultWidth"`
	ResultH        int        `json:"resultHeight"`
}

const defaultResultW, defaultResultH = 480, 320

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
