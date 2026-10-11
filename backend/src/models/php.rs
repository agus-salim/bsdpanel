use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PhpVersionInfo {
    pub version: String, // e.g. "8.2", "8.3", "8.4", "8.5", "8.6"
    pub is_installed: bool,
    pub is_running: bool,
    pub service_name: String, // "php-fpm-82", "php-fpm-83", etc.
    pub socket_path: String,
    pub port: u16,
    pub memory_mb: f32,
    pub active_processes: u32,
    pub ini_path: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PhpExtension {
    pub name: String,
    pub is_enabled: bool,
    pub category: String,
    pub description: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PhpIniSettings {
    pub memory_limit: String,         // e.g. "256M"
    pub upload_max_filesize: String,   // e.g. "64M"
    pub post_max_size: String,         // e.g. "128M"
    pub max_execution_time: u32,       // e.g. 300
    pub max_input_vars: u32,           // e.g. 3000
    pub display_errors: bool,
    pub opcache_enable: bool,
    pub opcache_memory_consumption: u32, // e.g. 128
    pub disable_functions: String,
    pub date_timezone: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PhpPoolSettings {
    pub version: String,
    pub pm_type: String, // "dynamic", "ondemand", "static"
    pub max_children: u32,
    pub start_servers: u32,
    pub min_spare_servers: u32,
    pub max_spare_servers: u32,
    pub max_requests: u32,
}
