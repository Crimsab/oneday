use crate::{
    assets, db, engine, error::PublicError, events::TurnStreamEvent, gateway_protocol as protocol,
    portability, telemetry, translation, AppState,
};
use axum::body::Body;
use axum::extract::{DefaultBodyLimit, Multipart, Path, Query, State};
use axum::http::{header, HeaderMap, HeaderValue, StatusCode};
use axum::response::sse::{Event, KeepAlive, Sse};
use axum::response::{IntoResponse, Response};
use axum::routing::{delete, get, patch, post, put};
use axum::{Json, Router};
use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
use serde_json::json;
use std::convert::Infallible;
use std::sync::Arc;
use std::time::Duration;
use tokio::io::AsyncReadExt;
use tokio::sync::broadcast::error::RecvError;
use tokio_util::io::ReaderStream;
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
            "/api/stories/import",
            post(import_story_archive).layer(DefaultBodyLimit::max(520 * 1024 * 1024)),
        )
        .route(
            "/api/stories/import-template",
            post(import_world_template).layer(DefaultBodyLimit::max(4 * 1024 * 1024)),
        )
        .route(
            "/api/stories/:story_id",
            patch(update_story).delete(delete_story),
        )
        .route("/api/stories/:story_id/delete-plan", get(story_delete_plan))
        .route("/api/stories/:story_id/overview", get(story_overview))
        .route("/api/stories/:story_id/snapshot", get(snapshot))
        .route(
            "/api/stories/:story_id/timeline",
            get(timeline).post(update_timeline),
        )
        .route("/api/stories/:story_id/history", get(history))
        .route("/api/stories/:story_id/agency-events", get(agency_events))
        .route("/api/stories/:story_id/chapters", get(chapters))
        .route("/api/stories/:story_id/export", get(export_story))
        .route("/api/stories/:story_id/archive", post(export_story_archive))
        .route(
            "/api/stories/:story_id/world-template",
            get(export_world_template),
        )
        .route(
            "/api/stories/:story_id/translations/jobs",
            get(translation_jobs).post(create_translation_job),
        )
        .route(
            "/api/stories/:story_id/translations/jobs/estimate",
            post(estimate_translation_job),
        )
        .route(
            "/api/stories/:story_id/translations/jobs/:job_id",
            get(translation_job).delete(delete_translation_job),
        )
        .route(
            "/api/stories/:story_id/translations/jobs/:job_id/:action",
            post(translation_job_action),
        )
        .route(
            "/api/stories/:story_id/translations/jobs/:job_id/browser-next",
            get(next_browser_translation_item),
        )
        .route(
            "/api/stories/:story_id/translations/jobs/:job_id/items/:item_id",
            post(complete_browser_translation_item),
        )
        .route(
            "/api/stories/:story_id/translations/glossary",
            get(translation_glossary).post(create_translation_glossary),
        )
        .route(
            "/api/stories/:story_id/translations/glossary/:entry_id",
            put(update_translation_glossary).delete(delete_translation_glossary),
        )
        .route(
            "/api/stories/:story_id/messages/:message_id/diagnostics",
            get(message_diagnostics),
        )
        .route(
            "/api/stories/:story_id/telemetry/export",
            get(export_telemetry),
        )
        .route("/api/stories/:story_id/visual-assets", get(visual_assets))
        .route(
            "/api/stories/:story_id/visual-assets/upload",
            post(upload_new_visual_asset).layer(DefaultBodyLimit::max(
                crate::asset_upload::MAX_UPLOAD_REQUEST_BYTES,
            )),
        )
        .route(
            "/api/stories/:story_id/minigames",
            get(active_minigame).post(start_minigame),
        )
        .route(
            "/api/stories/:story_id/minigames/:instance_id/input",
            post(input_minigame),
        )
        .route("/api/tts/providers", get(tts_catalog))
        .route("/api/tts/voices", get(tts_catalog))
        .route(
            "/api/stories/:story_id/tts/settings",
            get(tts_settings).put(update_tts_settings),
        )
        .route(
            "/api/stories/:story_id/voice-assignments",
            get(voice_assignments),
        )
        .route(
            "/api/stories/:story_id/voice-assignments/:assignment_id",
            put(update_voice_assignment),
        )
        .route(
            "/api/stories/:story_id/messages/:message_id/audio",
            get(message_audio).post(create_message_audio),
        )
        .route(
            "/api/stories/:story_id/audio/jobs/:job_id/:action",
            post(audio_job_action),
        )
        .route("/api/stories/:story_id/pronunciations", get(pronunciations))
        .route(
            "/api/stories/:story_id/pronunciations/:pronunciation_id",
            put(update_pronunciation).delete(delete_pronunciation),
        )
        .route("/api/stories/:story_id/audio/cleanup", post(cleanup_audio))
        .route("/api/stories/:story_id/audio/export", get(export_audio))
        .route("/api/audio/:audio_asset_id", get(audio_asset))
        .route(
            "/api/stories/:story_id/visual-assets/:asset_id",
            put(update_visual_asset_prompt),
        )
        .route(
            "/api/stories/:story_id/visual-assets/:asset_id/versions",
            get(visual_asset_versions),
        )
        .route(
            "/api/stories/:story_id/visual-assets/:asset_id/versions/upload",
            post(upload_visual_asset_version).layer(DefaultBodyLimit::max(
                crate::asset_upload::MAX_UPLOAD_REQUEST_BYTES,
            )),
        )
        .route(
            "/api/stories/:story_id/visual-assets/:asset_id/versions/:version_id/select",
            post(select_visual_asset_version),
        )
        .route(
            "/api/stories/:story_id/visual-assets/:asset_id/selection/:action",
            post(step_visual_asset_selection),
        )
        .route(
            "/api/stories/:story_id/visual-assets/generate",
            post(generate_visual_assets),
        )
        .route(
            "/api/stories/:story_id/visual-assets/:asset_id/operations",
            post(create_image_operation),
        )
        .route(
            "/api/stories/:story_id/image-operations/:operation_id",
            get(image_operation),
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
        .route("/api/stories/:story_id/craft", post(craft))
        .route("/api/stories/:story_id/actions", post(submit_action))
        .route("/api/stories/:story_id/meta", post(submit_meta))
        .route(
            "/api/stories/:story_id/saves",
            get(story_saves).post(create_save),
        )
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
) -> Result<Json<protocol::ModelRoutingSettings>, ApiError> {
    Ok(Json(engine::model_settings(state).await?))
}

