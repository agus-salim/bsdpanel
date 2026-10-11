package web

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"

	"bsdpanel/internal/config"
	"bsdpanel/internal/database"
	"bsdpanel/internal/php"
	"bsdpanel/internal/sites"
	"bsdpanel/internal/system"
)

// Server holds the dependencies for the web server and UI.
type Server struct {
	cfg         *config.Config
	exec        *system.Executor
	userMgr     *system.UserManager
	serviceMgr  *system.ServiceManager
	pkgMgr      *system.PkgManager
	firewallMgr *system.FirewallManager
	siteMgr     *sites.SiteManager
	dbMgr       *database.DatabaseManager
	phpMgr      *php.PHPManager
	archiveMgr  *system.ArchiveManager
	pages       map[string]*template.Template
	staticFS    http.FileSystem
	sessions    map[string]time.Time
	sessMu      sync.RWMutex
}

// NewServer constructs the Web Server instance.
func NewServer(
	cfg *config.Config,
	exec *system.Executor,
	userMgr *system.UserManager,
	serviceMgr *system.ServiceManager,
	pkgMgr *system.PkgManager,
	firewallMgr *system.FirewallManager,
	siteMgr *sites.SiteManager,
	dbMgr *database.DatabaseManager,
	phpMgr *php.PHPManager,
	archiveMgr *system.ArchiveManager,
	embeddedAssets embed.FS,
) (*Server, error) {
	// Build isolated template sets per page to avoid Go template namespace collisions
	pages := make(map[string]*template.Template)
	pageFiles := []string{
		"dashboard.html",
		"sites.html",
		"services.html",
		"databases.html",
		"php.html",
		"firewall.html",
		"terminal.html",
		"files.html",
	}

	for _, p := range pageFiles {
		t, err := template.ParseFS(embeddedAssets, "templates/layout.html", "templates/"+p)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %s: %w", p, err)
		}
		pages[p] = t
	}

	// Standalone login page without master layout
	tLogin, err := template.ParseFS(embeddedAssets, "templates/login.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse login template: %w", err)
	}
	pages["login.html"] = tLogin

	staticSub, err := fs.Sub(embeddedAssets, "static")
	if err != nil {
		return nil, fmt.Errorf("failed to load static sub-filesystem: %w", err)
	}

	s := &Server{
		cfg:         cfg,
		exec:        exec,
		userMgr:     userMgr,
		serviceMgr:  serviceMgr,
		pkgMgr:      pkgMgr,
		firewallMgr: firewallMgr,
		siteMgr:     siteMgr,
		dbMgr:       dbMgr,
		phpMgr:      phpMgr,
		archiveMgr:  archiveMgr,
		pages:       pages,
		staticFS:    http.FS(staticSub),
		sessions:    make(map[string]time.Time),
	}

	return s, nil
}

