package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agus-salim/bsdpanel/internal/api"
	"github.com/agus-salim/bsdpanel/internal/config"
	"github.com/agus-salim/bsdpanel/internal/database"
	"github.com/agus-salim/bsdpanel/internal/system"
	"github.com/go-chi/cors"
)

func main() {
	log.Println("==================================================")
	log.Println("  BSDPanel - Native FreeBSD Web Hosting Control Panel ")
	log.Println("==================================================")

	// 1. Load Konfigurasi
	cfgPath := "config/panel.yaml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// 2. Inisialisasi Database Internal (SQLite WAL)
	db, err := database.InitDB(cfg.Database.DSN)
	if err != nil {
		log.Fatalf("Gagal inisialisasi database: %v", err)
	}
	defer db.Close()
	log.Printf("Database SQLite aktif di: %s", cfg.Database.DSN)

	// 3. Inisialisasi FreeBSD System Manager
	bsdMgr := system.NewFreeBSDManager(cfg.FreeBSD.ZfsPool)

	// 4. Setup API Router & CORS
	apiHandler := api.NewAPI(cfg, db, bsdMgr)
	router := apiHandler.SetupRoutes()

	// Wrap CORS untuk akses frontend dev server
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	})

	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      corsMiddleware.Handler(router),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 5. Jalankan HTTP Server di Goroutine
	go func() {
		log.Printf("Panel daemon berjalan di http://%s", serverAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// 6. Graceful Shutdown (menangkap sinyal SIGINT / SIGTERM dari FreeBSD rc.d)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Menerima sinyal terminasi, mematikan server secara graceful...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown dipaksa: %v", err)
	}

	log.Println("BSDPanel berhasil dimatikan dengan aman.")
}
