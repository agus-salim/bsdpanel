use std::process::Command;
use tracing::{info, warn};

pub struct ServiceRc;

impl ServiceRc {
    pub fn is_service_running(service_name: &str, mock_mode: bool) -> bool {
        if mock_mode {
            return true;
        }

        // On real FreeBSD: `service <name> status`
        let output = Command::new("/usr/sbin/service")
            .arg(service_name)
            .arg("status")
            .output();

        match output {
            Ok(out) => out.status.success(),
            Err(e) => {
                warn!("Failed to query service status for {}: {}", service_name, e);
                false
            }
        }
    }

    pub fn manage_service(service_name: &str, action: &str, mock_mode: bool) -> Result<String, String> {
        info!("Executing service {} action: {}", service_name, action);
        if mock_mode {
            return Ok(format!("Mock FreeBSD: service {} {} succeeded", service_name, action));
        }

        let valid_actions = ["start", "stop", "restart", "reload", "status", "onerestart", "onestop", "onestart"];
        if !valid_actions.contains(&action) {
            return Err(format!("Invalid FreeBSD service action: {}", action));
        }

        let output = Command::new("/usr/sbin/service")
            .arg(service_name)
            .arg(action)
            .output()
            .map_err(|e| format!("Failed to execute /usr/sbin/service: {}", e))?;

        if output.status.success() {
            Ok(String::from_utf8_lossy(&output.stdout).trim().to_string())
        } else {
            let err = String::from_utf8_lossy(&output.stderr).trim().to_string();
            Err(if err.is_empty() {
                String::from_utf8_lossy(&output.stdout).trim().to_string()
            } else {
                err
            })
        }
    }

    pub fn set_boot_enable(service_name: &str, enable: bool, mock_mode: bool) -> Result<(), String> {
        if mock_mode {
            return Ok(());
        }

        let val = if enable { "YES" } else { "NO" };
        let setting = format!("{}_enable={}", service_name, val);

        let output = Command::new("/usr/sbin/sysrc")
            .arg(&setting)
            .output()
            .map_err(|e| format!("Failed to run sysrc: {}", e))?;

        if output.status.success() {
            Ok(())
        } else {
            Err(String::from_utf8_lossy(&output.stderr).to_string())
        }
    }
}
