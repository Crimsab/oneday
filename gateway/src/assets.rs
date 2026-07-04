use crate::{db, AppState};
use anyhow::{anyhow, Context};
use base64::Engine;
use reqwest::Client;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use sha2::{Digest, Sha256};
use sqlx::{Row, SqlitePool};
use std::collections::HashSet;
use std::path::Path;
use std::sync::Arc;
use std::time::Duration;
use tokio::fs;

#[derive(Debug, Serialize)]
pub struct VisualAssetsResponse {
    pub profile: VisualProfile,
    pub assets: Vec<VisualAsset>,
}

#[derive(Debug, Clone, Serialize)]
pub struct VisualProfile {
    pub story_id: String,
    pub world_style_prompt: String,
    pub character_style_prompt: String,
    pub negative_prompt: String,
    pub palette: String,
    pub updated_at: String,
}

#[derive(Debug, Deserialize)]
pub struct VisualProfileUpdate {
    #[serde(default)]
    pub world_style_prompt: String,
    #[serde(default)]
    pub character_style_prompt: String,
    #[serde(default)]
    pub negative_prompt: String,
    #[serde(default)]
    pub palette: String,
}

#[derive(Debug, Deserialize)]
pub struct GenerateVisualAssetsRequest {
    #[serde(default)]
    pub asset_ids: Vec<String>,
    #[serde(default)]
    pub force: bool,
    #[serde(default)]
    pub limit: Option<usize>,
}

#[derive(Debug, Clone, Serialize)]
pub struct VisualAsset {
    pub id: String,
    pub story_id: String,
    pub kind: String,
    pub subject: String,
    pub entity_id: String,
    pub prompt: String,
    pub negative_prompt: String,
    pub status: String,
    pub url: String,
    pub provider: String,
    pub source: String,
    pub error: String,
    pub turn: i64,
    pub updated_at: String,
}

#[derive(Debug, Clone)]
struct ImageGenerationConfig {
    base_url: String,
    api_key: String,
    model: String,
    provider: String,
    openclaw_bridge_url: String,
    default_size: String,
    location_size: String,
    character_size: String,
    quality: String,
    timeout_seconds: u64,
    auto_generate: bool,
}

#[derive(Debug, Deserialize)]
struct GatewayConfig {
    ai: Option<GatewayAiConfig>,
}

#[derive(Debug, Deserialize)]
struct GatewayAiConfig {
    litellm: Option<GatewayHttpProviderConfig>,
}

#[derive(Debug, Deserialize)]
struct GatewayHttpProviderConfig {
    base_url: Option<String>,
    api_key: Option<String>,
}

#[derive(Debug, Deserialize)]
struct ImageGenerateResponse {
    data: Vec<ImageGenerateData>,
}

#[derive(Debug, Deserialize)]
struct ImageGenerateData {
    b64_json: Option<String>,
    url: Option<String>,
}

#[derive(Debug, Deserialize)]
struct OpenClawGenerateResponse {
    ok: bool,
    image_b64: Option<String>,
    error: Option<String>,
}

#[derive(Debug)]
struct VisualSpec {
    kind: String,
    subject: String,
    entity_id: String,
    prompt: String,
    negative_prompt: String,
    turn: i64,
}

pub async fn visual_assets(
    pool: &SqlitePool,
    story_id: &str,
) -> anyhow::Result<VisualAssetsResponse> {
    let snapshot = db::snapshot(pool, story_id).await?;
    let profile = ensure_profile(pool, &snapshot).await?;
    let specs = visual_specs(&snapshot, &profile);
    ensure_asset_rows(pool, story_id, &specs).await?;
    let assets = list_assets(pool, story_id).await?;
    Ok(VisualAssetsResponse { profile, assets })
}

pub async fn ensure_visual_asset_version_schema(pool: &SqlitePool) -> anyhow::Result<()> {
    sqlx::query(
        r#"CREATE TABLE IF NOT EXISTS visual_asset_versions (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            asset_id TEXT NOT NULL REFERENCES visual_assets(id) ON DELETE CASCADE,
            story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
            kind TEXT NOT NULL,
            subject TEXT NOT NULL,
            url TEXT NOT NULL DEFAULT '',
            file_path TEXT NOT NULL DEFAULT '',
            prompt TEXT NOT NULL DEFAULT '',
            negative_prompt TEXT NOT NULL DEFAULT '',
            provider TEXT NOT NULL DEFAULT '',
            turn INTEGER NOT NULL DEFAULT 0,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )"#,
    )
    .execute(pool)
    .await
    .context("creating visual_asset_versions table")?;
    sqlx::query(
        r#"CREATE INDEX IF NOT EXISTS idx_visual_asset_versions_asset
           ON visual_asset_versions(asset_id, created_at DESC)"#,
    )
    .execute(pool)
    .await
    .context("creating visual asset version asset index")?;
    sqlx::query(
        r#"CREATE INDEX IF NOT EXISTS idx_visual_asset_versions_story
           ON visual_asset_versions(story_id, kind, subject, created_at DESC)"#,
    )
    .execute(pool)
    .await
    .context("creating visual asset version story index")?;
    Ok(())
}

