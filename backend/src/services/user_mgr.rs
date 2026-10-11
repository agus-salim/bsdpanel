use uuid::Uuid;
use crate::models::{AccountStatus, User, UserRole};

pub struct UserManager;

impl UserManager {
    pub fn get_initial_seed_users() -> Vec<User> {
        let now = chrono::Utc::now().timestamp();
        vec![
            User {
                id: Uuid::new_v4(),
                username: "root".to_string(),
                password_hash: "$argon2id$v=19$m=19456,t=2,p=1$abcdefg$bsdpanelsecretroot".to_string(),
                role: UserRole::Admin,
                reseller_id: None,
                email: "admin@bsdpanel.lan".to_string(),
                disk_limit_mb: 0, // 0 = unlimited
                disk_used_mb: 2450,
                bandwidth_limit_mb: 0,
                bandwidth_used_mb: 18200,
                domains_limit: 0,
                databases_limit: 0,
                status: AccountStatus::Active,
                created_at: now,
            },
            User {
                id: Uuid::new_v4(),
                username: "reseller_asia".to_string(),
                password_hash: "$argon2id$v=19$m=19456,t=2,p=1$abcdefg$resellerpass".to_string(),
                role: UserRole::Reseller,
                reseller_id: None,
                email: "reseller@asiahost.id".to_string(),
                disk_limit_mb: 102400, // 100 GB
                disk_used_mb: 14200,
                bandwidth_limit_mb: 1048576, // 1 TB
                bandwidth_used_mb: 85400,
                domains_limit: 50,
                databases_limit: 100,
                status: AccountStatus::Active,
                created_at: now,
            },
            User {
                id: Uuid::new_v4(),
                username: "client_company".to_string(),
                password_hash: "$argon2id$v=19$m=19456,t=2,p=1$abcdefg$clientpass".to_string(),
                role: UserRole::User,
                reseller_id: None,
                email: "dev@company.co.id".to_string(),
                disk_limit_mb: 10240, // 10 GB
                disk_used_mb: 3120,
                bandwidth_limit_mb: 102400, // 100 GB
                bandwidth_used_mb: 12400,
                domains_limit: 5,
                databases_limit: 10,
                status: AccountStatus::Active,
                created_at: now,
            },
        ]
    }
}