async fn update_model_settings(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<protocol::ModelRoutingUpdate>,
) -> Result<Json<protocol::ModelRoutingSettings>, ApiError> {
    Ok(Json(engine::update_model_settings(state, payload).await?))
}

async fn command_descriptors(
    State(state): State<Arc<AppState>>,
    Query(query): Query<CommandDescriptorQuery>,
) -> Result<Json<Vec<protocol::CommandDescriptor>>, ApiError> {
    let locale = normalize_interface_locale(query.locale.as_deref());
    let descriptors = engine::command_descriptors(state.clone(), locale).await?;
    Ok(Json(descriptors.commands))
}

#[derive(Debug, Default, serde::Deserialize)]
struct CommandDescriptorQuery {
    locale: Option<String>,
}

fn normalize_interface_locale(locale: Option<&str>) -> &'static str {
    match locale
        .unwrap_or_default()
        .trim()
        .split(['-', '_'])
        .next()
        .unwrap_or_default()
        .to_ascii_lowercase()
        .as_str()
    {
        "it" => "it",
        _ => "en",
    }
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
        "observability": {
            "otlp_traces": state.observability.otlp_traces,
        },
    })))
}

async fn stories(
    State(state): State<Arc<AppState>>,
) -> Result<Json<Vec<db::StorySummary>>, ApiError> {
    Ok(Json(db::list_stories(&state.pool).await?))
}

#[derive(Debug, Default, serde::Deserialize)]
struct LimitQuery {
    limit: Option<i64>,
}

async fn agency_events(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Query(query): Query<LimitQuery>,
) -> Result<Json<Vec<db::AgencyEventView>>, ApiError> {
    Ok(Json(
        db::agency_events(&state.pool, &story_id, query.limit.unwrap_or(20)).await?,
    ))
}

async fn start_minigame(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Json(payload): Json<engine::MiniGameStartEnvelope>,
) -> Result<Json<protocol::MiniGameResponse>, ApiError> {
    Ok(Json(
        engine::start_minigame(state, &story_id, payload).await?,
    ))
}

async fn active_minigame(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
) -> Result<Json<protocol::MiniGameResponse>, ApiError> {
    Ok(Json(engine::active_minigame(state, &story_id).await?))
}

async fn input_minigame(
    State(state): State<Arc<AppState>>,
    Path((story_id, instance_id)): Path<(String, String)>,
    Json(payload): Json<engine::MiniGameInputEnvelope>,
) -> Result<Json<protocol::MiniGameResponse>, ApiError> {
    Ok(Json(
        engine::input_minigame(state, &story_id, &instance_id, payload).await?,
    ))
}

#[derive(Debug, Default, serde::Deserialize)]
struct TTSCatalogQuery {
    #[serde(default)]
    provider: String,
    #[serde(default)]
    language: String,
}

fn audio_request(value: serde_json::Value) -> anyhow::Result<protocol::AudioRequest> {
    serde_json::from_value(value).map_err(|error| {
        anyhow::anyhow!("decoding audio request against gateway contract: {error}")
    })
}

async fn tts_catalog(
    State(state): State<Arc<AppState>>,
    Query(query): Query<TTSCatalogQuery>,
) -> Result<Json<protocol::AudioResponse>, ApiError> {
    Ok(Json(
        engine::audio(
            state,
            audio_request(json!({
                "operation": "catalog", "provider": query.provider, "language": query.language
            }))?,
        )
        .await?,
    ))
}

async fn tts_settings(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
) -> Result<Json<protocol::AudioResponse>, ApiError> {
    Ok(Json(
        engine::audio(
            state,
            audio_request(json!({"operation":"settings-get","story_id":story_id}))?,
        )
        .await?,
    ))
}

