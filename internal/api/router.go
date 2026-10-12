package api

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/agus-salim/bsdpanel/internal/auth"
	"github.com/agus-salim/bsdpanel/internal/config"
	"github.com/agus-salim/bsdpanel/internal/database"
	"github.com/agus-salim/bsdpanel/internal/php"
	"github.com/agus-salim/bsdpanel/internal/system"
	"github.com/agus-salim/bsdpanel/internal/webserver"
	"github.com/agus-salim/bsdpanel/web"
)

type userContextKey string

const userKey userContextKey = "user"

type Server struct {
	cfg        *config.Config
	authMgr    *auth.AuthManager
	sysRunner  system.Runner
	svcMgr     *system.ServiceManager
	statsCol   *system.StatsCollector
	userMgr    *system.UserManager
	pkgMgr     *system.PkgManager
	vhostMgr   *webserver.Manager
	phpMgr     *php.Manager
	dbMgr      *database.Manager

	vhostsMu   sync.RWMutex
	vhostsList []webserver.VHostConfig

	templates  map[string]*template.Template
}

func NewServer(
	cfg *config.Config,
	authMgr *auth.AuthManager,
	runner system.Runner,
	vhostMgr *webserver.Manager,
	phpMgr *php.Manager,
	dbMgr *database.Manager,
) (*Server, error) {
	s := &Server{
		cfg:       cfg,
		authMgr:   authMgr,
		sysRunner: runner,
		svcMgr:    system.NewServiceManager(runner),
		statsCol:  system.NewStatsCollector(runner),
		userMgr:   system.NewUserManager(runner),
		pkgMgr:    system.NewPkgManager(runner),
		vhostMgr:  vhostMgr,
		phpMgr:    phpMgr,
		dbMgr:     dbMgr,
		templates: make(map[string]*template.Template),
		vhostsList: []webserver.VHostConfig{
			{
				Domain:       "default.bsdpanel.local",
				User:         "www",
				Webserver:    "nginx",
				PHPVersion:   "php83",
				DocumentRoot: "/usr/local/www/nginx",
				CreatedAt:    "2026-10-12",
			},
		},
	}

	s.loadVHosts()

	if err := s.initTemplates(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Server) loadVHosts() {
	stateFile := filepath.Join(s.cfg.DataDir, "vhosts.json")
	if data, err := os.ReadFile(stateFile); err == nil {
		var list []webserver.VHostConfig
		if err := json.Unmarshal(data, &list); err == nil && len(list) > 0 {
			s.vhostsList = list
		}
	}

	// Disk discovery: temukan virtual host yang sudah ada di disk
	existingMap := make(map[string]bool)
	for _, vh := range s.vhostsList {
		existingMap[vh.Domain] = true
	}

	entries, err := os.ReadDir(s.cfg.VHostDir)
	if err == nil {
		updated := false
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".conf") {
				continue
			}
			domain := strings.TrimSuffix(e.Name(), ".conf")
			if domain == "" || existingMap[domain] {
				continue
			}

			confBytes, err := os.ReadFile(filepath.Join(s.cfg.VHostDir, e.Name()))
			if err != nil {
				continue
			}

			vh := s.parseVHostFromConf(domain, string(confBytes))
			s.vhostsList = append(s.vhostsList, vh)
			existingMap[domain] = true
			updated = true
		}
		if updated {
			s.saveVHosts()
		}
	}
}

func (s *Server) parseVHostFromConf(domain, content string) webserver.VHostConfig {
	vh := webserver.VHostConfig{
		Domain:       domain,
		User:         "www",
		Webserver:    "nginx",
		PHPVersion:   "php83",
		DocumentRoot: "/home/" + domain + "/public_html",
		CreatedAt:    time.Now().Format("2006-01-02"),
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "root ") && strings.HasSuffix(trimmed, ";") {
			r := strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(trimmed, ";"), "root "))
			if r != "" {
				vh.DocumentRoot = r
			}
		}
		if strings.Contains(trimmed, "fastcgi_pass unix:/var/run/") {
			parts := strings.Split(trimmed, "/var/run/")
			if len(parts) > 1 {
				sockPart := strings.TrimSuffix(strings.TrimSpace(parts[1]), ";")
				sockPart = strings.TrimSuffix(sockPart, ".sock")
				subParts := strings.Split(sockPart, "-fpm-")
				if len(subParts) == 2 {
					vh.PHPVersion = subParts[0]
					vh.User = subParts[1]
				}
			}
		}
	}

	certPaths := []string{
		fmt.Sprintf("/usr/local/etc/letsencrypt/live/%s/fullchain.pem", domain),
		fmt.Sprintf("/usr/local/etc/letsencrypt/live/%s-0001/fullchain.pem", domain),
		fmt.Sprintf("/etc/letsencrypt/live/%s/fullchain.pem", domain),
	}
	vh.SSLEnabled = false
	for _, cp := range certPaths {
		if _, err := os.Stat(cp); err == nil {
			if webserver.IsRealLetsEncryptCert(s.sysRunner, cp) {
				vh.SSLEnabled = true
				vh.SSLCertPath = cp
				break
			}
		}
	}

	return vh
}

