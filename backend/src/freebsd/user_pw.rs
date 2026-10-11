use std::process::Command;
use tracing::{info, warn};

pub struct UserPw;

impl UserPw {
    pub fn create_system_user(username: &str, home_dir: &str, mock_mode: bool) -> Result<(), String> {
        info!("Creating FreeBSD system user {} with home {}", username, home_dir);
        if mock_mode {
            return Ok(());
        }

        // On FreeBSD: pw useradd <username> -m -d <home_dir> -s /usr/sbin/nologin -c "BSD Panel User"
        let output = Command::new("/usr/sbin/pw")
            .arg("useradd")
            .arg(username)
            .arg("-m")
            .arg("-d")
            .arg(home_dir)
            .arg("-s")
            .arg("/usr/sbin/nologin")
            .arg("-c")
            .arg("BSD Panel Managed User")
            .output()
            .map_err(|e| format!("Failed to execute /usr/sbin/pw useradd: {}", e))?;

        if output.status.success() {
            Ok(())
        } else {
            Err(String::from_utf8_lossy(&output.stderr).to_string())
        }
    }

    pub fn delete_system_user(username: &str, mock_mode: bool) -> Result<(), String> {
        info!("Deleting FreeBSD system user {}", username);
        if mock_mode {
            return Ok(());
        }

        let output = Command::new("/usr/sbin/pw")
            .arg("userdel")
            .arg(username)
            .arg("-r") // Remove home directory
            .output()
            .map_err(|e| format!("Failed to execute /usr/sbin/pw userdel: {}", e))?;

        if output.status.success() {
            Ok(())
        } else {
            Err(String::from_utf8_lossy(&output.stderr).to_string())
        }
    }
}
