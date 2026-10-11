pub mod auth_api;
pub mod dashboard_api;
pub mod webservers_api;
pub mod php_api;
pub mod databases_api;
pub mod firewall_api;
pub mod users_api;
pub mod system_api;

use axum::Router;
use crate::state::AppState;

pub fn create_api_router(state: AppState) -> Router {
    Router::new()
        .nest("/api/auth", auth_api::router(state.clone()))
        .nest("/api/dashboard", dashboard_api::router(state.clone()))
        .nest("/api/webservers", webservers_api::router(state.clone()))
        .nest("/api/php", php_api::router(state.clone()))
        .nest("/api/databases", databases_api::router(state.clone()))
        .nest("/api/firewall", firewall_api::router(state.clone()))
        .nest("/api/users", users_api::router(state.clone()))
        .nest("/api/system", system_api::router(state))
}
