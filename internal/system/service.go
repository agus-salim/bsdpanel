package system

import (
	"fmt"
	"os"
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
	Installed bool   `json:"installed"`
}

// ServiceManager menangani kontrol daemon FreeBSD via command `service` dan `sysrc`.
type ServiceManager struct {
	runner Runner
}

func NewServiceManager(runner Runner) *ServiceManager {
	return &ServiceManager{runner: runner}
}

// getRcvar memetakan nama layanan ke variabel rcvar FreeBSD yang sesuai di /etc/rc.conf.
func getRcvar(name string) string {
	switch name {
	case "mysql-server":
		return "mysql"
	default:
		return name
	}
}

// Status memeriksa status daemon via `service <name> status`.
func (s *ServiceManager) Status(name string) (*ServiceStatus, error) {
	res, _ := s.runner.ExecWithTimeout(5*time.Second, "service", name, "status")

	out := ""
	if res != nil {
		out = res.Stdout
		if out == "" {
			out = res.Stderr
		}
	}

	st := &ServiceStatus{
		Name:      name,
		Label:     getFriendlyLabel(name),
		Output:    out,
		Running:   false,
		Installed: true,
	}

	// Deteksi apakah layanan terpasang di FreeBSD rc.d
	if strings.Contains(out, "does not exist in /etc/rc.d") || strings.Contains(out, "not found") {
		st.Installed = false
	}

	lowerOut := strings.ToLower(out)

	// 1. Output standar FreeBSD: "<daemon> is running as pid 1234."
	if strings.Contains(lowerOut, "is running as pid") {
		st.Running = true
		parts := strings.Split(lowerOut, "pid")
		if len(parts) > 1 {
			st.Pid = strings.Trim(strings.TrimSpace(parts[1]), ".")
		}
	} else if strings.Contains(lowerOut, "server is running (pid:") {
		// 2. Output khas PostgreSQL di FreeBSD: "pg_ctl: server is running (PID: 1234)"
		st.Running = true
		parts := strings.Split(lowerOut, "pid:")
		if len(parts) > 1 {
			st.Pid = strings.Trim(strings.Trim(strings.TrimSpace(parts[1]), ")"), ".")
		}
	} else if strings.Contains(lowerOut, "status: enabled") {
		// 3. Output khas PF (Packet Filter) di FreeBSD: "Status: Enabled for ..."
		st.Running = true
	}

	// Cek apakah aktif di rc.conf via `sysrc -n <rcvar>_enable`
	rcvar := getRcvar(name)
	sysrcRes, err := s.runner.ExecWithTimeout(3*time.Second, "sysrc", "-n", fmt.Sprintf("%s_enable", rcvar))
	if err == nil && strings.EqualFold(strings.TrimSpace(sysrcRes.Stdout), "YES") {
		st.Autostart = true
	}

	return st, nil
}

// Action menjalankan perintah start, stop, restart, reload, graceful, install, atau toggle_autostart.
func (s *ServiceManager) Action(name, action string) (string, error) {
	validActions := map[string]bool{
		"start":            true,
		"stop":             true,
		"restart":          true,
		"reload":           true,
		"graceful":         true,
		"install":          true,
		"toggle_autostart": true,
	}

	if !validActions[action] {
		return "", fmt.Errorf("aksi layanan tidak valid: %s", action)
	}

	if action == "toggle_autostart" {
		st, _ := s.Status(name)
		newVal := true
		if st != nil && st.Autostart {
			newVal = false
		}
		err := s.SetAutostart(name, newVal)
		if err != nil {
			return "", err
		}
		if newVal {
			return "Autostart diaktifkan", nil
		}
		return "Autostart dinonaktifkan", nil
	}

	if action == "install" {
		return s.InstallAndSetup(name)
	}

	// Persiapan khusus sebelum start jika diperlukan
	if action == "start" || action == "restart" {
		s.prepareService(name)
	}

	res, err := s.runner.ExecWithTimeout(20*time.Second, "service", name, action)
	if err != nil {
		// Fallback ke onestart jika layanan belum diaktifkan di rc.conf
		if action == "start" {
			oneRes, oneErr := s.runner.ExecWithTimeout(20*time.Second, "service", name, "onestart")
			if oneErr == nil {
				return oneRes.Stdout, nil
			}
		}
		return res.Stderr, err
	}
	return res.Stdout, nil
}

