mod api;
mod auth;
mod config;
mod freebsd;
mod models;
mod services;
mod state;

use std::net::SocketAddr;
use tower_http::cors::{Any, CorsLayer};
use tower_http::services::ServeDir;
use tracing::{info, Level};
use tracing_subscriber::FmtSubscriber;

use config::AppConfig;
use state::AppState;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // 1. Initialize logging
    let subscriber = FmtSubscriber::builder()
        .with_max_level(Level::INFO)
        .finish();
    tracing::subscriber::set_global_default(subscriber)
        .expect("setting default subscriber failed");

    info!("======================================================");
    info!("   BSD Panel - Lightweight & Powerful FreeBSD Panel   ");
    info!("   Target OS: FreeBSD | Backend: Rust | UI: Vue 3     ");
    info!("======================================================");

    // 2. Load Configuration
    let config = AppConfig::default();
    info!("Mock Mode Active: {}", config.mock_mode);
    info!("Data Directory: {:?}", config.data_dir);

    // 3. Initialize Shared Application State
    let app_state = AppState::new(config.clone());

    // 4. Configure CORS
    let cors = CorsLayer::new()
        .allow_origin(Any)
        .allow_methods(Any)
        .allow_headers(Any);

    // 5. Build Router
    let api_router = api::create_api_router(app_state);

    // Optional static SPA file server fallback
    let app = api_router
        .nest_service("/dist", ServeDir::new("./frontend/dist"))
        .nest_service("/assets", ServeDir::new("./frontend/dist/assets"))
        .layer(cors);

    // 6. Bind listener
    let addr = SocketAddr::new(
        config.bind_addr.parse().unwrap_or([0, 0, 0, 0].into()),
        config.port,
    );
    info!("Server listening on http://{}", addr);
    info!("Web Panel UI available at http://{}:{}", config.bind_addr, config.port);

    let listener = tokio::net::TcpListener::bind(addr).await?;
    axum::serve(listener, app).await?;

    Ok(())
}
