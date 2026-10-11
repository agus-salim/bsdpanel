package php

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"bsdpanel/internal/system"
)

// PHPVersionInfo describes a supported or detected PHP version on FreeBSD.
type PHPVersionInfo struct {
	Version     string `json:"version"`      // e.g. "8.3"
	Tag         string `json:"tag"`          // e.g. "php83"
	PkgName     string `json:"pkg_name"`     // e.g. "php83"
	IsInstalled bool   `json:"is_installed"`
	FullVersion string `json:"full_version"` // e.g. "8.3.33"
	BinaryPath  string `json:"binary_path"`  // e.g. "/usr/local/bin/php83"
	IsDefault   bool   `json:"is_default"`   // true if /usr/local/bin/php points to this version
	FPMRunning  bool   `json:"fpm_running"`
}

// PHPExtensionDef defines metadata for a known PHP extension.
type PHPExtensionDef struct {
	Name        string `json:"name"`        // e.g. "pdo_pgsql"
	PkgSuffix   string `json:"pkg_suffix"`   // e.g. "pdo_pgsql" or "pecl-redis"
	Category    string `json:"category"`    // e.g. "Database", "Core & Network", "Media", "Security & Cache"
	Description string `json:"description"` // e.g. "Driver PDO untuk database PostgreSQL"
}

