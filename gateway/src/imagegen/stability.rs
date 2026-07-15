use super::common::*;
use super::{
    http_error, provider_config, provider_error, transport_error, AdapterConfig, GenerateRequest,
    GeneratedImage, MAX_RESPONSE_BYTES,
};
use reqwest::Client;

pub(super) async fn edit(
    client: &Client,
    config: &AdapterConfig,
    request: &super::NativeImageRequest,
) -> anyhow::Result<GeneratedImage> {
    let direct = provider_config(config, &request.provider);
    let source = request.source.as_ref().expect("native request validated");
    let mask = request.mask.as_ref().expect("native request validated");
    let format = normalize_format(&request.output_format);
    let mut form = reqwest::multipart::Form::new()
        .text("prompt", request.prompt.clone())
        .text("output_format", format.clone())
        .part(
            "image",
            reqwest::multipart::Part::bytes(source.png.clone())
                .file_name("source.png")
                .mime_str("image/png")?,
        )
        .part(
            "mask",
            reqwest::multipart::Part::bytes(mask.png.clone())
                .file_name("mask.png")
                .mime_str("image/png")?,
        );
    if !request.negative_prompt.trim().is_empty() {
        form = form.text("negative_prompt", request.negative_prompt.clone());
    }
    let endpoint = format!(
        "{}/stable-image/edit/inpaint",
        direct.base_url.trim_end_matches('/')
    );
    let response = client
        .post(endpoint)
        .bearer_auth(direct.api_key.trim())
        .header("accept", format!("image/{format}"))
        .header("Idempotency-Key", &request.idempotency_key)
        .multipart(form)
        .send()
        .await
        .map_err(|error| {
            transport_error(&request.provider, "requesting Stability inpaint", error)
        })?;
    let status = response.status();
    let raw = read_limited(response, MAX_RESPONSE_BYTES).await?;
    if !status.is_success() {
        return Err(http_error(&request.provider, status, &raw));
    }
    validate_image(&raw, &format)?;
    Ok(GeneratedImage {
        bytes: raw,
        output_format: format,
        revised_prompt: String::new(),
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
    if request.background.trim() == "transparent" {
        return Err(provider_error(
            provider,
            "unsupported_parameter",
            "Stable Image Core does not expose transparent background generation",
            false,
        ));
    }
    let format = normalize_format(&request.output_format);
    let mut form = reqwest::multipart::Form::new()
        .text("prompt", request.prompt.clone())
        .text("output_format", format.clone());
    if !request.negative_prompt.trim().is_empty() {
        form = form.text("negative_prompt", request.negative_prompt.clone());
    }
    if let Some(aspect_ratio) = request
        .aspect_ratio
        .clone()
        .or_else(|| aspect_ratio_from_size(&request.size))
    {
        form = form.text("aspect_ratio", aspect_ratio);
    }
    let endpoint = format!(
        "{}/stable-image/generate/core",
        direct.base_url.trim_end_matches('/')
    );
    let response = client
        .post(endpoint)
        .bearer_auth(direct.api_key.trim())
        .header("accept", format!("image/{format}"))
        .multipart(form)
        .send()
        .await
        .map_err(|error| transport_error(provider, "requesting Stability image", error))?;
    let status = response.status();
    let raw = read_limited(response, MAX_RESPONSE_BYTES).await?;
    if !status.is_success() {
        return Err(http_error(provider, status, &raw));
    }
    validate_image(&raw, &format)?;
    Ok(GeneratedImage {
        bytes: raw,
        output_format: format,
        revised_prompt: String::new(),
        provider_label: format!("{provider}:{}", request.model.trim()),
    })
}
