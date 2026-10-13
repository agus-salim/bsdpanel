package php

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SupportedPHP kelola versi PHP dari 8.2 hingga 8.5
var SupportedPHP = []struct {
	Version string // e.g. "8.2"
	Key     string // e.g. "php82"
	PkgName string // e.g. "php82"
}{
	{Version: "8.2", Key: "php82", PkgName: "php82"},
	{Version: "8.3", Key: "php83", PkgName: "php83"},
	{Version: "8.4", Key: "php84", PkgName: "php84"},
	{Version: "8.5", Key: "php85", PkgName: "php85"},
}

// PHPVersionInfo memuat status satu versi PHP di host FreeBSD.
type PHPVersionInfo struct {
	Version     string `json:"version"`      // "8.2", "8.3", "8.4", "8.5"
	Key         string `json:"key"`          // "php82", "php83", "php84", "php85"
	PkgName     string `json:"pkg_name"`     // "php82", "php83"
	Installed   bool   `json:"installed"`    // true jika terinstall
	FullVersion string `json:"full_version"` // e.g. "8.3.35"
	BinPath     string `json:"bin_path"`     // "/usr/local/bin/php"
	IniPath     string `json:"ini_path"`     // "/usr/local/etc/php.ini"
	FPMStatus   string `json:"fpm_status"`   // "running", "stopped", "not_installed"
	ActiveExts  int    `json:"active_exts"`  // jumlah ekstensi aktif
	TotalExts   int    `json:"total_exts"`   // total ekstensi yang dikenali
}

// PHPINISettings menyimpan parameter umum php.ini.
type PHPINISettings struct {
	Version           string `json:"version"`
	Key               string `json:"key"`
	IniPath           string `json:"ini_path"`
	MemoryLimit       string `json:"memory_limit"`
	MaxExecutionTime  string `json:"max_execution_time"`
	UploadMaxFilesize string `json:"upload_max_filesize"`
	PostMaxSize       string `json:"post_max_size"`
	MaxInputTime      string `json:"max_input_time"`
	Timezone          string `json:"timezone"`
	DisplayErrors     string `json:"display_errors"`
	ErrorReporting    string `json:"error_reporting"`
	AllowUrlFopen     string `json:"allow_url_fopen"`
	ShortOpenTag      string `json:"short_open_tag"`
}

// ExtensionMeta definisi katalog ekstensi PHP.
type ExtensionMeta struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// ExtensionInfo status ekstensi per versi PHP.
type ExtensionInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Status      string `json:"status"` // "active", "disabled", "not_installed"
	PkgName     string `json:"pkg_name"`
	IniFile     string `json:"ini_file"`
}

// CatalogExtensions daftar lengkap ekstensi populer PHP.
var CatalogExtensions = []ExtensionMeta{
	// Jaringan & Web
	{Name: "curl", DisplayName: "cURL", Description: "Transfer data HTTP/HTTPS/FTP dengan cURL", Category: "Jaringan"},
	{Name: "soap", DisplayName: "SOAP", Description: "Simple Object Access Protocol untuk web service XML", Category: "Jaringan"},

	// String & Teks
	{Name: "mbstring", DisplayName: "Mbstring", Description: "Manipulasi string multibyte (UTF-8, Emoji, CJK)", Category: "String"},
	{Name: "intl", DisplayName: "Intl (ICU)", Description: "Internasionalisasi, format tanggal dan angka ICU", Category: "String"},
	{Name: "iconv", DisplayName: "Iconv", Description: "Konversi set karakter (character encoding)", Category: "String"},
	{Name: "ctype", DisplayName: "CType", Description: "Pengecekan tipe karakter (alfanumerik, digit)", Category: "String"},
	{Name: "tokenizer", DisplayName: "Tokenizer", Description: "Parser token kode sumber PHP bawaan Zend", Category: "String"},

	// Database
	{Name: "pdo", DisplayName: "PDO Core", Description: "PHP Data Objects abstraction layer (pondasi wajib driver PDO)", Category: "Database"},
	{Name: "mysqli", DisplayName: "MySQLi", Description: "Driver MySQL dan MariaDB modern bawaan PHP", Category: "Database"},
	{Name: "pdo_mysql", DisplayName: "PDO MySQL", Description: "Driver PDO untuk database MySQL & MariaDB", Category: "Database"},
	{Name: "pgsql", DisplayName: "PostgreSQL", Description: "Driver native untuk database PostgreSQL", Category: "Database"},
	{Name: "pdo_pgsql", DisplayName: "PDO PostgreSQL", Description: "Driver PDO untuk database PostgreSQL", Category: "Database"},
	{Name: "sqlite3", DisplayName: "SQLite3", Description: "Driver database berkas portabel SQLite3", Category: "Database"},
	{Name: "pdo_sqlite", DisplayName: "PDO SQLite", Description: "Driver PDO untuk database SQLite3", Category: "Database"},

	// Arsip & Kompresi
	{Name: "zip", DisplayName: "Zip Archive", Description: "Membaca, membuat, dan mengekstrak berkas arsip .zip", Category: "Arsip"},
	{Name: "bz2", DisplayName: "Bzip2", Description: "Kompresi dan dekompresi data algoritma bzip2", Category: "Arsip"},

	// Media & Gambar
	{Name: "gd", DisplayName: "GD Library", Description: "Pembuatan & manipulasi gambar (JPEG, PNG, WebP)", Category: "Media"},
	{Name: "exif", DisplayName: "EXIF", Description: "Membaca metadata foto/gambar kamera (EXIF tags)", Category: "Media"},
	{Name: "imagick", DisplayName: "ImageMagick", Description: "Manipulasi grafis bitmap canggih ImageMagick", Category: "Media"},

	// XML & Format Data
	{Name: "xml", DisplayName: "XML Core", Description: "Parser dokumen XML bawaan PHP", Category: "Format Data"},
	{Name: "dom", DisplayName: "DOM", Description: "Manipulasi hirarki objek dokumen XML & HTML", Category: "Format Data"},
	{Name: "simplexml", DisplayName: "SimpleXML", Description: "Akses data XML ke dalam bentuk objek PHP ringkas", Category: "Format Data"},

	// Kriptografi & Keamanan
	{Name: "sodium", DisplayName: "Sodium", Description: "Kriptografi modern (Argon2, ChaCha20, Ed25519)", Category: "Keamanan"},
	{Name: "filter", DisplayName: "Filter", Description: "Validasi dan sanitasi input data formulir", Category: "Keamanan"},

	// Performa & Cache
	{Name: "opcache", DisplayName: "OPcache", Description: "Zend Bytecode Caching untuk akselerasi eksekusi PHP", Category: "Performa"},
	{Name: "redis", DisplayName: "Redis", Description: "Driver koneksi memori cache dan session Redis", Category: "Performa"},

	// Utilitas
	{Name: "fileinfo", DisplayName: "Fileinfo", Description: "Deteksi MIME type berkas secara akurat", Category: "Utilitas"},
	{Name: "bcmath", DisplayName: "BCMath", Description: "Kalkulasi matematika presisi tinggi (Arbitrary Precision)", Category: "Utilitas"},
	{Name: "session", DisplayName: "Session", Description: "Manajemen session web user bawaan PHP", Category: "Utilitas"},
}

