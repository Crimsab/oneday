use crate::{error::PublicError, gateway_protocol as protocol};
use anyhow::{anyhow, Context};
use axum::body::Body;
use axum::extract::{Extension, Json, Query, State};
use axum::http::{header, HeaderMap, HeaderValue, Method, Request, StatusCode, Uri};
use axum::middleware::Next;
use axum::response::{IntoResponse, Response};
use serde::Deserialize;
use serde_json::json;
use sha2::{Digest, Sha256};
use std::collections::HashSet;
use std::net::{IpAddr, SocketAddr};
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};
use uuid::Uuid;

const SESSION_COOKIE: &str = "oneday_session";
const SECURE_SESSION_COOKIE: &str = "__Host-oneday_session";
const SESSION_TTL: Duration = Duration::from_secs(12 * 60 * 60);
const MIN_CREDENTIAL_BYTES: usize = 32;
const CONTENT_SECURITY_POLICY: &str = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; connect-src 'self' https: http:; img-src 'self' data: blob:; media-src 'self' blob:; font-src 'self' data: blob:; worker-src 'self' blob:; manifest-src 'self'; object-src 'none'; base-uri 'none'; frame-src 'none'; frame-ancestors 'none'; form-action 'self'";
const PERMISSIONS_POLICY: &str = "camera=(), microphone=(), geolocation=(), payment=()";

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
enum CredentialKind {
    Bearer,
    Cookie,
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
enum Access {
    Public,
    Authenticated(CredentialKind),
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
enum Rejection {
    Unauthorized,
    Forbidden,
}

#[derive(Debug)]
struct Credentials {
    bootstrap_hash: Option<[u8; 32]>,
    browser_session_hash: Option<[u8; 32]>,
    browser_session_expires_at: Option<Instant>,
    direct_bearer_hash: Option<[u8; 32]>,
}

#[derive(Debug)]
pub struct AuthState {
    allowed_hosts: HashSet<String>,
    credentials: Mutex<Credentials>,
}

pub struct Startup {
    pub state: Arc<AuthState>,
    pub interactive_bootstrap_url: Option<String>,
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
enum BootstrapProvision {
    Configured,
    GenerateForInteractiveTerminal,
}

impl AuthState {
    pub fn initialize(
        addr: SocketAddr,
        configured_hosts: &[String],
        stderr_is_terminal: bool,
    ) -> anyhow::Result<Startup> {
        let mut allowed_hosts = listener_hosts(addr);
        for raw in configured_hosts {
            allowed_hosts.insert(validate_allowed_host(raw)?);
        }

        let configured_bootstrap = environment_secret("ONEDAY_GATEWAY_BOOTSTRAP_TOKEN");
        let direct_bearer = environment_secret("ONEDAY_GATEWAY_AUTH_TOKEN");
        let provision = bootstrap_provision(
            configured_bootstrap.as_deref(),
            stderr_is_terminal,
            addr.ip().is_loopback(),
        )?;
        let bootstrap_token = match provision {
            BootstrapProvision::Configured => configured_bootstrap
                .as_deref()
                .expect("configured bootstrap provision has a token")
                .to_string(),
            BootstrapProvision::GenerateForInteractiveTerminal => random_token(),
        };
        validate_credential("ONEDAY_GATEWAY_BOOTSTRAP_TOKEN", &bootstrap_token)?;
        if let Some(token) = direct_bearer.as_deref() {
            validate_credential("ONEDAY_GATEWAY_AUTH_TOKEN", token)?;
            validate_separate_credentials(&bootstrap_token, token)?;
        }

        let interactive_bootstrap_url = (provision
            == BootstrapProvision::GenerateForInteractiveTerminal)
            .then(|| bootstrap_url(addr, &bootstrap_token));

        Ok(Startup {
            state: Arc::new(Self::new(
                &bootstrap_token,
                direct_bearer.as_deref(),
                allowed_hosts,
            )),
            interactive_bootstrap_url,
        })
    }

    fn new(
        bootstrap_token: &str,
        direct_bearer: Option<&str>,
        allowed_hosts: HashSet<String>,
    ) -> Self {
        Self {
            allowed_hosts,
            credentials: Mutex::new(Credentials {
                bootstrap_hash: Some(token_hash(bootstrap_token)),
                browser_session_hash: None,
                browser_session_expires_at: None,
                direct_bearer_hash: direct_bearer.map(token_hash),
            }),
        }
    }