pub async fn update_profile(
    pool: &SqlitePool,
    story_id: &str,
    update: VisualProfileUpdate,
) -> anyhow::Result<VisualAssetsResponse> {
    db::snapshot(pool, story_id).await?;
    sqlx::query(
        r#"INSERT INTO story_visual_profiles (
              story_id, world_style_prompt, character_style_prompt, negative_prompt, palette
           )
           VALUES (?, ?, ?, ?, ?)
           ON CONFLICT(story_id) DO UPDATE SET
              world_style_prompt = excluded.world_style_prompt,
              character_style_prompt = excluded.character_style_prompt,
              negative_prompt = excluded.negative_prompt,
              palette = excluded.palette,
              updated_at = CURRENT_TIMESTAMP"#,
    )
    .bind(story_id)
    .bind(update.world_style_prompt.trim())
    .bind(update.character_style_prompt.trim())
    .bind(update.negative_prompt.trim())
    .bind(update.palette.trim())
    .execute(pool)
    .await
    .with_context(|| format!("saving visual profile for {story_id}"))?;

    visual_assets(pool, story_id).await
}

pub async fn update_profile_with_defaults(
    pool: &SqlitePool,
    story_id: &str,
    update: VisualProfileUpdate,
) -> anyhow::Result<VisualAssetsResponse> {
    let snapshot = db::snapshot(pool, story_id).await?;
    let defaults = default_profile(&snapshot);
    update_profile(
        pool,
        story_id,
        VisualProfileUpdate {
            world_style_prompt: clean_or(&update.world_style_prompt, &defaults.world_style_prompt),
            character_style_prompt: clean_or(
                &update.character_style_prompt,
                &defaults.character_style_prompt,
            ),
            negative_prompt: clean_or(&update.negative_prompt, &defaults.negative_prompt),
            palette: clean_or(&update.palette, &defaults.palette),
        },
    )
    .await
}

pub fn spawn_auto_generation(state: Arc<AppState>, story_id: String) {
    if !image_generation_config(&state)
        .map(|config| config.auto_generate && image_generation_available(&config))
        .unwrap_or(false)
    {
        return;
    }

    tokio::spawn(async move {
        let request = GenerateVisualAssetsRequest {
            asset_ids: Vec::new(),
            force: false,
            limit: Some(3),
        };
        if let Err(err) = generate_visual_assets(state.as_ref(), &story_id, request).await {
            tracing::warn!(story_id = %story_id, error = %err, "visual asset auto-generation failed");
        }
    });
}

pub async fn generate_visual_assets(
    state: &AppState,
    story_id: &str,
    request: GenerateVisualAssetsRequest,
) -> anyhow::Result<VisualAssetsResponse> {
    visual_assets(&state.pool, story_id).await?;
    let config = image_generation_config(state)?;
    if !image_generation_available(&config) {
        return Err(anyhow!(
            "image generation provider is not configured; set ONEDAY_IMAGEGEN_PROVIDER=openclaw-bridge or configure ONEDAY_IMAGEGEN_API_KEY/ONEDAY_LITELLM_API_KEY"
        ));
    }

    let targets = generation_targets(&state.pool, story_id, &request).await?;
    let client = Client::builder()
        .timeout(Duration::from_secs(config.timeout_seconds))
        .build()
        .context("building image generation HTTP client")?;

    for asset in targets {
        if let Err(err) = mark_asset_running(&state.pool, &asset.id, &config).await {
            tracing::warn!(asset_id = %asset.id, error = %err, "could not mark visual asset running");
            continue;
        }
        match generate_one_asset(&client, state, &config, &asset).await {
            Ok((url, file_path)) => {
                mark_asset_ready(&state.pool, &asset.id, &url, &file_path, &config)
                    .await
                    .with_context(|| format!("marking visual asset {} ready", asset.id))?;
                if let Err(err) =
                    record_asset_version(&state.pool, &asset, &url, &file_path, &config).await
                {
                    tracing::warn!(asset_id = %asset.id, error = %err, "could not record visual asset version");
                }
            }
            Err(err) => {
                let _ = mark_asset_failed(&state.pool, &asset.id, &err.to_string(), &config).await;
            }
        }
    }

    visual_assets(&state.pool, story_id).await
}

