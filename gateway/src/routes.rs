use crate::{assets, db, engine, events::TurnStreamEvent, AppState};
use axum::extract::{Path, State};
use axum::http::StatusCode;
use axum::response::sse::{Event, KeepAlive, Sse};
use axum::response::{IntoResponse, Response};
use axum::routing::{delete, get, patch, post, put};
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
        .route("/api/story-wizard", post(story_wizard))
        .route("/api/story-enhance", post(story_enhance))
        .route("/api/stories", get(stories).post(create_story))
        .route(
            "/api/stories/:story_id",
            patch(update_story).delete(delete_story),
        )
        .route("/api/stories/:story_id/delete-plan", get(story_delete_plan))
        .route("/api/stories/:story_id/snapshot", get(snapshot))
        .route("/api/stories/:story_id/visual-assets", get(visual_assets))
        .route(
            "/api/stories/:story_id/visual-assets/:asset_id",
            put(update_visual_asset_prompt),
        )
        .route(
            "/api/stories/:story_id/visual-assets/:asset_id/versions",
            get(visual_asset_versions),
        )
        .route(
            "/api/stories/:story_id/visual-assets/:asset_id/versions/:version_id/select",
            post(select_visual_asset_version),
        )
        .route(
            "/api/stories/:story_id/visual-assets/generate",
            post(generate_visual_assets),
        )
        .route(
            "/api/stories/:story_id/visual-assets/jobs/:job_id/cancel",
            post(cancel_visual_generation_job),
        )
        .route(
            "/api/stories/:story_id/visual-assets/cleanup",
            post(cleanup_visual_asset_files),
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

async fn update_story(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Json(payload): Json<db::StoryUpdate>,
) -> Result<Json<db::StorySummary>, ApiError> {
    Ok(Json(
        db::update_story(&state.pool, &story_id, payload).await?,
    ))
}

async fn delete_story(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
) -> Result<Json<serde_json::Value>, ApiError> {
    let cancelled_visual_jobs = assets::cancel_story_visual_jobs(&state.pool, &story_id).await?;
    db::delete_story(&state.pool, &story_id).await?;
    Ok(Json(json!({
        "story_id": story_id,
        "cancelled_visual_jobs": cancelled_visual_jobs,
    })))
}

async fn story_delete_plan(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
) -> Result<Json<db::StoryDeletePlan>, ApiError> {
    Ok(Json(db::story_delete_plan(&state.pool, &story_id).await?))
}

async fn create_story(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<engine::StoryCreateEnvelope>,
) -> Result<Json<serde_json::Value>, ApiError> {
    let visual_profile = story_create_visual_profile(&payload);
    let created = engine::create_story(state.clone(), payload).await?;
    if let Some(profile) = visual_profile {
        assets::update_profile_with_defaults(&state.pool, &created.story_id, profile).await?;
    }
    let snapshot = db::snapshot(&state.pool, &created.story_id).await?;
    assets::spawn_auto_generation(state.clone(), created.story_id.clone());
    Ok(Json(json!({
        "story": created,
        "snapshot": snapshot,
    })))
}

async fn story_wizard(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<engine::StoryWizardEnvelope>,
) -> Result<Json<serde_json::Value>, ApiError> {
    let wizard = engine::story_wizard(state.clone(), payload).await?;
    let snapshot = if wizard.story_id.trim().is_empty() {
        None
    } else {
        Some(db::snapshot(&state.pool, &wizard.story_id).await?)
    };
    Ok(Json(json!({
        "wizard": wizard,
        "snapshot": snapshot,
    })))
}

async fn story_enhance(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<engine::StoryEnhanceEnvelope>,
) -> Result<Json<engine::GatewayStoryEnhanceResponse>, ApiError> {
    Ok(Json(engine::story_enhance(state, payload).await?))
}

fn story_create_visual_profile(
    payload: &engine::StoryCreateEnvelope,
) -> Option<assets::VisualProfileUpdate> {
    let update = assets::VisualProfileUpdate {
        world_style_prompt: payload.world_style_prompt.trim().to_string(),
        character_style_prompt: payload.character_style_prompt.trim().to_string(),
        negative_prompt: payload.negative_prompt.trim().to_string(),
        palette: payload.palette.trim().to_string(),
    };
    let has_visual_direction = !update.world_style_prompt.is_empty()
        || !update.character_style_prompt.is_empty()
        || !update.negative_prompt.is_empty()
        || !update.palette.is_empty();
    has_visual_direction.then_some(update)
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

async fn visual_asset_versions(
    State(state): State<Arc<AppState>>,
    Path((story_id, asset_id)): Path<(String, String)>,
) -> Result<Json<Vec<assets::VisualAssetVersion>>, ApiError> {
    Ok(Json(
        assets::visual_asset_versions(&state.pool, &story_id, &asset_id).await?,
    ))
}

async fn update_visual_asset_prompt(
    State(state): State<Arc<AppState>>,
    Path((story_id, asset_id)): Path<(String, String)>,
    Json(payload): Json<assets::VisualAssetPromptUpdate>,
) -> Result<Json<assets::VisualAssetsResponse>, ApiError> {
    Ok(Json(
        assets::update_asset_prompt(&state.pool, &story_id, &asset_id, payload).await?,
    ))
}

async fn select_visual_asset_version(
    State(state): State<Arc<AppState>>,
    Path((story_id, asset_id, version_id)): Path<(String, String, i64)>,
) -> Result<Json<assets::VisualAssetsResponse>, ApiError> {
    Ok(Json(
        assets::select_asset_version(&state.pool, &story_id, &asset_id, version_id).await?,
    ))
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
        assets::generate_visual_assets(state.clone(), &story_id, payload).await?,
    ))
}