// GetAvailableVersions mengembalikan daftar versi PHP 8.2 s/d 8.5 beserta statusnya.
func (m *Manager) GetAvailableVersions() []PHPVersionInfo {
	var list []PHPVersionInfo

	// Cek versi default dari /usr/local/etc/php.conf jika ada
	defaultVerKey := m.detectDefaultPHPVerKey()

	for _, item := range SupportedPHP {
		info := PHPVersionInfo{
			Version: item.Version,
			Key:     item.Key,
			PkgName: item.PkgName,
		}

		// Cek apakah terpasang
		installed, fullVer, binPath := m.checkInstalled(item.Key, defaultVerKey)
		info.Installed = installed
		info.FullVersion = fullVer
		info.BinPath = binPath
		info.IniPath = m.resolveIniPath(item.Key, installed)

		if installed {
			info.FPMStatus = m.checkFPMStatus(item.Key, defaultVerKey)
			exts, _ := m.GetExtensions(item.Key)
			activeCount := 0
			for _, ext := range exts {
				if ext.Status == "active" {
					activeCount++
				}
			}
			info.ActiveExts = activeCount
			info.TotalExts = len(CatalogExtensions)
		} else {
			info.FPMStatus = "not_installed"
			info.ActiveExts = 0
			info.TotalExts = len(CatalogExtensions)
		}

		list = append(list, info)
	}

	return list
}

// detectDefaultPHPVerKey membaca /usr/local/etc/php.conf untuk melihat PHP_VER aktif.
func (m *Manager) detectDefaultPHPVerKey() string {
	content, err := os.ReadFile("/usr/local/etc/php.conf")
	if err == nil {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "PHP_VER=") {
				verNum := strings.TrimPrefix(line, "PHP_VER=")
				verNum = strings.Trim(verNum, `"' `)
				return fmt.Sprintf("php%s", verNum)
			}
		}
	}
	return ""
}

// checkInstalled mengecek apakah versi PHP terinstall di FreeBSD.
func (m *Manager) checkInstalled(key string, defaultVerKey string) (bool, string, string) {
	// 1. Cek melalui binary khusus /usr/local/bin/<key>
	keyBin := fmt.Sprintf("/usr/local/bin/%s", key)
	if _, err := os.Stat(keyBin); err == nil {
		fullVer := m.getBinaryVersion(keyBin)
		return true, fullVer, keyBin
	}

	// 2. Cek apakah pkg terpasang: pkg info -e <key>
	res, err := m.runner.ExecWithTimeout(5*time.Second, "pkg", "info", "-e", key)
	if err == nil && res.ExitCode == 0 {
		binPath := "/usr/local/bin/php"
		if _, err := os.Stat(keyBin); err == nil {
			binPath = keyBin
		}
		fullVer := m.getBinaryVersion(binPath)
		return true, fullVer, binPath
	}

	// 3. Cek apakah binary default /usr/local/bin/php adalah versi ini
	phpBin := "/usr/local/bin/php"
	if _, err := os.Stat(phpBin); err == nil {
		fullVer := m.getBinaryVersion(phpBin)
		verNum := strings.TrimPrefix(key, "php") // "84" -> "8.4"
		if len(verNum) == 2 {
			prefix := fmt.Sprintf("%c.%c", verNum[0], verNum[1])
			if strings.HasPrefix(fullVer, prefix) {
				return true, fullVer, phpBin
			}
		}
	}

	// 4. Cek apakah ini default PHP yang sedang aktif
	if defaultVerKey == key {
		binPath := "/usr/local/bin/php"
		if _, err := os.Stat(binPath); err == nil {
			fullVer := m.getBinaryVersion(binPath)
			return true, fullVer, binPath
		}
	}

	return false, "", ""
}

// getBinaryVersion menjalankan `<bin> -v` untuk mengambil string versi.
func (m *Manager) getBinaryVersion(binPath string) string {
	res, err := m.runner.ExecWithTimeout(5*time.Second, binPath, "-v")
	if err == nil && res.Stdout != "" {
		re := regexp.MustCompile(`PHP\s+([0-9]+\.[0-9]+\.[0-9]+)`)
		matches := re.FindStringSubmatch(res.Stdout)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	// Alternatif baca dari /usr/local/etc/php.conf
	content, err := os.ReadFile("/usr/local/etc/php.conf")
	if err == nil {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "PHP_VERSION=") {
				return strings.Trim(strings.TrimPrefix(line, "PHP_VERSION="), `"' `)
			}
		}
	}

	return ""
}

