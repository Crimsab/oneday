#[cfg(test)]
mod adapter_tests;
mod azure;
mod codex_bridge;
mod common;
mod fal;
mod gemini;
mod openai;
mod openclaw;
mod operations;
mod replicate;
mod stability;

use common::compact_detail;
use reqwest::Client;
use std::collections::HashMap;
use std::fmt;

pub(crate) use operations::{
    canonicalize_source, normalize_mask, validate_native_request, CanonicalImage, ImageOperation,
    MaskRaster, NativeImageRequest,
};

pub(super) const MAX_IMAGE_BYTES: usize = 32 * 1024 * 1024;
pub(super) const MAX_RESPONSE_BYTES: usize = 48 * 1024 * 1024;

#[derive(Debug, Clone)]
pub(crate) struct AdapterConfig {
    pub provider: String,
    pub map_icon_provider: String,
    pub base_url: String,
    pub api_key: String,
    pub providers: HashMap<String, ProviderConfig>,
    pub openclaw_url: String,
    pub bridge_url: String,
    pub bridge_token: String,
    pub bridge_provider: String,
    pub bridge_map_icon_provider: String,
    pub bridge_fallbacks: Vec<String>,
    pub bridge_fallback_policy: String,
    pub bridge_compatibility: String,
}

#[derive(Debug, Clone, Default)]
pub(crate) struct ProviderConfig {
    pub base_url: String,
    pub api_key: String,
    pub auth_mode: String,
    pub capability_probe_url: String,
    pub api_version: String,
}

#[derive(Debug, Clone)]
pub(crate) struct GenerateRequest {
    pub subject: String,
    pub prompt: String,
    pub negative_prompt: String,
    pub model: String,
    pub is_map_icon: bool,
    pub size: String,
    pub resolution: Option<String>,
    pub aspect_ratio: Option<String>,
    pub quality: String,
    pub output_format: String,
    pub background: String,
    pub timeout_ms: u64,
    pub idempotency_key: String,
}

#[derive(Debug)]
pub(crate) struct GeneratedImage {
    pub bytes: Vec<u8>,
    pub output_format: String,
    pub revised_prompt: String,
    pub provider_label: String,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) enum AdapterKind {
    TextOnly,
    CodexOAuth,
    OpenClaw,
    OpenAi,
    OpenAiCompatible,
    Gemini,
    Fal,
    Replicate,
    Stability,
    AzureOpenAi,
}

pub(crate) fn adapter_kind(provider: &str) -> Option<AdapterKind> {
    match provider.trim().to_ascii_lowercase().as_str() {
        "none" | "text-only" => Some(AdapterKind::TextOnly),
        "codex-oauth" | "imagegen-bridge" | "imagegen_bridge" | "bridge-native" => {
            Some(AdapterKind::CodexOAuth)
        }
        "openclaw" | "openclaw-bridge" => Some(AdapterKind::OpenClaw),
        "openai" => Some(AdapterKind::OpenAi),
        "openai-compatible" | "litellm" => Some(AdapterKind::OpenAiCompatible),
        "gemini" => Some(AdapterKind::Gemini),
        "fal" => Some(AdapterKind::Fal),
        "replicate" => Some(AdapterKind::Replicate),
        "stability" => Some(AdapterKind::Stability),
        "azure-openai" => Some(AdapterKind::AzureOpenAi),
        _ => None,
    }
}

pub(crate) fn is_available(config: &AdapterConfig, model: &str) -> bool {
    validation_error(config, model, false).is_none()
}

pub(crate) async fn generate(
    client: &Client,
    config: &AdapterConfig,
    request: &GenerateRequest,
) -> anyhow::Result<GeneratedImage> {
    let provider = selected_provider(config, request.is_map_icon);
    if let Some(error) = validation_error(config, &request.model, request.is_map_icon) {
        return Err(provider_error(provider, "configuration", error, false));
    }
    let kind = adapter_kind(provider).expect("validated image provider");
    let result = match kind {
        AdapterKind::TextOnly => Err(provider_error(
            provider,
            "text_only",
            "text-only mode disables image generation",
            false,
        )),
        AdapterKind::CodexOAuth => codex_bridge::generate(client, config, request).await,
        AdapterKind::OpenClaw => openclaw::generate(client, config, request).await,
        AdapterKind::OpenAi | AdapterKind::OpenAiCompatible => {
            openai::generate(client, config, request, provider).await
        }
        AdapterKind::Gemini => gemini::generate(client, config, request, provider).await,
        AdapterKind::Fal => fal::generate(client, config, request, provider).await,
        AdapterKind::Replicate => replicate::generate(client, config, request, provider).await,
        AdapterKind::Stability => stability::generate(client, config, request, provider).await,
        AdapterKind::AzureOpenAi => azure::generate(client, config, request, provider).await,
    };
    result.map_err(|error| {
        if error.downcast_ref::<ProviderError>().is_some() {
            error
        } else {
            provider_error(provider, "invalid_response", error.to_string(), false)
        }
    })
}

