//go:build !windows

package terminal

import (
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// startPTY menjalankan shell interaktif di FreeBSD/Unix atau jexec jail
func startPTY(jailName string) (io.ReadWriteCloser, error) {
	var cmd *exec.Cmd

	if jailName != "" {
		// Jika jail spesifik dipilih, jalankan di dalam FreeBSD Jail via jexec
		cmd = exec.Command("/usr/sbin/jexec", jailName, "/bin/sh")
	} else {
		// Default host shell di FreeBSD
		cmd = exec.Command("/bin/sh")
	}

	cmd.Env = append(cmd.Environ(), "TERM=xterm-256color")

	// Start command dengan Pseudo Terminal
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	return ptmx, nil
}

func setPTYSize(ptm io.ReadWriteCloser, cols, rows uint16) {
	if f, ok := ptm.(*os.File); ok {
		_ = pty.Setsize(f, &pty.Winsize{
			Rows: rows,
			Cols: cols,
		})
	}
}
