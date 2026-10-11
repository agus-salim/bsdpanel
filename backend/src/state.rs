use parking_lot::RwLock;
use std::sync::Arc;
use uuid::Uuid;

use crate::config::AppConfig;
use crate::models::*;
use crate::services::db_mgr::DbManager;
use crate::services::firewall_mgr::FirewallManager;
use crate::services::php_mgr::PhpManager;
use crate::services::user_mgr::UserManager;
use crate::services::webserver_mgr::WebServerManager;

pub struct AppStateInner {
    pub config: AppConfig,
    pub users: Vec<User>,
    pub domains: Vec<Domain>,
    pub webservers: Vec<WebServerInfo>,
    pub php_versions: Vec<PhpVersionInfo>,
    pub databases: Vec<DatabaseItem>,
    pub db_users: Vec<DatabaseUser>,
    pub firewall_rules: Vec<FirewallRule>,
    pub fail2ban_jails: Vec<Fail2banJailInfo>,
    pub banned_ips: Vec<BannedIpRecord>,
}

#[derive(Clone)]
pub struct AppState {
    pub inner: Arc<RwLock<AppStateInner>>,
}

impl AppState {
    pub fn new(config: AppConfig) -> Self {
        let now = chrono::Utc::now().timestamp();
        let users = UserManager::get_initial_seed_users();
        let webservers = WebServerManager::get_available_servers(config.mock_mode);
        let php_versions = PhpManager::get_available_versions(config.mock_mode);
        let firewall_rules = FirewallManager::get_default_rules();
        let fail2ban_jails = FirewallManager::get_default_jails();

        let domains = vec![
            Domain {
                id: Uuid::new_v4(),
                domain_name: "example.com".to_string(),
                owner_username: "client_company".to_string(),
                webserver_type: WebServerType::Nginx,
                php_version: "8.3".to_string(),
                document_root: "/usr/local/www/client_company/example.com/public".to_string(),
                ssl_enabled: true,
                force_https: true,
                is_active: true,
                reverse_proxy_target: None,
                created_at: now - 86400 * 5,
            },
            Domain {
                id: Uuid::new_v4(),
                domain_name: "api.service.net".to_string(),
                owner_username: "client_company".to_string(),
                webserver_type: WebServerType::Caddy,
                php_version: "8.4".to_string(),
                document_root: "/usr/local/www/client_company/api.service.net/public".to_string(),
                ssl_enabled: true,
                force_https: true,
                is_active: true,
                reverse_proxy_target: None,
                created_at: now - 86400 * 2,
            },
            Domain {
                id: Uuid::new_v4(),
                domain_name: "legacy.store.id".to_string(),
                owner_username: "client_company".to_string(),
                webserver_type: WebServerType::Apache,
                php_version: "8.2".to_string(),
                document_root: "/usr/local/www/client_company/legacy.store.id/public".to_string(),
                ssl_enabled: false,
                force_https: false,
                is_active: true,
                reverse_proxy_target: None,
                created_at: now - 86400 * 10,
            },
        ];

        let databases = vec![
            DatabaseItem {
                id: Uuid::new_v4(),
                name: "company_prod".to_string(),
                engine: DatabaseEngine::Mariadb,
                owner_username: "client_company".to_string(),
                size_mb: 48.5,
                tables_count: 32,
                collation: "utf8mb4_unicode_ci".to_string(),
                created_at: now - 86400 * 5,
            },
            DatabaseItem {
                id: Uuid::new_v4(),
                name: "pg_analytics".to_string(),
                engine: DatabaseEngine::Postgresql,
                owner_username: "client_company".to_string(),
                size_mb: 182.1,
                tables_count: 14,
                collation: "en_US.UTF-8".to_string(),
                created_at: now - 86400 * 3,
            },
        ];

        let db_users = vec![
            DatabaseUser {
                id: Uuid::new_v4(),
                username: "user_company".to_string(),
                engine: DatabaseEngine::Mariadb,
                owner_username: "client_company".to_string(),
                allowed_host: "localhost".to_string(),
                assigned_databases: vec!["company_prod".to_string()],
                created_at: now - 86400 * 5,
            },
            DatabaseUser {
                id: Uuid::new_v4(),
                username: "pg_client".to_string(),
                engine: DatabaseEngine::Postgresql,
                owner_username: "client_company".to_string(),
                allowed_host: "127.0.0.1".to_string(),
                assigned_databases: vec!["pg_analytics".to_string()],
                created_at: now - 86400 * 3,
            },
        ];

        let banned_ips = vec![
            BannedIpRecord {
                ip: "198.51.100.23".to_string(),
                jail: "bsd-sshd".to_string(),
                banned_at: now - 1800,
                expires_at: now + 1800,
                country: Some("RU".to_string()),
            },
            BannedIpRecord {
                ip: "203.0.113.88".to_string(),
                jail: "bsd-sshd".to_string(),
                banned_at: now - 3200,
                expires_at: now + 400,
                country: Some("CN".to_string()),
            },
            BannedIpRecord {
                ip: "185.220.101.5".to_string(),
                jail: "bsd-sshd".to_string(),
                banned_at: now - 900,
                expires_at: now + 2700,
                country: Some("DE".to_string()),
            },
            BannedIpRecord {
                ip: "192.0.2.14".to_string(),
                jail: "bsd-panel-auth".to_string(),
                banned_at: now - 1200,
                expires_at: now + 6000,
                country: Some("US".to_string()),
            },
            BannedIpRecord {
                ip: "45.140.19.8".to_string(),
                jail: "nginx-badbots".to_string(),
                banned_at: now - 14400,
                expires_at: now + 72000,
                country: Some("NL".to_string()),
            },
        ];

        Self {
            inner: Arc::new(RwLock::new(AppStateInner {
                config,
                users,
                domains,
                webservers,
                php_versions,
                databases,
                db_users,
                firewall_rules,
                fail2ban_jails,
                banned_ips,
            })),
        }
    }
}