async fn update_tts_settings(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Json(mut payload): Json<serde_json::Value>,
) -> Result<Json<protocol::AudioResponse>, ApiError> {
    let revision = payload
        .get("client_revision")
        .and_then(|value| value.as_i64())
        .unwrap_or(-1);
    if let Some(object) = payload.as_object_mut() {
        object.remove("client_revision");
    }
    Ok(Json(engine::audio(state, audio_request(json!({"operation":"settings-put","story_id":story_id,"client_revision":revision,"settings":payload}))?).await?))
}

async fn voice_assignments(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
) -> Result<Json<protocol::AudioResponse>, ApiError> {
    Ok(Json(
        engine::audio(
            state,
            audio_request(json!({"operation":"assignments-get","story_id":story_id}))?,
        )
        .await?,
    ))
}

async fn update_voice_assignment(
    State(state): State<Arc<AppState>>,
    Path((story_id, assignment_id)): Path<(String, String)>,
    Json(mut payload): Json<serde_json::Value>,
) -> Result<Json<protocol::AudioResponse>, ApiError> {
    let revision = payload
        .get("client_revision")
        .and_then(|value| value.as_i64())
        .unwrap_or(-1);
    if let Some(object) = payload.as_object_mut() {
        object.remove("client_revision");
    }
    Ok(Json(engine::audio(state, audio_request(json!({"operation":"assignment-put","story_id":story_id,"assignment_id":assignment_id,"client_revision":revision,"assignment":payload}))?).await?))
}

async fn message_audio(
    State(state): State<Arc<AppState>>,
    Path((story_id, message_id)): Path<(String, i64)>,
) -> Result<Json<protocol::AudioResponse>, ApiError> {
    Ok(Json(
        engine::audio(
            state,
            audio_request(
                json!({"operation":"message-get","story_id":story_id,"message_id":message_id}),
            )?,
        )
        .await?,
    ))
}

async fn create_message_audio(
    State(state): State<Arc<AppState>>,
    Path((story_id, message_id)): Path<(String, i64)>,
) -> Result<Json<protocol::AudioResponse>, ApiError> {
    Ok(Json(
        engine::audio(
            state,
            audio_request(
                json!({"operation":"message-create","story_id":story_id,"message_id":message_id}),
            )?,
        )
        .await?,
    ))
}

async fn audio_job_action(
    State(state): State<Arc<AppState>>,
    Path((story_id, job_id, action)): Path<(String, String, String)>,
) -> Result<Json<protocol::AudioResponse>, ApiError> {
    if action != "retry" && action != "cancel" {
        return Err(ApiError::public(PublicError::bad_request(
            "invalid_audio_job_action",
            "audio job action must be retry or cancel",
        )));
    }
    Ok(Json(
        engine::audio(
            state,
            audio_request(json!({
                "operation": format!("job-{action}"), "story_id": story_id, "job_id": job_id
            }))?,
        )
        .await?,
    ))
}

async fn pronunciations(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Query(query): Query<TTSCatalogQuery>,
) -> Result<Json<protocol::AudioResponse>, ApiError> {
    Ok(Json(
        engine::audio(
            state,
            audio_request(json!({"operation":"pronunciations-get","story_id":story_id,"language":query.language}))?,
        )
        .await?,
    ))
}

async fn update_pronunciation(
    State(state): State<Arc<AppState>>,
    Path((story_id, pronunciation_id)): Path<(String, String)>,
    Json(mut payload): Json<serde_json::Value>,
) -> Result<Json<protocol::AudioResponse>, ApiError> {
    let revision = payload
        .get("client_revision")
        .and_then(|value| value.as_i64())
        .unwrap_or(-1);
    if let Some(object) = payload.as_object_mut() {
        object.remove("client_revision");
    }
    Ok(Json(engine::audio(state, audio_request(json!({"operation":"pronunciation-put","story_id":story_id,"pronunciation_id":pronunciation_id,"client_revision":revision,"pronunciation":payload}))?).await?))
}

#[derive(Debug, Default, serde::Deserialize)]
struct RevisionQuery {
    #[serde(default)]
    client_revision: i64,
}

async fn delete_pronunciation(
    State(state): State<Arc<AppState>>,
    Path((story_id, pronunciation_id)): Path<(String, String)>,
    Query(query): Query<RevisionQuery>,
) -> Result<Json<protocol::AudioResponse>, ApiError> {
    Ok(Json(engine::audio(state, audio_request(json!({"operation":"pronunciation-delete","story_id":story_id,"pronunciation_id":pronunciation_id,"client_revision":query.client_revision}))?).await?))
}

async fn cleanup_audio(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Json(payload): Json<serde_json::Value>,
) -> Result<Json<protocol::AudioResponse>, ApiError> {
    Ok(Json(engine::audio(state, audio_request(json!({"operation":"cleanup","story_id":story_id,"dry_run":payload.get("dry_run").and_then(|value|value.as_bool()).unwrap_or(true)}))?).await?))
}

async fn export_audio(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
) -> Result<Json<protocol::AudioResponse>, ApiError> {
    Ok(Json(
        engine::audio(
            state,
            audio_request(json!({"operation":"export","story_id":story_id}))?,
        )
        .await?,
    ))
}

