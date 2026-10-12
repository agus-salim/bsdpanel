package webserver

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
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
	SSLCertPath  string   `json:"ssl_cert_path"`
	SSLKeyPath   string   `json:"ssl_key_path"`
	DBEngine     string   `json:"db_engine,omitempty"` // "mariadb" atau "postgresql"
	DBName       string   `json:"db_name,omitempty"`
	DBUser       string   `json:"db_user,omitempty"`
	DBPass       string   `json:"db_pass,omitempty"`
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

// NormalizeDocRoot memastikan direktori root selalu absolut dan terisolasi per user.
func NormalizeDocRoot(user, rawRoot string) string {
	rawRoot = strings.TrimSpace(rawRoot)
	if rawRoot == "" {
		return filepath.Join("/home", user, "public_html")
	}
	if !filepath.IsAbs(rawRoot) {
		return filepath.Join("/home", user, rawRoot)
	}
	return rawRoot
}

// EnsureNginxInclude memastikan berkas /usr/local/etc/nginx/nginx.conf memuat direktori conf.d
func EnsureNginxInclude() {
	nginxConf := "/usr/local/etc/nginx/nginx.conf"
	data, err := os.ReadFile(nginxConf)
	if err == nil {
		content := string(data)
		if !strings.Contains(content, "conf.d/*.conf") && strings.Contains(content, "http {") {
			newContent := strings.Replace(content, "http {", "http {\n    include /usr/local/etc/nginx/conf.d/*.conf;\n", 1)
			_ = os.WriteFile(nginxConf, []byte(newContent), 0644)
		}
	}
}