    fn exchange_bootstrap(&self, presented: &str) -> Result<String, Rejection> {
        let mut credentials = self.credentials.lock().map_err(|_| Rejection::Forbidden)?;
        let Some(expected) = credentials.bootstrap_hash else {
            return Err(Rejection::Unauthorized);
        };
        if !constant_time_eq(&expected, &token_hash(presented)) {
            return Err(Rejection::Unauthorized);
        }

        let session_token = random_token();
        credentials.bootstrap_hash = None;
        credentials.browser_session_hash = Some(token_hash(&session_token));
        credentials.browser_session_expires_at = Some(Instant::now() + SESSION_TTL);
        Ok(session_token)
    }

    fn authenticate_headers(&self, headers: &HeaderMap) -> Option<CredentialKind> {
        if let Some(value) = headers.get(header::AUTHORIZATION) {
            let value = value.to_str().ok()?;
            let (scheme, token) = value.split_once(' ')?;
            if !scheme.eq_ignore_ascii_case("bearer") || token.trim().is_empty() {
                return None;
            }
            return self
                .bearer_matches(token.trim())
                .then_some(CredentialKind::Bearer);
        }

        let token = cookie_value(headers, SECURE_SESSION_COOKIE)
            .or_else(|| cookie_value(headers, SESSION_COOKIE))?;
        self.browser_session_matches(token)
            .then_some(CredentialKind::Cookie)
    }

    fn bearer_matches(&self, presented: &str) -> bool {
        let presented_hash = token_hash(presented);
        let direct_matches = self
            .credentials
            .lock()
            .ok()
            .and_then(|credentials| credentials.direct_bearer_hash)
            .is_some_and(|expected| constant_time_eq(&expected, &presented_hash));
        let browser_matches = self.browser_session_matches_hash(&presented_hash);
        direct_matches | browser_matches
    }

    fn browser_session_matches(&self, presented: &str) -> bool {
        self.browser_session_matches_hash(&token_hash(presented))
    }

    fn browser_session_matches_hash(&self, presented_hash: &[u8; 32]) -> bool {
        let Ok(mut credentials) = self.credentials.lock() else {
            return false;
        };
        if credentials
            .browser_session_expires_at
            .is_some_and(|expires_at| Instant::now() >= expires_at)
        {
            credentials.browser_session_hash = None;
            credentials.browser_session_expires_at = None;
            return false;
        }
        credentials
            .browser_session_hash
            .as_ref()
            .is_some_and(|expected| constant_time_eq(expected, presented_hash))
    }

    fn authorize_request(&self, request: &Request<Body>) -> Result<Access, Rejection> {
        let host = validated_request_host(request.headers(), &self.allowed_hosts)?;
        if is_public_request(request.method(), request.uri().path()) {
            return Ok(Access::Public);
        }

        let credential = self
            .authenticate_headers(request.headers())
            .ok_or(Rejection::Unauthorized)?;
        if let Some(origin) = single_header(request.headers(), &header::ORIGIN)? {
            validate_origin(origin, host)?;
        } else if credential == CredentialKind::Cookie && is_mutation(request.method()) {
            return Err(Rejection::Forbidden);
        }

        if let Some(fetch_site) = single_header(
            request.headers(),
            HeaderValueName::SecFetchSite.header_name(),
        )? {
            if !matches!(fetch_site, "same-origin" | "none") {
                return Err(Rejection::Forbidden);
            }
        }
        Ok(Access::Authenticated(credential))
    }

