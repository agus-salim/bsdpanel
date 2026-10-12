package system

import (
	"fmt"
	"strings"
	"time"
)

// ServiceStatus merepresentasikan kondisi daemon di FreeBSD.
type ServiceStatus struct {
	Name      string `json:"name"`
	Label     string `json:"label"`
	Running   bool   `json:"running"`
	Pid       string `json:"pid,omitempty"`
	Output    string `json:"output"`
	Autostart bool   `json:"autostart"`
}

// ServiceManager menangani kontrol daemon FreeBSD via command `service` dan `sysrc`.
type ServiceManager struct {
	runner Runner
}

func NewServiceManager(runner Runner) *ServiceManager {
	return &ServiceManager{runner: runner}
}

// Status memeriksa status daemon via `service <name> status`.
func (s *ServiceManager) Status(name string) (*ServiceStatus, error) {
	res, _ := s.runner.ExecWithTimeout(5*time.Second, "service", name, "status")

	st := &ServiceStatus{
		Name:    name,
		Label:   getFriendlyLabel(name),
		Output:  res.Stdout,
		Running: false,
	}

	// Output khas FreeBSD: "nginx is running as pid 1234."
	if strings.Contains(res.Stdout, "is running as pid") {
		st.Running = true
		parts := strings.Split(res.Stdout, "pid")
		if len(parts) > 1 {
			st.Pid = strings.Trim(strings.TrimSpace(parts[1]), ".")
		}
	}

	// Cek apakah aktif di rc.conf via `sysrc -n <name>_enable`
	sysrcRes, err := s.runner.ExecWithTimeout(3*time.Second, "sysrc", "-n", fmt.Sprintf("%s_enable", name))
	if err == nil && strings.EqualFold(strings.TrimSpace(sysrcRes.Stdout), "YES") {
		st.Autostart = true
	}

	return st, nil
}

// Action menjalankan perintah start, stop, restart, atau reload.
func (s *ServiceManager) Action(name, action string) (string, error) {
	validActions := map[string]bool{
		"start":   true,
		"stop":    true,
		"restart": true,
		"reload":  true,
		"graceful": true,
	}

	if !validActions[action] {
		return "", fmt.Errorf("aksi layanan tidak valid: %s", action)
	}

	res, err := s.runner.ExecWithTimeout(15*time.Second, "service", name, action)
	if err != nil {
		return res.Stderr, err
	}
	return res.Stdout, nil
}

// SetAutostart mengaktifkan atau menonaktifkan daemon saat boot FreeBSD via sysrc.
func (s *ServiceManager) SetAutostart(name string, enable bool) error {
	val := "NO"
	if enable {
		val = "YES"
	}
	_, err := s.runner.ExecWithTimeout(5*time.Second, "sysrc", fmt.Sprintf("%s_enable=%s", name, val))
	return err
}

// GetAllManagedServices mengambil status layanan penting web hosting di FreeBSD.
func (s *ServiceManager) GetAllManagedServices() ([]ServiceStatus, error) {
	services := []string{"nginx", "apache24", "php_fpm", "mysql-server", "postgresql", "pf"}
	var results []ServiceStatus

	for _, svc := range services {
		st, err := s.Status(svc)
		if err == nil {
			results = append(results, *st)
		}
	}
	return results, nil
}

func getFriendlyLabel(name string) string {
	switch name {
	case "nginx":
		return "Nginx Web Server"
	case "apache24":
		return "Apache HTTP Server"
	case "php_fpm":
		return "PHP-FPM Process Manager"
	case "mysql-server":
		return "MariaDB / MySQL Server"
	case "postgresql":
		return "PostgreSQL Database"
	case "pf":
		return "PF (Packet Filter Firewall)"
	default:
		return strings.Title(name)
	}
}