async fn cancel_visual_generation_job(
    State(state): State<Arc<AppState>>,
    Path((story_id, job_id)): Path<(String, i64)>,
) -> Result<Json<assets::VisualAssetsResponse>, ApiError> {
    let response = assets::cancel_visual_generation_job(&state.pool, &story_id, job_id).await?;
    let asset_id = response
        .jobs
        .iter()
        .find(|job| job.id == job_id)
        .map(|job| job.asset_id.clone())
        .unwrap_or_default();
    emit_turn_stream(
        &state,
        TurnStreamEvent::visual_asset(
            &story_id,
            "asset.cancelled",
            &asset_id,
            Some(job_id),
            "cancelled",
            "Visual asset generation cancelled.",
        ),
    );
    Ok(Json(response))
}

async fn cleanup_visual_asset_files(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Json(payload): Json<assets::VisualAssetCleanupRequest>,
) -> Result<Json<assets::VisualAssetCleanupResponse>, ApiError> {
    Ok(Json(
        assets::cleanup_visual_asset_files(
            &state.pool,
            &story_id,
            &state.paths.visual_asset_dir,
            payload,
        )
        .await?,
    ))
}

async fn submit_action(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Json(payload): Json<engine::ActionEnvelope>,
) -> Result<Json<serde_json::Value>, ApiError> {
    let client_turn = payload.client_turn;
    let stream_requested = payload.stream;
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
    if !stream_requested {
        for event in &events.events {
            emit_turn_stream(
                &state,
                TurnStreamEvent::contract(
                    &story_id,
                    client_turn,
                    &action_kind,
                    &action_text,
                    event,
                ),
            );
        }
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
        "snapshot_state": load.snapshot_state,
        "snapshot_detail": load.snapshot_detail,
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
            if let Ok((version, data)) = serialize_snapshot(&snapshot) {
                last_version = Some(version);
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
                            match db::snapshot(&state.pool, &story_id).await {
                                Ok(snapshot) => {
                                    match serialize_snapshot_changed(&story_id, &snapshot) {
                                        Ok((snapshot_version, turn_data, snapshot_data)) => {
                                            yield Ok(Event::default().event("turn").data(turn_data));
                                            last_version = Some(snapshot_version);
                                            yield Ok(Event::default().event("snapshot").data(snapshot_data));
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

fn serialize_snapshot(
    snapshot: &db::StorySnapshot,
) -> Result<(db::StoryVersion, String), serde_json::Error> {
    let data = serde_json::to_string(snapshot)?;
    Ok((snapshot.version.clone(), data))
}

fn serialize_snapshot_changed(
    story_id: &str,
    snapshot: &db::StorySnapshot,
) -> Result<(db::StoryVersion, String, String), serde_json::Error> {
    let (version, snapshot_data) = serialize_snapshot(snapshot)?;
    let turn_data = serde_json::to_string(&TurnStreamEvent::snapshot_changed(
        story_id,
        snapshot.version.turn,
        snapshot.version.revision,
    ))?;
    Ok((version, turn_data, snapshot_data))
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
        let is_not_found = message.contains("no rows returned")
            || message.contains("no rows in result set")
            || message.contains("story not found")
            || message.contains("visual asset not found")
            || message.contains("visual asset version not found");
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

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn serializes_snapshot_with_the_version_that_sse_should_mark_seen() {
        let snapshot = story_snapshot();

        let (version, data) = serialize_snapshot(&snapshot).expect("snapshot serializes");

        assert_eq!(version, snapshot.version);
        assert!(data.contains("\"story-1\""));
    }

    #[test]
    fn serializes_snapshot_changed_turn_event_before_marking_snapshot_seen() {
        let snapshot = story_snapshot();

        let (version, turn_data, snapshot_data) =
            serialize_snapshot_changed("story-1", &snapshot).expect("snapshot change serializes");

        assert_eq!(version, snapshot.version);
        assert!(turn_data.contains("\"snapshot_changed\""));
        assert!(turn_data.contains("\"revision\":2"));
        assert!(snapshot_data.contains("\"revision\":2"));
    }

    fn story_snapshot() -> db::StorySnapshot {
        db::StorySnapshot {
            server_time: "2026-01-01T00:00:00Z".to_string(),
            version: db::StoryVersion {
                turn: 5,
                revision: 2,
                story_updated_at: "2026-01-01T00:00:00Z".to_string(),
                active_session_id: "session-1".to_string(),
                last_message_id: 10,
                world_updated_at: "2026-01-01T00:00:00Z".to_string(),
                character_updated_at: "2026-01-01T00:00:00Z".to_string(),
                npc_count: 0,
                npc_updated_at: String::new(),
                chapter_count: 0,
                achievement_count: 0,
                latest_achievement_at: String::new(),
                save_count: 0,
                latest_save_at: String::new(),
                visual_asset_updated_at: String::new(),
                visual_job_updated_at: String::new(),
                active_visual_job_count: 0,
            },
            story: db::StorySummary {
                id: "story-1".to_string(),
                name: "Story".to_string(),
                description: String::new(),
                genre: String::new(),
                tone: String::new(),
                language: "en".to_string(),
                is_archived: false,
                updated_at: "2026-01-01T00:00:00Z".to_string(),
            },
            character: db::RecordView {
                id: "character-1".to_string(),
                name: "Hero".to_string(),
                fields: json!({}),
            },
            world: db::WorldView {
                id: "world-1".to_string(),
                current_location: "Dock".to_string(),
                current_location_id: "dock".to_string(),
                spatial_locations: json!([]),
                spatial_edges: json!([]),
                world_time: json!({"day":0,"minute_of_day":0,"display_text":"Day 0, 00:00"}),
                weather: json!({"tracked":false,"label":"Not tracked"}),
                current_chapter: 1,
                current_turn: 5,
                known_locations: json!([]),
                global_events: json!([]),
                faction_standings: json!({}),
                story_hooks: json!([]),
                world_reactions: json!([]),
                investigations: json!([]),
                projects: json!([]),
                guidance: json!({}),
                fronts: json!([]),
                timeline: json!([]),
                scene_contract: json!({}),
                updated_at: "2026-01-01T00:00:00Z".to_string(),
            },
            active_session: db::SessionView {
                id: "session-1".to_string(),
                story_id: "story-1".to_string(),
                started_at: "2026-01-01T00:00:00Z".to_string(),
                ended_at: None,
                summary: String::new(),
            },
            choices: vec![],
            messages: vec![],
            panels: db::PanelsView {
                chapters: vec![],
                achievements: vec![],
                npcs: vec![],
                sessions: vec![],
                saves: vec![],
            },
        }
    }
}
