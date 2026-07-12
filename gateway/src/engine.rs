use crate::{events::TurnStreamEvent, gateway_protocol as protocol, AppState};
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

pub async fn command_descriptors(
    state: Arc<AppState>,
) -> anyhow::Result<protocol::CommandDescriptorsResponse> {
    let output = run_gateway_command(
        &state,
        "gateway-command-descriptors",
        None,
        Duration::from_secs(30),
    )
    .await?;

    let stderr = String::from_utf8_lossy(&output.stderr).to_string();
    let parsed = serde_json::from_slice::<protocol::CommandDescriptorsResponse>(&output.stdout)
        .with_context(|| {
            format!(
                "decoding gateway-command-descriptors stdout; stderr={}",
                compact_stderr(&stderr)
            )
        })?;
    if let Some(error) = parsed.error.as_deref().filter(|error| !error.trim().is_empty()) {
        return Err(anyhow!(error.to_owned()));
    }
    if !output.status.success() {
        return Err(anyhow!(
            "gateway-command-descriptors failed: {}",
            compact_stderr(&stderr)
        ));
    }
    Ok(parsed)
}

pub async fn model_settings(state: Arc<AppState>) -> anyhow::Result<protocol::ModelRoutingSettings> {
    let output = run_gateway_command(
        &state,
        "gateway-model-settings",
        None,
        Duration::from_secs(30),
    )
    .await?;

    let stderr = String::from_utf8_lossy(&output.stderr).to_string();
    let parsed = serde_json::from_slice::<protocol::ModelSettingsResponse>(&output.stdout)
        .with_context(|| {
            format!(
                "decoding gateway-model-settings stdout; stderr={}",
                compact_stderr(&stderr)
            )
        })?;
    if let Some(error) = parsed.error.as_deref().filter(|error| !error.trim().is_empty()) {
        return Err(bridge_error(
            parsed.error_code.as_deref().unwrap_or("model_settings_failed"),
            error,
        ));
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
    update: protocol::ModelRoutingUpdate,
) -> anyhow::Result<protocol::ModelRoutingSettings> {
    let (parsed, status_ok, stderr) = call_gateway::<_, protocol::ModelSettingsResponse>(
        state,
        "gateway-model-settings-update",
        &update,
    )
    .await?;
    if let Some(error) = parsed.error.as_deref().filter(|error| !error.trim().is_empty()) {
        return Err(bridge_error(
            parsed.error_code.as_deref().unwrap_or("model_settings_update_failed"),
            error,
        ));
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
) -> anyhow::Result<protocol::TurnResponse> {
    if envelope.idempotency_key.trim().is_empty() {
        envelope.idempotency_key = Uuid::new_v4().to_string();
    }

    let req = protocol::SubmitActionRequest {
        story_id: story_id.to_string(),
        session_id: envelope.session_id,
        client_turn: envelope.client_turn,
        client_revision: envelope.client_revision,
        idempotency_key: envelope.idempotency_key,
        action: protocol::PlayerAction {
            kind: envelope.action.kind.clone(),
            text: (!envelope.action.text.is_empty()).then_some(envelope.action.text.clone()),
            choice_id: (envelope.action.choice_id != 0).then_some(envelope.action.choice_id),
        },
        stream: Some(envelope.stream),
        capabilities: Some(protocol::ClientCapabilities {
            images: Some(envelope.capabilities.images),
            ascii: Some(envelope.capabilities.ascii),
            roll_log: Some(envelope.capabilities.roll_log),
        }),
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
        call_gateway::<_, protocol::TurnResponse>(state, "gateway-turn", &req).await?;
    if let Some(error) = parsed.error.as_deref().filter(|error| !error.trim().is_empty()) {
        return Err(anyhow!(error.to_string()));
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
) -> anyhow::Result<protocol::CraftingResponse> {
    let request = protocol::CraftRequest {
        story_id: story_id.to_string(),
        message: envelope.message,
        history: envelope
            .history
            .into_iter()
            .map(|message| protocol::Message {
                role: message.role,
                content: message.content,
            })
            .collect(),
    };
    let (parsed, status_ok, stderr) =
        call_gateway::<_, protocol::CraftResponse>(state, "gateway-craft", &request).await?;
    if let Some(error) = parsed.error.as_deref().filter(|error| !error.trim().is_empty()) {
        return Err(anyhow!(error.to_string()));
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
) -> anyhow::Result<protocol::BrowserTimelineResponse> {
    let action = envelope.action;
    let request = protocol::BrowserTimelineRequest {
        story_id: story_id.to_string(),
        action: action.clone(),
        client_revision: envelope.client_revision,
        branch_id: (!envelope.branch_id.is_empty()).then_some(envelope.branch_id),
        from_commit_id: (!envelope.from_commit_id.is_empty()).then_some(envelope.from_commit_id),
        name: (!envelope.name.is_empty()).then_some(envelope.name),
    };
    let first = call_timeline_gateway(state.clone(), &request).await;
    match first {
        Ok(response) => Ok(response),
        Err(first_error) if action == "list" => {
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
    request: &protocol::BrowserTimelineRequest,
) -> anyhow::Result<protocol::BrowserTimelineResponse> {
    let (parsed, status_ok, stderr) =
        call_gateway::<_, protocol::BrowserTimelineResponse>(state, "gateway-timeline", request)
            .await?;
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
) -> anyhow::Result<protocol::MiniGameResponse> {
    let definition = serde_json::from_value(envelope.definition)
        .context("decoding minigame definition against gateway contract")?;
    let selection = if envelope.selection.is_null() {
        None
    } else {
        Some(
            serde_json::from_value(envelope.selection)
                .context("decoding minigame selection against gateway contract")?,
        )
    };
    call_minigame_gateway(
        state,
        "gateway-minigame-start",
        protocol::MiniGameRequest {
            story_id: story_id.to_owned(),
            instance_id: None,
            kind: None,
            definition: Some(definition),
            input: None,
            selection,
        },
    )
    .await
}

pub async fn active_minigame(
    state: Arc<AppState>,
    story_id: &str,
) -> anyhow::Result<protocol::MiniGameResponse> {
    call_minigame_gateway(
        state,
        "gateway-minigame-get",
        protocol::MiniGameRequest {
            story_id: story_id.to_owned(),
            instance_id: None,
            kind: None,
            definition: None,
            input: None,
            selection: None,
        },
    )
    .await
}

pub async fn input_minigame(
    state: Arc<AppState>,
    story_id: &str,
    instance_id: &str,
    envelope: MiniGameInputEnvelope,
) -> anyhow::Result<protocol::MiniGameResponse> {
    let input = serde_json::from_value(envelope.input)
        .context("decoding minigame input against gateway contract")?;
    call_minigame_gateway(
        state,
        "gateway-minigame-input",
        protocol::MiniGameRequest {
            story_id: story_id.to_owned(),
            instance_id: Some(instance_id.to_owned()),
            kind: None,
            definition: None,
            input: Some(input),
            selection: None,
        },
    )
    .await
}

async fn call_minigame_gateway(
    state: Arc<AppState>,
    command: &str,
    request: protocol::MiniGameRequest,
) -> anyhow::Result<protocol::MiniGameResponse> {
    let (parsed, status_ok, stderr) =
        call_gateway::<_, protocol::MiniGameResponse>(state, command, &request).await?;
    if let Some(error) = parsed.error.as_deref().filter(|error| !error.trim().is_empty()) {
        return Err(anyhow!(error.to_owned()));
    }
    if !status_ok {
        return Err(anyhow!("{command} failed: {}", compact_stderr(&stderr)));
    }
    Ok(parsed)
}

pub async fn audio(
    state: Arc<AppState>,
    request: protocol::AudioRequest,
) -> anyhow::Result<protocol::AudioResponse> {
    let (parsed, status_ok, stderr) =
        call_gateway::<_, protocol::AudioResponse>(state, "gateway-audio", &request).await?;
    if let Some(error) = parsed.error.as_deref().filter(|error| !error.trim().is_empty()) {
        return Err(anyhow!(error.to_owned()));
    }
    if !status_ok {
        return Err(anyhow!("gateway-audio failed: {}", compact_stderr(&stderr)));
    }
    Ok(parsed)
}

async fn call_gateway_turn_stream(
    state: Arc<AppState>,
    req: &protocol::SubmitActionRequest,
    story_id: &str,
    client_turn: i64,
    action_kind: String,
    action_text: String,
) -> anyhow::Result<protocol::TurnResponse> {
    let input = serde_json::to_vec(req).context("encoding gateway-turn stream request")?;
    let mut child = gateway_command(&state, "gateway-turn")
        .spawn()
        .with_context(|| format!("starting {}", state.paths.oneday_bin.display()))?;

    let mut stdin = child.stdin.take().context("opening gateway-turn stdin")?;
    let stdout = child.stdout.take().context("opening gateway-turn stdout")?;
    let stderr = child.stderr.take().context("opening gateway-turn stderr")?;
    let stderr_task = tokio::spawn(read_child_output(stderr, MAX_GATEWAY_STDERR_BYTES));

    let mut events = Vec::new();
    let mut saw_done = false;
    let mut lines = BufReader::new(stdout).lines();
    let run_result = tokio::time::timeout(Duration::from_secs(360), async {
        stdin.write_all(&input).await?;
        stdin.shutdown().await?;
        drop(stdin);
        while let Some(line) = lines.next_line().await? {
            let line = line.trim();
            if line.is_empty() {
                continue;
            }
            let parsed: protocol::TurnStreamLine = serde_json::from_str(line)
                .with_context(|| format!("decoding gateway-turn stream line: {line}"))?;
            if let Some(error) = parsed.error.as_deref().filter(|error| !error.trim().is_empty()) {
                return Err(anyhow!(error.to_string()));
            }
            if let Some(event) = parsed.event {
                let event_type = event.type_.clone();
                let event_json = serde_json::to_value(&event)
                    .context("encoding typed gateway-turn stream event")?;
                let _ = state.turn_events.send(TurnStreamEvent::contract(
                    story_id,
                    client_turn,
                    &action_kind,
                    &action_text,
                    &event_json,
                ));
                let is_error = event_type == "error";
                let error_message = event
                    .payload
                    .as_ref()
                    .and_then(|payload| payload.get("message"))
                    .and_then(serde_json::Value::as_str)
                    .unwrap_or("gateway-turn stream failed")
                    .to_string();
                if parsed.phase.as_deref() != Some("live") {
                    events.push(event);
                }
                if is_error {
                    return Err(anyhow!(error_message));
                }
            }
            if parsed.done.unwrap_or(false) {
                saw_done = true;
                break;
            }
        }
        if !saw_done {
            return Err(anyhow!("gateway-turn stream ended before done line"));
        }
        child
            .wait()
            .await
            .context("waiting for gateway-turn stream")
    })
    .await;

    let status = match run_result {
        Ok(Ok(status)) => status,
        Ok(Err(err)) => {
            let cleanup = terminate_child(&mut child).await;
            let _ = stderr_task.await;
            return match cleanup {
                Ok(()) => Err(err),
                Err(cleanup_err) => Err(err.context(format!(
                    "gateway-turn cleanup failed: {cleanup_err}"
                ))),
            };
        }
        Err(_) => {
            let cleanup = terminate_child(&mut child).await;
            let _ = stderr_task.await;
            return match cleanup {
                Ok(()) => Err(anyhow!("gateway-turn stream timed out")),
                Err(cleanup_err) => Err(anyhow!(
                    "gateway-turn stream timed out; cleanup failed: {cleanup_err}"
                )),
            };
        }
    };
    let stderr = stderr_task
        .await
        .context("joining gateway-turn stderr reader")??;
    let stderr = String::from_utf8_lossy(&stderr).to_string();

    if !status.success() {
        return Err(anyhow!(
            "gateway-turn stream failed: {}",
            compact_stderr(&stderr)
        ));
    }
    Ok(protocol::TurnResponse {
        events,
        error: None,
    })
}

pub async fn create_story(
    state: Arc<AppState>,
    envelope: StoryCreateEnvelope,
) -> anyhow::Result<protocol::StoryCreateResponse> {
    let req = protocol::StoryCreateRequest {
        brief: envelope.brief,
        character_name: envelope.character_name,
        character_background: envelope.character_background,
        start: envelope.start,
    };
    let (parsed, status_ok, stderr) =
        call_gateway::<_, protocol::StoryCreateResponse>(state, "gateway-story-create", &req)
            .await?;
    if let Some(error) = parsed.error.as_deref().filter(|error| !error.trim().is_empty()) {
        return Err(anyhow!(error.to_string()));
    }
    if !status_ok {
        return Err(anyhow!(
            "gateway-story-create failed: {}",
            compact_stderr(&stderr)
        ));
    }
    if parsed.story_id.as_deref().unwrap_or("").is_empty()
        || parsed.character_id.as_deref().unwrap_or("").is_empty()
    {
        return Err(anyhow!("gateway-story-create returned incomplete identifiers"));
    }
    Ok(parsed)
}

pub async fn story_wizard(
    state: Arc<AppState>,
    envelope: StoryWizardEnvelope,
) -> anyhow::Result<protocol::StoryWizardResponse> {
    let creator_state = envelope
        .state
        .map(serde_json::from_value)
        .transpose()
        .context("decoding story wizard state against gateway contract")?;
    let req = protocol::StoryWizardRequest {
        state: creator_state,
        input: (!envelope.input.is_empty()).then_some(envelope.input),
        action: (!envelope.action.is_empty()).then_some(envelope.action),
        start: envelope.start,
    };
    let (parsed, status_ok, stderr) =
        call_gateway::<_, protocol::StoryWizardResponse>(state, "gateway-story-wizard", &req)
            .await?;
    if let Some(error) = parsed.error.as_deref().filter(|error| !error.trim().is_empty()) {
        return Err(anyhow!(error.to_string()));
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
) -> anyhow::Result<protocol::StoryEnhanceResponse> {
    let creator_state = envelope
        .state
        .map(serde_json::from_value)
        .transpose()
        .context("decoding story enhancement state against gateway contract")?;
    let req = protocol::StoryEnhanceRequest {
        state: creator_state,
        stage: (!envelope.stage.is_empty()).then_some(envelope.stage),
        text: (!envelope.text.is_empty()).then_some(envelope.text),
        context: (!envelope.context.is_empty()).then_some(envelope.context),
    };
    let (parsed, status_ok, stderr) =
        call_gateway::<_, protocol::StoryEnhanceResponse>(state, "gateway-story-enhance", &req)
            .await?;
    if let Some(error) = parsed.error.as_deref().filter(|error| !error.trim().is_empty()) {
        return Err(anyhow!(error.to_string()));
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
) -> anyhow::Result<protocol::MetaResponse> {
    let req = protocol::BrowserMetaRequest {
        story_id: story_id.to_string(),
        session_id: envelope.session_id,
        client_turn: envelope.client_turn,
        client_revision: envelope.client_revision,
        kind: envelope.kind,
        text: (!envelope.text.is_empty()).then_some(envelope.text),
    };
    let (parsed, status_ok, stderr) =
        call_gateway::<_, protocol::MetaResponse>(state, "gateway-meta", &req).await?;
    if let Some(error) = parsed.error.as_deref().filter(|error| !error.trim().is_empty()) {
        return Err(anyhow!(error.to_string()));
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
) -> anyhow::Result<protocol::SaveResponse> {
    let req = protocol::BrowserSaveRequest {
        story_id: story_id.to_string(),
        session_id: envelope.session_id,
        client_turn: envelope.client_turn,
        client_revision: envelope.client_revision,
        name: Some(envelope.name),
        kind: Some(envelope.kind),
    };
    let (parsed, status_ok, stderr) =
        call_gateway::<_, protocol::SaveResponse>(state, "gateway-save", &req).await?;
    if let Some(error) = parsed.error.as_deref().filter(|error| !error.trim().is_empty()) {
        return Err(anyhow!(error.to_string()));
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
) -> anyhow::Result<protocol::LoadResponse> {
    let req = protocol::BrowserLoadRequest {
        story_id: story_id.to_string(),
        session_id: envelope.session_id,
        client_turn: envelope.client_turn,
        client_revision: envelope.client_revision,
        save_id: envelope.save_id,
    };
    let (parsed, status_ok, stderr) =
        call_gateway::<_, protocol::LoadResponse>(state, "gateway-load", &req).await?;
    if let Some(error) = parsed.error.as_deref().filter(|error| !error.trim().is_empty()) {
        return Err(anyhow!(error.to_string()));
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
) -> anyhow::Result<protocol::DeleteSaveResponse> {
    let req = protocol::BrowserDeleteSaveRequest {
        story_id: story_id.to_string(),
        session_id: envelope.session_id,
        client_turn: envelope.client_turn,
        client_revision: envelope.client_revision,
        save_id: envelope.save_id,
    };
    let (parsed, status_ok, stderr) =
        call_gateway::<_, protocol::DeleteSaveResponse>(state, "gateway-delete-save", &req).await?;
    if let Some(error) = parsed.error.as_deref().filter(|error| !error.trim().is_empty()) {
        return Err(anyhow!(error.to_string()));
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
    let output = run_gateway_command(
        &state,
        command,
        Some(&input),
        Duration::from_secs(360),
    )
    .await?;
    let stderr = String::from_utf8_lossy(&output.stderr).to_string();
    let parsed = match serde_json::from_slice(&output.stdout) {
        Ok(parsed) => parsed,
        Err(_) if !output.status.success() => {
            let stdout = String::from_utf8_lossy(&output.stdout);
            return Err(anyhow!(
                "{command} transport failed with {}; stderr={}; stdout={}",
                output.status,
                compact_stderr(&stderr),
                compact_stderr(&stdout),
            ));
        }
        Err(err) => {
            return Err(err).with_context(|| {
                format!(
                    "decoding {command} stdout; stderr={}",
                    compact_stderr(&stderr)
                )
            });
        }
    };
    Ok((parsed, output.status.success(), stderr))
}

const MAX_GATEWAY_STDOUT_BYTES: u64 = 16 * 1024 * 1024;
const MAX_GATEWAY_STDERR_BYTES: u64 = 1024 * 1024;

#[derive(Debug)]
struct GatewayProcessOutput {
    status: ExitStatus,
    stdout: Vec<u8>,
    stderr: Vec<u8>,
}

async fn run_gateway_command(
    state: &AppState,
    command: &str,
    input: Option<&[u8]>,
    timeout: Duration,
) -> anyhow::Result<GatewayProcessOutput> {
    let mut child = gateway_command(state, command)
        .spawn()
        .with_context(|| format!("starting {} {command}", state.paths.oneday_bin.display()))?;
    let mut stdin = child
        .stdin
        .take()
        .with_context(|| format!("opening {command} stdin"))?;
    let stdout = child
        .stdout
        .take()
        .with_context(|| format!("opening {command} stdout"))?;
    let stderr = child
        .stderr
        .take()
        .with_context(|| format!("opening {command} stderr"))?;
    let stdout_task = tokio::spawn(read_child_output(stdout, MAX_GATEWAY_STDOUT_BYTES));
    let stderr_task = tokio::spawn(read_child_output(stderr, MAX_GATEWAY_STDERR_BYTES));

    let run_result = tokio::time::timeout(timeout, async {
        if let Some(input) = input {
            stdin
                .write_all(input)
                .await
                .with_context(|| format!("writing {command} stdin"))?;
        }
        stdin
            .shutdown()
            .await
            .with_context(|| format!("closing {command} stdin"))?;
        drop(stdin);
        child
            .wait()
            .await
            .with_context(|| format!("waiting for {command}"))
    })
    .await;

    let status = match run_result {
        Ok(Ok(status)) => status,
        Ok(Err(err)) => {
            let cleanup = terminate_child(&mut child).await;
            let _ = stdout_task.await;
            let _ = stderr_task.await;
            return match cleanup {
                Ok(()) => Err(err),
                Err(cleanup_err) => {
                    Err(err.context(format!("{command} cleanup failed: {cleanup_err}")))
                }
            };
        }
        Err(_) => {
            let cleanup = terminate_child(&mut child).await;
            let _ = stdout_task.await;
            let _ = stderr_task.await;
            return match cleanup {
                Ok(()) => Err(anyhow!("{command} timed out")),
                Err(cleanup_err) => Err(anyhow!(
                    "{command} timed out; cleanup failed: {cleanup_err}"
                )),
            };
        }
    };

    let stdout = stdout_task
        .await
        .with_context(|| format!("joining {command} stdout reader"))??;
    let stderr = stderr_task
        .await
        .with_context(|| format!("joining {command} stderr reader"))??;
    Ok(GatewayProcessOutput {
        status,
        stdout,
        stderr,
    })
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
    #[cfg(unix)]
    child.process_group(0);
    child
}

async fn read_child_output<R>(reader: R, max_bytes: u64) -> std::io::Result<Vec<u8>>
where
    R: tokio::io::AsyncRead + Unpin,
{
    let mut reader = reader.take(max_bytes + 1);
    let mut output = Vec::new();
    reader.read_to_end(&mut output).await?;
    if output.len() as u64 > max_bytes {
        return Err(std::io::Error::new(
            std::io::ErrorKind::InvalidData,
            format!("child output exceeded {max_bytes} bytes"),
        ));
    }
    Ok(output)
}

async fn terminate_child(child: &mut Child) -> anyhow::Result<()> {
    if child.try_wait().context("checking child status")?.is_some() {
        return Ok(());
    }

    #[cfg(unix)]
    if let Some(pid) = child.id() {
        let result = unsafe { libc::kill(-(pid as i32), libc::SIGKILL) };
        if result != 0 {
            let err = std::io::Error::last_os_error();
            if err.raw_os_error() != Some(libc::ESRCH) {
                return Err(err).context("killing child process group");
            }
        }
    }

    #[cfg(not(unix))]
    child.kill().await.context("killing child")?;

    child.wait().await.context("reaping child")?;
    Ok(())
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
    async fn timed_out_gateway_kills_process_group_and_reaps_child() {
        let root = std::env::temp_dir().join(format!("oneday-timeout-test-{}", Uuid::new_v4()));
        fs::create_dir_all(&root).expect("create timeout test root");
        let script = root.join("oneday-fake");
        let parent_pid = root.join("parent.pid");
        let child_pid = root.join("child.pid");
        fs::write(
            &script,
            format!(
                "#!/usr/bin/env bash\nprintf '%s' \"$$\" > '{}'\nsleep 30 &\nprintf '%s' \"$!\" > '{}'\nwait\n",
                parent_pid.display(),
                child_pid.display(),
            ),
        )
        .expect("write timeout script");
        let mut permissions = fs::metadata(&script)
            .expect("timeout script metadata")
            .permissions();
        permissions.set_mode(0o755);
        fs::set_permissions(&script, permissions).expect("chmod timeout script");
        let state = test_state(script).await;

        let oversized_input = vec![b'x'; 1024 * 1024];
        let err = run_gateway_command(
            &state,
            "timeout-test",
            Some(&oversized_input),
            Duration::from_millis(200),
        )
        .await
        .expect_err("gateway process should time out");

        assert!(err.to_string().contains("timed out"), "{err}");
        let parent = read_pid(&parent_pid);
        let descendant = read_pid(&child_pid);
        wait_until_process_gone(parent).await;
        wait_until_process_gone(descendant).await;
    }

    fn read_pid(path: &std::path::Path) -> i32 {
        fs::read_to_string(path)
            .unwrap_or_else(|err| panic!("read {}: {err}", path.display()))
            .parse()
            .unwrap_or_else(|err| panic!("parse {}: {err}", path.display()))
    }

    async fn wait_until_process_gone(pid: i32) {
        for _ in 0..50 {
            if !process_is_running(pid) {
                return;
            }
            tokio::time::sleep(Duration::from_millis(20)).await;
        }
        panic!("process {pid} survived gateway timeout");
    }

    fn process_is_running(pid: i32) -> bool {
        let stat = match fs::read_to_string(format!("/proc/{pid}/stat")) {
            Ok(stat) => stat,
            Err(_) => return false,
        };
        stat.split_whitespace().nth(2) != Some("Z")
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
        let req = stream_test_request();

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
        assert_eq!(resp.events[0].id, "idem:1");
        assert_eq!(resp.events[1].id, "idem:2");

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
    async fn command_descriptor_bridge_preserves_typed_contract() {
        let script = fake_oneday_input_script(&[
            r#"{"commands":[{"id":"save","canonical":"/save","aliases":["s"],"title":"Save","description":"Save the story","group":"story","parity":"full","behavior":"immediate"}]}"#,
        ]);
        let state = test_state(script).await;
        let response = command_descriptors(state)
            .await
            .expect("command descriptor bridge");
        assert_eq!(response.commands[0].id, "save");
        assert_eq!(response.commands[0].aliases, vec!["s"]);
    }

    #[tokio::test]
    async fn minigame_bridge_preserves_player_instance_contract() {
        let script = fake_oneday_input_script(&[
            r#"{"instance":{"protocol_version":1,"id":"mini-1","turn":0,"seed":7,"definition":{"id":"deduction","kind":"deduction","difficulty":50},"runtime":{"phase":"active","revision":1}}}"#,
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
        assert_eq!(instance.id, "mini-1");
        assert_eq!(instance.runtime.phase, "active");
    }

    #[tokio::test]
    async fn audio_bridge_preserves_provider_and_branch_asset_contracts() {
        let script = fake_oneday_input_script(&[
            r#"{"providers":[{"id":"cloud","available":false,"reason":"disabled"}],"assets":[{"id":"audio-1","story_id":"story-1","branch_id":"branch-1","source_commit_id":"commit-1","source_message_id":42,"segment_index":0,"segment_kind":"narration","voice_profile_id":"voice-1","provider":"cloud","model":"tts-1","provider_voice_id":"voice-1","voice_version":"1","language_tag":"it-IT","pronunciation_revision":0,"text":"Test","text_hash":"hash","cache_key":"cache","style":null,"speed":1.0,"output_format":"mp3","status":"ready","duration_ms":100,"timings":null}],"jobs":[]}"#,
        ]);
        let state = test_state(script).await;
        let response = audio(
            state,
            serde_json::from_value(serde_json::json!({"operation":"message-get","story_id":"story-1","message_id":42}))
                .expect("typed audio request"),
        )
        .await
        .expect("audio bridge");
        assert_eq!(response.providers[0].reason.as_deref(), Some("disabled"));
        assert_eq!(response.assets[0].branch_id, "branch-1");
        assert_eq!(response.assets[0].source_commit_id, "commit-1");
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

    #[test]
    fn generated_gateway_contract_decodes_load_snapshot_metadata() {
        let response: crate::gateway_protocol::LoadResponse = serde_json::from_value(
            serde_json::json!({
                "save": {
                    "id": "save-1",
                    "name": "Before the gate",
                    "turn": 4,
                    "chapter": 2,
                    "created_at": "2026-01-01T00:00:00Z"
                },
                "legacy": true,
                "snapshot_state": "complete",
                "snapshot_detail": "canonical snapshot restored"
            }),
        )
        .expect("generated load contract");

        assert_eq!(response.snapshot_state, "complete");
        assert_eq!(response.snapshot_detail.as_deref(), Some("canonical snapshot restored"));
        assert_eq!(response.save.as_ref().map(|save| save.id.as_str()), Some("save-1"));
    }

    #[tokio::test]
    async fn call_gateway_turn_stream_requires_done_line() {
        let script = fake_oneday_input_script(&[
            r#"{"event":{"id":"idem:1","story_id":"story-1","session_id":"session-1","turn":1,"type":"turn.committed","payload":{},"created_at":"2026-01-01T00:00:00Z"},"phase":"final"}"#,
        ]);
        let state = test_state(script).await;
        let req = stream_test_request();

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
        let req = stream_test_request();

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

    #[tokio::test]
    async fn call_gateway_reports_transport_failure_before_json_decode() {
        let root = std::env::temp_dir().join(format!("oneday-transport-test-{}", Uuid::new_v4()));
        fs::create_dir_all(&root).expect("create transport test root");
        let script = root.join("oneday-fake");
        fs::write(
            &script,
            "#!/usr/bin/env bash\ncat >/dev/null\nprintf 'not-json'\nprintf 'bridge crashed' >&2\nexit 17\n",
        )
        .expect("write transport script");
        #[cfg(unix)]
        {
            let mut permissions = fs::metadata(&script).unwrap().permissions();
            permissions.set_mode(0o755);
            fs::set_permissions(&script, permissions).unwrap();
        }
        let state = test_state(script).await;

        let err = call_gateway::<_, serde_json::Value>(
            state,
            "gateway-crash",
            &serde_json::json!({"request": true}),
        )
        .await
        .expect_err("invalid stdout from failed process must be a transport error");
        let message = err.to_string();

        assert!(message.contains("transport failed"), "{message}");
        assert!(message.contains("bridge crashed"), "{message}");
        assert!(!message.starts_with("decoding"), "{message}");
    }

    fn stream_test_request() -> protocol::SubmitActionRequest {
        protocol::SubmitActionRequest {
            story_id: "story-1".into(),
            session_id: "session-1".into(),
            client_turn: 1,
            client_revision: 1,
            idempotency_key: "idem".into(),
            action: protocol::PlayerAction {
                kind: "free_text".into(),
                text: Some("look".into()),
                choice_id: None,
            },
            stream: Some(true),
            capabilities: Some(protocol::ClientCapabilities {
                images: Some(false),
                ascii: Some(false),
                roll_log: Some(false),
            }),
        }
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