    pub fn is_authenticated(&self, headers: &HeaderMap) -> bool {
        self.authenticate_headers(headers).is_some()
    }
}

#[derive(Clone, Copy)]
enum HeaderValueName {
    SecFetchSite,
}

impl HeaderValueName {
    fn header_name(self) -> &'static header::HeaderName {
        match self {
            Self::SecFetchSite => &SEC_FETCH_SITE,
        }
    }
}

static SEC_FETCH_SITE: header::HeaderName = header::HeaderName::from_static("sec-fetch-site");

pub async fn enforce(
    State(state): State<Arc<AuthState>>,
    request: Request<Body>,
    next: Next,
) -> Response {
    let request_path = request.uri().path().to_string();
    let access = match state.authorize_request(&request) {
        Ok(access) => access,
        Err(rejection) => return rejection_response(rejection),
    };

    let mut response = next.run(request).await;
    apply_security_headers(&mut response, &request_path, access);
    response
}

#[derive(Debug, Deserialize)]
pub struct BootstrapQuery {
    token: String,
}

#[derive(Debug, Deserialize)]
pub struct BootstrapRequest {
    token: String,
}

pub async fn bootstrap_get(
    Extension(state): Extension<Arc<AuthState>>,
    headers: HeaderMap,
    Query(query): Query<BootstrapQuery>,
) -> Response {
    match state.exchange_bootstrap(&query.token) {
        Ok(session_token) => bootstrap_response(&headers, session_token, true),
        Err(rejection) => rejection_response(rejection),
    }
}

pub async fn bootstrap_post(
    Extension(state): Extension<Arc<AuthState>>,
    headers: HeaderMap,
    Json(payload): Json<BootstrapRequest>,
) -> Response {
    match state.exchange_bootstrap(&payload.token) {
        Ok(session_token) => bootstrap_response(&headers, session_token, false),
        Err(rejection) => rejection_response(rejection),
    }
}

fn bootstrap_response(headers: &HeaderMap, session_token: String, redirect: bool) -> Response {
    let secure = headers
        .get(header::HOST)
        .and_then(|value| value.to_str().ok())
        .is_some_and(|host| !is_loopback_authority(host));
    let mut response = if redirect {
        (StatusCode::SEE_OTHER, [(header::LOCATION, "/")]).into_response()
    } else {
        (
            StatusCode::OK,
            Json(json!({
                "session_token": session_token,
                "token_type": "Bearer",
                "expires_in": SESSION_TTL.as_secs(),
            })),
        )
            .into_response()
    };
    let cookie_name = if secure {
        SECURE_SESSION_COOKIE
    } else {
        SESSION_COOKIE
    };
    let cookie = format!(
        "{cookie_name}={session_token}; HttpOnly; SameSite=Strict; Path=/; Max-Age={}{}",
        SESSION_TTL.as_secs(),
        if secure { "; Secure" } else { "" },
    );
    if let Ok(cookie) = HeaderValue::from_str(&cookie) {
        response.headers_mut().insert(header::SET_COOKIE, cookie);
    }
    apply_common_security_headers(response.headers_mut());
    apply_document_security_headers(response.headers_mut());
    response
        .headers_mut()
        .insert(header::CACHE_CONTROL, HeaderValue::from_static("no-store"));
    response
}

fn rejection_response(rejection: Rejection) -> Response {
    let (status, code, message) = match rejection {
        Rejection::Unauthorized => (
            StatusCode::UNAUTHORIZED,
            "authentication_required",
            "Authentication is required.",
        ),
        Rejection::Forbidden => (
            StatusCode::FORBIDDEN,
            "request_forbidden",
            "The request origin or host is not allowed.",
        ),
    };
    let mut response = (status, Json(json!({ "error": message, "code": code }))).into_response();
    apply_common_security_headers(response.headers_mut());
    apply_document_security_headers(response.headers_mut());
    response
        .headers_mut()
        .insert(header::CACHE_CONTROL, HeaderValue::from_static("no-store"));
    if rejection == Rejection::Unauthorized {
        response.headers_mut().insert(
            header::WWW_AUTHENTICATE,
            HeaderValue::from_static("Bearer realm=\"oneday\""),
        );
    }
    response
}

fn apply_security_headers(response: &mut Response, path: &str, access: Access) {
    let headers = response.headers_mut();
    apply_common_security_headers(headers);
    if path == "/api/auth/bootstrap" || is_static_spa_path(path) {
        apply_document_security_headers(headers);
    }
    if path.starts_with("/api/")
        || path.starts_with("/generated/")
        || matches!(access, Access::Authenticated(_))
    {
        headers.insert(header::CACHE_CONTROL, HeaderValue::from_static("no-store"));
    }
}

fn apply_common_security_headers(headers: &mut HeaderMap) {
    headers.insert(
        header::X_CONTENT_TYPE_OPTIONS,
        HeaderValue::from_static("nosniff"),
    );
    headers.insert(
        header::REFERRER_POLICY,
        HeaderValue::from_static("no-referrer"),
    );
    headers.insert(
        header::HeaderName::from_static("x-frame-options"),
        HeaderValue::from_static("DENY"),
    );
    headers.insert(
        header::HeaderName::from_static("permissions-policy"),
        HeaderValue::from_static(PERMISSIONS_POLICY),
    );
}

fn apply_document_security_headers(headers: &mut HeaderMap) {
    headers.insert(
        header::HeaderName::from_static("content-security-policy"),
        HeaderValue::from_static(CONTENT_SECURITY_POLICY),
    );
}

fn is_public_request(method: &Method, path: &str) -> bool {
    if method == Method::GET && matches!(path, "/api/health" | "/api/auth/bootstrap") {
        return true;
    }
    if method == Method::POST && path == "/api/auth/bootstrap" {
        return true;
    }
    is_static_spa_path(path)
}

fn is_static_spa_path(path: &str) -> bool {
    path != "/api"
        && path != "/generated"
        && !path.starts_with("/api/")
        && !path.starts_with("/generated/")
}

fn is_mutation(method: &Method) -> bool {
    !matches!(*method, Method::GET | Method::HEAD | Method::OPTIONS)
}

fn listener_hosts(addr: SocketAddr) -> HashSet<String> {
    let mut hosts = HashSet::new();
    let port = addr.port();
    if addr.ip().is_loopback() || addr.ip().is_unspecified() {
        hosts.insert(format!("localhost:{port}"));
        hosts.insert(format!("127.0.0.1:{port}"));
        hosts.insert(format!("[::1]:{port}"));
    }
    if !addr.ip().is_unspecified() {
        hosts.insert(socket_authority(addr));
    }
    hosts
}

fn socket_authority(addr: SocketAddr) -> String {
    match addr.ip() {
        IpAddr::V4(ip) => format!("{ip}:{}", addr.port()),
        IpAddr::V6(ip) => format!("[{ip}]:{}", addr.port()),
    }
}

fn validate_allowed_host(raw: &str) -> anyhow::Result<String> {
    let host = raw.trim().to_ascii_lowercase();
    if host.is_empty()
        || host.chars().any(char::is_whitespace)
        || host.contains(['/', '\\', '@', '?', '#'])
    {
        return Err(anyhow!(
            "ONEDAY_GATEWAY_ALLOWED_HOSTS contains an invalid entry"
        ));
    }
    let probe = format!("http://{host}/");
    let parsed =
        reqwest::Url::parse(&probe).context("parsing an ONEDAY_GATEWAY_ALLOWED_HOSTS entry")?;
    if parsed.host_str().is_none() || parsed.path() != "/" {
        return Err(anyhow!(
            "ONEDAY_GATEWAY_ALLOWED_HOSTS contains an invalid entry"
        ));
    }
    Ok(host)
}

fn validated_request_host<'a>(
    headers: &'a HeaderMap,
    allowed_hosts: &HashSet<String>,
) -> Result<&'a str, Rejection> {
    let host = single_header(headers, &header::HOST)?.ok_or(Rejection::Forbidden)?;
    let normalized = host.to_ascii_lowercase();
    if allowed_hosts.contains(&normalized) {
        Ok(host)
    } else {
        Err(Rejection::Forbidden)
    }
}