async fn audio_asset(
    State(state): State<Arc<AppState>>,
    Path(audio_asset_id): Path<String>,
) -> Result<Response, ApiError> {
    let response = engine::audio(
        state,
        audio_request(json!({"operation":"asset-path","asset_id":audio_asset_id}))?,
    )
    .await?;
    let path = response
        .file_path
        .ok_or_else(|| anyhow::anyhow!("gateway-audio returned no asset path"))?;
    let metadata = tokio::fs::metadata(&path)
        .await
        .map_err(|err| anyhow::anyhow!(err))?;
    validate_audio_asset_size(metadata.len())?;
    let file = tokio::fs::File::open(path)
        .await
        .map_err(|err| anyhow::anyhow!(err))?;
    let stream = ReaderStream::new(file.take(metadata.len()));
    let content_type = match response.format.as_deref() {
        Some("wav") => "audio/wav",
        Some("opus") => "audio/ogg",
        Some("aac") => "audio/aac",
        _ => "audio/mpeg",
    };
    Ok(Response::builder()
        .status(StatusCode::OK)
        .header(header::CONTENT_TYPE, content_type)
        .header(
            header::CACHE_CONTROL,
            "private, max-age=31536000, immutable",
        )
        .header(header::X_CONTENT_TYPE_OPTIONS, "nosniff")
        .header(header::CONTENT_LENGTH, metadata.len())
        .body(Body::from_stream(stream))
        .map_err(|err| anyhow::anyhow!(err))?)
}

const MAX_AUDIO_ASSET_BYTES: u64 = 64 << 20;

fn validate_audio_asset_size(size: u64) -> anyhow::Result<()> {
    if size > MAX_AUDIO_ASSET_BYTES {
        anyhow::bail!("audio asset exceeds the 64 MiB serving limit");
    }
    Ok(())
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
    let story_id = created
        .story_id
        .clone()
        .filter(|story_id| !story_id.is_empty())
        .ok_or_else(|| anyhow::anyhow!("created story is missing story_id"))?;
    if let Some(profile) = visual_profile {
        assets::update_profile_with_defaults(&state.pool, &story_id, profile).await?;
    }
    let snapshot = db::snapshot(&state.pool, &story_id).await?;
    assets::spawn_auto_generation(state.clone(), story_id);
    Ok(Json(json!({
        "story": created,
        "snapshot": snapshot,
    })))
}

async fn story_wizard(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<engine::StoryWizardEnvelope>,
) -> Result<Json<serde_json::Value>, ApiError> {
    let visual_profile = story_wizard_visual_profile(&payload);
    let wizard = engine::story_wizard(state.clone(), payload).await?;
    let wizard_story_id = wizard.story_id.clone().unwrap_or_default();
    let snapshot = if wizard_story_id.trim().is_empty() {
        None
    } else {
        if let Some(profile) = visual_profile {
            assets::update_profile_with_defaults(&state.pool, &wizard_story_id, profile).await?;
        }
        let snapshot = db::snapshot(&state.pool, &wizard_story_id).await?;
        assets::spawn_auto_generation(state.clone(), wizard_story_id);
        Some(snapshot)
    };
    Ok(Json(json!({
        "wizard": wizard,
        "snapshot": snapshot,
    })))
}

async fn story_enhance(
    State(state): State<Arc<AppState>>,
    Json(payload): Json<engine::StoryEnhanceEnvelope>,
) -> Result<Json<crate::gateway_protocol::StoryEnhanceResponse>, ApiError> {
    Ok(Json(engine::story_enhance(state, payload).await?))
}

fn story_create_visual_profile(
    payload: &engine::StoryCreateEnvelope,
) -> Option<assets::VisualProfileUpdate> {
    visual_profile_update(
        &payload.world_style_prompt,
        &payload.character_style_prompt,
        &payload.negative_prompt,
        &payload.palette,
    )
}

fn story_wizard_visual_profile(
    payload: &engine::StoryWizardEnvelope,
) -> Option<assets::VisualProfileUpdate> {
    visual_profile_update(
        &payload.world_style_prompt,
        &payload.character_style_prompt,
        &payload.negative_prompt,
        &payload.palette,
    )
}

