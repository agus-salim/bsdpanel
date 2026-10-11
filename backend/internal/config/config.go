package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config merepresentasikan seluruh konfigurasi server BSDPanel
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	FreeBSD  FreeBSDConfig  `yaml:"freebsd"`
	Auth     AuthConfig     `yaml:"auth"`
}

type ServerConfig struct {
	Host string `yaml:"host"` // misal "0.0.0.0"
	Port int    `yaml:"port"` // default 8083
	TLS  struct {
		Enabled  bool   `yaml:"enabled"`
		CertFile string `yaml:"cert_file"`
		KeyFile  string `yaml:"key_file"`
	} `yaml:"tls"`
}

type DatabaseConfig struct {
	Driver string `yaml:"driver"` // "sqlite"
	DSN    string `yaml:"dsn"`    // "/var/db/bsdpanel/panel.db"
}

type FreeBSDConfig struct {
	ZfsPool        string   `yaml:"zfs_pool"`        // misal "zroot"
	ZfsBaseDataset string   `yaml:"zfs_base_dataset"` // misal "zroot/bsdpanel"
	JailRoot       string   `yaml:"jail_root"`       // misal "/usr/jails"
	DefaultWebSrv  string   `yaml:"default_web_srv"` // "nginx", "apache", "openlitespeed"
	PhpVersions    []string `yaml:"php_versions"`    // ["8.1", "8.2", "8.3"]
	PfConfigPath   string   `yaml:"pf_config_path"`  // "/etc/pf.conf"
}

type AuthConfig struct {
	JWTSecret       string `yaml:"jwt_secret"`
	TokenExpiration int    `yaml:"token_expiration_hours"` // default 24 jam
}

// LoadConfig membaca file yaml dari path yang ditentukan
func LoadConfig(path string) (*Config, error) {
	// Fallback ke default config jika file belum ada
	cfg := &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8083,
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			DSN:    "/var/db/bsdpanel/panel.db",
		},
		FreeBSD: FreeBSDConfig{
			ZfsPool:        "zroot",
			ZfsBaseDataset: "zroot/bsdpanel",
			JailRoot:       "/usr/jails",
			DefaultWebSrv:  "nginx",
			PhpVersions:    []string{"8.1", "8.2", "8.3"},
			PfConfigPath:   "/etc/pf.conf",
		},
		Auth: AuthConfig{
			JWTSecret:       "bsdpanel-super-secret-key-change-in-production",
			TokenExpiration: 24,
		},
	}

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("gagal membaca file konfigurasi: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("gagal parse file konfigurasi YAML: %w", err)
	}

	return cfg, nil
}
