use crate::AppState;
use anyhow::{anyhow, Context};
use serde::de::DeserializeOwned;
use serde::{Deserialize, Serialize};
use std::process::Stdio;
use std::sync::Arc;
use std::time::Duration;
use tokio::io::AsyncWriteExt;
use tokio::process::Command;
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

#[derive(Debug, Deserialize, Serialize)]
pub struct GatewayTurnResponse {
    #[serde(default)]
    pub events: Vec<serde_json::Value>,
    #[serde(default)]
    pub error: String,
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
pub struct GatewayCommandDescriptorsResponse {
    #[serde(default)]
    pub commands: Vec<serde_json::Value>,
    #[serde(default)]
    pub error: String,
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
    pub embedding_providers: Vec<String>,
    pub active: ModelRoutingActive,
    pub tts_status: String,
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
    pub embedding_provider: Option<String>,
    pub embedding_model: Option<String>,
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
    let mut child = Command::new(&state.paths.oneday_bin)
        .arg(command)
        .env("ONEDAY_CONFIG", &state.paths.config_path)
        .current_dir(&state.paths.oneday_root)
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .with_context(|| format!("starting {}", state.paths.oneday_bin.display()))?;

    let mut stdin = child
        .stdin
        .take()
        .with_context(|| format!("opening {command} stdin"))?;
    stdin.write_all(&input).await?;
    stdin.shutdown().await?;

    let output = tokio::time::timeout(Duration::from_secs(360), child.wait_with_output())
        .await
        .with_context(|| format!("{command} timed out"))?
        .with_context(|| format!("waiting for {command}"))?;

    let stderr = String::from_utf8_lossy(&output.stderr).to_string();
    let parsed = serde_json::from_slice(&output.stdout).with_context(|| {
        format!(
            "decoding {command} stdout; stderr={}",
            compact_stderr(&stderr)
        )
    })?;
    Ok((parsed, output.status.success(), stderr))
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