async fn generation_targets(
    pool: &SqlitePool,
    story_id: &str,
    request: &GenerateVisualAssetsRequest,
) -> anyhow::Result<Vec<VisualAsset>> {
    let requested: HashSet<&str> = request.asset_ids.iter().map(String::as_str).collect();
    let limit = request.limit.unwrap_or(6).clamp(1, 12);
    let assets = list_assets(pool, story_id).await?;
    Ok(assets
        .into_iter()
        .filter(|asset| requested.is_empty() || requested.contains(asset.id.as_str()))
        .filter(|asset| request.force || asset.status != "ready")
        .filter(|asset| asset.status != "running")
        .take(limit)
        .collect())
}

async fn mark_asset_running(
    pool: &SqlitePool,
    asset_id: &str,
    config: &ImageGenerationConfig,
) -> anyhow::Result<()> {
    sqlx::query(
        r#"UPDATE visual_assets
           SET status = 'running', provider = ?, error = '', updated_at = CURRENT_TIMESTAMP
           WHERE id = ?"#,
    )
    .bind(provider_label(config))
    .bind(asset_id)
    .execute(pool)
    .await?;
    Ok(())
}

async fn mark_asset_ready(
    pool: &SqlitePool,
    asset_id: &str,
    url: &str,
    file_path: &str,
    config: &ImageGenerationConfig,
) -> anyhow::Result<()> {
    sqlx::query(
        r#"UPDATE visual_assets
           SET status = 'ready', url = ?, file_path = ?, provider = ?, error = '',
               updated_at = CURRENT_TIMESTAMP
           WHERE id = ?"#,
    )
    .bind(url)
    .bind(file_path)
    .bind(provider_label(config))
    .bind(asset_id)
    .execute(pool)
    .await?;
    Ok(())
}

async fn mark_asset_failed(
    pool: &SqlitePool,
    asset_id: &str,
    error: &str,
    config: &ImageGenerationConfig,
) -> anyhow::Result<()> {
    sqlx::query(
        r#"UPDATE visual_assets
           SET status = 'failed', provider = ?, error = ?, updated_at = CURRENT_TIMESTAMP
           WHERE id = ?"#,
    )
    .bind(provider_label(config))
    .bind(compact_error(error))
    .bind(asset_id)
    .execute(pool)
    .await?;
    Ok(())
}

async fn record_asset_version(
    pool: &SqlitePool,
    asset: &VisualAsset,
    url: &str,
    file_path: &str,
    config: &ImageGenerationConfig,
) -> anyhow::Result<()> {
    sqlx::query(
        r#"INSERT INTO visual_asset_versions (
              asset_id, story_id, kind, subject, url, file_path, prompt,
              negative_prompt, provider, turn
           )
           VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"#,
    )
    .bind(&asset.id)
    .bind(&asset.story_id)
    .bind(&asset.kind)
    .bind(&asset.subject)
    .bind(url)
    .bind(file_path)
    .bind(&asset.prompt)
    .bind(&asset.negative_prompt)
    .bind(provider_label(config))
    .bind(asset.turn)
    .execute(pool)
    .await?;
    Ok(())
}

async fn generate_one_asset(
    client: &Client,
    state: &AppState,
    config: &ImageGenerationConfig,
    asset: &VisualAsset,
) -> anyhow::Result<(String, String)> {
    if is_openclaw_bridge(config) {
        return generate_one_openclaw_asset(client, state, config, asset).await;
    }

    let size = asset_size(config, asset);
    let prompt = final_prompt(asset);
    let mut payload = serde_json::json!({
        "model": config.model,
        "prompt": prompt,
        "size": size,
        "n": 1
    });
    if !config.quality.trim().is_empty() {
        payload["quality"] = Value::String(config.quality.clone());
    }

    let endpoint = format!(
        "{}/images/generations",
        config.base_url.trim_end_matches('/')
    );
    let response = client
        .post(endpoint)
        .bearer_auth(&config.api_key)
        .json(&payload)
        .send()
        .await
        .with_context(|| format!("requesting image for {}", asset.subject))?;
    let status = response.status();
    if !status.is_success() {
        let detail = response.text().await.unwrap_or_default();
        return Err(anyhow!(
            "image provider returned {}: {}",
            status,
            compact_error(&detail)
        ));
    }

    let response: ImageGenerateResponse = response
        .json()
        .await
        .context("decoding image generation response")?;
    let first = response
        .data
        .first()
        .ok_or_else(|| anyhow!("image provider returned no images"))?;
    let bytes = if let Some(encoded) = &first.b64_json {
        base64::engine::general_purpose::STANDARD
            .decode(encoded)
            .context("decoding generated image base64")?
    } else if let Some(url) = &first.url {
        client
            .get(url)
            .send()
            .await
            .with_context(|| format!("downloading generated image for {}", asset.subject))?
            .error_for_status()
            .context("generated image download failed")?
            .bytes()
            .await
            .context("reading generated image bytes")?
            .to_vec()
    } else {
        return Err(anyhow!("image provider returned neither b64_json nor url"));
    };
    if bytes.is_empty() {
        return Err(anyhow!("image provider returned empty image bytes"));
    }

    persist_generated_asset(state, asset, bytes, "png").await
}

