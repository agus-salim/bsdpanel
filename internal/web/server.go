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
		sessions:    make(map[string]time.Time),
	}

	return s, nil
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
	mux.HandleFunc("/firewall", s.handleFirewallPage)
	mux.HandleFunc("/terminal", s.handleTerminalPage)

	// REST API Endpoints (Protected)
	mux.HandleFunc("/api/stats", s.handleAPIStats)
	mux.HandleFunc("/api/sites", s.handleAPISites)
	mux.HandleFunc("/api/services", s.handleAPIServices)
	mux.HandleFunc("/api/services/action", s.handleAPIServiceAction)
	mux.HandleFunc("/api/databases", s.handleAPIDatabases)
	mux.HandleFunc("/api/firewall", s.handleAPIFirewall)
	mux.HandleFunc("/api/logs", s.handleAPILogs)

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