// checkFPMStatus mengecek apakah service FPM berjalan.
func (m *Manager) checkFPMStatus(key string, defaultVerKey string) string {
	svcName := fmt.Sprintf("%s_fpm", key)
	res, err := m.runner.ExecWithTimeout(5*time.Second, "service", svcName, "status")
	if err == nil && res.ExitCode == 0 && strings.Contains(res.Stdout, "is running") {
		return "running"
	}

	// Cek juga lewat file PID
	pidFile := fmt.Sprintf("/var/run/%s-fpm.pid", key)
	if data, err := os.ReadFile(pidFile); err == nil {
		pidStr := strings.TrimSpace(string(data))
		if pidStr != "" {
			resKill, errKill := m.runner.ExecWithTimeout(5*time.Second, "kill", "-0", pidStr)
			if errKill == nil && resKill.ExitCode == 0 {
				return "running"
			}
		}
	}

	// Jika ini versi default, coba cek `service php_fpm status`
	if defaultVerKey == key || defaultVerKey == "" {
		resDefault, errDefault := m.runner.ExecWithTimeout(5*time.Second, "service", "php_fpm", "status")
		if errDefault == nil && resDefault.ExitCode == 0 && strings.Contains(resDefault.Stdout, "is running") {
			return "running"
		}
	}

	return "stopped"
}

// EnsureVersionFPMService memastikan konfigurasi FPM, pool directory, dan service rc.d ada dan aktif untuk versi ini.
func (m *Manager) EnsureVersionFPMService(key string) {
	binPath := fmt.Sprintf("/usr/local/sbin/%s-fpm", key)
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		// Jika belum ada bin spesifik tapi ada php-fpm default
		defBin := "/usr/local/sbin/php-fpm"
		if _, defErr := os.Stat(defBin); defErr == nil {
			if data, readErr := os.ReadFile(defBin); readErr == nil {
				_ = os.WriteFile(binPath, data, 0755)
			}
		} else {
			return
		}
	}

	confDir := fmt.Sprintf("/usr/local/etc/%s-fpm.d", key)
	_ = os.MkdirAll(confDir, 0755)

	confFile := fmt.Sprintf("/usr/local/etc/%s-fpm.conf", key)
	if _, err := os.Stat(confFile); os.IsNotExist(err) {
		confContent := fmt.Sprintf("[global]\npid = /var/run/%s-fpm.pid\nerror_log = /var/log/%s-fpm.log\ninclude = %s/*.conf\n", key, key, confDir)
		_ = os.WriteFile(confFile, []byte(confContent), 0644)
	}

	// Pastikan ada minimal 1 pool agar daemon bisa start
	wwwPool := filepath.Join(confDir, "www.conf")
	if _, err := os.Stat(wwwPool); os.IsNotExist(err) {
		poolContent := fmt.Sprintf("[www]\nuser = www\ngroup = www\nlisten = /var/run/%s-fpm.sock\nlisten.owner = www\nlisten.group = www\nlisten.mode = 0660\npm = dynamic\npm.max_children = 5\npm.start_servers = 1\npm.min_spare_servers = 1\npm.max_spare_servers = 3\nenv[PHP_INI_SCAN_DIR] = /usr/local/etc/%s\n", key, key)
		_ = os.WriteFile(wwwPool, []byte(poolContent), 0644)
	}

	iniPath := m.resolveIniPath(key, true)

	// rc.d script untuk daemon FPM versi spesifik
	rcFile := fmt.Sprintf("/usr/local/etc/rc.d/%s_fpm", key)
	rcContent := fmt.Sprintf(`#!/bin/sh
# PROVIDE: %[1]s_fpm
# REQUIRE: LOGIN
# KEYWORD: shutdown

. /etc/rc.subr

name="%[1]s_fpm"
rcvar="%[1]s_fpm_enable"

command="%[2]s"
pidfile="/var/run/%[1]s-fpm.pid"
required_files="%[3]s"
%[1]s_fpm_flags="-y %[3]s -g /var/run/%[1]s-fpm.pid -c %[4]s"

# Pastikan PHP membaca konfigurasi modul dari direktori versi spesifik
export PHP_INI_SCAN_DIR="/usr/local/etc/%[1]s"

load_rc_config "${name}"
: ${%[1]s_fpm_enable:="YES"}

run_rc_command "$1"
`, key, binPath, confFile, iniPath)
	_ = os.WriteFile(rcFile, []byte(rcContent), 0755)

	_, _ = m.runner.ExecWithTimeout(5*time.Second, "sysrc", fmt.Sprintf("%s_fpm_enable=YES", key))
	if m.checkFPMStatus(key, "") != "running" {
		resKey, errKey := m.runner.ExecWithTimeout(5*time.Second, "service", fmt.Sprintf("%s_fpm", key), "restart")
		if errKey != nil || resKey.ExitCode != 0 {
			_, _ = m.runner.ExecWithTimeout(5*time.Second, "service", fmt.Sprintf("%s_fpm", key), "start")
		}
	}
}

// resolveIniPath menentukan jalur php.ini yang sesuai.
func (m *Manager) resolveIniPath(key string, installed bool) string {
	versionIni := fmt.Sprintf("/usr/local/etc/%s/php.ini", key)
	if _, err := os.Stat(versionIni); err == nil {
		return versionIni
	}

	defaultIni := "/usr/local/etc/php.ini"
	if _, err := os.Stat(defaultIni); err == nil {
		return defaultIni
	}

	if installed {
		return versionIni
	}
	return defaultIni
}

