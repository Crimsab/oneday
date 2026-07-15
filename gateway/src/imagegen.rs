use anyhow::{anyhow, Context};
use base64::Engine;
use reqwest::{Client, Response};
use serde::Deserialize;
use serde_json::{json, Value};

const MAX_IMAGE_BYTES: usize = 32 * 1024 * 1024;
const MAX_RESPONSE_BYTES: usize = 48 * 1024 * 1024;

#[derive(Debug, Clone)]
pub(crate) struct AdapterConfig {
    pub provider: String,
    pub base_url: String,
    pub api_key: String,
    pub openclaw_url: String,
    pub bridge_url: String,
    pub bridge_token: String,
    pub bridge_provider: String,
    pub bridge_map_icon_provider: String,
    pub bridge_fallbacks: Vec<String>,
    pub bridge_fallback_policy: String,
    pub bridge_compatibility: String,
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
    ImagegenBridge,
    OpenClaw,
    OpenAiCompatible,
}

pub(crate) fn adapter_kind(provider: &str) -> AdapterKind {
    match provider.trim().to_ascii_lowercase().as_str() {
        "imagegen-bridge" | "imagegen_bridge" | "bridge-native" => AdapterKind::ImagegenBridge,
        "openclaw" | "openclaw-bridge" | "codex-oauth" => AdapterKind::OpenClaw,
        _ => AdapterKind::OpenAiCompatible,
    }
}

pub(crate) fn is_available(config: &AdapterConfig, model: &str) -> bool {
    if config.provider.trim().is_empty() {
        return false;
    }
    match adapter_kind(&config.provider) {
        AdapterKind::ImagegenBridge => !config.bridge_url.trim().is_empty(),
        AdapterKind::OpenClaw => !config.openclaw_url.trim().is_empty() && !model.trim().is_empty(),
        AdapterKind::OpenAiCompatible => {
            !config.base_url.trim().is_empty()
                && !config.api_key.trim().is_empty()
                && !model.trim().is_empty()
        }
    }
}

pub(crate) async fn generate(
    client: &Client,
    config: &AdapterConfig,
    request: &GenerateRequest,
) -> anyhow::Result<GeneratedImage> {
    match adapter_kind(&config.provider) {
        AdapterKind::ImagegenBridge => generate_via_bridge(client, config, request).await,
        AdapterKind::OpenClaw => generate_via_openclaw(client, config, request).await,
        AdapterKind::OpenAiCompatible => {
            generate_via_openai_compatible(client, config, request).await
        }
    }
}

async fn generate_via_bridge(
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
    let response = builder
        .send()
        .await
        .with_context(|| format!("requesting imagegen-bridge image for {}", request.subject))?;
    let status = response.status();
    let raw = read_limited(response, MAX_RESPONSE_BYTES).await?;
    if !status.is_success() {
        return Err(anyhow!(
            "imagegen-bridge returned {}: {}",
            status,
            bridge_error_detail(&raw)
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
        provider_label: format!("{}:{}", response.provider, response.model),
    })
}

async fn generate_via_openai_compatible(
    client: &Client,
    config: &AdapterConfig,
    request: &GenerateRequest,
) -> anyhow::Result<GeneratedImage> {
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
        config.base_url.trim_end_matches('/')
    );
    let response = client
        .post(endpoint)
        .bearer_auth(config.api_key.trim())
        .json(&payload)
        .send()
        .await
        .with_context(|| format!("requesting image for {}", request.subject))?;
    let status = response.status();
    let raw = read_limited(response, MAX_RESPONSE_BYTES).await?;
    if !status.is_success() {
        return Err(anyhow!(
            "image provider returned {}: {}",
            status,
            compact_detail(&raw)
        ));
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
        download_image(client, url, &request.output_format, &request.subject).await?
    } else {
        return Err(anyhow!("image provider returned neither b64_json nor url"));
    };
    Ok(GeneratedImage {
        bytes,
        output_format: normalize_format(&request.output_format),
        revised_prompt: first.revised_prompt.clone().unwrap_or_default(),
        provider_label: format!("{}:{}", config.provider.trim(), request.model.trim()),
    })
}

