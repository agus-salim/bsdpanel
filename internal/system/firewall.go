package system

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// FirewallRule represents a port rule in FreeBSD pf.
type FirewallRule struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"` // tcp, udp
	Action   string `json:"action"`   // pass, block
	Comment  string `json:"comment"`
}

// FirewallManager manages FreeBSD native PF (Packet Filter) and Fail2ban.
type FirewallManager struct {
	exec       *Executor
	serviceMgr *ServiceManager
	pfConfPath string
}

// NewFirewallManager creates a new firewall manager.
func NewFirewallManager(exec *Executor, srv *ServiceManager) *FirewallManager {
	return &FirewallManager{
		exec:       exec,
		serviceMgr: srv,
		pfConfPath: "/etc/pf.conf",
	}
}

// IsPFActive checks if PF is running on FreeBSD.
func (f *FirewallManager) IsPFActive(ctx context.Context) bool {
	res, err := f.exec.Execute(ctx, "/sbin/pfctl", "-s", "info")
	return err == nil && strings.Contains(res.Stdout, "Status: Enabled")
}

// Reload reloads the PF ruleset.
func (f *FirewallManager) Reload(ctx context.Context) error {
	cmd := "/sbin/pfctl"
	args := []string{"-f", f.pfConfPath}
	if f.exec.UseDoas {
		args = append([]string{cmd}, args...)
		cmd = "/usr/local/bin/doas"
	}
	_, err := f.exec.Execute(ctx, cmd, args...)
	return err
}

// GenerateStandardPFConfig creates a robust baseline /etc/pf.conf with protection against synflood and brute force.
func (f *FirewallManager) GenerateStandardPFConfig(openPorts []int) string {
	portsStr := "80 443 22"
	if len(openPorts) > 0 {
		var parts []string
		for _, p := range openPorts {
			parts = append(parts, fmt.Sprintf("%d", p))
		}
		portsStr = strings.Join(parts, " ")
	}

	return fmt.Sprintf(`# BSD Panel Managed pf.conf
ext_if = "vtnet0" # Adjust per network interface (e.g., em0, re0, vtnet0)

# Tables for blocking attackers & Fail2ban integration
table <fail2ban> persist
table <bruteforce> persist counters

# Default policies
set skip on lo0
set block-policy drop

# Scrub incoming packets
scrub in all fragment reassemble

# Quick block tables
block quick from <fail2ban>
block quick from <bruteforce>

# Standard filtering: block everything by default, allow stateful egress
block in all
pass out quick all keep state

# Inbound allowed services with rate-limiting
pass in on $ext_if proto tcp to any port { %s } \
    flags S/SA keep state \
    (max-src-conn 100, max-src-conn-rate 30/5, overload <bruteforce> flush global)

# ICMP ping
pass in inet proto icmp all icmp-type echoreq keep state
`, portsStr)
}

// ApplyConfig writes and reloads pf.conf safely.
func (f *FirewallManager) ApplyConfig(ctx context.Context, configContent string) error {
	tmpFile := "/tmp/pf.conf.test"
	if err := os.WriteFile(tmpFile, []byte(configContent), 0600); err != nil {
		return err
	}

	// Verify syntax first: pfctl -n -f /tmp/pf.conf.test
	res, err := f.exec.Execute(ctx, "/sbin/pfctl", "-n", "-f", tmpFile)
	if err != nil {
		return fmt.Errorf("pf syntax validation error: %s (%s)", err, res.Stderr)
	}

	// Move to actual destination
	cpCmd := "/bin/cp"
	cpArgs := []string{tmpFile, f.pfConfPath}
	if f.exec.UseDoas {
		cpArgs = append([]string{cpCmd}, cpArgs...)
		cpCmd = "/usr/local/bin/doas"
	}
	if _, err := f.exec.Execute(ctx, cpCmd, cpArgs...); err != nil {
		return err
	}

	_ = os.Remove(tmpFile)
	return f.Reload(ctx)
}

// GetFail2banStatus checks jail status from fail2ban-client.
func (f *FirewallManager) GetFail2banStatus(ctx context.Context) (string, error) {
	cmd := "/usr/local/bin/fail2ban-client"
	args := []string{"status"}
	if f.exec.UseDoas {
		args = append([]string{cmd}, args...)
		cmd = "/usr/local/bin/doas"
	}
	res, err := f.exec.Execute(ctx, cmd, args...)
	if err != nil {
		return "fail2ban not installed or inactive", nil
	}
	return res.Stdout, nil
}