fn single_header<'a>(
    headers: &'a HeaderMap,
    name: &header::HeaderName,
) -> Result<Option<&'a str>, Rejection> {
    let mut values = headers.get_all(name).iter();
    let Some(value) = values.next() else {
        return Ok(None);
    };
    if values.next().is_some() {
        return Err(Rejection::Forbidden);
    }
    value.to_str().map(Some).map_err(|_| Rejection::Forbidden)
}

fn validate_origin(origin: &str, request_host: &str) -> Result<(), Rejection> {
    if origin == "null" {
        return Err(Rejection::Forbidden);
    }
    let parsed = reqwest::Url::parse(origin).map_err(|_| Rejection::Forbidden)?;
    if !parsed.username().is_empty()
        || parsed.password().is_some()
        || parsed.query().is_some()
        || parsed.fragment().is_some()
        || parsed.path() != "/"
    {
        return Err(Rejection::Forbidden);
    }
    if parsed.scheme() != "https"
        && !(parsed.scheme() == "http" && is_loopback_authority(request_host))
    {
        return Err(Rejection::Forbidden);
    }

    let origin_host = parsed.host_str().ok_or(Rejection::Forbidden)?;
    let request_url = reqwest::Url::parse(&format!("{}://{request_host}/", parsed.scheme()))
        .map_err(|_| Rejection::Forbidden)?;
    if request_url.host_str() != Some(origin_host)
        || request_url.port_or_known_default() != parsed.port_or_known_default()
    {
        return Err(Rejection::Forbidden);
    }
    Ok(())
}

fn is_loopback_authority(authority: &str) -> bool {
    let Ok(parsed) = reqwest::Url::parse(&format!("http://{authority}/")) else {
        return false;
    };
    let Some(host) = parsed.host_str() else {
        return false;
    };
    host.eq_ignore_ascii_case("localhost")
        || host
            .parse::<IpAddr>()
            .is_ok_and(|address| address.is_loopback())
}

fn cookie_value<'a>(headers: &'a HeaderMap, name: &str) -> Option<&'a str> {
    headers
        .get_all(header::COOKIE)
        .iter()
        .filter_map(|value| value.to_str().ok())
        .flat_map(|value| value.split(';'))
        .filter_map(|part| part.trim().split_once('='))
        .find_map(|(cookie_name, value)| (cookie_name == name).then_some(value))
}

