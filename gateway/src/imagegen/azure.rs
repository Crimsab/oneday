use super::common::*;
use super::{
    http_error, provider_config, provider_error, transport_error, AdapterConfig, GenerateRequest,
    GeneratedImage, NativeImageRequest, MAX_RESPONSE_BYTES,
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
    let format = normalize_format(&request.output_format);
    if format == "webp" {
        return Err(provider_error(
            &request.provider,
            "CAPABILITY_UNSUPPORTED",
            "Azure image edits support png or jpeg output",
            false,
        ));
    }
    let source = request.source.as_ref().expect("native request validated");
    let mut form = reqwest::multipart::Form::new()
        .text("prompt", request.prompt.clone())
        .text("n", "1")
        .text("output_format", format.clone())
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
    if let Some(mask) = &request.mask {
        form = form.part(
            "mask",
            reqwest::multipart::Part::bytes(super::operations::alpha_edit_mask(mask)?)
                .file_name("mask.png")
                .mime_str("image/png")?,
        );
    }
    let mut endpoint = reqwest::Url::parse(direct.base_url.trim_end_matches('/'))
        .context("parsing Azure OpenAI endpoint")?;
    {
        let mut segments = endpoint
            .path_segments_mut()
            .map_err(|_| anyhow!("Azure OpenAI endpoint cannot be a base URL"))?;
        segments.pop_if_empty();
        segments.extend([
            "openai",
            "deployments",
            request.model.trim(),
            "images",
            "edits",
        ]);
    }
    endpoint.query_pairs_mut().append_pair(
        "api-version",
        if direct.api_version.trim().is_empty() {
            "preview"
        } else {
            direct.api_version.trim()
        },
    );
    let response = client
        .post(endpoint)
        .header("api-key", direct.api_key.trim())
        .header("Idempotency-Key", &request.idempotency_key)
        .multipart(form)
        .send()
        .await
        .map_err(|error| {
            transport_error(&request.provider, "requesting Azure image edit", error)
        })?;
    super::openai::decode_edit_response(client, response, request, &direct.base_url, &format).await
}

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
            "Azure OpenAI Images supports png or jpeg output",
            false,
        ));
    }
    let mut payload = json!({
        "model": request.model,
        "prompt": legacy_prompt(request),
        "size": request.size,
        "output_format": format,
        "n": 1
    });
    set_optional_string(&mut payload, "quality", Some(&request.quality));
    set_optional_string(&mut payload, "background", Some(&request.background));
    let base = direct.base_url.trim_end_matches('/');
    let mut endpoint = if base.ends_with("/openai/v1") {
        format!("{base}/images/generations")
    } else {
        format!("{base}/openai/v1/images/generations")
    };
    endpoint.push_str("?api-version=");
    endpoint.push_str(if direct.api_version.trim().is_empty() {
        "preview"
    } else {
        direct.api_version.trim()
    });
    let response = client
        .post(endpoint)
        .header("api-key", direct.api_key.trim())
        .header("Idempotency-Key", &request.idempotency_key)
        .json(&payload)
        .send()
        .await
        .map_err(|error| transport_error(provider, "requesting Azure OpenAI image", error))?;
    let status = response.status();
    let raw = read_limited(response, MAX_RESPONSE_BYTES).await?;
    if !status.is_success() {
        return Err(http_error(provider, status, &raw));
    }
    let response: OpenAiResponse =
        serde_json::from_slice(&raw).context("decoding Azure OpenAI image response")?;
    let first = response
        .data
        .first()
        .ok_or_else(|| anyhow!("Azure OpenAI returned no images"))?;
    let bytes = if let Some(encoded) = first.b64_json.as_deref() {
        decode_and_validate(encoded, &format)?
    } else if let Some(url) = first.url.as_deref() {
        download_image(
            client,
            url,
            &format,
            &request.subject,
            Some(&direct.base_url),
        )
        .await?
    } else {
        return Err(anyhow!(
            "Azure OpenAI returned neither b64_json nor image URL"
        ));
    };
    Ok(GeneratedImage {
        bytes,
        output_format: format,
        revised_prompt: first.revised_prompt.clone().unwrap_or_default(),
        provider_label: format!("{provider}:{}", request.model.trim()),
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
