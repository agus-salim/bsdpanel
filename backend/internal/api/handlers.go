package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/agus-salim/bsdpanel/internal/config"
	"github.com/agus-salim/bsdpanel/internal/ssl"
	"github.com/agus-salim/bsdpanel/internal/system"
	"github.com/agus-salim/bsdpanel/internal/terminal"
	"github.com/agus-salim/bsdpanel/internal/vhost"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type API struct {
	cfg     *config.Config
	db      *sql.DB
	bsdMgr  *system.FreeBSDManager
	acmeMgr *ssl.ACMEManager
}

func NewAPI(cfg *config.Config, db *sql.DB, bsdMgr *system.FreeBSDManager) *API {
	return &API{
		cfg:     cfg,
		db:      db,
		bsdMgr:  bsdMgr,
		acmeMgr: ssl.NewACMEManager(),
	}
}

// JSON Helper
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// Auth: Login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (a *API) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	var id int
	var passwordHash, role, status string
	err := a.db.QueryRow("SELECT id, password_hash, role, status FROM users WHERE username = ?", req.Username).
		Scan(&id, &passwordHash, &role, &status)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Username atau password salah")
		return
	}

	if status != "active" {
		writeError(w, http.StatusForbidden, "Akun Anda dinonaktifkan")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "Username atau password salah")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  id,
		"username": req.Username,
		"role":     role,
		"exp":      time.Now().Add(time.Duration(a.cfg.Auth.TokenExpiration) * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(a.cfg.Auth.JWTSecret))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Gagal generate auth token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token": tokenString,
		"user": map[string]interface{}{
			"id":       id,
			"username": req.Username,
			"role":     role,
		},
	})
}

// Dashboard: Get System Stats
func (a *API) HandleGetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := a.bsdMgr.GetSystemStats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var totalSites, totalDBs int
	_ = a.db.QueryRow("SELECT COUNT(*) FROM websites").Scan(&totalSites)
	_ = a.db.QueryRow("SELECT COUNT(*) FROM databases").Scan(&totalDBs)

	response := map[string]interface{}{
		"system": stats,
		"metrics": map[string]int{
			"total_websites":  totalSites,
			"total_databases": totalDBs,
		},
	}

	writeJSON(w, http.StatusOK, response)
}

// Services: List Service Status
func (a *API) HandleGetServices(w http.ResponseWriter, r *http.Request) {
	serviceNames := []string{
		"nginx",
		"apache24",
		"php81_fpm",
		"php82_fpm",
		"php83_fpm",
		"mysql_server",
		"postgresql",
	}

	results := make([]map[string]interface{}, 0, len(serviceNames))
	for _, s := range serviceNames {
		running := a.bsdMgr.CheckService(r.Context(), s)
		results = append(results, map[string]interface{}{
			"service": s,
			"running": running,
		})
	}

	writeJSON(w, http.StatusOK, results)
}

// Services: Action
type ServiceActionRequest struct {
	Service string `json:"service"`
	Action  string `json:"action"` // "restart", "reload", "start", "stop"
}

func (a *API) HandleServiceAction(w http.ResponseWriter, r *http.Request) {
	var req ServiceActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	out, err := a.bsdMgr.ServiceControl(r.Context(), req.Service, req.Action)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Operasi service berhasil",
		"output":  out,
	})
}

