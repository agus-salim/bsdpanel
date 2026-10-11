use axum::{
    extract::{Path, State},
    http::StatusCode,
    routing::{delete, get, post},
    Json, Router,
};
use serde::Deserialize;
use uuid::Uuid;
use crate::freebsd::pf_firewall::PfFirewall;
use crate::models::{BannedIpRecord, Fail2banJailInfo, FirewallAction, FirewallRule, FirewallStatus, RuleDirection, RuleProtocol};
use crate::state::AppState;

#[derive(Deserialize)]
pub struct AddRulePayload {
    pub label: String,
    pub port: String,
    pub protocol: RuleProtocol,
    pub action: FirewallAction,
    pub direction: RuleDirection,
    pub source: Option<String>,
}

#[derive(Deserialize)]
pub struct BanActionPayload {
    pub ip: String,
    pub jail: String,
}

pub fn router(state: AppState) -> Router {
    Router::new()
        .route("/status", get(get_status))
        .route("/toggle", post(toggle_firewall))
        .route("/rules", get(list_rules).post(add_rule))
        .route("/rules/:id", delete(delete_rule))
        .route("/jails", get(list_jails))
        .route("/banned-ips", get(list_banned_ips))
        .route("/unban", post(unban_ip))
        .route("/ban", post(ban_ip))
        .route("/pf-conf", get(get_pf_conf))
        .with_state(state)
}

async fn get_status(State(state): State<AppState>) -> Json<FirewallStatus> {
    let inner = state.inner.read();
    let pf_enabled = PfFirewall::is_enabled(inner.config.mock_mode);
    let pf_states_count = PfFirewall::get_states_count(inner.config.mock_mode);

    Json(FirewallStatus {
        pf_enabled,
        pf_rules_count: inner.firewall_rules.len(),
        pf_states_count,
        fail2ban_enabled: true,
        total_banned_ips: inner.banned_ips.len(),
        active_jails_count: inner.fail2ban_jails.len(),
    })
}

async fn toggle_firewall(State(state): State<AppState>) -> Json<serde_json::Value> {
    let inner = state.inner.read();
    let current = PfFirewall::is_enabled(inner.config.mock_mode);
    let _ = PfFirewall::toggle_pf(!current, inner.config.mock_mode);
    Json(serde_json::json!({
        "success": true,
        "is_enabled": !current
    }))
}

async fn list_rules(State(state): State<AppState>) -> Json<Vec<FirewallRule>> {
    let inner = state.inner.read();
    Json(inner.firewall_rules.clone())
}

async fn add_rule(
    State(state): State<AppState>,
    Json(payload): Json<AddRulePayload>,
) -> Result<Json<FirewallRule>, StatusCode> {
    let mut inner = state.inner.write();
    let rule = FirewallRule {
        id: Uuid::new_v4(),
        label: payload.label,
        port: payload.port,
        protocol: payload.protocol,
        action: payload.action,
        direction: payload.direction,
        source: payload.source.unwrap_or_else(|| "any".to_string()),
        is_enabled: true,
        created_at: chrono::Utc::now().timestamp(),
    };

    inner.firewall_rules.push(rule.clone());
    Ok(Json(rule))
}

async fn delete_rule(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
) -> Result<Json<serde_json::Value>, StatusCode> {
    let mut inner = state.inner.write();
    let before = inner.firewall_rules.len();
    inner.firewall_rules.retain(|r| r.id != id);

    if inner.firewall_rules.len() < before {
        Ok(Json(serde_json::json!({ "success": true })))
    } else {
        Err(StatusCode::NOT_FOUND)
    }
}

async fn list_jails(State(state): State<AppState>) -> Json<Vec<Fail2banJailInfo>> {
    let inner = state.inner.read();
    Json(inner.fail2ban_jails.clone())
}

async fn list_banned_ips(State(state): State<AppState>) -> Json<Vec<BannedIpRecord>> {
    let inner = state.inner.read();
    Json(inner.banned_ips.clone())
}

async fn unban_ip(
    State(state): State<AppState>,
    Json(payload): Json<BanActionPayload>,
) -> Json<serde_json::Value> {
    let mut inner = state.inner.write();
    inner.banned_ips.retain(|b| !(b.ip == payload.ip && b.jail == payload.jail));
    Json(serde_json::json!({
        "success": true,
        "message": format!("IP {} unbanned from jail {}", payload.ip, payload.jail)
    }))
}

async fn ban_ip(
    State(state): State<AppState>,
    Json(payload): Json<BanActionPayload>,
) -> Json<serde_json::Value> {
    let mut inner = state.inner.write();
    let now = chrono::Utc::now().timestamp();
    let record = BannedIpRecord {
        ip: payload.ip.clone(),
        jail: payload.jail.clone(),
        banned_at: now,
        expires_at: now + 3600,
        country: Some("MANUAL".to_string()),
    };
    inner.banned_ips.push(record);
    Json(serde_json::json!({
        "success": true,
        "message": format!("IP {} banned in jail {}", payload.ip, payload.jail)
    }))
}

async fn get_pf_conf(State(state): State<AppState>) -> Json<serde_json::Value> {
    let inner = state.inner.read();
    let conf = PfFirewall::generate_pf_conf(&inner.firewall_rules);
    Json(serde_json::json!({
        "path": "/etc/pf.conf",
        "content": conf
    }))
}
