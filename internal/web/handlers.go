package web

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"bsdpanel/internal/sites"
)

// SystemStats represents real-time FreeBSD machine statistics.
type SystemStats struct {
	OS          string  `json:"os"`
	Hostname    string  `json:"hostname"`
	Uptime      string  `json:"uptime"`
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryTotal uint64  `json:"memory_total"`
	MemoryUsed  uint64  `json:"memory_used"`
	MemoryPct   float64 `json:"memory_pct"`
	DiskTotal   string  `json:"disk_total"`
	DiskUsed    string  `json:"disk_used"`
	DiskPct     string  `json:"disk_pct"`
	ActiveSites int     `json:"active_sites"`
	PFActive    bool    `json:"pf_active"`
}

// handleLoginPage processes user login authentication.
func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		cookie, err := r.Cookie("bsdpanel_session")
		if err == nil && s.isValidSession(cookie.Value) {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		s.render(w, "login.html", map[string]interface{}{})
		return
	}

	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		if username == s.cfg.AdminUser && password == s.cfg.AdminPassword {
			token := s.createSession()
			http.SetCookie(w, &http.Cookie{
				Name:     "bsdpanel_session",
				Value:    token,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				MaxAge:   86400,
			})
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		s.render(w, "login.html", map[string]interface{}{
			"Error": "Username atau password salah!",
		})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleLogout ends the active user session.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("bsdpanel_session")
	if err == nil {
		s.deleteSession(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "bsdpanel_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/login", http.StatusFound)
}

// handleDashboardPage renders the main compact dashboard.
func (s *Server) handleDashboardPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	stats := s.getFreeBSDStats(r.Context())
	sitesList, _ := s.siteMgr.ListSites()

	data := map[string]interface{}{
		"Title":       "Overview Dashboard",
		"ActiveNav":   "dashboard",
		"Stats":       stats,
		"Sites":       sitesList,
		"PanelDomain": s.cfg.PanelDomain,
		"Timestamp":   time.Now().Format("02 Jan 2006, 15:04:05 MST"),
	}

	s.render(w, "dashboard.html", data)
}

// handleSitesPage renders site management view.
func (s *Server) handleSitesPage(w http.ResponseWriter, r *http.Request) {
	sitesList, _ := s.siteMgr.ListSites()
	data := map[string]interface{}{
		"Title":     "Websites & Domains",
		"ActiveNav": "sites",
		"Sites":     sitesList,
	}
	s.render(w, "sites.html", data)
}

// handleServicesPage renders services management view.
func (s *Server) handleServicesPage(w http.ResponseWriter, r *http.Request) {
	serviceNames := []string{"nginx", "apache24", "caddy", "mysql-server", "postgresql", "pf"}
	statuses := make(map[string]interface{})

	for _, name := range serviceNames {
		st, _ := s.serviceMgr.Status(r.Context(), name)
		statuses[name] = st
	}

	data := map[string]interface{}{
		"Title":     "Service & Package Manager",
		"ActiveNav": "services",
		"Services":  statuses,
	}
	s.render(w, "services.html", data)
}

// handleDatabasesPage renders database management view.
func (s *Server) handleDatabasesPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":     "Database Management",
		"ActiveNav": "databases",
	}
	s.render(w, "databases.html", data)
}

// handleFirewallPage renders PF and Fail2ban firewall control.
func (s *Server) handleFirewallPage(w http.ResponseWriter, r *http.Request) {
	pfActive := s.firewallMgr.IsPFActive(r.Context())
	f2bStatus, _ := s.firewallMgr.GetFail2banStatus(r.Context())

	data := map[string]interface{}{
		"Title":     "Firewall & Security",
		"ActiveNav": "firewall",
		"PFActive":  pfActive,
		"Fail2ban":  f2bStatus,
	}
	s.render(w, "firewall.html", data)
}

// handleTerminalPage renders the web terminal view.
func (s *Server) handleTerminalPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":     "Web Terminal",
		"ActiveNav": "terminal",
	}
	s.render(w, "terminal.html", data)
}

