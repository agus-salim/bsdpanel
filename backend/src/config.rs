use serde::{Deserialize, Serialize};
use std::path::PathBuf;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AppConfig {
    pub bind_addr: String,
    pub port: u16,
    pub jwt_secret: String,
    pub mock_mode: bool,
    pub data_dir: PathBuf,
    pub ssl_dir: PathBuf,
    pub web_root: PathBuf,
}

impl Default for AppConfig {
    fn default() -> Self {
        let is_freebsd = cfg!(target_os = "freebsd");
        let mock_mode = std::env::var("BSD_PANEL_MOCK")
            .map(|v| v == "1" || v == "true")
            .unwrap_or(!is_freebsd);

        Self {
            bind_addr: std::env::var("BSD_PANEL_HOST").unwrap_or_else(|_| "0.0.0.0".to_string()),
            port: std::env::var("BSD_PANEL_PORT")
                .ok()
                .and_then(|p| p.parse().ok())
                .unwrap_or(8888),
            jwt_secret: std::env::var("BSD_PANEL_JWT_SECRET")
                .unwrap_or_else(|_| "bsd_panel_super_secure_jwt_secret_freebsd_2026".to_string()),
            mock_mode,
            data_dir: PathBuf::from(if is_freebsd {
                "/usr/local/etc/bsdpanel"
            } else {
                "./data"
            }),
            ssl_dir: PathBuf::from(if is_freebsd {
                "/usr/local/etc/bsdpanel/ssl"
            } else {
                "./data/ssl"
            }),
            web_root: PathBuf::from(if is_freebsd {
                "/usr/local/www"
            } else {
                "./data/www"
            }),
        }
    }
}
