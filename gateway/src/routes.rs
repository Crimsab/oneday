use crate::{assets, db, engine, events::TurnStreamEvent, AppState};
use axum::extract::{Path, State};
use axum::http::StatusCode;
use axum::response::sse::{Event, KeepAlive, Sse};
use axum::response::{IntoResponse, Response};
use axum::routing::{delete, get, post, put};
use axum::{Json, Router};
use serde_json::json;
use std::convert::Infallible;
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::broadcast::error::RecvError;
use tower_http::services::{ServeDir, ServeFile};

pub fn router(state: Arc<AppState>) -> Router {
    let static_dir = state.paths.static_dir.clone();
    let visual_asset_dir = state.paths.visual_asset_dir.clone();
    let spa =
        ServeDir::new(&static_dir).not_found_service(ServeFile::new(static_dir.join("index.html")));

    Router::new()
        .route("/api/health", get(health))
        .route(
            "/api/config/models",
            get(model_settings).put(update_model_settings),
        )
        .route("/api/contracts/commands", get(command_descriptors))
        .route("/api/stories", get(stories).post(create_story))
        .route("/api/stories/:story_id/snapshot", get(snapshot))
        .route("/api/stories/:story_id/visual-assets", get(visual_assets))
        .route(
            "/api/stories/:story_id/visual-assets/generate",
            post(generate_visual_assets),
        )
        .route(
            "/api/stories/:story_id/visual-profile",
            put(update_visual_profile),
        )
        .route("/api/stories/:story_id/actions", post(submit_action))
        .route("/api/stories/:story_id/meta", post(submit_meta))
        .route("/api/stories/:story_id/saves", post(create_save))
        .route(
            "/api/stories/:story_id/saves/:save_id/load",
            post(load_save),
        )
        .route("/api/stories/:story_id/saves/:save_id", delete(delete_save))
        .route("/api/stories/:story_id/events", get(story_events))
        .nest_service("/generated/assets", ServeDir::new(visual_asset_dir))
        .fallback_service(spa)
        .with_state(state)
}

async fn model_settings(
    State(state): State<Arc<AppState>>,
) -> Result<Json<engine::ModelRoutingSettings>, ApiError> {
    Ok(Json(engine::model_settings(state).await?))
}

async fn update_model_settings(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<engine::ModelRoutingUpdate>,
) -> Result<Json<engine::ModelRoutingSettings>, ApiError> {
    Ok(Json(engine::update_model_settings(state, payload).await?))
}