// resolveExtDirs mengembalikan folder penyimpanan file .ini ekstensi (versi spesifik dan default FreeBSD).
func (m *Manager) resolveExtDirs(key string) []string {
	var dirs []string
	if key != "" {
		dirs = append(dirs, fmt.Sprintf("/usr/local/etc/%s", key))
	}
	dirs = append(dirs, "/usr/local/etc/php")
	return dirs
}

// resolveExtDir menentukan folder penyimpanan file .ini ekstensi.
func (m *Manager) resolveExtDir(key string) string {
	// 1. Cek folder versi khusus jika ada file .ini di dalamnya
	versionDir := fmt.Sprintf("/usr/local/etc/%s", key)
	if entries, err := os.ReadDir(versionDir); err == nil && len(entries) > 0 {
		return versionDir
	}

	// 2. Folder default FreeBSD untuk ekstensi PHP
	return "/usr/local/etc/php"
}

// InstallVersion memasang paket PHP FreeBSD via pkg.
func (m *Manager) InstallVersion(key string) (string, error) {
	// 1. Simpan binary dan ekstensi versi aktif saat ini ke direktori spesifiknya sebelum paket diubah
	phpBin := "/usr/local/bin/php"
	if _, err := os.Stat(phpBin); err == nil {
		curVer := m.getBinaryVersion(phpBin) // misal "8.4.26"
		if len(curVer) >= 3 {
			vMajorMinor := strings.ReplaceAll(curVer[:3], ".", "") // "84"
			oldKey := fmt.Sprintf("php%s", vMajorMinor)
			oldKeyBin := fmt.Sprintf("/usr/local/bin/%s", oldKey)
			if _, statErr := os.Stat(oldKeyBin); os.IsNotExist(statErr) {
				if binData, readErr := os.ReadFile(phpBin); readErr == nil {
					_ = os.WriteFile(oldKeyBin, binData, 0755)
				}
			}
			oldFpm := "/usr/local/sbin/php-fpm"
			oldKeyFpm := fmt.Sprintf("/usr/local/sbin/%s-fpm", oldKey)
			if _, statErr := os.Stat(oldKeyFpm); os.IsNotExist(statErr) {
				if fpmData, readErr := os.ReadFile(oldFpm); readErr == nil {
					_ = os.WriteFile(oldKeyFpm, fpmData, 0755)
				}
			}

			// Salin ekstensi yang sudah ada ke folder versi spesifik /usr/local/etc/<oldKey>
			oldExtDir := fmt.Sprintf("/usr/local/etc/%s", oldKey)
			_ = os.MkdirAll(oldExtDir, 0755)
			if entries, err := os.ReadDir("/usr/local/etc/php"); err == nil {
				for _, entry := range entries {
					src := filepath.Join("/usr/local/etc/php", entry.Name())
					dst := filepath.Join(oldExtDir, entry.Name())
					if data, err := os.ReadFile(src); err == nil {
						_ = os.WriteFile(dst, data, 0644)
					}
				}
			}
		}
	}

	// 2. Bersihkan konflik paket lama jika ada
	_, _ = m.runner.ExecWithTimeout(60*time.Second, "pkg", "delete", "-y", "php8*")

	// 3. Install paket PHP baru dan modul ekstensi lengkap via pkg FreeBSD
	pkgsToInstall := []string{
		key,
		fmt.Sprintf("%s-session", key),
		fmt.Sprintf("%s-filter", key),
		fmt.Sprintf("%s-ctype", key),
		fmt.Sprintf("%s-tokenizer", key),
		fmt.Sprintf("%s-dom", key),
		fmt.Sprintf("%s-xml", key),
		fmt.Sprintf("%s-iconv", key),
		fmt.Sprintf("%s-intl", key),
		fmt.Sprintf("%s-fileinfo", key),
		fmt.Sprintf("%s-curl", key),
		fmt.Sprintf("%s-pdo", key),
		fmt.Sprintf("%s-mysqli", key),
		fmt.Sprintf("%s-pdo_mysql", key),
		fmt.Sprintf("%s-pdo_pgsql", key),
		fmt.Sprintf("%s-pgsql", key),
		fmt.Sprintf("%s-zip", key),
		fmt.Sprintf("%s-gd", key),
		fmt.Sprintf("%s-mbstring", key),
	}
	// Di PHP 8.5 ke atas pada FreeBSD, OPcache telah disatukan ke core runtime PHP (built-in)
	// dan port www/php85-opcache dihapus dari FreeBSD ports/pkg.
	if key != "php85" {
		pkgsToInstall = append(pkgsToInstall, fmt.Sprintf("%s-opcache", key))
	}

	args := append([]string{"install", "-y"}, pkgsToInstall...)
	res, err := m.runner.ExecWithTimeout(180*time.Second, "pkg", args...)
	if err != nil {
		// Fallback: Jika batch install gagal karena ada salah satu ekstensi yang tidak tersedia di repo pkg lokal,
		// pasang core PHP terlebih dahulu agar runtime utama tetap terpasang, lalu pasang ekstensi secara best-effort.
		resCore, errCore := m.runner.ExecWithTimeout(120*time.Second, "pkg", "install", "-y", key)
		if errCore != nil {
			return res.Stderr, fmt.Errorf("pkg install gagal untuk %s: %s (%w)", key, res.Stderr, err)
		}
		for _, pkg := range pkgsToInstall[1:] {
			_, _ = m.runner.ExecWithTimeout(60*time.Second, "pkg", "install", "-y", pkg)
		}
		res = resCore
	}

	// 4. Salin binary ke /usr/local/bin/<key> agar versi spesifik selalu tersedia di FreeBSD
	keyBin := fmt.Sprintf("/usr/local/bin/%s", key)
	if binData, readErr := os.ReadFile(phpBin); readErr == nil {
		_ = os.WriteFile(keyBin, binData, 0755)
	}
	fpmBin := "/usr/local/sbin/php-fpm"
	keyFpm := fmt.Sprintf("/usr/local/sbin/%s-fpm", key)
	if fpmData, readErr := os.ReadFile(fpmBin); readErr == nil {
		_ = os.WriteFile(keyFpm, fpmData, 0755)
	}

	// Simpan juga ekstensi ke folder /usr/local/etc/<key>
	keyExtDir := fmt.Sprintf("/usr/local/etc/%s", key)
	_ = os.MkdirAll(keyExtDir, 0755)
	if entries, err := os.ReadDir("/usr/local/etc/php"); err == nil {
		for _, entry := range entries {
			src := filepath.Join("/usr/local/etc/php", entry.Name())
			dst := filepath.Join(keyExtDir, entry.Name())
			if data, err := os.ReadFile(src); err == nil {
				_ = os.WriteFile(dst, data, 0644)
			}
		}
	}

	// Untuk PHP 8.5, pastikan konfigurasi OPcache built-in aktif via opcache.enable=1 (tanpa zend_extension)
	if key == "php85" {
		opcacheContent := "[opcache]\nopcache.enable = 1\nopcache.enable_cli = 1\n"
		_ = os.WriteFile(filepath.Join(keyExtDir, "ext-10-opcache.ini"), []byte(opcacheContent), 0644)
		_ = os.WriteFile("/usr/local/etc/php/ext-10-opcache.ini", []byte(opcacheContent), 0644)
	}

	// 5. Pastikan php.ini tersedia
	iniPath := m.resolveIniPath(key, true)
	if _, err := os.Stat(iniPath); os.IsNotExist(err) {
		_ = os.MkdirAll(filepath.Dir(iniPath), 0755)
		prodSample := "/usr/local/etc/php.ini-production"
		if data, readErr := os.ReadFile(prodSample); readErr == nil {
			_ = os.WriteFile(iniPath, data, 0644)
		} else {
			_ = os.WriteFile(iniPath, []byte("[PHP]\nmemory_limit = 256M\nupload_max_filesize = 64M\npost_max_size = 64M\nmax_execution_time = 60\ndate.timezone = Asia/Jakarta\n"), 0644)
		}
	}

	// 6. Aktifkan di rc.conf
	svcKey := fmt.Sprintf("%s_fpm_enable", key)
	_, _ = m.runner.ExecWithTimeout(10*time.Second, "sysrc", fmt.Sprintf("%s=YES", svcKey))
	_, _ = m.runner.ExecWithTimeout(10*time.Second, "sysrc", "php_fpm_enable=YES")

	// 7. Restart FPM service & aktifkan daemon versi spesifik
	m.EnsureVersionFPMService(key)
	_, _ = m.runner.ExecWithTimeout(10*time.Second, "service", "php_fpm", "restart")
	_, _ = m.runner.ExecWithTimeout(10*time.Second, "service", fmt.Sprintf("%s_fpm", key), "restart")

	return res.Stdout, nil
}