async fn generate_via_openclaw(
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
        .with_context(|| format!("requesting OpenClaw image for {}", request.subject))?;
    let status = response.status();
    let raw = read_limited(response, MAX_RESPONSE_BYTES).await?;
    if !status.is_success() {
        return Err(anyhow!(
            "OpenClaw image bridge returned {}: {}",
            status,
            compact_detail(&raw)
        ));
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

fn parse_fallback_route(route: &str) -> Option<Value> {
    let route = route.trim();
    if route.is_empty() {
        return None;
    }
    let (provider, model) = route.split_once(':').unwrap_or((route, ""));
    let provider = provider.trim();
    if provider.is_empty() {
        return None;
    }
    let mut value = json!({"provider": provider});
    set_optional_string(&mut value, "model", Some(model.trim()));
    Some(value)
}

fn legacy_prompt(request: &GenerateRequest) -> String {
    if request.negative_prompt.trim().is_empty() {
        return request.prompt.clone();
    }
    format!(
        "{}\nAvoid: {}",
        request.prompt.trim(),
        request.negative_prompt.trim()
    )
}

fn normalize_bridge_model(model: &str) -> String {
    model
        .trim()
        .strip_prefix("openai/")
        .unwrap_or(model.trim())
        .to_string()
}

fn normalize_openclaw_model(model: &str) -> String {
    match model.trim() {
        "gpt-image-1" => "openai/gpt-image-1".to_string(),
        "gpt-image-2" => "openai/gpt-image-2".to_string(),
        value => value.to_string(),
    }
}

fn normalize_format(format: &str) -> String {
    match format.trim().to_ascii_lowercase().as_str() {
        "jpeg" | "jpg" => "jpeg".to_string(),
        "webp" => "webp".to_string(),
        _ => "png".to_string(),
    }
}

fn decode_and_validate(encoded: &str, expected_format: &str) -> anyhow::Result<Vec<u8>> {
    let bytes = base64::engine::general_purpose::STANDARD
        .decode(encoded)
        .context("decoding generated image base64")?;
    validate_image(&bytes, expected_format)?;
    Ok(bytes)
}

fn validate_image(bytes: &[u8], expected_format: &str) -> anyhow::Result<()> {
    if bytes.is_empty() {
        return Err(anyhow!("image provider returned empty image bytes"));
    }
    if bytes.len() > MAX_IMAGE_BYTES {
        return Err(anyhow!(
            "image provider returned an image larger than 32 MiB"
        ));
    }
    let actual = detect_format(bytes)
        .ok_or_else(|| anyhow!("image provider returned invalid image bytes"))?;
    if actual != normalize_format(expected_format) {
        return Err(anyhow!(
            "image provider returned {actual} bytes while {} was requested",
            normalize_format(expected_format)
        ));
    }
    Ok(())
}

fn detect_format(bytes: &[u8]) -> Option<String> {
    if bytes.starts_with(b"\x89PNG\r\n\x1a\n") {
        Some("png".to_string())
    } else if bytes.starts_with(&[0xff, 0xd8, 0xff]) {
        Some("jpeg".to_string())
    } else if bytes.len() >= 12 && &bytes[..4] == b"RIFF" && &bytes[8..12] == b"WEBP" {
        Some("webp".to_string())
    } else {
        None
    }
}

async fn download_image(
    client: &Client,
    url: &str,
    expected_format: &str,
    subject: &str,
) -> anyhow::Result<Vec<u8>> {
    let parsed = reqwest::Url::parse(url).context("parsing generated image URL")?;
    if !matches!(parsed.scheme(), "http" | "https") {
        return Err(anyhow!("generated image URL must use HTTP or HTTPS"));
    }
    let response = client
        .get(parsed)
        .send()
        .await
        .with_context(|| format!("downloading generated image for {subject}"))?
        .error_for_status()
        .context("generated image download failed")?;
    let bytes = read_limited(response, MAX_IMAGE_BYTES).await?;
    validate_image(&bytes, expected_format)?;
    Ok(bytes)
}

async fn read_limited(mut response: Response, limit: usize) -> anyhow::Result<Vec<u8>> {
    if response
        .content_length()
        .is_some_and(|length| length > limit as u64)
    {
        return Err(anyhow!("image provider response exceeded the size limit"));
    }
    let mut bytes = Vec::new();
    while let Some(chunk) = response
        .chunk()
        .await
        .context("reading image provider response")?
    {
        if bytes.len().saturating_add(chunk.len()) > limit {
            return Err(anyhow!("image provider response exceeded the size limit"));
        }
        bytes.extend_from_slice(&chunk);
    }
    Ok(bytes)
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

fn compact_detail(raw: &[u8]) -> String {
    let value = String::from_utf8_lossy(raw);
    let one_line = value.split_whitespace().collect::<Vec<_>>().join(" ");
    one_line.chars().take(500).collect()
}

fn clean_or(value: &str, fallback: &str) -> String {
    let value = value.trim();
    if value.is_empty() {
        fallback.to_string()
    } else {
        value.to_string()
    }
}

fn set_optional_string(target: &mut Value, key: &str, value: Option<&str>) {
    if let Some(value) = value.map(str::trim).filter(|value| !value.is_empty()) {
        target[key] = Value::String(value.to_string());
    }
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
struct BridgeErrorEnvelope {
    error: Option<BridgeErrorBody>,
}

#[derive(Debug, Deserialize)]
struct BridgeErrorBody {
    code: Option<String>,
    message: String,
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

#[derive(Debug, Deserialize)]
struct OpenClawResponse {
    ok: bool,
    image_b64: Option<String>,
    revised_prompt: Option<String>,
    error: Option<String>,
}

#[cfg(test)]
mod tests {
    use super::*;
    use axum::{
        extract::Json,
        http::{header, HeaderMap},
        routing::post,
        Router,
    };
    use tokio::sync::oneshot;

    const PNG: &[u8] = b"\x89PNG\r\n\x1a\nfixture";

    #[test]
    fn detects_supported_image_formats() {
        assert_eq!(detect_format(PNG).as_deref(), Some("png"));
        assert_eq!(
            detect_format(&[0xff, 0xd8, 0xff, 0x00]).as_deref(),
            Some("jpeg")
        );
        assert_eq!(detect_format(b"RIFF0000WEBP").as_deref(), Some("webp"));
        assert_eq!(detect_format(b"not-an-image"), None);
    }

    #[test]
    fn bridge_model_drops_only_openai_namespace() {
        assert_eq!(normalize_bridge_model("openai/gpt-image-2"), "gpt-image-2");
        assert_eq!(normalize_bridge_model("custom/model"), "custom/model");
    }

    #[test]
    fn fallback_route_supports_provider_and_optional_model() {
        assert_eq!(
            parse_fallback_route("codex-responses:gpt-image-2").unwrap(),
            json!({"provider": "codex-responses", "model": "gpt-image-2"})
        );
        assert_eq!(
            parse_fallback_route("codex-app-server").unwrap(),
            json!({"provider": "codex-app-server"})
        );
    }

    #[test]
    fn validates_format_and_size() {
        assert!(validate_image(PNG, "png").is_ok());
        assert!(validate_image(PNG, "webp").is_err());
        assert!(validate_image(&[], "png").is_err());
    }

    #[tokio::test]
    async fn native_bridge_request_preserves_routing_policies_and_negative_prompt() {
        let (capture_tx, capture_rx) = oneshot::channel::<(HeaderMap, Value)>();
        let capture_tx = std::sync::Arc::new(std::sync::Mutex::new(Some(capture_tx)));
        let app = Router::new().route(
            "/v1/images",
            post({
                let capture_tx = capture_tx.clone();
                move |headers: HeaderMap, Json(payload): Json<Value>| {
                    let capture_tx = capture_tx.clone();
                    async move {
                        if let Some(tx) = capture_tx.lock().unwrap().take() {
                            let _ = tx.send((headers, payload));
                        }
                        Json(json!({
                            "id": "img-test",
                            "created": 1,
                            "provider": "codex-responses",
                            "model": "gpt-image-2",
                            "requested": {},
                            "effective": {},
                            "data": [{
                                "type": "b64_json",
                                "index": 0,
                                "b64_json": base64::engine::general_purpose::STANDARD.encode(PNG),
                                "format": "png",
                                "width": 1,
                                "height": 1,
                                "bytes": PNG.len(),
                                "sha256": "fixture"
                            }],
                            "revised_prompt": "revised",
                            "timings": {}
                        }))
                    }
                }
            }),
        );
        let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
        let address = listener.local_addr().unwrap();
        tokio::spawn(async move { axum::serve(listener, app).await.unwrap() });

        let config = AdapterConfig {
            provider: "imagegen-bridge".into(),
            bridge_url: format!("http://{address}"),
            bridge_token: "test-token".into(),
            bridge_provider: "codex-app-server".into(),
            bridge_map_icon_provider: "codex-responses".into(),
            bridge_fallbacks: vec!["codex-app-server:gpt-image-2".into()],
            bridge_fallback_policy: "on_error".into(),
            bridge_compatibility: "normalize".into(),
            base_url: String::new(),
            api_key: String::new(),
            openclaw_url: String::new(),
        };
        let request = GenerateRequest {
            subject: "map icon".into(),
            prompt: "A clean landmark".into(),
            negative_prompt: "text".into(),
            model: "openai/gpt-image-2".into(),
            is_map_icon: true,
            size: "1024x1024".into(),
            resolution: Some("1k".into()),
            aspect_ratio: Some("1:1".into()),
            quality: "medium".into(),
            output_format: "png".into(),
            background: "transparent".into(),
            timeout_ms: 30_000,
            idempotency_key: "oneday-test".into(),
        };
        let generated = generate(&Client::new(), &config, &request).await.unwrap();
        let (headers, payload) = capture_rx.await.unwrap();

        assert_eq!(
            headers.get(header::AUTHORIZATION).unwrap(),
            "Bearer test-token"
        );
        assert_eq!(payload["negative_prompt"], "text");
        assert_eq!(payload["routing"]["provider"], "codex-responses");
        assert_eq!(payload["routing"]["model"], "gpt-image-2");
        assert_eq!(payload["routing"]["fallback_policy"], "on_error");
        assert_eq!(payload["parameters"]["action"], "auto");
        assert_eq!(
            payload["routing"]["fallbacks"][0],
            json!({"provider": "codex-app-server", "model": "gpt-image-2"})
        );
        assert_eq!(payload["session"]["mode"], "isolated");
        assert_eq!(payload["output"]["response_format"], "b64_json");
        assert_eq!(generated.provider_label, "codex-responses:gpt-image-2");
        assert_eq!(generated.revised_prompt, "revised");
        assert_eq!(generated.bytes, PNG);
    }
}
