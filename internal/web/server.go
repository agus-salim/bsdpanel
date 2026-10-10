package web

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"time"

	"bsdpanel/internal/config"
	"bsdpanel/internal/database"
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
	tmpl        *template.Template
	staticFS    http.FileSystem
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
	embeddedAssets embed.FS,
) (*Server, error) {
	// Parse templates from embedded assets or local disk
	tmpl, err := template.ParseFS(embeddedAssets, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

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
		tmpl:        tmpl,
		staticFS:    http.FS(staticSub),
	}

	return s, nil
}

// Router configures all HTTP routes.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Static assets (CSS, Fonts, SVG Icons)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(s.staticFS)))

	// Web UI Pages
	mux.HandleFunc("/", s.handleDashboardPage)
	mux.HandleFunc("/sites", s.handleSitesPage)
	mux.HandleFunc("/services", s.handleServicesPage)
	mux.HandleFunc("/databases", s.handleDatabasesPage)
	mux.HandleFunc("/firewall", s.handleFirewallPage)
	mux.HandleFunc("/terminal", s.handleTerminalPage)

	// REST API Endpoints
	mux.HandleFunc("/api/stats", s.handleAPIStats)
	mux.HandleFunc("/api/sites", s.handleAPISites)
	mux.HandleFunc("/api/services", s.handleAPIServices)
	mux.HandleFunc("/api/services/action", s.handleAPIServiceAction)
	mux.HandleFunc("/api/databases", s.handleAPIDatabases)
	mux.HandleFunc("/api/firewall", s.handleAPIFirewall)
	mux.HandleFunc("/api/logs", s.handleAPILogs)

	// Wrap with security & timing middleware
	return s.securityMiddleware(mux)
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
