use axum::{
    extract::{Path, State},
    http::StatusCode,
    routing::{delete, get, post, put},
    Json, Router,
};
use uuid::Uuid;
use crate::models::{AccountStatus, CreateUserPayload, UpdateUserPayload, User, UserRole};
use crate::state::AppState;

pub fn router(state: AppState) -> Router {
    Router::new()
        .route("/", get(list_users).post(create_user))
        .route("/:id", put(update_user).delete(delete_user))
        .route("/resellers", get(list_resellers))
        .with_state(state)
}

async fn list_users(State(state): State<AppState>) -> Json<Vec<User>> {
    let inner = state.inner.read();
    Json(inner.users.clone())
}

async fn list_resellers(State(state): State<AppState>) -> Json<Vec<User>> {
    let inner = state.inner.read();
    let resellers: Vec<User> = inner
        .users
        .iter()
        .filter(|u| u.role == UserRole::Reseller)
        .cloned()
        .collect();
    Json(resellers)
}

async fn create_user(
    State(state): State<AppState>,
    Json(payload): Json<CreateUserPayload>,
) -> Result<Json<User>, StatusCode> {
    let mut inner = state.inner.write();

    if inner.users.iter().any(|u| u.username == payload.username) {
        return Err(StatusCode::CONFLICT);
    }

    let user = User {
        id: Uuid::new_v4(),
        username: payload.username,
        password_hash: "$argon2id$v=19$m=19456,t=2,p=1$mockhash".to_string(),
        role: payload.role,
        reseller_id: None,
        email: payload.email,
        disk_limit_mb: payload.disk_limit_mb,
        disk_used_mb: 0,
        bandwidth_limit_mb: payload.bandwidth_limit_mb,
        bandwidth_used_mb: 0,
        domains_limit: payload.domains_limit,
        databases_limit: payload.databases_limit,
        status: AccountStatus::Active,
        created_at: chrono::Utc::now().timestamp(),
    };

    inner.users.push(user.clone());
    Ok(Json(user))
}

async fn update_user(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
    Json(payload): Json<UpdateUserPayload>,
) -> Result<Json<User>, StatusCode> {
    let mut inner = state.inner.write();
    if let Some(user) = inner.users.iter_mut().find(|u| u.id == id) {
        if let Some(email) = payload.email {
            user.email = email;
        }
        if let Some(status) = payload.status {
            user.status = status;
        }
        if let Some(disk) = payload.disk_limit_mb {
            user.disk_limit_mb = disk;
        }
        if let Some(bw) = payload.bandwidth_limit_mb {
            user.bandwidth_limit_mb = bw;
        }
        if let Some(dom) = payload.domains_limit {
            user.domains_limit = dom;
        }
        if let Some(db) = payload.databases_limit {
            user.databases_limit = db;
        }
        return Ok(Json(user.clone()));
    }

    Err(StatusCode::NOT_FOUND)
}

async fn delete_user(
    State(state): State<AppState>,
    Path(id): Path<Uuid>,
) -> Result<Json<serde_json::Value>, StatusCode> {
    let mut inner = state.inner.write();
    let before = inner.users.len();
    inner.users.retain(|u| u.id != id);

    if inner.users.len() < before {
        Ok(Json(serde_json::json!({ "success": true })))
    } else {
        Err(StatusCode::NOT_FOUND)
    }
}
