package database

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"bsdpanel/internal/system"
)

// DatabaseRecord represents a managed database instance.
type DatabaseRecord struct {
	Name      string `json:"name"`
	Type      string `json:"type"`       // mariadb, postgresql
	Site      string `json:"site"`       // linked website domain (e.g., go.mtsn1sekadau.sch.id)
	OwnerUser string `json:"owner_user"` // database username
	Host      string `json:"host"`
	Charset   string `json:"charset"`
	Size      string `json:"size"`
	CreatedAt string `json:"created_at"`
}

// DatabaseManager handles MariaDB and PostgreSQL provisioning, listing, dump, and restore.
type DatabaseManager struct {
	exec    *system.Executor
	dataDir string
	mu      sync.Mutex
}

// NewDatabaseManager creates a database manager instance.
func NewDatabaseManager(exec *system.Executor, dataDir string) *DatabaseManager {
	if dataDir == "" {
		dataDir = "/var/db/bsdpanel"
	}
	return &DatabaseManager{
		exec:    exec,
		dataDir: dataDir,
	}
}

func (d *DatabaseManager) getMetaFilePath() string {
	_ = os.MkdirAll(d.dataDir, 0755)
	return filepath.Join(d.dataDir, "databases.json")
}

func (d *DatabaseManager) loadMetadata() []DatabaseRecord {
	filePath := d.getMetaFilePath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	var records []DatabaseRecord
	_ = json.Unmarshal(data, &records)
	return records
}

