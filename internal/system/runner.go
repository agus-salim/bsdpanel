package system

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// CommandResult menyimpan output dari perintah sistem.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

// Runner mendefinisikan antarmuka eksekusi perintah sistem.
type Runner interface {
	Exec(ctx context.Context, name string, args ...string) (*CommandResult, error)
	ExecWithTimeout(timeout time.Duration, name string, args ...string) (*CommandResult, error)
}

// DefaultRunner mengimplementasikan Runner menggunakan os/exec.
type DefaultRunner struct {
	IsFreeBSD bool
	DryRun    bool
}

// NewRunner membuat instance runner baru yang mendeteksi platform FreeBSD.
func NewRunner(dryRun bool) *DefaultRunner {
	return &DefaultRunner{
		IsFreeBSD: runtime.GOOS == "freebsd",
		DryRun:    dryRun,
	}
}

// resolveBinary mencari lokasi absolut executable di FreeBSD (/usr/local/sbin, /usr/local/bin, dll).
func resolveBinary(name string) string {
	if filepath.IsAbs(name) {
		return name
	}

	// Prioritas direktori FreeBSD ports & standard binaries
	searchDirs := []string{
		"/usr/local/sbin",
		"/usr/local/bin",
		"/usr/sbin",
		"/usr/bin",
		"/sbin",
		"/bin",
	}

	for _, dir := range searchDirs {
		target := filepath.Join(dir, name)
		if info, err := os.Stat(target); err == nil && !info.IsDir() {
			return target
		}
	}

	if p, err := exec.LookPath(name); err == nil {
		return p
	}

	return name
}

// Exec menjalankan perintah dengan context dan memotong buffer output secara aman.
func (r *DefaultRunner) Exec(ctx context.Context, name string, args ...string) (*CommandResult, error) {
	start := time.Now()

	// Jika mode dry-run atau bukan FreeBSD (misal pembangunan lokal di Windows/Linux),
	// kita lakukan simulasi respon agar panel tetap dapat diuji.
	if r.DryRun || !r.IsFreeBSD {
		log.Printf("[MOCK/DRY-RUN] Exec: %s %s", name, strings.Join(args, " "))
		return r.mockResponse(name, args, time.Since(start))
	}

	binPath := resolveBinary(name)
	cmd := exec.CommandContext(ctx, binPath, args...)

	// Pastikan PATH mencakup /usr/local/sbin dan /usr/local/bin di FreeBSD
	freebsdPATH := "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	if cur := os.Getenv("PATH"); cur != "" {
		freebsdPATH = freebsdPATH + ":" + cur
	}
	cmd.Env = append(os.Environ(), "PATH="+freebsdPATH)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	duration := time.Since(start)

	res := &CommandResult{
		Stdout:   strings.TrimSpace(stdoutBuf.String()),
		Stderr:   strings.TrimSpace(stderrBuf.String()),
		Duration: duration,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			res.ExitCode = exitErr.ExitCode()
		} else {
			res.ExitCode = -1
		}
		return res, fmt.Errorf("command failed: %s %v (exit %d): %s: %w", name, args, res.ExitCode, res.Stderr, err)
	}

	res.ExitCode = 0
	return res, nil
}

// ExecWithTimeout adalah pembungkus ringkas dengan batas waktu eksekusi otomatis.
func (r *DefaultRunner) ExecWithTimeout(timeout time.Duration, name string, args ...string) (*CommandResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return r.Exec(ctx, name, args...)
}

// mockResponse menyediakan simulasi keluaran perintah FreeBSD untuk pengujian lokal di Windows/macOS.
func (r *DefaultRunner) mockResponse(name string, args []string, duration time.Duration) (*CommandResult, error) {
	argStr := strings.Join(args, " ")

	switch name {
	case "service":
		if strings.Contains(argStr, "status") {
			return &CommandResult{
				Stdout:   fmt.Sprintf("%s is running as pid 1337.", args[0]),
				ExitCode: 0,
				Duration: duration,
			}, nil
		}
		return &CommandResult{
			Stdout:   fmt.Sprintf("Performing %s for %s: done", args[1], args[0]),
			ExitCode: 0,
			Duration: duration,
		}, nil

	case "sysctl":
		if strings.Contains(argStr, "kern.cp_time") {
			return &CommandResult{
				Stdout:   "125674 345 54120 1200 8945210",
				ExitCode: 0,
				Duration: duration,
			}, nil
		}
		if strings.Contains(argStr, "hw.physmem") {
			return &CommandResult{
				Stdout:   "8589934592", // 8 GB RAM
				ExitCode: 0,
				Duration: duration,
			}, nil
		}
		if strings.Contains(argStr, "vm.stats.vm.v_free_count") {
			return &CommandResult{
				Stdout:   "524288",
				ExitCode: 0,
				Duration: duration,
			}, nil
		}
		return &CommandResult{Stdout: "0", ExitCode: 0, Duration: duration}, nil

	case "zfs", "df":
		return &CommandResult{
			Stdout:   "zroot/ROOT/default  100G   24G   76G    24%    /",
			ExitCode: 0,
			Duration: duration,
		}, nil

	case "sockstat":
		return &CommandResult{
			Stdout:   "USER     COMMAND    PID   FD PROTO  LOCAL ADDRESS         FOREIGN ADDRESS\nwww      nginx      1024  7  tcp4   *:80                  *:*\nwww      nginx      1024  8  tcp4   *:443                 *:*\nmysql    mariadbd   1040  10 tcp4   127.0.0.1:3306        *:*",
			ExitCode: 0,
			Duration: duration,
		}, nil

	case "pkg":
		return &CommandResult{
			Stdout:   "nginx-1.26.1\nphp83-8.3.8\nmariadb1011-server-10.11.8",
			ExitCode: 0,
			Duration: duration,
		}, nil

	default:
		return &CommandResult{
			Stdout:   "OK (Simulated)",
			ExitCode: 0,
			Duration: duration,
		}, nil
	}
}
