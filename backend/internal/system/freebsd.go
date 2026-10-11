package system

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// SystemStats menyimpan metrik spesifik FreeBSD
type SystemStats struct {
	OS            string  `json:"os"`
	KernelRelease string  `json:"kernel_release"`
	Hostname      string  `json:"hostname"`
	LoadAvg1      float64 `json:"load_avg_1"`
	LoadAvg5      float64 `json:"load_avg_5"`
	LoadAvg15     float64 `json:"load_avg_15"`
	MemoryTotalMB uint64  `json:"memory_total_mb"`
	MemoryUsedMB  uint64  `json:"memory_used_mb"`
	ZfsArcSizeMB  uint64  `json:"zfs_arc_size_mb"`
	ActiveJails   int     `json:"active_jails"`
	IsNativeBSD   bool    `json:"is_native_bsd"`
}

// ServiceStatus status daemon FreeBSD
type ServiceStatus struct {
	Name    string `json:"name"`
	Running bool   `json:"running"`
	Pid     int    `json:"pid,omitempty"`
}

// FreeBSDManager interface pembungkus perintah sistem FreeBSD
type FreeBSDManager struct {
	zfsPool string
}

func NewFreeBSDManager(zfsPool string) *FreeBSDManager {
	return &FreeBSDManager{
		zfsPool: zfsPool,
	}
}