func (d *DatabaseManager) saveMetadata(records []DatabaseRecord) error {
	filePath := d.getMetaFilePath()
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

func (d *DatabaseManager) saveSingleRecord(rec DatabaseRecord) error {
	records := d.loadMetadata()
	found := false
	for i, r := range records {
		if strings.EqualFold(r.Name, rec.Name) && strings.EqualFold(r.Type, rec.Type) {
			records[i] = rec
			found = true
			break
		}
	}
	if !found {
		records = append(records, rec)
	}
	return d.saveMetadata(records)
}

func (d *DatabaseManager) removeRecord(dbType, dbName string) error {
	records := d.loadMetadata()
	var updated []DatabaseRecord
	for _, r := range records {
		if strings.EqualFold(r.Name, dbName) && strings.EqualFold(r.Type, dbType) {
			continue
		}
		updated = append(updated, r)
	}
	return d.saveMetadata(updated)
}

// ListDatabases discovers live databases from MariaDB and PostgreSQL, merging with metadata.
func (d *DatabaseManager) ListDatabases(ctx context.Context, knownSites []string) ([]DatabaseRecord, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	recordsMap := make(map[string]DatabaseRecord)
	for _, rec := range d.loadMetadata() {
		key := fmt.Sprintf("%s:%s", strings.ToLower(rec.Type), strings.ToLower(rec.Name))
		recordsMap[key] = rec
	}

	var result []DatabaseRecord
	discoveredKeys := make(map[string]bool)

	// 1. Discover live MariaDB databases
	mariaSql := "SELECT table_schema, IFNULL(ROUND(SUM(data_length + index_length) / 1024 / 1024, 2), 0) FROM information_schema.tables WHERE table_schema NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys') GROUP BY table_schema;"
	mariaCmd := "/usr/local/bin/mariadb"
	mariaArgs := []string{"-u", "root", "-s", "-N", "-e", mariaSql}
	if d.exec.UseDoas {
		mariaArgs = append([]string{mariaCmd}, mariaArgs...)
		mariaCmd = "/usr/local/bin/doas"
	}

	if res, err := d.exec.Execute(ctx, mariaCmd, mariaArgs...); err == nil && res.Stdout != "" {
		for _, line := range strings.Split(res.Stdout, "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				dbName := fields[0]
				sizeStr := "0.0 MB"
				if len(fields) >= 2 {
					sizeStr = fields[1] + " MB"
				}

				key := "mariadb:" + strings.ToLower(dbName)
				discoveredKeys[key] = true

				if rec, ok := recordsMap[key]; ok {
					rec.Size = sizeStr
					result = append(result, rec)
				} else {
					matchedSite := "-"
					matchedUser := dbName
					for _, s := range knownSites {
						clean := strings.ReplaceAll(strings.Split(s, ".")[0], "-", "_")
						if strings.HasPrefix(strings.ToLower(dbName), strings.ToLower(clean)) || strings.HasPrefix(strings.ToLower(dbName), "u_"+strings.ToLower(clean)) {
							matchedSite = s
							matchedUser = "u_" + clean
							break
						}
					}
					result = append(result, DatabaseRecord{
						Name:      dbName,
						Type:      "mariadb",
						Site:      matchedSite,
						OwnerUser: matchedUser,
						Host:      "localhost",
						Charset:   "utf8mb4",
						Size:      sizeStr,
						CreatedAt: time.Now().Format("02 Jan 2006"),
					})
				}
			}
		}
	} else {
		// Fallback: simple SHOW DATABASES
		fallbackArgs := []string{"-u", "root", "-s", "-N", "-e", "SHOW DATABASES;"}
		if d.exec.UseDoas {
			fallbackArgs = append([]string{mariaCmd}, fallbackArgs...)
		}
		if res2, err2 := d.exec.Execute(ctx, mariaCmd, fallbackArgs...); err2 == nil {
			for _, line := range strings.Split(res2.Stdout, "\n") {
				dbName := strings.TrimSpace(line)
				if dbName == "" || dbName == "information_schema" || dbName == "mysql" || dbName == "performance_schema" || dbName == "sys" {
					continue
				}
				key := "mariadb:" + strings.ToLower(dbName)
				discoveredKeys[key] = true
				if rec, ok := recordsMap[key]; ok {
					result = append(result, rec)
				} else {
					result = append(result, DatabaseRecord{
						Name:      dbName,
						Type:      "mariadb",
						Site:      "-",
						OwnerUser: dbName,
						Host:      "localhost",
						Charset:   "utf8mb4",
						Size:      "-",
						CreatedAt: time.Now().Format("02 Jan 2006"),
					})
				}
			}
		}
	}

	// 2. Discover live PostgreSQL databases
	pgCmd := "/usr/local/bin/psql"
	pgArgs := []string{"-U", "postgres", "-t", "-A", "-F", "|", "-c", "SELECT datname, pg_size_pretty(pg_database_size(datname)) FROM pg_database WHERE datistemplate = false AND datname NOT IN ('postgres', 'template0', 'template1');"}
	if d.exec.UseDoas {
		pgArgs = append([]string{"-u", "postgres", pgCmd}, pgArgs...)
		pgCmd = "/usr/local/bin/doas"
	}
	if res, err := d.exec.Execute(ctx, pgCmd, pgArgs...); err == nil && res.Stdout != "" {
		for _, line := range strings.Split(res.Stdout, "\n") {
			parts := strings.Split(strings.TrimSpace(line), "|")
			if len(parts) >= 1 && parts[0] != "" {
				dbName := parts[0]
				sizeStr := "-"
				if len(parts) >= 2 {
					sizeStr = parts[1]
				}
				key := "postgresql:" + strings.ToLower(dbName)
				discoveredKeys[key] = true
				if rec, ok := recordsMap[key]; ok {
					rec.Size = sizeStr
					result = append(result, rec)
				} else {
					result = append(result, DatabaseRecord{
						Name:      dbName,
						Type:      "postgresql",
						Site:      "-",
						OwnerUser: dbName,
						Host:      "localhost",
						Charset:   "UTF8",
						Size:      sizeStr,
						CreatedAt: time.Now().Format("02 Jan 2006"),
					})
				}
			}
		}
	}

	// Add any records from metadata that were not returned by live discovery (e.g. database server stopped)
	for key, rec := range recordsMap {
		if !discoveredKeys[key] {
			if rec.Size == "" {
				rec.Size = "Offline"
			}
			result = append(result, rec)
		}
	}

	return result, nil
}

