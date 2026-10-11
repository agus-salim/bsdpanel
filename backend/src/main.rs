mod api;
mod auth;
mod config;
mod freebsd;
mod models;
mod services;
mod state;

use std::net::SocketAddr;
use std::path::{Path, PathBuf};
use tower_http::cors::{Any, CorsLayer};
use tower_http::services::{ServeDir, ServeFile};
use tracing::{info, Level};
use tracing_subscriber::FmtSubscriber;

use config::AppConfig;
use state::AppState;

fn find_ui_dir() -> PathBuf {
    let candidates = [
        "/usr/local/etc/bsdpanel/ui",
        "./frontend/dist",
        "../frontend/dist",
        "./dist",
    ];

    for path in &candidates {
        let p = Path::new(path);
        if p.exists() && p.join("index.html").exists() {
            info!("Found UI static directory: {:?}", p);
            return p.to_path_buf();
        }
    }

    info!("UI directory not found in candidates, defaulting to /usr/local/etc/bsdpanel/ui");
    PathBuf::from("/usr/local/etc/bsdpanel/ui")
}

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

    let ui_dir = find_ui_dir();
    let index_file = ui_dir.join("index.html");

    // SPA routing: static file server with fallback to index.html
    let app = if index_file.exists() {
        info!("Enabling SPA web server serving {:?}", ui_dir);
        let serve_dir = ServeDir::new(&ui_dir).fallback(ServeFile::new(&index_file));
        api_router
            .fallback_service(serve_dir)
            .layer(cors)
    } else {
        info!("index.html not found in {:?}, API only mode", ui_dir);
        api_router.layer(cors)
    };

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
