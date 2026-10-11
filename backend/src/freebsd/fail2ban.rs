use std::process::Command;
use tracing::{info, warn};

pub struct Fail2banMgr;

impl Fail2banMgr {
    pub fn is_installed(mock_mode: bool) -> bool {
        if mock_mode {
            return true;
        }
        std::path::Path::new("/usr/local/bin/fail2ban-client").exists()
    }

    pub fn unban_ip(jail: &str, ip: &str, mock_mode: bool) -> Result<(), String> {
        info!("Unbanning IP {} from fail2ban jail {}", ip, jail);
        if mock_mode {
            return Ok(());
        }

        let output = Command::new("/usr/local/bin/fail2ban-client")
            .arg("set")
            .arg(jail)
            .arg("unbanip")
            .arg(ip)
            .output()
            .map_err(|e| format!("Failed to run fail2ban-client unban: {}", e))?;

        if output.status.success() {
            Ok(())
        } else {
            Err(String::from_utf8_lossy(&output.stderr).to_string())
        }
    }

    pub fn ban_ip(jail: &str, ip: &str, mock_mode: bool) -> Result<(), String> {
        info!("Banning IP {} in fail2ban jail {}", ip, jail);
        if mock_mode {
            return Ok(());
        }

        let output = Command::new("/usr/local/bin/fail2ban-client")
            .arg("set")
            .arg(jail)
            .arg("banip")
            .arg(ip)
            .output()
            .map_err(|e| format!("Failed to run fail2ban-client ban: {}", e))?;

        if output.status.success() {
            Ok(())
        } else {
            Err(String::from_utf8_lossy(&output.stderr).to_string())
        }
    }
}