// UninstallVersion menghapus paket PHP FreeBSD via pkg.
func (m *Manager) UninstallVersion(key string) (string, error) {
	// Stop service
	svcName := fmt.Sprintf("%s_fpm", key)
	_, _ = m.runner.ExecWithTimeout(10*time.Second, "service", svcName, "stop")
	_, _ = m.runner.ExecWithTimeout(10*time.Second, "sysrc", fmt.Sprintf("%s_fpm_enable=NO", key))

	// Hapus binary khusus, rc script, konfigurasi FPM, dan folder ekstensi
	_ = os.Remove(fmt.Sprintf("/usr/local/bin/%s", key))
	_ = os.Remove(fmt.Sprintf("/usr/local/sbin/%s-fpm", key))
	_ = os.Remove(fmt.Sprintf("/usr/local/etc/rc.d/%s_fpm", key))
	_ = os.Remove(fmt.Sprintf("/usr/local/etc/%s-fpm.conf", key))
	_ = os.RemoveAll(fmt.Sprintf("/usr/local/etc/%s-fpm.d", key))
	_ = os.RemoveAll(fmt.Sprintf("/usr/local/etc/%s", key))

	// Jika ini versi yang sedang terpasang aktif di sistem FreeBSD, hapus via pkg
	phpBin := "/usr/local/bin/php"
	if curVer := m.getBinaryVersion(phpBin); curVer != "" {
		vMajorMinor := strings.ReplaceAll(curVer[:3], ".", "")
		if fmt.Sprintf("php%s", vMajorMinor) == key {
			res, err := m.runner.ExecWithTimeout(120*time.Second, "pkg", "delete", "-y", key)
			if err != nil {
				return res.Stderr, fmt.Errorf("pkg delete gagal untuk %s: %s (%w)", key, res.Stderr, err)
			}
			return res.Stdout, nil
		}
	}

	return fmt.Sprintf("PHP %s berhasil dicopot dari sistem.", key), nil
}