async fn command_descriptors(
    State(state): State<Arc<AppState>>,
) -> Result<Json<Vec<serde_json::Value>>, ApiError> {
    let descriptors = engine::command_descriptors(state.clone()).await?;
    Ok(Json(descriptors.commands))
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

async fn create_story(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<engine::StoryCreateEnvelope>,
) -> Result<Json<serde_json::Value>, ApiError> {
    let created = engine::create_story(state.clone(), payload).await?;
    let snapshot = db::snapshot(&state.pool, &created.story_id).await?;
    assets::spawn_auto_generation(state.clone(), created.story_id.clone());
    Ok(Json(json!({
        "story": created,
        "snapshot": snapshot,
    })))
}

async fn snapshot(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
) -> Result<Json<db::StorySnapshot>, ApiError> {
    Ok(Json(db::snapshot(&state.pool, &story_id).await?))
}

async fn visual_assets(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
) -> Result<Json<assets::VisualAssetsResponse>, ApiError> {
    Ok(Json(assets::visual_assets(&state.pool, &story_id).await?))
}

async fn update_visual_profile(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Json(payload): Json<assets::VisualProfileUpdate>,
) -> Result<Json<assets::VisualAssetsResponse>, ApiError> {
    Ok(Json(
        assets::update_profile(&state.pool, &story_id, payload).await?,
    ))
}

async fn generate_visual_assets(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Json(payload): Json<assets::GenerateVisualAssetsRequest>,
) -> Result<Json<assets::VisualAssetsResponse>, ApiError> {
    Ok(Json(
        assets::generate_visual_assets(state.as_ref(), &story_id, payload).await?,
    ))
}

async fn submit_action(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Json(payload): Json<engine::ActionEnvelope>,
) -> Result<Json<serde_json::Value>, ApiError> {
    let client_turn = payload.client_turn;
    let action_kind = payload.action.kind.clone();
    let action_text = payload.action.text.clone();
    emit_turn_stream(
        &state,
        TurnStreamEvent::status(
            &story_id,
            "submitted",
            client_turn,
            &action_kind,
            &action_text,
            "Action submitted to the live OneDay engine.",
        ),
    );

    let events = match engine::submit_action(state.clone(), &story_id, payload).await {
        Ok(events) => events,
        Err(err) => {
            emit_turn_stream(
                &state,
                TurnStreamEvent::status(
                    &story_id,
                    "failed",
                    client_turn,
                    &action_kind,
                    &action_text,
                    err.to_string(),
                ),
            );
            return Err(err.into());
        }
    };
    for event in &events.events {
        emit_turn_stream(
            &state,
            TurnStreamEvent::contract(&story_id, client_turn, &action_kind, &action_text, event),
        );
    }
    let snapshot = db::snapshot(&state.pool, &story_id).await?;
    assets::spawn_auto_generation(state.clone(), story_id.clone());
    emit_turn_stream(
        &state,
        TurnStreamEvent::status(
            &story_id,
            "completed",
            snapshot.version.turn,
            &action_kind,
            &action_text,
            "Browser and terminal state are synchronized.",
        ),
    );
    Ok(Json(json!({
        "events": events.events,
        "snapshot": snapshot,
    })))
}

async fn submit_meta(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Json(payload): Json<engine::MetaEnvelope>,
) -> Result<Json<serde_json::Value>, ApiError> {
    let meta = engine::submit_meta(state.clone(), &story_id, payload).await?;
    let snapshot = db::snapshot(&state.pool, &story_id).await?;
    Ok(Json(json!({
        "meta": meta.meta,
        "snapshot": snapshot,
    })))
}

async fn create_save(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Json(payload): Json<engine::SaveEnvelope>,
) -> Result<Json<serde_json::Value>, ApiError> {
    let save = engine::create_save(state.clone(), &story_id, payload).await?;
    let snapshot = db::snapshot(&state.pool, &story_id).await?;
    Ok(Json(json!({
        "save": save.save,
        "snapshot": snapshot,
    })))
}

async fn load_save(
    State(state): State<Arc<AppState>>,
    Path((story_id, save_id)): Path<(String, String)>,
    Json(mut payload): Json<engine::LoadEnvelope>,
) -> Result<Json<serde_json::Value>, ApiError> {
    payload.save_id = save_id;
    let load = engine::load_save(state.clone(), &story_id, payload).await?;
    let snapshot = db::snapshot(&state.pool, &story_id).await?;
    Ok(Json(json!({
        "save": load.save,
        "legacy": load.legacy,
        "snapshot": snapshot,
    })))
}

async fn delete_save(
    State(state): State<Arc<AppState>>,
    Path((story_id, save_id)): Path<(String, String)>,
    Json(mut payload): Json<engine::DeleteSaveEnvelope>,
) -> Result<Json<serde_json::Value>, ApiError> {
    payload.save_id = save_id;
    let deleted = engine::delete_save(state.clone(), &story_id, payload).await?;
    let snapshot = db::snapshot(&state.pool, &story_id).await?;
    Ok(Json(json!({
        "save": deleted.save,
        "snapshot": snapshot,
    })))
}

async fn story_events(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
) -> Sse<impl futures_core::Stream<Item = Result<Event, Infallible>>> {
    let stream = async_stream::stream! {
        let mut last_version = None;
        let mut turn_rx = state.turn_events.subscribe();
        if let Ok(snapshot) = db::snapshot(&state.pool, &story_id).await {
            last_version = Some(snapshot.version.clone());
            if let Ok(data) = serde_json::to_string(&snapshot) {
                yield Ok(Event::default().event("snapshot").data(data));
            }
        }

        let mut interval = tokio::time::interval(Duration::from_millis(750));
        loop {
            tokio::select! {
                _ = interval.tick() => {
                    match db::story_version(&state.pool, &story_id).await {
                        Ok(version) => {
                            if last_version.as_ref() == Some(&version) {
                                continue;
                            }
                            last_version = Some(version);
                            match db::snapshot(&state.pool, &story_id).await {
                                Ok(snapshot) => {
                                    if let Ok(data) = serde_json::to_string(&TurnStreamEvent::snapshot_changed(
                                        &story_id,
                                        snapshot.version.turn,
                                        snapshot.version.revision,
                                    )) {
                                        yield Ok(Event::default().event("turn").data(data));
                                    }
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
                },
                message = turn_rx.recv() => {
                    match message {
                        Ok(turn_event) => {
                            if turn_event.story_id != story_id {
                                continue;
                            }
                            if let Ok(data) = serde_json::to_string(&turn_event) {
                                yield Ok(Event::default().event("turn").data(data));
                            }
                        }
                        Err(RecvError::Lagged(skipped)) => {
                            if let Ok(data) = serde_json::to_string(&TurnStreamEvent::lagged(&story_id, skipped)) {
                                yield Ok(Event::default().event("turn").data(data));
                            }
                        }
                        Err(RecvError::Closed) => break,
                    }
                },
            }
        }
    };

    Sse::new(stream).keep_alive(
        KeepAlive::new()
            .interval(Duration::from_secs(10))
            .text("keepalive"),
    )
}

fn emit_turn_stream(state: &AppState, event: TurnStreamEvent) {
    let _ = state.turn_events.send(event);
}

#[derive(Debug)]
struct ApiError {
    status: StatusCode,
    message: String,
    code: Option<String>,
}

impl IntoResponse for ApiError {
    fn into_response(self) -> Response {
        let body = match self.code {
            Some(code) => json!({ "error": self.message, "code": code }),
            None => json!({ "error": self.message }),
        };
        (self.status, Json(body)).into_response()
    }
}

impl From<anyhow::Error> for ApiError {
    fn from(err: anyhow::Error) -> Self {
        if let Some(bridge) = err.downcast_ref::<engine::BridgeError>() {
            let status = match bridge.code.as_str() {
                "validation_failed" => StatusCode::BAD_REQUEST,
                "stale_config" | "config_locked" => StatusCode::CONFLICT,
                "write_failed" => StatusCode::INTERNAL_SERVER_ERROR,
                _ => StatusCode::INTERNAL_SERVER_ERROR,
            };
            return Self {
                status,
                message: bridge.message.clone(),
                code: Some(bridge.code.clone()),
            };
        }
        let message = err.to_string();
        let is_bad_request = message.contains("unknown provider")
            || message.contains("must be")
            || message.contains("at least one provider")
            || message.contains("image generation provider is not configured")
            || message.contains("story brief is required")
            || message.contains("character name is required")
            || message.contains("invalid gateway-model-settings-update JSON");
        let is_conflict = message.contains("stale client_turn")
            || message.contains("stale client_revision")
            || message.contains("stale session_id")
            || message.contains("turn idempotency key belongs to a different request")
            || message.contains("is required")
            || message.contains("belongs to story");
        let is_not_found =
            message.contains("no rows returned") || message.contains("no rows in result set");
        let status = if is_bad_request {
            StatusCode::BAD_REQUEST
        } else if is_conflict {
            StatusCode::CONFLICT
        } else if is_not_found {
            StatusCode::NOT_FOUND
        } else {
            StatusCode::INTERNAL_SERVER_ERROR
        };
        Self {
            status,
            message,
            code: None,
        }
    }
}

impl From<sqlx::Error> for ApiError {
    fn from(err: sqlx::Error) -> Self {
        Self {
            status: StatusCode::INTERNAL_SERVER_ERROR,
            message: err.to_string(),
            code: None,
        }
    }
}
