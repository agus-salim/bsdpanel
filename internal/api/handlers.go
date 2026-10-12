package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/agus-salim/bsdpanel/internal/auth"
	"github.com/agus-salim/bsdpanel/internal/php"
	"github.com/agus-salim/bsdpanel/internal/system"
	"github.com/agus-salim/bsdpanel/internal/webserver"
)

type PageData struct {
	Title        string
	ActiveMenu   string
	Username     string
	Stats        *system.SystemStats
	Services     []system.ServiceStatus
	VHosts       []webserver.VHostConfig
	Files        []FileInfo
	CurrentPath  string
	ParentPath   string
	FilePath     string
	Dir          string
	Content      string
	FlashError   string
	FlashSuccess string
	Error        string
}

type FileInfo struct {
	Name    string
	IsDir   bool
	SizeStr string
	Mode    string
	ModTime string
}

func (s *Server) getUsername(r *http.Request) string {
	if u, ok := r.Context().Value(userKey).(string); ok && u != "" {
		return u
	}
	return s.cfg.AdminUser
}

// Login & Auth Handlers

func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	// Jika sudah ada cookie valid, langsung lempar ke dashboard
	if cookie, err := r.Cookie("bsdpanel_session"); err == nil && cookie.Value != "" {
		if _, err := s.authMgr.VerifyToken(cookie.Value); err == nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}

	data := PageData{Title: "Login"}
	if r.URL.Query().Get("error") == "invalid" {
		data.Error = "Username atau password salah!"
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.templates["login.html"].Execute(w, data)
}

func (s *Server) handleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/login?error=invalid", http.StatusSeeOther)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	if username != s.cfg.AdminUser || !s.cfg.VerifyPassword(password) {
		http.Redirect(w, r, "/login?error=invalid", http.StatusSeeOther)
		return
	}

	token, err := s.authMgr.GenerateToken(username, auth.RoleAdmin, 7*24*time.Hour)
	if err != nil {
		http.Redirect(w, r, "/login?error=invalid", http.StatusSeeOther)
		return
	}

	s.authMgr.SetAuthCookie(w, token)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.authMgr.ClearAuthCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// Dashboard

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	stats, _ := s.statsCol.GetStats()
	services, _ := s.svcMgr.GetAllManagedServices()

	data := PageData{
		Title:      "Dashboard Metrik",
		ActiveMenu: "dashboard",
		Username:   s.getUsername(r),
		Stats:      stats,
		Services:   services,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.templates["dashboard.html"].ExecuteTemplate(w, "layout", data)
}

// VHosts

func (s *Server) handleVHostsPage(w http.ResponseWriter, r *http.Request) {
	s.vhostsMu.RLock()
	vhosts := append([]webserver.VHostConfig{}, s.vhostsList...)
	s.vhostsMu.RUnlock()

	data := PageData{
		Title:        "Virtual Hosts",
		ActiveMenu:   "vhosts",
		Username:     s.getUsername(r),
		VHosts:       vhosts,
		FlashSuccess: r.URL.Query().Get("msg"),
		FlashError:   r.URL.Query().Get("err"),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.templates["vhosts.html"].ExecuteTemplate(w, "layout", data)
}

// Users

func (s *Server) handleUsersPage(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Title:        "Manajemen Pengguna",
		ActiveMenu:   "users",
		Username:     s.getUsername(r),
		FlashSuccess: r.URL.Query().Get("msg"),
		FlashError:   r.URL.Query().Get("err"),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.templates["users.html"].ExecuteTemplate(w, "layout", data)
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/users?err=Form+tidak+valid", http.StatusSeeOther)
		return
	}

	newUsername := strings.TrimSpace(r.FormValue("username"))
	newPassword := r.FormValue("new_password")
	confirmPass := r.FormValue("confirm_password")

	if newUsername == "" {
		http.Redirect(w, r, "/users?err=Username+tidak+boleh+kosong", http.StatusSeeOther)
		return
	}

	if newPassword != "" && newPassword != confirmPass {
		http.Redirect(w, r, "/users?err=Password+dan+konfirmasi+tidak+cocok", http.StatusSeeOther)
		return
	}

	if err := s.cfg.UpdateCredentials(newUsername, newPassword); err != nil {
		http.Redirect(w, r, "/users?err=Gagal+menyimpan+konfigurasi", http.StatusSeeOther)
		return
	}

	// Buat token baru jika username berubah
	if newUsername != s.getUsername(r) {
		token, _ := s.authMgr.GenerateToken(newUsername, auth.RoleAdmin, 7*24*time.Hour)
		s.authMgr.SetAuthCookie(w, token)
	}

	http.Redirect(w, r, "/users?msg=Username+dan+password+berhasil+diperbarui!", http.StatusSeeOther)
}

// File Manager Handlers

func (s *Server) handleFilesPage(w http.ResponseWriter, r *http.Request) {
	currentPath := r.URL.Query().Get("path")
	if currentPath == "" {
		currentPath = "/home"
	}
	currentPath = filepath.Clean(currentPath)

	entries, err := os.ReadDir(currentPath)
	if err != nil {
		// Fallback ke direktori induk atau /home jika gagal
		currentPath = "/home"
		entries, _ = os.ReadDir(currentPath)
	}

	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		sizeStr := formatFileSize(info.Size())
		files = append(files, FileInfo{
			Name:    entry.Name(),
			IsDir:   entry.IsDir(),
			SizeStr: sizeStr,
			Mode:    info.Mode().String(),
			ModTime: info.ModTime().Format("02 Jan 15:04"),
		})
	}

	// Urutkan direktori di atas, kemudian berkas
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return files[i].Name < files[j].Name
	})

	var parentPath string
	if currentPath != "/" && currentPath != "." {
		parentPath = filepath.Dir(currentPath)
	}

	data := PageData{
		Title:        "File Manager",
		ActiveMenu:   "files",
		Username:     s.getUsername(r),
		Files:        files,
		CurrentPath:  currentPath,
		ParentPath:   parentPath,
		FlashSuccess: r.URL.Query().Get("msg"),
		FlashError:   r.URL.Query().Get("err"),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.templates["files.html"].ExecuteTemplate(w, "layout", data)
}