// GetIniSettings membaca konfigurasi php.ini untuk versi tertentu.
func (m *Manager) GetIniSettings(key string) (*PHPINISettings, error) {
	iniPath := m.resolveIniPath(key, true)
	settings := &PHPINISettings{
		Key:               key,
		IniPath:           iniPath,
		MemoryLimit:       "256M",
		MaxExecutionTime:  "60",
		UploadMaxFilesize: "64M",
		PostMaxSize:       "64M",
		MaxInputTime:      "60",
		Timezone:          "Asia/Jakarta",
		DisplayErrors:     "Off",
		ErrorReporting:    "E_ALL & ~E_NOTICE",
		AllowUrlFopen:     "On",
		ShortOpenTag:      "Off",
	}

	for _, item := range SupportedPHP {
		if item.Key == key {
			settings.Version = item.Version
			break
		}
	}

	file, err := os.Open(iniPath)
	if err != nil {
		// Jika belum ada berkasnya, kembalikan nilai default
		return settings, nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		k := strings.TrimSpace(parts[0])
		v := strings.Trim(strings.TrimSpace(parts[1]), `"' `)

		switch k {
		case "memory_limit":
			settings.MemoryLimit = v
		case "max_execution_time":
			settings.MaxExecutionTime = v
		case "upload_max_filesize":
			settings.UploadMaxFilesize = v
		case "post_max_size":
			settings.PostMaxSize = v
		case "max_input_time":
			settings.MaxInputTime = v
		case "date.timezone":
			settings.Timezone = v
		case "display_errors":
			settings.DisplayErrors = v
		case "error_reporting":
			settings.ErrorReporting = v
		case "allow_url_fopen":
			settings.AllowUrlFopen = v
		case "short_open_tag":
			settings.ShortOpenTag = v
		}
	}

	return settings, nil
}

// SaveIniSettings menyimpan konfigurasi ke dalam php.ini dan mereload FPM.
func (m *Manager) SaveIniSettings(key string, settings *PHPINISettings) error {
	iniPath := m.resolveIniPath(key, true)
	_ = os.MkdirAll(filepath.Dir(iniPath), 0755)

	targetMap := map[string]string{
		"memory_limit":        settings.MemoryLimit,
		"max_execution_time":  settings.MaxExecutionTime,
		"upload_max_filesize": settings.UploadMaxFilesize,
		"post_max_size":       settings.PostMaxSize,
		"max_input_time":      settings.MaxInputTime,
		"date.timezone":       settings.Timezone,
		"display_errors":      settings.DisplayErrors,
		"error_reporting":     settings.ErrorReporting,
		"allow_url_fopen":     settings.AllowUrlFopen,
		"short_open_tag":      settings.ShortOpenTag,
	}

	var lines []string
	foundKeys := make(map[string]bool)

	content, err := os.ReadFile(iniPath)
	if err == nil {
		rawLines := strings.Split(string(content), "\n")
		for _, rawLine := range rawLines {
			trimmed := strings.TrimSpace(rawLine)
			replaced := false

			for k, v := range targetMap {
				// Cek apakah baris mendefinisikan kunci (baik aktif maupun dikomentari)
				re := regexp.MustCompile(fmt.Sprintf(`^;?\s*%s\s*=`, regexp.QuoteMeta(k)))
				if re.MatchString(trimmed) {
					lines = append(lines, fmt.Sprintf("%s = %s", k, v))
					foundKeys[k] = true
					replaced = true
					break
				}
			}

			if !replaced {
				lines = append(lines, rawLine)
			}
		}
	} else {
		lines = append(lines, "[PHP]")
	}

	// Tambahkan kunci yang belum ada di berkas
	for k, v := range targetMap {
		if !foundKeys[k] && v != "" {
			lines = append(lines, fmt.Sprintf("%s = %s", k, v))
		}
	}

	outData := strings.Join(lines, "\n")
	if err := os.WriteFile(iniPath, []byte(outData), 0644); err != nil {
		return fmt.Errorf("gagal menulis php.ini di %s: %w", iniPath, err)
	}

	// Reload PHP-FPM
	m.ReloadFPM(key)
	return nil
}

// GetRawIni membaca seluruh isi php.ini secara mentah.
func (m *Manager) GetRawIni(key string) (string, error) {
	iniPath := m.resolveIniPath(key, true)
	content, err := os.ReadFile(iniPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// SaveRawIni menyimpan teks mentah ke php.ini dan mereload FPM.
func (m *Manager) SaveRawIni(key, content string) error {
	iniPath := m.resolveIniPath(key, true)
	_ = os.MkdirAll(filepath.Dir(iniPath), 0755)
	if err := os.WriteFile(iniPath, []byte(content), 0644); err != nil {
		return err
	}
	m.ReloadFPM(key)
	return nil
}

// GetExtensions memeriksa status setiap ekstensi untuk versi PHP tertentu.
func (m *Manager) GetExtensions(key string) ([]ExtensionInfo, error) {
	dirs := m.resolveExtDirs(key)
	var list []ExtensionInfo

	// Baca seluruh file dari direktori versi spesifik dan direktori default FreeBSD
	filesMap := make(map[string]string) // lowerName -> filePath
	for _, d := range dirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				lower := strings.ToLower(entry.Name())
				if _, exists := filesMap[lower]; !exists {
					filesMap[lower] = filepath.Join(d, entry.Name())
				}
			}
		}
	}

	for _, meta := range CatalogExtensions {
		info := ExtensionInfo{
			Name:        meta.Name,
			DisplayName: meta.DisplayName,
			Description: meta.Description,
			Category:    meta.Category,
			PkgName:     fmt.Sprintf("%s-%s", key, meta.Name),
			Status:      "not_installed",
		}
		if key == "php85" && meta.Name == "opcache" {
			info.PkgName = "php85 (built-in)"
		}

		// Cari file ekstensi, misal `ext-20-pdo.ini`, `ext-30-pdo_pgsql.ini`, atau `ext-20-curl.ini`
		var matchingIniPath string
		var matchingSamplePath string
		var matchingDisabledPath string

		for lowerName, fullPath := range filesMap {
			if strings.Contains(lowerName, "-"+meta.Name+".ini") || lowerName == meta.Name+".ini" {
				matchingIniPath = fullPath
			} else if strings.Contains(lowerName, "-"+meta.Name+".ini.sample") {
				matchingSamplePath = fullPath
			} else if strings.Contains(lowerName, "-"+meta.Name+".ini.disabled") {
				matchingDisabledPath = fullPath
			}
		}

		if matchingIniPath != "" {
			info.IniFile = filepath.Base(matchingIniPath)
			// Baca apakah baris extension di-comment out dengan titik koma (;)
			content, err := os.ReadFile(matchingIniPath)
			if err == nil {
				text := strings.TrimSpace(string(content))
				if text != "" && !strings.HasPrefix(text, ";") {
					if key == "php85" && meta.Name == "opcache" && (strings.Contains(text, "opcache.enable=0") || strings.Contains(text, "opcache.enable = 0")) {
						info.Status = "disabled"
					} else {
						info.Status = "active"
					}
				} else {
					info.Status = "disabled"
				}
			} else {
				info.Status = "active"
			}
		} else if matchingDisabledPath != "" || matchingSamplePath != "" {
			info.Status = "disabled"
			if matchingDisabledPath != "" {
				info.IniFile = filepath.Base(matchingDisabledPath)
			} else {
				info.IniFile = filepath.Base(matchingSamplePath)
			}
		} else {
			// Khusus PHP 8.5: OPcache merupakan modul built-in pada binary core php85 di FreeBSD
			if key == "php85" && meta.Name == "opcache" {
				installed, _, _ := m.checkInstalled(key, "")
				if installed {
					info.Status = "active"
				} else {
					info.Status = "not_installed"
				}
			} else {
				// Cek apakah paket terpasang via pkg info
				res, err := m.runner.ExecWithTimeout(3*time.Second, "pkg", "info", "-e", info.PkgName)
				if err == nil && res.ExitCode == 0 {
					info.Status = "disabled"
				} else {
					info.Status = "not_installed"
				}
			}
		}

		list = append(list, info)
	}

	return list, nil
}

