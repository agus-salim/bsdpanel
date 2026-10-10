package sites

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bsdpanel/internal/system"
)

// Site represents a configured domain and virtual host.
type Site struct {
	Domain          string    `json:"domain"`
	SystemUser      string    `json:"system_user"`
	WebServer       string    `json:"web_server"` // nginx, apache, caddy, litespeed
	PHPVersion      string    `json:"php_version"` // 8.2, 8.3, 8.4, 8.5
	DocumentRoot    string    `json:"document_root"`
	SSLEnabled      bool      `json:"ssl_enabled"`
	IsReverseProxy  bool      `json:"is_reverse_proxy"`
	ProxyUpstream   string    `json:"proxy_upstream"`
	CreatedAt       time.Time `json:"created_at"`
	IsActive        bool      `json:"is_active"`
}

// SiteManager coordinates site lifecycle, system user isolation, and web server configs.
type SiteManager struct {
	userMgr    *system.UserManager
	serviceMgr *system.ServiceManager
	exec       *system.Executor
	dataDir    string
	vhostDir   string
	phpPoolDir string
}

// NewSiteManager initializes a site manager.
func NewSiteManager(userMgr *system.UserManager, serviceMgr *system.ServiceManager, exec *system.Executor, dataDir string) *SiteManager {
	return &SiteManager{
		userMgr:    userMgr,
		serviceMgr: serviceMgr,
		exec:       exec,
		dataDir:    dataDir,
		vhostDir:   "/usr/local/etc/nginx/conf.d",
		phpPoolDir: "/usr/local/etc/php-fpm.d",
	}
}

