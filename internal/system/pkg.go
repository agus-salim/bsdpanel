package system

import (
	"fmt"
	"strings"
	"time"
)

// PkgInfo merepresentasikan informasi paket FreeBSD.
type PkgInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Installed   bool   `json:"installed"`
}

// PkgManager menangani operasi paket FreeBSD menggunakan `pkg`.
type PkgManager struct {
	runner Runner
}

func NewPkgManager(runner Runner) *PkgManager {
	return &PkgManager{runner: runner}
}

// IsInstalled memeriksa apakah suatu paket terpasang di FreeBSD (`pkg info <name>`).
func (p *PkgManager) IsInstalled(packageName string) bool {
	res, err := p.runner.ExecWithTimeout(5*time.Second, "pkg", "info", "-e", packageName)
	return err == nil && res.ExitCode == 0
}

// Install memasang paket via `pkg install -y <package>`.
func (p *PkgManager) Install(packageName string) (string, error) {
	res, err := p.runner.ExecWithTimeout(120*time.Second, "pkg", "install", "-y", packageName)
	if err != nil {
		return res.Stderr, fmt.Errorf("pkg install gagal untuk %s: %s (%w)", packageName, res.Stderr, err)
	}
	return res.Stdout, nil
}

// ListInstalledPackageNames mengembalikan daftar paket yang terpasang ringkas.
func (p *PkgManager) ListInstalledPackageNames() ([]string, error) {
	res, err := p.runner.ExecWithTimeout(10*time.Second, "pkg", "query", "%n")
	if err != nil {
		return nil, err
	}
	var pkgs []string
	for _, line := range strings.Split(res.Stdout, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			pkgs = append(pkgs, trimmed)
		}
	}
	return pkgs, nil
}