// Websites: List All
func (a *API) HandleListWebsites(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(`
		SELECT id, user_id, domain, document_root, web_server, php_version, ssl_enabled,
		       COALESCE(ssl_cert_path, ''), COALESCE(ssl_key_path, ''), status,
		       COALESCE(is_reverse_proxy, 0), COALESCE(proxy_target, ''),
		       COALESCE(proxy_websocket, 1), COALESCE(proxy_buffering, 1), created_at
		FROM websites ORDER BY id DESC
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var sites []map[string]interface{}
	for rows.Next() {
		var id, userID, sslEnabled, isReverseProxy, proxyWS, proxyBuffering int
		var domain, docRoot, webServer, phpVer, certPath, keyPath, status, proxyTarget, createdAt string
		if err := rows.Scan(&id, &userID, &domain, &docRoot, &webServer, &phpVer, &sslEnabled,
			&certPath, &keyPath, &status, &isReverseProxy, &proxyTarget, &proxyWS, &proxyBuffering, &createdAt); err == nil {
			sites = append(sites, map[string]interface{}{
				"id":               id,
				"user_id":          userID,
				"domain":           domain,
				"document_root":    docRoot,
				"web_server":       webServer,
				"php_version":      phpVer,
				"ssl_enabled":      sslEnabled == 1,
				"ssl_cert_path":    certPath,
				"ssl_key_path":     keyPath,
				"status":           status,
				"is_reverse_proxy": isReverseProxy == 1,
				"proxy_target":     proxyTarget,
				"proxy_websocket":  proxyWS == 1,
				"proxy_buffering":  proxyBuffering == 1,
				"created_at":       createdAt,
			})
		}
	}

	if sites == nil {
		sites = []map[string]interface{}{}
	}

	writeJSON(w, http.StatusOK, sites)
}

// Websites: Create New
type CreateWebsiteRequest struct {
	Domain         string `json:"domain"`
	WebServer      string `json:"web_server"`
	PhpVersion     string `json:"php_version"`
	EnableSSL      bool   `json:"enable_ssl"`
	IsReverseProxy bool   `json:"is_reverse_proxy"`
	ProxyTarget    string `json:"proxy_target"`
	ProxyWebSocket bool   `json:"proxy_websocket"`
}

func (a *API) HandleCreateWebsite(w http.ResponseWriter, r *http.Request) {
	var req CreateWebsiteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Domain == "" {
		writeError(w, http.StatusBadRequest, "Nama domain wajib diisi")
		return
	}

	if req.WebServer == "" {
		req.WebServer = "nginx"
	}
	if req.PhpVersion == "" {
		req.PhpVersion = "8.3"
	}

	docRoot := "/usr/local/www/" + req.Domain + "/public_html"
	sslInt := 0
	if req.EnableSSL {
		sslInt = 1
	}

	isProxyInt := 0
	if req.IsReverseProxy {
		isProxyInt = 1
	}

	wsInt := 1
	if !req.ProxyWebSocket {
		wsInt = 0
	}

	// 1. Simpan ke database
	res, err := a.db.Exec(`
		INSERT INTO websites (user_id, domain, document_root, web_server, php_version, ssl_enabled, is_reverse_proxy, proxy_target, proxy_websocket, status, created_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, 'active', CURRENT_TIMESTAMP)
	`, req.Domain, docRoot, req.WebServer, req.PhpVersion, sslInt, isProxyInt, req.ProxyTarget, wsInt)
	if err != nil {
		writeError(w, http.StatusConflict, "Domain sudah terdaftar atau terjadi error database: "+err.Error())
		return
	}

	lastID, _ := res.LastInsertId()

	// 2. Buat ZFS Dataset untuk isolasi kuota
	zfsDataset := a.cfg.FreeBSD.ZfsBaseDataset + "/" + strings.ReplaceAll(req.Domain, ".", "_")
	_ = a.bsdMgr.ZfsCreateDataset(r.Context(), zfsDataset, "2G")

	// 3. Generate Virtual Host Config file
	vhostPath := "/usr/local/etc/nginx/conf.d/" + req.Domain + ".conf"
	_ = vhost.GenerateVHost(req.WebServer, vhost.VHostConfig{
		Domain:         req.Domain,
		DocumentRoot:   docRoot,
		PhpVersion:     req.PhpVersion,
		SSLEnabled:     req.EnableSSL,
		IsReverseProxy: req.IsReverseProxy,
		ProxyTarget:    req.ProxyTarget,
		ProxyWebSocket: req.ProxyWebSocket,
	}, vhostPath)

	// 4. Verifikasi dan reload Nginx
	_, _ = vhost.TestNginxConfig(r.Context())
	_, _ = a.bsdMgr.ServiceControl(r.Context(), req.WebServer, "reload")

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":      lastID,
		"message": "Website berhasil dibuat dan vhost telah digenerate",
		"domain":  req.Domain,
	})
}

// SSL: Let's Encrypt Issue
type IssueSSLRequest struct {
	WebsiteID int    `json:"website_id"`
	Email     string `json:"email"`
}

func (a *API) HandleIssueSSL(w http.ResponseWriter, r *http.Request) {
	var req IssueSSLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.WebsiteID == 0 {
		writeError(w, http.StatusBadRequest, "ID Website tidak valid")
		return
	}

	var domain, docRoot, webServer, phpVersion, proxyTarget string
	var isReverseProxy, proxyWS int
	err := a.db.QueryRow(`
		SELECT domain, document_root, web_server, php_version,
		       COALESCE(is_reverse_proxy, 0), COALESCE(proxy_target, ''), COALESCE(proxy_websocket, 1)
		FROM websites WHERE id = ?
	`, req.WebsiteID).Scan(&domain, &docRoot, &webServer, &phpVersion, &isReverseProxy, &proxyTarget, &proxyWS)
	if err != nil {
		writeError(w, http.StatusNotFound, "Website tidak ditemukan")
		return
	}

	// Terbitkan sertifikat Let's Encrypt
	res, err := a.acmeMgr.IssueCertificate(r.Context(), domain, docRoot, req.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Gagal menerbitkan sertifikat: "+err.Error())
		return
	}

	// Update database
	_, _ = a.db.Exec(`
		UPDATE websites SET ssl_enabled = 1, ssl_cert_path = ?, ssl_key_path = ? WHERE id = ?
	`, res.CertPath, res.KeyPath, req.WebsiteID)

	// Regenerate vhost dengan SSL
	vhostPath := "/usr/local/etc/nginx/conf.d/" + domain + ".conf"
	_ = vhost.GenerateVHost(webServer, vhost.VHostConfig{
		Domain:         domain,
		DocumentRoot:   docRoot,
		PhpVersion:     phpVersion,
		SSLEnabled:     true,
		SSLCertPath:    res.CertPath,
		SSLKeyPath:     res.KeyPath,
		IsReverseProxy: isReverseProxy == 1,
		ProxyTarget:    proxyTarget,
		ProxyWebSocket: proxyWS == 1,
	}, vhostPath)

	// Uji dan reload Nginx
	_, _ = vhost.TestNginxConfig(r.Context())
	_, _ = a.bsdMgr.ServiceControl(r.Context(), "nginx", "reload")

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Sertifikat SSL Let's Encrypt berhasil dipasang dan Nginx telah direload",
		"result":  res,
	})
}

// Nginx Reverse Proxy: Update Settings via GUI
type UpdateProxyRequest struct {
	IsReverseProxy bool   `json:"is_reverse_proxy"`
	ProxyTarget    string `json:"proxy_target"`
	ProxyWebSocket bool   `json:"proxy_websocket"`
	ProxyBuffering bool   `json:"proxy_buffering"`
	MaxBodySizeMB  int    `json:"max_body_size_mb"`
}

func (a *API) HandleUpdateProxy(w http.ResponseWriter, r *http.Request) {
	siteIDStr := chi.URLParam(r, "id")
	siteID, err := strconv.Atoi(siteIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "ID Website tidak valid")
		return
	}

	var req UpdateProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	var domain, docRoot, webServer, phpVersion, certPath, keyPath string
	var sslEnabled int
	err = a.db.QueryRow(`
		SELECT domain, document_root, web_server, php_version, ssl_enabled,
		       COALESCE(ssl_cert_path, ''), COALESCE(ssl_key_path, '')
		FROM websites WHERE id = ?
	`, siteID).Scan(&domain, &docRoot, &webServer, &phpVersion, &sslEnabled, &certPath, &keyPath)
	if err != nil {
		writeError(w, http.StatusNotFound, "Website tidak ditemukan")
		return
	}

	isProxyInt := 0
	if req.IsReverseProxy {
		isProxyInt = 1
	}
	wsInt := 0
	if req.ProxyWebSocket {
		wsInt = 1
	}
	bufInt := 0
	if req.ProxyBuffering {
		bufInt = 1
	}

	// Update record di database
	_, err = a.db.Exec(`
		UPDATE websites
		SET is_reverse_proxy = ?, proxy_target = ?, proxy_websocket = ?, proxy_buffering = ?
		WHERE id = ?
	`, isProxyInt, req.ProxyTarget, wsInt, bufInt, siteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Gagal memperbarui konfigurasi di database")
		return
	}

	// Generate file VHost Nginx
	vhostPath := "/usr/local/etc/nginx/conf.d/" + domain + ".conf"
	err = vhost.GenerateVHost("nginx", vhost.VHostConfig{
		Domain:         domain,
		DocumentRoot:   docRoot,
		PhpVersion:     phpVersion,
		SSLEnabled:     sslEnabled == 1,
		SSLCertPath:    certPath,
		SSLKeyPath:     keyPath,
		IsReverseProxy: req.IsReverseProxy,
		ProxyTarget:    req.ProxyTarget,
		ProxyWebSocket: req.ProxyWebSocket,
		ProxyBuffering: req.ProxyBuffering,
		MaxBodySizeMB:  req.MaxBodySizeMB,
	}, vhostPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Gagal generate vhost: "+err.Error())
		return
	}

	// Test syntax nginx -t
	testOut, testErr := vhost.TestNginxConfig(r.Context())
	if testErr != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Syntax Nginx invalid! Pengaturan dibatalkan:\n%s", testOut))
		return
	}

	// Reload nginx
	reloadOut, _ := a.bsdMgr.ServiceControl(r.Context(), "nginx", "reload")

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":       "Pengaturan Nginx Reverse Proxy berhasil diperbarui dan direload!",
		"nginx_test":    testOut,
		"reload_output": reloadOut,
	})
}

// Router Setup
func (a *API) SetupRoutes() http.Handler {
	r := chi.NewRouter()

	// Auth routes
	r.Post("/api/auth/login", a.HandleLogin)

	// WebSocket Terminal (xterm.js)
	r.Get("/api/ws/terminal", terminal.HandleTerminalWebSocket)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Get("/api/dashboard/stats", a.HandleGetStats)
		r.Get("/api/services", a.HandleGetServices)
		r.Post("/api/services/control", a.HandleServiceAction)

		r.Get("/api/websites", a.HandleListWebsites)
		r.Post("/api/websites", a.HandleCreateWebsite)
		r.Post("/api/websites/{id}/proxy", a.HandleUpdateProxy)

		// Let's Encrypt SSL
		r.Post("/api/ssl/issue", a.HandleIssueSSL)
	})

	return r
}
