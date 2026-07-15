use super::common::*;
use super::{
    provider_error, transport_error, AdapterConfig, GenerateRequest, GeneratedImage,
    NativeImageRequest, MAX_RESPONSE_BYTES,
};
use anyhow::{anyhow, Context};
use reqwest::Client;
use serde::Deserialize;
use serde_json::{json, Value};

pub(super) async fn edit(
    client: &Client,
    config: &AdapterConfig,
    request: &NativeImageRequest,
) -> anyhow::Result<GeneratedImage> {
    if request.mask.is_some() {
        return Err(provider_error(
            "codex-oauth",
            "CAPABILITY_UNSUPPORTED",
            "Codex OAuth providers currently reject raster masks",
            false,
        ));
    }
    let route = config.bridge_provider.trim();
    if !matches!(route, "codex-responses" | "codex-app-server") {
        return Err(provider_error(
            "codex-oauth",
            "configuration",
            "imagegen-bridge provider must be codex-responses or codex-app-server",
            false,
        ));
    }
    verify_edit_capability(client, config, route, &request.model).await?;
    let source = request.source.as_ref().expect("native request validated");
    let mut form = reqwest::multipart::Form::new()
        .text("prompt", request.prompt.clone())
        .text("model", normalize_bridge_model(&request.model))
        .text("provider", route.to_string())
        .text("n", "1")
        .text("response_format", "b64_json")
        .text("output_format", normalize_format(&request.output_format))
        .text(
            "compatibility",
            clean_or(&config.bridge_compatibility, "strict"),
        )
        .part(
            "image",
            reqwest::multipart::Part::bytes(source.png.clone())
                .file_name("source.png")
                .mime_str("image/png")?,
        );
    if !request.size.trim().is_empty() {
        form = form.text("size", request.size.clone());
    }
    if !request.quality.trim().is_empty() {
        form = form.text("quality", request.quality.clone());
    }
    let endpoint = format!(
        "{}/v1/images/edits",
        config.bridge_url.trim_end_matches('/')
    );
    let mut builder = client
        .post(endpoint)
        .header("Idempotency-Key", &request.idempotency_key)
        .multipart(form);
    if !config.bridge_token.trim().is_empty() {
        builder = builder.bearer_auth(config.bridge_token.trim());
    }
    let response = builder.send().await.map_err(|error| {
        transport_error("codex-oauth", "requesting imagegen-bridge edit", error)
    })?;
    let status = response.status();
    let raw = read_limited(response, MAX_RESPONSE_BYTES).await?;
    if !status.is_success() {
        return Err(provider_error(
            "codex-oauth",
            bridge_status_code(status),
            format!("HTTP {status}: {}", bridge_error_detail(&raw)),
            matches!(status.as_u16(), 408 | 429 | 500..=599),
        ));
    }
    let response: CompatibleBridgeResponse =
        serde_json::from_slice(&raw).context("decoding imagegen-bridge edit response")?;
    let first = response
        .data
        .first()
        .ok_or_else(|| anyhow!("imagegen-bridge edit returned no images"))?;
    let encoded = first
        .b64_json
        .as_deref()
        .filter(|value| !value.trim().is_empty())
        .ok_or_else(|| anyhow!("imagegen-bridge edit returned no b64_json"))?;
    let format = normalize_format(&request.output_format);
    Ok(GeneratedImage {
        bytes: decode_and_validate(encoded, &format)?,
        output_format: format,
        revised_prompt: first.revised_prompt.clone().unwrap_or_default(),
        provider_label: format!(
            "codex-oauth/{}:{}",
            response.imagegen_bridge.provider, response.imagegen_bridge.model
        ),
    })
}

