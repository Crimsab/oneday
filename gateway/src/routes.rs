use crate::{db, engine, AppState};
use axum::extract::{Path, State};
use axum::http::StatusCode;
use axum::response::sse::{Event, KeepAlive, Sse};
use axum::response::{IntoResponse, Response};
use axum::routing::{get, post};
use axum::{Json, Router};
use serde_json::json;
use std::convert::Infallible;
use std::sync::Arc;
use std::time::Duration;
use tower_http::services::{ServeDir, ServeFile};

pub fn router(state: Arc<AppState>) -> Router {
    let static_dir = state.paths.static_dir.clone();
    let spa =
        ServeDir::new(&static_dir).not_found_service(ServeFile::new(static_dir.join("index.html")));

    Router::new()
        .route("/api/health", get(health))
        .route("/api/stories", get(stories))
        .route("/api/stories/:story_id/snapshot", get(snapshot))
        .route("/api/stories/:story_id/actions", post(submit_action))
        .route("/api/stories/:story_id/events", get(story_events))
        .fallback_service(spa)
        .with_state(state)
}

async fn health(State(state): State<Arc<AppState>>) -> Result<Json<serde_json::Value>, ApiError> {
    let story_count: i64 = sqlx::query_scalar("SELECT COUNT(*) FROM stories")
        .fetch_one(&state.pool)
        .await?;
    Ok(Json(json!({
        "status": "ok",
        "stories": story_count,
        "db_path": state.paths.db_path,
        "config_path": state.paths.config_path,
        "oneday_bin": state.paths.oneday_bin,
        "static_dir": state.paths.static_dir,
    })))
}

async fn stories(
    State(state): State<Arc<AppState>>,
) -> Result<Json<Vec<db::StorySummary>>, ApiError> {
    Ok(Json(db::list_stories(&state.pool).await?))
}

async fn snapshot(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
) -> Result<Json<db::StorySnapshot>, ApiError> {
    Ok(Json(db::snapshot(&state.pool, &story_id).await?))
}

async fn submit_action(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Json(payload): Json<engine::ActionEnvelope>,
) -> Result<Json<serde_json::Value>, ApiError> {
    let events = engine::submit_action(state.clone(), &story_id, payload).await?;
    let snapshot = db::snapshot(&state.pool, &story_id).await?;
    Ok(Json(json!({
        "events": events.events,
        "snapshot": snapshot,
    })))
}

async fn story_events(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
) -> Sse<impl futures_core::Stream<Item = Result<Event, Infallible>>> {
    let stream = async_stream::stream! {
        let mut last_version = None;
        if let Ok(snapshot) = db::snapshot(&state.pool, &story_id).await {
            last_version = Some(snapshot.version.clone());
            if let Ok(data) = serde_json::to_string(&snapshot) {
                yield Ok(Event::default().event("snapshot").data(data));
            }
        }

        let mut interval = tokio::time::interval(Duration::from_millis(750));
        loop {
            interval.tick().await;
            match db::story_version(&state.pool, &story_id).await {
                Ok(version) => {
                    if last_version.as_ref() == Some(&version) {
                        continue;
                    }
                    last_version = Some(version);
                    match db::snapshot(&state.pool, &story_id).await {
                        Ok(snapshot) => {
                            if let Ok(data) = serde_json::to_string(&snapshot) {
                                yield Ok(Event::default().event("snapshot").data(data));
                            }
                        }
                        Err(err) => {
                            yield Ok(Event::default().event("error").data(err.to_string()));
                        }
                    }
                }
                Err(err) => {
                    yield Ok(Event::default().event("error").data(err.to_string()));
                }
            }
        }
    };

    Sse::new(stream).keep_alive(
        KeepAlive::new()
            .interval(Duration::from_secs(10))
            .text("keepalive"),
    )
}

#[derive(Debug)]
struct ApiError {
    status: StatusCode,
    message: String,
}

impl IntoResponse for ApiError {
    fn into_response(self) -> Response {
        (self.status, Json(json!({ "error": self.message }))).into_response()
    }
}

impl From<anyhow::Error> for ApiError {
    fn from(err: anyhow::Error) -> Self {
        let message = err.to_string();
        let status = if message.contains("stale client_turn")
            || message.contains("stale session_id")
            || message.contains("is required")
        {
            StatusCode::CONFLICT
        } else if message.contains("no rows returned") {
            StatusCode::NOT_FOUND
        } else {
            StatusCode::INTERNAL_SERVER_ERROR
        };
        Self { status, message }
    }
}

impl From<sqlx::Error> for ApiError {
    fn from(err: sqlx::Error) -> Self {
        Self {
            status: StatusCode::INTERNAL_SERVER_ERROR,
            message: err.to_string(),
        }
    }
}
