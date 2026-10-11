use axum::{
    extract::{Path, State},
    http::StatusCode,
    routing::{get, post},
    Json, Router,
};
use serde::{Deserialize, Serialize};
use crate::freebsd::service_rc::ServiceRc;
use crate::state::AppState;

#[derive(Deserialize)]
pub struct ServiceActionPayload {
    pub service_name: String,
    pub action: String, // start, stop, restart, reload, status
}

#[derive(Serialize)]
pub struct SystemServiceItem {
    pub name: String,
    pub description: String,
    pub is_running: bool,
    pub is_enabled_boot: bool,
}

pub fn router(state: AppState) -> Router {
    Router::new()
        .route("/services", get(list_system_services))
        .route("/services/action", post(execute_service_action))
        .route("/logs/:log_type", get(get_system_log))
        .with_state(state)
}

async fn list_system_services(State(state): State<AppState>) -> Json<Vec<SystemServiceItem>> {
    let mock = state.inner.read().config.mock_mode;
    let services = vec![
        SystemServiceItem {
            name: "nginx".to_string(),
            description: "High Performance HTTP & Reverse Proxy Server".to_string(),
            is_running: ServiceRc::is_service_running("nginx", mock),
            is_enabled_boot: true,
        },
        SystemServiceItem {
            name: "php-fpm-83".to_string(),
            description: "PHP 8.3 FastCGI Process Manager".to_string(),
            is_running: ServiceRc::is_service_running("php-fpm-83", mock),
            is_enabled_boot: true,
        },
        SystemServiceItem {
            name: "mysql-server".to_string(),
            description: "MariaDB SQL Database Daemon".to_string(),
            is_running: ServiceRc::is_service_running("mysql-server", mock),
            is_enabled_boot: true,
        },
        SystemServiceItem {
            name: "postgresql".to_string(),
            description: "PostgreSQL Database Engine".to_string(),
            is_running: ServiceRc::is_service_running("postgresql", mock),
            is_enabled_boot: true,
        },
        SystemServiceItem {
            name: "pf".to_string(),
            description: "FreeBSD Packet Filter Firewall Engine".to_string(),
            is_running: true,
            is_enabled_boot: true,
        },
        SystemServiceItem {
            name: "fail2ban".to_string(),
            description: "Automated Intrusion Prevention & Ban Service".to_string(),
            is_running: true,
            is_enabled_boot: true,
        },
        SystemServiceItem {
            name: "sshd".to_string(),
            description: "OpenSSH Secure Shell Daemon".to_string(),
            is_running: true,
            is_enabled_boot: true,
        },
    ];

    Json(services)
}

async fn execute_service_action(
    State(state): State<AppState>,
    Json(payload): Json<ServiceActionPayload>,
) -> Result<Json<serde_json::Value>, (StatusCode, Json<serde_json::Value>)> {
    let mock = state.inner.read().config.mock_mode;
    match ServiceRc::manage_service(&payload.service_name, &payload.action, mock) {
        Ok(out) => Ok(Json(serde_json::json!({
            "success": true,
            "message": out
        }))),
        Err(e) => Err((
            StatusCode::INTERNAL_SERVER_ERROR,
            Json(serde_json::json!({ "error": e })),
        )),
    }
}

async fn get_system_log(
    Path(log_type): Path<String>,
) -> Json<serde_json::Value> {
    let lines = match log_type.as_str() {
        "messages" => vec![
            "FreeBSD kernel: [vtnet0] link state changed to UP",
            "pf: Rule 14 pass in quick on vtnet0 proto tcp from 198.51.100.23 to any port 80",
            "bsdpaneld[1204]: Authentication attempt for root from 127.0.0.1: SUCCESS",
            "php-fpm[3812]: [pool example.com] child 3822 started",
            "syslogd: restart",
        ],
        "pf" => vec![
            "pf: block drop in on vtnet0: 203.0.113.88.42391 > 10.0.0.1.22: [S] (table <bruteforce>)",
            "pf: pass in on vtnet0: 172.56.21.9.51234 > 10.0.0.1.443: [S] keep state",
            "pf: pass out on vtnet0: 10.0.0.1.443 > 172.56.21.9.51234: [S.] keep state",
        ],
        "fail2ban" => vec![
            "fail2ban.filter [1450]: INFO [bsd-sshd] Found 198.51.100.23 - 2026-10-11 19:40:02",
            "fail2ban.actions [1450]: NOTICE [bsd-sshd] Ban 198.51.100.23",
            "fail2ban.actions [1450]: NOTICE [nginx-badbots] Ban 45.140.19.8",
        ],
        _ => vec!["[INFO] System log initialized."],
    };

    Json(serde_json::json!({
        "log_type": log_type,
        "lines": lines
    }))
}
