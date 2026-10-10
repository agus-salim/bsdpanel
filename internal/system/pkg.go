package system

import (
	"context"
	"fmt"
	"strings"
)

// PkgManager handles FreeBSD package installation and querying using 'pkg'.
type PkgManager struct {
	exec *Executor
}

// NewPkgManager initializes a pkg manager.
func NewPkgManager(exec *Executor) *PkgManager {
	return &PkgManager{exec: exec}
}

// IsInstalled checks if a FreeBSD package is already present on the system.
func (p *PkgManager) IsInstalled(ctx context.Context, pkgName string) bool {
	res, err := p.exec.Execute(ctx, "/usr/sbin/pkg", "info", "-e", pkgName)
	return err == nil && res.ExitCode == 0
}

// FilterAvailable checks the remote FreeBSD package repository and returns which packages exist.
func (p *PkgManager) FilterAvailable(ctx context.Context, pkgs ...string) ([]string, error) {
	if len(pkgs) == 0 {
		return nil, nil
	}
	cmd := "/usr/sbin/pkg"
	args := append([]string{"rquery", "%n"}, pkgs...)
	if p.exec.UseDoas {
		args = append([]string{cmd}, args...)
		cmd = "/usr/local/bin/doas"
	}
	res, err := p.exec.Execute(ctx, cmd, args...)
	if err != nil && (res == nil || res.Stdout == "") {
		return pkgs, nil // If rquery fails, fallback to original list
	}

	foundMap := make(map[string]bool)
	for _, line := range strings.Split(res.Stdout, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			foundMap[trimmed] = true
		}
	}

	var available []string
	for _, pkg := range pkgs {
		if foundMap[pkg] {
			available = append(available, pkg)
		}
	}
	return available, nil
}

// Install installs one or more FreeBSD packages non-interactively with repository validation.
func (p *PkgManager) Install(ctx context.Context, pkgs ...string) error {
	if len(pkgs) == 0 {
		return nil
	}

	// Filter packages that actually exist in the repository
	available, err := p.FilterAvailable(ctx, pkgs...)
	if err == nil && len(available) > 0 {
		// If the primary requested package is NOT available at all
		primaryPkg := pkgs[0]
		primaryFound := false
		for _, a := range available {
			if a == primaryPkg {
				primaryFound = true
				break
			}
		}

		if !primaryFound {
			return fmt.Errorf("paket '%s' tidak ditemukan di repository resmi FreeBSD pkg. Paket ini belum dirilis secara resmi untuk FreeBSD. Silakan gunakan versi stabil yang tersedia (seperti PHP 8.3 atau PHP 8.4).", primaryPkg)
		}

		// Use the available packages so missing optional extensions don't break install
		pkgs = available
	} else if err == nil && len(available) == 0 {
		return fmt.Errorf("paket '%s' tidak ditemukan di repository FreeBSD pkg. Versi ini belum tersedia. Silakan gunakan versi paket yang tersedia (seperti PHP 8.3 atau PHP 8.4).", pkgs[0])
	}

	cmd := "/usr/sbin/pkg"
	args := append([]string{"install", "-y"}, pkgs...)
	if p.exec.UseDoas {
		args = append([]string{cmd}, args...)
		cmd = "/usr/local/bin/doas"
	}
	_, err = p.exec.Execute(ctx, cmd, args...)
	return err
}

// Uninstall removes a package from FreeBSD.
func (p *PkgManager) Uninstall(ctx context.Context, pkgName string) error {
	cmd := "/usr/sbin/pkg"
	args := []string{"delete", "-y", pkgName}
	if p.exec.UseDoas {
		args = append([]string{cmd}, args...)
		cmd = "/usr/local/bin/doas"
	}
	_, err := p.exec.Execute(ctx, cmd, args...)
	return err
}

// GetPHPPkgs returns the canonical FreeBSD package list for a specified PHP version.
func (p *PkgManager) GetPHPPkgs(version string) []string {
	// version e.g., "8.2", "8.3", "8.4", "8.5"
	tag := strings.ReplaceAll(version, ".", "")
	return []string{
		fmt.Sprintf("php%s", tag),
		fmt.Sprintf("php%s-extensions", tag),
		fmt.Sprintf("php%s-mysqli", tag),
		fmt.Sprintf("php%s-pgsql", tag),
		fmt.Sprintf("php%s-curl", tag),
		fmt.Sprintf("php%s-mbstring", tag),
		fmt.Sprintf("php%s-zip", tag),
		fmt.Sprintf("php%s-gd", tag),
		fmt.Sprintf("php%s-bcmath", tag),
		fmt.Sprintf("php%s-intl", tag),
		fmt.Sprintf("php%s-openssl", tag),
		fmt.Sprintf("php%s-pdo_mysql", tag),
		fmt.Sprintf("php%s-pdo_pgsql", tag),
	}
}

// GetWebServerPkgs returns FreeBSD pkg names for chosen web server.
func (p *PkgManager) GetWebServerPkgs(webServer string) []string {
	switch strings.ToLower(webServer) {
	case "nginx":
		return []string{"nginx"}
	case "apache":
		return []string{"apache24"}
	case "caddy":
		return []string{"caddy"}
	case "litespeed", "openlitespeed":
		return []string{"openlitespeed"}
	default:
		return []string{"nginx"}
	}
}

// GetDatabasePkgs returns FreeBSD pkg names for database software.
func (p *PkgManager) GetDatabasePkgs(dbType string) []string {
	switch strings.ToLower(dbType) {
	case "mariadb":
		return []string{"mariadb1011-server", "mariadb1011-client"}
	case "postgresql":
		return []string{"postgresql16-server", "postgresql16-client"}
	case "phpmyadmin":
		return []string{"phpMyAdmin5-php83"}
	case "phppgadmin":
		return []string{"phppgadmin-php83"}
	default:
		return nil
	}
}
