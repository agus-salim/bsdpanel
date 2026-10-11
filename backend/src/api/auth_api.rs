use axum::{
    extract::State,
    http::StatusCode,
    response::IntoResponse,
    routing::{get, post},
    Json, Router,
};
use serde::{Deserialize, Serialize};
use crate::auth::{create_token, verify_token};
use crate::models::{User, UserRole};
use crate::state::AppState;

#[derive(Deserialize)]
pub struct LoginPayload {
    pub username: String,
    pub password: Option<String>,
}

#[derive(Serialize)]
pub struct LoginResponse {
    pub token: String,
    pub user: User,
}

pub fn router(state: AppState) -> Router {
    Router::new()
        .route("/login", post(login_handler))
        .route("/me", get(me_handler))
        .with_state(state)
}

async fn login_handler(
    State(state): State<AppState>,
    Json(payload): Json<LoginPayload>,
) -> Result<Json<LoginResponse>, (StatusCode, Json<serde_json::Value>)> {
    let inner = state.inner.read();
    let user = inner.users.iter().find(|u| u.username == payload.username);

    match user {
        Some(u) => {
            let secret = &inner.config.jwt_secret;
            let token = create_token(&u.username, u.id, u.role.clone(), secret)
                .map_err(|e| (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({ "error": e }))))?;

            Ok(Json(LoginResponse {
                token,
                user: u.clone(),
            }))
        }
        None => Err((
            StatusCode::UNAUTHORIZED,
            Json(serde_json::json!({ "error": "Invalid username or password" })),
        )),
    }
}

async fn me_handler(
    State(state): State<AppState>,
) -> Json<serde_json::Value> {
    let inner = state.inner.read();
    // Default to the first admin user
    let user = inner.users.first().cloned();
    Json(serde_json::json!({ "user": user }))
}