fn environment_secret(name: &str) -> Option<String> {
    std::env::var(name)
        .ok()
        .map(|token| token.trim().to_string())
        .filter(|token| !token.is_empty())
}

fn bootstrap_provision(
    configured_token: Option<&str>,
    stderr_is_terminal: bool,
    listener_is_loopback: bool,
) -> anyhow::Result<BootstrapProvision> {
    if configured_token.is_some() {
        return Ok(BootstrapProvision::Configured);
    }
    if stderr_is_terminal && listener_is_loopback {
        return Ok(BootstrapProvision::GenerateForInteractiveTerminal);
    }
    Err(anyhow!(
        "ONEDAY_GATEWAY_BOOTSTRAP_TOKEN is required unless the gateway is started on loopback from an interactive terminal"
    ))
}

fn validate_credential(name: &str, token: &str) -> anyhow::Result<()> {
    if token.len() < MIN_CREDENTIAL_BYTES {
        return Err(anyhow!(
            "{name} must contain at least {MIN_CREDENTIAL_BYTES} bytes"
        ));
    }
    Ok(())
}

fn validate_separate_credentials(bootstrap: &str, direct_bearer: &str) -> anyhow::Result<()> {
    if constant_time_eq(&token_hash(bootstrap), &token_hash(direct_bearer)) {
        return Err(anyhow!(
            "ONEDAY_GATEWAY_AUTH_TOKEN and ONEDAY_GATEWAY_BOOTSTRAP_TOKEN must be distinct"
        ));
    }
    Ok(())
}

fn random_token() -> String {
    format!("{}{}", Uuid::new_v4().simple(), Uuid::new_v4().simple())
}

fn token_hash(token: &str) -> [u8; 32] {
    Sha256::digest(token.as_bytes()).into()
}

fn constant_time_eq(left: &[u8; 32], right: &[u8; 32]) -> bool {
    left.iter()
        .zip(right.iter())
        .fold(0_u8, |difference, (left, right)| {
            difference | (left ^ right)
        })
        == 0
}

pub fn bootstrap_url(addr: SocketAddr, token: &str) -> String {
    let host = if addr.ip().is_unspecified() {
        match addr.ip() {
            IpAddr::V4(_) => "127.0.0.1".to_string(),
            IpAddr::V6(_) => "[::1]".to_string(),
        }
    } else {
        match addr.ip() {
            IpAddr::V4(ip) => ip.to_string(),
            IpAddr::V6(ip) => format!("[{ip}]"),
        }
    };
    format!(
        "http://{host}:{}/api/auth/bootstrap?token={token}",
        addr.port()
    )
}

pub fn redacted_request_target(uri: &Uri) -> String {
    if uri.query().is_some() {
        format!("{}?<redacted>", uri.path())
    } else {
        uri.path().to_string()
    }
}

pub fn redact_model_settings(settings: &mut protocol::ModelRoutingSettings) {
    settings.image_generation.base_url = redact_url(&settings.image_generation.base_url);
    settings.image_generation.openclaw_bridge_url =
        redact_url(&settings.image_generation.openclaw_bridge_url);
    settings.image_generation.imagegen_bridge_url =
        redact_url(&settings.image_generation.imagegen_bridge_url);
    for provider in &mut settings.image_providers {
        provider.base_url = redact_url(&provider.base_url);
    }
}

pub fn validate_model_settings_update(
    current: &protocol::ModelRoutingSettings,
    update: &mut protocol::ModelRoutingUpdate,
) -> Result<(), PublicError> {
    let Some(image_update) = update.image_generation.as_mut() else {
        return Ok(());
    };

    suppress_redacted_echo(
        &current.image_generation.base_url,
        &mut image_update.base_url,
    );
    suppress_redacted_echo(
        &current.image_generation.openclaw_bridge_url,
        &mut image_update.openclaw_bridge_url,
    );
    suppress_redacted_echo(
        &current.image_generation.imagegen_bridge_url,
        &mut image_update.imagegen_bridge_url,
    );

    if endpoint_change_reuses_secret(
        &current.image_generation.base_url,
        current.image_generation.api_key_configured
            || environment_credential_is_set(&[
                "ONEDAY_IMAGEGEN_API_KEY",
                "ONEDAY_IMAGE_API_KEY",
                "ONEDAY_LITELLM_API_KEY",
            ]),
        image_update.base_url.as_deref(),
        false,
        false,
    ) {
        return Err(credential_reentry_error());
    }
    if endpoint_change_reuses_secret(
        &current.image_generation.imagegen_bridge_url,
        current.image_generation.imagegen_bridge_token_configured
            || environment_credential_is_set(&["ONEDAY_IMAGEGEN_BRIDGE_TOKEN"]),
        image_update.imagegen_bridge_url.as_deref(),
        image_update.imagegen_bridge_token.is_some(),
        image_update.clear_imagegen_bridge_token.unwrap_or(false),
    ) {
        return Err(credential_reentry_error());
    }

    for provider_update in &mut image_update.provider_configs {
        let Some(provider) = current.image_providers.iter().find(|provider| {
            provider
                .id
                .trim()
                .eq_ignore_ascii_case(provider_update.id.trim())
        }) else {
            continue;
        };
        suppress_redacted_echo(&provider.base_url, &mut provider_update.base_url);
        if endpoint_change_reuses_secret(
            &provider.base_url,
            provider.api_key_configured || provider_environment_credential_is_set(&provider.id),
            provider_update.base_url.as_deref(),
            provider_update.api_key.is_some(),
            provider_update.clear_api_key.unwrap_or(false),
        ) {
            return Err(credential_reentry_error());
        }
    }
    Ok(())
}

