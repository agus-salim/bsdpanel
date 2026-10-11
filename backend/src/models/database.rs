use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum DatabaseEngine {
    Postgresql,
    Mariadb,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DatabaseServerInfo {
    pub engine: DatabaseEngine,
    pub display_name: String,
    pub is_installed: bool,
    pub is_running: bool,
    pub version: String,
    pub service_name: String,
    pub port: u16,
    pub memory_mb: f32,
    pub databases_count: u32,
    pub connections_count: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DatabaseItem {
    pub id: Uuid,
    pub name: String,
    pub engine: DatabaseEngine,
    pub owner_username: String,
    pub size_mb: f32,
    pub tables_count: u32,
    pub collation: String,
    pub created_at: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DatabaseUser {
    pub id: Uuid,
    pub username: String,
    pub engine: DatabaseEngine,
    pub owner_username: String,
    pub allowed_host: String,
    pub assigned_databases: Vec<String>,
    pub created_at: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateDatabasePayload {
    pub name: String,
    pub engine: DatabaseEngine,
    pub collation: Option<String>,
    pub create_user: Option<bool>,
    pub db_username: Option<String>,
    pub db_password: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateDbUserPayload {
    pub username: String,
    pub password: String,
    pub engine: DatabaseEngine,
    pub allowed_host: Option<String>,
    pub assigned_databases: Vec<String>,
}