pub(crate) async fn operate(
    client: &Client,
    config: &AdapterConfig,
    request: &NativeImageRequest,
) -> anyhow::Result<GeneratedImage> {
    operations::validate_native_request(config, request)?;
    // Capability and shape validation deliberately runs again at this last
    // boundary so stale UI/catalog data cannot reach a provider.
    let result = match adapter_kind(&request.provider) {
        Some(AdapterKind::CodexOAuth) => codex_bridge::edit(client, config, request).await,
        Some(AdapterKind::OpenAi) => openai::edit(client, config, request).await,
        Some(AdapterKind::AzureOpenAi) => azure::edit(client, config, request).await,
        Some(AdapterKind::Gemini) => gemini::edit(client, config, request).await,
        Some(AdapterKind::Stability) => stability::edit(client, config, request).await,
        _ => Err(provider_error(
            &request.provider,
            "CAPABILITY_UNSUPPORTED",
            format!(
                "{}:{} does not implement {}",
                request.provider,
                request.model,
                request.operation.as_str()
            ),
            false,
        )),
    };
    result.map_err(|error| {
        if error.downcast_ref::<ProviderError>().is_some() {
            error
        } else {
            provider_error(
                &request.provider,
                "PROVIDER_BAD_RESPONSE",
                error.to_string(),
                false,
            )
        }
    })
}

pub(crate) fn validation_error(
    config: &AdapterConfig,
    model: &str,
    is_map_icon: bool,
) -> Option<String> {
    let provider = selected_provider(config, is_map_icon);
    if provider.is_empty() {
        return Some("image provider is not selected".to_string());
    }
    let Some(kind) = adapter_kind(provider) else {
        return Some(format!("image provider {provider:?} is not supported"));
    };
    if kind == AdapterKind::TextOnly {
        return Some("text-only mode disables image generation".to_string());
    }
    if model.trim().is_empty() {
        return Some(format!("image model is required for provider {provider}"));
    }
    let direct = provider_config(config, provider);
    let missing = match kind {
        AdapterKind::CodexOAuth => {
            validate_bridge_endpoint(&config.bridge_url, &config.bridge_token).err()
        }
        AdapterKind::OpenClaw => config
            .openclaw_url
            .trim()
            .is_empty()
            .then(|| "OpenClaw bridge URL".to_string()),
        _ if direct.base_url.trim().is_empty() => Some("base URL".to_string()),
        AdapterKind::OpenAiCompatible => validate_compatible_config(&direct).err(),
        _ if direct.api_key.trim().is_empty() => Some("server-side API key".to_string()),
        _ => None,
    };
    if let Some(missing) = missing {
        return Some(format!("{provider} is not configured: missing {missing}"));
    }
    validate_model(kind, model).err()
}

fn selected_provider(config: &AdapterConfig, is_map_icon: bool) -> &str {
    if is_map_icon && !config.map_icon_provider.trim().is_empty() {
        config.map_icon_provider.trim()
    } else {
        config.provider.trim()
    }
}

fn validate_model(kind: AdapterKind, model: &str) -> Result<(), String> {
    let model = model.trim();
    match kind {
        AdapterKind::CodexOAuth | AdapterKind::OpenAi => {
            let model = model.strip_prefix("openai/").unwrap_or(model);
            if matches!(
                model,
                "gpt-image-2" | "gpt-image-1.5" | "gpt-image-1" | "gpt-image-1-mini"
            ) {
                Ok(())
            } else {
                Err(format!(
                    "model {model:?} is not a supported GPT Image model"
                ))
            }
        }
        AdapterKind::Fal if !model.contains('/') => {
            Err("fal.ai model must use a vendor/model slug".to_string())
        }
        AdapterKind::Replicate if model.split('/').count() != 2 => {
            Err("Replicate model must use an owner/model slug".to_string())
        }
        AdapterKind::Stability if model != "stable-image-core" => {
            Err("Stability currently supports only stable-image-core".to_string())
        }
        _ => Ok(()),
    }
}

fn validate_compatible_config(config: &ProviderConfig) -> Result<(), String> {
    let auth_mode = compatible_auth_mode(config)?;
    if auth_mode == "bearer" && config.api_key.trim().is_empty() {
        return Err("server-side API key".to_string());
    }
    let probe = config.capability_probe_url.trim();
    if probe.is_empty() {
        return Err("explicit image capability probe URL".to_string());
    }
    if !common::same_origin(&config.base_url, probe).map_err(|error| error.to_string())? {
        return Err("capability probe URL must be same-origin with base URL".to_string());
    }
    Ok(())
}

pub(super) fn compatible_auth_mode(config: &ProviderConfig) -> Result<&str, String> {
    match config.auth_mode.trim() {
        "" | "bearer" => Ok("bearer"),
        "none" => Ok("none"),
        value => Err(format!(
            "unsupported OpenAI-compatible auth_mode {value:?}; use bearer or none"
        )),
    }
}

