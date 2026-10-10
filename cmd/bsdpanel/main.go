package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"bsdpanel/internal/config"
	"bsdpanel/internal/database"
	"bsdpanel/internal/sites"
	"bsdpanel/internal/system"
	"bsdpanel/internal/web"
	webassets "bsdpanel/web"
)

func main() {
	configPath := flag.String("config", "config.json", "Path to bsdpanel configuration file")
	flag.Parse()

	log.Println("[INFO] Starting BSD Panel (Lightweight FreeBSD VPS Control Panel)...")

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("[FATAL] Failed to load configuration: %v", err)
	}

	// Initialize System Abstraction Layer
	exec := system.NewExecutor(cfg.DoasEnabled)
	userMgr := system.NewUserManager(exec)
	serviceMgr := system.NewServiceManager(exec)
	pkgMgr := system.NewPkgManager(exec)
	firewallMgr := system.NewFirewallManager(exec, serviceMgr)
	siteMgr := sites.NewSiteManager(userMgr, serviceMgr, exec, cfg.DataDir)
	dbMgr := database.NewDatabaseManager(exec)

	// Initialize Web Server with embedded UI assets
	server, err := web.NewServer(
		cfg,
		exec,
		userMgr,
		serviceMgr,
		pkgMgr,
		firewallMgr,
		siteMgr,
		dbMgr,
		webassets.Assets,
	)
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize web server: %v", err)
	}

	// Graceful shutdown channel
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-shutdownChan
		log.Println("[INFO] Shutting down BSD Panel safely...")
		os.Exit(0)
	}()

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("[INFO] BSD Panel running and listening on http://%s (Bound for Nginx Reverse Proxy)\n", addr)
	if err := server.Start(); err != nil {
		log.Fatalf("[FATAL] Server error: %v", err)
	}
}
