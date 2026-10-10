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

// Install installs one or more FreeBSD packages non-interactively.
func (p *PkgManager) Install(ctx context.Context, pkgs ...string) error {
	if len(pkgs) == 0 {
		return nil
	}
	cmd := "/usr/sbin/pkg"
	args := append([]string{"install", "-y"}, pkgs...)
	if p.exec.UseDoas {
		args = append([]string{cmd}, args...)
		cmd = "/usr/local/bin/doas"
	}
	_, err := p.exec.Execute(ctx, cmd, args...)
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
