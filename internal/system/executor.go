package system

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ExecutionResult captures the output and status of an executed command.
type ExecutionResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

// Executor handles executing system commands on FreeBSD with privilege isolation.
type Executor struct {
	UseDoas     bool
	DefaultWait time.Duration
}

// NewExecutor creates a new safe execution helper.
func NewExecutor(useDoas bool) *Executor {
	return &Executor{
		UseDoas:     useDoas,
		DefaultWait: 30 * time.Second,
	}
}

// Execute runs a command with default system privileges in a safe, non-interpolated manner.
func (e *Executor) Execute(ctx context.Context, command string, args ...string) (*ExecutionResult, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), e.DefaultWait)
		defer cancel()
	}

	start := time.Now()
	cmd := exec.CommandContext(ctx, command, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	res := &ExecutionResult{
		Stdout:   strings.TrimSpace(stdout.String()),
		Stderr:   strings.TrimSpace(stderr.String()),
		ExitCode: exitCode,
		Duration: duration,
	}

	if err != nil {
		return res, fmt.Errorf("command '%s' failed (exit %d): %s (stderr: %s)", command, exitCode, err, res.Stderr)
	}

	return res, nil
}

// ExecuteAsUser executes a command strictly isolated under a specific system username.
// This prevents tasks (like site scripts, PHP workers, git pulls) from having root or panel daemon privileges.
func (e *Executor) ExecuteAsUser(ctx context.Context, targetUser string, command string, args ...string) (*ExecutionResult, error) {
	if targetUser == "" || targetUser == "root" {
		return nil, fmt.Errorf("isolated execution must target an unprivileged user, got '%s'", targetUser)
	}

	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), e.DefaultWait)
		defer cancel()
	}

	// Lookup system user on FreeBSD (maps to /etc/passwd or master.passwd)
	u, err := user.Lookup(targetUser)
	if err != nil {
		return nil, fmt.Errorf("target user '%s' not found: %w", targetUser, err)
	}

	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid UID for user %s: %w", targetUser, err)
	}
	gid, err := strconv.ParseUint(u.Gid, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid GID for user %s: %w", targetUser, err)
	}

	var cmd *exec.Cmd

	// If using doas (FreeBSD's lightweight sudo alternative)
	if e.UseDoas {
		doasArgs := append([]string{"-u", targetUser, command}, args...)
		cmd = exec.CommandContext(ctx, "/usr/local/bin/doas", doasArgs...)
	} else {
		// Native POSIX privilege dropping via Credential
		cmd = exec.CommandContext(ctx, command, args...)
		cmd.Dir = u.HomeDir
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{
				Uid: uint32(uid),
				Gid: uint32(gid),
			},
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err = cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	res := &ExecutionResult{
		Stdout:   strings.TrimSpace(stdout.String()),
		Stderr:   strings.TrimSpace(stderr.String()),
		ExitCode: exitCode,
		Duration: duration,
	}

	if err != nil {
		return res, fmt.Errorf("isolated execution as '%s' failed: %s (stderr: %s)", targetUser, err, res.Stderr)
	}

	return res, nil
}