// CreateSite sets up a dedicated FreeBSD system user, directory structure, PHP pool, and webserver vhost.
func (m *SiteManager) CreateSite(ctx context.Context, site *Site) error {
	if site.Domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	// Generate clean system user name if not specified
	if site.SystemUser == "" {
		cleanDomain := strings.ReplaceAll(strings.Split(site.Domain, ".")[0], "-", "_")
		if len(cleanDomain) > 12 {
			cleanDomain = cleanDomain[:12]
		}
		site.SystemUser = fmt.Sprintf("u_%s", cleanDomain)
	}

	homeDir := filepath.Join("/usr/home", site.SystemUser)
	site.DocumentRoot = filepath.Join(homeDir, "public_html")
	site.CreatedAt = time.Now()
	site.IsActive = true

	// Step 1: Create isolated FreeBSD user and web structure
	if err := m.userMgr.CreateSiteUser(ctx, site.SystemUser, homeDir); err != nil {
		return fmt.Errorf("user creation error: %w", err)
	}

	// Step 2: Create a starter index.php or index.html in public_html
	indexFile := filepath.Join(site.DocumentRoot, "index.php")
	starterContent := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Welcome to %s</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #0f172a; color: #f8fafc; display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0; }
        .card { background: #1e293b; padding: 2.5rem; border-radius: 1rem; border: 1px solid #334155; text-align: center; max-width: 500px; box-shadow: 0 10px 25px -5px rgba(0,0,0,0.3); }
        h1 { color: #38bdf8; margin-top: 0; font-size: 1.75rem; }
        .badge { display: inline-block; background: #0284c7; color: white; padding: 0.25rem 0.75rem; border-radius: 9999px; font-size: 0.75rem; font-weight: 600; text-transform: uppercase; margin-bottom: 1rem; }
        p { color: #94a3b8; font-size: 0.95rem; line-height: 1.5; }
    </style>
</head>
<body>
    <div class="card">
        <span class="badge">FreeBSD Managed Site</span>
        <h1>%s is Live!</h1>
        <p>This virtual host is successfully deployed on FreeBSD with non-root user isolation (User: <code>%s</code>) and PHP <code>%s</code>.</p>
    </div>
</body>
</html>`, site.Domain, site.Domain, site.SystemUser, site.PHPVersion)

	_ = os.WriteFile(indexFile, []byte(starterContent), 0644)
	_, _ = m.exec.Execute(ctx, "/usr/sbin/chown", fmt.Sprintf("%s:%s", site.SystemUser, site.SystemUser), indexFile)

	// Step 3: Generate PHP-FPM Pool configuration isolated for this user
	if err := m.generatePHPFPMPool(site); err != nil {
		return fmt.Errorf("php pool error: %w", err)
	}

	// Step 4: Generate Web Server VirtualHost config
	if err := m.generateVHostConfig(site); err != nil {
		return fmt.Errorf("vhost config error: %w", err)
	}

	// Step 5: Start/Restart PHP-FPM pool & reload Web Server
	_ = m.restartPHP(ctx, site.PHPVersion)
	_ = m.serviceMgr.Reload(ctx, site.WebServer)

	// Save site metadata
	return m.saveSiteRecord(site)
}

// generatePHPFPMPool creates an isolated PHP-FPM pool config in /usr/local/etc/php-fpm.d/<user>.conf
func (m *SiteManager) generatePHPFPMPool(site *Site) error {
	sockPath := fmt.Sprintf("/var/run/php-fpm-%s.sock", site.SystemUser)
	confContent := fmt.Sprintf(`; Isolated PHP-FPM pool for %s
[%s]
user = %s
group = %s

listen = %s
listen.owner = %s
listen.group = www
listen.mode = 0660

pm = dynamic
pm.max_children = 5
pm.start_servers = 2
pm.min_spare_servers = 1
pm.max_spare_servers = 3

chdir = %s
php_admin_value[open_basedir] = /usr/home/%s:/tmp:/var/tmp
php_admin_value[session.save_path] = /tmp
php_admin_value[upload_tmp_dir] = /tmp
`, site.Domain, site.SystemUser, site.SystemUser, site.SystemUser, sockPath, site.SystemUser, site.DocumentRoot, site.SystemUser)

	confFile := filepath.Join(m.phpPoolDir, fmt.Sprintf("%s.conf", site.SystemUser))
	_ = os.MkdirAll(m.phpPoolDir, 0755)
	return os.WriteFile(confFile, []byte(confContent), 0644)
}

// generateVHostConfig generates Nginx vhost config in /usr/local/etc/nginx/conf.d/<domain>.conf
func (m *SiteManager) generateVHostConfig(site *Site) error {
	sockPath := fmt.Sprintf("unix:/var/run/php-fpm-%s.sock", site.SystemUser)
	var config string

	if site.IsReverseProxy {
		config = fmt.Sprintf(`server {
    listen 80;
    server_name %s www.%s;

    access_log /usr/home/%s/logs/access.log;
    error_log /usr/home/%s/logs/error.log;

    location / {
        proxy_pass %s;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
`, site.Domain, site.Domain, site.SystemUser, site.SystemUser, site.ProxyUpstream)
	} else {
		config = fmt.Sprintf(`server {
    listen 80;
    server_name %s www.%s;
    root %s;
    index index.php index.html index.htm;

    access_log /usr/home/%s/logs/access.log;
    error_log /usr/home/%s/logs/error.log;

    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }

    location ~ \.php$ {
        fastcgi_pass %s;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        include fastcgi_params;
    }

    location ~ /\.ht {
        deny all;
    }
}
`, site.Domain, site.Domain, site.DocumentRoot, site.SystemUser, site.SystemUser, sockPath)
	}

	_ = os.MkdirAll(m.vhostDir, 0755)
	vhostPath := filepath.Join(m.vhostDir, fmt.Sprintf("%s.conf", site.Domain))
	if err := os.WriteFile(vhostPath, []byte(config), 0644); err != nil {
		return err
	}

	// Also write to sites-available for backwards compatibility
	legacyDir := "/usr/local/etc/nginx/sites-available"
	_ = os.MkdirAll(legacyDir, 0755)
	_ = os.WriteFile(filepath.Join(legacyDir, fmt.Sprintf("%s.conf", site.Domain)), []byte(config), 0644)

	return nil
}

// RequestLetEncryptSSL issues automated SSL certificate via certbot or acme.sh
func (m *SiteManager) RequestLetEncryptSSL(ctx context.Context, domain string) error {
	cmd := "/usr/local/bin/certbot"
	args := []string{"certonly", "--webroot", "-w", filepath.Join("/usr/home", domain, "public_html"), "-d", domain, "--non-interactive", "--agree-tos", "-m", "admin@" + domain}
	if m.exec.UseDoas {
		args = append([]string{cmd}, args...)
		cmd = "/usr/local/bin/doas"
	}
	_, err := m.exec.Execute(ctx, cmd, args...)
	return err
}

func (m *SiteManager) saveSiteRecord(site *Site) error {
	dbDir := filepath.Join(m.dataDir, "sites")
	_ = os.MkdirAll(dbDir, 0755)
	filePath := filepath.Join(dbDir, fmt.Sprintf("%s.json", site.Domain))
	data, err := json.MarshalIndent(site, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

// ListSites returns all registered sites from data storage.
func (m *SiteManager) ListSites() ([]Site, error) {
	dbDir := filepath.Join(m.dataDir, "sites")
	files, err := os.ReadDir(dbDir)
	if os.IsNotExist(err) {
		return []Site{}, nil
	} else if err != nil {
		return nil, err
	}

	var result []Site
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".json") {
			data, err := os.ReadFile(filepath.Join(dbDir, f.Name()))
			if err == nil {
				var s Site
				if err := json.Unmarshal(data, &s); err == nil {
					result = append(result, s)
				}
			}
		}
	}
	return result, nil
}

// EnsureNginxConfigured verifies that /usr/local/etc/nginx/nginx.conf includes conf.d/*.conf.
func (m *SiteManager) EnsureNginxConfigured(ctx context.Context) error {
	_ = os.MkdirAll("/usr/local/etc/nginx/conf.d", 0755)
	_ = os.MkdirAll("/usr/local/etc/php-fpm.d", 0755)

	nginxConfPath := "/usr/local/etc/nginx/nginx.conf"
	content, err := os.ReadFile(nginxConfPath)
	if err != nil {
		return nil // Nginx config not yet present
	}

	confStr := string(content)
	if !strings.Contains(confStr, "conf.d/*.conf") {
		// Insert include directive inside http { ... }
		if lastBrace := strings.LastIndex(confStr, "}"); lastBrace != -1 {
			insertion := "\n    # Virtual hosts included by BSD Panel\n    include /usr/local/etc/nginx/conf.d/*.conf;\n"
			newConf := confStr[:lastBrace] + insertion + confStr[lastBrace:]
			_ = os.WriteFile(nginxConfPath, []byte(newConf), 0644)
		}
	}

	return nil
}

// EnsurePHPConfigured verifies that /usr/local/etc/php-fpm.conf exists and includes php-fpm.d.
func (m *SiteManager) EnsurePHPConfigured(ctx context.Context) error {
	_ = os.MkdirAll("/usr/local/etc/php-fpm.d", 0755)

	mainConf := "/usr/local/etc/php-fpm.conf"
	defaultConf := "/usr/local/etc/php-fpm.conf.default"

	// 1. If php-fpm.conf does not exist, copy from default or create clean minimal config
	if _, err := os.Stat(mainConf); os.IsNotExist(err) {
		if content, err := os.ReadFile(defaultConf); err == nil {
			_ = os.WriteFile(mainConf, content, 0644)
		} else {
			minimal := `[global]
pid = /var/run/php-fpm.pid
error_log = /var/log/php-fpm.log
log_level = notice
include = /usr/local/etc/php-fpm.d/*.conf
`
			_ = os.WriteFile(mainConf, []byte(minimal), 0644)
		}
	}

	// 2. Ensure include = /usr/local/etc/php-fpm.d/*.conf is uncommented
	if content, err := os.ReadFile(mainConf); err == nil {
		str := string(content)
		modified := false
		if strings.Contains(str, ";include=/usr/local/etc/php-fpm.d/*.conf") {
			str = strings.ReplaceAll(str, ";include=/usr/local/etc/php-fpm.d/*.conf", "include=/usr/local/etc/php-fpm.d/*.conf")
			modified = true
		} else if strings.Contains(str, ";include = /usr/local/etc/php-fpm.d/*.conf") {
			str = strings.ReplaceAll(str, ";include = /usr/local/etc/php-fpm.d/*.conf", "include = /usr/local/etc/php-fpm.d/*.conf")
			modified = true
		} else if !strings.Contains(str, "php-fpm.d/*.conf") {
			str += "\ninclude = /usr/local/etc/php-fpm.d/*.conf\n"
			modified = true
		}
		if modified {
			_ = os.WriteFile(mainConf, []byte(str), 0644)
		}
	}

	// 3. Ensure php_fpm is enabled in /etc/rc.conf
	_, _ = m.exec.Execute(ctx, "/usr/sbin/sysrc", "php_fpm_enable=YES")

	return nil
}

// restartPHP restarts FreeBSD php-fpm daemon using canonical and version-tagged service names.
func (m *SiteManager) restartPHP(ctx context.Context, version string) error {
	// Canonical FreeBSD rc service
	_, _ = m.exec.Execute(ctx, "/usr/sbin/service", "php-fpm", "onerestart")
	_, _ = m.exec.Execute(ctx, "/usr/sbin/service", "php-fpm", "onestart")

	// Versioned service (e.g. php83-fpm) if custom port
	if version != "" {
		tag := strings.ReplaceAll(version, ".", "")
		svcName := fmt.Sprintf("php%s-fpm", tag)
		_, _ = m.exec.Execute(ctx, "/usr/sbin/service", svcName, "onerestart")
		_, _ = m.exec.Execute(ctx, "/usr/sbin/service", svcName, "onestart")
	}

	return nil
}

// SyncAllVHosts regenerates vhost configs directly into conf.d, repairs permissions, and reloads web services.
func (m *SiteManager) SyncAllVHosts(ctx context.Context) error {
	_ = m.EnsureNginxConfigured(ctx)
	_ = m.EnsurePHPConfigured(ctx)

	sites, err := m.ListSites()
	if err != nil {
		return err
	}

	for _, s := range sites {
		siteCopy := s
		_ = m.generateVHostConfig(&siteCopy)
		_ = m.generatePHPFPMPool(&siteCopy)

		// Fix filesystem permissions so user 'www' can read public_html
		homeDir := filepath.Join("/usr/home", siteCopy.SystemUser)
		_, _ = m.exec.Execute(ctx, "/bin/chmod", "0755", homeDir)
		_, _ = m.exec.Execute(ctx, "/bin/chmod", "-R", "0755", filepath.Join(homeDir, "public_html"))
		_, _ = m.exec.Execute(ctx, "/usr/sbin/pw", "groupmod", siteCopy.SystemUser, "-m", "www")
	}

	// Restart PHP-FPM first to create sockets
	_ = m.restartPHP(ctx, "8.3")

	// Reload web server
	_ = m.serviceMgr.Reload(ctx, "nginx")

	return nil
}
