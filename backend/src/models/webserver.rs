use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum WebServerType {
    Nginx,
    Apache,
    LiteSpeed,
    Caddy,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebServerInfo {
    pub server_type: WebServerType,
    pub display_name: String,
    pub is_installed: bool,
    pub is_running: bool,
    pub is_primary: bool,
    pub version: String,
    pub port_http: u16,
    pub port_https: u16,
    pub service_name: String,
    pub config_path: String,
    pub memory_mb: f32,
    pub cpu_percent: f32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Domain {
    pub id: Uuid,
    pub domain_name: String,
    pub owner_username: String,
    pub webserver_type: WebServerType,
    pub php_version: String,
    pub document_root: String,
    pub ssl_enabled: bool,
    pub force_https: bool,
    pub is_active: bool,
    pub reverse_proxy_target: Option<String>,
    pub created_at: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateDomainPayload {
    pub domain_name: String,
    pub webserver_type: WebServerType,
    pub php_version: String,
    pub force_https: bool,
    pub ssl_auto_cert: bool,
    pub reverse_proxy_target: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateDomainPayload {
    pub webserver_type: Option<WebServerType>,
    pub php_version: Option<String>,
    pub force_https: Option<bool>,
    pub ssl_enabled: Option<bool>,
    pub is_active: Option<bool>,
    pub reverse_proxy_target: Option<String>,
}
