package php

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	Name        string `json:"name"`         // e.g. "pdo_pgsql"
	PkgSuffix   string `json:"pkg_suffix"`   // e.g. "pdo_pgsql" or "pecl-redis"
	Category    string `json:"category"`     // e.g. "Database", "Core & Network", "Media", "Security & Cache"
	Description string `json:"description"`  // e.g. "Driver PDO untuk database PostgreSQL"
	IniFileName string `json:"ini_filename"` // e.g. "ext-30-pdo_pgsql.ini"
	Directive   string `json:"directive"`    // e.g. "extension=pdo_pgsql.so"
	ModuleName  string `json:"module_name"`  // e.g. "pdo_pgsql"
	IsPDO       bool   `json:"is_pdo,omitempty"`
}

// PHPExtensionStatus represents the live status of an extension for a given PHP version.
type PHPExtensionStatus struct {
	Name        string `json:"name"`
	PkgName     string `json:"pkg_name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	IsInstalled bool   `json:"is_installed"`
	IsActive    bool   `json:"is_active"` // true if extension is loaded into PHP runtime
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
	{
		Name: "pdo_pgsql", PkgSuffix: "pdo_pgsql", Category: "Database",
		Description: "Driver PDO untuk koneksi database PostgreSQL (Enterprise)",
		IniFileName: "ext-30-pdo_pgsql.ini", Directive: "extension=pdo_pgsql.so", ModuleName: "pdo_pgsql", IsPDO: true,
	},
	{
		Name: "pgsql", PkgSuffix: "pgsql", Category: "Database",
		Description: "Driver native procedural/OOP PostgreSQL (pg_connect)",
		IniFileName: "ext-20-pgsql.ini", Directive: "extension=pgsql.so", ModuleName: "pgsql",
	},
	{
		Name: "pdo_mysql", PkgSuffix: "pdo_mysql", Category: "Database",
		Description: "Driver PDO untuk database MariaDB & MySQL",
		IniFileName: "ext-30-pdo_mysql.ini", Directive: "extension=pdo_mysql.so", ModuleName: "pdo_mysql", IsPDO: true,
	},
	{
		Name: "mysqli", PkgSuffix: "mysqli", Category: "Database",
		Description: "Driver native MySQLi untuk MariaDB & MySQL",
		IniFileName: "ext-20-mysqli.ini", Directive: "extension=mysqli.so", ModuleName: "mysqli",
	},
	{
		Name: "pdo_sqlite", PkgSuffix: "pdo_sqlite", Category: "Database",
		Description: "Driver PDO untuk database file SQLite",
		IniFileName: "ext-30-pdo_sqlite.ini", Directive: "extension=pdo_sqlite.so", ModuleName: "pdo_sqlite", IsPDO: true,
	},
	{
		Name: "sqlite3", PkgSuffix: "sqlite3", Category: "Database",
		Description: "Driver native SQLite3 database engine",
		IniFileName: "ext-20-sqlite3.ini", Directive: "extension=sqlite3.so", ModuleName: "sqlite3",
	},
	{
		Name: "pecl-redis", PkgSuffix: "pecl-redis", Category: "Database",
		Description: "Klien Redis in-memory cache, session & queue",
		IniFileName: "ext-30-redis.ini", Directive: "extension=redis.so", ModuleName: "redis",
	},

	// Core, Web & Network
	{
		Name: "curl", PkgSuffix: "curl", Category: "Core & Network",
		Description: "Client URL Library untuk komunikasi HTTP/REST API",
		IniFileName: "ext-20-curl.ini", Directive: "extension=curl.so", ModuleName: "curl",
	},
	{
		Name: "mbstring", PkgSuffix: "mbstring", Category: "Core & Network",
		Description: "Multibyte string support untuk encoding UTF-8",
		IniFileName: "ext-20-mbstring.ini", Directive: "extension=mbstring.so", ModuleName: "mbstring",
	},
	{
		Name: "intl", PkgSuffix: "intl", Category: "Core & Network",
		Description: "Internationalization (ICU) untuk tanggal, mata uang, dan angka",
		IniFileName: "ext-20-intl.ini", Directive: "extension=intl.so", ModuleName: "intl",
	},
	{
		Name: "bcmath", PkgSuffix: "bcmath", Category: "Core & Network",
		Description: "Arbitrary Precision Mathematics untuk kalkulasi finansial",
		IniFileName: "ext-20-bcmath.ini", Directive: "extension=bcmath.so", ModuleName: "bcmath",
	},
	{
		Name: "soap", PkgSuffix: "soap", Category: "Core & Network",
		Description: "Simple Object Access Protocol (SOAP) Client & Server",
		IniFileName: "ext-20-soap.ini", Directive: "extension=soap.so", ModuleName: "soap",
	},
	{
		Name: "sockets", PkgSuffix: "sockets", Category: "Core & Network",
		Description: "Low-level socket interface untuk koneksi TCP/UDP",
		IniFileName: "ext-20-sockets.ini", Directive: "extension=sockets.so", ModuleName: "sockets",
	},
	{
		Name: "openssl", PkgSuffix: "openssl", Category: "Core & Network",
		Description: "Kriptografi OpenSSL & dukungan koneksi HTTPS",
		IniFileName: "ext-20-openssl.ini", Directive: "extension=openssl.so", ModuleName: "openssl",
	},

	// Document & File Handling
	{
		Name: "zip", PkgSuffix: "zip", Category: "Files & Documents",
		Description: "Kompresi & ekstraksi file ZIP (Wajib untuk Composer & Excel)",
		IniFileName: "ext-20-zip.ini", Directive: "extension=zip.so", ModuleName: "zip",
	},
	{
		Name: "fileinfo", PkgSuffix: "fileinfo", Category: "Files & Documents",
		Description: "Deteksi MIME type file secara akurat untuk upload aman",
		IniFileName: "ext-20-fileinfo.ini", Directive: "extension=fileinfo.so", ModuleName: "fileinfo",
	},
	{
		Name: "xml", PkgSuffix: "xml", Category: "Files & Documents",
		Description: "Dukungan XML parsing standar",
		IniFileName: "ext-20-xml.ini", Directive: "extension=xml.so", ModuleName: "xml",
	},
	{
		Name: "simplexml", PkgSuffix: "simplexml", Category: "Files & Documents",
		Description: "SimpleXML parser untuk manipulasi dokumen XML",
		IniFileName: "ext-20-simplexml.ini", Directive: "extension=simplexml.so", ModuleName: "SimpleXML",
	},
	{
		Name: "bz2", PkgSuffix: "bz2", Category: "Files & Documents",
		Description: "Kompresi bzip2 file archive",
		IniFileName: "ext-20-bz2.ini", Directive: "extension=bz2.so", ModuleName: "bz2",
	},

	// Graphics & Media
	{
		Name: "gd", PkgSuffix: "gd", Category: "Graphics & Media",
		Description: "GD Graphics Library untuk resize, crop & watermark foto",
		IniFileName: "ext-20-gd.ini", Directive: "extension=gd.so", ModuleName: "gd",
	},
	{
		Name: "exif", PkgSuffix: "exif", Category: "Graphics & Media",
		Description: "Membaca metadata EXIF kamera dari gambar JPEG/TIFF",
		IniFileName: "ext-20-exif.ini", Directive: "extension=exif.so", ModuleName: "exif",
	},
	{
		Name: "pecl-imagick", PkgSuffix: "pecl-imagick", Category: "Graphics & Media",
		Description: "ImageMagick library untuk pengolahan gambar tingkat lanjut",
		IniFileName: "ext-30-imagick.ini", Directive: "extension=imagick.so", ModuleName: "imagick",
	},

	// Security & Performance
	{
		Name: "opcache", PkgSuffix: "opcache", Category: "Security & Cache",
		Description: "Zend OPcache bytecode caching untuk akselerasi eksekusi PHP",
		IniFileName: "ext-30-opcache.ini", Directive: "zend_extension=opcache.so", ModuleName: "Zend OPcache",
	},
	{
		Name: "sodium", PkgSuffix: "sodium", Category: "Security & Cache",
		Description: "Modern cryptography libsodium (enkripsi password & token)",
		IniFileName: "ext-20-sodium.ini", Directive: "extension=sodium.so", ModuleName: "sodium",
	},
	{
		Name: "gmp", PkgSuffix: "gmp", Category: "Security & Cache",
		Description: "GNU Multiple Precision library untuk kriptografi tingkat tinggi",
		IniFileName: "ext-20-gmp.ini", Directive: "extension=gmp.so", ModuleName: "gmp",
	},
}

// EnsurePHPIni verifies and creates /usr/local/etc/php.ini from default production template if missing.
func (p *PHPManager) EnsurePHPIni(ctx context.Context) error {
	phpIniPath := "/usr/local/etc/php.ini"
	if _, err := os.Stat(phpIniPath); err == nil {
		return nil
	}

	prodTemplate := "/usr/local/etc/php.ini-production"
	devTemplate := "/usr/local/etc/php.ini-development"

	if data, err := os.ReadFile(prodTemplate); err == nil {
		return os.WriteFile(phpIniPath, data, 0644)
	}
	if data, err := os.ReadFile(devTemplate); err == nil {
		return os.WriteFile(phpIniPath, data, 0644)
	}

	minimalIni := `; PHP Configuration managed by BSD Panel
engine = On
short_open_tag = Off
precision = 14
output_buffering = 4096
zlib.output_compression = Off
implicit_flush = Off
max_execution_time = 300
max_input_time = 300
memory_limit = 256M
error_reporting = E_ALL & ~E_DEPRECATED & ~E_STRICT
display_errors = Off
display_startup_errors = Off
log_errors = On
log_errors_max_len = 1024
ignore_repeated_errors = Off
report_memleaks = On
post_max_size = 64M
file_uploads = On
upload_max_filesize = 64M
max_file_uploads = 20
allow_url_fopen = On
allow_url_include = Off
date.timezone = UTC
`
	return os.WriteFile(phpIniPath, []byte(minimalIni), 0644)
}

// EnsureExtensionIni ensures that an extension configuration file exists in /usr/local/etc/php/.
func (p *PHPManager) EnsureExtensionIni(ctx context.Context, ext PHPExtensionDef) error {
	phpEtcDir := "/usr/local/etc/php"
	_ = os.MkdirAll(phpEtcDir, 0755)

	targetPath := filepath.Join(phpEtcDir, ext.IniFileName)

	// If the file already exists and is non-empty, nothing to do
	if fi, err := os.Stat(targetPath); err == nil && fi.Size() > 0 {
		return nil
	}

	// If a .sample exists, copy it
	samplePath := targetPath + ".sample"
	if data, err := os.ReadFile(samplePath); err == nil && len(data) > 0 {
		return os.WriteFile(targetPath, data, 0644)
	}

	// Otherwise, generate the standard directive
	content := fmt.Sprintf("; %s configuration enabled by BSD Panel\n%s\n", ext.Name, ext.Directive)
	return os.WriteFile(targetPath, []byte(content), 0644)
}

// SyncAndRepairExtensions synchronizes all installed FreeBSD PHP packages with their active INI configuration.
func (p *PHPManager) SyncAndRepairExtensions(ctx context.Context, version string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	tag := "php" + strings.ReplaceAll(version, ".", "")

	// 1. Ensure /usr/local/etc/php.ini exists
	_ = p.EnsurePHPIni(ctx)

	// 2. Query all locally installed packages
	installedPkgMap := make(map[string]bool)
	cmd := "/usr/sbin/pkg"
	args := []string{"query", "%n"}
	if res, err := p.exec.Execute(ctx, cmd, args...); err == nil && res.Stdout != "" {
		for _, line := range strings.Split(res.Stdout, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				installedPkgMap[trimmed] = true
			}
		}
	}

	// 3. For every known extension that is installed, ensure its .ini is active
	phpEtcDir := "/usr/local/etc/php"
	_ = os.MkdirAll(phpEtcDir, 0755)

	// Ensure base PDO .ini exists if any PDO extension is present
	pdoPkgName := fmt.Sprintf("%s-pdo", tag)
	hasPDO := installedPkgMap[pdoPkgName] || p.pkgMgr.IsInstalled(ctx, pdoPkgName)
	if hasPDO {
		pdoIni := filepath.Join(phpEtcDir, "ext-20-pdo.ini")
		if _, err := os.Stat(pdoIni); os.IsNotExist(err) {
			_ = os.WriteFile(pdoIni, []byte("; PDO Core enabled by BSD Panel\nextension=pdo.so\n"), 0644)
		}
	}

	for _, ext := range KnownExtensionCatalog {
		fullPkgName := fmt.Sprintf("%s-%s", tag, ext.PkgSuffix)
		isPkgInstalled := installedPkgMap[fullPkgName] || p.pkgMgr.IsInstalled(ctx, fullPkgName)

		if isPkgInstalled {
			_ = p.EnsureExtensionIni(ctx, ext)
		}
	}

	// 4. Thoroughly restart PHP-FPM
	return p.restartPHPService(ctx, version)
}

// restartPHPService restarts FreeBSD php-fpm using multiple fallback mechanisms.
func (p *PHPManager) restartPHPService(ctx context.Context, version string) error {
	_, _ = p.exec.Execute(ctx, "/usr/sbin/sysrc", "php_fpm_enable=YES")

	// Standard service restarts
	_, _ = p.exec.Execute(ctx, "/usr/sbin/service", "php-fpm", "restart")
	_, _ = p.exec.Execute(ctx, "/usr/sbin/service", "php-fpm", "start")
	_, _ = p.exec.Execute(ctx, "/usr/sbin/service", "php-fpm", "onerestart")

	// Version-tagged service (e.g. php83-fpm) if FreeBSD uses separate rc.d scripts
	if version != "" {
		tag := strings.ReplaceAll(version, ".", "")
		svcName := fmt.Sprintf("php%s-fpm", tag)
		rcVar := fmt.Sprintf("php%s_fpm_enable=YES", tag)
		_, _ = p.exec.Execute(ctx, "/usr/sbin/sysrc", rcVar)
		_, _ = p.exec.Execute(ctx, "/usr/sbin/service", svcName, "restart")
		_, _ = p.exec.Execute(ctx, "/usr/sbin/service", svcName, "start")
		_, _ = p.exec.Execute(ctx, "/usr/sbin/service", svcName, "onerestart")
	}

	// Graceful reload signal to running PHP-FPM master processes so pool workers load new extensions
	_, _ = p.exec.Execute(ctx, "/usr/bin/killall", "-USR2", "php-fpm")

	return nil
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

// ListExtensions returns the live status of all known extensions for the specified PHP version.
func (p *PHPManager) ListExtensions(ctx context.Context, version string) ([]PHPExtensionStatus, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	tag := "php" + strings.ReplaceAll(version, ".", "")

	// 1. Query installed packages from FreeBSD local pkg database
	installedPkgMap := make(map[string]string)
	cmd := "/usr/sbin/pkg"
	args := []string{"query", "%n %v"}
	if res, err := p.exec.Execute(ctx, cmd, args...); err == nil && res.Stdout != "" {
		for _, line := range strings.Split(res.Stdout, "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				pkgName := fields[0] // e.g. "php83-mysqli"
				pkgVer := fields[1]  // e.g. "8.3.33"
				if strings.HasPrefix(pkgName, tag+"-") || pkgName == tag {
					installedPkgMap[pkgName] = pkgVer
				}
			}
		}
	}

	// 2. Query live loaded extensions from PHP runtime (CLI)
	activeModulesMap := make(map[string]bool)
	phpBin := fmt.Sprintf("/usr/local/bin/%s", tag)
	if _, err := os.Stat(phpBin); err != nil {
		phpBin = "/usr/local/bin/php"
	}

	if _, err := os.Stat(phpBin); err == nil {
		phpCode := "echo json_encode(array_merge(get_loaded_extensions(), class_exists('PDO') ? PDO::getAvailableDrivers() : []));"
		if res, err := p.exec.Execute(ctx, phpBin, "-r", phpCode); err == nil && res.Stdout != "" {
			var loaded []string
			if err := json.Unmarshal([]byte(res.Stdout), &loaded); err == nil {
				for _, mod := range loaded {
					activeModulesMap[strings.ToLower(mod)] = true
				}
			}
		}
	}

	// 3. Evaluate and auto-heal missing configuration for each extension
	phpEtcDir := "/usr/local/etc/php"
	var results []PHPExtensionStatus

	for _, ext := range KnownExtensionCatalog {
		fullPkgName := fmt.Sprintf("%s-%s", tag, ext.PkgSuffix)

		isInstalled := false
		extVer := ""

		// Check pkg database
		if ver, ok := installedPkgMap[fullPkgName]; ok {
			isInstalled = true
			extVer = ver
		} else if p.pkgMgr.IsInstalled(ctx, fullPkgName) {
			isInstalled = true
		}

		// Check INI configuration file
		iniPath := filepath.Join(phpEtcDir, ext.IniFileName)
		iniExists := false
		if _, err := os.Stat(iniPath); err == nil {
			iniExists = true
		}

		// Also check alternate ini naming
		if !iniExists {
			candidates := []string{
				fmt.Sprintf("/usr/local/etc/php/ext-20-%s.ini", ext.PkgSuffix),
				fmt.Sprintf("/usr/local/etc/php/ext-30-%s.ini", ext.PkgSuffix),
				fmt.Sprintf("/usr/local/etc/%s/ext-20-%s.ini", tag, ext.PkgSuffix),
			}
			for _, c := range candidates {
				if _, err := os.Stat(c); err == nil {
					iniExists = true
					break
				}
			}
		}

		// Check if actively loaded in runtime
		modNameLower := strings.ToLower(ext.ModuleName)
		isActive := activeModulesMap[modNameLower] || activeModulesMap[strings.ToLower(ext.Name)]
		if ext.IsPDO {
			pdoDriver := strings.TrimPrefix(ext.Name, "pdo_")
			if activeModulesMap[strings.ToLower(pdoDriver)] {
				isActive = true
			}
		}

		// If package is installed, auto-heal missing INI if needed
		if isInstalled && !iniExists {
			_ = p.EnsureExtensionIni(ctx, ext)
			iniExists = true
		}

		// Considered installed if package exists, ini exists, or active in runtime
		if iniExists && !isInstalled {
			// Extension enabled via existing ini file
			isInstalled = true
		}

		results = append(results, PHPExtensionStatus{
			Name:        ext.Name,
			PkgName:     fullPkgName,
			Category:    ext.Category,
			Description: ext.Description,
			IsInstalled: isInstalled,
			IsActive:    isActive,
			Version:     extVer,
		})
	}

	return results, nil
}

// InstallExtension installs a specific PHP extension via FreeBSD pkg, enables its INI, and restarts PHP-FPM.
func (p *PHPManager) InstallExtension(ctx context.Context, version, extName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	tag := "php" + strings.ReplaceAll(version, ".", "")
	var targetExt *PHPExtensionDef
	for _, e := range KnownExtensionCatalog {
		if strings.EqualFold(e.Name, extName) || strings.EqualFold(e.PkgSuffix, extName) {
			def := e
			targetExt = &def
			break
		}
	}

	pkgSuffix := extName
	if targetExt != nil {
		pkgSuffix = targetExt.PkgSuffix
	}
	fullPkgName := fmt.Sprintf("%s-%s", tag, pkgSuffix)

	// If extension is a PDO driver, ensure core php-pdo is installed first
	if targetExt != nil && targetExt.IsPDO {
		pdoPkg := fmt.Sprintf("%s-pdo", tag)
		if !p.pkgMgr.IsInstalled(ctx, pdoPkg) {
			_ = p.pkgMgr.Install(ctx, pdoPkg)
		}
		pdoIni := "/usr/local/etc/php/ext-20-pdo.ini"
		if _, err := os.Stat(pdoIni); os.IsNotExist(err) {
			_ = os.WriteFile(pdoIni, []byte("; PDO Core enabled by BSD Panel\nextension=pdo.so\n"), 0644)
		}
	}

	// Install package via FreeBSD pkg
	if err := p.pkgMgr.Install(ctx, fullPkgName); err != nil {
		return fmt.Errorf("gagal menginstall paket '%s': %w", fullPkgName, err)
	}

	// Ensure php.ini exists
	_ = p.EnsurePHPIni(ctx)

	// Ensure extension .ini configuration file exists in /usr/local/etc/php/
	if targetExt != nil {
		_ = p.EnsureExtensionIni(ctx, *targetExt)
	} else {
		genericIni := fmt.Sprintf("/usr/local/etc/php/ext-30-%s.ini", extName)
		_ = os.WriteFile(genericIni, []byte(fmt.Sprintf("extension=%s.so\n", extName)), 0644)
	}

	// Restart PHP-FPM service so runtime loads the new extension immediately
	return p.restartPHPService(ctx, version)
}

// UninstallExtension removes a specific PHP extension via FreeBSD pkg, removes its INI, and restarts PHP-FPM.
func (p *PHPManager) UninstallExtension(ctx context.Context, version, extName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	tag := "php" + strings.ReplaceAll(version, ".", "")
	var targetExt *PHPExtensionDef
	for _, e := range KnownExtensionCatalog {
		if strings.EqualFold(e.Name, extName) || strings.EqualFold(e.PkgSuffix, extName) {
			def := e
			targetExt = &def
			break
		}
	}

	pkgSuffix := extName
	if targetExt != nil {
		pkgSuffix = targetExt.PkgSuffix
	}
	fullPkgName := fmt.Sprintf("%s-%s", tag, pkgSuffix)

	// Uninstall package
	if err := p.pkgMgr.Uninstall(ctx, fullPkgName); err != nil {
		return fmt.Errorf("gagal menghapus paket '%s': %w", fullPkgName, err)
	}

	// Remove .ini files to avoid "unable to load dynamic library" warnings
	if targetExt != nil {
		_ = os.Remove(filepath.Join("/usr/local/etc/php", targetExt.IniFileName))
	}
	_ = os.Remove(fmt.Sprintf("/usr/local/etc/php/ext-20-%s.ini", pkgSuffix))
	_ = os.Remove(fmt.Sprintf("/usr/local/etc/php/ext-30-%s.ini", pkgSuffix))

	// Restart PHP-FPM service
	return p.restartPHPService(ctx, version)
}

// RestartPHP restarts the PHP-FPM service with all fallbacks.
func (p *PHPManager) RestartPHP(ctx context.Context) error {
	return p.restartPHPService(ctx, "")
}
