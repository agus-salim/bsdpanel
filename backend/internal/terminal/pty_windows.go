//go:build windows

package terminal

import (
	"fmt"
	"io"
	"os/exec"
)

type winPty struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

func (w *winPty) Read(p []byte) (n int, err error) {
	return w.stdout.Read(p)
}

func (w *winPty) Write(p []byte) (n int, err error) {
	return w.stdin.Write(p)
}

func (w *winPty) Close() error {
	_ = w.stdin.Close()
	_ = w.stdout.Close()
	if w.cmd.Process != nil {
		return w.cmd.Process.Kill()
	}
	return nil
}

// startPTY untuk development di Windows (menggunakan cmd.exe pipe)
func startPTY(jailName string) (io.ReadWriteCloser, error) {
	cmd := exec.Command("cmd.exe")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("gagal membuat stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("gagal membuat stdout pipe: %w", err)
	}

	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("gagal memulai process terminal di Windows: %w", err)
	}

	return &winPty{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
	}, nil
}

func setPTYSize(ptm io.ReadWriteCloser, cols, rows uint16) {
	// Di Windows dev mode, resize diabaikan
}
