package system

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// UserManager handles FreeBSD user account operations using native 'pw' utilities.
type UserManager struct {
	exec *Executor
}

// NewUserManager initializes a user manager.
func NewUserManager(exec *Executor) *UserManager {
	return &UserManager{exec: exec}
}

// CreateSiteUser provisions an isolated FreeBSD user for a web property.
// Standard FreeBSD path is /usr/home/<username> with shell set to /usr/sbin/nologin for security.
func (u *UserManager) CreateSiteUser(ctx context.Context, username, homedir string) error {
	if username == "" || strings.ContainsAny(username, " /\\;$&|`'\"") {
		return fmt.Errorf("invalid or unsafe username: '%s'", username)
	}

	if homedir == "" {
		homedir = filepath.Join("/usr/home", username)
	}

	// FreeBSD native command: pw useradd <username> -m -d <homedir> -s /usr/sbin/nologin -c "BSD Panel Site User"
	cmd := "/usr/sbin/pw"
	args := []string{
		"useradd", username,
		"-m",
		"-d", homedir,
		"-s", "/usr/sbin/nologin",
		"-c", fmt.Sprintf("BSDPanel Managed User (%s)", username),
	}

	if u.exec.UseDoas {
		args = append([]string{cmd}, args...)
		cmd = "/usr/local/bin/doas"
	}

	_, err := u.exec.Execute(ctx, cmd, args...)
	if err != nil {
		return fmt.Errorf("failed to create FreeBSD user '%s': %w", username, err)
	}

	// Restrict home directory permissions so other site users cannot inspect it
	// chmod 0750 /usr/home/<username>
	chmodCmd := "/bin/chmod"
	chmodArgs := []string{"0750", homedir}
	if u.exec.UseDoas {
		chmodArgs = append([]string{chmodCmd}, chmodArgs...)
		chmodCmd = "/usr/local/bin/doas"
	}
	_, _ = u.exec.Execute(ctx, chmodCmd, chmodArgs...)

	// Create standard web folder structure: public_html, logs, ssl, tmp
	subdirs := []string{"public_html", "logs", "ssl", "tmp"}
	for _, sub := range subdirs {
		dirPath := filepath.Join(homedir, sub)
		mkdirCmd := "/bin/mkdir"
		mkdirArgs := []string{"-p", dirPath}
		if u.exec.UseDoas {
			mkdirArgs = append([]string{mkdirCmd}, mkdirArgs...)
			mkdirCmd = "/usr/local/bin/doas"
		}
		_, _ = u.exec.Execute(ctx, mkdirCmd, mkdirArgs...)

		// Chown to the created user
		chownCmd := "/usr/sbin/chown"
		chownArgs := []string{"-R", fmt.Sprintf("%s:%s", username, username), dirPath}
		if u.exec.UseDoas {
			chownArgs = append([]string{chownCmd}, chownArgs...)
			chownCmd = "/usr/local/bin/doas"
		}
		_, _ = u.exec.Execute(ctx, chownCmd, chownArgs...)
	}

	return nil
}

// DeleteSiteUser removes a FreeBSD user and optionally removes their home directory.
func (u *UserManager) DeleteSiteUser(ctx context.Context, username string, removeHome bool) error {
	if username == "" || strings.ContainsAny(username, " /\\;$&|`'\"") {
		return fmt.Errorf("invalid username")
	}

	// pw userdel <username> -r
	args := []string{"userdel", username}
	if removeHome {
		args = append(args, "-r")
	}

	cmd := "/usr/sbin/pw"
	if u.exec.UseDoas {
		args = append([]string{cmd}, args...)
		cmd = "/usr/local/bin/doas"
	}

	_, err := u.exec.Execute(ctx, cmd, args...)
	if err != nil {
		return fmt.Errorf("failed to delete FreeBSD user '%s': %w", username, err)
	}
	return nil
}

// UserExists checks if the specified user exists on the system.
func (u *UserManager) UserExists(username string) bool {
	if _, err := os.Stat(filepath.Join("/usr/home", username)); err == nil {
		return true
	}
	return false
}