async fn generate_one_openclaw_asset(
    client: &Client,
    state: &AppState,
    config: &ImageGenerationConfig,
    asset: &VisualAsset,
) -> anyhow::Result<(String, String)> {
    let prompt = final_prompt(asset);
    let payload = serde_json::json!({
        "prompt": prompt,
        "size": asset_size(config, asset),
        "output_format": "png"
    });
    let response = client
        .post(&config.openclaw_bridge_url)
        .json(&payload)
        .send()
        .await
        .with_context(|| format!("requesting OpenClaw image for {}", asset.subject))?;
    let status = response.status();
    let raw = response
        .text()
        .await
        .context("reading OpenClaw image response")?;
    if !status.is_success() {
        return Err(anyhow!(
            "OpenClaw image bridge returned {}: {}",
            status,
            compact_error(&raw)
        ));
    }

    let response: OpenClawGenerateResponse =
        serde_json::from_str(&raw).context("decoding OpenClaw image response")?;
    if !response.ok {
        return Err(anyhow!(
            "OpenClaw image bridge failed: {}",
            compact_error(response.error.as_deref().unwrap_or("unknown error"))
        ));
    }
    let encoded = response
        .image_b64
        .filter(|value| !value.trim().is_empty())
        .ok_or_else(|| anyhow!("OpenClaw image bridge returned no image_b64"))?;
    let bytes = base64::engine::general_purpose::STANDARD
        .decode(encoded)
        .context("decoding OpenClaw image base64")?;
    if bytes.is_empty() {
        return Err(anyhow!("OpenClaw image bridge returned empty image bytes"));
    }

    persist_generated_asset(state, asset, bytes, "png").await
}

async fn persist_generated_asset(
    state: &AppState,
    asset: &VisualAsset,
    bytes: Vec<u8>,
    extension: &str,
) -> anyhow::Result<(String, String)> {
    let story_slug = slug(&asset.story_id);
    let subject_slug = slug(&format!("{}-{}", asset.kind, asset.subject));
    let hash = short_hash(&bytes);
    let extension = extension.trim_start_matches('.').trim();
    let extension = if extension.is_empty() {
        "png"
    } else {
        extension
    };
    let filename = format!("{subject_slug}-{hash}.{extension}");
    let dir = state.paths.visual_asset_dir.join(&story_slug);
    fs::create_dir_all(&dir)
        .await
        .with_context(|| format!("creating visual asset directory {}", dir.display()))?;
    let file_path = dir.join(&filename);
    fs::write(&file_path, bytes)
        .await
        .with_context(|| format!("writing generated image {}", file_path.display()))?;
    let url = format!("/generated/assets/{story_slug}/{filename}");
    Ok((url, file_path.to_string_lossy().to_string()))
}

fn final_prompt(asset: &VisualAsset) -> String {
    let mut prompt = clean_or(
        &asset.prompt,
        "Create a polished visual asset for this story.",
    );
    if !asset.negative_prompt.trim().is_empty() {
        prompt.push_str("\nAvoid: ");
        prompt.push_str(asset.negative_prompt.trim());
    }
    prompt
}

fn asset_size(config: &ImageGenerationConfig, asset: &VisualAsset) -> String {
    match asset.kind.as_str() {
        "location" => clean_or(&config.location_size, &config.default_size),
        "character" => clean_or(&config.character_size, &config.default_size),
        _ => config.default_size.clone(),
    }
}

