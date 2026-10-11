use axum::{
    extract::{Path, State},
    http::StatusCode,
    routing::{delete, get, post, put},
    Json, Router,
};
use uuid::Uuid;
use crate::models::{CreateDomainPayload, Domain, UpdateDomainPayload, WebServerInfo, WebServerType};
use crate::services::webserver_mgr::WebServerManager;
use crate::state::AppState;

pub fn router(state: AppState) -> Router {
    Router::new()
        .route("/", get(list_webservers))
        .route("/:server_type/toggle", post(toggle_webserver))
        .route("/:server_type/restart", post(restart_webserver))
        .route("/domains", get(list_domains).post(create_domain))
        .route("/domains/:id", put(update_domain).delete(delete_domain))
        .route("/domains/:id/preview", get(preview_vhost))
        .with_state(state)
}

async fn list_webservers(State(state): State<AppState>) -> Json<Vec<WebServerInfo>> {
    let inner = state.inner.read();
    Json(inner.webservers.clone())
}

async fn toggle_webserver(
    State(state): State<AppState>,
    Path(server_type): Path<String>,
) -> Result<Json<serde_json::Value>, StatusCode> {
    let mut inner = state.inner.write();
    let s_type = match server_type.to_lowercase().as_str() {
        "nginx" => WebServerType::Nginx,
        "apache" => WebServerType::Apache,
        "litespeed" => WebServerType::LiteSpeed,
        "caddy" => WebServerType::Caddy,
        _ => return Err(StatusCode::BAD_REQUEST),
    };

    if let Some(srv) = inner.webservers.iter_mut().find(|w| w.server_type == s_type) {
        srv.is_running = !srv.is_running;
        srv.memory_mb = if srv.is_running { 54.0 } else { 0.0 };
        return Ok(Json(serde_json::json!({
            "success": true,
            "server_type": server_type,
            "is_running": srv.is_running
        })));
    }

    Err(StatusCode::NOT_FOUND)
}

async fn restart_webserver(
    State(state): State<AppState>,
    Path(server_type): Path<String>,
) -> Result<Json<serde_json::Value>, StatusCode> {
    let inner = state.inner.read();
    let s_type = match server_type.to_lowercase().as_str() {
        "nginx" => WebServerType::Nginx,
        "apache" => WebServerType::Apache,
        "litespeed" => WebServerType::LiteSpeed,
        "caddy" => WebServerType::Caddy,
        _ => return Err(StatusCode::BAD_REQUEST),
    };

    if inner.webservers.iter().any(|w| w.server_type == s_type) {
        return Ok(Json(serde_json::json!({
            "success": true,
            "message": format!("Service {} reloaded successfully via FreeBSD rc.d", server_type)
        })));
    }

    Err(StatusCode::NOT_FOUND)
}

async fn list_domains(State(state): State<AppState>) -> Json<Vec<Domain>> {
    let inner = state.inner.read();
    Json(inner.domains.clone())
}

async fn create_domain(
    State(state): State<AppState>,
    Json(payload): Json<CreateDomainPayload>,
) -> Result<Json<Domain>, StatusCode> {
    let mut inner = state.inner.write();
    let doc_root = format!("/usr/local/www/{}/public", payload.domain_name);

    let domain = Domain {
        id: Uuid::new_v4(),
        domain_name: payload.domain_name,
        owner_username: "client_company".to_string(),
        webserver_type: payload.webserver_type,
        php_version: payload.php_version,
        document_root: doc_root,
        ssl_enabled: payload.ssl_auto_cert,
        force_https: payload.force_https,
        is_active: true,
        reverse_proxy_target: payload.reverse_proxy_target,
        created_at: chrono::Utc::now().timestamp(),
    };

    inner.domains.push(domain.clone());
    Ok(Json(domain))
}

async fn update_domain(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(payload): Json<UpdateDomainPayload>,
) -> Result<Json<Domain>, StatusCode> {
    let mut inner = state.inner.write();
    if let Some(domain) = inner.domains.iter_mut().find(|d| d.id == id) {
        if let Some(ws) = payload.webserver_type {
            domain.webserver_type = ws;
        }
        if let Some(php) = payload.php_version {
            domain.php_version = php;
        }
        if let Some(force_https) = payload.force_https {
            domain.force_https = force_https;
        }
        if let Some(ssl) = payload.ssl_enabled {
            domain.ssl_enabled = ssl;
        }
        if let Some(active) = payload.is_active {
            domain.is_active = active;
        }
        if let Some(proxy) = payload.reverse_proxy_target {
            domain.reverse_proxy_target = Some(proxy);
        }
        return Ok(Json(domain.clone()));
    }

    Err(StatusCode::NOT_FOUND)
}

async fn delete_domain(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
) -> Result<Json<serde_json::Value>, StatusCode> {
    let mut inner = state.inner.write();
    let len_before = inner.domains.len();
    inner.domains.retain(|d| d.id != id);

    if inner.domains.len() < len_before {
        Ok(Json(serde_json::json!({ "success": true })))
    } else {
        Err(StatusCode::NOT_FOUND)
    }
}

async fn preview_vhost(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
) -> Result<Json<serde_json::Value>, StatusCode> {
    let inner = state.inner.read();
    if let Some(domain) = inner.domains.iter().find(|d| d.id == id) {
        let clean_ver = domain.php_version.replace('.', "");
        let php_sock = format!("/var/run/php-fpm-{}.sock", clean_ver);

        let content = match domain.webserver_type {
            WebServerType::Nginx => WebServerManager::generate_nginx_vhost(domain, &php_sock),
            WebServerType::Apache => WebServerManager::generate_apache_vhost(domain, 9000),
            WebServerType::Caddy => WebServerManager::generate_caddy_vhost(domain, &php_sock),
            WebServerType::LiteSpeed => WebServerManager::generate_litespeed_vhost(domain, &php_sock),
        };

        return Ok(Json(serde_json::json!({
            "domain": domain.domain_name,
            "webserver": domain.webserver_type,
            "config": content
        })));
    }

    Err(StatusCode::NOT_FOUND)
}
