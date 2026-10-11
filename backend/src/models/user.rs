use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum UserRole {
    Admin,
    Reseller,
    User,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum AccountStatus {
    Active,
    Suspended,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct User {
    pub id: Uuid,
    pub username: String,
    #[serde(skip_serializing)]
    pub password_hash: String,
    pub role: UserRole,
    pub reseller_id: Option<Uuid>,
    pub email: String,
    pub disk_limit_mb: u64,
    pub disk_used_mb: u64,
    pub bandwidth_limit_mb: u64,
    pub bandwidth_used_mb: u64,
    pub domains_limit: u32,
    pub databases_limit: u32,
    pub status: AccountStatus,
    pub created_at: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateUserPayload {
    pub username: String,
    pub password: Option<String>,
    pub role: UserRole,
    pub email: String,
    pub disk_limit_mb: u64,
    pub bandwidth_limit_mb: u64,
    pub domains_limit: u32,
    pub databases_limit: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateUserPayload {
    pub email: Option<String>,
    pub password: Option<String>,
    pub status: Option<AccountStatus>,
    pub disk_limit_mb: Option<u64>,
    pub bandwidth_limit_mb: Option<u64>,
    pub domains_limit: Option<u32>,
    pub databases_limit: Option<u32>,
}