fn image_generation_config(state: &AppState) -> anyhow::Result<ImageGenerationConfig> {
    let file_config = read_gateway_config(&state.paths.config_path)?;
    let litellm = file_config.ai.and_then(|ai| ai.litellm);
    let config_base_url = litellm
        .as_ref()
        .and_then(|provider| provider.base_url.clone())
        .unwrap_or_default();
    let config_api_key = litellm
        .as_ref()
        .and_then(|provider| provider.api_key.clone())
        .map(|value| expand_env_refs(&value))
        .unwrap_or_default();
    let base_url = first_env(&["ONEDAY_IMAGEGEN_BASE_URL", "ONEDAY_IMAGE_BASE_URL"])
        .or_else(|| non_empty(config_base_url))
        .unwrap_or_else(|| "https://api.openai.com/v1".to_string());
    let api_key = first_env(&[
        "ONEDAY_IMAGEGEN_API_KEY",
        "ONEDAY_IMAGE_API_KEY",
        "ONEDAY_LITELLM_API_KEY",
        "OPENAI_API_KEY",
    ])
    .or_else(|| non_empty(config_api_key))
    .unwrap_or_default();
    let model = first_env(&["ONEDAY_IMAGEGEN_MODEL", "ONEDAY_IMAGE_MODEL"])
        .unwrap_or_else(|| "gpt-image-2".to_string());

    Ok(ImageGenerationConfig {
        base_url,
        api_key,
        model,
        provider: first_env(&["ONEDAY_IMAGEGEN_PROVIDER", "ONEDAY_IMAGE_PROVIDER"])
            .unwrap_or_else(|| "openai-compatible".to_string()),
        openclaw_bridge_url: first_env(&[
            "ONEDAY_IMAGEGEN_OPENCLAW_URL",
            "ONEDAY_OPENCLAW_IMAGEGEN_URL",
        ])
        .unwrap_or_else(|| "http://openclaw-imagegen:8099/generate".to_string()),
        default_size: first_env(&["ONEDAY_IMAGEGEN_SIZE", "ONEDAY_IMAGE_SIZE"])
            .unwrap_or_else(|| "1024x1024".to_string()),
        location_size: first_env(&["ONEDAY_IMAGEGEN_LOCATION_SIZE"])
            .unwrap_or_else(|| "1536x1024".to_string()),
        character_size: first_env(&["ONEDAY_IMAGEGEN_CHARACTER_SIZE"])
            .unwrap_or_else(|| "1024x1024".to_string()),
        quality: first_env(&["ONEDAY_IMAGEGEN_QUALITY"]).unwrap_or_default(),
        timeout_seconds: first_env(&["ONEDAY_IMAGEGEN_TIMEOUT_SECONDS"])
            .and_then(|value| value.parse::<u64>().ok())
            .unwrap_or(180),
        auto_generate: first_env(&["ONEDAY_IMAGEGEN_AUTOGENERATE"])
            .map(|value| parse_bool(&value))
            .unwrap_or(true),
    })
}

fn is_openclaw_bridge(config: &ImageGenerationConfig) -> bool {
    matches!(
        config.provider.trim().to_ascii_lowercase().as_str(),
        "openclaw" | "openclaw-bridge" | "codex-oauth"
    )
}

fn image_generation_available(config: &ImageGenerationConfig) -> bool {
    if is_openclaw_bridge(config) {
        return !config.openclaw_bridge_url.trim().is_empty();
    }
    !config.api_key.trim().is_empty()
}

fn read_gateway_config(path: &Path) -> anyhow::Result<GatewayConfig> {
    let raw = std::fs::read_to_string(path)
        .with_context(|| format!("reading gateway image config {}", path.display()))?;
    serde_yaml::from_str(&raw).with_context(|| format!("parsing {}", path.display()))
}

fn provider_label(config: &ImageGenerationConfig) -> String {
    format!("{}:{}", config.provider, config.model)
}

fn short_hash(bytes: &[u8]) -> String {
    let digest = Sha256::digest(bytes);
    hex_prefix(&digest, 12)
}

fn hex_prefix(bytes: &[u8], len: usize) -> String {
    let mut out = String::new();
    for byte in bytes {
        out.push_str(&format!("{byte:02x}"));
        if out.len() >= len {
            break;
        }
    }
    out.truncate(len);
    out
}

fn first_env(names: &[&str]) -> Option<String> {
    names.iter().find_map(|name| {
        std::env::var(name)
            .ok()
            .map(|value| value.trim().to_string())
            .filter(|value| !value.is_empty())
    })
}

fn non_empty(value: String) -> Option<String> {
    let trimmed = value.trim().to_string();
    (!trimmed.is_empty()).then_some(trimmed)
}

fn parse_bool(value: &str) -> bool {
    matches!(
        value.trim().to_ascii_lowercase().as_str(),
        "1" | "true" | "yes" | "on"
    )
}

fn expand_env_refs(value: &str) -> String {
    let trimmed = value.trim();
    if let Some(name) = trimmed
        .strip_prefix("${")
        .and_then(|rest| rest.strip_suffix('}'))
    {
        return std::env::var(name).unwrap_or_default();
    }
    trimmed.to_string()
}

fn compact_error(value: &str) -> String {
    let collapsed = value.split_whitespace().collect::<Vec<_>>().join(" ");
    if collapsed.len() > 900 {
        format!("{}...", &collapsed[..900])
    } else {
        collapsed
    }
}

