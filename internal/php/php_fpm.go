package php

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/agus-salim/bsdpanel/internal/system"
)

// PoolConfig merepresentasikan konfigurasi pool PHP-FPM per user FreeBSD.
type PoolConfig struct {
	User               string `json:"user"`
	PHPVersion         string `json:"php_version"` // misal "php81", "php82", "php83"
	SocketPath         string `json:"socket_path"`
	MaxChildren        int    `json:"max_children"`
	StartServers       int    `json:"start_servers"`
	MinSpare           int    `json:"min_spare"`
	MaxSpare           int    `json:"max_spare"`
	MemoryLimit        string `json:"memory_limit"`        // e.g. "256M"
	UploadMaxFilesize  string `json:"upload_max_filesize"` // e.g. "64M"
	PostMaxSize        string `json:"post_max_size"`       // e.g. "64M"
	MaxExecutionTime   int    `json:"max_execution_time"`  // e.g. 60
}

type Manager struct {
	poolDir string
	runner  system.Runner
	tmpl    *template.Template
}

func NewManager(poolDir string, runner system.Runner, tmplContent string) (*Manager, error) {
	t, err := template.New("php_pool").Parse(tmplContent)
	if err != nil {
		return nil, fmt.Errorf("gagal parse template php-fpm pool: %w", err)
	}
	return &Manager{
		poolDir: poolDir,
		runner:  runner,
		tmpl:    t,
	}, nil
}

// CreateOrUpdatePool membuat konfigurasi pool PHP-FPM dan mereload servicenya di FreeBSD.
func (m *Manager) CreateOrUpdatePool(cfg PoolConfig) error {
	if cfg.SocketPath == "" {
		cfg.SocketPath = fmt.Sprintf("/var/run/%s-fpm-%s.sock", cfg.PHPVersion, cfg.User)
	}

	var buf bytes.Buffer
	if err := m.tmpl.Execute(&buf, cfg); err != nil {
		return fmt.Errorf("gagal render template pool: %w", err)
	}

	// 1. Bersihkan pool lama milik user ini dari versi PHP lain agar tidak bentrok socket
	supportedVersions := []string{"php82", "php83", "php84", "php85"}
	for _, v := range supportedVersions {
		if v != cfg.PHPVersion {
			oldPoolFile := fmt.Sprintf("/usr/local/etc/%s-fpm.d/%s.conf", v, cfg.User)
			if _, err := os.Stat(oldPoolFile); err == nil {
				_ = os.Remove(oldPoolFile)
				_, _ = m.runner.ExecWithTimeout(5*time.Second, "service", fmt.Sprintf("%s_fpm", v), "reload")
			}
		}
	}
	// Bersihkan juga dari folder generic php-fpm.d
	_ = os.Remove(filepath.Join(m.poolDir, fmt.Sprintf("%s.conf", cfg.User)))

	// Hapus socket lama jika ada agar bersih saat re-bind
	if cfg.SocketPath != "" {
		_ = os.Remove(cfg.SocketPath)
	}

	// 2. Simpan di direktori pool versi spesifik (misal /usr/local/etc/php83-fpm.d)
	targetDir := fmt.Sprintf("/usr/local/etc/%s-fpm.d", cfg.PHPVersion)
	if cfg.PHPVersion == "" {
		targetDir = m.poolDir
	}
	_ = os.MkdirAll(targetDir, 0755)
	targetFile := filepath.Join(targetDir, fmt.Sprintf("%s.conf", cfg.User))
	if err := os.WriteFile(targetFile, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("gagal menulis file pool %s: %w", targetFile, err)
	}

	// 3. Restart daemon FPM versi spesifik
	if cfg.PHPVersion != "" {
		serviceName := fmt.Sprintf("%s_fpm", cfg.PHPVersion)
		res, err := m.runner.ExecWithTimeout(5*time.Second, "service", serviceName, "restart")
		if err != nil || res.ExitCode != 0 {
			_, _ = m.runner.ExecWithTimeout(5*time.Second, "service", serviceName, "start")
		}
	} else {
		_, _ = m.runner.ExecWithTimeout(5*time.Second, "service", "php_fpm", "restart")
	}

	return nil
}

// DeletePool menghapus file pool dan mereload daemon.
func (m *Manager) DeletePool(user, phpVersion string) error {
	supportedVersions := []string{"php82", "php83", "php84", "php85"}
	for _, v := range supportedVersions {
		poolFile := fmt.Sprintf("/usr/local/etc/%s-fpm.d/%s.conf", v, user)
		if _, err := os.Stat(poolFile); err == nil {
			_ = os.Remove(poolFile)
			_, _ = m.runner.ExecWithTimeout(5*time.Second, "service", fmt.Sprintf("%s_fpm", v), "reload")
		}
	}
	_ = os.Remove(filepath.Join(m.poolDir, fmt.Sprintf("%s.conf", user)))
	return nil
}
