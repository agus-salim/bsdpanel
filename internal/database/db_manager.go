package database

import (
	"context"
	"fmt"
	"strings"

	"bsdpanel/internal/system"
)

// DatabaseRecord represents a managed database instance.
type DatabaseRecord struct {
	Name      string `json:"name"`
	Type      string `json:"type"` // mariadb, postgresql
	OwnerUser string `json:"owner_user"`
	Host      string `json:"host"`
	Charset   string `json:"charset"`
}

// DatabaseManager handles MariaDB and PostgreSQL provisioning and dump operations.
type DatabaseManager struct {
	exec *system.Executor
}

// NewDatabaseManager creates a database manager instance.
func NewDatabaseManager(exec *system.Executor) *DatabaseManager {
	return &DatabaseManager{exec: exec}
}

// CreateMariaDBDatabase creates a database, dedicated user, and grants privileges.
func (d *DatabaseManager) CreateMariaDBDatabase(ctx context.Context, dbName, dbUser, dbPass string) error {
	if strings.ContainsAny(dbName, " ;'\"`") || strings.ContainsAny(dbUser, " ;'\"`") {
		return fmt.Errorf("unsafe database or username")
	}

	sql := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci; "+
			"CREATE USER IF NOT EXISTS '%s'@'localhost' IDENTIFIED BY '%s'; "+
			"GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'localhost'; "+
			"FLUSH PRIVILEGES;",
		dbName, dbUser, dbPass, dbName, dbUser,
	)

	cmd := "/usr/local/bin/mariadb"
	args := []string{"-u", "root", "-e", sql}
	if d.exec.UseDoas {
		args = append([]string{cmd}, args...)
		cmd = "/usr/local/bin/doas"
	}

	_, err := d.exec.Execute(ctx, cmd, args...)
	return err
}

// DropMariaDBDatabase removes a database.
func (d *DatabaseManager) DropMariaDBDatabase(ctx context.Context, dbName string) error {
	if strings.ContainsAny(dbName, " ;'\"`") {
		return fmt.Errorf("unsafe database name")
	}

	sql := fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", dbName)
	cmd := "/usr/local/bin/mariadb"
	args := []string{"-u", "root", "-e", sql}
	if d.exec.UseDoas {
		args = append([]string{cmd}, args...)
		cmd = "/usr/local/bin/doas"
	}

	_, err := d.exec.Execute(ctx, cmd, args...)
	return err
}

// ExportMariaDBDump exports a database to a SQL dump file.
func (d *DatabaseManager) ExportMariaDBDump(ctx context.Context, dbName, targetPath string) error {
	cmd := "/usr/local/bin/mariadb-dump"
	args := []string{"-u", "root", dbName, fmt.Sprintf("-r%s", targetPath)}
	if d.exec.UseDoas {
		args = append([]string{cmd}, args...)
		cmd = "/usr/local/bin/doas"
	}

	_, err := d.exec.Execute(ctx, cmd, args...)
	return err
}

// CreatePostgreSQLDatabase creates a Postgres DB and user role under the 'postgres' FreeBSD user.
func (d *DatabaseManager) CreatePostgreSQLDatabase(ctx context.Context, dbName, dbUser, dbPass string) error {
	if strings.ContainsAny(dbName, " ;'\"`") || strings.ContainsAny(dbUser, " ;'\"`") {
		return fmt.Errorf("unsafe postgres database or username")
	}

	sql := fmt.Sprintf(
		"CREATE USER \"%s\" WITH PASSWORD '%s'; "+
			"CREATE DATABASE \"%s\" OWNER \"%s\"; "+
			"GRANT ALL PRIVILEGES ON DATABASE \"%s\" TO \"%s\";",
		dbUser, dbPass, dbName, dbUser, dbName, dbUser,
	)

	cmd := "/usr/local/bin/psql"
	args := []string{"-U", "postgres", "-c", sql}
	if d.exec.UseDoas {
		// Run as postgres user
		args = append([]string{"-u", "postgres", cmd}, args...)
		cmd = "/usr/local/bin/doas"
	}

	_, err := d.exec.Execute(ctx, cmd, args...)
	return err
}