fn suppress_redacted_echo(current: &str, proposed: &mut Option<String>) {
    if proposed
        .as_deref()
        .is_some_and(|value| value.trim() == redact_url(current))
    {
        *proposed = None;
    }
}

fn endpoint_change_reuses_secret(
    current_url: &str,
    secret_configured: bool,
    proposed_url: Option<&str>,
    replacement_supplied: bool,
    clear_secret: bool,
) -> bool {
    let Some(proposed_url) = proposed_url else {
        return false;
    };
    secret_configured
        && current_url.trim() != proposed_url.trim()
        && !replacement_supplied
        && !clear_secret
}

fn provider_environment_credential_is_set(provider: &str) -> bool {
    let names: &[&str] = match provider.trim().to_ascii_lowercase().as_str() {
        "openai" => &["ONEDAY_IMAGEGEN_OPENAI_API_KEY", "OPENAI_API_KEY"],
        "openai-compatible" | "litellm" => &["ONEDAY_IMAGEGEN_OPENAI_COMPATIBLE_API_KEY"],
        "gemini" => &[
            "ONEDAY_IMAGEGEN_GEMINI_API_KEY",
            "GEMINI_API_KEY",
            "GOOGLE_API_KEY",
        ],
        "fal" => &["ONEDAY_IMAGEGEN_FAL_API_KEY", "FAL_KEY"],
        "replicate" => &["ONEDAY_IMAGEGEN_REPLICATE_API_TOKEN", "REPLICATE_API_TOKEN"],
        "stability" => &["ONEDAY_IMAGEGEN_STABILITY_API_KEY", "STABILITY_API_KEY"],
        "azure-openai" => &[
            "ONEDAY_IMAGEGEN_AZURE_OPENAI_API_KEY",
            "AZURE_OPENAI_API_KEY",
        ],
        _ => &[],
    };
    environment_credential_is_set(names)
}

fn environment_credential_is_set(names: &[&str]) -> bool {
    names
        .iter()
        .any(|name| std::env::var(name).is_ok_and(|value| !value.trim().is_empty()))
}

fn credential_reentry_error() -> PublicError {
    PublicError::bad_request(
        "credential_reentry_required",
        "Re-enter or clear the provider credential when changing its base URL.",
    )
}

fn redact_url(value: &str) -> String {
    let trimmed = value.trim();
    if trimmed.is_empty() {
        return String::new();
    }
    let Ok(mut parsed) = reqwest::Url::parse(trimmed) else {
        return if trimmed.contains(['@', '?', '#']) {
            "<redacted-url>".to_string()
        } else {
            trimmed.to_string()
        };
    };
    let _ = parsed.set_username("");
    let _ = parsed.set_password(None);
    parsed.set_query(None);
    parsed.set_fragment(None);
    parsed.to_string()
}

#[cfg(test)]
mod tests {
    use super::*;

    fn test_state() -> AuthState {
        AuthState::new(
            "bootstrap-token-with-at-least-thirty-two-characters",
            None,
            HashSet::from(["localhost:8788".to_string()]),
        )
    }

    fn request(method: Method, path: &str) -> Request<Body> {
        Request::builder()
            .method(method)
            .uri(path)
            .header(header::HOST, "localhost:8788")
            .body(Body::empty())
            .unwrap()
    }

