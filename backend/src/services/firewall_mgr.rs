use crate::freebsd::fail2ban::Fail2banMgr;
use crate::freebsd::pf_firewall::PfFirewall;
use crate::models::{
    BannedIpRecord, Fail2banJailInfo, FirewallAction, FirewallRule, FirewallStatus, RuleDirection,
    RuleProtocol,
};

pub struct FirewallManager;

impl FirewallManager {
    pub fn get_default_rules() -> Vec<FirewallRule> {
        let now = chrono::Utc::now().timestamp();
        vec![
            FirewallRule {
                id: uuid::Uuid::new_v4(),
                label: "SSH Remote Shell Access".to_string(),
                port: "22".to_string(),
                protocol: RuleProtocol::Tcp,
                action: FirewallAction::Pass,
                direction: RuleDirection::In,
                source: "any".to_string(),
                is_enabled: true,
                created_at: now,
            },
            FirewallRule {
                id: uuid::Uuid::new_v4(),
                label: "HTTP Web Traffic".to_string(),
                port: "80".to_string(),
                protocol: RuleProtocol::Tcp,
                action: FirewallAction::Pass,
                direction: RuleDirection::In,
                source: "any".to_string(),
                is_enabled: true,
                created_at: now,
            },
            FirewallRule {
                id: uuid::Uuid::new_v4(),
                label: "HTTPS Secure Web Traffic".to_string(),
                port: "443".to_string(),
                protocol: RuleProtocol::Tcp,
                action: FirewallAction::Pass,
                direction: RuleDirection::In,
                source: "any".to_string(),
                is_enabled: true,
                created_at: now,
            },
            FirewallRule {
                id: uuid::Uuid::new_v4(),
                label: "BSD Panel Control Port".to_string(),
                port: "8888".to_string(),
                protocol: RuleProtocol::Tcp,
                action: FirewallAction::Pass,
                direction: RuleDirection::In,
                source: "any".to_string(),
                is_enabled: true,
                created_at: now,
            },
            FirewallRule {
                id: uuid::Uuid::new_v4(),
                label: "MariaDB MySQL Access".to_string(),
                port: "3306".to_string(),
                protocol: RuleProtocol::Tcp,
                action: FirewallAction::Block,
                direction: RuleDirection::In,
                source: "any".to_string(),
                is_enabled: true,
                created_at: now,
            },
            FirewallRule {
                id: uuid::Uuid::new_v4(),
                label: "PostgreSQL Database Access".to_string(),
                port: "5432".to_string(),
                protocol: RuleProtocol::Tcp,
                action: FirewallAction::Block,
                direction: RuleDirection::In,
                source: "any".to_string(),
                is_enabled: true,
                created_at: now,
            },
        ]
    }

    pub fn get_default_jails() -> Vec<Fail2banJailInfo> {
        vec![
            Fail2banJailInfo {
                name: "bsd-sshd".to_string(),
                is_active: true,
                filter: "sshd".to_string(),
                log_path: "/var/log/auth.log".to_string(),
                max_retry: 4,
                find_time: 600,
                ban_time: 3600,
                currently_banned: 3,
                total_banned: 48,
                banned_ips: vec![
                    "198.51.100.23".to_string(),
                    "203.0.113.88".to_string(),
                    "185.220.101.5".to_string(),
                ],
            },
            Fail2banJailInfo {
                name: "bsd-panel-auth".to_string(),
                is_active: true,
                filter: "bsd-panel".to_string(),
                log_path: "/var/log/bsdpanel/access.log".to_string(),
                max_retry: 5,
                find_time: 300,
                ban_time: 7200,
                currently_banned: 1,
                total_banned: 12,
                banned_ips: vec!["192.0.2.14".to_string()],
            },
            Fail2banJailInfo {
                name: "nginx-badbots".to_string(),
                is_active: true,
                filter: "apache-badbots".to_string(),
                log_path: "/var/log/nginx/error.log".to_string(),
                max_retry: 2,
                find_time: 600,
                ban_time: 86400,
                currently_banned: 2,
                total_banned: 104,
                banned_ips: vec!["45.140.19.8".to_string(), "91.240.118.52".to_string()],
            },
        ]
    }
}