// render executes the appropriate template tree for a given page.
func (s *Server) render(w http.ResponseWriter, page string, data interface{}) {
	tmpl, ok := s.pages[page]
	if !ok {
		http.Error(w, "template not found: "+page, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var err error
	if page == "login.html" {
		err = tmpl.Execute(w, data)
	} else {
		err = tmpl.ExecuteTemplate(w, "layout", data)
	}

	if err != nil {
		http.Error(w, "Render error: "+err.Error(), http.StatusInternalServerError)
	}
}

// Router configures all HTTP routes.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Public static assets
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(s.staticFS)))

	// Authentication routes
	mux.HandleFunc("/login", s.handleLoginPage)
	mux.HandleFunc("/logout", s.handleLogout)

	// Web UI Pages (Protected)
	mux.HandleFunc("/", s.handleDashboardPage)
	mux.HandleFunc("/sites", s.handleSitesPage)
	mux.HandleFunc("/services", s.handleServicesPage)
	mux.HandleFunc("/databases", s.handleDatabasesPage)
	mux.HandleFunc("/php", s.handlePHPManagementPage)
	mux.HandleFunc("/firewall", s.handleFirewallPage)
	mux.HandleFunc("/terminal", s.handleTerminalPage)
	mux.HandleFunc("/files", s.handleFilesPage)

	// PHP Management APIs
	mux.HandleFunc("/api/php/versions", s.handleAPIPHPVersions)
	mux.HandleFunc("/api/php/extensions", s.handleAPIPHPExtensions)
	mux.HandleFunc("/api/php/extension/install", s.handleAPIPHPExtensionInstall)
	mux.HandleFunc("/api/php/extension/uninstall", s.handleAPIPHPExtensionUninstall)
	mux.HandleFunc("/api/php/restart", s.handleAPIPHPRestart)

	// REST API Endpoints (Protected)
	mux.HandleFunc("/api/stats", s.handleAPIStats)
	mux.HandleFunc("/api/sites", s.handleAPISites)
	mux.HandleFunc("/api/sites/update", s.handleAPISiteUpdate)
	mux.HandleFunc("/api/sites/delete", s.handleAPISiteDelete)
	mux.HandleFunc("/api/sites/ssl", s.handleAPISiteSSL)
	mux.HandleFunc("/api/services", s.handleAPIServices)
	mux.HandleFunc("/api/services/action", s.handleAPIServiceAction)
	mux.HandleFunc("/api/services/install", s.handleAPIServiceInstall)
	mux.HandleFunc("/api/services/uninstall", s.handleAPIServiceUninstall)
	// Database Extended APIs
	mux.HandleFunc("/api/databases", s.handleAPIDatabases)
	mux.HandleFunc("/api/databases/delete", s.handleAPIDatabaseDelete)
	mux.HandleFunc("/api/databases/update-password", s.handleAPIDatabaseUpdatePassword)
	mux.HandleFunc("/api/databases/export", s.handleAPIDatabaseExport)
	mux.HandleFunc("/api/databases/import", s.handleAPIDatabaseImport)

	// Admin Tools (phpMyAdmin & phpPgAdmin)
	mux.HandleFunc("/api/tools/status", s.handleAPIToolsStatus)
	mux.HandleFunc("/api/tools/install", s.handleAPIToolsInstall)
	mux.HandleFunc("/phpmyadmin", s.handlePHPMyAdminProxy)
	mux.HandleFunc("/phpmyadmin/", s.handlePHPMyAdminProxy)
	mux.HandleFunc("/phppgadmin", s.handlePHPPgAdminProxy)
	mux.HandleFunc("/phppgadmin/", s.handlePHPPgAdminProxy)
	mux.HandleFunc("/api/firewall", s.handleAPIFirewall)
	mux.HandleFunc("/api/terminal/exec", s.handleAPITerminalExec)
	mux.HandleFunc("/api/logs", s.handleAPILogs)

	// File Manager APIs
	mux.HandleFunc("/api/files/list", s.handleAPIFilesList)
	mux.HandleFunc("/api/files/read", s.handleAPIFilesRead)
	mux.HandleFunc("/api/files/save", s.handleAPIFilesSave)
	mux.HandleFunc("/api/files/create", s.handleAPIFilesCreate)
	mux.HandleFunc("/api/files/delete", s.handleAPIFilesDelete)
	mux.HandleFunc("/api/files/upload", s.handleAPIFilesUpload)
	mux.HandleFunc("/api/files/download", s.handleAPIFilesDownload)
	mux.HandleFunc("/api/files/rename", s.handleAPIFilesRename)
	mux.HandleFunc("/api/files/archive", s.handleAPIFilesArchive)
	mux.HandleFunc("/api/files/extract", s.handleAPIFilesExtract)

	// Wrap with security & authentication middleware
	return s.securityMiddleware(s.authMiddleware(mux))
}

// authMiddleware protects private routes from unauthenticated access.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/login" || strings.HasPrefix(path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie("bsdpanel_session")
		if err != nil || !s.isValidSession(cookie.Value) {
			if strings.HasPrefix(path, "/api/") {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Session helper methods
func (s *Server) createSession() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	token := hex.EncodeToString(b)

	s.sessMu.Lock()
	defer s.sessMu.Unlock()
	s.sessions[token] = time.Now().Add(24 * time.Hour)
	return token
}

func (s *Server) isValidSession(token string) bool {
	if token == "" {
		return false
	}
	s.sessMu.RLock()
	defer s.sessMu.RUnlock()
	exp, exists := s.sessions[token]
	if !exists {
		return false
	}
	return time.Now().Before(exp)
}

func (s *Server) deleteSession(token string) {
	if token == "" {
		return
	}
	s.sessMu.Lock()
	defer s.sessMu.Unlock()
	delete(s.sessions, token)
}

// Start begins listening on the configured address.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return srv.ListenAndServe()
}

// securityMiddleware enforces reverse-proxy header checks and secure HTTP attributes.
func (s *Server) securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}