func (s *Server) handleFileEditPage(w http.ResponseWriter, r *http.Request) {
	targetPath := filepath.Clean(r.URL.Query().Get("path"))
	content, err := os.ReadFile(targetPath)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/files?path=%s&err=Gagal+membuka+berkas", filepath.Dir(targetPath)), http.StatusSeeOther)
		return
	}

	data := PageData{
		Title:      "Edit Berkas",
		ActiveMenu: "files",
		Username:   s.getUsername(r),
		FilePath:   targetPath,
		Dir:        filepath.Dir(targetPath),
		Content:    string(content),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.templates["file_edit.html"].ExecuteTemplate(w, "layout", data)
}

func (s *Server) handleFileCreate(w http.ResponseWriter, r *http.Request) {
	dir := filepath.Clean(r.FormValue("dir"))
	name := strings.TrimSpace(r.FormValue("name"))
	fileType := r.FormValue("type")

	if name == "" {
		http.Redirect(w, r, fmt.Sprintf("/files?path=%s&err=Nama+tidak+boleh+kosong", dir), http.StatusSeeOther)
		return
	}

	target := filepath.Join(dir, name)
	if fileType == "dir" {
		_ = os.MkdirAll(target, 0755)
	} else {
		_ = os.WriteFile(target, []byte(""), 0644)
	}

	http.Redirect(w, r, fmt.Sprintf("/files?path=%s&msg=Berhasil+dibuat!", dir), http.StatusSeeOther)
}

func (s *Server) handleFileSave(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(r.FormValue("path"))
	content := r.FormValue("content")

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		http.Redirect(w, r, fmt.Sprintf("/files/edit?path=%s&err=Gagal+menyimpan", path), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/files?path=%s&msg=Berkas+berhasil+disimpan!", filepath.Dir(path)), http.StatusSeeOther)
}

func (s *Server) handleFileDelete(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(r.FormValue("path"))
	parent := filepath.Dir(path)
	_ = os.RemoveAll(path)
	http.Redirect(w, r, fmt.Sprintf("/files?path=%s&msg=Dihapus+berhasil", parent), http.StatusSeeOther)
}

func formatFileSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	} else {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
}

// Services & PHP Pages

func (s *Server) handleServicesPage(w http.ResponseWriter, r *http.Request) {
	services, _ := s.svcMgr.GetAllManagedServices()

	data := PageData{
		Title:      "FreeBSD Services",
		ActiveMenu: "services",
		Username:   s.getUsername(r),
		Services:   services,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.templates["services.html"].ExecuteTemplate(w, "layout", data)
}

func (s *Server) handlePHPPage(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Title:      "PHP Configuration",
		ActiveMenu: "php",
		Username:   s.getUsername(r),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.templates["php.html"].ExecuteTemplate(w, "layout", data)
}

// REST / Form Action APIs

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
		http.Redirect(w, r, "/vhosts?err=Form+tidak+valid", http.StatusSeeOther)
		return
	}

	domain := strings.TrimSpace(r.FormValue("domain"))
	user := strings.TrimSpace(r.FormValue("user"))
	webserverType := strings.TrimSpace(r.FormValue("webserver"))
	phpVer := strings.TrimSpace(r.FormValue("php_version"))
	docRoot := strings.TrimSpace(r.FormValue("document_root"))

	if domain == "" || user == "" {
		http.Redirect(w, r, "/vhosts?err=Domain+dan+user+wajib+diisi", http.StatusSeeOther)
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

	// 3. Buat konfigurasi VHost Nginx & otomatis buat direktori/starter file
	vh := webserver.VHostConfig{
		Domain:       domain,
		User:         user,
		Webserver:    webserverType,
		PHPVersion:   phpVer,
		DocumentRoot: docRoot,
		CreatedAt:    time.Now().Format("2006-01-02"),
	}

	if err := s.vhostMgr.CreateOrUpdate(&vh); err != nil {
		http.Redirect(w, r, fmt.Sprintf("/vhosts?err=%s", err.Error()), http.StatusSeeOther)
		return
	}

	// Simpan ke in-memory list
	s.vhostsMu.Lock()
	s.vhostsList = append(s.vhostsList, vh)
	s.vhostsMu.Unlock()

	http.Redirect(w, r, "/vhosts?msg=Virtual+host+dan+folder+berhasil+dibuat!", http.StatusSeeOther)
}

func (s *Server) handleVHostSSL(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/vhosts?err=Form+invalid", http.StatusSeeOther)
		return
	}

	domain := strings.TrimSpace(r.FormValue("domain"))
	s.vhostsMu.Lock()
	defer s.vhostsMu.Unlock()

	for i := range s.vhostsList {
		if s.vhostsList[i].Domain == domain {
			_ = s.vhostMgr.IssueSSL(domain, s.vhostsList[i].DocumentRoot)
			s.vhostsList[i].SSLEnabled = true
			_ = s.vhostMgr.CreateOrUpdate(&s.vhostsList[i])
			break
		}
	}

	http.Redirect(w, r, "/vhosts?msg=Sertifikat+SSL+Let%27s+Encrypt+berhasil+diterbitkan!", http.StatusSeeOther)
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

	http.Redirect(w, r, "/vhosts?msg=Domain+berhasil+dihapus", http.StatusSeeOther)
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
