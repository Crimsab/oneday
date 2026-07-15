use super::{provider_error, GenerateRequest, GeneratedImage, MAX_IMAGE_BYTES};
use anyhow::{anyhow, Context};
use base64::Engine;
use reqwest::{Client, Response};
use serde_json::{json, Value};
use std::net::{IpAddr, Ipv4Addr, Ipv6Addr, SocketAddr};
use std::time::Duration;

pub(super) fn parse_fallback_route(route: &str) -> Option<Value> {
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

pub(super) fn legacy_prompt(request: &GenerateRequest) -> String {
    if request.negative_prompt.trim().is_empty() {
        return request.prompt.clone();
    }
    format!(
        "{}\nAvoid: {}",
        request.prompt.trim(),
        request.negative_prompt.trim()
    )
}

pub(super) fn aspect_ratio_from_size(size: &str) -> Option<String> {
    let (width, height) = size.trim().split_once('x')?;
    let width = width.parse::<u64>().ok()?;
    let height = height.parse::<u64>().ok()?;
    if width == 0 || height == 0 {
        return None;
    }
    let divisor = gcd(width, height);
    Some(format!("{}:{}", width / divisor, height / divisor))
}

fn gcd(mut left: u64, mut right: u64) -> u64 {
    while right != 0 {
        let remainder = left % right;
        left = right;
        right = remainder;
    }
    left.max(1)
}

pub(super) fn format_from_mime(mime: &str) -> Option<String> {
    match mime.trim().to_ascii_lowercase().as_str() {
        "image/png" => Some("png".to_string()),
        "image/jpeg" | "image/jpg" => Some("jpeg".to_string()),
        "image/webp" => Some("webp".to_string()),
        _ => None,
    }
}

pub(super) fn validate_vendor_slug(model: &str, provider: &str) -> anyhow::Result<()> {
    let model = model.trim_matches('/');
    if model.is_empty()
        || model.contains("..")
        || model.contains(['?', '#', '\\'])
        || !model
            .chars()
            .all(|character| character.is_ascii_alphanumeric() || "-_/._".contains(character))
    {
        return Err(provider_error(
            provider,
            "invalid_model",
            "model slug contains unsupported path characters",
            false,
        ));
    }
    Ok(())
}

pub(super) fn image_url_from_value(value: &Value) -> Option<&str> {
    value
        .as_str()
        .or_else(|| value.get("url").and_then(Value::as_str))
        .or_else(|| {
            value
                .get("images")
                .and_then(Value::as_array)
                .and_then(|images| images.first())
                .and_then(image_url_from_value)
        })
        .or_else(|| {
            value
                .as_array()
                .and_then(|images| images.first())
                .and_then(image_url_from_value)
        })
}

pub(super) async fn image_result_from_url(
    client: &Client,
    value: &Value,
    expected_format: &str,
    subject: &str,
    trusted_origin: &str,
    provider_label: String,
) -> anyhow::Result<GeneratedImage> {
    let url = image_url_from_value(value)
        .ok_or_else(|| anyhow!("image provider result contained no image URL"))?;
    let bytes = download_image(client, url, expected_format, subject, Some(trusted_origin)).await?;
    Ok(GeneratedImage {
        bytes,
        output_format: normalize_format(expected_format),
        revised_prompt: value
            .get("prompt")
            .and_then(Value::as_str)
            .unwrap_or_default()
            .to_string(),
        provider_label,
    })
}

pub(super) fn normalize_bridge_model(model: &str) -> String {
    model
        .trim()
        .strip_prefix("openai/")
        .unwrap_or(model.trim())
        .to_string()
}

pub(super) fn normalize_openclaw_model(model: &str) -> String {
    match model.trim() {
        "gpt-image-1" => "openai/gpt-image-1".to_string(),
        "gpt-image-2" => "openai/gpt-image-2".to_string(),
        value => value.to_string(),
    }
}

pub(super) fn normalize_format(format: &str) -> String {
    match format.trim().to_ascii_lowercase().as_str() {
        "jpeg" | "jpg" => "jpeg".to_string(),
        "webp" => "webp".to_string(),
        _ => "png".to_string(),
    }
}

pub(super) fn decode_and_validate(encoded: &str, expected_format: &str) -> anyhow::Result<Vec<u8>> {
    let bytes = base64::engine::general_purpose::STANDARD
        .decode(encoded)
        .context("decoding generated image base64")?;
    validate_image(&bytes, expected_format)?;
    Ok(bytes)
}

pub(super) fn validate_image(bytes: &[u8], expected_format: &str) -> anyhow::Result<()> {
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

pub(super) fn detect_format(bytes: &[u8]) -> Option<String> {
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

pub(super) async fn download_image(
    _client: &Client,
    url: &str,
    expected_format: &str,
    subject: &str,
    trusted_origin: Option<&str>,
) -> anyhow::Result<Vec<u8>> {
    let mut current = reqwest::Url::parse(url).context("parsing generated image URL")?;
    let allow_test_local = cfg!(test) && current.fragment() == Some("oneday-test-local");
    current.set_fragment(None);
    for redirect_count in 0..=3 {
        // Local destinations are permitted only for the first hop of explicit
        // contract-test URLs. A redirect must pass the production policy.
        let trusted_non_public =
            trusted_origin.is_some_and(|base| same_origin(base, current.as_str()).unwrap_or(false));
        let addresses = validate_public_destination(
            &current,
            trusted_non_public || (allow_test_local && redirect_count == 0),
        )
        .await?;
        let host = current
            .host_str()
            .ok_or_else(|| anyhow!("generated image URL has no host"))?;
        let client = Client::builder()
            .timeout(Duration::from_secs(60))
            .redirect(reqwest::redirect::Policy::none())
            .resolve(host, addresses[0])
            .build()
            .context("building safe image download client")?;
        let response = client
            .get(current.clone())
            .send()
            .await
            .with_context(|| format!("downloading generated image for {subject}"))?;
        if response.status().is_redirection() {
            if redirect_count == 3 {
                return Err(anyhow!("generated image URL exceeded redirect limit"));
            }
            let location = response
                .headers()
                .get(reqwest::header::LOCATION)
                .and_then(|value| value.to_str().ok())
                .ok_or_else(|| anyhow!("generated image redirect omitted Location"))?;
            current = current
                .join(location)
                .context("parsing generated image redirect")?;
            continue;
        }
        let response = response
            .error_for_status()
            .context("generated image download failed")?;
        let bytes = read_limited(response, MAX_IMAGE_BYTES).await?;
        validate_image(&bytes, expected_format)?;
        return Ok(bytes);
    }
    Err(anyhow!("generated image download failed"))
}

pub(super) fn same_origin(base: &str, candidate: &str) -> anyhow::Result<bool> {
    let base = reqwest::Url::parse(base).context("parsing configured provider base URL")?;
    let candidate = reqwest::Url::parse(candidate).context("parsing provider polling URL")?;
    Ok(base.scheme() == candidate.scheme()
        && base.host_str() == candidate.host_str()
        && base.port_or_known_default() == candidate.port_or_known_default())
}

pub(super) async fn validate_public_destination(
    url: &reqwest::Url,
    allow_test_local: bool,
) -> anyhow::Result<Vec<SocketAddr>> {
    if !matches!(url.scheme(), "http" | "https") {
        return Err(anyhow!("generated image URL must use HTTP or HTTPS"));
    }
    let host = url
        .host_str()
        .ok_or_else(|| anyhow!("generated image URL has no host"))?;
    if !allow_test_local
        && (host.eq_ignore_ascii_case("localhost")
            || host.to_ascii_lowercase().ends_with(".localhost")
            || host.eq_ignore_ascii_case("metadata.google.internal"))
    {
        return Err(anyhow!("generated image URL targets a non-public host"));
    }
    let port = url
        .port_or_known_default()
        .ok_or_else(|| anyhow!("generated image URL has no effective port"))?;
    let addresses = tokio::net::lookup_host((host, port))
        .await
        .context("resolving generated image host")?
        .collect::<Vec<_>>();
    if addresses.is_empty()
        || (!allow_test_local && addresses.iter().any(|address| !is_public_ip(address.ip())))
    {
        return Err(anyhow!(
            "generated image URL resolves to a non-public address"
        ));
    }
    Ok(addresses)
}

fn is_public_ip(ip: IpAddr) -> bool {
    match ip {
        IpAddr::V4(ip) => is_public_ipv4(ip),
        IpAddr::V6(ip) => is_public_ipv6(ip),
    }
}

fn is_public_ipv4(ip: Ipv4Addr) -> bool {
    let [a, b, c, _] = ip.octets();
    let explicitly_non_public = (a == 100 && (64..=127).contains(&b)) // shared address space (CGNAT)
            || (a == 192 && b == 0 && (c == 0 || c == 2)) // IETF assignments / TEST-NET-1
            || (a == 198 && (b == 18 || b == 19)) // benchmark networks
            || (a == 198 && b == 51 && c == 100) // TEST-NET-2
            || (a == 203 && b == 0 && c == 113) // TEST-NET-3
            || a >= 240; // reserved and limited broadcast
    !(explicitly_non_public
        || ip.is_private()
        || ip.is_loopback()
        || ip.is_link_local()
        || ip.is_unspecified()
        || ip.is_multicast()
        || ip.is_broadcast()
        || ip.octets()[0] == 0)
}

fn is_public_ipv6(ip: Ipv6Addr) -> bool {
    if let Some(mapped) = ip.to_ipv4_mapped() {
        return is_public_ipv4(mapped);
    }
    !(ip.is_loopback()
        || ip.is_unspecified()
        || ip.is_multicast()
        || (ip.segments()[0] & 0xfe00) == 0xfc00
        || (ip.segments()[0] & 0xffc0) == 0xfe80)
}

pub(super) async fn read_limited(mut response: Response, limit: usize) -> anyhow::Result<Vec<u8>> {
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

pub(super) fn compact_detail(raw: &[u8]) -> String {
    let value = String::from_utf8_lossy(raw);
    let one_line = value.split_whitespace().collect::<Vec<_>>().join(" ");
    one_line.chars().take(500).collect()
}

pub(super) fn clean_or(value: &str, fallback: &str) -> String {
    let value = value.trim();
    if value.is_empty() {
        fallback.to_string()
    } else {
        value.to_string()
    }
}

pub(super) fn set_optional_string(target: &mut Value, key: &str, value: Option<&str>) {
    if let Some(value) = value.map(str::trim).filter(|value| !value.is_empty()) {
        target[key] = Value::String(value.to_string());
    }
}
