package database

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/agus-salim/bsdpanel/internal/system"
)

type DBType string

const (
	TypeMariaDB    DBType = "mariadb"
	TypePostgreSQL DBType = "postgresql"
)

// DatabaseItem merepresentasikan database yang terhubung ke sebuah virtual host.
type DatabaseItem struct {
	ID        string `json:"id"`
	Domain    string `json:"domain"`
	Engine    string `json:"engine"` // "mariadb" atau "postgresql"
	DBName    string `json:"db_name"`
	DBUser    string `json:"db_user"`
	DBPass    string `json:"db_pass"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	CreatedAt string `json:"created_at"`
}

// Manager mengelola pembuatan, persistensi, import, dan export database MariaDB & PostgreSQL di FreeBSD.
type Manager struct {
	runner  system.Runner
	dataDir string
	mu      sync.RWMutex
	items   []DatabaseItem
}

func NewManager(runner system.Runner, dataDir string) *Manager {
	m := &Manager{
		runner:  runner,
		dataDir: dataDir,
	}
	m.load()
	return m
}

func (m *Manager) stateFilePath() string {
	return filepath.Join(m.dataDir, "databases.json")
}

func (m *Manager) load() {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.stateFilePath())
	if err == nil {
		var list []DatabaseItem
		if err := json.Unmarshal(data, &list); err == nil {
			m.items = list
		}
	}
}

func (m *Manager) save() error {
	_ = os.MkdirAll(m.dataDir, 0750)
	data, err := json.MarshalIndent(m.items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.stateFilePath(), data, 0644)
}

// GetAll mengembalikan seluruh daftar database yang terdaftar.
func (m *Manager) GetAll() []DatabaseItem {
	m.mu.RLock()
	defer m.mu.RUnlock()

	res := make([]DatabaseItem, len(m.items))
	copy(res, m.items)
	return res
}

// GetByID mencari database berdasarkan ID.
func (m *Manager) GetByID(id string) *DatabaseItem {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, item := range m.items {
		if item.ID == id {
			cp := item
			return &cp
		}
	}
	return nil
}

// GetByDomain mencari database yang terhubung ke domain tertentu.
func (m *Manager) GetByDomain(domain string) *DatabaseItem {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, item := range m.items {
		if strings.EqualFold(item.Domain, domain) {
			cp := item
			return &cp
		}
	}
	return nil
}

// SaveOrUpdate menyimpan atau memperbarui database dan opsional membuatnya di DBMS.
func (m *Manager) SaveOrUpdate(item DatabaseItem, createInDBMS bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	item.Domain = strings.TrimSpace(item.Domain)
	item.Engine = strings.ToLower(strings.TrimSpace(item.Engine))
	if item.Engine == "" {
		item.Engine = "mariadb"
	}
	item.DBName = strings.TrimSpace(item.DBName)
	item.DBUser = strings.TrimSpace(item.DBUser)
	item.DBPass = strings.TrimSpace(item.DBPass)

	if item.DBName == "" || item.DBUser == "" {
		return fmt.Errorf("nama database dan user database wajib diisi")
	}

	if item.Engine == "postgresql" {
		item.Port = 5432
	} else {
		item.Port = 3306
	}
	item.Host = "localhost"

	if item.ID == "" {
		item.ID = fmt.Sprintf("db_%d", time.Now().UnixNano())
	}
	if item.CreatedAt == "" {
		item.CreatedAt = time.Now().Format("2006-01-02")
	}

	// Buat di DBMS jika diminta
	if createInDBMS {
		if item.Engine == "postgresql" {
			if err := m.createPostgreSQL(item.DBName, item.DBUser, item.DBPass); err != nil {
				return err
			}
		} else {
			if err := m.createMariaDB(item.DBName, item.DBUser, item.DBPass); err != nil {
				return err
			}
		}
	}

	// Update list in memory
	found := false
	for i, existing := range m.items {
		if existing.ID == item.ID || (existing.Domain != "" && strings.EqualFold(existing.Domain, item.Domain)) {
			m.items[i] = item
			found = true
			break
		}
	}
	if !found {
		m.items = append(m.items, item)
	}

	_ = m.save()
	return nil
}

// Delete menghapus database dari daftar panel dan opsional drop dari DBMS.
func (m *Manager) Delete(id string, dropFromDBMS bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var target *DatabaseItem
	var filtered []DatabaseItem

	for _, item := range m.items {
		if item.ID == id {
			target = &item
		} else {
			filtered = append(filtered, item)
		}
	}

	if target == nil {
		return fmt.Errorf("database tidak ditemukan")
	}

	if dropFromDBMS {
		if target.Engine == "postgresql" {
			_ = m.dropPostgreSQL(target.DBName, target.DBUser)
		} else {
			_ = m.dropMariaDB(target.DBName, target.DBUser)
		}
	}

	m.items = filtered
	_ = m.save()
	return nil
}

// createMariaDB membuat database & user di MariaDB/MySQL.
func (m *Manager) createMariaDB(dbName, dbUser, dbPass string) error {
	sql := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci; "+
			"CREATE USER IF NOT EXISTS '%s'@'localhost' IDENTIFIED BY '%s'; "+
			"GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'localhost'; "+
			"FLUSH PRIVILEGES;",
		dbName, dbUser, dbPass, dbName, dbUser,
	)
	res, err := m.runner.ExecWithTimeout(15*time.Second, "mysql", "-e", sql)
	if err != nil {
		return fmt.Errorf("gagal membuat database MariaDB: %s (%w)", res.Stderr, err)
	}
	return nil
}

// dropMariaDB menghapus database & user dari MariaDB/MySQL.
func (m *Manager) dropMariaDB(dbName, dbUser string) error {
	sql := fmt.Sprintf(
		"DROP DATABASE IF EXISTS `%s`; "+
			"DROP USER IF EXISTS '%s'@'localhost'; "+
			"FLUSH PRIVILEGES;",
		dbName, dbUser,
	)
	_, err := m.runner.ExecWithTimeout(15*time.Second, "mysql", "-e", sql)
	return err
}

// createPostgreSQL membuat database & role di PostgreSQL.
func (m *Manager) createPostgreSQL(dbName, dbUser, dbPass string) error {
	createUserSQL := fmt.Sprintf("DO $$ BEGIN IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '%s') THEN CREATE ROLE %s WITH LOGIN PASSWORD '%s'; END IF; END $$;", dbUser, dbUser, dbPass)
	createDBSQL := fmt.Sprintf("CREATE DATABASE %s OWNER %s;", dbName, dbUser)

	// Superuser postgres FreeBSD default adalah pgsql
	_, _ = m.runner.ExecWithTimeout(10*time.Second, "psql", "-U", "pgsql", "-c", createUserSQL)
	res, err := m.runner.ExecWithTimeout(15*time.Second, "psql", "-U", "pgsql", "-c", createDBSQL)
	if err != nil {
		// Coba lagi dengan superuser unix postgres jika pgsql gagal
		_, _ = m.runner.ExecWithTimeout(10*time.Second, "psql", "-U", "postgres", "-c", createUserSQL)
		res2, err2 := m.runner.ExecWithTimeout(15*time.Second, "psql", "-U", "postgres", "-c", createDBSQL)
		if err2 != nil && !strings.Contains(res2.Stderr, "already exists") {
			return fmt.Errorf("gagal membuat database PostgreSQL: %s (%w)", res.Stderr, err)
		}
	}
	return nil
}

// dropPostgreSQL menghapus database & role di PostgreSQL.
func (m *Manager) dropPostgreSQL(dbName, dbUser string) error {
	dropDBSQL := fmt.Sprintf("DROP DATABASE IF EXISTS %s;", dbName)
	dropUserSQL := fmt.Sprintf("DROP ROLE IF EXISTS %s;", dbUser)

	_, _ = m.runner.ExecWithTimeout(10*time.Second, "psql", "-U", "pgsql", "-c", dropDBSQL)
	_, _ = m.runner.ExecWithTimeout(10*time.Second, "psql", "-U", "pgsql", "-c", dropUserSQL)
	return nil
}

// Export melakukan dump database dan mengembalikan berkas SQL.
func (m *Manager) Export(id string) ([]byte, string, error) {
	item := m.GetByID(id)
	if item == nil {
		return nil, "", fmt.Errorf("database dengan id %s tidak ditemukan", id)
	}

	filename := fmt.Sprintf("%s_backup_%s.sql", item.DBName, time.Now().Format("20060102_150405"))

	if item.Engine == "postgresql" {
		res, err := m.runner.ExecWithTimeout(60*time.Second, "pg_dump", "-U", "pgsql", "-d", item.DBName)
		if err != nil {
			res, err = m.runner.ExecWithTimeout(60*time.Second, "pg_dump", "-U", "postgres", "-d", item.DBName)
			if err != nil {
				return nil, "", fmt.Errorf("pg_dump gagal: %s (%w)", res.Stderr, err)
			}
		}
		return []byte(res.Stdout), filename, nil
	}

	// MariaDB / MySQL dump
	res, err := m.runner.ExecWithTimeout(60*time.Second, "mariadb-dump", "--single-transaction", "--quick", item.DBName)
	if err != nil {
		res, err = m.runner.ExecWithTimeout(60*time.Second, "mysqldump", "--single-transaction", "--quick", item.DBName)
		if err != nil && item.DBUser != "" {
			res, err = m.runner.ExecWithTimeout(60*time.Second, "mysqldump", "-u", item.DBUser, fmt.Sprintf("-p%s", item.DBPass), item.DBName)
		}
		if err != nil {
			return nil, "", fmt.Errorf("mysqldump gagal: %s (%w)", res.Stderr, err)
		}
	}

	return []byte(res.Stdout), filename, nil
}

// Import memasukkan file dump SQL ke database target.
func (m *Manager) Import(id string, sqlData []byte) error {
	item := m.GetByID(id)
	if item == nil {
		return fmt.Errorf("database dengan id %s tidak ditemukan", id)
	}

	// Tulis berkas dump sementara
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("bsdpanel_import_%d.sql", time.Now().UnixNano()))
	if err := os.WriteFile(tmpFile, sqlData, 0600); err != nil {
		return fmt.Errorf("gagal menulis file import sementara: %w", err)
	}
	defer os.Remove(tmpFile)

	if item.Engine == "postgresql" {
		res, err := m.runner.ExecWithTimeout(120*time.Second, "psql", "-U", "pgsql", "-d", item.DBName, "-f", tmpFile)
		if err != nil {
			res, err = m.runner.ExecWithTimeout(120*time.Second, "psql", "-U", "postgres", "-d", item.DBName, "-f", tmpFile)
			if err != nil {
				return fmt.Errorf("psql import gagal: %s (%w)", res.Stderr, err)
			}
		}
		return nil
	}

	// MariaDB import
	res, err := m.runner.ExecWithTimeout(120*time.Second, "mysql", item.DBName, "-e", fmt.Sprintf("source %s", tmpFile))
	if err != nil && item.DBUser != "" {
		res, err = m.runner.ExecWithTimeout(120*time.Second, "mysql", "-u", item.DBUser, fmt.Sprintf("-p%s", item.DBPass), item.DBName, "-e", fmt.Sprintf("source %s", tmpFile))
	}
	if err != nil {
		return fmt.Errorf("mysql import gagal: %s (%w)", res.Stderr, err)
	}

	return nil
}
