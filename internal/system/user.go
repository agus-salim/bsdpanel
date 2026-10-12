package system

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// UserManager mengelola akun UNIX FreeBSD untuk isolasi tingkat proses dan file.
type UserManager struct {
	runner Runner
}

func NewUserManager(runner Runner) *UserManager {
	return &UserManager{runner: runner}
}

// CreateUser membuat user sistem FreeBSD dengan direktori home terisolasi.
// Perintah: `pw useradd <username> -m -d /home/<username> -s /usr/sbin/nologin -G www`
func (u *UserManager) CreateUser(username string) error {
	if username == "" || username == "root" || username == "wheel" || username == "www" {
		return fmt.Errorf("nama pengguna tidak diizinkan atau terlarang: %s", username)
	}

	homeDir := filepath.Join("/home", username)
	args := []string{
		"useradd", username,
		"-m",
		"-d", homeDir,
		"-s", "/usr/sbin/nologin",
		"-c", fmt.Sprintf("bsdpanel account %s", username),
		"-G", "www", // User masuk ke group www agar webserver dapat mengakses socket
	}

	res, err := u.runner.ExecWithTimeout(10*time.Second, "pw", args...)
	if err != nil {
		return fmt.Errorf("gagal membuat user FreeBSD via pw: %s (%w)", res.Stderr, err)
	}

	// Buat struktur direktori web publik standar
	publicHTML := filepath.Join(homeDir, "public_html")
	logsDir := filepath.Join(homeDir, "logs")

	// Abaikan error pembuatan dir jika running di lingkungan tes lokal
	_ = os.MkdirAll(publicHTML, 0750)
	_ = os.MkdirAll(logsDir, 0750)

	// Ubah kepemilikan file di FreeBSD
	_, _ = u.runner.ExecWithTimeout(5*time.Second, "chown", "-R", fmt.Sprintf("%s:www", username), homeDir)
	_, _ = u.runner.ExecWithTimeout(5*time.Second, "chmod", "750", homeDir)

	return nil
}

// DeleteUser menghapus user sistem FreeBSD beserta direktori home-nya.
func (u *UserManager) DeleteUser(username string) error {
	if username == "" || username == "root" || username == "wheel" || username == "www" {
		return fmt.Errorf("tidak dapat menghapus user sistem esensial")
	}

	res, err := u.runner.ExecWithTimeout(10*time.Second, "pw", "userdel", username, "-r")
	if err != nil {
		return fmt.Errorf("gagal menghapus user %s: %s (%w)", username, res.Stderr, err)
	}
	return nil
}
