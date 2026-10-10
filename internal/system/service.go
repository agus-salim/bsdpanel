package system

import (
	"context"
	"fmt"
	"strings"
)

// ServiceStatus holds current state of a FreeBSD rc.d service.
type ServiceStatus struct {
	Name      string `json:"name"`
	IsRunning bool   `json:"is_running"`
	IsEnabled bool   `json:"is_enabled"`
	Pid       int    `json:"pid"`
	Details   string `json:"details"`
}

// ServiceManager controls FreeBSD daemon states using native rc.d and sysrc.
type ServiceManager struct {
	exec *Executor
}

// NewServiceManager initializes a service manager.
func NewServiceManager(exec *Executor) *ServiceManager {
	return &ServiceManager{exec: exec}
}

// Status checks whether a FreeBSD service is running.
func (s *ServiceManager) Status(ctx context.Context, serviceName string) (*ServiceStatus, error) {
	cmd := "/usr/sbin/service"
	args := []string{serviceName, "onestatus"}

	res, err := s.exec.Execute(ctx, cmd, args...)
	isRunning := false
	if err == nil && (strings.Contains(res.Stdout, "is running as pid") || strings.Contains(res.Stdout, "is running")) {
		isRunning = true
	}

	enabled := s.IsEnabled(ctx, serviceName)

	return &ServiceStatus{
		Name:      serviceName,
		IsRunning: isRunning,
		IsEnabled: enabled,
		Details:   res.Stdout,
	}, nil
}

// Start boots a service via FreeBSD rc.d.
func (s *ServiceManager) Start(ctx context.Context, serviceName string) error {
	return s.action(ctx, serviceName, "start")
}

// Stop halts a running service.
func (s *ServiceManager) Stop(ctx context.Context, serviceName string) error {
	return s.action(ctx, serviceName, "stop")
}

// Restart restarts a FreeBSD service.
func (s *ServiceManager) Restart(ctx context.Context, serviceName string) error {
	return s.action(ctx, serviceName, "restart")
}

// Reload gracefully reloads configuration without dropping active connections.
func (s *ServiceManager) Reload(ctx context.Context, serviceName string) error {
	return s.action(ctx, serviceName, "reload")
}

// Enable configures the service to boot automatically via sysrc.
func (s *ServiceManager) Enable(ctx context.Context, serviceName string) error {
	cmd := "/usr/sbin/sysrc"
	args := []string{fmt.Sprintf("%s_enable=YES", serviceName)}
	if s.exec.UseDoas {
		args = append([]string{cmd}, args...)
		cmd = "/usr/local/bin/doas"
	}
	_, err := s.exec.Execute(ctx, cmd, args...)
	return err
}

// Disable turns off auto-start in /etc/rc.conf.
func (s *ServiceManager) Disable(ctx context.Context, serviceName string) error {
	cmd := "/usr/sbin/sysrc"
	args := []string{fmt.Sprintf("%s_enable=NO", serviceName)}
	if s.exec.UseDoas {
		args = append([]string{cmd}, args...)
		cmd = "/usr/local/bin/doas"
	}
	_, err := s.exec.Execute(ctx, cmd, args...)
	return err
}

// IsEnabled checks if rc.conf has the service enabled.
func (s *ServiceManager) IsEnabled(ctx context.Context, serviceName string) bool {
	res, err := s.exec.Execute(ctx, "/usr/sbin/sysrc", "-n", fmt.Sprintf("%s_enable", serviceName))
	if err != nil {
		return false
	}
	val := strings.ToUpper(strings.TrimSpace(res.Stdout))
	return val == "YES" || val == "TRUE" || val == "ON" || val == "1"
}

func (s *ServiceManager) action(ctx context.Context, serviceName, action string) error {
	cmd := "/usr/sbin/service"
	args := []string{serviceName, action}
	if s.exec.UseDoas {
		args = append([]string{cmd}, args...)
		cmd = "/usr/local/bin/doas"
	}
	res, err := s.exec.Execute(ctx, cmd, args...)
	if err != nil {
		// On FreeBSD, if a service is not yet enabled in /etc/rc.conf,
		// standard actions (start/restart/reload/stop) fail with exit 1.
		// Retrying with the 'one' prefix (e.g., onerestart, onestart) bypasses rc.conf checks.
		oneAction := "one" + action
		argsOne := []string{serviceName, oneAction}
		if s.exec.UseDoas {
			argsOne = append([]string{cmd}, argsOne...)
		}
		resOne, errOne := s.exec.Execute(ctx, cmd, argsOne...)
		if errOne == nil {
			return nil
		}

		errMsg := res.Stderr
		if errMsg == "" {
			errMsg = res.Stdout
		}
		if errMsg == "" {
			errMsg = resOne.Stderr
		}
		if errMsg == "" {
			errMsg = resOne.Stdout
		}
		return fmt.Errorf("service %s %s failed: %s", serviceName, action, errMsg)
	}
	return nil
}