    #[test]
    fn unauthenticated_data_is_rejected_while_health_and_static_are_public() {
        let state = test_state();

        assert_eq!(
            state.authorize_request(&request(Method::GET, "/api/stories")),
            Err(Rejection::Unauthorized)
        );
        assert_eq!(
            state.authorize_request(&request(Method::GET, "/api/config/models")),
            Err(Rejection::Unauthorized)
        );
        assert_eq!(
            state.authorize_request(&request(Method::POST, "/api/stories")),
            Err(Rejection::Unauthorized)
        );
        assert_eq!(
            state.authorize_request(&request(Method::GET, "/generated/assets/private.png")),
            Err(Rejection::Unauthorized)
        );
        assert_eq!(
            state.authorize_request(&request(Method::GET, "/api/health")),
            Ok(Access::Public)
        );
        assert_eq!(
            state.authorize_request(&request(Method::GET, "/assets/app.js")),
            Ok(Access::Public)
        );
    }

    #[test]
    fn bootstrap_is_one_shot_and_issues_a_distinct_session() {
        let state = test_state();
        let session = state
            .exchange_bootstrap("bootstrap-token-with-at-least-thirty-two-characters")
            .expect("first exchange succeeds");

        assert_ne!(
            session,
            "bootstrap-token-with-at-least-thirty-two-characters"
        );
        assert_eq!(
            state.exchange_bootstrap("bootstrap-token-with-at-least-thirty-two-characters"),
            Err(Rejection::Unauthorized)
        );
        assert!(state.browser_session_matches(&session));
    }

    #[test]
    fn direct_bearer_is_separate_from_one_shot_bootstrap() {
        let bootstrap = "browser-bootstrap-token-with-thirty-two-plus-characters";
        let direct = "desktop-per-launch-bearer-with-thirty-two-plus-characters";
        let state = AuthState::new(
            bootstrap,
            Some(direct),
            HashSet::from(["localhost:8788".to_string()]),
        );

        let mut native = request(Method::GET, "/api/stories");
        native.headers_mut().insert(
            header::AUTHORIZATION,
            HeaderValue::from_str(&format!("Bearer {direct}")).unwrap(),
        );
        assert_eq!(
            state.authorize_request(&native),
            Ok(Access::Authenticated(CredentialKind::Bearer))
        );
        let mut ambient_cookie = request(Method::GET, "/api/stories");
        ambient_cookie.headers_mut().insert(
            header::COOKIE,
            HeaderValue::from_str(&format!("{SESSION_COOKIE}={direct}")).unwrap(),
        );
        assert_eq!(
            state.authorize_request(&ambient_cookie),
            Err(Rejection::Unauthorized)
        );
        assert_eq!(
            state.exchange_bootstrap(direct),
            Err(Rejection::Unauthorized)
        );

        let browser_session = state
            .exchange_bootstrap(bootstrap)
            .expect("the independent bootstrap token remains valid");
        assert!(state.browser_session_matches(&browser_session));
        assert_eq!(
            state.exchange_bootstrap(bootstrap),
            Err(Rejection::Unauthorized)
        );
        assert!(!format!("{state:?}").contains(direct));
    }

    #[test]
    fn startup_generation_requires_an_interactive_loopback_terminal() {
        assert_eq!(
            bootstrap_provision(None, true, true).unwrap(),
            BootstrapProvision::GenerateForInteractiveTerminal
        );
        assert_eq!(
            bootstrap_provision(Some("operator-supplied-token"), false, false).unwrap(),
            BootstrapProvision::Configured
        );
        assert!(bootstrap_provision(None, false, true).is_err());
        assert!(bootstrap_provision(None, true, false).is_err());
        assert!(validate_credential("ONEDAY_GATEWAY_AUTH_TOKEN", "too-short").is_err());
        assert!(validate_separate_credentials(
            "same-credential-with-thirty-two-plus-characters",
            "same-credential-with-thirty-two-plus-characters"
        )
        .is_err());
    }

    #[test]
    fn host_and_origin_defenses_reject_rebinding_and_cross_site_mutation() {
        let state = test_state();
        let session = state
            .exchange_bootstrap("bootstrap-token-with-at-least-thirty-two-characters")
            .unwrap();

        let mut rebinding = request(Method::GET, "/api/stories");
        rebinding
            .headers_mut()
            .insert(header::HOST, HeaderValue::from_static("attacker.example"));
        rebinding.headers_mut().insert(
            header::AUTHORIZATION,
            HeaderValue::from_str(&format!("Bearer {session}")).unwrap(),
        );
        assert_eq!(
            state.authorize_request(&rebinding),
            Err(Rejection::Forbidden)
        );

        let mut cross_site = request(Method::POST, "/api/stories");
        cross_site.headers_mut().insert(
            header::COOKIE,
            HeaderValue::from_str(&format!("{SESSION_COOKIE}={session}")).unwrap(),
        );
        assert_eq!(
            state.authorize_request(&cross_site),
            Err(Rejection::Forbidden)
        );
        cross_site.headers_mut().insert(
            header::ORIGIN,
            HeaderValue::from_static("https://attacker.example"),
        );
        assert_eq!(
            state.authorize_request(&cross_site),
            Err(Rejection::Forbidden)
        );

        cross_site.headers_mut().insert(
            header::ORIGIN,
            HeaderValue::from_static("http://localhost:8788"),
        );
        assert_eq!(
            state.authorize_request(&cross_site),
            Ok(Access::Authenticated(CredentialKind::Cookie))
        );

        let mut native = request(Method::POST, "/api/stories");
        native.headers_mut().insert(
            header::AUTHORIZATION,
            HeaderValue::from_str(&format!("Bearer {session}")).unwrap(),
        );
        assert_eq!(
            state.authorize_request(&native),
            Ok(Access::Authenticated(CredentialKind::Bearer))
        );
    }