// RunCommand menjalankan command dengan timeout dan sanitasi
func (m *FreeBSDManager) RunCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("perintah '%s %s' gagal: %w (stderr: %s)", name, strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

// ServiceControl memanipulasi rc.d FreeBSD: start, stop, restart, reload, status
func (m *FreeBSDManager) ServiceControl(ctx context.Context, serviceName string, action string) (string, error) {
	if runtime.GOOS != "freebsd" {
		return fmt.Sprintf("[Simulasi %s] service %s %s berhasil", runtime.GOOS, serviceName, action), nil
	}
	// Di FreeBSD: /usr/sbin/service <name> <action>
	return m.RunCommand(ctx, "/usr/sbin/service", serviceName, action)
}

// CheckService memeriksa apakah daemon sedang berjalan via 'service <name> onestatus'
func (m *FreeBSDManager) CheckService(ctx context.Context, serviceName string) bool {
	if runtime.GOOS != "freebsd" {
		return true // Mock aktif saat development
	}
	out, err := m.RunCommand(ctx, "/usr/sbin/service", serviceName, "onestatus")
	if err != nil {
		return false
	}
	return strings.Contains(out, "is running as pid")
}

// ZfsCreateDataset membuat dataset baru dengan quota
func (m *FreeBSDManager) ZfsCreateDataset(ctx context.Context, datasetPath string, quota string) error {
	if runtime.GOOS != "freebsd" {
		return nil
	}
	// Contoh: zfs create -o mountpoint=/usr/home/user1/site1 zroot/bsdpanel/user1/site1
	_, err := m.RunCommand(ctx, "/sbin/zfs", "create", datasetPath)
	if err != nil {
		return err
	}
	if quota != "" {
		_, err = m.RunCommand(ctx, "/sbin/zfs", "set", fmt.Sprintf("quota=%s", quota), datasetPath)
	}
	return err
}

// ZfsCreateSnapshot membuat ZFS snapshot untuk backup instan
func (m *FreeBSDManager) ZfsCreateSnapshot(ctx context.Context, datasetPath string, snapName string) error {
	if runtime.GOOS != "freebsd" {
		return nil
	}
	fullSnap := fmt.Sprintf("%s@%s", datasetPath, snapName)
	_, err := m.RunCommand(ctx, "/sbin/zfs", "snapshot", fullSnap)
	return err
}

// ZfsRollback mengembalikan dataset ke snapshot tertentu
func (m *FreeBSDManager) ZfsRollback(ctx context.Context, datasetPath string, snapName string) error {
	if runtime.GOOS != "freebsd" {
		return nil
	}
	fullSnap := fmt.Sprintf("%s@%s", datasetPath, snapName)
	_, err := m.RunCommand(ctx, "/sbin/zfs", "rollback", "-r", fullSnap)
	return err
}

// JailStart menyalakan jail FreeBSD
func (m *FreeBSDManager) JailStart(ctx context.Context, jailName string) error {
	if runtime.GOOS != "freebsd" {
		return nil
	}
	_, err := m.RunCommand(ctx, "/usr/sbin/jail", "-c", jailName)
	return err
}

// JailStop mematikan jail FreeBSD
func (m *FreeBSDManager) JailStop(ctx context.Context, jailName string) error {
	if runtime.GOOS != "freebsd" {
		return nil
	}
	_, err := m.RunCommand(ctx, "/usr/sbin/jail", "-r", jailName)
	return err
}

// GetSystemStats membaca metrik sistem FreeBSD (Sysctl & ZFS ARC)
func (m *FreeBSDManager) GetSystemStats(ctx context.Context) (*SystemStats, error) {
	stats := &SystemStats{
		OS:          runtime.GOOS,
		IsNativeBSD: runtime.GOOS == "freebsd",
	}

	if runtime.GOOS != "freebsd" {
		// Mock data untuk pengujian lokal di Windows/macOS/Linux
		stats.KernelRelease = "FreeBSD 14.1-RELEASE-p2 GENERIC (Dev Mode)"
		stats.Hostname = "freebsd-panel.local"
		stats.LoadAvg1 = 0.24
		stats.LoadAvg5 = 0.38
		stats.LoadAvg15 = 0.15
		stats.MemoryTotalMB = 16384
		stats.MemoryUsedMB = 4820
		stats.ZfsArcSizeMB = 2540
		stats.ActiveJails = 3
		return stats, nil
	}

	// Membaca Kernel & Hostname di FreeBSD
	if out, err := m.RunCommand(ctx, "/sbin/sysctl", "-n", "kern.version"); err == nil {
		lines := strings.Split(out, "\n")
		if len(lines) > 0 {
			stats.KernelRelease = lines[0]
		}
	}
	if out, err := m.RunCommand(ctx, "/bin/hostname"); err == nil {
		stats.Hostname = out
	}

	// Membaca Memory Stats via FreeBSD sysctl
	// pageSize = hw.pagesize, totalPages = vm.stats.vm.v_page_count, freePages = vm.stats.vm.v_free_count
	var pageSize, totalPages, freePages uint64
	_, _ = fmt.Sscanf(m.sysctlValue(ctx, "hw.pagesize"), "%d", &pageSize)
	_, _ = fmt.Sscanf(m.sysctlValue(ctx, "vm.stats.vm.v_page_count"), "%d", &totalPages)
	_, _ = fmt.Sscanf(m.sysctlValue(ctx, "vm.stats.vm.v_free_count"), "%d", &freePages)

	if pageSize > 0 && totalPages > 0 {
		stats.MemoryTotalMB = (totalPages * pageSize) / (1024 * 1024)
		stats.MemoryUsedMB = ((totalPages - freePages) * pageSize) / (1024 * 1024)
	}

	// ZFS ARC Cache Size
	var arcBytes uint64
	_, _ = fmt.Sscanf(m.sysctlValue(ctx, "kstat.zfs.misc.arcstats.size"), "%d", &arcBytes)
	stats.ZfsArcSizeMB = arcBytes / (1024 * 1024)

	// Jumlah Jails yang Aktif via jls
	if out, err := m.RunCommand(ctx, "/usr/sbin/jls", "-q"); err == nil {
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) == 1 && lines[0] == "" {
			stats.ActiveJails = 0
		} else {
			stats.ActiveJails = len(lines)
		}
	}

	return stats, nil
}

func (m *FreeBSDManager) sysctlValue(ctx context.Context, key string) string {
	ctxTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	val, err := m.RunCommand(ctxTimeout, "/sbin/sysctl", "-n", key)
	if err != nil {
		return "0"
	}
	return val
}