// PHPExtensionStatus represents the live status of an extension for a given PHP version.
type PHPExtensionStatus struct {
	Name        string `json:"name"`
	PkgName     string `json:"pkg_name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	IsInstalled bool   `json:"is_installed"`
	Version     string `json:"version,omitempty"`
}

// PHPManager handles PHP version discovery, extensions, and lifecycle on FreeBSD.
type PHPManager struct {
	exec       *system.Executor
	pkgMgr     *system.PkgManager
	serviceMgr *system.ServiceManager
	mu         sync.Mutex
}

// NewPHPManager creates a new PHPManager instance.
func NewPHPManager(exec *system.Executor, pkgMgr *system.PkgManager, serviceMgr *system.ServiceManager) *PHPManager {
	return &PHPManager{
		exec:       exec,
		pkgMgr:     pkgMgr,
		serviceMgr: serviceMgr,
	}
}

// KnownExtensionCatalog contains common and enterprise PHP extensions for FreeBSD.
var KnownExtensionCatalog = []PHPExtensionDef{
	// Database
	{Name: "pdo_pgsql", PkgSuffix: "pdo_pgsql", Category: "Database", Description: "Driver PDO untuk koneksi database PostgreSQL (Enterprise)"},
	{Name: "pgsql", PkgSuffix: "pgsql", Category: "Database", Description: "Driver native procedural/OOP PostgreSQL (pg_connect)"},
	{Name: "pdo_mysql", PkgSuffix: "pdo_mysql", Category: "Database", Description: "Driver PDO untuk database MariaDB & MySQL"},
	{Name: "mysqli", PkgSuffix: "mysqli", Category: "Database", Description: "Driver native MySQLi untuk MariaDB & MySQL"},
	{Name: "pdo_sqlite", PkgSuffix: "pdo_sqlite", Category: "Database", Description: "Driver PDO untuk database file SQLite"},
	{Name: "sqlite3", PkgSuffix: "sqlite3", Category: "Database", Description: "Driver native SQLite3 database engine"},
	{Name: "pecl-redis", PkgSuffix: "pecl-redis", Category: "Database", Description: "Klien Redis in-memory cache, session & queue"},

	// Core, Web & Network
	{Name: "curl", PkgSuffix: "curl", Category: "Core & Network", Description: "Client URL Library untuk komunikasi HTTP/REST API"},
	{Name: "mbstring", PkgSuffix: "mbstring", Category: "Core & Network", Description: "Multibyte string support untuk encoding UTF-8"},
	{Name: "intl", PkgSuffix: "intl", Category: "Core & Network", Description: "Internationalization (ICU) untuk tanggal, mata uang, dan angka"},
	{Name: "bcmath", PkgSuffix: "bcmath", Category: "Core & Network", Description: "Arbitrary Precision Mathematics untuk kalkulasi finansial"},
	{Name: "soap", PkgSuffix: "soap", Category: "Core & Network", Description: "Simple Object Access Protocol (SOAP) Client & Server"},
	{Name: "sockets", PkgSuffix: "sockets", Category: "Core & Network", Description: "Low-level socket interface untuk koneksi TCP/UDP"},
	{Name: "openssl", PkgSuffix: "openssl", Category: "Core & Network", Description: "Kriptografi OpenSSL & dukungan koneksi HTTPS"},

	// Document & File Handling
	{Name: "zip", PkgSuffix: "zip", Category: "Files & Documents", Description: "Kompresi & ekstraksi file ZIP (Wajib untuk Composer & Excel)"},
	{Name: "fileinfo", PkgSuffix: "fileinfo", Category: "Files & Documents", Description: "Deteksi MIME type file secara akurat untuk upload aman"},
	{Name: "xml", PkgSuffix: "xml", Category: "Files & Documents", Description: "Dukungan XML parsing standar"},
	{Name: "simplexml", PkgSuffix: "simplexml", Category: "Files & Documents", Description: "SimpleXML parser untuk manipulasi dokumen XML"},
	{Name: "bz2", PkgSuffix: "bz2", Category: "Files & Documents", Description: "Kompresi bzip2 file archive"},

	// Graphics & Media
	{Name: "gd", PkgSuffix: "gd", Category: "Graphics & Media", Description: "GD Graphics Library untuk resize, crop & watermark foto"},
	{Name: "exif", PkgSuffix: "exif", Category: "Graphics & Media", Description: "Membaca metadata EXIF kamera dari gambar JPEG/TIFF"},
	{Name: "pecl-imagick", PkgSuffix: "pecl-imagick", Category: "Graphics & Media", Description: "ImageMagick library untuk pengolahan gambar tingkat lanjut"},

	// Security & Performance
	{Name: "opcache", PkgSuffix: "opcache", Category: "Security & Cache", Description: "Zend OPcache bytecode caching untuk akselerasi eksekusi PHP"},
	{Name: "sodium", PkgSuffix: "sodium", Category: "Security & Cache", Description: "Modern cryptography libsodium (enkripsi password & token)"},
	{Name: "gmp", PkgSuffix: "gmp", Category: "Security & Cache", Description: "GNU Multiple Precision library untuk kriptografi tingkat tinggi"},
}

// ListVersions discovers all installed and available PHP versions on FreeBSD.
func (p *PHPManager) ListVersions(ctx context.Context) ([]PHPVersionInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	supportedVersions := []string{"8.2", "8.3", "8.4"}
	var results []PHPVersionInfo

	// Detect default CLI PHP version
	defaultVer := ""
	cliCmd := "/usr/local/bin/php"
	if _, err := os.Stat(cliCmd); err == nil {
		res, err := p.exec.Execute(ctx, cliCmd, "-r", "echo PHP_MAJOR_VERSION . '.' . PHP_MINOR_VERSION;")
		if err == nil {
			defaultVer = strings.TrimSpace(res.Stdout)
		}
	}

	// Check if php-fpm is running
	fpmStatus, _ := p.serviceMgr.Status(ctx, "php-fpm")
	fpmRunning := fpmStatus != nil && fpmStatus.IsRunning

	for _, ver := range supportedVersions {
		tag := "php" + strings.ReplaceAll(ver, ".", "")
		binPath := fmt.Sprintf("/usr/local/bin/%s", tag)

		isInstalled := false
		fullVersion := ""

		// Check binary directly
		if _, err := os.Stat(binPath); err == nil {
			isInstalled = true
			if res, err := p.exec.Execute(ctx, binPath, "-r", "echo PHP_VERSION;"); err == nil {
				fullVersion = strings.TrimSpace(res.Stdout)
			}
		} else if ver == defaultVer && defaultVer != "" {
			// If binary is /usr/local/bin/php
			isInstalled = true
			if res, err := p.exec.Execute(ctx, cliCmd, "-r", "echo PHP_VERSION;"); err == nil {
				fullVersion = strings.TrimSpace(res.Stdout)
			}
		} else {
			// Fallback: check pkg info
			if p.pkgMgr.IsInstalled(ctx, tag) {
				isInstalled = true
				fullVersion = ver + " (pkg)"
			}
		}

		results = append(results, PHPVersionInfo{
			Version:     ver,
			Tag:         tag,
			PkgName:     tag,
			IsInstalled: isInstalled,
			FullVersion: fullVersion,
			BinaryPath:  binPath,
			IsDefault:   (ver == defaultVer),
			FPMRunning:  fpmRunning,
		})
	}

	return results, nil
}

// ListExtensions returns the status of all known extensions for the specified PHP version.
func (p *PHPManager) ListExtensions(ctx context.Context, version string) ([]PHPExtensionStatus, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	tag := "php" + strings.ReplaceAll(version, ".", "")

	// 1. Query installed packages starting with the php tag (e.g. "php83-*")
	installedPkgMap := make(map[string]string)
	cmd := "/usr/sbin/pkg"
	args := []string{"info", tag + "*"}
	if res, err := p.exec.Execute(ctx, cmd, args...); err == nil && res.Stdout != "" {
		for _, line := range strings.Split(res.Stdout, "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				pkgWithVer := fields[0] // e.g. "php83-pdo_pgsql-8.3.33"
				// Extract package base name
				parts := strings.Split(pkgWithVer, "-")
				if len(parts) >= 2 {
					// join all except the last component which is version
					baseName := strings.Join(parts[:len(parts)-1], "-")
					installedPkgMap[baseName] = parts[len(parts)-1]
				}
			}
		}
	}

	// 2. Query active PHP modules via CLI (e.g., `php83 -m` or `php -m`)
	activeModules := make(map[string]bool)
	binCandidates := []string{
		fmt.Sprintf("/usr/local/bin/%s", tag),
		"/usr/local/bin/php",
	}
	for _, bin := range binCandidates {
		if _, err := os.Stat(bin); err == nil {
			if res, err := p.exec.Execute(ctx, bin, "-m"); err == nil && res.Stdout != "" {
				for _, line := range strings.Split(res.Stdout, "\n") {
					mod := strings.ToLower(strings.TrimSpace(line))
					if mod != "" {
						activeModules[mod] = true
					}
				}
				break
			}
		}
	}

	var results []PHPExtensionStatus
	for _, ext := range KnownExtensionCatalog {
		fullPkgName := fmt.Sprintf("%s-%s", tag, ext.PkgSuffix)

		isInstalled := false
		extVer := ""

		// Check if package is installed via pkg info
		if ver, ok := installedPkgMap[fullPkgName]; ok {
			isInstalled = true
			extVer = ver
		} else if activeModules[strings.ToLower(ext.Name)] {
			// Module is active in PHP runtime
			isInstalled = true
		}

		results = append(results, PHPExtensionStatus{
			Name:        ext.Name,
			PkgName:     fullPkgName,
			Category:    ext.Category,
			Description: ext.Description,
			IsInstalled: isInstalled,
			Version:     extVer,
		})
	}

	return results, nil
}

// InstallExtension installs a specific PHP extension via FreeBSD pkg and restarts PHP-FPM.
func (p *PHPManager) InstallExtension(ctx context.Context, version, extName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	tag := "php" + strings.ReplaceAll(version, ".", "")
	pkgSuffix := extName
	for _, e := range KnownExtensionCatalog {
		if strings.EqualFold(e.Name, extName) || strings.EqualFold(e.PkgSuffix, extName) {
			pkgSuffix = e.PkgSuffix
			break
		}
	}

	fullPkgName := fmt.Sprintf("%s-%s", tag, pkgSuffix)

	// Install package
	if err := p.pkgMgr.Install(ctx, fullPkgName); err != nil {
		return fmt.Errorf("gagal menginstall paket '%s': %w", fullPkgName, err)
	}

	// Restart PHP-FPM service so PHP runtime loads the new extension immediately
	_ = p.serviceMgr.Restart(ctx, "php-fpm")
	return nil
}

// UninstallExtension removes a specific PHP extension via FreeBSD pkg and restarts PHP-FPM.
func (p *PHPManager) UninstallExtension(ctx context.Context, version, extName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	tag := "php" + strings.ReplaceAll(version, ".", "")
	pkgSuffix := extName
	for _, e := range KnownExtensionCatalog {
		if strings.EqualFold(e.Name, extName) || strings.EqualFold(e.PkgSuffix, extName) {
			pkgSuffix = e.PkgSuffix
			break
		}
	}

	fullPkgName := fmt.Sprintf("%s-%s", tag, pkgSuffix)

	// Uninstall package
	if err := p.pkgMgr.Uninstall(ctx, fullPkgName); err != nil {
		return fmt.Errorf("gagal menghapus paket '%s': %w", fullPkgName, err)
	}

	// Restart PHP-FPM service
	_ = p.serviceMgr.Restart(ctx, "php-fpm")
	return nil
}

// RestartPHP restarts the PHP-FPM service.
func (p *PHPManager) RestartPHP(ctx context.Context) error {
	return p.serviceMgr.Restart(ctx, "php-fpm")
}