// prepareService memastikan prasyarat konfigurasi daemon FreeBSD terpenuhi sebelum start.
func (s *ServiceManager) prepareService(name string) {
	rcvar := getRcvar(name)
	_ = s.SetAutostart(name, true)

	switch name {
	case "apache24":
		// Pastikan Apache mendengarkan port 8081 agar tidak bentrok dengan Nginx port 80
		httpdConf := "/usr/local/etc/apache24/httpd.conf"
		data, err := os.ReadFile(httpdConf)
		if err == nil {
			content := string(data)
			changed := false
			if strings.Contains(content, "Listen 80\n") || strings.Contains(content, "Listen 80\r\n") {
				content = strings.Replace(content, "Listen 80", "Listen 8081", 1)
				changed = true
			}
			if !strings.Contains(content, "ServerName") {
				content += "\nServerName localhost:8081\n"
				changed = true
			}
			if changed {
				_ = os.WriteFile(httpdConf, []byte(content), 0644)
			}
		}

	case "postgresql":
		// Inisialisasi cluster database jika belum pernah diinisialisasi
		dataDir := "/var/db/postgres/data16"
		if _, err := os.Stat(dataDir); os.IsNotExist(err) {
			_, _ = s.runner.ExecWithTimeout(60*time.Second, "service", "postgresql", "initdb")
		}

	case "pf":
		// Pastikan /etc/pf.conf ada
		pfConf := "/etc/pf.conf"
		if _, err := os.Stat(pfConf); os.IsNotExist(err) {
			defaultRules := "# BSDPanel Default PF Ruleset\nset skip on lo0\npass in all\npass out all\n"
			_ = os.WriteFile(pfConf, []byte(defaultRules), 0644)
		}
	}
	_ = rcvar
}

// InstallAndSetup memasang paket yang belum terpasang via pkg dan menginisialisasi konfigurasinya.
func (s *ServiceManager) InstallAndSetup(name string) (string, error) {
	pkgMap := map[string]string{
		"nginx":        "nginx",
		"apache24":     "apache24",
		"php_fpm":      "php83 php83-curl php83-mysqli php83-pdo_mysql php83-mbstring php83-zip php83-gd",
		"mysql-server": "mariadb1011-server",
		"postgresql":   "postgresql16-server",
	}

	pkgs, ok := pkgMap[name]
	if !ok && name != "pf" {
		return "", fmt.Errorf("paket untuk layanan %s tidak terdaftar", name)
	}

	if pkgs != "" {
		pkgArgs := append([]string{"install", "-y"}, strings.Fields(pkgs)...)
		res, err := s.runner.ExecWithTimeout(300*time.Second, "pkg", pkgArgs...)
		if err != nil {
			return res.Stderr, fmt.Errorf("gagal memasang paket %s: %s", pkgs, res.Stderr)
		}
	}

	s.prepareService(name)
	_ = s.SetAutostart(name, true)

	// Jalankan layanan
	res, err := s.runner.ExecWithTimeout(20*time.Second, "service", name, "start")
	if err != nil {
		oneRes, oneErr := s.runner.ExecWithTimeout(20*time.Second, "service", name, "onestart")
		if oneErr == nil {
			return oneRes.Stdout, nil
		}
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
	rcvar := getRcvar(name)
	_, err := s.runner.ExecWithTimeout(5*time.Second, "sysrc", fmt.Sprintf("%s_enable=%s", rcvar, val))
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