async fn ensure_profile(
    pool: &SqlitePool,
    snapshot: &db::StorySnapshot,
) -> anyhow::Result<VisualProfile> {
    if let Some(profile) = get_profile(pool, &snapshot.story.id).await? {
        return Ok(profile);
    }

    let defaults = default_profile(snapshot);
    sqlx::query(
        r#"INSERT OR IGNORE INTO story_visual_profiles (
              story_id, world_style_prompt, character_style_prompt, negative_prompt, palette
           )
           VALUES (?, ?, ?, ?, ?)"#,
    )
    .bind(&snapshot.story.id)
    .bind(&defaults.world_style_prompt)
    .bind(&defaults.character_style_prompt)
    .bind(&defaults.negative_prompt)
    .bind(&defaults.palette)
    .execute(pool)
    .await
    .with_context(|| format!("creating visual profile for {}", snapshot.story.id))?;

    get_profile(pool, &snapshot.story.id)
        .await?
        .ok_or_else(|| anyhow::anyhow!("visual profile was not created"))
}

async fn get_profile(pool: &SqlitePool, story_id: &str) -> anyhow::Result<Option<VisualProfile>> {
    let row = sqlx::query(
        r#"SELECT story_id, world_style_prompt, character_style_prompt,
                  negative_prompt, palette, CAST(updated_at AS TEXT) AS updated_at
           FROM story_visual_profiles
           WHERE story_id = ?"#,
    )
    .bind(story_id)
    .fetch_optional(pool)
    .await?;

    Ok(row.map(|row| VisualProfile {
        story_id: row_string(&row, "story_id"),
        world_style_prompt: row_string(&row, "world_style_prompt"),
        character_style_prompt: row_string(&row, "character_style_prompt"),
        negative_prompt: row_string(&row, "negative_prompt"),
        palette: row_string(&row, "palette"),
        updated_at: row_string(&row, "updated_at"),
    }))
}

fn default_profile(snapshot: &db::StorySnapshot) -> VisualProfile {
    let genre = clean_or(&snapshot.story.genre, "narrative mystery");
    let tone = clean_or(&snapshot.story.tone, "grounded and atmospheric");
    let setting = value_to_text(&snapshot.world.known_locations);
    let style_basis = if setting.is_empty() {
        format!("{genre}, {tone}")
    } else {
        format!("{genre}, {tone}, {setting}")
    };
    VisualProfile {
        story_id: snapshot.story.id.clone(),
        world_style_prompt: format!(
            "Polished cinematic concept art for a text RPG world. Base style: {style_basis}. Use real place texture, readable silhouettes, no UI text."
        ),
        character_style_prompt: format!(
            "Painterly realistic RPG portraits for {genre}. Faces should feel specific, grounded, emotionally readable, and coherent with the world style."
        ),
        negative_prompt: "no text, no logos, no watermark, no UI, no unreadable signage, no cartoon, no anime, no pure black".to_string(),
        palette: "charcoal, warm amber, smoke blue, restrained contrast".to_string(),
        updated_at: String::new(),
    }
}

fn visual_specs(snapshot: &db::StorySnapshot, profile: &VisualProfile) -> Vec<VisualSpec> {
    let mut specs = Vec::new();
    let location = snapshot.world.current_location.trim();
    if !location.is_empty() {
        let details = first_string(
            &snapshot.world.known_locations,
            &["details", "description", "notes", "summary"],
        )
        .unwrap_or_else(|| value_to_text(&snapshot.world.known_locations));
        specs.push(VisualSpec {
            kind: "location".to_string(),
            subject: location.to_string(),
            entity_id: snapshot.world.id.clone(),
            prompt: format!(
                "{} Current location: {}. Details: {}. Composition: wide browser hero banner, deep perspective, safe area for overlay text at left. Palette: {}.",
                profile.world_style_prompt,
                location,
                clean_or(&details, "derive visual detail from the story context"),
                clean_or(&profile.palette, "dark warm noir")
            ),
            negative_prompt: profile.negative_prompt.clone(),
            turn: snapshot.world.current_turn,
        });
    }

    for npc in snapshot.panels.npcs.iter().take(8) {
        let appearance = npc
            .fields
            .get("appearance")
            .and_then(Value::as_str)
            .unwrap_or("");
        let role = npc.fields.get("role").and_then(Value::as_str).unwrap_or("");
        let relationship = value_to_text(npc.fields.get("relationship").unwrap_or(&Value::Null));
        specs.push(VisualSpec {
            kind: "character".to_string(),
            subject: npc.name.clone(),
            entity_id: npc.id.clone(),
            prompt: format!(
                "{} Character: {}. Role: {}. Appearance: {}. Relationship context: {}. Composition: square bust-up portrait, readable at small card size, coherent lighting with the current scene.",
                profile.character_style_prompt,
                npc.name,
                clean_or(role, "unknown"),
                clean_or(appearance, "derive from story context"),
                clean_or(&relationship, "unknown")
            ),
            negative_prompt: profile.negative_prompt.clone(),
            turn: snapshot.world.current_turn,
        });
    }
    specs
}

