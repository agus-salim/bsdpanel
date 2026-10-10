package config

import (
	"encoding/json"
	"os"
)

// Config represents the core settings for BSD Panel.
type Config struct {
	Host             string   `json:"host"`
	Port             int      `json:"port"`
	DataDir          string   `json:"data_dir"`
	SitesDir         string   `json:"sites_dir"`
	DefaultWebServer string   `json:"default_web_server"` // nginx, apache, litespeed, caddy
	DefaultPHP       string   `json:"default_php"`        // 8.2, 8.3, 8.4, 8.5
	DoasEnabled      bool     `json:"doas_enabled"`
	PanelDomain      string   `json:"panel_domain"`
	AllowedHosts     []string `json:"allowed_hosts"`
	SessionSecret    string   `json:"session_secret"`
}

// DefaultConfig provides safe, standard defaults tailored for FreeBSD.
func DefaultConfig() *Config {
	return &Config{
		Host:             "127.0.0.1",
		Port:             8880,
		DataDir:          "/var/db/bsdpanel",
		SitesDir:         "/usr/home",
		DefaultWebServer: "nginx",
		DefaultPHP:       "8.3",
		DoasEnabled:      true,
		PanelDomain:      "panel.local",
		AllowedHosts:     []string{"127.0.0.1", "localhost"},
		SessionSecret:    "change-this-ultra-secure-secret-key",
	}
}

// Load reads config from file or creates a default one if not exists.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		_ = Save(path, cfg)
		return cfg, nil
	} else if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save writes current config to file with secure permissions.
func Save(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
