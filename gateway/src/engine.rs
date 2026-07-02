use crate::AppState;
use anyhow::{anyhow, Context};
use serde::{Deserialize, Serialize};
use std::process::Stdio;
use std::sync::Arc;
use std::time::Duration;
use tokio::io::AsyncWriteExt;
use tokio::process::Command;
use uuid::Uuid;

#[derive(Debug, Deserialize, Serialize)]
pub struct ActionEnvelope {
    pub session_id: String,
    pub client_turn: i64,
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

#[derive(Debug, Serialize)]
struct GatewayTurnRequest<'a> {
    story_id: &'a str,
    session_id: &'a str,
    client_turn: i64,
    idempotency_key: &'a str,
    action: &'a PlayerAction,
    stream: bool,
    capabilities: &'a ClientCapabilities,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct GatewayTurnResponse {
    #[serde(default)]
    pub events: Vec<serde_json::Value>,
    #[serde(default)]
    pub error: String,
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
        idempotency_key: &envelope.idempotency_key,
        action: &envelope.action,
        stream: envelope.stream,
        capabilities: &envelope.capabilities,
    };
    let input = serde_json::to_vec(&req).context("encoding gateway-turn request")?;

    let mut child = Command::new(&state.paths.oneday_bin)
        .arg("gateway-turn")
        .current_dir(&state.paths.oneday_root)
        .stdin(Stdio::piped())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .with_context(|| format!("starting {}", state.paths.oneday_bin.display()))?;

    let mut stdin = child.stdin.take().context("opening gateway-turn stdin")?;
    stdin.write_all(&input).await?;
    stdin.shutdown().await?;

    let output = tokio::time::timeout(Duration::from_secs(360), child.wait_with_output())
        .await
        .context("gateway-turn timed out")?
        .context("waiting for gateway-turn")?;

    let parsed: GatewayTurnResponse =
        serde_json::from_slice(&output.stdout).with_context(|| {
            let stderr = String::from_utf8_lossy(&output.stderr);
            format!(
                "decoding gateway-turn stdout; stderr={}",
                compact_stderr(&stderr)
            )
        })?;

    if !parsed.error.trim().is_empty() {
        return Err(anyhow!(parsed.error));
    }
    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        return Err(anyhow!("gateway-turn failed: {}", compact_stderr(&stderr)));
    }
    Ok(parsed)
}

fn compact_stderr(stderr: &str) -> String {
    let mut text = stderr.trim().replace('\n', " ");
    if text.len() > 800 {
        text.truncate(800);
        text.push_str("...");
    }
    text
}
