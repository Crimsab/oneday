use super::common::*;
use super::{
    http_error, provider_config, provider_error, transport_error, AdapterConfig, GenerateRequest,
    GeneratedImage, MAX_RESPONSE_BYTES,
};
use anyhow::{anyhow, Context};
use reqwest::Client;
use serde::Deserialize;
use serde_json::json;

pub(super) async fn generate(
    client: &Client,
    config: &AdapterConfig,
    request: &GenerateRequest,
    provider: &str,
) -> anyhow::Result<GeneratedImage> {
    let direct = provider_config(config, provider);
    let format = normalize_format(&request.output_format);
    if format == "webp" {
        return Err(provider_error(
            provider,
            "unsupported_parameter",
            "Gemini interactions image output supports png or jpeg in OneDay",
            false,
        ));
    }
    let mut response_format = json!({
        "type": "image",
        "mime_type": if format == "jpeg" { "image/jpeg" } else { "image/png" }
    });
    let aspect_ratio = request
        .aspect_ratio
        .clone()
        .or_else(|| aspect_ratio_from_size(&request.size));
    set_optional_string(
        &mut response_format,
        "aspect_ratio",
        aspect_ratio.as_deref(),
    );
    set_optional_string(
        &mut response_format,
        "image_size",
        request
            .resolution
            .as_deref()
            .map(|value| value.to_ascii_uppercase())
            .as_deref(),
    );
    let payload = json!({
        "model": request.model,
        "input": [{"type": "text", "text": legacy_prompt(request)}],
        "response_format": response_format
    });
    let endpoint = format!("{}/interactions", direct.base_url.trim_end_matches('/'));
    let response = client
        .post(endpoint)
        .header("x-goog-api-key", direct.api_key.trim())
        .header("Idempotency-Key", &request.idempotency_key)
        .json(&payload)
        .send()
        .await
        .map_err(|error| transport_error(provider, "requesting Gemini image", error))?;
    let status = response.status();
    let raw = read_limited(response, MAX_RESPONSE_BYTES).await?;
    if !status.is_success() {
        return Err(http_error(provider, status, &raw));
    }
    let response: GeminiResponse =
        serde_json::from_slice(&raw).context("decoding Gemini interactions response")?;
    let image = response
        .output_image
        .or_else(|| {
            response.steps.into_iter().find_map(|step| {
                step.content
                    .into_iter()
                    .find(|content| content.kind.as_deref() == Some("image"))
            })
        })
        .ok_or_else(|| anyhow!("Gemini returned no image output"))?;
    let actual_format = image
        .mime_type
        .as_deref()
        .and_then(format_from_mime)
        .unwrap_or_else(|| format.clone());
    let bytes = decode_and_validate(&image.data, &actual_format)?;
    Ok(GeneratedImage {
        bytes,
        output_format: actual_format,
        revised_prompt: String::new(),
        provider_label: format!("{provider}:{}", request.model.trim()),
    })
}

#[derive(Debug, Deserialize)]
struct GeminiResponse {
    output_image: Option<GeminiImageData>,
    #[serde(default)]
    steps: Vec<GeminiStep>,
}

#[derive(Debug, Deserialize)]
struct GeminiStep {
    #[serde(default)]
    content: Vec<GeminiImageData>,
}

#[derive(Debug, Deserialize)]
struct GeminiImageData {
    #[serde(rename = "type")]
    kind: Option<String>,
    data: String,
    mime_type: Option<String>,
}