// CreateOrUpdate membuat berkas konfigurasi Nginx di FreeBSD, membuat folder otomatis, dan reload service.
func (m *Manager) CreateOrUpdate(vh *VHostConfig) error {
	vh.DocumentRoot = NormalizeDocRoot(vh.User, vh.DocumentRoot)

	if vh.PHPVersion != "" && vh.PHPVersion != "none" {
		vh.HasPHP = true
		if vh.PHPSocket == "" {
			vh.PHPSocket = fmt.Sprintf("/var/run/%s-fpm-%s.sock", vh.PHPVersion, vh.User)
		}
	}

	// 1. Buat folder DocumentRoot & ACME challenge secara otomatis
	_ = os.MkdirAll(vh.DocumentRoot, 0755)
	acmeChallengeDir := filepath.Join(vh.DocumentRoot, ".well-known", "acme-challenge")
	_ = os.MkdirAll(acmeChallengeDir, 0755)

	// Buat starter index.php jika folder masih kosong
	indexPath := filepath.Join(vh.DocumentRoot, "index.php")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		starterHTML := fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <title>Selamat Datang di %s</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #0b1120; color: #f8fafc; display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0; }
        .card { background: #1e293b; padding: 2.5rem; border-radius: 1rem; border: 1px solid #334155; text-align: center; max-width: 500px; box-shadow: 0 20px 25px -5px rgba(0,0,0,0.5); }
        h1 { color: #10b981; margin-bottom: 0.5rem; font-size: 1.5rem; }
        p { color: #94a3b8; font-size: 0.9rem; line-height: 1.5; }
        .badge { display: inline-block; background: #064e3b; color: #34d399; padding: 0.25rem 0.75rem; border-radius: 9999px; font-size: 0.75rem; margin-top: 1rem; }
    </style>
</head>
<body>
    <div class="card">
        <div style="font-size: 2.5rem; margin-bottom: 0.5rem;">😈</div>
        <h1>BSDPanel Web Server</h1>
        <p>Virtual host <strong>%s</strong> telah berhasil dibuat dan aktif di FreeBSD.</p>
        <span class="badge">PHP & Nginx Active</span>
    </div>
</body>
</html>
`, vh.Domain, vh.Domain)
		_ = os.WriteFile(indexPath, []byte(starterHTML), 0644)
	}

	// Berikan izin akses folder ke user dan group www FreeBSD
	_, _ = m.runner.ExecWithTimeout(5*time.Second, "chown", "-R", fmt.Sprintf("%s:www", vh.User), vh.DocumentRoot)
	_, _ = m.runner.ExecWithTimeout(5*time.Second, "chmod", "-R", "755", vh.DocumentRoot)

	// Pastikan jalur sertifikat terpasang jika SSL aktif
	if vh.SSLEnabled {
		if vh.SSLCertPath == "" {
			certPaths := []string{
				fmt.Sprintf("/usr/local/etc/letsencrypt/live/%s/fullchain.pem", vh.Domain),
				fmt.Sprintf("/etc/letsencrypt/live/%s/fullchain.pem", vh.Domain),
			}
			for _, cp := range certPaths {
				if fileExists(cp) {
					vh.SSLCertPath = cp
					break
				}
			}
			if vh.SSLCertPath == "" {
				vh.SSLCertPath = certPaths[0]
			}
		}
		if vh.SSLKeyPath == "" {
			keyPaths := []string{
				fmt.Sprintf("/usr/local/etc/letsencrypt/live/%s/privkey.pem", vh.Domain),
				fmt.Sprintf("/etc/letsencrypt/live/%s/privkey.pem", vh.Domain),
			}
			for _, kp := range keyPaths {
				if fileExists(kp) {
					vh.SSLKeyPath = kp
					break
				}
			}
			if vh.SSLKeyPath == "" {
				vh.SSLKeyPath = keyPaths[0]
			}
		}
	}

	// 2. Pastikan nginx.conf memuat include conf.d/*.conf
	EnsureNginxInclude()

	// 3. Render Template Nginx
	var buf bytes.Buffer
	if err := m.tmpl.Execute(&buf, vh); err != nil {
		return fmt.Errorf("eksekusi template gagal: %w", err)
	}

	confPath := filepath.Join(m.vhostDir, fmt.Sprintf("%s.conf", vh.Domain))
	_ = os.MkdirAll(m.vhostDir, 0755)

	if err := os.WriteFile(confPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("gagal menulis file konfigurasi %s: %w", confPath, err)
	}

	// 4. Tes konfigurasi Nginx di FreeBSD: `nginx -t`
	testRes, err := m.runner.ExecWithTimeout(5*time.Second, "nginx", "-t")
	if err != nil {
		// Rollback file jika config error
		_ = os.Remove(confPath)
		return fmt.Errorf("nginx -t gagal, konfigurasi di-rollback: %s (%w)", testRes.Stderr, err)
	}

	// 5. Reload Nginx tanpa downtime: `service nginx reload`
	_, _ = m.runner.ExecWithTimeout(5*time.Second, "service", "nginx", "reload")
	return nil
}

// IssueSSL menerbitkan sertifikat resmi Let's Encrypt menggunakan Certbot.
func (m *Manager) IssueSSL(vh *VHostConfig) error {
	domain := vh.Domain
	docRoot := NormalizeDocRoot(vh.User, vh.DocumentRoot)

	// Pastikan parent directory dan docRoot dapat diakses publik oleh Nginx (user www)
	parentDir := filepath.Dir(docRoot)
	_, _ = m.runner.ExecWithTimeout(5*time.Second, "chmod", "755", parentDir)
	_, _ = m.runner.ExecWithTimeout(5*time.Second, "chmod", "755", docRoot)

	// Pastikan direktori acme challenge ada dan dapat dibaca oleh publik
	acmeDir := filepath.Join(docRoot, ".well-known", "acme-challenge")
	_ = os.MkdirAll(acmeDir, 0755)
	_, _ = m.runner.ExecWithTimeout(5*time.Second, "chmod", "-R", "755", filepath.Join(docRoot, ".well-known"))

	// Hapus sertifikat self-signed palsu lama jika belum ada renewal config resmi Let's Encrypt
	renewalConf := fmt.Sprintf("/usr/local/etc/letsencrypt/renewal/%s.conf", domain)
	if !fileExists(renewalConf) {
		_ = os.RemoveAll(filepath.Join("/usr/local/etc/letsencrypt/live", domain))
		_ = os.RemoveAll(filepath.Join("/usr/local/etc/letsencrypt/archive", domain))
	}

	// Cek apakah certbot terpasang di FreeBSD; jika belum, otomatis install
	if !m.isCertbotInstalled() {
		_, _ = m.runner.ExecWithTimeout(120*time.Second, "pkg", "install", "-y", "security/py-certbot", "security/py-certbot-nginx")
		if !m.isCertbotInstalled() {
			_, _ = m.runner.ExecWithTimeout(120*time.Second, "pkg", "install", "-y", "py312-certbot", "py312-certbot-nginx")
		}
		if !m.isCertbotInstalled() {
			_, _ = m.runner.ExecWithTimeout(120*time.Second, "pkg", "install", "-y", "certbot")
		}
	}

	// 1. Jalankan Certbot resmi Let's Encrypt dengan webroot mode
	resWebroot, errWebroot := m.runner.ExecWithTimeout(120*time.Second, "certbot", "certonly",
		"--webroot",
		"-w", docRoot,
		"-d", domain,
		"--non-interactive",
		"--agree-tos",
		"--register-unsafely-without-email",
		"--force-renewal",
	)

	// Cari sertifikat resmi Let's Encrypt
	foundCert, foundKey := m.findValidLetsEncryptCert(domain)

	// 2. Jika webroot belum membuahkan hasil, coba metode Nginx plugin
	if foundCert == "" {
		resNginx, _ := m.runner.ExecWithTimeout(120*time.Second, "certbot", "certonly",
			"--nginx",
			"-d", domain,
			"--non-interactive",
			"--agree-tos",
			"--register-unsafely-without-email",
			"--force-renewal",
		)
		foundCert, foundKey = m.findValidLetsEncryptCert(domain)

		if foundCert == "" {
			errMsg := strings.TrimSpace(resWebroot.Stderr)
			if errMsg == "" {
				errMsg = strings.TrimSpace(resWebroot.Stdout)
			}
			if errMsg == "" && resNginx.Stderr != "" {
				errMsg = strings.TrimSpace(resNginx.Stderr)
			}
			if errMsg == "" {
				errMsg = strings.TrimSpace(resNginx.Stdout)
			}
			if errMsg == "" && errWebroot != nil {
				errMsg = errWebroot.Error()
			}
			if errMsg == "" {
				errMsg = "Certbot tidak menghasilkan sertifikat resmi Let's Encrypt. Pastikan DNS domain telah mengarah ke server ini."
			}
			return fmt.Errorf("Certbot gagal menerbitkan sertifikat Let's Encrypt: %s", errMsg)
		}
	}

	vh.SSLCertPath = foundCert
	vh.SSLKeyPath = foundKey
	vh.SSLEnabled = true

	return m.CreateOrUpdate(vh)
}

func (m *Manager) isCertbotInstalled() bool {
	res, err := m.runner.ExecWithTimeout(5*time.Second, "certbot", "--version")
	return err == nil && res.ExitCode == 0
}

func (m *Manager) findValidLetsEncryptCert(domain string) (string, string) {
	candidates := []string{
		fmt.Sprintf("/usr/local/etc/letsencrypt/live/%s", domain),
		fmt.Sprintf("/usr/local/etc/letsencrypt/live/%s-0001", domain),
		fmt.Sprintf("/etc/letsencrypt/live/%s", domain),
		fmt.Sprintf("/etc/letsencrypt/live/%s-0001", domain),
	}

	for _, dir := range candidates {
		cert := filepath.Join(dir, "fullchain.pem")
		key := filepath.Join(dir, "privkey.pem")
		if fileExists(cert) && fileExists(key) {
			if IsRealLetsEncryptCert(m.runner, cert) {
				return cert, key
			}
		}
	}
	return "", ""
}

// IsRealLetsEncryptCert memeriksa apakah berkas sertifikat benar-benar diterbitkan oleh Let's Encrypt resmi.
func IsRealLetsEncryptCert(runner system.Runner, certPath string) bool {
	if certPath == "" || !fileExists(certPath) {
		return false
	}
	res, err := runner.ExecWithTimeout(5*time.Second, "openssl", "x509", "-in", certPath, "-noout", "-issuer")
	if err != nil || res.ExitCode != 0 {
		return false
	}
	issuer := strings.ToLower(res.Stdout)
	return strings.Contains(issuer, "let's encrypt") || strings.Contains(issuer, "isrg")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
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