pub(super) fn validate_bridge_endpoint(url: &str, token: &str) -> Result<(), String> {
    let parsed = reqwest::Url::parse(url.trim())
        .map_err(|_| "imagegen-bridge URL must be an HTTP or HTTPS URL".to_string())?;
    let host = parsed
        .host_str()
        .ok_or_else(|| "imagegen-bridge URL must include a host".to_string())?;
    match parsed.scheme() {
        "http" if is_loopback_host(host) => Ok(()),
        "https" if !token.trim().is_empty() => Ok(()),
        "https" => Err("remote imagegen-bridge requires a bearer token".to_string()),
        "http" => Err("remote imagegen-bridge requires HTTPS and a bearer token".to_string()),
        _ => Err("imagegen-bridge URL must be an HTTP or HTTPS URL".to_string()),
    }
}

fn is_loopback_host(host: &str) -> bool {
    host.eq_ignore_ascii_case("localhost")
        || host
            .parse::<std::net::IpAddr>()
            .is_ok_and(|ip| ip.is_loopback())
}

pub(super) fn provider_config(config: &AdapterConfig, provider: &str) -> ProviderConfig {
    let normalized = provider.trim().to_ascii_lowercase();
    let canonical = match normalized.as_str() {
        "litellm" => "openai-compatible",
        value => value,
    };
    if let Some(value) = config.providers.get(canonical) {
        let mut value = value.clone();
        if value.base_url.trim().is_empty() {
            value.base_url = default_base_url(canonical).to_string();
        }
        return value;
    }
    ProviderConfig {
        base_url: if config.base_url.trim().is_empty() {
            default_base_url(canonical).to_string()
        } else {
            config.base_url.clone()
        },
        api_key: config.api_key.clone(),
        auth_mode: "bearer".to_string(),
        capability_probe_url: String::new(),
        api_version: String::new(),
    }
}

fn default_base_url(provider: &str) -> &'static str {
    match provider {
        "openai" => "https://api.openai.com/v1",
        "gemini" => "https://generativelanguage.googleapis.com/v1beta",
        "fal" => "https://queue.fal.run",
        "replicate" => "https://api.replicate.com/v1",
        "stability" => "https://api.stability.ai/v2beta",
        _ => "",
    }
}

#[derive(Debug)]
struct ProviderError {
    code: &'static str,
    provider: String,
    detail: String,
    retryable: bool,
}

impl fmt::Display for ProviderError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{} [{}]: {}", self.provider, self.code, self.detail)
    }
}

impl std::error::Error for ProviderError {}

pub(super) fn provider_error(
    provider: &str,
    code: &'static str,
    detail: impl Into<String>,
    retryable: bool,
) -> anyhow::Error {
    ProviderError {
        code,
        provider: provider.to_string(),
        detail: detail.into(),
        retryable,
    }
    .into()
}

pub(crate) fn is_retryable(error: &anyhow::Error) -> bool {
    error
        .chain()
        .find_map(|cause| cause.downcast_ref::<ProviderError>())
        .is_some_and(|error| error.retryable)
}

pub(crate) fn error_code(error: &anyhow::Error) -> &'static str {
    error
        .chain()
        .find_map(|cause| cause.downcast_ref::<ProviderError>())
        .map_or("internal_error", |error| error.code)
}

pub(crate) fn operation_error(code: &'static str, detail: impl Into<String>) -> anyhow::Error {
    provider_error("image-operation", code, detail, false)
}

pub(super) fn http_error(provider: &str, status: reqwest::StatusCode, raw: &[u8]) -> anyhow::Error {
    let (code, retryable) = match status.as_u16() {
        400 | 404 | 409 | 422 => ("invalid_request", false),
        401 | 403 => ("authentication", false),
        408 | 429 => ("rate_limited", true),
        500..=599 => ("upstream_unavailable", true),
        _ => ("upstream_error", false),
    };
    provider_error(
        provider,
        code,
        format!("HTTP {status}: {}", compact_detail(raw)),
        retryable,
    )
}

pub(super) fn transport_error(
    provider: &str,
    operation: &str,
    error: reqwest::Error,
) -> anyhow::Error {
    // Once a request body has been handed to the HTTP client, a connection
    // failure or timeout cannot prove that an upstream did not accept a paid
    // generation. Keep the job terminal rather than dispatching it again.
    provider_error(
        provider,
        "unknown_outcome",
        format!("{operation}: {error}; provider outcome is unknown and will not be retried"),
        false,
    )
}

pub(super) fn probe_transport_error(
    provider: &str,
    operation: &str,
    error: reqwest::Error,
) -> anyhow::Error {
    provider_error(
        provider,
        if error.is_timeout() {
            "probe_timeout"
        } else {
            "probe_transport"
        },
        format!("{operation}: {error}"),
        error.is_connect() || error.is_timeout(),
    )
}