    #[test]
    fn credentials_are_removed_from_urls_and_request_targets() {
        assert_eq!(
            redact_url("https://user:password@provider.example/v1?api_key=secret#token"),
            "https://provider.example/v1"
        );
        let uri: Uri = "/api/auth/bootstrap?token=top-secret".parse().unwrap();
        let redacted = redacted_request_target(&uri);
        assert_eq!(redacted, "/api/auth/bootstrap?<redacted>");
        assert!(!redacted.contains("top-secret"));

        let error = validate_allowed_host("user:password@provider.example")
            .expect_err("credential-bearing Host configuration is invalid")
            .to_string();
        assert!(!error.contains("password"));
    }

    #[test]
    fn endpoint_changes_cannot_reuse_an_existing_credential() {
        assert!(endpoint_change_reuses_secret(
            "https://provider.example/v1",
            true,
            Some("https://attacker.example/collect"),
            false,
            false,
        ));
        assert!(!endpoint_change_reuses_secret(
            "https://provider.example/v1",
            true,
            Some("https://attacker.example/collect"),
            true,
            false,
        ));
        assert!(!endpoint_change_reuses_secret(
            "https://provider.example/v1",
            true,
            Some("https://attacker.example/collect"),
            false,
            true,
        ));
    }

    #[test]
    fn remote_sessions_use_secure_cookies_and_loopback_development_remains_usable() {
        let remote_headers =
            HeaderMap::from_iter([(header::HOST, HeaderValue::from_static("oneday.example.com"))]);
        let remote = bootstrap_response(&remote_headers, "session".into(), false);
        assert!(remote.headers()[header::SET_COOKIE]
            .to_str()
            .unwrap()
            .starts_with("__Host-oneday_session=session;"));
        assert!(remote.headers()[header::SET_COOKIE]
            .to_str()
            .unwrap()
            .contains("; Secure"));

        let local_headers =
            HeaderMap::from_iter([(header::HOST, HeaderValue::from_static("localhost:8788"))]);
        let local = bootstrap_response(&local_headers, "session".into(), true);
        let cookie = local.headers()[header::SET_COOKIE].to_str().unwrap();
        assert!(cookie.contains("HttpOnly; SameSite=Strict"));
        assert!(!cookie.contains("; Secure"));
    }

    #[test]
    fn static_and_bootstrap_responses_receive_a_compatible_csp_baseline() {
        let mut static_response = Response::new(Body::empty());
        apply_security_headers(&mut static_response, "/", Access::Public);
        let static_csp = static_response.headers()["content-security-policy"]
            .to_str()
            .unwrap();
        for directive in [
            "default-src 'self'",
            "script-src 'self'",
            "style-src 'self' 'unsafe-inline'",
            "connect-src 'self' https: http:",
            "img-src 'self' data: blob:",
            "media-src 'self' blob:",
            "object-src 'none'",
            "base-uri 'none'",
            "frame-src 'none'",
            "frame-ancestors 'none'",
        ] {
            assert!(static_csp.contains(directive), "missing {directive}");
        }
        assert_eq!(
            static_response.headers()["permissions-policy"],
            PERMISSIONS_POLICY
        );

        let headers =
            HeaderMap::from_iter([(header::HOST, HeaderValue::from_static("localhost:8788"))]);
        let bootstrap = bootstrap_response(&headers, "session".into(), true);
        assert_eq!(
            bootstrap.headers()["content-security-policy"],
            CONTENT_SECURITY_POLICY
        );
        assert_eq!(
            bootstrap.headers()[header::X_CONTENT_TYPE_OPTIONS],
            "nosniff"
        );
        assert_eq!(bootstrap.headers()[header::REFERRER_POLICY], "no-referrer");
    }
}
