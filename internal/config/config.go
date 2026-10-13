package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Config merepresentasikan konfigurasi utama bsdpanel.
type Config struct {
	mu            sync.RWMutex `json:"-"`
	FilePath      string       `json:"-"`
	PanelName     string       `json:"panel_name"`     // Nama kustom panel (default "BSDPanel")
	PanelDomain   string       `json:"panel_domain"`   // Domain kustom panel, misal "panel.mtsn1sekadau.sch.id"
	PanelSSL      bool         `json:"panel_ssl"`      // Opsi aktifkan SSL Let's Encrypt untuk domain panel
	ListenAddr    string       `json:"listen_addr"`    // Default ":8080" atau ":2087"
	SecretKey     string       `json:"secret_key"`     // JWT & Session secret key
	AdminUser     string       `json:"admin_user"`     // Default "pengelola"
	AdminPassword string       `json:"admin_password"` // Hashed password
	DataDir       string       `json:"data_dir"`       // "/usr/local/etc/bsdpanel" di FreeBSD
	VHostDir      string       `json:"vhost_dir"`      // "/usr/local/etc/nginx/conf.d"
	PHPPoolDir    string       `json:"php_pool_dir"`   // "/usr/local/etc/php-fpm.d"
	DefaultShell  string       `json:"default_shell"`  // "/usr/sbin/nologin"
	DemoMode      bool         `json:"demo_mode"`      // Mode simulasi jika di luar FreeBSD
}

func HashPassword(password, salt string) string {
	h := sha256.New()
	h.Write([]byte(password + "::bsdpanel::" + salt))
	return hex.EncodeToString(h.Sum(nil))
}

// DefaultConfig mengembalikan konfigurasi bawaan ramah FreeBSD.
func DefaultConfig() *Config {
	salt := "change-me-bsdpanel-secure-key-32bytes!"
	return &Config{
		PanelName:     "BSDPanel",
		PanelDomain:   "",
		PanelSSL:      false,
		ListenAddr:    ":8080",
		SecretKey:     salt,
		AdminUser:     "pengelola",
		AdminPassword: HashPassword("Suksesterus123#", salt),
		DataDir:       "/usr/local/etc/bsdpanel",
		VHostDir:      "/usr/local/etc/nginx/conf.d",
		PHPPoolDir:    "/usr/local/etc/php-fpm.d",
		DefaultShell:  "/usr/sbin/nologin",
		DemoMode:      false,
	}
}

// Load membaca konfigurasi dari berkas JSON atau membuat default jika belum ada.
func Load(cfgPath string) (*Config, error) {
	cfg := DefaultConfig()
	cfg.FilePath = cfgPath

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		_ = os.MkdirAll(filepath.Dir(cfgPath), 0750)
		_ = cfg.Save()
		return cfg, nil
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return cfg, nil
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return cfg, nil
	}
	cfg.FilePath = cfgPath

	// Jika file lama masih menggunakan admin default atau belum diisi, pastikan pengelola ada
	if cfg.AdminUser == "" || cfg.AdminUser == "admin" {
		cfg.AdminUser = "pengelola"
		cfg.AdminPassword = HashPassword("Suksesterus123#", cfg.SecretKey)
		_ = cfg.Save()
	}

	return cfg, nil
}

// VerifyPassword memeriksa kecocokan password.
func (c *Config) VerifyPassword(password string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	expectedHash := HashPassword(password, c.SecretKey)
	return c.AdminPassword == expectedHash || c.AdminPassword == password
}

// GetPanelName mengembalikan nama panel kustom atau default BSDPanel.
func (c *Config) GetPanelName() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.PanelName == "" {
		return "BSDPanel"
	}
	return c.PanelName
}

// UpdatePanelSettings memperbarui nama panel, domain, dan opsi SSL.
func (c *Config) UpdatePanelSettings(panelName, panelDomain string, panelSSL bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	pName := strings.TrimSpace(panelName)
	if pName == "" {
		pName = "BSDPanel"
	}
	c.PanelName = pName
	c.PanelDomain = strings.TrimSpace(panelDomain)
	c.PanelSSL = panelSSL

	return c.saveUnlocked()
}

// UpdateCredentials memperbarui username dan password serta menyimpannya ke disk.
func (c *Config) UpdateCredentials(newUsername, newPassword string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.AdminUser = newUsername
	if newPassword != "" {
		c.AdminPassword = HashPassword(newPassword, c.SecretKey)
	}

	return c.saveUnlocked()
}

// Save menyimpan konfigurasi ke file JSON secara aman.
func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.saveUnlocked()
}

func (c *Config) saveUnlocked() error {
	if c.FilePath == "" {
		return nil
	}
	_ = os.MkdirAll(filepath.Dir(c.FilePath), 0750)
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.FilePath, data, 0600)
}
