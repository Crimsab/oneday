use super::common::*;
use super::{
    http_error, transport_error, AdapterConfig, GenerateRequest, GeneratedImage, MAX_RESPONSE_BYTES,
};
use anyhow::{anyhow, Context};
use reqwest::Client;
use serde::Deserialize;
use serde_json::json;

pub(super) async fn generate(
    client: &Client,
    config: &AdapterConfig,
    request: &GenerateRequest,
) -> anyhow::Result<GeneratedImage> {
    let mut payload = json!({
        "prompt": legacy_prompt(request),
        "output_format": request.output_format
    });
    set_optional_string(&mut payload, "size", Some(&request.size));
    set_optional_string(&mut payload, "resolution", request.resolution.as_deref());
    set_optional_string(
        &mut payload,
        "aspect_ratio",
        request.aspect_ratio.as_deref(),
    );
    set_optional_string(&mut payload, "background", Some(&request.background));
    let model = normalize_openclaw_model(&request.model);
    set_optional_string(&mut payload, "model", Some(&model));
    let response = client
        .post(config.openclaw_url.trim())
        .json(&payload)
        .send()
        .await
        .map_err(|error| transport_error("openclaw-bridge", "requesting OpenClaw image", error))?;
    let status = response.status();
    let raw = read_limited(response, MAX_RESPONSE_BYTES).await?;
    if !status.is_success() {
        return Err(http_error("openclaw-bridge", status, &raw));
    }
    let response: OpenClawResponse =
        serde_json::from_slice(&raw).context("decoding OpenClaw image response")?;
    if !response.ok {
        return Err(anyhow!(
            "OpenClaw image bridge failed: {}",
            response.error.as_deref().unwrap_or("unknown error")
        ));
    }
    let encoded = response
        .image_b64
        .as_deref()
        .filter(|value| !value.trim().is_empty())
        .ok_or_else(|| anyhow!("OpenClaw image bridge returned no image_b64"))?;
    let bytes = decode_and_validate(encoded, &request.output_format)?;
    Ok(GeneratedImage {
        bytes,
        output_format: normalize_format(&request.output_format),
        revised_prompt: response.revised_prompt.unwrap_or_default(),
        provider_label: format!("{}:{}", config.provider.trim(), model),
    })
}

#[derive(Debug, Deserialize)]
struct OpenClawResponse {
    ok: bool,
    image_b64: Option<String>,
    revised_prompt: Option<String>,
    error: Option<String>,
}
