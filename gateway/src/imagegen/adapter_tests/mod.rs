use super::common::*;
use super::*;
use axum::{
    extract::Json,
    http::{header, HeaderMap},
    routing::{get, post},
    Router,
};
use base64::Engine;
use reqwest::Client;
use serde_json::{json, Value};
use std::collections::HashMap;
use tokio::sync::oneshot;

const PNG: &[u8] = b"\x89PNG\r\n\x1a\nfixture";

fn direct_config(provider: &str, base_url: String) -> AdapterConfig {
    let capability_probe_url = if provider == "openai-compatible" {
        format!("{}/images/capabilities", base_url.trim_end_matches('/'))
    } else {
        String::new()
    };
    AdapterConfig {
        provider: provider.into(),
        map_icon_provider: provider.into(),
        providers: HashMap::from([(
            provider.to_string(),
            ProviderConfig {
                base_url,
                api_key: "test-key".into(),
                auth_mode: "bearer".into(),
                capability_probe_url,
                api_version: String::new(),
            },
        )]),
        base_url: String::new(),
        api_key: String::new(),
        openclaw_url: String::new(),
        bridge_url: String::new(),
        bridge_token: String::new(),
        bridge_provider: "codex-responses".into(),
        bridge_map_icon_provider: "codex-responses".into(),
        bridge_fallbacks: Vec::new(),
        bridge_fallback_policy: "on_error".into(),
        bridge_compatibility: "normalize".into(),
    }
}

fn direct_request(model: &str) -> GenerateRequest {
    GenerateRequest {
        subject: "test subject".into(),
        prompt: "A clean landmark".into(),
        negative_prompt: "text".into(),
        model: model.into(),
        is_map_icon: false,
        size: "1024x1024".into(),
        resolution: Some("1K".into()),
        aspect_ratio: Some("1:1".into()),
        quality: "medium".into(),
        output_format: "png".into(),
        background: "opaque".into(),
        timeout_ms: 30_000,
        idempotency_key: "oneday-contract-test".into(),
    }
}

mod azure;
mod codex;
mod common;
mod fal;
mod gemini;
mod openai;
mod replicate;
mod security;
mod stability;
