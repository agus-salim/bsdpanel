use std::process::Command;
use tracing::info;
use uuid::Uuid;
use crate::models::{DatabaseEngine, DatabaseItem, DatabaseServerInfo, DatabaseUser};

pub struct DbManager;

impl DbManager {
    pub fn get_servers_info(mock_mode: bool) -> Vec<DatabaseServerInfo> {
        vec![
            DatabaseServerInfo {
                engine: DatabaseEngine::Mariadb,
                display_name: "MariaDB SQL Server (10.11 LTS)".to_string(),
                is_installed: true,
                is_running: true,
                version: "10.11.8-MariaDB".to_string(),
                service_name: "mysql-server".to_string(),
                port: 3306,
                memory_mb: 184.2,
                databases_count: 5,
                connections_count: 12,
            },
            DatabaseServerInfo {
                engine: DatabaseEngine::Postgresql,
                display_name: "PostgreSQL Object-Relational Database (16)".to_string(),
                is_installed: true,
                is_running: true,
                version: "PostgreSQL 16.4".to_string(),
                service_name: "postgresql".to_string(),
                port: 5432,
                memory_mb: 112.5,
                databases_count: 3,
                connections_count: 7,
            },
        ]
    }

    pub fn create_mariadb_database(db_name: &str, mock_mode: bool) -> Result<(), String> {
        info!("Creating MariaDB database: {}", db_name);
        if mock_mode {
            return Ok(());
        }

        let query = format!("CREATE DATABASE `{}` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;", db_name);
        let output = Command::new("mysql")
            .arg("-e")
            .arg(&query)
            .output()
            .map_err(|e| format!("Failed to execute mysql command: {}", e))?;

        if output.status.success() {
            Ok(())
        } else {
            Err(String::from_utf8_lossy(&output.stderr).to_string())
        }
    }

    pub fn create_mariadb_user(username: &str, password: &str, db_name: &str, host: &str, mock_mode: bool) -> Result<(), String> {
        info!("Creating MariaDB user: {} for db {}", username, db_name);
        if mock_mode {
            return Ok(());
        }

        let query = format!(
            "CREATE USER IF NOT EXISTS '{}'@'{}' IDENTIFIED BY '{}'; GRANT ALL PRIVILEGES ON `{}`.* TO '{}'@'{}'; FLUSH PRIVILEGES;",
            username, host, password, db_name, username, host
        );
        let output = Command::new("mysql")
            .arg("-e")
            .arg(&query)
            .output()
            .map_err(|e| format!("Failed to execute mysql command: {}", e))?;

        if output.status.success() {
            Ok(())
        } else {
            Err(String::from_utf8_lossy(&output.stderr).to_string())
        }
    }

    pub fn create_postgresql_database(db_name: &str, owner: &str, mock_mode: bool) -> Result<(), String> {
        info!("Creating PostgreSQL database: {} (owner: {})", db_name, owner);
        if mock_mode {
            return Ok(());
        }

        // Run as user 'postgres'
        let output = Command::new("su")
            .arg("-l")
            .arg("pgsql")
            .arg("-c")
            .arg(&format!("createdb -O \"{}\" \"{}\"", owner, db_name))
            .output()
            .map_err(|e| format!("Failed to execute createdb: {}", e))?;

        if output.status.success() {
            Ok(())
        } else {
            Err(String::from_utf8_lossy(&output.stderr).to_string())
        }
    }

    pub fn create_postgresql_user(username: &str, password: &str, mock_mode: bool) -> Result<(), String> {
        info!("Creating PostgreSQL user: {}", username);
        if mock_mode {
            return Ok(());
        }

        let query = format!("CREATE USER \"{}\" WITH ENCRYPTED PASSWORD '{}';", username, password);
        let output = Command::new("su")
            .arg("-l")
            .arg("pgsql")
            .arg("-c")
            .arg(&format!("psql -c \"{}\"", query))
            .output()
            .map_err(|e| format!("Failed to execute psql user creation: {}", e))?;

        if output.status.success() {
            Ok(())
        } else {
            Err(String::from_utf8_lossy(&output.stderr).to_string())
        }
    }
}
