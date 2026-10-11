use axum::{
    extract::State,
    routing::get,
    Json, Router,
};
use serde::Serialize;
use crate::freebsd::system_info::{SystemInfoProvider, SystemStats};
use crate::state::AppState;

#[derive(Serialize)]
pub struct DashboardOverview {
    pub stats: SystemStats,
    pub total_domains: usize,
    pub active_domains: usize,
    pub total_databases: usize,
    pub total_users: usize,
    pub active_webservers: usize,
    pub running_php_pools: usize,
    pub pf_status: String,
    pub banned_ips_count: usize,
}

pub fn router(state: AppState) -> Router {
    Router::new()
        .route("/overview", get(get_overview))
        .route("/stats", get(get_stats))
        .with_state(state)
}

async fn get_stats(State(state): State<AppState>) -> Json<SystemStats> {
    let mock_mode = state.inner.read().config.mock_mode;
    let stats = SystemInfoProvider::get_stats(mock_mode);
    Json(stats)
}

async fn get_overview(State(state): State<AppState>) -> Json<DashboardOverview> {
    let inner = state.inner.read();
    let stats = SystemInfoProvider::get_stats(inner.config.mock_mode);

    let total_domains = inner.domains.len();
    let active_domains = inner.domains.iter().filter(|d| d.is_active).count();
    let total_databases = inner.databases.len();
    let total_users = inner.users.len();
    let active_webservers = inner.webservers.iter().filter(|w| w.is_running).count();
    let running_php_pools = inner.php_versions.iter().filter(|p| p.is_running).count();
    let pf_status = if inner.firewall_rules.iter().any(|r| r.is_enabled) { "Active (Enforcing)" } else { "Disabled" };
    let banned_ips_count = inner.banned_ips.len();

    Json(DashboardOverview {
        stats,
        total_domains,
        active_domains,
        total_databases,
        total_users,
        active_webservers,
        running_php_pools,
        pf_status: pf_status.to_string(),
        banned_ips_count,
    })
}
