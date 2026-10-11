use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum FirewallAction {
    Pass,
    Block,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum RuleDirection {
    In,
    Out,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum RuleProtocol {
    Tcp,
    Udp,
    Both,
    Icmp,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FirewallRule {
    pub id: Uuid,
    pub label: String,
    pub port: String, // e.g. "80", "443", "8000-9000", "any"
    pub protocol: RuleProtocol,
    pub action: FirewallAction,
    pub direction: RuleDirection,
    pub source: String, // "any", "192.168.1.0/24", etc.
    pub is_enabled: bool,
    pub created_at: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Fail2banJailInfo {
    pub name: String,
    pub is_active: bool,
    pub filter: String,
    pub log_path: String,
    pub max_retry: u32,
    pub find_time: u32,
    pub ban_time: u32,
    pub currently_banned: u32,
    pub total_banned: u32,
    pub banned_ips: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BannedIpRecord {
    pub ip: String,
    pub jail: String,
    pub banned_at: i64,
    pub expires_at: i64,
    pub country: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FirewallStatus {
    pub pf_enabled: bool,
    pub pf_rules_count: usize,
    pub pf_states_count: u32,
    pub fail2ban_enabled: bool,
    pub total_banned_ips: usize,
    pub active_jails_count: usize,
}
