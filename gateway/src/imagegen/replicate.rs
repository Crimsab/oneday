use super::common::*;
use super::{
    http_error, provider_config, provider_error, transport_error, AdapterConfig, GenerateRequest,
    GeneratedImage, MAX_RESPONSE_BYTES,
};
use anyhow::{anyhow, Context};
use reqwest::Client;
use serde_json::{json, Value};
use std::time::{Duration, Instant};

pub(super) async fn generate(
    client: &Client,
    config: &AdapterConfig,
    request: &GenerateRequest,
    provider: &str,
) -> anyhow::Result<GeneratedImage> {
    let direct = provider_config(config, provider);
    validate_vendor_slug(&request.model, provider)?;
    let mut input = json!({
        "prompt": legacy_prompt(request),
        "num_outputs": 1,
        "output_format": normalize_format(&request.output_format)
    });
    let aspect_ratio = request
        .aspect_ratio
        .clone()
        .or_else(|| aspect_ratio_from_size(&request.size));
    set_optional_string(&mut input, "aspect_ratio", aspect_ratio.as_deref());
    let endpoint = format!(
        "{}/models/{}/predictions",
        direct.base_url.trim_end_matches('/'),
        request.model.trim_matches('/')
    );
    let wait_seconds = (request.timeout_ms / 1_000).clamp(1, 60);
    let response = client
        .post(endpoint)
        .bearer_auth(direct.api_key.trim())
        .header("Prefer", format!("wait={wait_seconds}"))
        .header(
            "Cancel-After",
            format!("{}s", (request.timeout_ms / 1_000).max(5)),
        )
        .header("Idempotency-Key", &request.idempotency_key)
        .json(&json!({"input": input}))
        .send()
        .await
        .map_err(|error| transport_error(provider, "submitting Replicate prediction", error))?;
    let status = response.status();
    let raw = read_limited(response, MAX_RESPONSE_BYTES).await?;
    if !status.is_success() {
        return Err(http_error(provider, status, &raw));
    }
    let mut prediction: Value =
        serde_json::from_slice(&raw).context("decoding Replicate prediction")?;
    let deadline = Instant::now() + Duration::from_millis(request.timeout_ms.max(1));
    while matches!(
        prediction.get("status").and_then(Value::as_str),
        Some("starting" | "processing")
    ) {
        if Instant::now() >= deadline {
            return Err(provider_error(
                provider,
                "timeout_after_accept",
                "Replicate accepted the prediction but it did not complete before the deadline",
                false,
            ));
        }
        let get_url = prediction
            .pointer("/urls/get")
            .and_then(Value::as_str)
            .ok_or_else(|| anyhow!("Replicate pending prediction returned no urls.get"))?;
        if !same_origin(&direct.base_url, get_url)? {
            return Err(provider_error(
                provider,
                "unsafe_polling_url",
                "Replicate returned a cross-origin polling URL; credentials were not sent",
                false,
            ));
        }
        tokio::time::sleep(Duration::from_secs(1)).await;
        let response = client
            .get(get_url)
            .bearer_auth(direct.api_key.trim())
            .send()
            .await
            .map_err(|error| transport_error(provider, "polling Replicate prediction", error))?;
        let status = response.status();
        let raw = read_limited(response, MAX_RESPONSE_BYTES).await?;
        if !status.is_success() {
            return Err(http_error(provider, status, &raw));
        }
        prediction = serde_json::from_slice(&raw).context("decoding Replicate status")?;
    }
    if prediction.get("status").and_then(Value::as_str) != Some("succeeded") {
        return Err(provider_error(
            provider,
            "generation_failed",
            prediction
                .get("error")
                .and_then(Value::as_str)
                .unwrap_or("Replicate prediction did not succeed"),
            false,
        ));
    }
    let output = prediction
        .get("output")
        .ok_or_else(|| anyhow!("Replicate returned no output"))?;
    image_result_from_url(
        client,
        output,
        &request.output_format,
        &request.subject,
        &direct.base_url,
        format!("{provider}:{}", request.model.trim()),
    )
    .await
}
