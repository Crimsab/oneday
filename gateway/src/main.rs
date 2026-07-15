mod assets;
mod challenge;
mod config;
mod db;
mod engine;
mod events;
mod imagegen;
#[allow(dead_code, clippy::derivable_impls)]
mod gateway_protocol {
    include!(concat!(env!("OUT_DIR"), "/gateway_protocol.rs"));
}
mod routes;
mod telemetry;

use anyhow::Context;
use axum::body::Body;
use axum::http::{header, HeaderName, HeaderValue, Request};
use axum::middleware::{self, Next};
use axum::response::Response;
use axum::Router;
use sqlx::sqlite::{SqliteConnectOptions, SqliteJournalMode, SqlitePoolOptions};
use std::net::SocketAddr;
use std::process::Command;
use std::sync::Arc;
use tokio::sync::{broadcast, Semaphore};
use tower_http::compression::CompressionLayer;
use tower_http::request_id::{MakeRequestUuid, PropagateRequestIdLayer, SetRequestIdLayer};
use tower_http::trace::{DefaultOnResponse, TraceLayer};
use tower_http::LatencyUnit;
use tracing::Level;
use tracing_subscriber::EnvFilter;

#[derive(Clone)]
pub struct AppState {
    pub pool: sqlx::SqlitePool,
    pub paths: config::ResolvedPaths,
    pub turn_events: broadcast::Sender<events::TurnStreamEvent>,
    pub visual_workers: Arc<Semaphore>,
}

tokio::task_local! {
    static HTTP_REQUEST_ID: String;
}

pub fn current_request_id() -> Option<String> {
    HTTP_REQUEST_ID.try_with(Clone::clone).ok()
}

async fn scope_request_id(request: Request<Body>, next: Next) -> Response {
    let request_id = request
        .headers()
        .get("x-request-id")
        .and_then(|value| value.to_str().ok())
        .unwrap_or("-")
        .to_string();
    HTTP_REQUEST_ID.scope(request_id, next.run(request)).await
}

async fn static_cache_headers(request: Request<Body>, next: Next) -> Response {
    let policy = cache_control_for_path(request.uri().path());
    let mut response = next.run(request).await;
    if response.status().is_success() {
        if let Some(policy) = policy {
            response
                .headers_mut()
                .insert(header::CACHE_CONTROL, HeaderValue::from_static(policy));
        }
    }
    response
}

fn cache_control_for_path(path: &str) -> Option<&'static str> {
    if path.starts_with("/assets/") {
        return Some("public, max-age=31536000, immutable");
    }
    if !path.starts_with("/api/") && !path.starts_with("/generated/") {
        return Some("no-cache");
    }
    None
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::try_from_default_env().unwrap_or_else(|_| "info".into()))
        .init();

    let args = config::Args::parse_args();
    let paths = config::resolve_paths(&args).context("resolving OneDay gateway paths")?;
    let addr: SocketAddr = args.addr.parse().context("parsing --addr")?;
    if let Some(parent) = paths.db_path.parent() {
        std::fs::create_dir_all(parent)
            .with_context(|| format!("creating data directory {}", parent.display()))?;
    }
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
        visual_workers: Arc::new(Semaphore::new(4)),
    });
    assets::spawn_visual_generation_maintenance(state.clone());
    assets::spawn_visual_generation_worker(state.clone());
    assets::spawn_automatic_visual_catchup(state.clone());
    let request_id_header = HeaderName::from_static("x-request-id");
    let trace_request_id_header = request_id_header.clone();
    let app: Router = routes::router(state)
        .layer(CompressionLayer::new())
        .layer(middleware::from_fn(static_cache_headers))
        .layer(
            TraceLayer::new_for_http()
                .make_span_with(move |request: &Request<Body>| {
                    let request_id = request
                        .headers()
                        .get(&trace_request_id_header)
                        .and_then(|value| value.to_str().ok())
                        .unwrap_or("-");
                    tracing::info_span!(
                        "http_request",
                        request_id,
                        method = %request.method(),
                        uri = %request.uri(),
                    )
                })
                .on_response(
                    DefaultOnResponse::new()
                        .level(Level::INFO)
                        .latency_unit(LatencyUnit::Millis),
                ),
        )
        .layer(middleware::from_fn(scope_request_id))
        .layer(PropagateRequestIdLayer::new(request_id_header.clone()))
        .layer(SetRequestIdLayer::new(request_id_header, MakeRequestUuid));
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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn cache_policy_only_marks_fingerprinted_assets_immutable() {
        assert_eq!(
            cache_control_for_path("/assets/index-abc123.js"),
            Some("public, max-age=31536000, immutable")
        );
        assert_eq!(cache_control_for_path("/"), Some("no-cache"));
        assert_eq!(cache_control_for_path("/story/deep-link"), Some("no-cache"));
        assert_eq!(cache_control_for_path("/api/stories"), None);
        assert_eq!(cache_control_for_path("/generated/assets/image.png"), None);
    }
}
