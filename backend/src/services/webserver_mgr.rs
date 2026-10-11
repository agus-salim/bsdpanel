use std::path::Path;
use tracing::info;
use crate::models::{Domain, WebServerInfo, WebServerType};

pub struct WebServerManager;

impl WebServerManager {
    pub fn get_available_servers(mock_mode: bool) -> Vec<WebServerInfo> {
        vec![
            WebServerInfo {
                server_type: WebServerType::Nginx,
                display_name: "NGINX High Performance HTTP Server".to_string(),
                is_installed: true,
                is_running: true,
                is_primary: true,
                version: "1.26.2".to_string(),
                port_http: 80,
                port_https: 443,
                service_name: "nginx".to_string(),
                config_path: "/usr/local/etc/nginx/nginx.conf".to_string(),
                memory_mb: 64.5,
                cpu_percent: 1.2,
            },
            WebServerInfo {
                server_type: WebServerType::Apache,
                display_name: "Apache HTTP Server (httpd24)".to_string(),
                is_installed: true,
                is_running: false,
                is_primary: false,
                version: "2.4.62".to_string(),
                port_http: 8080,
                port_https: 8443,
                service_name: "apache24".to_string(),
                config_path: "/usr/local/etc/apache24/httpd.conf".to_string(),
                memory_mb: 0.0,
                cpu_percent: 0.0,
            },
            WebServerInfo {
                server_type: WebServerType::LiteSpeed,
                display_name: "OpenLiteSpeed Web Server".to_string(),
                is_installed: true,
                is_running: false,
                is_primary: false,
                version: "1.8.2".to_string(),
                port_http: 8088,
                port_https: 7080,
                service_name: "lsws".to_string(),
                config_path: "/usr/local/lsws/conf/httpd_config.conf".to_string(),
                memory_mb: 0.0,
                cpu_percent: 0.0,
            },
            WebServerInfo {
                server_type: WebServerType::Caddy,
                display_name: "Caddy Server (Automated HTTPS)".to_string(),
                is_installed: true,
                is_running: false,
                is_primary: false,
                version: "2.8.4".to_string(),
                port_http: 2015,
                port_https: 2016,
                service_name: "caddy".to_string(),
                config_path: "/usr/local/etc/caddy/Caddyfile".to_string(),
                memory_mb: 0.0,
                cpu_percent: 0.0,
            },
        ]
    }

    pub fn generate_nginx_vhost(domain: &Domain, php_sock: &str) -> String {
        format!(
r#"# BSD Panel Managed VHost: {}
server {{
    listen 80;
    listen [::]:80;
    server_name {domain_name} www.{domain_name};

    root {doc_root};
    index index.php index.html index.htm;

    access_log /var/log/nginx/{domain_name}_access.log;
    error_log /var/log/nginx/{domain_name}_error.log;

    {https_redirect}

    location / {{
        try_files $uri $uri/ /index.php?$args;
    }}

    location ~ \.php$ {{
        try_files $uri =404;
        fastcgi_split_path_info ^(.+\.php)(/.+)$;
        fastcgi_pass unix:{php_sock};
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        include fastcgi_params;
        fastcgi_buffers 16 16k;
        fastcgi_buffer_size 32k;
    }}

    location ~ /\.ht {{
        deny all;
    }}
}}
"#,
            domain.domain_name,
            domain_name = domain.domain_name,
            doc_root = domain.document_root,
            https_redirect = if domain.force_https {
                "return 301 https://$host$request_uri;"
            } else {
                "# HTTP allowed"
            },
            php_sock = php_sock
        )
    }

    pub fn generate_apache_vhost(domain: &Domain, php_port: u16) -> String {
        format!(
r#"# BSD Panel Managed VHost: {}
<VirtualHost *:80>
    ServerName {domain_name}
    ServerAlias www.{domain_name}
    DocumentRoot "{doc_root}"

    <Directory "{doc_root}">
        Options -Indexes +FollowSymLinks
        AllowOverride All
        Require all granted
    </Directory>

    <FilesMatch \.php$>
        SetHandler "proxy:fcgi://127.0.0.1:{php_port}"
    </FilesMatch>

    ErrorLog "/var/log/httpd/{domain_name}_error.log"
    CustomLog "/var/log/httpd/{domain_name}_access.log" combined
</VirtualHost>
"#,
            domain.domain_name,
            domain_name = domain.domain_name,
            doc_root = domain.document_root,
            php_port = php_port
        )
    }

    pub fn generate_caddy_vhost(domain: &Domain, php_sock: &str) -> String {
        format!(
r#"# BSD Panel Managed Caddyfile: {}
{domain_name}, www.{domain_name} {{
    root * {doc_root}
    php_fastcgi unix/{php_sock}
    file_server
    encode gzip zstd
    log {{
        output file /var/log/caddy/{domain_name}.log
    }}
}}
"#,
            domain.domain_name,
            domain_name = domain.domain_name,
            doc_root = domain.document_root,
            php_sock = php_sock
        )
    }

    pub fn generate_litespeed_vhost(domain: &Domain, php_sock: &str) -> String {
        format!(
r#"# BSD Panel Managed OpenLiteSpeed VHost: {}
docRoot                   {doc_root}
enableGzip                1

errorlog /usr/local/lsws/logs/{domain_name}.error_log {{
  useServer               0
  logLevel                DEBUG
  rollingSize             10M
}}

accesslog /usr/local/lsws/logs/{domain_name}.access_log {{
  useServer               0
  rollingSize             10M
  keepDays                30
}}

index  {{
  useServer               0
  indexFiles              index.php, index.html
}}

scripthandler  {{
  add                     lsapi:{php_sock} php
}}
"#,
            domain.domain_name,
            domain_name = domain.domain_name,
            doc_root = domain.document_root,
            php_sock = php_sock
        )
    }
}
