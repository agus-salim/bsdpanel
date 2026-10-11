use std::process::Command;
use tracing::{info, warn};
use crate::models::{FirewallAction, FirewallRule, RuleDirection, RuleProtocol};

pub struct PfFirewall;

impl PfFirewall {
    pub fn is_enabled(mock_mode: bool) -> bool {
        if mock_mode {
            return true;
        }

        let output = Command::new("/sbin/pfctl")
            .arg("-s")
            .arg("info")
            .output();

        match output {
            Ok(out) => {
                let stdout = String::from_utf8_lossy(&out.stdout);
                stdout.contains("Status: Enabled")
            }
            Err(_) => false,
        }
    }

    pub fn toggle_pf(enable: bool, mock_mode: bool) -> Result<(), String> {
        if mock_mode {
            return Ok(());
        }

        let flag = if enable { "-e" } else { "-d" };
        let output = Command::new("/sbin/pfctl")
            .arg(flag)
            .output()
            .map_err(|e| format!("Failed to toggle pfctl: {}", e))?;

        if output.status.success() {
            Ok(())
        } else {
            Err(String::from_utf8_lossy(&output.stderr).to_string())
        }
    }

    pub fn get_states_count(mock_mode: bool) -> u32 {
        if mock_mode {
            return 142;
        }

        let output = Command::new("/sbin/pfctl")
            .arg("-s")
            .arg("info")
            .output();

        if let Ok(out) = output {
            let text = String::from_utf8_lossy(&out.stdout);
            for line in text.lines() {
                if line.contains("current entries") {
                    let parts: Vec<&str> = line.split_whitespace().collect();
                    if let Some(count_str) = parts.get(2) {
                        if let Ok(count) = count_str.parse::<u32>() {
                            return count;
                        }
                    }
                }
            }
        }
        0
    }

    pub fn ban_ip_in_table(table_name: &str, ip: &str, mock_mode: bool) -> Result<(), String> {
        info!("PF adding IP {} to table <{}>", ip, table_name);
        if mock_mode {
            return Ok(());
        }

        let output = Command::new("/sbin/pfctl")
            .arg("-t")
            .arg(table_name)
            .arg("-T")
            .arg("add")
            .arg(ip)
            .output()
            .map_err(|e| format!("Failed to add to pf table: {}", e))?;

        if output.status.success() {
            Ok(())
        } else {
            Err(String::from_utf8_lossy(&output.stderr).to_string())
        }
    }

    pub fn unban_ip_in_table(table_name: &str, ip: &str, mock_mode: bool) -> Result<(), String> {
        info!("PF removing IP {} from table <{}>", ip, table_name);
        if mock_mode {
            return Ok(());
        }

        let output = Command::new("/sbin/pfctl")
            .arg("-t")
            .arg(table_name)
            .arg("-T")
            .arg("delete")
            .arg(ip)
            .output()
            .map_err(|e| format!("Failed to delete from pf table: {}", e))?;

        if output.status.success() {
            Ok(())
        } else {
            Err(String::from_utf8_lossy(&output.stderr).to_string())
        }
    }

    pub fn generate_pf_conf(rules: &[FirewallRule]) -> String {
        let mut conf = String::new();
        conf.push_str("# ==========================================================\n");
        conf.push_str("# FreeBSD PF (Packet Filter) Configuration - Managed by BSD Panel\n");
        conf.push_str("# ==========================================================\n\n");
        conf.push_str("# Macros & Interfaces\n");
        conf.push_str("ext_if = \"vtnet0\"\n");
        conf.push_str("table <bruteforce> persist\n");
        conf.push_str("table <whitelist> persist { 127.0.0.1, ::1 }\n\n");
        conf.push_str("# Options\n");
        conf.push_str("set block-policy drop\n");
        conf.push_str("set skip on lo0\n\n");
        conf.push_str("# Normalization\n");
        conf.push_str("scrub in all fragment reassemble\n\n");
        conf.push_str("# Default Filtering\n");
        conf.push_str("block in all\n");
        conf.push_str("pass out all keep state\n");
        conf.push_str("block quick from <bruteforce>\n");
        conf.push_str("pass quick from <whitelist>\n\n");
        conf.push_str("# Custom Firewall Rules\n");

        for rule in rules {
            if !rule.is_enabled {
                continue;
            }
            let action_str = match rule.action {
                FirewallAction::Pass => "pass",
                FirewallAction::Block => "block",
            };
            let dir_str = match rule.direction {
                RuleDirection::In => "in",
                RuleDirection::Out => "out",
            };

            let proto_str = match rule.protocol {
                RuleProtocol::Tcp => "proto tcp ",
                RuleProtocol::Udp => "proto udp ",
                RuleProtocol::Both => "{ proto tcp, proto udp } ",
                RuleProtocol::Icmp => "proto icmp ",
            };

            let port_str = if rule.port == "any" || rule.protocol == RuleProtocol::Icmp {
                String::new()
            } else {
                format!("to any port {} ", rule.port)
            };

            let from_str = if rule.source == "any" {
                "from any "
            } else {
                &format!("from {} ", rule.source)
            };

            conf.push_str(&format!(
                "{} {} on $ext_if {}{}{}keep state # {}\n",
                action_str, dir_str, proto_str, from_str, port_str, rule.label
            ));
        }

        conf
    }

    pub fn apply_pf_conf(conf_content: &str, mock_mode: bool) -> Result<(), String> {
        if mock_mode {
            return Ok(());
        }

        let pf_conf_path = "/etc/pf.conf";
        std::fs::write(pf_conf_path, conf_content)
            .map_err(|e| format!("Failed to write /etc/pf.conf: {}", e))?;

        let output = Command::new("/sbin/pfctl")
            .arg("-f")
            .arg(pf_conf_path)
            .output()
            .map_err(|e| format!("Failed to reload pf rules: {}", e))?;

        if output.status.success() {
            Ok(())
        } else {
            Err(String::from_utf8_lossy(&output.stderr).to_string())
        }
    }
}
