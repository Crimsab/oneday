use chrono::Utc;
use serde::Serialize;
use serde_json::{json, Value};

#[derive(Clone, Debug, Serialize)]
pub struct TurnStreamEvent {
    pub story_id: String,
    pub status: String,
    pub client_turn: Option<i64>,
    pub action_kind: Option<String>,
    pub action_text: Option<String>,
    pub event_type: Option<String>,
    pub event: Option<Value>,
    pub message_key: String,
    pub message_args: Value,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error_code: Option<String>,
    pub message: String,
    pub created_at: String,
}

impl TurnStreamEvent {
    pub fn status(
        story_id: &str,
        status: &str,
        client_turn: i64,
        action_kind: &str,
        action_text: &str,
        message: impl Into<String>,
    ) -> Self {
        Self {
            story_id: story_id.to_string(),
            status: status.to_string(),
            client_turn: Some(client_turn),
            action_kind: Some(action_kind.to_string()),
            action_text: Some(action_text.to_string()),
            event_type: None,
            event: None,
            message_key: format!("turn.status.{status}"),
            message_args: json!({}),
            error_code: None,
            message: message.into(),
            created_at: Utc::now().to_rfc3339(),
        }
    }

    pub fn status_error(
        story_id: &str,
        client_turn: i64,
        action_kind: &str,
        action_text: &str,
        error_code: &str,
        message: impl Into<String>,
    ) -> Self {
        let mut event = Self::status(
            story_id,
            "failed",
            client_turn,
            action_kind,
            action_text,
            message,
        );
        event.error_code = Some(error_code.to_string());
        event
    }

    pub fn contract(
        story_id: &str,
        client_turn: i64,
        action_kind: &str,
        action_text: &str,
        event: &Value,
    ) -> Self {
        let event_type = event
            .get("type")
            .and_then(Value::as_str)
            .unwrap_or("turn.event")
            .to_string();
        Self {
            story_id: story_id.to_string(),
            status: "event".to_string(),
            client_turn: Some(client_turn),
            action_kind: Some(action_kind.to_string()),
            action_text: Some(action_text.to_string()),
            event_type: Some(event_type.clone()),
            event: Some(event.clone()),
            message_key: format!("turn.event.{event_type}"),
            message_args: json!({}),
            error_code: None,
            message: contract_event_message(&event_type),
            created_at: Utc::now().to_rfc3339(),
        }
    }

    pub fn snapshot_changed(story_id: &str, turn: i64, revision: i64) -> Self {
        Self {
            story_id: story_id.to_string(),
            status: "snapshot_changed".to_string(),
            client_turn: Some(turn),
            action_kind: None,
            action_text: None,
            event_type: Some("snapshot.changed".to_string()),
            event: Some(json!({
                "type": "snapshot.changed",
                "turn": turn,
                "revision": revision,
            })),
            message_key: "turn.snapshot_changed".to_string(),
            message_args: json!({ "turn": turn, "revision": revision }),
            error_code: None,
            message: format!("Canonical snapshot advanced to turn {turn}, revision {revision}."),
            created_at: Utc::now().to_rfc3339(),
        }
    }

    pub fn visual_asset(
        story_id: &str,
        event_type: &str,
        asset_id: &str,
        job_id: Option<i64>,
        status: &str,
        message: impl Into<String>,
    ) -> Self {
        let message = message.into();
        Self {
            story_id: story_id.to_string(),
            status: "event".to_string(),
            client_turn: None,
            action_kind: None,
            action_text: None,
            event_type: Some(event_type.to_string()),
            event: Some(json!({
                "type": event_type,
                "asset_id": asset_id,
                "job_id": job_id,
                "status": status,
                "message": message,
                "payload": {
                    "asset_id": asset_id,
                    "job_id": job_id,
                    "status": status,
                    "message": message,
                },
            })),
            message_key: format!("turn.event.{event_type}"),
            message_args: json!({
                "asset_id": asset_id,
                "job_id": job_id,
                "status": status,
            }),
            error_code: None,
            message,
            created_at: Utc::now().to_rfc3339(),
        }
    }

    pub fn lagged(story_id: &str, skipped: u64) -> Self {
        Self {
            story_id: story_id.to_string(),
            status: "lagged".to_string(),
            client_turn: None,
            action_kind: None,
            action_text: None,
            event_type: Some("stream.lagged".to_string()),
            event: Some(json!({ "type": "stream.lagged", "skipped": skipped })),
            message_key: "turn.stream_lagged".to_string(),
            message_args: json!({ "skipped": skipped }),
            error_code: None,
            message: format!(
                "Missed {skipped} live turn event(s); snapshot sync will recover state."
            ),
            created_at: Utc::now().to_rfc3339(),
        }
    }
}

fn contract_event_message(event_type: &str) -> String {
    match event_type {
        "turn.started" => "Turn accepted by the live engine.".to_string(),
        "narrative.delta" => "Narrative is streaming.".to_string(),
        "narrative.final" => "Narrative generated; applying state changes.".to_string(),
        "choices.updated" => "Choices refreshed from the engine.".to_string(),
        "state.delta" => "Canonical state changed.".to_string(),
        "roll.resolved" => "Roll resolved.".to_string(),
        "challenge.started" => "Challenge started.".to_string(),
        "challenge.resolved" => "Challenge resolved.".to_string(),
        "combat.started" => "Combat started.".to_string(),
        "combat.updated" => "Combat updated.".to_string(),
        "social.started" => "Social exchange started.".to_string(),
        "social.updated" => "Social exchange updated.".to_string(),
        "asset.queued" => "Visual asset request queued.".to_string(),
        "asset.running" => "Visual asset generation started.".to_string(),
        "asset.ready" => "Visual asset is ready.".to_string(),
        "asset.failed" => "Visual asset generation failed.".to_string(),
        "asset.cancelled" => "Visual asset generation cancelled.".to_string(),
        "turn.committed" => "Turn committed to the shared story.".to_string(),
        "error" => "The engine reported an error.".to_string(),
        other => format!("Engine event: {other}."),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn visual_asset_event_carries_refreshable_asset_identity() {
        let event = TurnStreamEvent::visual_asset(
            "story-1",
            "asset.ready",
            "asset-location",
            Some(42),
            "ready",
            "Image ready.",
        );

        assert_eq!(event.story_id, "story-1");
        assert_eq!(event.status, "event");
        assert_eq!(event.event_type.as_deref(), Some("asset.ready"));
        assert_eq!(event.client_turn, None);
        let payload = event.event.expect("payload");
        assert_eq!(payload["type"], "asset.ready");
        assert_eq!(payload["asset_id"], "asset-location");
        assert_eq!(payload["job_id"], 42);
        assert_eq!(payload["status"], "ready");
        assert_eq!(event.message_key, "turn.event.asset.ready");
        assert_eq!(event.message_args["job_id"], 42);
    }

    #[test]
    fn lagged_event_exposes_count_without_parsing_legacy_prose() {
        let event = TurnStreamEvent::lagged("story-1", 3);

        assert_eq!(event.message_key, "turn.stream_lagged");
        assert_eq!(event.message_args["skipped"], 3);
        assert_eq!(event.event.expect("semantic event")["skipped"], 3);
    }

    #[test]
    fn failed_status_exposes_stable_error_code_for_sse_consumers() {
        let event = TurnStreamEvent::status_error(
            "story-1",
            4,
            "free",
            "Open the gate",
            "stale_request",
            "The request is stale.",
        );

        assert_eq!(event.status, "failed");
        assert_eq!(event.error_code.as_deref(), Some("stale_request"));
        assert_eq!(event.message_key, "turn.status.failed");
    }
}