async fn verify_edit_capability(
    client: &Client,
    config: &AdapterConfig,
    provider: &str,
    model: &str,
) -> anyhow::Result<()> {
    let mut endpoint = reqwest::Url::parse(&format!(
        "{}/v1/providers/{provider}/capabilities",
        config.bridge_url.trim_end_matches('/')
    ))?;
    endpoint.query_pairs_mut().append_pair("model", model);
    let mut builder = client.get(endpoint);
    if !config.bridge_token.trim().is_empty() {
        builder = builder.bearer_auth(config.bridge_token.trim());
    }
    let response = builder.send().await.map_err(|error| {
        transport_error(
            "codex-oauth",
            "probing imagegen-bridge edit capability",
            error,
        )
    })?;
    let status = response.status();
    let raw = read_limited(response, MAX_RESPONSE_BYTES).await?;
    if !status.is_success() {
        return Err(provider_error(
            "codex-oauth",
            bridge_status_code(status),
            format!(
                "capability probe HTTP {status}: {}",
                bridge_error_detail(&raw)
            ),
            matches!(status.as_u16(), 408 | 429 | 500..=599),
        ));
    }
    let value: Value = serde_json::from_slice(&raw).context("decoding bridge capabilities")?;
    let edits = value.get("edits").and_then(Value::as_bool) == Some(true);
    let image_support = value
        .pointer("/edit_images/support")
        .and_then(Value::as_str)
        .is_some_and(|support| support != "unsupported");
    if !edits || !image_support {
        return Err(provider_error(
            "codex-oauth",
            "CAPABILITY_UNSUPPORTED",
            format!("bridge route {provider}:{model} does not advertise source-image edits"),
            false,
        ));
    }
    Ok(())
}

fn bridge_status_code(status: reqwest::StatusCode) -> &'static str {
    match status.as_u16() {
        400 | 404 | 409 | 422 => "invalid_request",
        401 | 403 => "authentication",
        408 | 429 => "rate_limited",
        500..=599 => "upstream_unavailable",
        _ => "upstream_error",
    }
}