func (s *Server) saveVHosts() {
	_ = os.MkdirAll(s.cfg.DataDir, 0750)
	stateFile := filepath.Join(s.cfg.DataDir, "vhosts.json")
	data, err := json.MarshalIndent(s.vhostsList, "", "  ")
	if err == nil {
		_ = os.WriteFile(stateFile, data, 0644)
	}
}

func (s *Server) initTemplates() error {
	pages := []string{
		"dashboard.html",
		"vhosts.html",
		"databases.html",
		"services.html",
		"php.html",
		"users.html",
		"files.html",
		"file_edit.html",
	}

	for _, page := range pages {
		tmpl, err := template.ParseFS(web.Content, "templates/layout.html", "templates/"+page)
		if err != nil {
			return err
		}
		s.templates[page] = tmpl
	}

	// Template login mandiri tanpa layout
	loginTmpl, err := template.ParseFS(web.Content, "templates/login.html")
	if err != nil {
		return err
	}
	s.templates["login.html"] = loginTmpl

	return nil
}

// SetupRouter mendaftarkan rute-rute HTTP standar Go 1.22+.
func (s *Server) SetupRouter() http.Handler {
	mux := http.NewServeMux()

	// Static Assets dari embed.FS
	staticFS, err := fs.Sub(web.Content, "static")
	if err == nil {
		mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	}

	// Login & Logout
	mux.HandleFunc("GET /login", s.handleLoginPage)
	mux.HandleFunc("POST /login", s.handleLoginSubmit)
	mux.HandleFunc("GET /logout", s.handleLogout)

	// Web UI Pages
	mux.HandleFunc("GET /{$}", s.handleDashboard)
	mux.HandleFunc("GET /vhosts", s.handleVHostsPage)
	mux.HandleFunc("GET /databases", s.handleDatabasesPage)
	mux.HandleFunc("GET /files", s.handleFilesPage)
	mux.HandleFunc("GET /files/edit", s.handleFileEditPage)
	mux.HandleFunc("GET /services", s.handleServicesPage)
	mux.HandleFunc("GET /php", s.handlePHPPage)
	mux.HandleFunc("GET /users", s.handleUsersPage)

	// REST / Form Action APIs
	mux.HandleFunc("GET /api/stats", s.handleAPIStats)
	mux.HandleFunc("POST /api/vhosts/create", s.handleAPIVHostCreate)
	mux.HandleFunc("POST /api/vhosts/update", s.handleAPIVHostUpdate)
	mux.HandleFunc("POST /api/vhosts/delete", s.handleAPIVHostDelete)
	mux.HandleFunc("POST /api/vhosts/ssl", s.handleVHostSSL)
	mux.HandleFunc("GET /api/vhosts/logs", s.handleVHostLogs)
	mux.HandleFunc("POST /api/databases/save", s.handleAPIDatabaseSave)
	mux.HandleFunc("POST /api/databases/delete", s.handleAPIDatabaseDelete)
	mux.HandleFunc("GET /api/databases/export", s.handleAPIDatabaseExport)
	mux.HandleFunc("POST /api/databases/import", s.handleAPIDatabaseImport)
	mux.HandleFunc("POST /api/services/action", s.handleAPIServiceAction)
	mux.HandleFunc("POST /api/php/pool", s.handleAPIPHPPool)
	mux.HandleFunc("POST /api/users/update", s.handleUpdateUser)
	mux.HandleFunc("POST /api/files/create", s.handleFileCreate)
	mux.HandleFunc("POST /api/files/save", s.handleFileSave)
	mux.HandleFunc("POST /api/files/delete", s.handleFileDelete)
	mux.HandleFunc("POST /api/files/upload", s.handleFileUpload)
	mux.HandleFunc("POST /api/files/compress", s.handleFileCompress)
	mux.HandleFunc("POST /api/files/extract", s.handleFileExtract)

	return s.recoveryMiddleware(s.authMiddleware(s.loggingMiddleware(mux)))
}

func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[PANIC RECOVERED] %v\nStack: %s", rec, debug.Stack())
				if strings.Contains(r.Header.Get("Accept"), "application/json") || strings.Contains(r.URL.Path, "/api/") {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"success": false,
						"error":   fmt.Sprintf("Terjadi kesalahan internal server: %v", rec),
					})
				} else {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Server", "bsdpanel/1.0 (FreeBSD)")
		next.ServeHTTP(w, r)
	})
}

// authMiddleware memproteksi seluruh halaman agar wajib login kecuali rute publik.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Rute publik yang diizinkan tanpa autentikasi
		if path == "/login" || strings.HasPrefix(path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie("bsdpanel_session")
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		payload, err := s.authMgr.VerifyToken(cookie.Value)
		if err != nil {
			s.authMgr.ClearAuthCookie(w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Masukkan user ke request context
		ctx := context.WithValue(r.Context(), userKey, payload.Username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
