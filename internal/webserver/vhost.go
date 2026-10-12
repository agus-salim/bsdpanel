package webserver

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/agus-salim/bsdpanel/internal/system"
)

// VHostConfig merepresentasikan konfigurasi sebuah Virtual Host.
type VHostConfig struct {
	Domain       string   `json:"domain"`
	Aliases      []string `json:"aliases"`
	User         string   `json:"user"`
	DocumentRoot string   `json:"document_root"`
	Webserver    string   `json:"webserver"` // "nginx" atau "apache24"
	PHPVersion   string   `json:"php_version"` // "php81", "php82", "php83", atau "none"
	PHPSocket    string   `json:"php_socket"`
	HasPHP       bool     `json:"has_php"`
	SSLEnabled   bool     `json:"ssl_enabled"`
	CreatedAt    string   `json:"created_at"`
}

type Manager struct {
	vhostDir string
	runner   system.Runner
	tmpl     *template.Template
}

func NewManager(vhostDir string, runner system.Runner, tmplContent string) (*Manager, error) {
	t, err := template.New("vhost").Parse(tmplContent)
	if err != nil {
		return nil, fmt.Errorf("gagal parse template vhost: %w", err)
	}
	return &Manager{
		vhostDir: vhostDir,
		runner:   runner,
		tmpl:     t,
	}, nil
}

// CreateOrUpdate membuat berkas konfigurasi Nginx di FreeBSD dan reload service.
func (m *Manager) CreateOrUpdate(vh VHostConfig) error {
	if vh.DocumentRoot == "" {
		vh.DocumentRoot = filepath.Join("/home", vh.User, "public_html")
	}

	if vh.PHPVersion != "" && vh.PHPVersion != "none" {
		vh.HasPHP = true
		if vh.PHPSocket == "" {
			vh.PHPSocket = fmt.Sprintf("/var/run/%s-fpm-%s.sock", vh.PHPVersion, vh.User)
		}
	}

	var buf bytes.Buffer
	if err := m.tmpl.Execute(&buf, vh); err != nil {
		return fmt.Errorf("eksekusi template gagal: %w", err)
	}

	confPath := filepath.Join(m.vhostDir, fmt.Sprintf("%s.conf", vh.Domain))
	_ = os.MkdirAll(m.vhostDir, 0755)

	if err := os.WriteFile(confPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("gagal menulis file konfigurasi %s: %w", confPath, err)
	}

	// Tes konfigurasi Nginx di FreeBSD: `nginx -t`
	testRes, err := m.runner.ExecWithTimeout(5*time.Second, "nginx", "-t")
	if err != nil {
		// Rollback file jika config error
		_ = os.Remove(confPath)
		return fmt.Errorf("nginx -t gagal, konfigurasi di-rollback: %s (%w)", testRes.Stderr, err)
	}

	// Reload Nginx tanpa downtime: `service nginx reload`
	_, _ = m.runner.ExecWithTimeout(5*time.Second, "service", "nginx", "reload")
	return nil
}

// Delete menghapus berkas konfigurasi VHost dan reload Nginx.
func (m *Manager) Delete(domain string) error {
	confPath := filepath.Join(m.vhostDir, fmt.Sprintf("%s.conf", domain))
	if err := os.Remove(confPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("gagal menghapus config %s: %w", confPath, err)
	}

	_, _ = m.runner.ExecWithTimeout(5*time.Second, "service", "nginx", "reload")
	return nil
}
