mod assets;
mod challenge;
mod config;
mod db;
mod engine;
mod events;
mod routes;
mod telemetry;

use anyhow::Context;
use axum::Router;
use sqlx::sqlite::{SqliteConnectOptions, SqliteJournalMode, SqlitePoolOptions};
use std::net::SocketAddr;
use std::process::Command;
use std::sync::Arc;
use tokio::sync::{broadcast, Mutex};
use tracing_subscriber::EnvFilter;

#[derive(Clone)]
pub struct AppState {
    pub pool: sqlx::SqlitePool,
    pub paths: config::ResolvedPaths,
    pub turn_events: broadcast::Sender<events::TurnStreamEvent>,
    pub visual_worker: Arc<Mutex<()>>,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::try_from_default_env().unwrap_or_else(|_| "info".into()))
        .init();

    let args = config::Args::parse_args();
    let paths = config::resolve_paths(&args).context("resolving OneDay gateway paths")?;
    let addr: SocketAddr = args.addr.parse().context("parsing --addr")?;
    run_schema_preflight(&paths).context("running OneDay schema preflight")?;
    std::fs::create_dir_all(&paths.visual_asset_dir).with_context(|| {
        format!(
            "creating visual asset directory {}",
            paths.visual_asset_dir.display()
        )
    })?;

    let db_options = SqliteConnectOptions::new()
        .filename(&paths.db_path)
        .create_if_missing(false)
        .journal_mode(SqliteJournalMode::Wal)
        .foreign_keys(true);
    let pool = SqlitePoolOptions::new()
        .max_connections(8)
        .connect_with(db_options)
        .await
        .with_context(|| format!("opening SQLite database {}", paths.db_path.display()))?;
    assets::ensure_visual_asset_version_schema(&pool)
        .await
        .context("ensuring visual asset version schema")?;
    if let Err(err) = telemetry::prune_expired(&pool).await {
        tracing::warn!(error = %err, "generation telemetry retention pass failed");
    }

    let (turn_events, _) = broadcast::channel(128);
    let state = Arc::new(AppState {
        pool,
        paths,
        turn_events,
        visual_worker: Arc::new(Mutex::new(())),
    });
    assets::spawn_visual_generation_maintenance(state.clone());
    assets::spawn_visual_generation_worker(state.clone());
    let app: Router = routes::router(state);
    let listener = tokio::net::TcpListener::bind(addr)
        .await
        .with_context(|| format!("binding {addr}"))?;

    tracing::info!("OneDay gateway listening on http://{addr}");
    axum::serve(listener, app)
        .with_graceful_shutdown(shutdown_signal())
        .await
        .context("serving gateway")?;
    Ok(())
}

fn run_schema_preflight(paths: &config::ResolvedPaths) -> anyhow::Result<()> {
    let output = Command::new(&paths.oneday_bin)
        .arg("gateway-schema-preflight")
        .env("ONEDAY_CONFIG", &paths.config_path)
        .current_dir(&paths.oneday_root)
        .output()
        .with_context(|| format!("starting {}", paths.oneday_bin.display()))?;
    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        return Err(anyhow::anyhow!(
            "schema preflight failed with status {}: {}",
            output.status,
            stderr.trim()
        ));
    }
    Ok(())
}

async fn shutdown_signal() {
    let _ = tokio::signal::ctrl_c().await;
}