async fn ensure_asset_rows(
    pool: &SqlitePool,
    story_id: &str,
    specs: &[VisualSpec],
) -> anyhow::Result<()> {
    for spec in specs {
        let id = asset_id(story_id, &spec.kind, &spec.subject);
        sqlx::query(
            r#"INSERT INTO visual_assets (
                  id, story_id, kind, subject, entity_id, prompt, negative_prompt,
                  status, provider, source, turn
               )
               VALUES (?, ?, ?, ?, ?, ?, ?, 'pending', 'codex-imagegen', 'auto-profile', ?)
               ON CONFLICT(story_id, kind, subject) DO UPDATE SET
                  entity_id = CASE WHEN visual_assets.entity_id = '' THEN excluded.entity_id ELSE visual_assets.entity_id END,
                  prompt = CASE WHEN visual_assets.prompt = '' THEN excluded.prompt ELSE visual_assets.prompt END,
                  negative_prompt = CASE WHEN visual_assets.negative_prompt = '' THEN excluded.negative_prompt ELSE visual_assets.negative_prompt END,
                  turn = excluded.turn,
                  updated_at = CURRENT_TIMESTAMP"#,
        )
        .bind(id)
        .bind(story_id)
        .bind(&spec.kind)
        .bind(&spec.subject)
        .bind(&spec.entity_id)
        .bind(&spec.prompt)
        .bind(&spec.negative_prompt)
        .bind(spec.turn)
        .execute(pool)
        .await
        .with_context(|| format!("ensuring visual asset {} {}", spec.kind, spec.subject))?;
    }
    Ok(())
}

async fn list_assets(pool: &SqlitePool, story_id: &str) -> anyhow::Result<Vec<VisualAsset>> {
    let rows = sqlx::query(
        r#"SELECT id, story_id, kind, subject, entity_id, prompt, negative_prompt,
                  status, url, provider, source, error, turn,
                  CAST(updated_at AS TEXT) AS updated_at
           FROM visual_assets
           WHERE story_id = ?
           ORDER BY
             CASE kind WHEN 'location' THEN 0 WHEN 'character' THEN 1 ELSE 2 END,
             updated_at DESC,
             subject ASC"#,
    )
    .bind(story_id)
    .fetch_all(pool)
    .await?;

    Ok(rows
        .into_iter()
        .map(|row| VisualAsset {
            id: row_string(&row, "id"),
            story_id: row_string(&row, "story_id"),
            kind: row_string(&row, "kind"),
            subject: row_string(&row, "subject"),
            entity_id: row_string(&row, "entity_id"),
            prompt: row_string(&row, "prompt"),
            negative_prompt: row_string(&row, "negative_prompt"),
            status: row_string(&row, "status"),
            url: row_string(&row, "url"),
            provider: row_string(&row, "provider"),
            source: row_string(&row, "source"),
            error: row_string(&row, "error"),
            turn: row.try_get("turn").unwrap_or_default(),
            updated_at: row_string(&row, "updated_at"),
        })
        .collect())
}

fn asset_id(story_id: &str, kind: &str, subject: &str) -> String {
    format!(
        "vis_{}_{}_{}",
        slug(story_id).chars().take(24).collect::<String>(),
        slug(kind),
        slug(subject).chars().take(48).collect::<String>()
    )
}

fn slug(value: &str) -> String {
    let mut out = String::new();
    let mut last_dash = false;
    for ch in value.chars() {
        if ch.is_ascii_alphanumeric() {
            out.push(ch.to_ascii_lowercase());
            last_dash = false;
        } else if !last_dash {
            out.push('-');
            last_dash = true;
        }
    }
    let trimmed = out.trim_matches('-').to_string();
    if trimmed.is_empty() {
        "asset".to_string()
    } else {
        trimmed
    }
}

fn row_string(row: &sqlx::sqlite::SqliteRow, key: &str) -> String {
    row.try_get::<Option<String>, _>(key)
        .ok()
        .flatten()
        .unwrap_or_default()
}

fn clean_or(value: &str, fallback: &str) -> String {
    let cleaned = value.split_whitespace().collect::<Vec<_>>().join(" ");
    if cleaned.is_empty() {
        fallback.to_string()
    } else {
        cleaned
    }
}

fn first_string(value: &Value, keys: &[&str]) -> Option<String> {
    match value {
        Value::Object(map) => {
            for key in keys {
                if let Some(found) = map.get(*key).and_then(Value::as_str) {
                    let cleaned = clean_or(found, "");
                    if !cleaned.is_empty() {
                        return Some(cleaned);
                    }
                }
            }
            for child in map.values() {
                if let Some(found) = first_string(child, keys) {
                    return Some(found);
                }
            }
            None
        }
        Value::Array(items) => items.iter().find_map(|item| first_string(item, keys)),
        Value::String(text) => {
            let cleaned = clean_or(text, "");
            (!cleaned.is_empty()).then_some(cleaned)
        }
        _ => None,
    }
}

