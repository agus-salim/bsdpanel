package database

import (
	"fmt"
	"time"

	"github.com/agus-salim/bsdpanel/internal/system"
)

type DBType string

const (
	TypeMariaDB    DBType = "mariadb"
	TypePostgreSQL DBType = "postgresql"
)

// Manager mengelola pembuatan database dan user untuk MariaDB/MySQL dan PostgreSQL di FreeBSD.
type Manager struct {
	runner system.Runner
}

func NewManager(runner system.Runner) *Manager {
	return &Manager{runner: runner}
}

// CreateMariaDBDatabase membuat database, user, dan memberikan hak akses di MariaDB/MySQL.
func (m *Manager) CreateMariaDBDatabase(dbName, dbUser, dbPass string) error {
	sql := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci; "+
			"CREATE USER IF NOT EXISTS '%s'@'localhost' IDENTIFIED BY '%s'; "+
			"GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'localhost'; "+
			"FLUSH PRIVILEGES;",
		dbName, dbUser, dbPass, dbName, dbUser,
	)

	res, err := m.runner.ExecWithTimeout(10*time.Second, "mysql", "-e", sql)
	if err != nil {
		return fmt.Errorf("gagal membuat database MariaDB: %s (%w)", res.Stderr, err)
	}
	return nil
}

// CreatePostgreSQLDatabase membuat database dan role di PostgreSQL FreeBSD.
func (m *Manager) CreatePostgreSQLDatabase(dbName, dbUser, dbPass string) error {
	createUserSQL := fmt.Sprintf("CREATE USER %s WITH ENCRYPTED PASSWORD '%s';", dbUser, dbPass)
	createDBSQL := fmt.Sprintf("CREATE DATABASE %s OWNER %s;", dbName, dbUser)

	// Di FreeBSD, superuser postgres default adalah user unix 'pgsql' atau 'postgres'
	_, _ = m.runner.ExecWithTimeout(10*time.Second, "psql", "-U", "pgsql", "-c", createUserSQL)
	res, err := m.runner.ExecWithTimeout(10*time.Second, "psql", "-U", "pgsql", "-c", createDBSQL)
	if err != nil {
		return fmt.Errorf("gagal membuat database PostgreSQL: %s (%w)", res.Stderr, err)
	}
	return nil
}
