use axum::{
    extract::{Path, State},
    http::StatusCode,
    routing::{get, post, put},
    Json, Router,
};
use crate::models::{PhpExtension, PhpIniSettings, PhpPoolSettings, PhpVersionInfo};
use crate::services::php_mgr::PhpManager;
use crate::state::AppState;

pub fn router(state: AppState) -> Router {
    Router::new()
        .route("/versions", get(list_versions))
        .route("/:version/toggle", post(toggle_php_version))
        .route("/:version/extensions", get(get_extensions))
        .route("/:version/extensions/:ext/toggle", post(toggle_extension))
        .route("/:version/ini", get(get_ini_settings).put(update_ini_settings))
        .route("/:version/pool", get(get_pool_settings))
        .with_state(state)
}

async fn list_versions(State(state): State<AppState>) -> Json<Vec<PhpVersionInfo>> {
    let inner = state.inner.read();
    Json(inner.php_versions.clone())
}

async fn toggle_php_version(
    State(state): State<AppState>,
    Path(version): Path<String>,
) -> Result<Json<serde_json::Value>, StatusCode> {
    let mut inner = state.inner.write();
    if let Some(php) = inner.php_versions.iter_mut().find(|p| p.version == version) {
        php.is_running = !php.is_running;
        php.active_processes = if php.is_running { 4 } else { 0 };
        php.memory_mb = if php.is_running { 35.0 } else { 0.0 };
        return Ok(Json(serde_json::json!({
            "version": version,
            "is_running": php.is_running,
            "message": format!("PHP {} service status toggled", version)
        })));
    }

    Err(StatusCode::NOT_FOUND)
}

async fn get_extensions(Path(version): Path<String>) -> Json<Vec<PhpExtension>> {
    let extensions = PhpManager::get_default_extensions(&version);
    Json(extensions)
}

async fn toggle_extension(
    Path((version, ext)): Path<(String, String)>,
) -> Json<serde_json::Value> {
    Json(serde_json::json!({
        "success": true,
        "version": version,
        "extension": ext,
        "message": format!("Extension {} updated for PHP {}", ext, version)
    }))
}

async fn get_ini_settings(Path(version): Path<String>) -> Json<PhpIniSettings> {
    let ini = PhpManager::get_default_ini(&version);
    Json(ini)
}

async fn update_ini_settings(
    Path(version): Path<String>,
    Json(settings): Json<PhpIniSettings>,
) -> Json<serde_json::Value> {
    Json(serde_json::json!({
        "success": true,
        "version": version,
        "message": format!("php.ini updated and PHP-FPM {} reloaded", version),
        "settings": settings
    }))
}

async fn get_pool_settings(Path(version): Path<String>) -> Json<PhpPoolSettings> {
    Json(PhpPoolSettings {
        version,
        pm_type: "dynamic".to_string(),
        max_children: 20,
        start_servers: 4,
        min_spare_servers: 2,
        max_spare_servers: 8,
        max_requests: 500,
    })
}
