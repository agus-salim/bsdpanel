package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
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

// handleAPISiteUpdate handles modifications to an existing site (web server, PHP version, proxy, SSL).
func (s *Server) handleAPISiteUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var updated sites.Site
	if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if updated.Domain == "" {
		http.Error(w, "domain is required", http.StatusBadRequest)
		return
	}

	if err := s.siteMgr.UpdateSite(r.Context(), &updated); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Site %s updated successfully", updated.Domain),
	})
}

// handleAPISiteDelete removes a site and its configurations.
func (s *Server) handleAPISiteDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Domain      string `json:"domain"`
		RemoveFiles bool   `json:"remove_files"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Domain == "" {
		http.Error(w, "domain is required", http.StatusBadRequest)
		return
	}

	if err := s.siteMgr.DeleteSite(r.Context(), req.Domain, req.RemoveFiles); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Site %s deleted successfully", req.Domain),
	})
}

// handleAPISiteSSL triggers Let's Encrypt automated SSL issuance and vhost configuration.
func (s *Server) handleAPISiteSSL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Domain == "" {
		http.Error(w, "domain is required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := s.siteMgr.RequestLetEncryptSSL(r.Context(), req.Domain); err != nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Sertifikat SSL Let's Encrypt untuk %s berhasil diterbitkan dan diaktifkan!", req.Domain),
	})
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
	if targetUser == "" {
		targetUser = "root"
	}

	cmd := parts[0]
	args := parts[1:]

	res, err := s.exec.ExecuteAsUser(r.Context(), targetUser, cmd, args...)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		out := res.Stderr
		if out == "" {
			out = res.Stdout
		}
		if out == "" {
			out = err.Error()
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"output":    out,
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

	if domain == "" || strings.ContainsAny(domain, "/\\") {
		http.Error(w, "invalid domain", http.StatusBadRequest)
		return
	}

	fileName := "access.log"
	if logType == "error" {
		fileName = "error.log"
	}

	site, err := s.siteMgr.GetSite(domain)
	var logPath string
	if err == nil && site.SystemUser != "" {
		logPath = filepath.Join("/usr/home", site.SystemUser, "logs", fileName)
	} else {
		logPath = filepath.Join("/var/log/nginx", fileName)
	}

	res, err := s.exec.Execute(r.Context(), "/usr/bin/tail", "-n", "200", logPath)
	var content string
	if err == nil && strings.TrimSpace(res.Stdout) != "" {
		content = res.Stdout
	} else {
		// If user site log is empty, check global nginx log as fallback
		globalPath := filepath.Join("/var/log/nginx", fileName)
		if globalRes, gErr := s.exec.Execute(r.Context(), "/usr/bin/tail", "-n", "80", globalPath); gErr == nil && strings.TrimSpace(globalRes.Stdout) != "" {
			content = fmt.Sprintf("--- Log Khusus Situs Masih Bersih (Belum ada request baru) ---\nFile: %s\n\n--- Catatan Global Nginx (%s) ---\n%s", logPath, globalPath, globalRes.Stdout)
		} else {
			content = fmt.Sprintf("--- Log %s untuk %s ---\nFile: %s\n(File log kosong atau belum ada catatan aktivitas)", strings.ToUpper(logType), domain, logPath)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"domain":   domain,
		"log_type": logType,
		"path":     logPath,
		"content":  content,
	})
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

	// Read real Used RAM on FreeBSD matching htop (Active + Wire pages)
	pageSize := uint64(4096)
	if res, err := s.exec.Execute(ctx, "/sbin/sysctl", "-n", "hw.pagesize"); err == nil {
		if ps, err := strconv.ParseUint(strings.TrimSpace(res.Stdout), 10, 64); err == nil && ps > 0 {
			pageSize = ps
		}
	}

	var activePages, wirePages uint64
	if res, err := s.exec.Execute(ctx, "/sbin/sysctl", "-n", "vm.stats.vm.v_active_count"); err == nil {
		activePages, _ = strconv.ParseUint(strings.TrimSpace(res.Stdout), 10, 64)
	}
	if res, err := s.exec.Execute(ctx, "/sbin/sysctl", "-n", "vm.stats.vm.v_wire_count"); err == nil {
		wirePages, _ = strconv.ParseUint(strings.TrimSpace(res.Stdout), 10, 64)
	}

	if activePages > 0 || wirePages > 0 {
		usedBytes := (activePages + wirePages) * pageSize
		stats.MemoryUsed = usedBytes / 1024 / 1024
		if stats.MemoryTotal > 0 {
			stats.MemoryPct = math.Round((float64(stats.MemoryUsed)/float64(stats.MemoryTotal)*100)*10) / 10
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

// FileItem represents a directory entry in the File Manager.
type FileItem struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	IsDir     bool   `json:"is_dir"`
	Size      int64  `json:"size"`
	SizeStr   string `json:"size_str"`
	Mode      string `json:"mode"`
	ModTime   string `json:"mod_time"`
	Ext       string `json:"ext"`
	IsArchive bool   `json:"is_archive"`
}

func isArchiveFilename(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".zip") ||
		strings.HasSuffix(lower, ".tar.gz") ||
		strings.HasSuffix(lower, ".tgz") ||
		strings.HasSuffix(lower, ".tar.bz2") ||
		strings.HasSuffix(lower, ".tbz2") ||
		strings.HasSuffix(lower, ".tar.xz") ||
		strings.HasSuffix(lower, ".txz") ||
		strings.HasSuffix(lower, ".tar") ||
		strings.HasSuffix(lower, ".rar") ||
		strings.HasSuffix(lower, ".7z")
}

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// handleFilesPage renders the Web File Manager interface.
func (s *Server) handleFilesPage(w http.ResponseWriter, r *http.Request) {
	sitesList, _ := s.siteMgr.ListSites()
	currentPath := r.URL.Query().Get("path")
	if currentPath == "" {
		if len(sitesList) > 0 {
			currentPath = sitesList[0].DocumentRoot
		} else {
			currentPath = "/usr/home"
		}
	}
	currentPath = filepath.Clean(currentPath)

	data := map[string]interface{}{
		"Title":       "File Manager",
		"ActiveNav":   "files",
		"CurrentPath": currentPath,
		"Sites":       sitesList,
	}
	s.render(w, "files.html", data)
}

// handleAPIFilesList returns directory items.
func (s *Server) handleAPIFilesList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	targetPath := r.URL.Query().Get("path")
	if targetPath == "" {
		targetPath = "/usr/home"
	}
	targetPath = filepath.Clean(targetPath)

	entries, err := os.ReadDir(targetPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read directory: %v", err), http.StatusInternalServerError)
		return
	}

	var items []FileItem
	for _, e := range entries {
		info, err := e.Info()
		var size int64 = 0
		var mode string = "rw-r--r--"
		var modTime string = ""
		if err == nil {
			size = info.Size()
			mode = info.Mode().String()
			modTime = info.ModTime().Format("02 Jan 2006 15:04")
		}

		itemPath := filepath.Join(targetPath, e.Name())
		ext := strings.ToLower(filepath.Ext(e.Name()))
		items = append(items, FileItem{
			Name:      e.Name(),
			Path:      itemPath,
			IsDir:     e.IsDir(),
			Size:      size,
			SizeStr:   formatFileSize(size),
			Mode:      mode,
			ModTime:   modTime,
			Ext:       ext,
			IsArchive: isArchiveFilename(e.Name()),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})

	parentPath := filepath.Dir(targetPath)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"current_path": targetPath,
		"parent_path":  parentPath,
		"items":        items,
	})
}

// handleAPIFilesRead reads text content of a file.
func (s *Server) handleAPIFilesRead(w http.ResponseWriter, r *http.Request) {
	targetPath := filepath.Clean(r.URL.Query().Get("path"))
	if targetPath == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"path":    targetPath,
		"content": string(data),
		"size":    len(data),
	})
}

// handleAPIFilesSave saves text content of a file.
func (s *Server) handleAPIFilesSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	targetPath := filepath.Clean(req.Path)
	if err := os.WriteFile(targetPath, []byte(req.Content), 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// handleAPIFilesCreate creates a new file or directory.
func (s *Server) handleAPIFilesCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path  string `json:"path"`
		Name  string `json:"name"`
		IsDir bool   `json:"is_dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	newPath := filepath.Join(filepath.Clean(req.Path), req.Name)
	var err error
	if req.IsDir {
		err = os.MkdirAll(newPath, 0755)
	} else {
		err = os.WriteFile(newPath, []byte(""), 0644)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// handleAPIFilesDelete deletes a file or directory (or multiple items).
func (s *Server) handleAPIFilesDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path  string   `json:"path"`
		Paths []string `json:"paths"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	targets := req.Paths
	if len(targets) == 0 && req.Path != "" {
		targets = []string{req.Path}
	}

	if len(targets) == 0 {
		http.Error(w, "no target specified", http.StatusBadRequest)
		return
	}

	for _, p := range targets {
		targetPath := filepath.Clean(p)
		if targetPath == "/" || targetPath == "/root" || targetPath == "/usr" || targetPath == "/etc" || targetPath == "/var" {
			http.Error(w, fmt.Sprintf("cannot delete core system directory '%s'", targetPath), http.StatusBadRequest)
			return
		}
		if err := os.RemoveAll(targetPath); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// handleAPIFilesUpload uploads a file to the specified directory.
func (s *Server) handleAPIFilesUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_ = r.ParseMultipartForm(50 << 20) // 50MB
	targetDir := filepath.Clean(r.FormValue("path"))
	if targetDir == "" {
		targetDir = "/usr/home"
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "failed to get uploaded file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	destPath := filepath.Join(targetDir, header.Filename)
	out, err := os.Create(destPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// handleAPIFilesDownload serves a file download.
func (s *Server) handleAPIFilesDownload(w http.ResponseWriter, r *http.Request) {
	targetPath := filepath.Clean(r.URL.Query().Get("path"))
	if targetPath == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}

	fileName := filepath.Base(targetPath)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	http.ServeFile(w, r, targetPath)
}

// handleAPIFilesRename renames a file or directory.
func (s *Server) handleAPIFilesRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path    string `json:"path"`
		NewName string `json:"new_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	oldPath := filepath.Clean(req.Path)
	newName := strings.TrimSpace(req.NewName)
	if newName == "" || strings.ContainsAny(newName, `/\`) {
		http.Error(w, "invalid new name", http.StatusBadRequest)
		return
	}

	if oldPath == "/" || oldPath == "/root" || oldPath == "/usr" || oldPath == "/etc" || oldPath == "/var" {
		http.Error(w, "cannot rename core system directories", http.StatusBadRequest)
		return
	}

	newPath := filepath.Join(filepath.Dir(oldPath), newName)
	if err := os.Rename(oldPath, newPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// handleAPIFilesArchive packages selected items into an archive.
func (s *Server) handleAPIFilesArchive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		BasePath    string   `json:"base_path"`
		Items       []string `json:"items"`
		ArchiveName string   `json:"archive_name"`
		Format      string   `json:"format"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if len(req.Items) == 0 {
		http.Error(w, "no items selected for archiving", http.StatusBadRequest)
		return
	}

	basePath := filepath.Clean(req.BasePath)
	if basePath == "" {
		http.Error(w, "base path required", http.StatusBadRequest)
		return
	}

	if s.archiveMgr == nil {
		http.Error(w, "archive manager not initialized", http.StatusInternalServerError)
		return
	}

	archivePath, err := s.archiveMgr.CreateArchive(r.Context(), basePath, req.Items, req.ArchiveName, req.Format)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"archive_path": archivePath,
		"filename":     filepath.Base(archivePath),
	})
}

// handleAPIFilesExtract extracts an archive file into a target directory.
func (s *Server) handleAPIFilesExtract(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ArchivePath string `json:"archive_path"`
		DestPath    string `json:"dest_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	archivePath := filepath.Clean(req.ArchivePath)
	if archivePath == "" {
		http.Error(w, "archive path required", http.StatusBadRequest)
		return
	}

	destPath := filepath.Clean(req.DestPath)
	if destPath == "" || destPath == "." {
		destPath = filepath.Dir(archivePath)
	}

	if s.archiveMgr == nil {
		http.Error(w, "archive manager not initialized", http.StatusInternalServerError)
		return
	}

	if err := s.archiveMgr.ExtractArchive(r.Context(), archivePath, destPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"dest_path": destPath,
	})
}
