package api

import (
	"html/template"
	"io/fs"
	"net/http"
	"sync"

	"github.com/agus-salim/bsdpanel/internal/auth"
	"github.com/agus-salim/bsdpanel/internal/config"
	"github.com/agus-salim/bsdpanel/internal/database"
	"github.com/agus-salim/bsdpanel/internal/php"
	"github.com/agus-salim/bsdpanel/internal/system"
	"github.com/agus-salim/bsdpanel/internal/webserver"
	"github.com/agus-salim/bsdpanel/web"
)

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

	// Simpan data vhost in-memory / local state
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

	if err := s.initTemplates(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Server) initTemplates() error {
	pages := []string{"dashboard.html", "vhosts.html", "services.html", "php.html"}

	for _, page := range pages {
		tmpl, err := template.ParseFS(web.Content, "templates/layout.html", "templates/"+page)
		if err != nil {
			return err
		}
		s.templates[page] = tmpl
	}
	return nil
}

// SetupRouter mendaftarkan rute-rute HTTP standar Go 1.22+.
func (s *Server) SetupRouter() http.Handler {
	mux := http.NewServeMux()

	// Static Assets dari embed.FS
	staticFS, err := fs.Sub(web.Content, "static")
	if err == nil {
		mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	}

	// Web UI Pages
	mux.HandleFunc("GET /", s.handleDashboard)
	mux.HandleFunc("GET /vhosts", s.handleVHostsPage)
	mux.HandleFunc("GET /services", s.handleServicesPage)
	mux.HandleFunc("GET /php", s.handlePHPPage)
	mux.HandleFunc("GET /logout", s.handleLogout)

	// REST / Form Action APIs
	mux.HandleFunc("GET /api/stats", s.handleAPIStats)
	mux.HandleFunc("POST /api/vhosts/create", s.handleAPIVHostCreate)
	mux.HandleFunc("POST /api/vhosts/delete", s.handleAPIVHostDelete)
	mux.HandleFunc("POST /api/services/action", s.handleAPIServiceAction)
	mux.HandleFunc("POST /api/php/pool", s.handleAPIPHPPool)

	return s.loggingMiddleware(mux)
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Server", "bsdpanel/1.0 (FreeBSD)")
		next.ServeHTTP(w, r)
	})
}
