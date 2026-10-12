package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config merepresentasikan konfigurasi utama bsdpanel.
type Config struct {
	ListenAddr   string `json:"listen_addr"`   // Default ":8080" atau ":2087"
	SecretKey    string `json:"secret_key"`    // JWT & Session secret key
	AdminUser    string `json:"admin_user"`    // Default "admin"
	AdminPassHash string `json:"admin_pass"`   // Argon2 / bcrypt hash
	DataDir      string `json:"data_dir"`      // "/usr/local/etc/bsdpanel" di FreeBSD
	VHostDir     string `json:"vhost_dir"`     // "/usr/local/etc/nginx/conf.d"
	PHPPoolDir   string `json:"php_pool_dir"`  // "/usr/local/etc/php-fpm.d"
	DefaultShell string `json:"default_shell"` // "/usr/sbin/nologin"
	DemoMode     bool   `json:"demo_mode"`     // Mode simulasi jika di luar FreeBSD
}

// DefaultConfig mengembalikan konfigurasi bawaan ramah FreeBSD.
func DefaultConfig() *Config {
	return &Config{
		ListenAddr:   ":8080",
		SecretKey:    "change-me-bsdpanel-secure-key-32bytes!",
		AdminUser:    "admin",
		AdminPassHash: "$2a$12$DkY68FqL6z2Hj1lQ/pGkO.1H5jW/gM9mUv5GZl0y7bK4Jz...", // default "admin123"
		DataDir:      "/usr/local/etc/bsdpanel",
		VHostDir:     "/usr/local/etc/nginx/conf.d",
		PHPPoolDir:   "/usr/local/etc/php-fpm.d",
		DefaultShell: "/usr/sbin/nologin",
		DemoMode:     false,
	}
}

// Load membaca konfigurasi dari berkas JSON atau membuat default jika belum ada.
func Load(cfgPath string) (*Config, error) {
	cfg := DefaultConfig()

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		// Buat direktori dan tulis default config
		if err := os.MkdirAll(filepath.Dir(cfgPath), 0750); err != nil {
			return cfg, nil // fallback in-memory jika belum ada akses root
		}
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err == nil {
			_ = os.WriteFile(cfgPath, data, 0600)
		}
		return cfg, nil
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
