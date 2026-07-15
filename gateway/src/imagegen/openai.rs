use super::common::*;
use super::{
    http_error, provider_config, transport_error, AdapterConfig, GenerateRequest, GeneratedImage,
    NativeImageRequest, MAX_RESPONSE_BYTES,
};
use anyhow::{anyhow, Context};
use reqwest::Client;
use serde::Deserialize;
use serde_json::json;

pub(super) async fn edit(
    client: &Client,
    config: &AdapterConfig,
    request: &NativeImageRequest,
) -> anyhow::Result<GeneratedImage> {
    let direct = provider_config(config, &request.provider);
    let source = request.source.as_ref().expect("native request validated");
    let format = normalize_format(&request.output_format);
    let image = reqwest::multipart::Part::bytes(source.png.clone())
        .file_name("source.png")
        .mime_str("image/png")?;
    let mut form = reqwest::multipart::Form::new()
        .text("model", request.model.clone())
        .text("prompt", request.prompt.clone())
        .text("n", "1")
        .text("output_format", format.clone())
        .part("image[]", image);
    if !request.size.trim().is_empty() {
        form = form.text("size", request.size.clone());
    }
    if !request.quality.trim().is_empty() {
        form = form.text("quality", request.quality.clone());
    }
    if let Some(mask) = &request.mask {
        form = form.part(
            "mask",
            reqwest::multipart::Part::bytes(super::operations::alpha_edit_mask(mask)?)
                .file_name("mask.png")
                .mime_str("image/png")?,
        );
    }
    let endpoint = format!("{}/images/edits", direct.base_url.trim_end_matches('/'));
    let response = client
        .post(endpoint)
        .bearer_auth(direct.api_key.trim())
        .header("Idempotency-Key", &request.idempotency_key)
        .multipart(form)
        .send()
        .await
        .map_err(|error| transport_error(&request.provider, "requesting image edit", error))?;
    decode_edit_response(client, response, request, &direct.base_url, &format).await
}

pub(super) async fn decode_edit_response(
    client: &Client,
    response: reqwest::Response,
    request: &NativeImageRequest,
    trusted_origin: &str,
    format: &str,
) -> anyhow::Result<GeneratedImage> {
    let status = response.status();
    let raw = read_limited(response, MAX_RESPONSE_BYTES).await?;
    if !status.is_success() {
        return Err(http_error(&request.provider, status, &raw));
    }
    let response: OpenAiResponse =
        serde_json::from_slice(&raw).context("decoding image edit response")?;
    let first = response
        .data
        .first()
        .ok_or_else(|| anyhow!("image edit provider returned no images"))?;
    let bytes = if let Some(encoded) = first.b64_json.as_deref() {
        decode_and_validate(encoded, format)?
    } else if let Some(url) = first.url.as_deref() {
        download_image(
            client,
            url,
            format,
            "edited visual asset",
            Some(trusted_origin),
        )
        .await?
    } else {
        return Err(anyhow!("image edit returned neither b64_json nor url"));
    };
    Ok(GeneratedImage {
        bytes,
        output_format: format.to_string(),
        revised_prompt: first.revised_prompt.clone().unwrap_or_default(),
        provider_label: format!("{}:{}", request.provider, request.model.trim()),
    })
}

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
pub(super) struct OpenAiResponse {
    data: Vec<OpenAiImageData>,
}

#[derive(Debug, Deserialize)]
pub(super) struct OpenAiImageData {
    b64_json: Option<String>,
    revised_prompt: Option<String>,
    url: Option<String>,
}