// handleAPIStats returns real-time JSON metrics for dashboard charts.
func (s *Server) handleAPIStats(w http.ResponseWriter, r *http.Request) {
	stats := s.getFreeBSDStats(r.Context())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

// handleAPISites provides CRUD operations for websites.
func (s *Server) handleAPISites(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		sitesList, err := s.siteMgr.ListSites()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(sitesList)
		return
	}

	if r.Method == http.MethodPost {
		var newSite sites.Site
		if err := json.NewDecoder(r.Body).Decode(&newSite); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if err := s.siteMgr.CreateSite(r.Context(), &newSite); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Site %s created under system user %s", newSite.Domain, newSite.SystemUser),
			"site":    newSite,
		})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleAPIServices returns status or triggers service actions.
func (s *Server) handleAPIServices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	svcs := []string{"nginx", "apache24", "caddy", "mysql-server", "postgresql", "pf"}
	res := make(map[string]interface{})
	for _, sv := range svcs {
		st, _ := s.serviceMgr.Status(r.Context(), sv)
		res[sv] = st
	}
	_ = json.NewEncoder(w).Encode(res)
}

// handleAPIServiceAction starts, stops, or restarts a service.
func (s *Server) handleAPIServiceAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Service string `json:"service"`
		Action  string `json:"action"` // start, stop, restart, reload, enable, disable
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var err error
	switch req.Action {
	case "start":
		err = s.serviceMgr.Start(r.Context(), req.Service)
	case "stop":
		err = s.serviceMgr.Stop(r.Context(), req.Service)
	case "restart":
		err = s.serviceMgr.Restart(r.Context(), req.Service)
	case "reload":
		err = s.serviceMgr.Reload(r.Context(), req.Service)
	case "enable":
		err = s.serviceMgr.Enable(r.Context(), req.Service)
	case "disable":
		err = s.serviceMgr.Disable(r.Context(), req.Service)
	default:
		http.Error(w, "unknown action", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// handleAPIServiceInstall installs packages via FreeBSD pkg.
func (s *Server) handleAPIServiceInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Package string `json:"package"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var pkgs []string
	switch strings.ToLower(req.Package) {
	case "nginx", "apache", "caddy", "litespeed":
		pkgs = s.pkgMgr.GetWebServerPkgs(req.Package)
	case "mariadb", "postgresql", "phpmyadmin", "phppgadmin":
		pkgs = s.pkgMgr.GetDatabasePkgs(req.Package)
	case "php82", "php83", "php84", "php85":
		version := strings.TrimPrefix(req.Package, "php")
		if len(version) == 2 {
			version = string(version[0]) + "." + string(version[1])
		}
		pkgs = s.pkgMgr.GetPHPPkgs(version)
	default:
		pkgs = []string{req.Package}
	}

	err := s.pkgMgr.Install(r.Context(), pkgs...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Successfully installed %s", req.Package),
	})
}

// handleAPIDatabases handles DB operations.
func (s *Server) handleAPIDatabases(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct {
			Type     string `json:"type"` // mariadb, postgresql
			Database string `json:"database"`
			User     string `json:"user"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		var err error
		if req.Type == "postgresql" {
			err = s.dbMgr.CreatePostgreSQLDatabase(r.Context(), req.Database, req.User, req.Password)
		} else {
			err = s.dbMgr.CreateMariaDBDatabase(r.Context(), req.Database, req.User, req.Password)
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleAPIFirewall returns or applies firewall configuration.
func (s *Server) handleAPIFirewall(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet {
		active := s.firewallMgr.IsPFActive(r.Context())
		f2b, _ := s.firewallMgr.GetFail2banStatus(r.Context())
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"pf_active": active,
			"fail2ban":  f2b,
		})
		return
	}
}

// handleAPITerminalExec executes shell command safely within an unprivileged context.
func (s *Server) handleAPITerminalExec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Command string `json:"command"`
		User    string `json:"user"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	parts := strings.Fields(req.Command)
	if len(parts) == 0 {
		http.Error(w, "empty command", http.StatusBadRequest)
		return
	}

	targetUser := req.User
	if targetUser == "" || targetUser == "root" {
		targetUser = "bsdpanel"
	}

	cmd := parts[0]
	args := parts[1:]

	res, err := s.exec.ExecuteAsUser(r.Context(), targetUser, cmd, args...)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"output":    res.Stderr + " " + err.Error(),
			"exit_code": res.ExitCode,
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"output":    res.Stdout,
		"exit_code": res.ExitCode,
	})
}

// handleAPILogs provides access to site logs safely isolated.
func (s *Server) handleAPILogs(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	logType := r.URL.Query().Get("type") // access, error

	if domain == "" || strings.ContainsAny(domain, "/\\..") {
		http.Error(w, "invalid domain", http.StatusBadRequest)
		return
	}

	fileName := "access.log"
	if logType == "error" {
		fileName = "error.log"
	}

	sitePath := fmt.Sprintf("/usr/home/%s/logs/%s", domain, fileName)
	data, err := os.ReadFile(sitePath)
	if err != nil {
		data = []byte(fmt.Sprintf("Log file empty or not found: %s", sitePath))
	}

	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write(data)
}

// getFreeBSDStats collects real hardware and runtime metrics on FreeBSD.
func (s *Server) getFreeBSDStats(ctx context.Context) SystemStats {
	hostname, _ := os.Hostname()
	sitesList, _ := s.siteMgr.ListSites()
	pfActive := s.firewallMgr.IsPFActive(ctx)

	stats := SystemStats{
		OS:          fmt.Sprintf("FreeBSD (%s)", runtime.GOARCH),
		Hostname:    hostname,
		Uptime:      "N/A",
		CPUUsage:    0.0,
		MemoryTotal: 1024,
		MemoryUsed:  256,
		MemoryPct:   25.0,
		DiskTotal:   "20G",
		DiskUsed:    "2G",
		DiskPct:     "10%",
		ActiveSites: len(sitesList),
		PFActive:    pfActive,
	}

	// 1. Read uptime
	if res, err := s.exec.Execute(ctx, "/usr/bin/uptime"); err == nil {
		parts := strings.Split(res.Stdout, "up ")
		if len(parts) > 1 {
			stats.Uptime = strings.Split(parts[1], ",")[0]
		}
	}

	// 2. Read real RAM via sysctl hw.physmem
	if res, err := s.exec.Execute(ctx, "/sbin/sysctl", "-n", "hw.physmem"); err == nil {
		if bytesTotal, err := strconv.ParseUint(strings.TrimSpace(res.Stdout), 10, 64); err == nil {
			stats.MemoryTotal = bytesTotal / 1024 / 1024
		}
	}

	// Read free pages via sysctl vm.stats.vm.v_free_count
	if res, err := s.exec.Execute(ctx, "/sbin/sysctl", "-n", "vm.stats.vm.v_free_count"); err == nil {
		if freePages, err := strconv.ParseUint(strings.TrimSpace(res.Stdout), 10, 64); err == nil {
			freeMB := (freePages * 4096) / 1024 / 1024
			if stats.MemoryTotal > freeMB {
				stats.MemoryUsed = stats.MemoryTotal - freeMB
				stats.MemoryPct = math.Round((float64(stats.MemoryUsed)/float64(stats.MemoryTotal)*100)*10) / 10
			}
		}
	}

	// 3. Read real Disk via df -h /
	if res, err := s.exec.Execute(ctx, "/bin/df", "-h", "/"); err == nil {
		lines := strings.Split(res.Stdout, "\n")
		if len(lines) >= 2 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 5 {
				stats.DiskTotal = fields[1]
				stats.DiskUsed = fields[2]
				stats.DiskPct = fields[4]
			}
		}
	}

	// 4. Read CPU Load Average via sysctl vm.loadavg
	if res, err := s.exec.Execute(ctx, "/sbin/sysctl", "-n", "vm.loadavg"); err == nil {
		fields := strings.Fields(strings.Trim(res.Stdout, "{} \n"))
		if len(fields) > 0 {
			if load1, err := strconv.ParseFloat(fields[0], 64); err == nil {
				stats.CPUUsage = math.Round(load1*100) / 10 // e.g. 1.6%
				if stats.CPUUsage > 100 {
					stats.CPUUsage = 100
				}
			}
		}
	}

	return stats
}
