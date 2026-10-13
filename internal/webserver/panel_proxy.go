package webserver

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

type panelProxyTemplateData struct {
	Domain      string
	Port        string
	SSLEnabled  bool
	SSLCertPath string
	SSLKeyPath  string
	ACMEDir     string
}

const panelProxyTmplContent = `# ==============================================================================
# FreeBSD Nginx Reverse Proxy for BSDPanel
# Managed automatically by bsdpanel
# Domain: {{.Domain}}
# ==============================================================================

server {
    listen 80;
    listen [::]:80;
    server_name {{.Domain}};

    # Let's Encrypt ACME Challenge
    location /.well-known/acme-challenge/ {
        root {{.ACMEDir}};
        try_files $uri =404;
    }

    {{if .SSLEnabled}}
    # Redirect seluruh HTTP ke HTTPS
    location / {
        return 301 https://$host$request_uri;
    }
    {{else}}
    location / {
        proxy_pass http://127.0.0.1:{{.Port}};
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }
    {{end}}
}

{{if .SSLEnabled}}
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name {{.Domain}};

    ssl_certificate     {{.SSLCertPath}};
    ssl_certificate_key {{.SSLKeyPath}};
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # Security Headers
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "SAMEORIGIN" always;

    location / {
        proxy_pass http://127.0.0.1:{{.Port}};
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }
}
{{end}}
`

// ConfigurePanelProxy mengonfigurasi reverse proxy Nginx untuk akses domain khusus BSDPanel dan SSL.
func (m *Manager) ConfigurePanelProxy(domain, rawPort string, sslEnabled bool) (bool, error) {
	confPath := filepath.Join(m.vhostDir, "bsdpanel_proxy.conf")

	domain = strings.TrimSpace(strings.ToLower(domain))
	if domain == "" {
		// Hapus konfigurasi reverse proxy panel jika domain dikosongkan
		if fileExists(confPath) {
			_ = os.Remove(confPath)
			if m.runner != nil {
				_, _ = m.runner.ExecWithTimeout(5*time.Second, "nginx", "-t")
				_, _ = m.runner.ExecWithTimeout(5*time.Second, "service", "nginx", "reload")
			}
		}
		return false, nil
	}

	port := strings.TrimPrefix(strings.TrimSpace(rawPort), ":")
	if port == "" {
		port = "8080"
	}

	tmpl, err := template.New("panelProxy").Parse(panelProxyTmplContent)
	if err != nil {
		return false, fmt.Errorf("gagal parse template proxy panel: %w", err)
	}

	acmeDir := "/usr/local/www/bsdpanel_acme"
	_ = os.MkdirAll(filepath.Join(acmeDir, ".well-known", "acme-challenge"), 0755)
	if m.runner != nil {
		_, _ = m.runner.ExecWithTimeout(5*time.Second, "chmod", "-R", "755", acmeDir)
	}

	data := panelProxyTemplateData{
		Domain:     domain,
		Port:       port,
		SSLEnabled: false,
		ACMEDir:    acmeDir,
	}

	// 1. Tulis konfigurasi HTTP awal port 80 untuk verifikasi Let's Encrypt / akses HTTP
	var bufHTTP bytes.Buffer
	if err := tmpl.Execute(&bufHTTP, data); err != nil {
		return false, err
	}
	_ = os.MkdirAll(m.vhostDir, 0755)
	if err := os.WriteFile(confPath, bufHTTP.Bytes(), 0644); err != nil {
		return false, fmt.Errorf("gagal menulis berkas konfigurasi Nginx %s: %w", confPath, err)
	}

	if m.runner != nil {
		if testRes, err := m.runner.ExecWithTimeout(5*time.Second, "nginx", "-t"); err != nil {
			_ = os.Remove(confPath)
			return false, fmt.Errorf("nginx -t gagal: %s", testRes.Stderr)
		}
		_, _ = m.runner.ExecWithTimeout(5*time.Second, "service", "nginx", "reload")
	}

	// 2. Jika SSL diaktifkan, terbitkan sertifikat Let's Encrypt
	if sslEnabled {
		certPath, keyPath := m.findValidLetsEncryptCert(domain)

		if certPath == "" && m.runner != nil {
			// Pastikan certbot terpasang
			if !m.isCertbotInstalled() {
				_, _ = m.runner.ExecWithTimeout(120*time.Second, "pkg", "install", "-y", "security/py-certbot")
			}

			// Jalankan certbot webroot mode
			resCert, _ := m.runner.ExecWithTimeout(120*time.Second, "certbot", "certonly",
				"--webroot",
				"-w", acmeDir,
				"-d", domain,
				"--non-interactive",
				"--agree-tos",
				"--register-unsafely-without-email",
				"--force-renewal",
			)

			certPath, keyPath = m.findValidLetsEncryptCert(domain)
			if certPath == "" {
				// Coba fallback dengan nginx plugin
				_, _ = m.runner.ExecWithTimeout(120*time.Second, "certbot", "certonly",
					"--nginx",
					"-d", domain,
					"--non-interactive",
					"--agree-tos",
					"--register-unsafely-without-email",
					"--force-renewal",
				)
				certPath, keyPath = m.findValidLetsEncryptCert(domain)
			}

			if certPath == "" {
				errMsg := strings.TrimSpace(resCert.Stderr)
				if errMsg == "" {
					errMsg = strings.TrimSpace(resCert.Stdout)
				}
				if errMsg == "" {
					errMsg = "DNS domain belum terarah ke IP server ini."
				}
				return false, fmt.Errorf("Domain berhasil diarahkan via HTTP, namun SSL Let's Encrypt gagal diterbitkan: %s", errMsg)
			}
		}

		if certPath != "" && keyPath != "" {
			data.SSLEnabled = true
			data.SSLCertPath = certPath
			data.SSLKeyPath = keyPath

			var bufSSL bytes.Buffer
			if err := tmpl.Execute(&bufSSL, data); err == nil {
				_ = os.WriteFile(confPath, bufSSL.Bytes(), 0644)
				if m.runner != nil {
					_, _ = m.runner.ExecWithTimeout(5*time.Second, "nginx", "-t")
					_, _ = m.runner.ExecWithTimeout(5*time.Second, "service", "nginx", "reload")
				}
			}
			return true, nil
		}
	}

	return false, nil
}
