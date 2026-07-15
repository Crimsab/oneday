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
    let mut payload = json!({
        "prompt": legacy_prompt(request),
        "num_images": 1,
        "output_format": normalize_format(&request.output_format)
    });
    let aspect_ratio = request
        .aspect_ratio
        .clone()
        .or_else(|| aspect_ratio_from_size(&request.size));
    set_optional_string(&mut payload, "aspect_ratio", aspect_ratio.as_deref());
    let endpoint = format!(
        "{}/{}",
        direct.base_url.trim_end_matches('/'),
        request.model.trim_matches('/')
    );
    let response = client
        .post(&endpoint)
        .header("Authorization", format!("Key {}", direct.api_key.trim()))
        .json(&payload)
        .send()
        .await
        .map_err(|error| transport_error(provider, "submitting fal.ai request", error))?;
    let status = response.status();
    let raw = read_limited(response, MAX_RESPONSE_BYTES).await?;
    if !status.is_success() {
        return Err(http_error(provider, status, &raw));
    }
    let initial: Value = serde_json::from_slice(&raw).context("decoding fal.ai response")?;
    let result = if image_url_from_value(&initial).is_some() {
        initial
    } else {
        let request_id = initial
            .get("request_id")
            .and_then(Value::as_str)
            .ok_or_else(|| anyhow!("fal.ai returned neither images nor request_id"))?;
        let status_url = initial
            .get("status_url")
            .and_then(Value::as_str)
            .map(str::to_string)
            .unwrap_or_else(|| format!("{endpoint}/requests/{request_id}/status"));
        let response_url = initial
            .get("response_url")
            .and_then(Value::as_str)
            .map(str::to_string)
            .unwrap_or_else(|| format!("{endpoint}/requests/{request_id}"));
        if !same_origin(&direct.base_url, &status_url)?
            || !same_origin(&direct.base_url, &response_url)?
        {
            return Err(provider_error(
                provider,
                "unsafe_polling_url",
                "fal.ai returned a cross-origin polling URL; credentials were not sent",
                false,
            ));
        }
        wait_for_fal_result(
            client,
            &direct.api_key,
            &status_url,
            &response_url,
            request.timeout_ms,
        )
        .await?
    };
    image_result_from_url(
        client,
        &result,
        &request.output_format,
        &request.subject,
        &direct.base_url,
        format!("{provider}:{}", request.model.trim()),
    )
    .await
}

async fn wait_for_fal_result(
    client: &Client,
    api_key: &str,
    status_url: &str,
    response_url: &str,
    timeout_ms: u64,
) -> anyhow::Result<Value> {
    let deadline = Instant::now() + Duration::from_millis(timeout_ms.max(1));
    loop {
        if Instant::now() >= deadline {
            return Err(provider_error(
                "fal",
                "timeout_after_accept",
                "fal.ai request was accepted but did not complete before the deadline",
                false,
            ));
        }
        let response = client
            .get(status_url)
            .header("Authorization", format!("Key {}", api_key.trim()))
            .send()
            .await
            .map_err(|error| transport_error("fal", "polling fal.ai request", error))?;
        let status = response.status();
        let raw = read_limited(response, MAX_RESPONSE_BYTES).await?;
        if !status.is_success() {
            return Err(http_error("fal", status, &raw));
        }
        let value: Value = serde_json::from_slice(&raw).context("decoding fal.ai status")?;
        match value
            .get("status")
            .and_then(Value::as_str)
            .unwrap_or_default()
        {
            "COMPLETED" => break,
            "FAILED" | "CANCELLED" => {
                return Err(provider_error(
                    "fal",
                    "generation_failed",
                    value
                        .get("error")
                        .and_then(Value::as_str)
                        .unwrap_or("fal.ai generation failed"),
                    false,
                ));
            }
            _ => tokio::time::sleep(Duration::from_secs(1)).await,
        }
    }
    let response = client
        .get(response_url)
        .header("Authorization", format!("Key {}", api_key.trim()))
        .send()
        .await
        .map_err(|error| transport_error("fal", "fetching fal.ai result", error))?;
    let status = response.status();
    let raw = read_limited(response, MAX_RESPONSE_BYTES).await?;
    if !status.is_success() {
        return Err(http_error("fal", status, &raw));
    }
    serde_json::from_slice(&raw).context("decoding fal.ai result")
}
