use crate::{events::TurnStreamEvent, AppState};
use anyhow::{anyhow, Context};
use serde::de::DeserializeOwned;
use serde::{Deserialize, Serialize};
use std::process::{ExitStatus, Stdio};
use std::sync::Arc;
use std::time::Duration;
use tokio::io::{AsyncBufReadExt, AsyncReadExt, AsyncWriteExt, BufReader};
use tokio::process::{Child, Command};
use uuid::Uuid;

#[derive(Debug, thiserror::Error)]
#[error("{message}")]
pub struct BridgeError {
    pub code: String,
    pub message: String,
}

fn bridge_error(code: impl Into<String>, message: impl Into<String>) -> anyhow::Error {
    BridgeError {
        code: code.into(),
        message: message.into(),
    }
    .into()
}

#[derive(Debug, Deserialize, Serialize)]
pub struct ActionEnvelope {
    pub session_id: String,
    pub client_turn: i64,
    pub client_revision: i64,
    #[serde(default)]
    pub idempotency_key: String,
    pub action: PlayerAction,
    #[serde(default)]
    pub stream: bool,
    #[serde(default)]
    pub capabilities: ClientCapabilities,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct PlayerAction {
    pub kind: String,
    #[serde(default)]
    pub text: String,
    #[serde(default)]
    pub choice_id: i64,
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct CraftMessage {
    pub role: String,
    pub content: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct CraftEnvelope {
    pub message: String,
    #[serde(default)]
    pub history: Vec<CraftMessage>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct GatewayCraftResponse {
    pub crafting: Option<serde_json::Value>,
    #[serde(default)]
    pub error: String,
}

#[derive(Debug, Default, Deserialize, Serialize)]
pub struct ClientCapabilities {
    #[serde(default)]
    pub images: bool,
    #[serde(default)]
    pub ascii: bool,
    #[serde(default)]
    pub roll_log: bool,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct MetaEnvelope {
    pub session_id: String,
    pub client_turn: i64,
    pub client_revision: i64,
    pub kind: String,
    #[serde(default)]
    pub text: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct SaveEnvelope {
    pub session_id: String,
    pub client_turn: i64,
    pub client_revision: i64,
    #[serde(default)]
    pub name: String,
    #[serde(default = "default_save_kind")]
    pub kind: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct LoadEnvelope {
    pub session_id: String,
    pub client_turn: i64,
    pub client_revision: i64,
    pub save_id: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct DeleteSaveEnvelope {
    pub session_id: String,
    pub client_turn: i64,
    pub client_revision: i64,
    pub save_id: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct TimelineEnvelope {
    pub action: String,
    pub client_revision: i64,
    #[serde(default)]
    pub branch_id: String,
    #[serde(default)]
    pub from_commit_id: String,
    #[serde(default)]
    pub name: String,
}

#[derive(Debug, Serialize)]
struct GatewayTimelineRequest<'a> {
    story_id: &'a str,
    action: &'a str,
    client_revision: i64,
    branch_id: &'a str,
    from_commit_id: &'a str,
    name: &'a str,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct TimelineBranchView {
    pub id: String,
    pub story_id: String,
    pub name: String,
    #[serde(default)]
    pub fork_commit_id: String,
    pub head_commit_id: String,
    pub head_turn: i64,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct TimelineCommitView {
    pub id: String,
    pub branch_id: String,
    #[serde(default)]
    pub parent_commit_id: String,
    pub canonical_turn: i64,
    pub kind: String,
    #[serde(default)]
    pub message: String,
    pub created_at: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct TimelineResponse {
    pub active_branch_id: String,
    pub revision: i64,
    pub branches: Vec<TimelineBranchView>,
    pub head: Option<TimelineCommitView>,
    #[serde(default)]
    pub commits: Vec<TimelineCommitView>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct StoryCreateEnvelope {
    pub brief: String,
    pub character_name: String,
    #[serde(default)]
    pub character_background: String,
    #[serde(default)]
    pub world_style_prompt: String,
    #[serde(default)]
    pub character_style_prompt: String,
    #[serde(default)]
    pub negative_prompt: String,
    #[serde(default)]
    pub palette: String,
    #[serde(default = "default_start_story")]
    pub start: bool,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct StoryWizardEnvelope {
    pub state: Option<serde_json::Value>,
    #[serde(default)]
    pub input: String,
    #[serde(default)]
    pub action: String,
    #[serde(default = "default_start_story")]
    pub start: bool,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct StoryEnhanceEnvelope {
    pub state: Option<serde_json::Value>,
    #[serde(default)]
    pub stage: String,
    #[serde(default)]
    pub text: String,
    #[serde(default)]
    pub context: String,
}

#[derive(Debug, Serialize)]
struct GatewayTurnRequest<'a> {
    story_id: &'a str,
    session_id: &'a str,
    client_turn: i64,
    client_revision: i64,
    idempotency_key: &'a str,
    action: &'a PlayerAction,
    stream: bool,
    capabilities: &'a ClientCapabilities,
}

#[derive(Debug, Serialize)]
struct GatewayMetaRequest<'a> {
    story_id: &'a str,
    session_id: &'a str,
    client_turn: i64,
    client_revision: i64,
    kind: &'a str,
    text: &'a str,
}

#[derive(Debug, Serialize)]
struct GatewaySaveRequest<'a> {
    story_id: &'a str,
    session_id: &'a str,
    client_turn: i64,
    client_revision: i64,
    name: &'a str,
    kind: &'a str,
}

#[derive(Debug, Serialize)]
struct GatewayLoadRequest<'a> {
    story_id: &'a str,
    session_id: &'a str,
    client_turn: i64,
    client_revision: i64,
    save_id: &'a str,
}

#[derive(Debug, Serialize)]
struct GatewayDeleteSaveRequest<'a> {
    story_id: &'a str,
    session_id: &'a str,
    client_turn: i64,
    client_revision: i64,
    save_id: &'a str,
}

#[derive(Debug, Serialize)]
struct GatewayStoryCreateRequest<'a> {
    brief: &'a str,
    character_name: &'a str,
    character_background: &'a str,
    start: bool,
}

#[derive(Debug, Serialize)]
struct GatewayStoryWizardRequest<'a> {
    state: Option<&'a serde_json::Value>,
    input: &'a str,
    action: &'a str,
    start: bool,
}

#[derive(Debug, Serialize)]
struct GatewayStoryEnhanceRequest<'a> {
    state: Option<&'a serde_json::Value>,
    stage: &'a str,
    text: &'a str,
    context: &'a str,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct GatewayTurnResponse {
    #[serde(default)]
    pub events: Vec<serde_json::Value>,
    #[serde(default)]
    pub error: String,
}

#[derive(Debug, Deserialize, Serialize)]
struct GatewayTurnStreamLine {
    pub event: Option<serde_json::Value>,
    #[serde(default)]
    pub phase: String,
    #[serde(default)]
    pub error: String,
    #[serde(default)]
    pub done: bool,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct GatewayMetaResult {
    pub kind: String,
    pub title: String,
    pub message: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct GatewayMetaResponse {
    pub meta: Option<GatewayMetaResult>,
    #[serde(default)]
    pub error: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct GatewaySaveView {
    pub id: String,
    pub name: String,
    pub turn: i64,
    pub chapter: i64,
    #[serde(default)]
    pub location: String,
    #[serde(default)]
    pub session_id: String,
    #[serde(default)]
    pub metadata: serde_json::Value,
    pub created_at: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct GatewaySaveResponse {
    pub save: Option<GatewaySaveView>,
    #[serde(default)]
    pub error: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct GatewayLoadResponse {
    pub save: Option<GatewaySaveView>,
    #[serde(default)]
    pub legacy: bool,
    #[serde(default)]
    pub snapshot_state: String,
    #[serde(default)]
    pub snapshot_detail: String,
    #[serde(default)]
    pub error: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct GatewayDeleteSaveResponse {
    pub save: Option<GatewaySaveView>,
    #[serde(default)]
    pub error: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct GatewayStoryCreateResponse {
    pub story_id: String,
    pub character_id: String,
    #[serde(default)]
    pub session_id: String,
    #[serde(default)]
    pub started: bool,
    #[serde(default)]
    pub start_error: String,
    #[serde(default)]
    pub error: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct GatewayStoryWizardResponse {
    #[serde(default)]
    pub state: serde_json::Value,
    #[serde(default)]
    pub phase: String,
    #[serde(default)]
    pub stage: String,
    #[serde(default)]
    pub stage_label: String,
    #[serde(default)]
    pub placeholder: String,
    #[serde(default)]
    pub message: String,
    #[serde(default)]
    pub actions: Vec<serde_json::Value>,
    #[serde(default)]
    pub definition: serde_json::Value,
    #[serde(default)]
    pub last_model: String,
    #[serde(default)]
    pub last_latency_ms: i64,
    #[serde(default)]
    pub story_id: String,
    #[serde(default)]
    pub character_id: String,
    #[serde(default)]
    pub session_id: String,
    #[serde(default)]
    pub started: bool,
    #[serde(default)]
    pub start_error: String,
    #[serde(default)]
    pub error: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct GatewayStoryEnhanceResponse {
    #[serde(default)]
    pub text: String,
    #[serde(default)]
    pub model: String,
    #[serde(default)]
    pub provider: String,
    #[serde(default)]
    pub latency_ms: i64,
    #[serde(default)]
    pub error: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct GatewayCommandDescriptorsResponse {
    #[serde(default)]
    pub commands: Vec<serde_json::Value>,
    #[serde(default)]
    pub error: String,
}

#[derive(Debug, Default, Deserialize, Serialize)]
pub struct GatewayMiniGameResponse {
    #[serde(default)]
    pub instance: Option<serde_json::Value>,
    #[serde(default)]
    pub error: String,
}

#[derive(Debug, Default, Deserialize, Serialize)]
pub struct GatewayAudioResponse {
    #[serde(default)]
    pub providers: Vec<serde_json::Value>,
    #[serde(default)]
    pub voices: Vec<serde_json::Value>,
    #[serde(default)]
    pub settings: Option<serde_json::Value>,
    #[serde(default)]
    pub assignments: Vec<serde_json::Value>,
    #[serde(default)]
    pub assignment: Option<serde_json::Value>,
    #[serde(default)]
    pub pronunciations: Vec<serde_json::Value>,
    #[serde(default)]
    pub pronunciation: Option<serde_json::Value>,
    #[serde(default)]
    pub assets: Vec<serde_json::Value>,
    #[serde(default)]
    pub jobs: Vec<serde_json::Value>,
    #[serde(default)]
    pub asset: Option<serde_json::Value>,
    #[serde(default)]
    pub file_path: String,
    #[serde(default)]
    pub format: String,
    #[serde(default)]
    pub cleanup: Option<serde_json::Value>,
    #[serde(default)]
    pub export: Option<serde_json::Value>,
    #[serde(default)]
    pub error: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct MiniGameStartEnvelope {
    pub definition: serde_json::Value,
    #[serde(default)]
    pub selection: serde_json::Value,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct MiniGameInputEnvelope {
    pub input: serde_json::Value,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct ModelProviderSetting {
    pub id: String,
    pub label: String,
    pub enabled: bool,
    #[serde(default)]
    pub model: String,
    #[serde(default)]
    pub reasoning: String,
    pub supports_model: bool,
    pub supports_reasoning: bool,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct ModelRoutingActive {
    pub provider: String,
    pub narrative_model: String,
    pub utility_model: String,
    pub repair_model: String,
    #[serde(default)]
    pub repair_fallback_models: Vec<String>,
    pub image_model: String,
    #[serde(default)]
    pub ascii_model: String,
    pub embedding_provider: String,
    pub embedding_model: String,
    pub codex_reasoning: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct ModelRoutingSettings {
    pub config_path: String,
    pub config_revision: String,
    #[serde(default)]
    pub provider_priority: Vec<String>,
    #[serde(default)]
    pub providers: Vec<ModelProviderSetting>,
    #[serde(default)]
    pub narrative_models: Vec<String>,
    #[serde(default)]
    pub utility_models: Vec<String>,
    #[serde(default)]
    pub repair_models: Vec<String>,
    #[serde(default)]
    pub image_models: Vec<String>,
    #[serde(default)]
    pub ascii_models: Vec<String>,
    #[serde(default)]
    pub embedding_providers: Vec<String>,
    pub image_generation: ImageGenerationSetting,
    pub active: ModelRoutingActive,
    pub tts_status: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct ImageGenerationSetting {
    pub provider: String,
    pub base_url: String,
    pub api_key_configured: bool,
    pub model: String,
    pub openclaw_bridge_url: String,
    pub default_size: String,
    pub location_size: String,
    pub character_size: String,
    pub default_resolution: String,
    pub location_resolution: String,
    pub character_resolution: String,
    pub default_aspect_ratio: String,
    pub location_aspect_ratio: String,
    pub character_aspect_ratio: String,
    pub quality: String,
    pub output_format: String,
    pub background: String,
    pub timeout_seconds: i64,
    pub auto_generate: bool,
    pub append_negative_prompt: bool,
    pub available: bool,
    pub status: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct ModelProviderUpdate {
    pub id: String,
    pub enabled: Option<bool>,
    pub model: Option<String>,
    pub reasoning: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct ModelRoutingUpdate {
    pub base_revision: Option<String>,
    pub provider_priority: Option<Vec<String>>,
    #[serde(default)]
    pub providers: Vec<ModelProviderUpdate>,
    pub utility_model: Option<String>,
    pub repair_model: Option<String>,
    pub repair_fallback_models: Option<Vec<String>>,
    pub image_model: Option<String>,
    pub image_generation: Option<ImageGenerationUpdate>,
    pub ascii_model: Option<String>,
    pub embedding_provider: Option<String>,
    pub embedding_model: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct ImageGenerationUpdate {
    pub provider: Option<String>,
    pub base_url: Option<String>,
    pub model: Option<String>,
    pub openclaw_bridge_url: Option<String>,
    pub default_size: Option<String>,
    pub location_size: Option<String>,
    pub character_size: Option<String>,
    pub default_resolution: Option<String>,
    pub location_resolution: Option<String>,
    pub character_resolution: Option<String>,
    pub default_aspect_ratio: Option<String>,
    pub location_aspect_ratio: Option<String>,
    pub character_aspect_ratio: Option<String>,
    pub quality: Option<String>,
    pub output_format: Option<String>,
    pub background: Option<String>,
    pub timeout_seconds: Option<i64>,
    pub auto_generate: Option<bool>,
    pub append_negative_prompt: Option<bool>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct GatewayModelSettingsResponse {
    pub settings: Option<ModelRoutingSettings>,
    #[serde(default)]
    pub error: String,
    #[serde(default)]
    pub error_code: String,
}

pub async fn command_descriptors(
    state: Arc<AppState>,
) -> anyhow::Result<GatewayCommandDescriptorsResponse> {
    let output = tokio::time::timeout(
        Duration::from_secs(30),
        Command::new(&state.paths.oneday_bin)
            .arg("gateway-command-descriptors")
            .env("ONEDAY_CONFIG", &state.paths.config_path)
            .current_dir(&state.paths.oneday_root)
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .output(),
    )
    .await
    .context("gateway-command-descriptors timed out")?
    .with_context(|| {
        format!(
            "starting {} gateway-command-descriptors",
            state.paths.oneday_bin.display()
        )
    })?;

    let stderr = String::from_utf8_lossy(&output.stderr).to_string();
    let parsed = serde_json::from_slice::<GatewayCommandDescriptorsResponse>(&output.stdout)
        .with_context(|| {
            format!(
                "decoding gateway-command-descriptors stdout; stderr={}",
                compact_stderr(&stderr)
            )
        })?;
    if !parsed.error.trim().is_empty() {
        return Err(anyhow!(parsed.error));
    }
    if !output.status.success() {
        return Err(anyhow!(
            "gateway-command-descriptors failed: {}",
            compact_stderr(&stderr)
        ));
    }
    Ok(parsed)
}

pub async fn model_settings(state: Arc<AppState>) -> anyhow::Result<ModelRoutingSettings> {
    let output = tokio::time::timeout(
        Duration::from_secs(30),
        Command::new(&state.paths.oneday_bin)
            .arg("gateway-model-settings")
            .env("ONEDAY_CONFIG", &state.paths.config_path)
            .current_dir(&state.paths.oneday_root)
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .output(),
    )
    .await
    .context("gateway-model-settings timed out")?
    .with_context(|| {
        format!(
            "starting {} gateway-model-settings",
            state.paths.oneday_bin.display()
        )
    })?;

    let stderr = String::from_utf8_lossy(&output.stderr).to_string();
    let parsed = serde_json::from_slice::<GatewayModelSettingsResponse>(&output.stdout)
        .with_context(|| {
            format!(
                "decoding gateway-model-settings stdout; stderr={}",
                compact_stderr(&stderr)
            )
        })?;
    if !parsed.error.trim().is_empty() {
        return Err(bridge_error(parsed.error_code, parsed.error));
    }
    if !output.status.success() {
        return Err(anyhow!(
            "gateway-model-settings failed: {}",
            compact_stderr(&stderr)
        ));
    }
    parsed
        .settings
        .ok_or_else(|| anyhow!("gateway-model-settings returned no settings"))
}

pub async fn update_model_settings(
    state: Arc<AppState>,
    update: ModelRoutingUpdate,
) -> anyhow::Result<ModelRoutingSettings> {
    let (parsed, status_ok, stderr) = call_gateway::<_, GatewayModelSettingsResponse>(
        state,
        "gateway-model-settings-update",
        &update,
    )
    .await?;
    if !parsed.error.trim().is_empty() {
        return Err(bridge_error(parsed.error_code, parsed.error));
    }
    if !status_ok {
        return Err(anyhow!(
            "gateway-model-settings-update failed: {}",
            compact_stderr(&stderr)
        ));
    }
    parsed
        .settings
        .ok_or_else(|| anyhow!("gateway-model-settings-update returned no settings"))
}

pub async fn submit_action(
    state: Arc<AppState>,
    story_id: &str,
    mut envelope: ActionEnvelope,
) -> anyhow::Result<GatewayTurnResponse> {
    if envelope.idempotency_key.trim().is_empty() {
        envelope.idempotency_key = Uuid::new_v4().to_string();
    }

    let req = GatewayTurnRequest {
        story_id,
        session_id: &envelope.session_id,
        client_turn: envelope.client_turn,
        client_revision: envelope.client_revision,
        idempotency_key: &envelope.idempotency_key,
        action: &envelope.action,
        stream: envelope.stream,
        capabilities: &envelope.capabilities,
    };

    if envelope.stream {
        return call_gateway_turn_stream(
            state,
            &req,
            story_id,
            envelope.client_turn,
            envelope.action.kind.clone(),
            envelope.action.text.clone(),
        )
        .await;
    }

    let (parsed, status_ok, stderr) =
        call_gateway::<_, GatewayTurnResponse>(state, "gateway-turn", &req).await?;
    if !parsed.error.trim().is_empty() {
        return Err(anyhow!(parsed.error));
    }
    if !status_ok {
        return Err(anyhow!("gateway-turn failed: {}", compact_stderr(&stderr)));
    }
    Ok(parsed)
}

pub async fn craft(
    state: Arc<AppState>,
    story_id: &str,
    envelope: CraftEnvelope,
) -> anyhow::Result<serde_json::Value> {
    let request = serde_json::json!({
        "story_id": story_id,
        "message": envelope.message,
        "history": envelope.history,
    });
    let (parsed, status_ok, stderr) =
        call_gateway::<_, GatewayCraftResponse>(state, "gateway-craft", &request).await?;
    if !parsed.error.trim().is_empty() {
        return Err(anyhow!(parsed.error));
    }
    if !status_ok {
        return Err(anyhow!("gateway-craft failed: {}", compact_stderr(&stderr)));
    }
    parsed
        .crafting
        .ok_or_else(|| anyhow!("gateway-craft returned no crafting response"))
}

pub async fn timeline(
    state: Arc<AppState>,
    story_id: &str,
    envelope: TimelineEnvelope,
) -> anyhow::Result<TimelineResponse> {
    let request = GatewayTimelineRequest {
        story_id,
        action: &envelope.action,
        client_revision: envelope.client_revision,
        branch_id: &envelope.branch_id,
        from_commit_id: &envelope.from_commit_id,
        name: &envelope.name,
    };
    let first = call_timeline_gateway(state.clone(), &request).await;
    match first {
        Ok(response) => Ok(response),
        Err(first_error) if envelope.action == "list" => {
            tokio::time::sleep(Duration::from_millis(75)).await;
            call_timeline_gateway(state, &request)
                .await
                .map_err(|retry_error| {
                    anyhow!(
                        "gateway-timeline list failed after retry: {}; first error: {}",
                        retry_error,
                        first_error
                    )
                })
        }
        Err(error) => Err(error),
    }
}

async fn call_timeline_gateway(
    state: Arc<AppState>,
    request: &GatewayTimelineRequest<'_>,
) -> anyhow::Result<TimelineResponse> {
    let (parsed, status_ok, stderr) =
        call_gateway::<_, TimelineResponse>(state, "gateway-timeline", request).await?;
    if !status_ok {
        return Err(anyhow!(
            "gateway-timeline failed: {}",
            compact_stderr(&stderr)
        ));
    }
    Ok(parsed)
}

pub async fn start_minigame(
    state: Arc<AppState>,
    story_id: &str,
    envelope: MiniGameStartEnvelope,
) -> anyhow::Result<GatewayMiniGameResponse> {
    call_minigame_gateway(
        state,
        "gateway-minigame-start",
        serde_json::json!({"story_id": story_id, "definition": envelope.definition, "selection": envelope.selection}),
    )
    .await
}

pub async fn active_minigame(
    state: Arc<AppState>,
    story_id: &str,
) -> anyhow::Result<GatewayMiniGameResponse> {
    call_minigame_gateway(
        state,
        "gateway-minigame-get",
        serde_json::json!({"story_id": story_id}),
    )
    .await
}

pub async fn input_minigame(
    state: Arc<AppState>,
    story_id: &str,
    instance_id: &str,
    envelope: MiniGameInputEnvelope,
) -> anyhow::Result<GatewayMiniGameResponse> {
    call_minigame_gateway(
        state,
        "gateway-minigame-input",
        serde_json::json!({"story_id": story_id, "instance_id": instance_id, "input": envelope.input}),
    )
    .await
}

async fn call_minigame_gateway(
    state: Arc<AppState>,
    command: &str,
    request: serde_json::Value,
) -> anyhow::Result<GatewayMiniGameResponse> {
    let (parsed, status_ok, stderr) =
        call_gateway::<_, GatewayMiniGameResponse>(state, command, &request).await?;
    if !parsed.error.trim().is_empty() {
        return Err(anyhow!(parsed.error));
    }
    if !status_ok {
        return Err(anyhow!("{command} failed: {}", compact_stderr(&stderr)));
    }
    Ok(parsed)
}

pub async fn audio(
    state: Arc<AppState>,
    request: serde_json::Value,
) -> anyhow::Result<GatewayAudioResponse> {
    let (parsed, status_ok, stderr) =
        call_gateway::<_, GatewayAudioResponse>(state, "gateway-audio", &request).await?;
    if !parsed.error.trim().is_empty() {
        return Err(anyhow!(parsed.error));
    }
    if !status_ok {
        return Err(anyhow!("gateway-audio failed: {}", compact_stderr(&stderr)));
    }
    Ok(parsed)
}

async fn call_gateway_turn_stream(
    state: Arc<AppState>,
    req: &GatewayTurnRequest<'_>,
    story_id: &str,
    client_turn: i64,
    action_kind: String,
    action_text: String,
) -> anyhow::Result<GatewayTurnResponse> {
    let input = serde_json::to_vec(req).context("encoding gateway-turn stream request")?;
    let mut child = gateway_command(&state, "gateway-turn")
        .spawn()
        .with_context(|| format!("starting {}", state.paths.oneday_bin.display()))?;

    let mut stdin = child.stdin.take().context("opening gateway-turn stdin")?;
    stdin.write_all(&input).await?;
    stdin.shutdown().await?;
    drop(stdin);

    let stdout = child.stdout.take().context("opening gateway-turn stdout")?;
    let stderr = child.stderr.take().context("opening gateway-turn stderr")?;
    let stderr_task = tokio::spawn(async move {
        let mut reader = BufReader::new(stderr);
        let mut text = String::new();
        let _ = reader.read_to_string(&mut text).await;
        text
    });

    let mut events = Vec::new();
    let mut saw_done = false;
    let mut lines = BufReader::new(stdout).lines();
    let read_result = match tokio::time::timeout(Duration::from_secs(360), async {
        while let Some(line) = lines.next_line().await? {
            let line = line.trim();
            if line.is_empty() {
                continue;
            }
            let parsed: GatewayTurnStreamLine = serde_json::from_str(line)
                .with_context(|| format!("decoding gateway-turn stream line: {line}"))?;
            if !parsed.error.trim().is_empty() {
                return Err(anyhow!(parsed.error));
            }
            if let Some(event) = parsed.event {
                let event_type = event
                    .get("type")
                    .and_then(serde_json::Value::as_str)
                    .unwrap_or("turn.event")
                    .to_string();
                let _ = state.turn_events.send(TurnStreamEvent::contract(
                    story_id,
                    client_turn,
                    &action_kind,
                    &action_text,
                    &event,
                ));
                let is_error = event_type == "error";
                let error_message = event
                    .get("payload")
                    .and_then(|payload| payload.get("message"))
                    .and_then(serde_json::Value::as_str)
                    .unwrap_or("gateway-turn stream failed")
                    .to_string();
                if parsed.phase != "live" {
                    events.push(event);
                }
                if is_error {
                    return Err(anyhow!(error_message));
                }
            }
            if parsed.done {
                saw_done = true;
                break;
            }
        }
        if !saw_done {
            return Err(anyhow!("gateway-turn stream ended before done line"));
        }
        Ok::<(), anyhow::Error>(())
    })
    .await
    {
        Ok(result) => result,
        Err(_) => {
            terminate_child(&mut child).await;
            let _ = stderr_task.await;
            return Err(anyhow!("gateway-turn stream timed out"));
        }
    };

    if let Err(err) = read_result {
        terminate_child(&mut child).await;
        let _ = stderr_task.await;
        return Err(err);
    }

    let status = wait_for_child(
        &mut child,
        Duration::from_secs(30),
        "waiting for gateway-turn stream",
    )
    .await?;
    let stderr = stderr_task.await.unwrap_or_default();

    if !status.success() {
        return Err(anyhow!(
            "gateway-turn stream failed: {}",
            compact_stderr(&stderr)
        ));
    }
    Ok(GatewayTurnResponse {
        events,
        error: String::new(),
    })
}

pub async fn create_story(
    state: Arc<AppState>,
    envelope: StoryCreateEnvelope,
) -> anyhow::Result<GatewayStoryCreateResponse> {
    let req = GatewayStoryCreateRequest {
        brief: &envelope.brief,
        character_name: &envelope.character_name,
        character_background: &envelope.character_background,
        start: envelope.start,
    };
    let (parsed, status_ok, stderr) =
        call_gateway::<_, GatewayStoryCreateResponse>(state, "gateway-story-create", &req).await?;
    if !parsed.error.trim().is_empty() {
        return Err(anyhow!(parsed.error));
    }
    if !status_ok {
        return Err(anyhow!(
            "gateway-story-create failed: {}",
            compact_stderr(&stderr)
        ));
    }
    Ok(parsed)
}

pub async fn story_wizard(
    state: Arc<AppState>,
    envelope: StoryWizardEnvelope,
) -> anyhow::Result<GatewayStoryWizardResponse> {
    let req = GatewayStoryWizardRequest {
        state: envelope.state.as_ref(),
        input: &envelope.input,
        action: &envelope.action,
        start: envelope.start,
    };
    let (parsed, status_ok, stderr) =
        call_gateway::<_, GatewayStoryWizardResponse>(state, "gateway-story-wizard", &req).await?;
    if !parsed.error.trim().is_empty() {
        return Err(anyhow!(parsed.error));
    }
    if !status_ok {
        return Err(anyhow!(
            "gateway-story-wizard failed: {}",
            compact_stderr(&stderr)
        ));
    }
    Ok(parsed)
}

pub async fn story_enhance(
    state: Arc<AppState>,
    envelope: StoryEnhanceEnvelope,
) -> anyhow::Result<GatewayStoryEnhanceResponse> {
    let req = GatewayStoryEnhanceRequest {
        state: envelope.state.as_ref(),
        stage: &envelope.stage,
        text: &envelope.text,
        context: &envelope.context,
    };
    let (parsed, status_ok, stderr) =
        call_gateway::<_, GatewayStoryEnhanceResponse>(state, "gateway-story-enhance", &req)
            .await?;
    if !parsed.error.trim().is_empty() {
        return Err(anyhow!(parsed.error));
    }
    if !status_ok {
        return Err(anyhow!(
            "gateway-story-enhance failed: {}",
            compact_stderr(&stderr)
        ));
    }
    Ok(parsed)
}

pub async fn submit_meta(
    state: Arc<AppState>,
    story_id: &str,
    envelope: MetaEnvelope,
) -> anyhow::Result<GatewayMetaResponse> {
    let req = GatewayMetaRequest {
        story_id,
        session_id: &envelope.session_id,
        client_turn: envelope.client_turn,
        client_revision: envelope.client_revision,
        kind: &envelope.kind,
        text: &envelope.text,
    };
    let (parsed, status_ok, stderr) =
        call_gateway::<_, GatewayMetaResponse>(state, "gateway-meta", &req).await?;
    if !parsed.error.trim().is_empty() {
        return Err(anyhow!(parsed.error));
    }
    if !status_ok {
        return Err(anyhow!("gateway-meta failed: {}", compact_stderr(&stderr)));
    }
    Ok(parsed)
}

pub async fn create_save(
    state: Arc<AppState>,
    story_id: &str,
    envelope: SaveEnvelope,
) -> anyhow::Result<GatewaySaveResponse> {
    let req = GatewaySaveRequest {
        story_id,
        session_id: &envelope.session_id,
        client_turn: envelope.client_turn,
        client_revision: envelope.client_revision,
        name: &envelope.name,
        kind: &envelope.kind,
    };
    let (parsed, status_ok, stderr) =
        call_gateway::<_, GatewaySaveResponse>(state, "gateway-save", &req).await?;
    if !parsed.error.trim().is_empty() {
        return Err(anyhow!(parsed.error));
    }
    if !status_ok {
        return Err(anyhow!("gateway-save failed: {}", compact_stderr(&stderr)));
    }
    Ok(parsed)
}

pub async fn load_save(
    state: Arc<AppState>,
    story_id: &str,
    envelope: LoadEnvelope,
) -> anyhow::Result<GatewayLoadResponse> {
    let req = GatewayLoadRequest {
        story_id,
        session_id: &envelope.session_id,
        client_turn: envelope.client_turn,
        client_revision: envelope.client_revision,
        save_id: &envelope.save_id,
    };
    let (parsed, status_ok, stderr) =
        call_gateway::<_, GatewayLoadResponse>(state, "gateway-load", &req).await?;
    if !parsed.error.trim().is_empty() {
        return Err(anyhow!(parsed.error));
    }
    if !status_ok {
        return Err(anyhow!("gateway-load failed: {}", compact_stderr(&stderr)));
    }
    Ok(parsed)
}

pub async fn delete_save(
    state: Arc<AppState>,
    story_id: &str,
    envelope: DeleteSaveEnvelope,
) -> anyhow::Result<GatewayDeleteSaveResponse> {
    let req = GatewayDeleteSaveRequest {
        story_id,
        session_id: &envelope.session_id,
        client_turn: envelope.client_turn,
        client_revision: envelope.client_revision,
        save_id: &envelope.save_id,
    };
    let (parsed, status_ok, stderr) =
        call_gateway::<_, GatewayDeleteSaveResponse>(state, "gateway-delete-save", &req).await?;
    if !parsed.error.trim().is_empty() {
        return Err(anyhow!(parsed.error));
    }
    if !status_ok {
        return Err(anyhow!(
            "gateway-delete-save failed: {}",
            compact_stderr(&stderr)
        ));
    }
    Ok(parsed)
}

async fn call_gateway<TReq, TResp>(
    state: Arc<AppState>,
    command: &str,
    req: &TReq,
) -> anyhow::Result<(TResp, bool, String)>
where
    TReq: Serialize,
    TResp: DeserializeOwned,
{
    let input = serde_json::to_vec(req).with_context(|| format!("encoding {command} request"))?;
    let mut child = gateway_command(&state, command)
        .spawn()
        .with_context(|| format!("starting {}", state.paths.oneday_bin.display()))?;

    let mut stdin = child
        .stdin
        .take()
        .with_context(|| format!("opening {command} stdin"))?;
    stdin.write_all(&input).await?;
    stdin.shutdown().await?;
    drop(stdin);

    let stdout = child
        .stdout
        .take()
        .with_context(|| format!("opening {command} stdout"))?;
    let stderr = child
        .stderr
        .take()
        .with_context(|| format!("opening {command} stderr"))?;
    let stdout_task = tokio::spawn(read_child_output(stdout));
    let stderr_task = tokio::spawn(read_child_output(stderr));

    let status = wait_for_child(
        &mut child,
        Duration::from_secs(360),
        &format!("waiting for {command}"),
    )
    .await?;
    let stdout = stdout_task
        .await
        .with_context(|| format!("joining {command} stdout reader"))??;
    let stderr = stderr_task
        .await
        .with_context(|| format!("joining {command} stderr reader"))??;
    let stderr = String::from_utf8_lossy(&stderr).to_string();
    let parsed = serde_json::from_slice(&stdout).with_context(|| {
        format!(
            "decoding {command} stdout; stderr={}",
            compact_stderr(&stderr)
        )
    })?;
    Ok((parsed, status.success(), stderr))
}

fn gateway_command(state: &AppState, command: &str) -> Command {
    let mut child = Command::new(&state.paths.oneday_bin);
    child
        .arg(command)
        .env("ONEDAY_CONFIG", &state.paths.config_path)
        .current_dir(&state.paths.oneday_root)
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .kill_on_drop(true);
    child
}

async fn read_child_output<R>(mut reader: R) -> std::io::Result<Vec<u8>>
where
    R: tokio::io::AsyncRead + Unpin,
{
    let mut output = Vec::new();
    reader.read_to_end(&mut output).await?;
    Ok(output)
}

async fn wait_for_child(
    child: &mut Child,
    timeout: Duration,
    operation: &str,
) -> anyhow::Result<ExitStatus> {
    match tokio::time::timeout(timeout, child.wait()).await {
        Ok(status) => status.with_context(|| operation.to_string()),
        Err(_) => {
            terminate_child(child).await;
            Err(anyhow!("{operation} timed out"))
        }
    }
}

async fn terminate_child(child: &mut Child) {
    if matches!(child.try_wait(), Ok(None)) {
        let _ = child.kill().await;
    }
    let _ = child.wait().await;
}

fn default_save_kind() -> String {
    "manual".to_string()
}

fn default_start_story() -> bool {
    true
}

fn compact_stderr(stderr: &str) -> String {
    let mut text = stderr.trim().replace('\n', " ");
    if text.len() > 800 {
        text.truncate(800);
        text.push_str("...");
    }
    text
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{config, AppState};
    use std::fs;
    use std::io::Write;
    use std::path::PathBuf;
    use tokio::sync::broadcast;

    #[cfg(unix)]
    use std::os::unix::fs::PermissionsExt;

    #[tokio::test]
    async fn timed_out_child_is_killed_and_reaped() {
        let mut command = Command::new("bash");
        command.arg("-c").arg("sleep 30").kill_on_drop(true);
        let mut child = command.spawn().expect("spawn sleeping child");

        let err = wait_for_child(
            &mut child,
            Duration::from_millis(25),
            "waiting for sleeping child",
        )
        .await
        .expect_err("sleeping child should time out");

        assert!(err.to_string().contains("timed out"), "{err}");
        assert!(
            child.try_wait().expect("inspect child status").is_some(),
            "timed-out child must be reaped"
        );
    }

    #[tokio::test]
    async fn call_gateway_turn_stream_broadcasts_live_but_returns_final_only() {
        let script = fake_oneday_input_script(&[
            r#"{"event":{"id":"idem:live:1","story_id":"story-1","session_id":"session-1","turn":1,"type":"narrative.delta","payload":{"text":"Hello"},"created_at":"2026-01-01T00:00:00Z"},"phase":"live"}"#,
            r#"{"event":{"id":"idem:1","story_id":"story-1","session_id":"session-1","turn":1,"type":"turn.started","payload":{},"created_at":"2026-01-01T00:00:01Z"},"phase":"final"}"#,
            r#"{"event":{"id":"idem:2","story_id":"story-1","session_id":"session-1","turn":1,"type":"turn.committed","payload":{},"created_at":"2026-01-01T00:00:02Z"},"phase":"final"}"#,
            r#"{"done":true}"#,
        ]);
        let state = test_state(script).await;
        let mut rx = state.turn_events.subscribe();
        let action = PlayerAction {
            kind: "free_text".to_string(),
            text: "look".to_string(),
            choice_id: 0,
        };
        let caps = ClientCapabilities::default();
        let req = GatewayTurnRequest {
            story_id: "story-1",
            session_id: "session-1",
            client_turn: 1,
            client_revision: 1,
            idempotency_key: "idem",
            action: &action,
            stream: true,
            capabilities: &caps,
        };

        let resp = call_gateway_turn_stream(
            state.clone(),
            &req,
            "story-1",
            1,
            "free_text".to_string(),
            "look".to_string(),
        )
        .await
        .expect("stream response");

        assert_eq!(resp.events.len(), 2);
        assert_eq!(resp.events[0]["id"], "idem:1");
        assert_eq!(resp.events[1]["id"], "idem:2");

        let live = rx.recv().await.expect("live event");
        let final_started = rx.recv().await.expect("final started event");
        let final_committed = rx.recv().await.expect("final committed event");
        assert_eq!(live.event_type.as_deref(), Some("narrative.delta"));
        assert_eq!(final_started.event_type.as_deref(), Some("turn.started"));
        assert_eq!(
            final_committed.event_type.as_deref(),
            Some("turn.committed")
        );
    }

    #[tokio::test]
    async fn minigame_bridge_preserves_player_instance_contract() {
        let script = fake_oneday_input_script(&[
            r#"{"instance":{"protocol_version":1,"id":"mini-1","definition":{"id":"deduction","kind":"deduction","difficulty":50},"runtime":{"phase":"active","revision":1}}}"#,
        ]);
        let state = test_state(script).await;
        let response = start_minigame(
            state,
            "story-1",
            MiniGameStartEnvelope {
                definition: serde_json::json!({"id":"deduction","kind":"deduction","difficulty":50}),
                selection: serde_json::Value::Null,
            },
        )
        .await
        .expect("minigame bridge");
        let instance = response.instance.expect("instance");
        assert_eq!(instance["id"], "mini-1");
        assert_eq!(instance["runtime"]["phase"], "active");
    }

    #[tokio::test]
    async fn audio_bridge_preserves_provider_and_branch_asset_contracts() {
        let script = fake_oneday_input_script(&[
            r#"{"providers":[{"id":"cloud","available":false,"reason":"disabled"}],"assets":[{"id":"audio-1","branch_id":"branch-1","source_commit_id":"commit-1","status":"ready"}],"jobs":[]}"#,
        ]);
        let state = test_state(script).await;
        let response = audio(
            state,
            serde_json::json!({"operation":"message-get","story_id":"story-1","message_id":42}),
        )
        .await
        .expect("audio bridge");
        assert_eq!(response.providers[0]["reason"], "disabled");
        assert_eq!(response.assets[0]["branch_id"], "branch-1");
        assert_eq!(response.assets[0]["source_commit_id"], "commit-1");
    }

    #[tokio::test]
    async fn timeline_list_retries_one_transient_bridge_failure() {
        let root =
            std::env::temp_dir().join(format!("oneday-timeline-retry-test-{}", Uuid::new_v4()));
        fs::create_dir_all(&root).expect("create temp root");
        let script = root.join("oneday-fake");
        let counter = root.join("attempts");
        let response = r#"{"active_branch_id":"branch-main","revision":7,"branches":[],"head":null,"commits":[]}"#;
        fs::write(
            &script,
            format!(
                "#!/usr/bin/env bash\ncat >/dev/null\nif [ ! -f '{}' ]; then touch '{}'; echo 'database is locked' >&2; exit 1; fi\nprintf '%s\\n' '{}'\n",
                counter.display(),
                counter.display(),
                response.replace('\'', "'\\''"),
            ),
        )
        .expect("write retry script");
        #[cfg(unix)]
        {
            let mut permissions = fs::metadata(&script)
                .expect("script metadata")
                .permissions();
            permissions.set_mode(0o755);
            fs::set_permissions(&script, permissions).expect("chmod retry script");
        }
        let state = test_state(script).await;

        let response = timeline(
            state,
            "story-1",
            TimelineEnvelope {
                action: "list".into(),
                client_revision: 0,
                branch_id: String::new(),
                from_commit_id: String::new(),
                name: String::new(),
            },
        )
        .await
        .expect("timeline retry response");

        assert_eq!(response.active_branch_id, "branch-main");
        assert_eq!(response.revision, 7);
        assert!(counter.exists());
    }

    #[test]
    fn canonical_minigame_fixture_is_player_safe() {
        let fixture: serde_json::Value =
            serde_json::from_str(include_str!("../../contracts/minigame-v1.json")).unwrap();
        assert_eq!(fixture["instance"]["protocol_version"], 1);
        assert_eq!(fixture["instance"]["runtime"]["phase"], "active");
        assert_eq!(fixture["input"]["action"], "submit");
        assert!(fixture["instance"]["definition"].get("answers").is_none());
    }

    #[tokio::test]
    async fn call_gateway_turn_stream_requires_done_line() {
        let script = fake_oneday_input_script(&[
            r#"{"event":{"id":"idem:1","story_id":"story-1","session_id":"session-1","turn":1,"type":"turn.committed","payload":{},"created_at":"2026-01-01T00:00:00Z"},"phase":"final"}"#,
        ]);
        let state = test_state(script).await;
        let action = PlayerAction {
            kind: "free_text".to_string(),
            text: "look".to_string(),
            choice_id: 0,
        };
        let caps = ClientCapabilities::default();
        let req = GatewayTurnRequest {
            story_id: "story-1",
            session_id: "session-1",
            client_turn: 1,
            client_revision: 1,
            idempotency_key: "idem",
            action: &action,
            stream: true,
            capabilities: &caps,
        };

        let err = call_gateway_turn_stream(
            state,
            &req,
            "story-1",
            1,
            "free_text".to_string(),
            "look".to_string(),
        )
        .await
        .expect_err("missing done should fail");

        assert!(err.to_string().contains("before done line"), "{err}");
    }

    #[tokio::test]
    async fn call_gateway_turn_stream_error_event_returns_error() {
        let script = fake_oneday_input_script(&[
            r#"{"event":{"id":"idem:live:1","story_id":"story-1","session_id":"session-1","turn":1,"type":"error","payload":{"message":"provider failed"},"created_at":"2026-01-01T00:00:00Z"},"phase":"live"}"#,
        ]);
        let state = test_state(script).await;
        let mut rx = state.turn_events.subscribe();
        let action = PlayerAction {
            kind: "free_text".to_string(),
            text: "look".to_string(),
            choice_id: 0,
        };
        let caps = ClientCapabilities::default();
        let req = GatewayTurnRequest {
            story_id: "story-1",
            session_id: "session-1",
            client_turn: 1,
            client_revision: 1,
            idempotency_key: "idem",
            action: &action,
            stream: true,
            capabilities: &caps,
        };

        let err = call_gateway_turn_stream(
            state,
            &req,
            "story-1",
            1,
            "free_text".to_string(),
            "look".to_string(),
        )
        .await
        .expect_err("error event should fail");

        assert!(err.to_string().contains("provider failed"), "{err}");
        let event = rx.recv().await.expect("broadcast error event");
        assert_eq!(event.event_type.as_deref(), Some("error"));
    }

    async fn test_state(oneday_bin: PathBuf) -> Arc<AppState> {
        let pool = sqlx::SqlitePool::connect("sqlite::memory:")
            .await
            .expect("memory sqlite");
        let root = oneday_bin.parent().expect("script parent").to_path_buf();
        let (turn_events, _) = broadcast::channel(16);
        Arc::new(AppState {
            pool,
            paths: config::ResolvedPaths {
                oneday_root: root.clone(),
                config_path: root.join("config.yaml"),
                db_path: root.join("oneday.db"),
                oneday_bin,
                static_dir: root.clone(),
                visual_asset_dir: root.join("visual_assets"),
            },
            turn_events,
            visual_workers: Arc::new(tokio::sync::Semaphore::new(4)),
        })
    }

    fn fake_oneday_input_script(lines: &[&str]) -> PathBuf {
        let root = std::env::temp_dir().join(format!("oneday-gateway-test-{}", Uuid::new_v4()));
        fs::create_dir_all(&root).expect("create temp root");
        let script = root.join("oneday-fake");
        let mut file = fs::File::create(&script).expect("create fake oneday script");
        writeln!(file, "#!/usr/bin/env bash\ncat >/dev/null").expect("write input reader");
        for line in lines {
            writeln!(file, "printf '%s\\n' '{}'", line.replace('\'', "'\\''"))
                .expect("write response line");
        }
        #[cfg(unix)]
        {
            let mut permissions = file.metadata().expect("script metadata").permissions();
            permissions.set_mode(0o755);
            fs::set_permissions(&script, permissions).expect("chmod fake script");
        }
        script
    }
}