fn visual_profile_update(
    world_style_prompt: &str,
    character_style_prompt: &str,
    negative_prompt: &str,
    palette: &str,
) -> Option<assets::VisualProfileUpdate> {
    let update = assets::VisualProfileUpdate {
        world_style_prompt: world_style_prompt.trim().to_string(),
        character_style_prompt: character_style_prompt.trim().to_string(),
        negative_prompt: negative_prompt.trim().to_string(),
        palette: palette.trim().to_string(),
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

async fn story_overview(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
) -> Result<Json<db::StoryOverview>, ApiError> {
    Ok(Json(db::story_overview(&state.pool, &story_id).await?))
}

async fn story_saves(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
) -> Result<Json<Vec<db::SaveView>>, ApiError> {
    Ok(Json(db::story_saves(&state.pool, &story_id).await?))
}

async fn timeline(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
) -> Result<Json<crate::gateway_protocol::BrowserTimelineResponse>, ApiError> {
    Ok(Json(
        engine::timeline(
            state,
            &story_id,
            engine::TimelineEnvelope {
                action: "list".into(),
                client_revision: 0,
                branch_id: String::new(),
                from_commit_id: String::new(),
                name: String::new(),
            },
        )
        .await?,
    ))
}

async fn update_timeline(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Json(payload): Json<engine::TimelineEnvelope>,
) -> Result<Json<serde_json::Value>, ApiError> {
    let timeline = engine::timeline(state.clone(), &story_id, payload).await?;
    let snapshot = db::snapshot(&state.pool, &story_id).await?;
    Ok(Json(json!({"timeline":timeline,"snapshot":snapshot})))
}

#[derive(Debug, serde::Deserialize)]
struct PageQuery {
    cursor: Option<i64>,
    limit: Option<i64>,
    #[serde(default)]
    q: String,
    #[serde(default)]
    format: String,
    #[serde(default)]
    language: String,
    #[serde(default)]
    mode: String,
}

async fn history(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Query(query): Query<PageQuery>,
) -> Result<Json<db::HistoryPage>, ApiError> {
    Ok(Json(
        db::history_page(
            &state.pool,
            &story_id,
            query.cursor,
            query.limit.unwrap_or(40),
            &query.q,
        )
        .await?,
    ))
}

async fn chapters(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Query(query): Query<PageQuery>,
) -> Result<Json<db::ChapterPage>, ApiError> {
    Ok(Json(
        db::chapter_page(
            &state.pool,
            &story_id,
            query.cursor,
            query.limit.unwrap_or(30),
            &query.q,
        )
        .await?,
    ))
}

async fn export_story(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Query(query): Query<PageQuery>,
) -> Result<Response, ApiError> {
    if query.format.eq_ignore_ascii_case("epub") {
        let (filename, bytes) = if query.language.trim().is_empty() || query.mode == "original" {
            db::export_story_epub(&state.pool, &story_id).await?
        } else {
            let export = db::export_story_readable(
                &state.pool,
                &story_id,
                "epub",
                &query.language,
                &query.mode,
            )
            .await?;
            (
                export.filename,
                BASE64.decode(export.content).map_err(anyhow::Error::from)?,
            )
        };
        let mut headers = HeaderMap::new();
        headers.insert(
            header::CONTENT_TYPE,
            HeaderValue::from_static("application/epub+zip"),
        );
        headers.insert(
            header::CONTENT_DISPOSITION,
            HeaderValue::from_str(&format!("attachment; filename=\"{filename}\""))
                .map_err(anyhow::Error::from)?,
        );
        return Ok((headers, bytes).into_response());
    }
    Ok(Json(
        db::export_story_readable(
            &state.pool,
            &story_id,
            &query.format,
            &query.language,
            &query.mode,
        )
        .await?,
    )
    .into_response())
}

async fn export_story_archive(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Json(options): Json<portability::ArchiveOptions>,
) -> Result<Response, ApiError> {
    let (filename, bytes) = portability::export_story_archive(state, &story_id, options).await?;
    let mut headers = HeaderMap::new();
    headers.insert(
        header::CONTENT_TYPE,
        HeaderValue::from_static("application/zip"),
    );
    headers.insert(
        header::CONTENT_DISPOSITION,
        HeaderValue::from_str(&format!("attachment; filename=\"{filename}\""))
            .map_err(anyhow::Error::from)?,
    );
    Ok((headers, bytes).into_response())
}

async fn import_story_archive(
    State(state): State<Arc<AppState>>,
    mut multipart: Multipart,
) -> Result<(StatusCode, Json<portability::ImportResult>), ApiError> {
    let mut archive = None;
    while let Some(field) = multipart.next_field().await.map_err(anyhow::Error::from)? {
        if field.name() == Some("file") {
            let bytes = field.bytes().await.map_err(anyhow::Error::from)?;
            if bytes.len() > 512 * 1024 * 1024 {
                return Err(ApiError::from(anyhow::Error::from(
                    PublicError::payload_too_large(
                        "archive_too_large",
                        "Story archive exceeds the import limit.",
                    ),
                )));
            }
            archive = Some(bytes.to_vec());
            break;
        }
    }
    let bytes = archive.ok_or_else(|| {
        anyhow::Error::from(PublicError::bad_request(
            "missing_archive",
            "Choose a OneDay ZIP archive.",
        ))
    })?;
    Ok((
        StatusCode::CREATED,
        Json(portability::import_story_archive(state, bytes).await?),
    ))
}

async fn export_world_template(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
) -> Result<Response, ApiError> {
    let (filename, bytes) = portability::export_world_template(&state.pool, &story_id).await?;
    let mut headers = HeaderMap::new();
    headers.insert(
        header::CONTENT_TYPE,
        HeaderValue::from_static("application/json; charset=utf-8"),
    );
    headers.insert(
        header::CONTENT_DISPOSITION,
        HeaderValue::from_str(&format!("attachment; filename=\"{filename}\""))
            .map_err(anyhow::Error::from)?,
    );
    Ok((headers, bytes).into_response())
}

async fn import_world_template(
    State(state): State<Arc<AppState>>,
    body: axum::body::Bytes,
) -> Result<(StatusCode, Json<portability::ImportResult>), ApiError> {
    Ok((
        StatusCode::CREATED,
        Json(portability::import_world_template(state, &body).await?),
    ))
}

async fn translation_jobs(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
) -> Result<Json<Vec<translation::JobView>>, ApiError> {
    Ok(Json(translation::list_jobs(&state.pool, &story_id).await?))
}

async fn translation_job(
    State(state): State<Arc<AppState>>,
    Path((story_id, job_id)): Path<(String, String)>,
) -> Result<Json<translation::JobView>, ApiError> {
    Ok(Json(
        translation::get_job(&state.pool, &story_id, &job_id).await?,
    ))
}

async fn estimate_translation_job(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Json(payload): Json<translation::CreateJobRequest>,
) -> Result<Json<translation::EstimateView>, ApiError> {
    Ok(Json(
        translation::estimate(&state.pool, &story_id, &payload).await?,
    ))
}

async fn create_translation_job(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Json(payload): Json<translation::CreateJobRequest>,
) -> Result<(StatusCode, Json<translation::JobView>), ApiError> {
    Ok((
        StatusCode::ACCEPTED,
        Json(translation::create_job(&state.pool, &story_id, payload).await?),
    ))
}

async fn translation_job_action(
    State(state): State<Arc<AppState>>,
    Path((story_id, job_id, action)): Path<(String, String, String)>,
) -> Result<Json<translation::JobView>, ApiError> {
    Ok(Json(
        translation::job_action(&state.pool, &story_id, &job_id, &action).await?,
    ))
}

#[derive(Debug, Default, serde::Deserialize)]
struct TranslationDeleteQuery {
    #[serde(default)]
    delete_translations: bool,
}

async fn delete_translation_job(
    State(state): State<Arc<AppState>>,
    Path((story_id, job_id)): Path<(String, String)>,
    Query(query): Query<TranslationDeleteQuery>,
) -> Result<StatusCode, ApiError> {
    translation::delete_job(&state.pool, &story_id, &job_id, query.delete_translations).await?;
    Ok(StatusCode::NO_CONTENT)
}

async fn next_browser_translation_item(
    State(state): State<Arc<AppState>>,
    Path((story_id, job_id)): Path<(String, String)>,
) -> Result<Json<Option<translation::BrowserItemView>>, ApiError> {
    Ok(Json(
        translation::next_browser_item(&state.pool, &story_id, &job_id).await?,
    ))
}

async fn complete_browser_translation_item(
    State(state): State<Arc<AppState>>,
    Path((story_id, job_id, item_id)): Path<(String, String, String)>,
    Json(payload): Json<translation::CompleteBrowserItemRequest>,
) -> Result<Json<translation::JobView>, ApiError> {
    Ok(Json(
        translation::complete_browser_item(
            &state.pool,
            &story_id,
            &job_id,
            &item_id,
            payload.translated_text,
        )
        .await?,
    ))
}

async fn translation_glossary(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
) -> Result<Json<Vec<translation::GlossaryEntry>>, ApiError> {
    Ok(Json(
        translation::list_glossary(&state.pool, &story_id).await?,
    ))
}

async fn create_translation_glossary(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Json(payload): Json<translation::GlossaryEntryRequest>,
) -> Result<(StatusCode, Json<translation::GlossaryEntry>), ApiError> {
    Ok((
        StatusCode::CREATED,
        Json(translation::upsert_glossary(&state.pool, &story_id, None, payload).await?),
    ))
}

async fn update_translation_glossary(
    State(state): State<Arc<AppState>>,
    Path((story_id, entry_id)): Path<(String, String)>,
    Json(payload): Json<translation::GlossaryEntryRequest>,
) -> Result<Json<translation::GlossaryEntry>, ApiError> {
    Ok(Json(
        translation::upsert_glossary(&state.pool, &story_id, Some(&entry_id), payload).await?,
    ))
}

async fn delete_translation_glossary(
    State(state): State<Arc<AppState>>,
    Path((story_id, entry_id)): Path<(String, String)>,
) -> Result<StatusCode, ApiError> {
    translation::delete_glossary(&state.pool, &story_id, &entry_id).await?;
    Ok(StatusCode::NO_CONTENT)
}

async fn message_diagnostics(
    State(state): State<Arc<AppState>>,
    Path((story_id, message_id)): Path<(String, i64)>,
) -> Result<Json<telemetry::GenerationDiagnostics>, ApiError> {
    Ok(Json(
        telemetry::message_diagnostics(&state.pool, &story_id, message_id).await?,
    ))
}

async fn export_telemetry(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Query(query): Query<PageQuery>,
) -> Result<Json<telemetry::TelemetryExport>, ApiError> {
    Ok(Json(
        telemetry::export_story_telemetry(&state.pool, &story_id, query.limit.unwrap_or(1000))
            .await?,
    ))
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

async fn upload_visual_asset_version(
    State(state): State<Arc<AppState>>,
    Path((story_id, asset_id)): Path<(String, String)>,
    multipart: Multipart,
) -> Result<
    (
        StatusCode,
        Json<crate::asset_upload::VisualAssetUploadResponse>,
    ),
    ApiError,
> {
    Ok((
        StatusCode::CREATED,
        Json(
            crate::asset_upload::upload_visual_asset_version(
                state, &story_id, &asset_id, multipart,
            )
            .await?,
        ),
    ))
}

async fn upload_new_visual_asset(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    multipart: Multipart,
) -> Result<
    (
        StatusCode,
        Json<crate::asset_upload::VisualAssetUploadResponse>,
    ),
    ApiError,
> {
    Ok((
        StatusCode::CREATED,
        Json(crate::asset_upload::upload_new_visual_asset(state, &story_id, multipart).await?),
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

async fn step_visual_asset_selection(
    State(state): State<Arc<AppState>>,
    Path((story_id, asset_id, action)): Path<(String, String, String)>,
) -> Result<Json<assets::VisualAssetsResponse>, ApiError> {
    Ok(Json(
        assets::step_asset_selection(&state.pool, &story_id, &asset_id, &action).await?,
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

async fn create_image_operation(
    State(state): State<Arc<AppState>>,
    Path((story_id, asset_id)): Path<(String, String)>,
    Json(payload): Json<assets::ImageOperationRequest>,
) -> Result<(StatusCode, Json<assets::VisualAssetsResponse>), ApiError> {
    Ok((
        StatusCode::ACCEPTED,
        Json(assets::create_image_operation(state, &story_id, &asset_id, payload).await?),
    ))
}

async fn image_operation(
    State(state): State<Arc<AppState>>,
    Path((story_id, operation_id)): Path<(String, String)>,
) -> Result<Json<assets::ImageOperationView>, ApiError> {
    Ok(Json(
        assets::get_image_operation(&state.pool, &story_id, &operation_id).await?,
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
            let api_error = ApiError::from(err);
            emit_turn_stream(
                &state,
                TurnStreamEvent::status_error(
                    &story_id,
                    client_turn,
                    &action_kind,
                    &action_text,
                    &api_error.code,
                    api_error.message.clone(),
                ),
            );
            return Err(api_error);
        }
    };
    if !stream_requested {
        for event in &events.events {
            let event = serde_json::to_value(event).map_err(anyhow::Error::from)?;
            emit_turn_stream(
                &state,
                TurnStreamEvent::contract(
                    &story_id,
                    client_turn,
                    &action_kind,
                    &action_text,
                    &event,
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

async fn craft(
    State(state): State<Arc<AppState>>,
    Path(story_id): Path<String>,
    Json(payload): Json<engine::CraftEnvelope>,
) -> Result<Json<serde_json::Value>, ApiError> {
    let crafting = engine::craft(state.clone(), &story_id, payload).await?;
    let snapshot = db::snapshot(&state.pool, &story_id).await?;
    Ok(Json(json!({ "crafting": crafting, "snapshot": snapshot })))
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
        "legacy": load.legacy.unwrap_or(false),
        "snapshot_state": load.snapshot_state,
        "snapshot_detail": load.snapshot_detail.unwrap_or_default(),
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

        let mut interval = tokio::time::interval(story_reconciliation_interval());
        interval.set_missed_tick_behavior(tokio::time::MissedTickBehavior::Skip);
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
                                            yield Ok(Event::default().event("error").data(sse_internal_error(&err)));
                                        }
                                    }
                                }
                                Err(err) => {
                                    yield Ok(Event::default().event("error").data(sse_internal_error(&err)));
                                }
                            }
                        }
                        Err(err) => {
                            yield Ok(Event::default().event("error").data(sse_internal_error(&err)));
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

fn story_reconciliation_interval() -> Duration {
    parse_story_reconciliation_interval(
        std::env::var("ONEDAY_SSE_RECONCILE_SECONDS")
            .ok()
            .as_deref(),
    )
}

fn parse_story_reconciliation_interval(value: Option<&str>) -> Duration {
    let seconds = value
        .and_then(|raw| raw.trim().parse::<u64>().ok())
        .unwrap_or(15)
        .clamp(2, 300);
    Duration::from_secs(seconds)
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

fn sse_internal_error(error: &dyn std::fmt::Display) -> String {
    tracing::error!(
        request_id = crate::current_request_id().as_deref().unwrap_or("-"),
        error = %error,
        "story event reconciliation failed"
    );
    "An internal gateway error occurred.".to_string()
}

#[derive(Clone, Debug)]
struct ApiError {
    status: StatusCode,
    message: String,
    code: String,
}

impl IntoResponse for ApiError {
    fn into_response(self) -> Response {
        let body = json!({ "error": self.message, "code": self.code });
        (self.status, Json(body)).into_response()
    }
}

impl ApiError {
    fn public(error: PublicError) -> Self {
        Self {
            status: error.kind.status(),
            message: error.message,
            code: error.code.to_string(),
        }
    }

    fn internal(error: &dyn std::fmt::Display) -> Self {
        tracing::error!(
            request_id = crate::current_request_id().as_deref().unwrap_or("-"),
            error = %error,
            "gateway request failed"
        );
        Self {
            status: StatusCode::INTERNAL_SERVER_ERROR,
            message: "An internal gateway error occurred.".to_string(),
            code: "internal_error".to_string(),
        }
    }
}

fn bridge_error_status(code: &str) -> StatusCode {
    match code {
        "validation_failed"
        | "invalid_request"
        | "invalid_audio_request"
        | "invalid_minigame_request" => StatusCode::BAD_REQUEST,
        "stale_request" | "conflict" | "stale_config" | "config_locked" => StatusCode::CONFLICT,
        "not_found" => StatusCode::NOT_FOUND,
        _ => StatusCode::INTERNAL_SERVER_ERROR,
    }
}

impl From<anyhow::Error> for ApiError {
    fn from(err: anyhow::Error) -> Self {
        if let Some(bridge) = err.downcast_ref::<engine::BridgeError>() {
            let status = bridge_error_status(&bridge.code);
            let message = if status.is_server_error() {
                tracing::error!(
                    request_id = crate::current_request_id().as_deref().unwrap_or("-"),
                    error_code = %bridge.code,
                    error = %bridge.message,
                    "Go bridge request failed"
                );
                "An internal gateway error occurred.".to_string()
            } else {
                bridge.message.clone()
            };
            return Self {
                status,
                message,
                code: bridge.code.clone(),
            };
        }
        if let Some(public) = err.downcast_ref::<PublicError>() {
            return Self {
                status: public.kind.status(),
                message: public.message.clone(),
                code: public.code.to_string(),
            };
        }
        Self::internal(&err)
    }
}

impl From<sqlx::Error> for ApiError {
    fn from(err: sqlx::Error) -> Self {
        Self::internal(&err)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn command_descriptor_locale_normalizes_supported_regional_variants() {
        assert_eq!(normalize_interface_locale(None), "en");
        assert_eq!(normalize_interface_locale(Some("en-US")), "en");
        assert_eq!(normalize_interface_locale(Some("it-IT")), "it");
        assert_eq!(normalize_interface_locale(Some("IT_it")), "it");
        assert_eq!(normalize_interface_locale(Some("fr-FR")), "en");
        assert_eq!(normalize_interface_locale(Some("")), "en");
    }

    #[test]
    fn api_errors_use_stable_codes_without_exposing_internal_details() {
        let internal = ApiError::from(anyhow::anyhow!(
            "opening SQLite database /private/path failed"
        ));
        assert_eq!(internal.status, StatusCode::INTERNAL_SERVER_ERROR);
        assert_eq!(internal.code, "internal_error");
        assert_eq!(internal.message, "An internal gateway error occurred.");

        let public = ApiError::from(anyhow::Error::new(PublicError::not_found(
            "story_not_found",
            "story not found: story-1",
        )));
        assert_eq!(public.status, StatusCode::NOT_FOUND);
        assert_eq!(public.code, "story_not_found");
    }

    #[test]
    fn sse_errors_keep_the_plaintext_contract_while_redacting_details() {
        let payload = sse_internal_error(&anyhow::anyhow!("database /private/path failed"));

        assert_eq!(payload, "An internal gateway error occurred.");
        assert!(!payload.contains("/private/path"));
    }

    #[test]
    fn bridge_error_status_is_determined_by_code_only() {
        assert_eq!(
            bridge_error_status("validation_failed"),
            StatusCode::BAD_REQUEST
        );
        assert_eq!(
            bridge_error_status("invalid_request"),
            StatusCode::BAD_REQUEST
        );
        assert_eq!(bridge_error_status("stale_config"), StatusCode::CONFLICT);
        assert_eq!(bridge_error_status("stale_request"), StatusCode::CONFLICT);
        assert_eq!(bridge_error_status("conflict"), StatusCode::CONFLICT);
        assert_eq!(bridge_error_status("not_found"), StatusCode::NOT_FOUND);
        assert_eq!(
            bridge_error_status("turn_failed"),
            StatusCode::INTERNAL_SERVER_ERROR
        );

        let bridge = ApiError::from(anyhow::Error::new(engine::BridgeError {
            code: "turn_failed".to_string(),
            message: "database /private/path failed".to_string(),
        }));
        assert_eq!(bridge.code, "turn_failed");
        assert_eq!(bridge.message, "An internal gateway error occurred.");
    }

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

    #[test]
    fn rejects_audio_assets_before_allocating_oversized_files() {
        assert!(validate_audio_asset_size(MAX_AUDIO_ASSET_BYTES).is_ok());
        assert!(validate_audio_asset_size(MAX_AUDIO_ASSET_BYTES + 1).is_err());
    }

    #[test]
    fn external_story_reconciliation_is_slow_and_bounded() {
        assert_eq!(
            parse_story_reconciliation_interval(None),
            Duration::from_secs(15)
        );
        assert_eq!(
            parse_story_reconciliation_interval(Some("1")),
            Duration::from_secs(2)
        );
        assert_eq!(
            parse_story_reconciliation_interval(Some("999")),
            Duration::from_secs(300)
        );
    }

    #[test]
    fn story_wizard_preserves_the_selected_visual_direction() {
        let payload = engine::StoryWizardEnvelope {
            state: None,
            input: String::new(),
            action: "create_story".to_string(),
            world_style_prompt: "  physically grounded world  ".to_string(),
            character_style_prompt: " real human characters ".to_string(),
            negative_prompt: " plastic skin ".to_string(),
            palette: " amber and smoke ".to_string(),
            start: true,
        };

        let profile = story_wizard_visual_profile(&payload).expect("visual profile");
        assert_eq!(profile.world_style_prompt, "physically grounded world");
        assert_eq!(profile.character_style_prompt, "real human characters");
        assert_eq!(profile.negative_prompt, "plastic skin");
        assert_eq!(profile.palette, "amber and smoke");
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
                spatial_regions: json!([]),
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
