package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/agus-salim/bsdpanel/internal/php"
	"github.com/agus-salim/bsdpanel/internal/system"
	"github.com/agus-salim/bsdpanel/internal/webserver"
)

type PageData struct {
	Title      string
	ActiveMenu string
	Stats      *system.SystemStats
	Services   []system.ServiceStatus
	VHosts     []webserver.VHostConfig
	FlashError string
	FlashSuccess string
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	stats, _ := s.statsCol.GetStats()
	services, _ := s.svcMgr.GetAllManagedServices()

	data := PageData{
		Title:      "Dashboard Metrik",
		ActiveMenu: "dashboard",
		Stats:      stats,
		Services:   services,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.templates["dashboard.html"].ExecuteTemplate(w, "layout", data)
}

func (s *Server) handleVHostsPage(w http.ResponseWriter, r *http.Request) {
	s.vhostsMu.RLock()
	vhosts := append([]webserver.VHostConfig{}, s.vhostsList...)
	s.vhostsMu.RUnlock()

	data := PageData{
		Title:      "Virtual Hosts",
		ActiveMenu: "vhosts",
		VHosts:     vhosts,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.templates["vhosts.html"].ExecuteTemplate(w, "layout", data)
}

func (s *Server) handleServicesPage(w http.ResponseWriter, r *http.Request) {
	services, _ := s.svcMgr.GetAllManagedServices()

	data := PageData{
		Title:      "FreeBSD Services",
		ActiveMenu: "services",
		Services:   services,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.templates["services.html"].ExecuteTemplate(w, "layout", data)
}

func (s *Server) handlePHPPage(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Title:      "PHP Configuration",
		ActiveMenu: "php",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.templates["php.html"].ExecuteTemplate(w, "layout", data)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.authMgr.ClearAuthCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// REST / Action Handlers

func (s *Server) handleAPIStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.statsCol.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleAPIVHostCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form invalid", http.StatusBadRequest)
		return
	}

	domain := strings.TrimSpace(r.FormValue("domain"))
	user := strings.TrimSpace(r.FormValue("user"))
	webserverType := strings.TrimSpace(r.FormValue("webserver"))
	phpVer := strings.TrimSpace(r.FormValue("php_version"))
	docRoot := strings.TrimSpace(r.FormValue("document_root"))

	if domain == "" || user == "" {
		http.Error(w, "Domain dan user wajib diisi", http.StatusBadRequest)
		return
	}

	if webserverType == "" {
		webserverType = "nginx"
	}

	// 1. Buat user sistem FreeBSD via pw jika belum ada
	_ = s.userMgr.CreateUser(user)

	// 2. Buat konfigurasi pool PHP-FPM jika PHP diaktifkan
	if phpVer != "" && phpVer != "none" {
		_ = s.phpMgr.CreateOrUpdatePool(php.PoolConfig{
			User:              user,
			PHPVersion:        phpVer,
			MemoryLimit:       "256M",
			UploadMaxFilesize: "64M",
			MaxExecutionTime:  60,
		})
	}

	// 3. Buat konfigurasi VHost Nginx
	vh := webserver.VHostConfig{
		Domain:       domain,
		User:         user,
		Webserver:    webserverType,
		PHPVersion:   phpVer,
		DocumentRoot: docRoot,
		CreatedAt:    time.Now().Format("2006-01-02"),
	}

	_ = s.vhostMgr.CreateOrUpdate(vh)

	// Simpan ke in-memory list
	s.vhostsMu.Lock()
	s.vhostsList = append(s.vhostsList, vh)
	s.vhostsMu.Unlock()

	http.Redirect(w, r, "/vhosts", http.StatusSeeOther)
}

func (s *Server) handleAPIVHostDelete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form invalid", http.StatusBadRequest)
		return
	}

	domain := strings.TrimSpace(r.FormValue("domain"))
	_ = s.vhostMgr.Delete(domain)

	s.vhostsMu.Lock()
	filtered := make([]webserver.VHostConfig, 0, len(s.vhostsList))
	for _, v := range s.vhostsList {
		if v.Domain != domain {
			filtered = append(filtered, v)
		}
	}
	s.vhostsList = filtered
	s.vhostsMu.Unlock()

	http.Redirect(w, r, "/vhosts", http.StatusSeeOther)
}

func (s *Server) handleAPIServiceAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form invalid", http.StatusBadRequest)
		return
	}

	svcName := r.FormValue("service")
	action := r.FormValue("action")

	_, _ = s.svcMgr.Action(svcName, action)
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func (s *Server) handleAPIPHPPool(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form invalid", http.StatusBadRequest)
		return
	}

	user := r.FormValue("user")
	phpVer := r.FormValue("php_version")
	memLimit := r.FormValue("memory_limit")
	uploadMax := r.FormValue("upload_max_filesize")
	maxExec, _ := strconv.Atoi(r.FormValue("max_execution_time"))

	_ = s.phpMgr.CreateOrUpdatePool(php.PoolConfig{
		User:              user,
		PHPVersion:        phpVer,
		MemoryLimit:       memLimit,
		UploadMaxFilesize: uploadMax,
		MaxExecutionTime:  maxExec,
	})

	http.Redirect(w, r, "/php", http.StatusSeeOther)
}