// ToggleExtension mengaktifkan atau menonaktifkan ekstensi PHP dan menyinkronkan direktori konfigurasi.
func (m *Manager) ToggleExtension(key, extName, action string) error {
	// Jika mengaktifkan driver PDO (pdo_pgsql, pdo_mysql, pdo_sqlite), pastikan PDO Core juga aktif
	if (action == "enable" || action == "aktifkan" || action == "install" || action == "pasang") && strings.HasPrefix(extName, "pdo_") {
		_ = m.toggleExtensionInternal(key, "pdo", "enable")
	}

	if err := m.toggleExtensionInternal(key, extName, action); err != nil {
		return err
	}

	// Restart PHP-FPM penuh agar modul .so terbaca oleh master process
	m.RestartFPM(key)
	return nil
}

func (m *Manager) toggleExtensionInternal(key, extName, action string) error {
	dirs := m.resolveExtDirs(key)
	for _, d := range dirs {
		_ = os.MkdirAll(d, 0755)
	}

	// Penamaan file prioritas FreeBSD:
	// ext-10-* untuk opcache
	// ext-20-pdo.ini untuk PDO core (harus dimuat sebelum driver pdo_*)
	// ext-30-pdo_*.ini untuk driver PDO
	// ext-20-* untuk ekstensi reguler lainnya
	iniFileName := fmt.Sprintf("ext-20-%s.ini", extName)
	if extName == "pdo" {
		iniFileName = "ext-20-pdo.ini"
	} else if strings.HasPrefix(extName, "pdo_") {
		iniFileName = fmt.Sprintf("ext-30-%s.ini", extName)
	} else if extName == "opcache" {
		iniFileName = "ext-10-opcache.ini"
	}

	directive := "extension"
	if extName == "opcache" {
		directive = "zend_extension"
	}

	switch action {
	case "enable", "aktifkan":
		if key == "php85" && extName == "opcache" {
			for _, dir := range dirs {
				entries, _ := os.ReadDir(dir)
				for _, entry := range entries {
					if strings.Contains(entry.Name(), "opcache.ini.disabled") {
						src := filepath.Join(dir, entry.Name())
						dest := filepath.Join(dir, strings.TrimSuffix(entry.Name(), ".disabled"))
						_ = os.Rename(src, dest)
					}
				}
				newIni := filepath.Join(dir, "ext-10-opcache.ini")
				content := "[opcache]\nopcache.enable = 1\nopcache.enable_cli = 1\n"
				_ = os.WriteFile(newIni, []byte(content), 0644)
			}
			return nil
		}

		for _, dir := range dirs {
			entries, _ := os.ReadDir(dir)
			var found string
			var foundDisabled string
			var foundSample string

			for _, entry := range entries {
				name := strings.ToLower(entry.Name())
				if strings.Contains(name, "-"+extName+".ini") || name == extName+".ini" {
					found = entry.Name()
				} else if strings.Contains(name, "-"+extName+".ini.disabled") {
					foundDisabled = entry.Name()
				} else if strings.Contains(name, "-"+extName+".ini.sample") {
					foundSample = entry.Name()
				}
			}

			if foundDisabled != "" {
				src := filepath.Join(dir, foundDisabled)
				dest := filepath.Join(dir, strings.TrimSuffix(foundDisabled, ".disabled"))
				_ = os.Rename(src, dest)
			} else if foundSample != "" {
				src := filepath.Join(dir, foundSample)
				dest := filepath.Join(dir, strings.TrimSuffix(foundSample, ".sample"))
				data, err := os.ReadFile(src)
				if err == nil {
					_ = os.WriteFile(dest, data, 0644)
				}
			} else if found != "" {
				iniPath := filepath.Join(dir, found)
				content, err := os.ReadFile(iniPath)
				if err == nil {
					lines := strings.Split(string(content), "\n")
					for i, l := range lines {
						t := strings.TrimSpace(l)
						if strings.HasPrefix(t, ";extension=") || strings.HasPrefix(t, ";zend_extension=") {
							lines[i] = strings.TrimPrefix(t, ";")
						}
					}
					_ = os.WriteFile(iniPath, []byte(strings.Join(lines, "\n")), 0644)
				}
			} else {
				// Buat file baru
				newIni := filepath.Join(dir, iniFileName)
				content := fmt.Sprintf("%s=%s.so\n", directive, extName)
				_ = os.WriteFile(newIni, []byte(content), 0644)
			}
		}

	case "disable", "nonaktifkan":
		if key == "php85" && extName == "opcache" {
			for _, dir := range dirs {
				entries, _ := os.ReadDir(dir)
				for _, entry := range entries {
					name := strings.ToLower(entry.Name())
					if strings.Contains(name, "-opcache.ini") || name == "opcache.ini" {
						src := filepath.Join(dir, entry.Name())
						dest := src + ".disabled"
						_ = os.Rename(src, dest)
					}
				}
				newIni := filepath.Join(dir, "ext-10-opcache.ini")
				content := "[opcache]\nopcache.enable = 0\nopcache.enable_cli = 0\n"
				_ = os.WriteFile(newIni, []byte(content), 0644)
			}
			return nil
		}

		for _, dir := range dirs {
			entries, _ := os.ReadDir(dir)
			for _, entry := range entries {
				name := strings.ToLower(entry.Name())
				if strings.Contains(name, "-"+extName+".ini") || name == extName+".ini" {
					iniPath := filepath.Join(dir, entry.Name())
					content, err := os.ReadFile(iniPath)
					if err == nil {
						lines := strings.Split(string(content), "\n")
						for i, l := range lines {
							t := strings.TrimSpace(l)
							if strings.HasPrefix(t, "extension=") || strings.HasPrefix(t, "zend_extension=") {
								lines[i] = ";" + t
							}
						}
						_ = os.WriteFile(iniPath, []byte(strings.Join(lines, "\n")), 0644)
					}
				}
			}
		}

	case "install", "pasang":
		if key == "php85" && extName == "opcache" {
			// OPcache sudah built-in di php85 FreeBSD (tidak ada port www/php85-opcache)
			for _, dir := range dirs {
				newIni := filepath.Join(dir, "ext-10-opcache.ini")
				content := "[opcache]\nopcache.enable = 1\nopcache.enable_cli = 1\n"
				_ = os.WriteFile(newIni, []byte(content), 0644)
			}
			return nil
		}

		pkgs := []string{fmt.Sprintf("%s-%s", key, extName)}
		if strings.HasPrefix(extName, "pdo_") {
			pkgs = append(pkgs, fmt.Sprintf("%s-pdo", key))
		}
		if extName == "pdo_pgsql" {
			pkgs = append(pkgs, fmt.Sprintf("%s-pgsql", key))
		}

		args := append([]string{"install", "-y"}, pkgs...)
		_, err := m.runner.ExecWithTimeout(120*time.Second, "pkg", args...)
		if err != nil {
			return err
		}

		// Pastikan file .ini tercipta di semua direktori
		for _, dir := range dirs {
			newIni := filepath.Join(dir, iniFileName)
			if _, err := os.Stat(newIni); os.IsNotExist(err) {
				content := fmt.Sprintf("%s=%s.so\n", directive, extName)
				_ = os.WriteFile(newIni, []byte(content), 0644)
			}
		}

		// Jika driver PDO, pastikan ext-20-pdo.ini juga tercipta di semua direktori
		if strings.HasPrefix(extName, "pdo_") {
			for _, dir := range dirs {
				pdoIni := filepath.Join(dir, "ext-20-pdo.ini")
				if _, err := os.Stat(pdoIni); os.IsNotExist(err) {
					_ = os.WriteFile(pdoIni, []byte("extension=pdo.so\n"), 0644)
				}
			}
		}

	case "uninstall", "hapus":
		if key == "php85" && extName == "opcache" {
			// OPcache bagian dari binary inti php85, matikan di berkas konfigurasi
			for _, dir := range dirs {
				entries, _ := os.ReadDir(dir)
				for _, entry := range entries {
					name := strings.ToLower(entry.Name())
					if strings.Contains(name, "-opcache.ini") || name == "opcache.ini" {
						_ = os.Remove(filepath.Join(dir, entry.Name()))
					}
				}
				newIni := filepath.Join(dir, "ext-10-opcache.ini")
				content := "[opcache]\nopcache.enable = 0\nopcache.enable_cli = 0\n"
				_ = os.WriteFile(newIni, []byte(content), 0644)
			}
			return nil
		}

		for _, dir := range dirs {
			entries, _ := os.ReadDir(dir)
			for _, entry := range entries {
				name := strings.ToLower(entry.Name())
				if strings.Contains(name, "-"+extName+".ini") || name == extName+".ini" || strings.Contains(name, "-"+extName+".ini.disabled") {
					_ = os.Remove(filepath.Join(dir, entry.Name()))
				}
			}
		}
		pkgName := fmt.Sprintf("%s-%s", key, extName)
		_, _ = m.runner.ExecWithTimeout(60*time.Second, "pkg", "delete", "-y", pkgName)
	}

	return nil
}

// RestartFPM me-restart service PHP-FPM FreeBSD untuk memuat ulang modul ekstensi .so.
func (m *Manager) RestartFPM(key string) {
	svcName := fmt.Sprintf("%s_fpm", key)
	res, err := m.runner.ExecWithTimeout(10*time.Second, "service", svcName, "restart")
	if err != nil || res.ExitCode != 0 {
		_, _ = m.runner.ExecWithTimeout(10*time.Second, "service", "php_fpm", "restart")
	}
}

// ReloadFPM me-reload service PHP-FPM FreeBSD.
func (m *Manager) ReloadFPM(key string) {
	svcName := fmt.Sprintf("%s_fpm", key)
	res, err := m.runner.ExecWithTimeout(5*time.Second, "service", svcName, "reload")
	if err != nil || res.ExitCode != 0 {
		_, _ = m.runner.ExecWithTimeout(5*time.Second, "service", "php_fpm", "reload")
	}
}