// CreateMariaDBDatabase creates a database, dedicated user, and grants privileges.
func (d *DatabaseManager) CreateMariaDBDatabase(ctx context.Context, dbName, dbUser, dbPass, siteDomain string) error {
	if strings.ContainsAny(dbName, " ;'\"`\\") || strings.ContainsAny(dbUser, " ;'\"`\\") {
		return fmt.Errorf("nama database atau user mengandung karakter tidak aman")
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
	if err != nil {
		return err
	}

	_ = d.saveSingleRecord(DatabaseRecord{
		Name:      dbName,
		Type:      "mariadb",
		Site:      siteDomain,
		OwnerUser: dbUser,
		Host:      "localhost",
		Charset:   "utf8mb4",
		CreatedAt: time.Now().Format("02 Jan 2006 15:04"),
	})

	return nil
}

// CreatePostgreSQLDatabase creates a Postgres DB and user role under the 'postgres' FreeBSD user.
func (d *DatabaseManager) CreatePostgreSQLDatabase(ctx context.Context, dbName, dbUser, dbPass, siteDomain string) error {
	if strings.ContainsAny(dbName, " ;'\"`\\") || strings.ContainsAny(dbUser, " ;'\"`\\") {
		return fmt.Errorf("nama postgres database atau username mengandung karakter tidak aman")
	}

	sql := fmt.Sprintf(
		"DO $$ BEGIN IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '%s') THEN CREATE ROLE \"%s\" WITH LOGIN PASSWORD '%s'; ELSE ALTER ROLE \"%s\" WITH PASSWORD '%s'; END IF; END $$; "+
			"SELECT 'CREATE DATABASE \"%s\" OWNER \"%s\"' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '%s')\\gexec\n"+
			"GRANT ALL PRIVILEGES ON DATABASE \"%s\" TO \"%s\";",
		dbUser, dbUser, dbPass, dbUser, dbPass,
		dbName, dbUser, dbName,
		dbName, dbUser,
	)

	cmd := "/usr/local/bin/psql"
	args := []string{"-U", "postgres", "-c", sql}
	if d.exec.UseDoas {
		args = append([]string{"-u", "postgres", cmd}, args...)
		cmd = "/usr/local/bin/doas"
	}

	_, err := d.exec.Execute(ctx, cmd, args...)
	if err != nil {
		return err
	}

	_ = d.saveSingleRecord(DatabaseRecord{
		Name:      dbName,
		Type:      "postgresql",
		Site:      siteDomain,
		OwnerUser: dbUser,
		Host:      "localhost",
		Charset:   "UTF8",
		CreatedAt: time.Now().Format("02 Jan 2006 15:04"),
	})

	return nil
}

// DropDatabase removes a database and optionally its user.
func (d *DatabaseManager) DropDatabase(ctx context.Context, dbType, dbName, dbUser string) error {
	if strings.ContainsAny(dbName, " ;'\"`\\") {
		return fmt.Errorf("nama database mengandung karakter tidak valid")
	}

	if strings.EqualFold(dbType, "postgresql") {
		sql := fmt.Sprintf("DROP DATABASE IF EXISTS \"%s\";", dbName)
		if dbUser != "" && dbUser != "postgres" {
			sql += fmt.Sprintf(" DROP ROLE IF EXISTS \"%s\";", dbUser)
		}
		cmd := "/usr/local/bin/psql"
		args := []string{"-U", "postgres", "-c", sql}
		if d.exec.UseDoas {
			args = append([]string{"-u", "postgres", cmd}, args...)
			cmd = "/usr/local/bin/doas"
		}
		_, err := d.exec.Execute(ctx, cmd, args...)
		if err != nil {
			return err
		}
	} else {
		sql := fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", dbName)
		if dbUser != "" && dbUser != "root" {
			sql += fmt.Sprintf(" DROP USER IF EXISTS '%s'@'localhost'; FLUSH PRIVILEGES;", dbUser)
		}
		cmd := "/usr/local/bin/mariadb"
		args := []string{"-u", "root", "-e", sql}
		if d.exec.UseDoas {
			args = append([]string{cmd}, args...)
			cmd = "/usr/local/bin/doas"
		}
		_, err := d.exec.Execute(ctx, cmd, args...)
		if err != nil {
			return err
		}
	}

	_ = d.removeRecord(dbType, dbName)
	return nil
}

// UpdatePassword updates user password.
func (d *DatabaseManager) UpdatePassword(ctx context.Context, dbType, dbUser, newPassword string) error {
	if strings.ContainsAny(dbUser, " ;'\"`\\") {
		return fmt.Errorf("nama user database tidak valid")
	}

	if strings.EqualFold(dbType, "postgresql") {
		sql := fmt.Sprintf("ALTER ROLE \"%s\" WITH PASSWORD '%s';", dbUser, newPassword)
		cmd := "/usr/local/bin/psql"
		args := []string{"-U", "postgres", "-c", sql}
		if d.exec.UseDoas {
			args = append([]string{"-u", "postgres", cmd}, args...)
			cmd = "/usr/local/bin/doas"
		}
		_, err := d.exec.Execute(ctx, cmd, args...)
		return err
	}

	sql := fmt.Sprintf("ALTER USER '%s'@'localhost' IDENTIFIED BY '%s'; FLUSH PRIVILEGES;", dbUser, newPassword)
	cmd := "/usr/local/bin/mariadb"
	args := []string{"-u", "root", "-e", sql}
	if d.exec.UseDoas {
		args = append([]string{cmd}, args...)
		cmd = "/usr/local/bin/doas"
	}
	_, err := d.exec.Execute(ctx, cmd, args...)
	return err
}

// ExportDatabase dumps SQL to a stream.
func (d *DatabaseManager) ExportDatabase(ctx context.Context, dbType, dbName string, out io.Writer) error {
	if strings.ContainsAny(dbName, " ;'\"`\\") {
		return fmt.Errorf("nama database tidak valid")
	}

	if strings.EqualFold(dbType, "postgresql") {
		cmd := "/usr/local/bin/pg_dump"
		args := []string{"-U", "postgres", "--clean", "--if-exists", dbName}
		if d.exec.UseDoas {
			args = append([]string{"-u", "postgres", cmd}, args...)
			cmd = "/usr/local/bin/doas"
		}
		return d.exec.ExecuteWithStreams(ctx, nil, out, cmd, args...)
	}

	cmd := "/usr/local/bin/mariadb-dump"
	args := []string{"-u", "root", "--routines", "--triggers", dbName}
	if d.exec.UseDoas {
		args = append([]string{cmd}, args...)
		cmd = "/usr/local/bin/doas"
	}
	return d.exec.ExecuteWithStreams(ctx, nil, out, cmd, args...)
}

// ImportDatabase restores SQL from a stream.
func (d *DatabaseManager) ImportDatabase(ctx context.Context, dbType, dbName string, in io.Reader) error {
	if strings.ContainsAny(dbName, " ;'\"`\\") {
		return fmt.Errorf("nama database tidak valid")
	}

	if strings.EqualFold(dbType, "postgresql") {
		cmd := "/usr/local/bin/psql"
		args := []string{"-U", "postgres", "-d", dbName}
		if d.exec.UseDoas {
			args = append([]string{"-u", "postgres", cmd}, args...)
			cmd = "/usr/local/bin/doas"
		}
		return d.exec.ExecuteWithStreams(ctx, in, nil, cmd, args...)
	}

	cmd := "/usr/local/bin/mariadb"
	args := []string{"-u", "root", dbName}
	if d.exec.UseDoas {
		args = append([]string{cmd}, args...)
		cmd = "/usr/local/bin/doas"
	}
	return d.exec.ExecuteWithStreams(ctx, in, nil, cmd, args...)
}
