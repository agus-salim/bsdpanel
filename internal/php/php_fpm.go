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

	_ = os.MkdirAll(m.poolDir, 0755)
	poolFilePath := filepath.Join(m.poolDir, fmt.Sprintf("%s.conf", cfg.User))

	if err := os.WriteFile(poolFilePath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("gagal menulis file pool %s: %w", poolFilePath, err)
	}

	// Reload daemon php-fpm FreeBSD: coba service spesifik lalu fallback generic php_fpm
	if cfg.PHPVersion != "" {
		serviceName := fmt.Sprintf("%s_fpm", cfg.PHPVersion)
		_, _ = m.runner.ExecWithTimeout(5*time.Second, "service", serviceName, "reload")
	}
	_, _ = m.runner.ExecWithTimeout(5*time.Second, "service", "php_fpm", "reload")
	return nil
}

// DeletePool menghapus file pool dan mereload daemon.
func (m *Manager) DeletePool(user, phpVersion string) error {
	poolFilePath := filepath.Join(m.poolDir, fmt.Sprintf("%s.conf", user))
	_ = os.Remove(poolFilePath)

	serviceName := "php_fpm"
	if phpVersion != "" {
		serviceName = fmt.Sprintf("%s_fpm", phpVersion)
	}
	_, _ = m.runner.ExecWithTimeout(5*time.Second, "service", serviceName, "reload")
	return nil
}
