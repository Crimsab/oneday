use super::common::*;
use super::{
    http_error, provider_config, transport_error, AdapterConfig, GenerateRequest, GeneratedImage,
    MAX_RESPONSE_BYTES,
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
    let mut payload = json!({
        "model": request.model,
        "prompt": legacy_prompt(request),
        "size": request.size,
        "output_format": request.output_format,
        "n": 1
    });
    set_optional_string(&mut payload, "quality", Some(&request.quality));
    set_optional_string(&mut payload, "background", Some(&request.background));
    let endpoint = format!(
        "{}/images/generations",
        direct.base_url.trim_end_matches('/')
    );
    let response = client
        .post(endpoint)
        .bearer_auth(direct.api_key.trim())
        .header("Idempotency-Key", &request.idempotency_key)
        .json(&payload)
        .send()
        .await
        .map_err(|error| transport_error(provider, "requesting image", error))?;
    let status = response.status();
    let raw = read_limited(response, MAX_RESPONSE_BYTES).await?;
    if !status.is_success() {
        return Err(http_error(provider, status, &raw));
    }
    let response: OpenAiResponse =
        serde_json::from_slice(&raw).context("decoding image generation response")?;
    let first = response
        .data
        .first()
        .ok_or_else(|| anyhow!("image provider returned no images"))?;
    let bytes = if let Some(encoded) = first.b64_json.as_deref() {
        decode_and_validate(encoded, &request.output_format)?
    } else if let Some(url) = first.url.as_deref() {
        download_image(
            client,
            url,
            &request.output_format,
            &request.subject,
            Some(&direct.base_url),
        )
        .await?
    } else {
        return Err(anyhow!("image provider returned neither b64_json nor url"));
    };
    Ok(GeneratedImage {
        bytes,
        output_format: normalize_format(&request.output_format),
        revised_prompt: first.revised_prompt.clone().unwrap_or_default(),
        provider_label: format!("{}:{}", provider, request.model.trim()),
    })
}

#[derive(Debug, Deserialize)]
struct OpenAiResponse {
    data: Vec<OpenAiImageData>,
}

#[derive(Debug, Deserialize)]
struct OpenAiImageData {
    b64_json: Option<String>,
    revised_prompt: Option<String>,
    url: Option<String>,
}
