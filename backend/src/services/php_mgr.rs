use crate::models::{PhpExtension, PhpIniSettings, PhpPoolSettings, PhpVersionInfo};

pub struct PhpManager;

impl PhpManager {
    pub const SUPPORTED_VERSIONS: &'static [&'static str] = &["8.2", "8.3", "8.4", "8.5", "8.6"];

    pub fn get_available_versions(mock_mode: bool) -> Vec<PhpVersionInfo> {
        let versions = [
            ("8.2", true, true, 9002, 38.4, 4),
            ("8.3", true, true, 9003, 42.1, 5),
            ("8.4", true, false, 9004, 0.0, 0),
            ("8.5", false, false, 9005, 0.0, 0),
            ("8.6", false, false, 9006, 0.0, 0),
        ];

        versions
            .into_iter()
            .map(|(ver, installed, running, port, mem, procs)| {
                let clean_ver = ver.replace('.', "");
                PhpVersionInfo {
                    version: ver.to_string(),
                    is_installed: installed,
                    is_running: running,
                    service_name: format!("php-fpm-{}", clean_ver),
                    socket_path: format!("/var/run/php-fpm-{}.sock", clean_ver),
                    port,
                    memory_mb: mem,
                    active_processes: procs,
                    ini_path: format!("/usr/local/etc/php{}/php.ini", clean_ver),
                }
            })
            .collect()
    }

    pub fn get_default_extensions(version: &str) -> Vec<PhpExtension> {
        vec![
            PhpExtension {
                name: "opcache".to_string(),
                is_enabled: true,
                category: "Performance".to_string(),
                description: "Zend OPcache bytecode caching engine".to_string(),
            },
            PhpExtension {
                name: "mbstring".to_string(),
                is_enabled: true,
                category: "String".to_string(),
                description: "Multibyte string support for internationalization".to_string(),
            },
            PhpExtension {
                name: "curl".to_string(),
                is_enabled: true,
                category: "Network".to_string(),
                description: "libcurl client URL library binding".to_string(),
            },
            PhpExtension {
                name: "pdo_mysql".to_string(),
                is_enabled: true,
                category: "Database".to_string(),
                description: "MySQL & MariaDB database driver for PDO".to_string(),
            },
            PhpExtension {
                name: "pdo_pgsql".to_string(),
                is_enabled: true,
                category: "Database".to_string(),
                description: "PostgreSQL database driver for PDO".to_string(),
            },
            PhpExtension {
                name: "gd".to_string(),
                is_enabled: true,
                category: "Media".to_string(),
                description: "GD Graphics Library for image creation and manipulation".to_string(),
            },
            PhpExtension {
                name: "zip".to_string(),
                is_enabled: true,
                category: "Compression".to_string(),
                description: "ZIP archive management library".to_string(),
            },
            PhpExtension {
                name: "intl".to_string(),
                is_enabled: true,
                category: "Localization".to_string(),
                description: "Internationalization functions (ICU)".to_string(),
            },
            PhpExtension {
                name: "redis".to_string(),
                is_enabled: true,
                category: "Cache".to_string(),
                description: "PHP extension for Redis in-memory cache".to_string(),
            },
            PhpExtension {
                name: "imagick".to_string(),
                is_enabled: false,
                category: "Media".to_string(),
                description: "ImageMagick graphics manipulation".to_string(),
            },
            PhpExtension {
                name: "sodium".to_string(),
                is_enabled: true,
                category: "Security".to_string(),
                description: "Modern and easy-to-use crypto library".to_string(),
            },
        ]
    }

    pub fn get_default_ini(version: &str) -> PhpIniSettings {
        PhpIniSettings {
            memory_limit: "256M".to_string(),
            upload_max_filesize: "64M".to_string(),
            post_max_size: "128M".to_string(),
            max_execution_time: 300,
            max_input_vars: 3000,
            display_errors: false,
            opcache_enable: true,
            opcache_memory_consumption: 128,
            disable_functions: "exec,passthru,shell_exec,system,proc_open,popen".to_string(),
            date_timezone: "Asia/Jakarta".to_string(),
        }
    }

    pub fn generate_pool_conf(domain_name: &str, username: &str, version: &str, pool: &PhpPoolSettings) -> String {
        let clean_ver = version.replace('.', "");
        format!(
r#"; BSD Panel Managed PHP-FPM Pool
[{domain_name}]
user = {username}
group = {username}

listen = /var/run/php-fpm-{clean_ver}-{domain_name}.sock
listen.owner = www
listen.group = www
listen.mode = 0660

pm = {pm_type}
pm.max_children = {max_children}
pm.start_servers = {start_servers}
pm.min_spare_servers = {min_spare_servers}
pm.max_spare_servers = {max_spare_servers}
pm.max_requests = {max_requests}

php_admin_value[error_log] = /var/log/php-fpm/{domain_name}-error.log
php_admin_flag[log_errors] = on
"#,
            domain_name = domain_name,
            username = username,
            clean_ver = clean_ver,
            pm_type = pool.pm_type,
            max_children = pool.max_children,
            start_servers = pool.start_servers,
            min_spare_servers = pool.min_spare_servers,
            max_spare_servers = pool.max_spare_servers,
            max_requests = pool.max_requests,
        )
    }
}
