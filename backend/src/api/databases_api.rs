use axum::{
    extract::{Path, State},
    http::StatusCode,
    routing::{delete, get, post},
    Json, Router,
};
use uuid::Uuid;
use crate::models::{
    CreateDatabasePayload, CreateDbUserPayload, DatabaseEngine, DatabaseItem, DatabaseServerInfo,
    DatabaseUser,
};
use crate::services::db_mgr::DbManager;
use crate::state::AppState;

pub fn router(state: AppState) -> Router {
    Router::new()
        .route("/servers", get(list_servers))
        .route("/items", get(list_databases).post(create_database))
        .route("/items/:id", delete(delete_database))
        .route("/users", get(list_db_users).post(create_db_user))
        .route("/users/:id", delete(delete_db_user))
        .with_state(state)
}

async fn list_servers(State(state): State<AppState>) -> Json<Vec<DatabaseServerInfo>> {
    let mock = state.inner.read().config.mock_mode;
    let servers = DbManager::get_servers_info(mock);
    Json(servers)
}

async fn list_databases(State(state): State<AppState>) -> Json<Vec<DatabaseItem>> {
    let inner = state.inner.read();
    Json(inner.databases.clone())
}

async fn create_database(
    State(state): State<AppState>,
    Json(payload): Json<CreateDatabasePayload>,
) -> Result<Json<DatabaseItem>, StatusCode> {
    let mut inner = state.inner.write();
    let collation = payload.collation.unwrap_or_else(|| {
        match payload.engine {
            DatabaseEngine::Mariadb => "utf8mb4_unicode_ci".to_string(),
            DatabaseEngine::Postgresql => "en_US.UTF-8".to_string(),
        }
    });

    let db = DatabaseItem {
        id: Uuid::new_v4(),
        name: payload.name,
        engine: payload.engine,
        owner_username: "client_company".to_string(),
        size_mb: 0.1,
        tables_count: 0,
        collation,
        created_at: chrono::Utc::now().timestamp(),
    };

    inner.databases.push(db.clone());
    Ok(Json(db))
}

async fn delete_database(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
) -> Result<Json<serde_json::Value>, StatusCode> {
    let mut inner = state.inner.write();
    let before = inner.databases.len();
    inner.databases.retain(|d| d.id != id);

    if inner.databases.len() < before {
        Ok(Json(serde_json::json!({ "success": true })))
    } else {
        Err(StatusCode::NOT_FOUND)
    }
}

async fn list_db_users(State(state): State<AppState>) -> Json<Vec<DatabaseUser>> {
    let inner = state.inner.read();
    Json(inner.db_users.clone())
}

async fn create_db_user(
    State(state): State<AppState>,
    Json(payload): Json<CreateDbUserPayload>,
) -> Result<Json<DatabaseUser>, StatusCode> {
    let mut inner = state.inner.write();
    let user = DatabaseUser {
        id: Uuid::new_v4(),
        username: payload.username,
        engine: payload.engine,
        owner_username: "client_company".to_string(),
        allowed_host: payload.allowed_host.unwrap_or_else(|| "localhost".to_string()),
        assigned_databases: payload.assigned_databases,
        created_at: chrono::Utc::now().timestamp(),
    };

    inner.db_users.push(user.clone());
    Ok(Json(user))
}

async fn delete_db_user(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
) -> Result<Json<serde_json::Value>, StatusCode> {
    let mut inner = state.inner.write();
    let before = inner.db_users.len();
    inner.db_users.retain(|u| u.id != id);

    if inner.db_users.len() < before {
        Ok(Json(serde_json::json!({ "success": true })))
    } else {
        Err(StatusCode::NOT_FOUND)
    }
}
