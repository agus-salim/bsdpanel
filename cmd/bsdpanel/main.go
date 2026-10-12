package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/agus-salim/bsdpanel/internal/api"
	"github.com/agus-salim/bsdpanel/internal/auth"
	"github.com/agus-salim/bsdpanel/internal/config"
	"github.com/agus-salim/bsdpanel/internal/database"
	"github.com/agus-salim/bsdpanel/internal/php"
	"github.com/agus-salim/bsdpanel/internal/system"
	"github.com/agus-salim/bsdpanel/internal/webserver"
)

var (
	version = "1.0.0"
	commit  = "none"
	date    = "unknown"
)

const banner = `
  ____  ____  ____  ____                  _ 
 | __ )/ ___||  _ \|  _ \ __ _ _ __   ___| |
 |  _ \\___ \| | | | |_) / _` + "`" + ` | '_ \ / _ \ |
 | |_) |___) | |_| |  __/ (_| | | | |  __/ |
 |____/|____/|____/|_|   \__,_|_| |_|\___|_|
   FreeBSD Native Lightweight Web Panel
`

func main() {
	cfgPath := flag.String("config", "/usr/local/etc/bsdpanel/config.json", "Jalur berkas konfigurasi")
	listenAddr := flag.String("listen", "", "Alamat dengar (contoh :8080 atau 127.0.0.1:8080)")
	dryRun := flag.Bool("dry-run", false, "Mode simulasi tanpa mengeksekusi perintah FreeBSD riil")
	showVer := flag.Bool("version", false, "Tampilkan versi")
	flag.Parse()

	if *showVer {
		fmt.Printf("bsdpanel v%s (%s) built at %s for %s/%s\n", version, commit, date, runtime.GOOS, runtime.GOARCH)
		return
	}

	fmt.Print(banner)
	log.Printf("[INFO] Memulai BSDPanel v%s pada %s/%s...", version, runtime.GOOS, runtime.GOARCH)

	// Muat konfigurasi
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Printf("[WARN] Gagal membaca konfigurasi dari %s, menggunakan default: %v", *cfgPath, err)
		cfg = config.DefaultConfig()
	}

	if *listenAddr != "" {
		cfg.ListenAddr = *listenAddr
	}

	// Inisialisasi komponen sistem
	runner := system.NewRunner(*dryRun || cfg.DemoMode)
	authMgr := auth.NewAuthManager(cfg.SecretKey)
	dbMgr := database.NewManager(runner)

	// Template bawaan untuk Nginx VHost & PHP Pool
	nginxTmpl := `# Nginx VHost Managed by bsdpanel
server {
    listen 80;
    server_name {{.Domain}};
    root {{.DocumentRoot}};
    index index.php index.html;
    access_log /var/log/nginx/{{.Domain}}_access.log;
    error_log /var/log/nginx/{{.Domain}}_error.log;
    location / {
        try_files $uri $uri/ /index.php?$args;
    }
    {{if .HasPHP}}
    location ~ \.php$ {
        try_files $uri =404;
        fastcgi_pass unix:{{.PHPSocket}};
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        include fastcgi_params;
    }
    {{end}}
}`

	phpPoolTmpl := `[{{.User}}]
user = {{.User}}
group = www
listen = {{.SocketPath}}
listen.owner = www
listen.group = www
listen.mode = 0660
pm = dynamic
pm.max_children = 10
pm.start_servers = 2
pm.min_spare_servers = 1
pm.max_spare_servers = 3
php_admin_value[memory_limit] = {{.MemoryLimit}}
php_admin_value[upload_max_filesize] = {{.UploadMaxFilesize}}
php_admin_value[max_execution_time] = {{.MaxExecutionTime}}
`

	vhostMgr, err := webserver.NewManager(cfg.VHostDir, runner, nginxTmpl)
	if err != nil {
		log.Fatalf("[FATAL] Gagal inisialisasi VHost Manager: %v", err)
	}

	phpMgr, err := php.NewManager(cfg.PHPPoolDir, runner, phpPoolTmpl)
	if err != nil {
		log.Fatalf("[FATAL] Gagal inisialisasi PHP Pool Manager: %v", err)
	}

	// Buat server HTTP
	srv, err := api.NewServer(cfg, authMgr, runner, vhostMgr, phpMgr, dbMgr)
	if err != nil {
		log.Fatalf("[FATAL] Gagal inisialisasi Server: %v", err)
	}

	httpServer := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      srv.SetupRouter(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel shutdown anggun (graceful shutdown)
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("[INFO] BSDPanel siap melayani di http://%s", cfg.ListenAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] HTTP server error: %v", err)
		}
	}()

	<-stopChan
	log.Println("[INFO] Menerima sinyal terminasi, mematikan server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("[FATAL] Server terpaksa berhenti: %v", err)
	}

	log.Println("[INFO] BSDPanel berhasil berhenti dengan aman.")
}
