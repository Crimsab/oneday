mod config;
mod db;
mod engine;
mod routes;

use anyhow::Context;
use axum::Router;
use sqlx::sqlite::{SqliteConnectOptions, SqliteJournalMode, SqlitePoolOptions};
use std::net::SocketAddr;
use std::sync::Arc;
use tracing_subscriber::EnvFilter;

#[derive(Clone)]
pub struct AppState {
    pub pool: sqlx::SqlitePool,
    pub paths: config::ResolvedPaths,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::try_from_default_env().unwrap_or_else(|_| "info".into()))
        .init();

    let args = config::Args::parse_args();
    let paths = config::resolve_paths(&args).context("resolving OneDay gateway paths")?;
    let addr: SocketAddr = args.addr.parse().context("parsing --addr")?;

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

    let state = Arc::new(AppState { pool, paths });
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

async fn shutdown_signal() {
    let _ = tokio::signal::ctrl_c().await;
}