fn value_to_text(value: &Value) -> String {
    match value {
        Value::Null => String::new(),
        Value::String(text) => clean_or(text, ""),
        Value::Number(number) => number.to_string(),
        Value::Bool(flag) => flag.to_string(),
        Value::Array(items) => items
            .iter()
            .take(4)
            .map(value_to_text)
            .filter(|text| !text.is_empty())
            .collect::<Vec<_>>()
            .join("; "),
        Value::Object(map) => map
            .iter()
            .take(6)
            .map(|(key, value)| {
                let text = value_to_text(value);
                if text.is_empty() {
                    key.to_string()
                } else {
                    format!("{key}: {text}")
                }
            })
            .collect::<Vec<_>>()
            .join("; "),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn expands_simple_env_reference() {
        std::env::set_var("ONEDAY_TEST_IMAGE_KEY", "secret-value");
        assert_eq!(expand_env_refs("${ONEDAY_TEST_IMAGE_KEY}"), "secret-value");
        std::env::remove_var("ONEDAY_TEST_IMAGE_KEY");
    }

    #[test]
    fn chooses_asset_specific_sizes() {
        let config = ImageGenerationConfig {
            base_url: "http://example.test/v1".to_string(),
            api_key: "key".to_string(),
            model: "gpt-image-2".to_string(),
            provider: "test".to_string(),
            openclaw_bridge_url: "http://openclaw-imagegen:8099/generate".to_string(),
            default_size: "1024x1024".to_string(),
            location_size: "1536x1024".to_string(),
            character_size: "768x768".to_string(),
            quality: String::new(),
            timeout_seconds: 10,
            auto_generate: true,
        };
        let mut asset = VisualAsset {
            id: "asset".to_string(),
            story_id: "story".to_string(),
            kind: "location".to_string(),
            subject: "Station".to_string(),
            entity_id: "world".to_string(),
            prompt: String::new(),
            negative_prompt: String::new(),
            status: "pending".to_string(),
            url: String::new(),
            provider: String::new(),
            source: String::new(),
            error: String::new(),
            turn: 1,
            updated_at: String::new(),
        };
        assert_eq!(asset_size(&config, &asset), "1536x1024");
        asset.kind = "character".to_string();
        assert_eq!(asset_size(&config, &asset), "768x768");
        asset.kind = "item".to_string();
        assert_eq!(asset_size(&config, &asset), "1024x1024");
    }

    #[test]
    fn prompt_includes_negative_direction() {
        let asset = VisualAsset {
            id: "asset".to_string(),
            story_id: "story".to_string(),
            kind: "character".to_string(),
            subject: "Barista".to_string(),
            entity_id: "npc".to_string(),
            prompt: "portrait".to_string(),
            negative_prompt: "no text".to_string(),
            status: "pending".to_string(),
            url: String::new(),
            provider: String::new(),
            source: String::new(),
            error: String::new(),
            turn: 1,
            updated_at: String::new(),
        };
        assert!(final_prompt(&asset).contains("portrait"));
        assert!(final_prompt(&asset).contains("Avoid: no text"));
    }

    #[test]
    fn openclaw_bridge_does_not_require_api_key() {
        let config = ImageGenerationConfig {
            base_url: String::new(),
            api_key: String::new(),
            model: "gpt-image-2".to_string(),
            provider: "openclaw-bridge".to_string(),
            openclaw_bridge_url: "http://openclaw-imagegen:8099/generate".to_string(),
            default_size: "1024x1024".to_string(),
            location_size: "1536x1024".to_string(),
            character_size: "1024x1024".to_string(),
            quality: String::new(),
            timeout_seconds: 10,
            auto_generate: true,
        };
        assert!(is_openclaw_bridge(&config));
        assert!(image_generation_available(&config));
    }

    #[test]
    fn openai_compatible_provider_requires_api_key() {
        let config = ImageGenerationConfig {
            base_url: "http://example.test/v1".to_string(),
            api_key: String::new(),
            model: "gpt-image-2".to_string(),
            provider: "openai-compatible".to_string(),
            openclaw_bridge_url: "http://openclaw-imagegen:8099/generate".to_string(),
            default_size: "1024x1024".to_string(),
            location_size: "1536x1024".to_string(),
            character_size: "1024x1024".to_string(),
            quality: String::new(),
            timeout_seconds: 10,
            auto_generate: true,
        };
        assert!(!is_openclaw_bridge(&config));
        assert!(!image_generation_available(&config));
    }
}
