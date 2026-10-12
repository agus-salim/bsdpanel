package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/agus-salim/bsdpanel/internal/auth"
	"github.com/agus-salim/bsdpanel/internal/database"
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
	Databases    []database.DatabaseItem
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
		Title:        "FreeBSD Services",
		ActiveMenu:   "services",
		Username:     s.getUsername(r),
		Services:     services,
		FlashSuccess: r.URL.Query().Get("msg"),
		FlashError:   r.URL.Query().Get("err"),
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

	// Simpan ke in-memory list dan persistensi disk
	s.vhostsMu.Lock()
	s.vhostsList = append(s.vhostsList, vh)
	s.saveVHosts()
	s.vhostsMu.Unlock()

	http.Redirect(w, r, "/vhosts?msg=Virtual+host+dan+folder+berhasil+dibuat!", http.StatusSeeOther)
}

func (s *Server) handleVHostSSL(w http.ResponseWriter, r *http.Request) {
	isJSON := strings.Contains(r.Header.Get("Accept"), "application/json") || r.Header.Get("X-Requested-With") == "XMLHttpRequest"

	_ = r.ParseForm()
	_ = r.ParseMultipartForm(32 << 20)

	domain := strings.TrimSpace(r.FormValue("domain"))
	if domain == "" {
		domain = strings.TrimSpace(r.URL.Query().Get("domain"))
	}
	if domain == "" && r.Body != nil {
		var payload struct {
			Domain string `json:"domain"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		domain = strings.TrimSpace(payload.Domain)
	}

	if domain == "" {
		s.respondSSL(w, r, isJSON, false, "Nama domain tidak terdeteksi dalam permintaan")
		return
	}

	s.vhostsMu.Lock()
	defer s.vhostsMu.Unlock()

	var target *webserver.VHostConfig
	for i := range s.vhostsList {
		if s.vhostsList[i].Domain == domain {
			target = &s.vhostsList[i]
			break
		}
	}

	// Jika domain tidak ada di state in-memory, periksa apakah file konfigurasinya ada di disk
	if target == nil {
		confPath := filepath.Join(s.cfg.VHostDir, domain+".conf")
		if confBytes, err := os.ReadFile(confPath); err == nil {
			newVh := s.parseVHostFromConf(domain, string(confBytes))
			s.vhostsList = append(s.vhostsList, newVh)
			target = &s.vhostsList[len(s.vhostsList)-1]
			s.saveVHosts()
		}
	}

	if target == nil {
		s.respondSSL(w, r, isJSON, false, fmt.Sprintf("Domain '%s' tidak ditemukan dalam daftar virtual host", domain))
		return
	}

	// Terbitkan sertifikat Let's Encrypt
	err := s.vhostMgr.IssueSSL(target)
	if err != nil {
		target.SSLEnabled = false
		_ = s.vhostMgr.CreateOrUpdate(target)
		s.saveVHosts()
		s.respondSSL(w, r, isJSON, false, err.Error())
		return
	}

	s.saveVHosts()
	s.respondSSL(w, r, isJSON, true, fmt.Sprintf("Sertifikat SSL Let's Encrypt untuk %s berhasil diterbitkan dan aktif!", domain))
}

func (s *Server) respondSSL(w http.ResponseWriter, r *http.Request, isJSON, success bool, msg string) {
	if isJSON {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": success,
			"message": msg,
			"error":   msg,
		})
		return
	}

	if success {
		http.Redirect(w, r, fmt.Sprintf("/vhosts?msg=%s", url.QueryEscape(msg)), http.StatusSeeOther)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/vhosts?err=%s", url.QueryEscape(msg)), http.StatusSeeOther)
	}
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
	s.saveVHosts()
	s.vhostsMu.Unlock()

	http.Redirect(w, r, "/vhosts?msg=Domain+berhasil+dihapus", http.StatusSeeOther)
}

func (s *Server) handleAPIServiceAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/services?err=Form+tidak+valid", http.StatusSeeOther)
		return
	}

	svcName := r.FormValue("service")
	action := r.FormValue("action")

	out, err := s.svcMgr.Action(svcName, action)
	if err != nil {
		errMsg := fmt.Sprintf("Gagal %s layanan %s: %s", action, svcName, err.Error())
		http.Redirect(w, r, fmt.Sprintf("/services?err=%s", url.QueryEscape(errMsg)), http.StatusSeeOther)
		return
	}

	msg := fmt.Sprintf("Layanan %s berhasil di-%s", svcName, action)
	if out != "" && strings.Contains(out, "Autostart") {
		msg = fmt.Sprintf("%s untuk %s", out, svcName)
	}
	http.Redirect(w, r, fmt.Sprintf("/services?msg=%s", url.QueryEscape(msg)), http.StatusSeeOther)
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

// Database Management Handlers

func (s *Server) handleDatabasesPage(w http.ResponseWriter, r *http.Request) {
	dbs := s.dbMgr.GetAll()

	s.vhostsMu.RLock()
	vhosts := make([]webserver.VHostConfig, len(s.vhostsList))
	copy(vhosts, s.vhostsList)
	s.vhostsMu.RUnlock()

	data := PageData{
		Title:        "Manajemen Database",
		ActiveMenu:   "databases",
		Username:     s.getUsername(r),
		Databases:    dbs,
		VHosts:       vhosts,
		FlashSuccess: r.URL.Query().Get("msg"),
		FlashError:   r.URL.Query().Get("err"),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.templates["databases.html"].ExecuteTemplate(w, "layout", data)
}

func (s *Server) handleAPIDatabaseSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/databases?err=Form+tidak+valid", http.StatusSeeOther)
		return
	}

	id := strings.TrimSpace(r.FormValue("id"))
	domain := strings.TrimSpace(r.FormValue("domain"))
	engine := strings.TrimSpace(r.FormValue("engine"))
	dbName := strings.TrimSpace(r.FormValue("db_name"))
	dbUser := strings.TrimSpace(r.FormValue("db_user"))
	dbPass := strings.TrimSpace(r.FormValue("db_pass"))
	createInDBMS := r.FormValue("create_dbms") == "1" || r.FormValue("create_dbms") == "true" || r.FormValue("create_dbms") == "on"

	item := database.DatabaseItem{
		ID:     id,
		Domain: domain,
		Engine: engine,
		DBName: dbName,
		DBUser: dbUser,
		DBPass: dbPass,
	}

	if err := s.dbMgr.SaveOrUpdate(item, createInDBMS); err != nil {
		redir := r.Header.Get("Referer")
		if redir == "" {
			redir = "/databases"
		}
		cleanRedir := strings.Split(redir, "?")[0]
		http.Redirect(w, r, fmt.Sprintf("%s?err=%s", cleanRedir, url.QueryEscape(err.Error())), http.StatusSeeOther)
		return
	}

	// Sinkronkan ke vhost list jika ada domain terkait
	if domain != "" {
		s.vhostsMu.Lock()
		for i, vh := range s.vhostsList {
			if strings.EqualFold(vh.Domain, domain) {
				s.vhostsList[i].DBEngine = engine
				s.vhostsList[i].DBName = dbName
				s.vhostsList[i].DBUser = dbUser
				s.vhostsList[i].DBPass = dbPass
				break
			}
		}
		s.saveVHosts()
		s.vhostsMu.Unlock()
	}

	redir := r.Header.Get("Referer")
	if redir == "" {
		redir = "/databases"
	}
	cleanRedir := strings.Split(redir, "?")[0]
	http.Redirect(w, r, fmt.Sprintf("%s?msg=Database+%s+berhasil+disimpan!", cleanRedir, url.QueryEscape(dbName)), http.StatusSeeOther)
}

func (s *Server) handleAPIDatabaseDelete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/databases?err=Form+tidak+valid", http.StatusSeeOther)
		return
	}

	id := strings.TrimSpace(r.FormValue("id"))
	dropDBMS := r.FormValue("drop_dbms") == "1" || r.FormValue("drop_dbms") == "true"

	item := s.dbMgr.GetByID(id)
	domain := ""
	if item != nil {
		domain = item.Domain
	}

	if err := s.dbMgr.Delete(id, dropDBMS); err != nil {
		http.Redirect(w, r, fmt.Sprintf("/databases?err=%s", url.QueryEscape(err.Error())), http.StatusSeeOther)
		return
	}

	if domain != "" {
		s.vhostsMu.Lock()
		for i, vh := range s.vhostsList {
			if strings.EqualFold(vh.Domain, domain) {
				s.vhostsList[i].DBEngine = ""
				s.vhostsList[i].DBName = ""
				s.vhostsList[i].DBUser = ""
				s.vhostsList[i].DBPass = ""
				break
			}
		}
		s.saveVHosts()
		s.vhostsMu.Unlock()
	}

	redir := r.Header.Get("Referer")
	if redir == "" {
		redir = "/databases"
	}
	cleanRedir := strings.Split(redir, "?")[0]
	http.Redirect(w, r, fmt.Sprintf("%s?msg=Database+berhasil+dihapus", cleanRedir), http.StatusSeeOther)
}

func (s *Server) handleAPIDatabaseExport(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		http.Error(w, "ID database tidak boleh kosong", http.StatusBadRequest)
		return
	}

	data, filename, err := s.dbMgr.Export(id)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/databases?err=%s", url.QueryEscape("Export gagal: "+err.Error())), http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", "application/sql")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	_, _ = w.Write(data)
}

func (s *Server) handleAPIDatabaseImport(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(128 << 20); err != nil {
		http.Redirect(w, r, "/databases?err=Gagal+mengunggah+file+SQL", http.StatusSeeOther)
		return
	}

	id := strings.TrimSpace(r.FormValue("id"))
	file, _, err := r.FormFile("sql_file")
	if err != nil {
		http.Redirect(w, r, "/databases?err=Pilih+file+SQL+terlebih+dahulu", http.StatusSeeOther)
		return
	}
	defer file.Close()

	sqlBytes, err := io.ReadAll(file)
	if err != nil {
		http.Redirect(w, r, "/databases?err=Gagal+membaca+file+SQL", http.StatusSeeOther)
		return
	}

	if err := s.dbMgr.Import(id, sqlBytes); err != nil {
		http.Redirect(w, r, fmt.Sprintf("/databases?err=%s", url.QueryEscape("Import gagal: "+err.Error())), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/databases?msg=File+SQL+berhasil+di-import+ke+database!", http.StatusSeeOther)
}
