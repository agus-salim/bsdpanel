package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const SchemaSQL = `
-- Tabel Pengguna (RBAC)
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username VARCHAR(64) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(128) UNIQUE NOT NULL,
    role VARCHAR(32) NOT NULL DEFAULT 'user', -- 'admin', 'reseller', 'user'
    status VARCHAR(32) NOT NULL DEFAULT 'active', -- 'active', 'suspended'
    disk_quota_mb INTEGER DEFAULT 5120, -- default 5GB
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Tabel FreeBSD Jails (Isolasi per user atau group)
CREATE TABLE IF NOT EXISTS jails (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name VARCHAR(64) UNIQUE NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    jail_path VARCHAR(255) NOT NULL,
    zfs_dataset VARCHAR(255) NOT NULL,
    status VARCHAR(32) DEFAULT 'stopped', -- 'running', 'stopped'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Tabel Website / Virtual Hosts
CREATE TABLE IF NOT EXISTS websites (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    domain VARCHAR(255) UNIQUE NOT NULL,
    document_root VARCHAR(255) NOT NULL,
    web_server VARCHAR(32) NOT NULL DEFAULT 'nginx', -- 'nginx', 'apache', 'openlitespeed'
    php_version VARCHAR(16) NOT NULL DEFAULT '8.3', -- '8.1', '8.2', '8.3', 'none'
    ssl_enabled INTEGER DEFAULT 0,
    ssl_cert_path VARCHAR(255),
    ssl_key_path VARCHAR(255),
    jail_id INTEGER,
    status VARCHAR(32) DEFAULT 'active',
    is_reverse_proxy INTEGER DEFAULT 0,
    proxy_target VARCHAR(255),
    proxy_websocket INTEGER DEFAULT 1,
    proxy_buffering INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(jail_id) REFERENCES jails(id) ON DELETE SET NULL
);

-- Tabel Multi-Database (MySQL & PostgreSQL)
CREATE TABLE IF NOT EXISTS databases (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    type VARCHAR(32) NOT NULL, -- 'mysql', 'postgresql'
    db_name VARCHAR(64) NOT NULL,
    db_user VARCHAR(64) NOT NULL,
    host VARCHAR(64) DEFAULT 'localhost',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(type, db_name)
);

-- Tabel Audit Logs
CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER,
    action VARCHAR(64) NOT NULL,
    target VARCHAR(255),
    details TEXT,
    ip_address VARCHAR(45),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

// InitDB membuka koneksi SQLite, menyetel optimasi WAL, dan mengeksekusi skema
func InitDB(dsn string) (*sql.DB, error) {
	dir := filepath.Dir(dsn)
	if dir != "." && dir != "/" {
		_ = os.MkdirAll(dir, 0750)
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("gagal membuka database SQLite: %w", err)
	}

	// Performance Pragma untuk FreeBSD ZFS / UFS
	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return nil, fmt.Errorf("gagal menjalankan pragma '%s': %w", pragma, err)
		}
	}

	// Migrasi Schema
	if _, err := db.Exec(SchemaSQL); err != nil {
		return nil, fmt.Errorf("gagal mengeksekusi schema migration: %w", err)
	}

	// Auto-upgrade kolom reverse proxy jika tabel sudah ada sebelumnya
	_, _ = db.Exec("ALTER TABLE websites ADD COLUMN is_reverse_proxy INTEGER DEFAULT 0;")
	_, _ = db.Exec("ALTER TABLE websites ADD COLUMN proxy_target VARCHAR(255);")
	_, _ = db.Exec("ALTER TABLE websites ADD COLUMN proxy_websocket INTEGER DEFAULT 1;")
	_, _ = db.Exec("ALTER TABLE websites ADD COLUMN proxy_buffering INTEGER DEFAULT 1;")

	// Buat default admin jika belum ada pengguna sama sekali
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err == nil && count == 0 {
		// Default bcrypt hash untuk 'admin123456'
		defaultHash := "$2a$10$8cI5pE3y4lq0Yw43/Uf7VesGeqQe1h3t687a4s3vR09N92q0x2fxe"
		_, _ = db.Exec(`
			INSERT INTO users (username, password_hash, email, role, status, disk_quota_mb, created_at, updated_at)
			VALUES ('admin', ?, 'admin@freebsd-panel.local', 'admin', 'active', 51200, ?, ?)
		`, defaultHash, time.Now(), time.Now())
	}

	return db, nil
}