pub(super) async fn generate(
    client: &Client,
    config: &AdapterConfig,
    request: &GenerateRequest,
) -> anyhow::Result<GeneratedImage> {
    let provider = if request.is_map_icon && !config.bridge_map_icon_provider.trim().is_empty() {
        config.bridge_map_icon_provider.trim()
    } else {
        config.bridge_provider.trim()
    };
    let fallbacks = config
        .bridge_fallbacks
        .iter()
        .filter_map(|route| parse_fallback_route(route))
        .collect::<Vec<_>>();
    if !matches!(provider, "codex-responses" | "codex-app-server") {
        return Err(provider_error(
            "codex-oauth",
            "configuration",
            "imagegen-bridge provider must be codex-responses or codex-app-server",
            false,
        ));
    }
    if let Some(invalid) = fallbacks.iter().find(|route| {
        !matches!(
            route.get("provider").and_then(Value::as_str),
            Some("codex-responses" | "codex-app-server")
        )
    }) {
        return Err(provider_error(
            "codex-oauth",
            "configuration",
            format!("non-Codex bridge fallback is not allowed: {invalid}"),
            false,
        ));
    }
    let mut parameters = json!({
        "n": 1,
        "size": request.size,
        "output_format": request.output_format,
        "failure_policy": "fail_fast",
        "action": "auto"
    });
    set_optional_string(&mut parameters, "resolution", request.resolution.as_deref());
    set_optional_string(
        &mut parameters,
        "aspect_ratio",
        request.aspect_ratio.as_deref(),
    );
    set_optional_string(&mut parameters, "quality", Some(&request.quality));
    set_optional_string(&mut parameters, "background", Some(&request.background));

    let mut routing = json!({
        "fallbacks": fallbacks,
        "fallback_policy": clean_or(&config.bridge_fallback_policy, "on_unavailable")
    });
    set_optional_string(&mut routing, "provider", Some(provider));
    set_optional_string(
        &mut routing,
        "model",
        Some(&normalize_bridge_model(&request.model)),
    );

    let mut payload = json!({
        "version": "1",
        "prompt": request.prompt,
        "operation": "generate",
        "parameters": parameters,
        "routing": routing,
        "session": {"mode": "isolated"},
        "output": {
            "response_format": "b64_json",
            "transparency": {"mode": "auto"}
        },
        "policies": {
            "compatibility": clean_or(&config.bridge_compatibility, "normalize"),
            "negative_prompt": "auto",
            "revised_prompt": "include"
        },
        "idempotency_key": request.idempotency_key,
        "timeout_ms": request.timeout_ms
    });
    set_optional_string(
        &mut payload,
        "negative_prompt",
        Some(request.negative_prompt.trim()),
    );

    let endpoint = format!("{}/v1/images", config.bridge_url.trim_end_matches('/'));
    let mut builder = client.post(endpoint).json(&payload);
    if !config.bridge_token.trim().is_empty() {
        builder = builder.bearer_auth(config.bridge_token.trim());
    }
    let response = builder.send().await.map_err(|error| {
        transport_error("codex-oauth", "requesting imagegen-bridge image", error)
    })?;
    let status = response.status();
    let raw = read_limited(response, MAX_RESPONSE_BYTES).await?;
    if !status.is_success() {
        let detail = bridge_error_detail(&raw);
        let (code, retryable) = match status.as_u16() {
            401 | 403 => ("authentication", false),
            400 | 404 | 409 | 422 => ("invalid_request", false),
            408 | 429 => ("rate_limited", true),
            500..=599 => ("upstream_unavailable", true),
            _ => ("upstream_error", false),
        };
        return Err(provider_error(
            "codex-oauth",
            code,
            format!("HTTP {status}: {detail}"),
            retryable,
        ));
    }
    let response: BridgeResponse =
        serde_json::from_slice(&raw).context("decoding imagegen-bridge response")?;
    let first = response
        .data
        .first()
        .ok_or_else(|| anyhow!("imagegen-bridge returned no images"))?;
    if first.kind != "b64_json" {
        return Err(anyhow!(
            "imagegen-bridge returned unsupported payload type {}",
            first.kind
        ));
    }
    let encoded = first
        .b64_json
        .as_deref()
        .filter(|value| !value.trim().is_empty())
        .ok_or_else(|| anyhow!("imagegen-bridge returned no b64_json"))?;
    let bytes = decode_and_validate(encoded, &first.format)?;
    Ok(GeneratedImage {
        bytes,
        output_format: normalize_format(&first.format),
        revised_prompt: response.revised_prompt.unwrap_or_default(),
        provider_label: format!("codex-oauth/{}:{}", response.provider, response.model),
    })
}

#[derive(Debug, Deserialize)]
struct BridgeResponse {
    provider: String,
    model: String,
    data: Vec<BridgeImageData>,
    revised_prompt: Option<String>,
}

#[derive(Debug, Deserialize)]
struct BridgeImageData {
    #[serde(rename = "type")]
    kind: String,
    b64_json: Option<String>,
    format: String,
}

#[derive(Debug, Deserialize)]
struct CompatibleBridgeResponse {
    data: Vec<CompatibleBridgeImageData>,
    imagegen_bridge: CompatibleBridgeMetadata,
}

#[derive(Debug, Deserialize)]
struct CompatibleBridgeImageData {
    b64_json: Option<String>,
    revised_prompt: Option<String>,
}

#[derive(Debug, Deserialize)]
struct CompatibleBridgeMetadata {
    provider: String,
    model: String,
}

#[derive(Debug, Deserialize)]
struct BridgeErrorEnvelope {
    error: Option<BridgeErrorBody>,
}

#[derive(Debug, Deserialize)]
struct BridgeErrorBody {
    code: Option<String>,
    message: String,
}

fn bridge_error_detail(raw: &[u8]) -> String {
    serde_json::from_slice::<BridgeErrorEnvelope>(raw)
        .ok()
        .and_then(|value| value.error)
        .map(|error| match error.code {
            Some(code) => format!("{code}: {}", error.message),
            None => error.message,
        })
        .filter(|value| !value.trim().is_empty())
        .unwrap_or_else(|| compact_detail(raw))
}
