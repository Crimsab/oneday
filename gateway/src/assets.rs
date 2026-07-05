use crate::{db, events::TurnStreamEvent, AppState};
use anyhow::{anyhow, Context};
use base64::Engine;
use reqwest::Client;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use sha2::{Digest, Sha256};
use sqlx::{Row, SqlitePool};
use std::collections::HashSet;
use std::path::{Path, PathBuf};
use std::sync::Arc;
use std::time::Duration;
use tokio::fs;

#[derive(Debug, Serialize)]
pub struct VisualAssetsResponse {
    pub profile: VisualProfile,
    pub assets: Vec<VisualAsset>,
    pub jobs: Vec<VisualGenerationJobView>,
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

#[derive(Debug, Clone, Deserialize, Serialize)]
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

#[derive(Debug, Clone, Serialize)]
pub struct VisualAssetVersion {
    pub id: i64,
    pub asset_id: String,
    pub story_id: String,
    pub kind: String,
    pub subject: String,
    pub url: String,
    pub prompt: String,
    pub revised_prompt: String,
    pub negative_prompt: String,
    pub provider: String,
    pub turn: i64,
    pub created_at: String,
}

#[derive(Debug, Clone, Serialize)]
pub struct VisualGenerationJobView {
    pub id: i64,
    pub asset_id: String,
    pub story_id: String,
    pub status: String,
    pub attempts: i64,
    pub max_attempts: i64,
    pub locked_until: String,
    pub error: String,
    pub provider: String,
    pub started_at: String,
    pub finished_at: String,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Debug, Deserialize)]
pub struct VisualAssetCleanupRequest {
    #[serde(default)]
    pub dry_run: bool,
}

#[derive(Debug, Serialize)]
pub struct VisualAssetCleanupResponse {
    pub story_id: String,
    pub dry_run: bool,
    pub deleted_files: Vec<String>,
    pub kept_files: Vec<String>,
}

#[derive(Debug, Deserialize)]
pub struct VisualAssetPromptUpdate {
    #[serde(default)]
    pub prompt: String,
    #[serde(default)]
    pub negative_prompt: String,
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
    default_resolution: String,
    location_resolution: String,
    character_resolution: String,
    default_aspect_ratio: String,
    location_aspect_ratio: String,
    character_aspect_ratio: String,
    quality: String,
    output_format: String,
    background: String,
    timeout_seconds: u64,
    auto_generate: bool,
    append_negative_prompt: bool,
}

#[derive(Debug, Deserialize)]
struct GatewayConfig {
    ai: Option<GatewayAiConfig>,
}

#[derive(Debug, Deserialize)]
struct GatewayAiConfig {
    litellm: Option<GatewayHttpProviderConfig>,
    image_generation: Option<GatewayImageGenerationConfig>,
}

#[derive(Debug, Deserialize)]
struct GatewayHttpProviderConfig {
    base_url: Option<String>,
    api_key: Option<String>,
}

#[derive(Debug, Deserialize)]
struct GatewayImageGenerationConfig {
    provider: Option<String>,
    base_url: Option<String>,
    api_key: Option<String>,
    model: Option<String>,
    openclaw_bridge_url: Option<String>,
    default_size: Option<String>,
    location_size: Option<String>,
    character_size: Option<String>,
    default_resolution: Option<String>,
    location_resolution: Option<String>,
    character_resolution: Option<String>,
    default_aspect_ratio: Option<String>,
    location_aspect_ratio: Option<String>,
    character_aspect_ratio: Option<String>,
    quality: Option<String>,
    output_format: Option<String>,
    background: Option<String>,
    timeout_seconds: Option<u64>,
    auto_generate: Option<bool>,
    append_negative_prompt: Option<bool>,
}

#[derive(Debug, Deserialize)]
struct ImageGenerateResponse {
    data: Vec<ImageGenerateData>,
}

#[derive(Debug, Deserialize)]
struct ImageGenerateData {
    b64_json: Option<String>,
    revised_prompt: Option<String>,
    url: Option<String>,
}

#[derive(Debug, Deserialize)]
struct OpenClawGenerateResponse {
    ok: bool,
    image_b64: Option<String>,
    revised_prompt: Option<String>,
    error: Option<String>,
}

#[derive(Debug)]
struct GeneratedAsset {
    url: String,
    file_path: String,
    revised_prompt: String,
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

#[derive(Debug)]
struct VisualGenerationJob {
    id: i64,
    asset: VisualAsset,
    attempts: i64,
    max_attempts: i64,
}

#[derive(Debug, Clone)]
struct QueuedVisualGenerationJob {
    asset: VisualAsset,
    job_id: i64,
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
    let jobs = list_visual_generation_jobs(pool, story_id).await?;
    Ok(VisualAssetsResponse {
        profile,
        assets,
        jobs,
    })
}

pub async fn visual_asset_versions(
    pool: &SqlitePool,
    story_id: &str,
    asset_id: &str,
) -> anyhow::Result<Vec<VisualAssetVersion>> {
    ensure_asset_belongs_to_story(pool, story_id, asset_id).await?;
    let rows = sqlx::query(
        r#"SELECT id, asset_id, story_id, kind, subject, url, prompt,
                  revised_prompt, negative_prompt, provider, turn,
                  CAST(created_at AS TEXT) AS created_at
           FROM visual_asset_versions
           WHERE story_id = ? AND asset_id = ?
           ORDER BY id DESC"#,
    )
    .bind(story_id)
    .bind(asset_id)
    .fetch_all(pool)
    .await?;

    Ok(rows
        .into_iter()
        .map(|row| VisualAssetVersion {
            id: row.try_get("id").unwrap_or_default(),
            asset_id: row_string(&row, "asset_id"),
            story_id: row_string(&row, "story_id"),
            kind: row_string(&row, "kind"),
            subject: row_string(&row, "subject"),
            url: row_string(&row, "url"),
            prompt: row_string(&row, "prompt"),
            revised_prompt: row_string(&row, "revised_prompt"),
            negative_prompt: row_string(&row, "negative_prompt"),
            provider: row_string(&row, "provider"),
            turn: row.try_get("turn").unwrap_or_default(),
            created_at: row_string(&row, "created_at"),
        })
        .collect())
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
            revised_prompt TEXT NOT NULL DEFAULT '',
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
    ensure_text_column(
        pool,
        "visual_asset_versions",
        "revised_prompt",
        "TEXT NOT NULL DEFAULT ''",
    )
    .await?;
    ensure_visual_generation_job_schema(pool).await?;
    recover_stale_visual_jobs(pool).await?;
    Ok(())
}

async fn ensure_visual_generation_job_schema(pool: &SqlitePool) -> anyhow::Result<()> {
    sqlx::query(
        r#"CREATE TABLE IF NOT EXISTS visual_generation_jobs (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            asset_id TEXT NOT NULL REFERENCES visual_assets(id) ON DELETE CASCADE,
            story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
            status TEXT NOT NULL DEFAULT 'queued' CHECK(status IN ('queued', 'running', 'succeeded', 'failed', 'cancelled')),
            attempts INTEGER NOT NULL DEFAULT 0,
            max_attempts INTEGER NOT NULL DEFAULT 3,
            locked_until TEXT NOT NULL DEFAULT '',
            request_payload_json TEXT NOT NULL DEFAULT '{}',
            error TEXT NOT NULL DEFAULT '',
            provider TEXT NOT NULL DEFAULT '',
            started_at TEXT NOT NULL DEFAULT '',
            finished_at TEXT NOT NULL DEFAULT '',
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )"#,
    )
    .execute(pool)
    .await
    .context("creating visual_generation_jobs table")?;
    sqlx::query(
        r#"CREATE INDEX IF NOT EXISTS idx_visual_generation_jobs_status_lock
           ON visual_generation_jobs(status, locked_until, created_at)"#,
    )
    .execute(pool)
    .await
    .context("creating visual generation job status index")?;
    sqlx::query(
        r#"CREATE INDEX IF NOT EXISTS idx_visual_generation_jobs_story
           ON visual_generation_jobs(story_id, status, created_at)"#,
    )
    .execute(pool)
    .await
    .context("creating visual generation job story index")?;
    sqlx::query(
        r#"CREATE UNIQUE INDEX IF NOT EXISTS idx_visual_generation_jobs_active_asset
           ON visual_generation_jobs(asset_id)
           WHERE status IN ('queued', 'running')"#,
    )
    .execute(pool)
    .await
    .context("creating visual generation active asset index")?;
    Ok(())
}

async fn recover_stale_visual_jobs(pool: &SqlitePool) -> anyhow::Result<u64> {
    let now = chrono::Utc::now().to_rfc3339();
    let result = sqlx::query(
        r#"UPDATE visual_generation_jobs
           SET status = 'queued',
               locked_until = '',
               error = CASE WHEN error = '' THEN 'Recovered after gateway restart or stale lock.' ELSE error END,
               updated_at = ?
           WHERE status = 'running' AND locked_until != '' AND locked_until <= ?"#,
    )
    .bind(&now)
    .bind(&now)
    .execute(pool)
    .await
    .context("recovering stale visual generation jobs")?;
    Ok(result.rows_affected())
}

async fn ensure_text_column(
    pool: &SqlitePool,
    table: &str,
    column: &str,
    definition: &str,
) -> anyhow::Result<()> {
    let statement = format!("ALTER TABLE {table} ADD COLUMN {column} {definition}");
    match sqlx::query(&statement).execute(pool).await {
        Ok(_) => Ok(()),
        Err(err)
            if err
                .to_string()
                .to_ascii_lowercase()
                .contains("duplicate column") =>
        {
            Ok(())
        }
        Err(err) => Err(anyhow!("adding column {table}.{column}: {err}")),
    }
}

pub async fn update_asset_prompt(
    pool: &SqlitePool,
    story_id: &str,
    asset_id: &str,
    update: VisualAssetPromptUpdate,
) -> anyhow::Result<VisualAssetsResponse> {
    ensure_asset_belongs_to_story(pool, story_id, asset_id).await?;
    sqlx::query(
        r#"UPDATE visual_assets
           SET prompt = ?, negative_prompt = ?, source = 'manual-asset',
               error = CASE WHEN status = 'failed' THEN '' ELSE error END,
               status = CASE WHEN status = 'failed' THEN 'pending' ELSE status END,
               updated_at = CURRENT_TIMESTAMP
           WHERE story_id = ? AND id = ?"#,
    )
    .bind(update.prompt.trim())
    .bind(update.negative_prompt.trim())
    .bind(story_id)
    .bind(asset_id)
    .execute(pool)
    .await
    .with_context(|| format!("updating visual asset prompt {asset_id}"))?;

    visual_assets(pool, story_id).await
}

pub async fn select_asset_version(
    pool: &SqlitePool,
    story_id: &str,
    asset_id: &str,
    version_id: i64,
) -> anyhow::Result<VisualAssetsResponse> {
    ensure_asset_belongs_to_story(pool, story_id, asset_id).await?;
    let version = sqlx::query(
        r#"SELECT url, file_path, prompt, negative_prompt, provider, turn
           FROM visual_asset_versions
           WHERE story_id = ? AND asset_id = ? AND id = ?"#,
    )
    .bind(story_id)
    .bind(asset_id)
    .bind(version_id)
    .fetch_optional(pool)
    .await?
    .ok_or_else(|| anyhow!("visual asset version not found"))?;

    sqlx::query(
        r#"UPDATE visual_assets
           SET status = 'ready', url = ?, file_path = ?, prompt = ?,
               negative_prompt = ?, provider = ?, error = '', turn = ?,
               updated_at = CURRENT_TIMESTAMP
           WHERE story_id = ? AND id = ?"#,
    )
    .bind(row_string(&version, "url"))
    .bind(row_string(&version, "file_path"))
    .bind(row_string(&version, "prompt"))
    .bind(row_string(&version, "negative_prompt"))
    .bind(row_string(&version, "provider"))
    .bind(version.try_get::<i64, _>("turn").unwrap_or_default())
    .bind(story_id)
    .bind(asset_id)
    .execute(pool)
    .await
    .with_context(|| format!("selecting visual asset version {version_id}"))?;

    visual_assets(pool, story_id).await
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
        if let Err(err) = generate_visual_assets(state.clone(), &story_id, request).await {
            tracing::warn!(story_id = %story_id, error = %err, "visual asset auto-generation failed");
        }
    });
}

pub async fn generate_visual_assets(
    state: Arc<AppState>,
    story_id: &str,
    request: GenerateVisualAssetsRequest,
) -> anyhow::Result<VisualAssetsResponse> {
    visual_assets(&state.pool, story_id).await?;
    let config = image_generation_config(&state)?;
    if !image_generation_available(&config) {
        return Err(anyhow!(
            "image generation provider is not configured; set ONEDAY_IMAGEGEN_PROVIDER=openclaw-bridge or configure ONEDAY_IMAGEGEN_API_KEY/ONEDAY_LITELLM_API_KEY"
        ));
    }

    let targets = generation_targets(&state.pool, story_id, &request).await?;
    let queued = enqueue_visual_generation_jobs(&state.pool, &targets, &request, &config).await?;
    for queued_job in &queued {
        emit_visual_asset_event(
            &state,
            "asset.queued",
            &queued_job.asset,
            Some(queued_job.job_id),
            "queued",
            format!("Queued image generation for {}.", queued_job.asset.subject),
        );
    }
    spawn_visual_generation_worker(state.clone());

    visual_assets(&state.pool, story_id).await
}

pub async fn cancel_story_visual_jobs(pool: &SqlitePool, story_id: &str) -> anyhow::Result<u64> {
    let now = chrono::Utc::now().to_rfc3339();
    let result = sqlx::query(
        r#"UPDATE visual_generation_jobs
           SET status = 'cancelled',
               locked_until = '',
               error = 'Cancelled because the story was deleted.',
               finished_at = ?,
               updated_at = ?
           WHERE story_id = ? AND status IN ('queued', 'running')"#,
    )
    .bind(&now)
    .bind(&now)
    .bind(story_id)
    .execute(pool)
    .await
    .with_context(|| format!("cancelling visual generation jobs for story {story_id}"))?;
    Ok(result.rows_affected())
}

pub async fn cancel_visual_generation_job(
    pool: &SqlitePool,
    story_id: &str,
    job_id: i64,
) -> anyhow::Result<VisualAssetsResponse> {
    let now = chrono::Utc::now().to_rfc3339();
    let row = sqlx::query(
        r#"UPDATE visual_generation_jobs
           SET status = 'cancelled',
               locked_until = '',
               error = 'Cancelled by user.',
               finished_at = ?,
               updated_at = ?
           WHERE story_id = ? AND id = ? AND status IN ('queued', 'running')
           RETURNING asset_id"#,
    )
    .bind(&now)
    .bind(&now)
    .bind(story_id)
    .bind(job_id)
    .fetch_optional(pool)
    .await
    .with_context(|| format!("cancelling visual generation job {job_id}"))?;

    let row = row.ok_or_else(|| anyhow!("active visual generation job not found"))?;
    let asset_id = row_string(&row, "asset_id");
    sqlx::query(
        r#"UPDATE visual_assets
           SET status = CASE WHEN url = '' THEN 'pending' ELSE 'ready' END,
               error = 'Generation cancelled.',
               updated_at = CURRENT_TIMESTAMP
           WHERE story_id = ? AND id = ?"#,
    )
    .bind(story_id)
    .bind(asset_id)
    .execute(pool)
    .await
    .with_context(|| format!("marking visual asset after cancelling job {job_id}"))?;

    visual_assets(pool, story_id).await
}

pub async fn cleanup_visual_asset_files(
    pool: &SqlitePool,
    story_id: &str,
    root: &Path,
    request: VisualAssetCleanupRequest,
) -> anyhow::Result<VisualAssetCleanupResponse> {
    db::snapshot(pool, story_id).await?;
    let story_dir = root.join(slug(story_id));
    let mut kept_files = Vec::new();
    let mut deleted_files = Vec::new();
    let referenced = referenced_visual_asset_paths(pool, story_id).await?;

    let mut entries = match fs::read_dir(&story_dir).await {
        Ok(entries) => entries,
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => {
            return Ok(VisualAssetCleanupResponse {
                story_id: story_id.to_string(),
                dry_run: request.dry_run,
                deleted_files,
                kept_files,
            });
        }
        Err(err) => {
            return Err(err).with_context(|| {
                format!("reading visual asset directory {}", story_dir.display())
            });
        }
    };

    while let Some(entry) = entries.next_entry().await? {
        let file_type = entry.file_type().await?;
        if !file_type.is_file() {
            continue;
        }
        let path = entry.path();
        let display_path = path.to_string_lossy().to_string();
        if referenced.contains(&path) {
            kept_files.push(display_path);
            continue;
        }
        if !request.dry_run {
            fs::remove_file(&path).await.with_context(|| {
                format!("removing unreferenced visual asset {}", path.display())
            })?;
        }
        deleted_files.push(display_path);
    }

    kept_files.sort();
    deleted_files.sort();
    Ok(VisualAssetCleanupResponse {
        story_id: story_id.to_string(),
        dry_run: request.dry_run,
        deleted_files,
        kept_files,
    })
}

pub fn spawn_visual_generation_worker(state: Arc<AppState>) {
    tokio::spawn(async move {
        let Ok(_guard) = state.visual_worker.clone().try_lock_owned() else {
            return;
        };
        if let Err(err) = run_visual_generation_worker(state).await {
            tracing::warn!(error = %err, "visual generation worker stopped");
        }
    });
}

pub fn spawn_visual_generation_maintenance(state: Arc<AppState>) {
    tokio::spawn(async move {
        let mut interval = tokio::time::interval(Duration::from_secs(60));
        loop {
            interval.tick().await;
            match recover_stale_visual_jobs(&state.pool).await {
                Ok(count) if count > 0 => {
                    tracing::info!(count, "recovered stale visual generation jobs");
                    spawn_visual_generation_worker(state.clone());
                }
                Ok(_) => {}
                Err(err) => {
                    tracing::warn!(error = %err, "visual generation maintenance failed");
                }
            }
        }
    });
}

async fn run_visual_generation_worker(state: Arc<AppState>) -> anyhow::Result<()> {
    let config = image_generation_config(&state)?;
    if !image_generation_available(&config) {
        return Ok(());
    }
    let client = Client::builder()
        .timeout(Duration::from_secs(config.timeout_seconds))
        .build()
        .context("building image generation HTTP client")?;

    recover_stale_visual_jobs(&state.pool).await?;
    loop {
        let Some(job) = claim_visual_generation_job(&state.pool, &config).await? else {
            let Some(delay) = next_visual_generation_wakeup(&state.pool).await? else {
                break;
            };
            tokio::time::sleep(delay).await;
            recover_stale_visual_jobs(&state.pool).await?;
            continue;
        };
        if let Err(err) = mark_asset_running(&state.pool, &job.asset.id, &config).await {
            tracing::warn!(asset_id = %job.asset.id, error = %err, "could not mark visual asset running");
            continue;
        }
        emit_visual_asset_event(
            &state,
            "asset.running",
            &job.asset,
            Some(job.id),
            "running",
            format!("Generating image for {}.", job.asset.subject),
        );
        match generate_one_asset(&client, &state, &config, &job.asset).await {
            Ok(generated) => {
                if visual_generation_job_is_cancelled(&state.pool, job.id).await? {
                    discard_generated_asset(&generated).await;
                    emit_visual_asset_event(
                        &state,
                        "asset.cancelled",
                        &job.asset,
                        Some(job.id),
                        "cancelled",
                        format!("Image generation cancelled for {}.", job.asset.subject),
                    );
                    continue;
                }
                if !visual_asset_exists(&state.pool, &job.asset.story_id, &job.asset.id).await? {
                    discard_generated_asset(&generated).await;
                    mark_generation_job_terminal(
                        &state.pool,
                        job.id,
                        "cancelled",
                        "visual asset was deleted before generation completed",
                    )
                    .await?;
                    continue;
                }
                mark_asset_ready(
                    &state.pool,
                    &job.asset.id,
                    &generated.url,
                    &generated.file_path,
                    &config,
                )
                .await
                .with_context(|| format!("marking visual asset {} ready", job.asset.id))?;
                if let Err(err) =
                    record_asset_version(&state.pool, &job.asset, &generated, &config).await
                {
                    tracing::warn!(asset_id = %job.asset.id, error = %err, "could not record visual asset version");
                }
                mark_generation_job_succeeded(&state.pool, job.id).await?;
                emit_visual_asset_event(
                    &state,
                    "asset.ready",
                    &job.asset,
                    Some(job.id),
                    "ready",
                    format!("Image ready for {}.", job.asset.subject),
                );
            }
            Err(err) => {
                if visual_generation_job_is_cancelled(&state.pool, job.id).await? {
                    emit_visual_asset_event(
                        &state,
                        "asset.cancelled",
                        &job.asset,
                        Some(job.id),
                        "cancelled",
                        format!("Image generation cancelled for {}.", job.asset.subject),
                    );
                    continue;
                }
                let terminal = job.attempts >= job.max_attempts;
                let error = err.to_string();
                mark_generation_job_failed_or_retry(&state.pool, &job, &error, &config).await?;
                if terminal {
                    emit_visual_asset_event(
                        &state,
                        "asset.failed",
                        &job.asset,
                        Some(job.id),
                        "failed",
                        format!("Image generation failed for {}.", job.asset.subject),
                    );
                } else {
                    emit_visual_asset_event(
                        &state,
                        "asset.queued",
                        &job.asset,
                        Some(job.id),
                        "queued",
                        format!("Image generation will retry for {}.", job.asset.subject),
                    );
                }
            }
        }
    }

    Ok(())
}

fn emit_visual_asset_event(
    state: &AppState,
    event_type: &str,
    asset: &VisualAsset,
    job_id: Option<i64>,
    status: &str,
    message: String,
) {
    let _ = state.turn_events.send(TurnStreamEvent::visual_asset(
        &asset.story_id,
        event_type,
        &asset.id,
        job_id,
        status,
        message,
    ));
}

async fn next_visual_generation_wakeup(pool: &SqlitePool) -> anyhow::Result<Option<Duration>> {
    let locked_until: Option<String> = sqlx::query_scalar(
        r#"SELECT locked_until FROM visual_generation_jobs
           WHERE status IN ('queued', 'running') AND locked_until != ''
           ORDER BY locked_until ASC
           LIMIT 1"#,
    )
    .fetch_optional(pool)
    .await
    .context("loading next visual generation wakeup")?;
    let Some(locked_until) = locked_until else {
        return Ok(None);
    };
    let parsed = chrono::DateTime::parse_from_rfc3339(&locked_until)
        .map(|value| value.with_timezone(&chrono::Utc))
        .unwrap_or_else(|_| chrono::Utc::now());
    let now = chrono::Utc::now();
    if parsed <= now {
        return Ok(Some(Duration::from_millis(250)));
    }
    let wait = (parsed - now)
        .to_std()
        .unwrap_or_else(|_| Duration::from_millis(250))
        .min(Duration::from_secs(60));
    Ok(Some(wait.max(Duration::from_millis(250))))
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

async fn enqueue_visual_generation_jobs(
    pool: &SqlitePool,
    targets: &[VisualAsset],
    request: &GenerateVisualAssetsRequest,
    config: &ImageGenerationConfig,
) -> anyhow::Result<Vec<QueuedVisualGenerationJob>> {
    let mut queued = Vec::new();
    if targets.is_empty() {
        return Ok(queued);
    }
    let request_payload = serde_json::to_string(request).unwrap_or_else(|_| "{}".to_string());
    for asset in targets {
        sqlx::query(
            r#"UPDATE visual_assets
               SET status = CASE WHEN status = 'running' THEN status ELSE 'queued' END,
                   provider = ?,
                   error = '',
                   updated_at = CURRENT_TIMESTAMP
               WHERE id = ?"#,
        )
        .bind(provider_label(config))
        .bind(&asset.id)
        .execute(pool)
        .await
        .with_context(|| format!("marking visual asset {} queued", asset.id))?;

        let inserted_job_id: Option<i64> = sqlx::query_scalar(
            r#"INSERT OR IGNORE INTO visual_generation_jobs (
                  asset_id, story_id, status, attempts, max_attempts,
                  locked_until, request_payload_json, error, provider
               )
               VALUES (?, ?, 'queued', 0, 3, '', ?, '', ?)
               RETURNING id"#,
        )
        .bind(&asset.id)
        .bind(&asset.story_id)
        .bind(&request_payload)
        .bind(provider_label(config))
        .fetch_optional(pool)
        .await
        .with_context(|| format!("enqueueing visual generation job {}", asset.id))?;
        if let Some(job_id) = inserted_job_id {
            queued.push(QueuedVisualGenerationJob {
                asset: asset.clone(),
                job_id,
            });
        }
    }
    Ok(queued)
}

async fn claim_visual_generation_job(
    pool: &SqlitePool,
    config: &ImageGenerationConfig,
) -> anyhow::Result<Option<VisualGenerationJob>> {
    let now = chrono::Utc::now().to_rfc3339();
    let lock_until = (chrono::Utc::now()
        + chrono::Duration::seconds((config.timeout_seconds.max(30) * 2) as i64))
    .to_rfc3339();
    let row = sqlx::query(
        r#"UPDATE visual_generation_jobs
           SET status = 'running',
               attempts = attempts + 1,
               locked_until = ?,
               started_at = CASE WHEN started_at = '' THEN ? ELSE started_at END,
               updated_at = ?
           WHERE id = (
               SELECT id FROM visual_generation_jobs
               WHERE (
                    status = 'queued'
                    AND (locked_until = '' OR locked_until <= ?)
               ) OR (
                    status = 'running'
                    AND locked_until != ''
                    AND locked_until <= ?
               )
               ORDER BY created_at ASC, id ASC
               LIMIT 1
           )
           RETURNING id, asset_id, story_id, attempts, max_attempts"#,
    )
    .bind(&lock_until)
    .bind(&now)
    .bind(&now)
    .bind(&now)
    .bind(&now)
    .fetch_optional(pool)
    .await
    .context("claiming visual generation job")?;

    let Some(row) = row else {
        return Ok(None);
    };
    let story_id = row_string(&row, "story_id");
    let asset_id = row_string(&row, "asset_id");
    let Some(asset) = load_asset(pool, &story_id, &asset_id).await? else {
        mark_generation_job_terminal(
            pool,
            row.try_get("id").unwrap_or_default(),
            "failed",
            "visual asset was deleted before generation",
        )
        .await?;
        return Ok(None);
    };
    Ok(Some(VisualGenerationJob {
        id: row.try_get("id").unwrap_or_default(),
        asset,
        attempts: row.try_get("attempts").unwrap_or_default(),
        max_attempts: row.try_get("max_attempts").unwrap_or(3),
    }))
}

async fn mark_generation_job_succeeded(pool: &SqlitePool, job_id: i64) -> anyhow::Result<()> {
    mark_generation_job_terminal(pool, job_id, "succeeded", "").await
}

async fn mark_generation_job_terminal(
    pool: &SqlitePool,
    job_id: i64,
    status: &str,
    error: &str,
) -> anyhow::Result<()> {
    let now = chrono::Utc::now().to_rfc3339();
    sqlx::query(
        r#"UPDATE visual_generation_jobs
           SET status = ?,
               locked_until = '',
               error = ?,
               finished_at = ?,
               updated_at = ?
           WHERE id = ?"#,
    )
    .bind(status)
    .bind(compact_error(error))
    .bind(&now)
    .bind(&now)
    .bind(job_id)
    .execute(pool)
    .await
    .with_context(|| format!("marking visual generation job {job_id} {status}"))?;
    Ok(())
}

async fn mark_generation_job_failed_or_retry(
    pool: &SqlitePool,
    job: &VisualGenerationJob,
    error: &str,
    config: &ImageGenerationConfig,
) -> anyhow::Result<()> {
    let message = compact_error(error);
    if job.attempts >= job.max_attempts {
        mark_asset_failed(pool, &job.asset.id, &message, config).await?;
        mark_generation_job_terminal(pool, job.id, "failed", &message).await?;
        return Ok(());
    }

    let now = chrono::Utc::now().to_rfc3339();
    let retry_after =
        (chrono::Utc::now() + chrono::Duration::seconds(retry_delay_seconds(job))).to_rfc3339();
    sqlx::query(
        r#"UPDATE visual_generation_jobs
           SET status = 'queued',
               locked_until = ?,
               error = ?,
               updated_at = ?
           WHERE id = ?"#,
    )
    .bind(&retry_after)
    .bind(&message)
    .bind(&now)
    .bind(job.id)
    .execute(pool)
    .await
    .with_context(|| format!("requeueing visual generation job {}", job.id))?;

    sqlx::query(
        r#"UPDATE visual_assets
           SET status = 'queued', provider = ?, error = ?, updated_at = CURRENT_TIMESTAMP
           WHERE id = ?"#,
    )
    .bind(provider_label(config))
    .bind(message)
    .bind(&job.asset.id)
    .execute(pool)
    .await
    .with_context(|| format!("marking visual asset {} queued after retry", job.asset.id))?;
    Ok(())
}

fn retry_delay_seconds(job: &VisualGenerationJob) -> i64 {
    let exponent = (job.attempts.max(1) - 1).clamp(0, 4) as u32;
    let base = 30_i64 * 2_i64.pow(exponent);
    base + job.id.rem_euclid(11)
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
    generated: &GeneratedAsset,
    config: &ImageGenerationConfig,
) -> anyhow::Result<()> {
    sqlx::query(
        r#"INSERT INTO visual_asset_versions (
              asset_id, story_id, kind, subject, url, file_path, prompt,
              revised_prompt, negative_prompt, provider, turn
           )
           VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"#,
    )
    .bind(&asset.id)
    .bind(&asset.story_id)
    .bind(&asset.kind)
    .bind(&asset.subject)
    .bind(&generated.url)
    .bind(&generated.file_path)
    .bind(&asset.prompt)
    .bind(&generated.revised_prompt)
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
) -> anyhow::Result<GeneratedAsset> {
    if is_openclaw_bridge(config) {
        return generate_one_openclaw_asset(client, state, config, asset).await;
    }

    let output_format = image_output_format(config);
    let prompt = final_prompt(asset, config);
    let mut payload = serde_json::json!({
        "model": config.model,
        "prompt": prompt,
        "size": asset_size(config, asset),
        "output_format": output_format.clone(),
        "n": 1
    });
    if !config.quality.trim().is_empty() {
        payload["quality"] = Value::String(config.quality.trim().to_string());
    }
    if !config.background.trim().is_empty() {
        payload["background"] = Value::String(config.background.trim().to_string());
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

    let mut generated = persist_generated_asset(state, asset, bytes, &output_format).await?;
    generated.revised_prompt = first.revised_prompt.clone().unwrap_or_default();
    Ok(generated)
}

async fn generate_one_openclaw_asset(
    client: &Client,
    state: &AppState,
    config: &ImageGenerationConfig,
    asset: &VisualAsset,
) -> anyhow::Result<GeneratedAsset> {
    let output_format = image_output_format(config);
    let prompt = final_prompt(asset, config);
    let mut payload = serde_json::json!({
        "prompt": prompt,
        "output_format": output_format.clone()
    });
    maybe_set_string(&mut payload, "size", Some(asset_size(config, asset)));
    maybe_set_string(
        &mut payload,
        "resolution",
        asset_generation_value(
            &asset.kind,
            &config.location_resolution,
            &config.character_resolution,
            &config.default_resolution,
        ),
    );
    maybe_set_string(
        &mut payload,
        "aspect_ratio",
        asset_generation_value(
            &asset.kind,
            &config.location_aspect_ratio,
            &config.character_aspect_ratio,
            &config.default_aspect_ratio,
        ),
    );
    maybe_set_string(&mut payload, "background", Some(config.background.clone()));
    if let Some(model) = openclaw_payload_model(config) {
        maybe_set_string(&mut payload, "model", Some(model));
    }
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

    let mut generated = persist_generated_asset(state, asset, bytes, &output_format).await?;
    generated.revised_prompt = response.revised_prompt.unwrap_or_default();
    Ok(generated)
}

async fn persist_generated_asset(
    state: &AppState,
    asset: &VisualAsset,
    bytes: Vec<u8>,
    extension: &str,
) -> anyhow::Result<GeneratedAsset> {
    if !visual_asset_exists(&state.pool, &asset.story_id, &asset.id).await? {
        anyhow::bail!("visual asset was deleted before image persistence");
    }
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
    Ok(GeneratedAsset {
        url,
        file_path: file_path.to_string_lossy().to_string(),
        revised_prompt: String::new(),
    })
}

async fn visual_asset_exists(
    pool: &SqlitePool,
    story_id: &str,
    asset_id: &str,
) -> anyhow::Result<bool> {
    let exists: Option<i64> =
        sqlx::query_scalar("SELECT 1 FROM visual_assets WHERE story_id = ? AND id = ?")
            .bind(story_id)
            .bind(asset_id)
            .fetch_optional(pool)
            .await?;
    Ok(exists.is_some())
}

async fn visual_generation_job_is_cancelled(
    pool: &SqlitePool,
    job_id: i64,
) -> anyhow::Result<bool> {
    let status: Option<String> =
        sqlx::query_scalar("SELECT status FROM visual_generation_jobs WHERE id = ?")
            .bind(job_id)
            .fetch_optional(pool)
            .await?;
    Ok(status.as_deref() == Some("cancelled"))
}

async fn referenced_visual_asset_paths(
    pool: &SqlitePool,
    story_id: &str,
) -> anyhow::Result<HashSet<PathBuf>> {
    let mut paths = HashSet::new();
    let rows = sqlx::query(
        r#"SELECT file_path FROM visual_assets
           WHERE story_id = ? AND file_path != ''
           UNION
           SELECT file_path FROM visual_asset_versions
           WHERE story_id = ? AND file_path != ''"#,
    )
    .bind(story_id)
    .bind(story_id)
    .fetch_all(pool)
    .await
    .with_context(|| format!("loading referenced visual asset paths for {story_id}"))?;

    for row in rows {
        let path = row_string(&row, "file_path");
        if !path.trim().is_empty() {
            paths.insert(PathBuf::from(path));
        }
    }
    Ok(paths)
}

async fn discard_generated_asset(generated: &GeneratedAsset) {
    if generated.file_path.trim().is_empty() {
        return;
    }
    if let Err(err) = fs::remove_file(&generated.file_path).await {
        tracing::warn!(file_path = %generated.file_path, error = %err, "could not remove discarded generated asset");
    }
}

fn final_prompt(asset: &VisualAsset, config: &ImageGenerationConfig) -> String {
    let mut prompt = clean_or(
        &asset.prompt,
        "Create a polished visual asset for this story.",
    );
    if config.append_negative_prompt && !asset.negative_prompt.trim().is_empty() {
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

fn asset_generation_value(
    kind: &str,
    location_value: &str,
    character_value: &str,
    default_value: &str,
) -> Option<String> {
    let value = match kind {
        "location" => clean_or(location_value, default_value),
        "character" => clean_or(character_value, default_value),
        _ => default_value.trim().to_string(),
    };
    (!value.trim().is_empty()).then_some(value)
}

fn maybe_set_string(payload: &mut Value, key: &str, value: Option<String>) {
    if let Some(value) = value {
        let value = value.trim();
        if !value.is_empty() {
            payload[key] = Value::String(value.to_string());
        }
    }
}

fn image_output_format(config: &ImageGenerationConfig) -> String {
    match config.output_format.trim().to_ascii_lowercase().as_str() {
        "jpeg" | "jpg" => "jpeg".to_string(),
        "webp" => "webp".to_string(),
        _ => "png".to_string(),
    }
}

fn openclaw_payload_model(config: &ImageGenerationConfig) -> Option<String> {
    let model = config.model.trim();
    if !model.is_empty() {
        Some(model.to_string())
    } else {
        None
    }
}

fn image_generation_config(state: &AppState) -> anyhow::Result<ImageGenerationConfig> {
    let file_config = read_gateway_config(&state.paths.config_path)?;
    let ai_config = file_config.ai.as_ref();
    let litellm = ai_config.and_then(|ai| ai.litellm.as_ref());
    let image_generation = ai_config.and_then(|ai| ai.image_generation.as_ref());
    let config_base_url = litellm
        .and_then(|provider| provider.base_url.clone())
        .unwrap_or_default();
    let config_api_key = litellm
        .and_then(|provider| provider.api_key.clone())
        .map(|value| expand_env_refs(&value))
        .unwrap_or_default();
    let base_url = first_env(&["ONEDAY_IMAGEGEN_BASE_URL", "ONEDAY_IMAGE_BASE_URL"])
        .or_else(|| image_config_string(image_generation, |config| &config.base_url))
        .or_else(|| non_empty(config_base_url))
        .unwrap_or_default();
    let api_key = first_env(&[
        "ONEDAY_IMAGEGEN_API_KEY",
        "ONEDAY_IMAGE_API_KEY",
        "ONEDAY_LITELLM_API_KEY",
        "OPENAI_API_KEY",
    ])
    .or_else(|| {
        image_config_string(image_generation, |config| &config.api_key)
            .map(|value| expand_env_refs(&value))
            .and_then(non_empty)
    })
    .or_else(|| non_empty(config_api_key))
    .unwrap_or_default();
    let model = first_env(&["ONEDAY_IMAGEGEN_MODEL", "ONEDAY_IMAGE_MODEL"])
        .or_else(|| image_config_string(image_generation, |config| &config.model))
        .unwrap_or_default();

    Ok(ImageGenerationConfig {
        base_url,
        api_key,
        model,
        provider: first_env(&["ONEDAY_IMAGEGEN_PROVIDER", "ONEDAY_IMAGE_PROVIDER"])
            .or_else(|| image_config_string(image_generation, |config| &config.provider))
            .unwrap_or_else(|| "openclaw-bridge".to_string()),
        openclaw_bridge_url: first_env(&[
            "ONEDAY_IMAGEGEN_OPENCLAW_URL",
            "ONEDAY_OPENCLAW_IMAGEGEN_URL",
        ])
        .or_else(|| image_config_string(image_generation, |config| &config.openclaw_bridge_url))
        .unwrap_or_else(|| "http://openclaw-imagegen:8099/generate".to_string()),
        default_size: first_env(&["ONEDAY_IMAGEGEN_SIZE", "ONEDAY_IMAGE_SIZE"])
            .or_else(|| image_config_string(image_generation, |config| &config.default_size))
            .unwrap_or_else(|| "1024x1024".to_string()),
        location_size: first_env(&["ONEDAY_IMAGEGEN_LOCATION_SIZE"])
            .or_else(|| image_config_string(image_generation, |config| &config.location_size))
            .unwrap_or_else(|| "1536x1024".to_string()),
        character_size: first_env(&["ONEDAY_IMAGEGEN_CHARACTER_SIZE"])
            .or_else(|| image_config_string(image_generation, |config| &config.character_size))
            .unwrap_or_else(|| "1024x1024".to_string()),
        default_resolution: first_env(&["ONEDAY_IMAGEGEN_RESOLUTION"])
            .or_else(|| image_config_string(image_generation, |config| &config.default_resolution))
            .unwrap_or_default(),
        location_resolution: first_env(&["ONEDAY_IMAGEGEN_LOCATION_RESOLUTION"])
            .or_else(|| image_config_string(image_generation, |config| &config.location_resolution))
            .unwrap_or_default(),
        character_resolution: first_env(&["ONEDAY_IMAGEGEN_CHARACTER_RESOLUTION"])
            .or_else(|| {
                image_config_string(image_generation, |config| &config.character_resolution)
            })
            .unwrap_or_default(),
        default_aspect_ratio: first_env(&["ONEDAY_IMAGEGEN_ASPECT_RATIO"])
            .or_else(|| {
                image_config_string(image_generation, |config| &config.default_aspect_ratio)
            })
            .unwrap_or_default(),
        location_aspect_ratio: first_env(&["ONEDAY_IMAGEGEN_LOCATION_ASPECT_RATIO"])
            .or_else(|| {
                image_config_string(image_generation, |config| &config.location_aspect_ratio)
            })
            .unwrap_or_default(),
        character_aspect_ratio: first_env(&["ONEDAY_IMAGEGEN_CHARACTER_ASPECT_RATIO"])
            .or_else(|| {
                image_config_string(image_generation, |config| &config.character_aspect_ratio)
            })
            .unwrap_or_default(),
        quality: first_env(&["ONEDAY_IMAGEGEN_QUALITY"])
            .or_else(|| image_config_string(image_generation, |config| &config.quality))
            .unwrap_or_default(),
        output_format: first_env(&["ONEDAY_IMAGEGEN_OUTPUT_FORMAT"])
            .or_else(|| image_config_string(image_generation, |config| &config.output_format))
            .unwrap_or_else(|| "png".to_string()),
        background: first_env(&["ONEDAY_IMAGEGEN_BACKGROUND"])
            .or_else(|| image_config_string(image_generation, |config| &config.background))
            .unwrap_or_default(),
        timeout_seconds: first_env(&["ONEDAY_IMAGEGEN_TIMEOUT_SECONDS"])
            .and_then(|value| value.parse::<u64>().ok())
            .or_else(|| image_generation.and_then(|config| config.timeout_seconds))
            .unwrap_or(180),
        auto_generate: first_env(&["ONEDAY_IMAGEGEN_AUTOGENERATE"])
            .map(|value| parse_bool(&value))
            .or_else(|| image_generation.and_then(|config| config.auto_generate))
            .unwrap_or(false),
        append_negative_prompt: first_env(&["ONEDAY_IMAGEGEN_APPEND_NEGATIVE_PROMPT"])
            .map(|value| parse_bool(&value))
            .or_else(|| image_generation.and_then(|config| config.append_negative_prompt))
            .unwrap_or(true),
    })
}

fn image_config_string(
    config: Option<&GatewayImageGenerationConfig>,
    pick: impl Fn(&GatewayImageGenerationConfig) -> &Option<String>,
) -> Option<String> {
    config
        .and_then(|config| pick(config).clone())
        .and_then(non_empty)
}

fn is_openclaw_bridge(config: &ImageGenerationConfig) -> bool {
    matches!(
        config.provider.trim().to_ascii_lowercase().as_str(),
        "openclaw" | "openclaw-bridge" | "codex-oauth"
    )
}

fn image_generation_available(config: &ImageGenerationConfig) -> bool {
    if config.provider.trim().is_empty() || config.model.trim().is_empty() {
        return false;
    }
    if is_openclaw_bridge(config) {
        return !config.openclaw_bridge_url.trim().is_empty();
    }
    !config.base_url.trim().is_empty() && !config.api_key.trim().is_empty()
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
                  updated_at = CASE
                    WHEN visual_assets.entity_id = ''
                      OR visual_assets.prompt = ''
                      OR visual_assets.negative_prompt = ''
                      OR visual_assets.turn != excluded.turn
                    THEN CURRENT_TIMESTAMP
                    ELSE visual_assets.updated_at
                  END,
                  entity_id = CASE WHEN visual_assets.entity_id = '' THEN excluded.entity_id ELSE visual_assets.entity_id END,
                  prompt = CASE WHEN visual_assets.prompt = '' THEN excluded.prompt ELSE visual_assets.prompt END,
                  negative_prompt = CASE WHEN visual_assets.negative_prompt = '' THEN excluded.negative_prompt ELSE visual_assets.negative_prompt END,
                  turn = CASE WHEN visual_assets.turn != excluded.turn THEN excluded.turn ELSE visual_assets.turn END"#,
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

    Ok(rows.into_iter().map(visual_asset_from_row).collect())
}

async fn list_visual_generation_jobs(
    pool: &SqlitePool,
    story_id: &str,
) -> anyhow::Result<Vec<VisualGenerationJobView>> {
    let rows = sqlx::query(
        r#"SELECT id, asset_id, story_id, status, attempts, max_attempts,
                  locked_until, error, provider, started_at, finished_at,
                  CAST(created_at AS TEXT) AS created_at,
                  CAST(updated_at AS TEXT) AS updated_at
           FROM visual_generation_jobs
           WHERE story_id = ?
           ORDER BY
             CASE status WHEN 'running' THEN 0 WHEN 'queued' THEN 1 WHEN 'failed' THEN 2 ELSE 3 END,
             updated_at DESC,
             id DESC
           LIMIT 25"#,
    )
    .bind(story_id)
    .fetch_all(pool)
    .await?;

    Ok(rows
        .into_iter()
        .map(|row| VisualGenerationJobView {
            id: row.try_get("id").unwrap_or_default(),
            asset_id: row_string(&row, "asset_id"),
            story_id: row_string(&row, "story_id"),
            status: row_string(&row, "status"),
            attempts: row.try_get("attempts").unwrap_or_default(),
            max_attempts: row.try_get("max_attempts").unwrap_or_default(),
            locked_until: row_string(&row, "locked_until"),
            error: row_string(&row, "error"),
            provider: row_string(&row, "provider"),
            started_at: row_string(&row, "started_at"),
            finished_at: row_string(&row, "finished_at"),
            created_at: row_string(&row, "created_at"),
            updated_at: row_string(&row, "updated_at"),
        })
        .collect())
}

async fn load_asset(
    pool: &SqlitePool,
    story_id: &str,
    asset_id: &str,
) -> anyhow::Result<Option<VisualAsset>> {
    let row = sqlx::query(
        r#"SELECT id, story_id, kind, subject, entity_id, prompt, negative_prompt,
                  status, url, provider, source, error, turn,
                  CAST(updated_at AS TEXT) AS updated_at
           FROM visual_assets
           WHERE story_id = ? AND id = ?"#,
    )
    .bind(story_id)
    .bind(asset_id)
    .fetch_optional(pool)
    .await?;
    Ok(row.map(visual_asset_from_row))
}

fn visual_asset_from_row(row: sqlx::sqlite::SqliteRow) -> VisualAsset {
    VisualAsset {
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
    }
}

async fn ensure_asset_belongs_to_story(
    pool: &SqlitePool,
    story_id: &str,
    asset_id: &str,
) -> anyhow::Result<()> {
    let exists: Option<i64> =
        sqlx::query_scalar("SELECT 1 FROM visual_assets WHERE story_id = ? AND id = ?")
            .bind(story_id)
            .bind(asset_id)
            .fetch_optional(pool)
            .await?;
    exists
        .map(|_| ())
        .ok_or_else(|| anyhow!("visual asset not found"))
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
    use sqlx::sqlite::SqlitePoolOptions;

    fn test_config() -> ImageGenerationConfig {
        ImageGenerationConfig {
            base_url: "http://example.test/v1".to_string(),
            api_key: "key".to_string(),
            model: "test-image-model".to_string(),
            provider: "test".to_string(),
            openclaw_bridge_url: "http://openclaw-imagegen:8099/generate".to_string(),
            default_size: "1024x1024".to_string(),
            location_size: "1536x1024".to_string(),
            character_size: "768x768".to_string(),
            default_resolution: String::new(),
            location_resolution: String::new(),
            character_resolution: String::new(),
            default_aspect_ratio: String::new(),
            location_aspect_ratio: String::new(),
            character_aspect_ratio: String::new(),
            quality: String::new(),
            output_format: "png".to_string(),
            background: String::new(),
            timeout_seconds: 10,
            auto_generate: true,
            append_negative_prompt: true,
        }
    }

    async fn visual_job_pool() -> SqlitePool {
        let pool = SqlitePoolOptions::new()
            .max_connections(1)
            .connect("sqlite::memory:")
            .await
            .expect("memory sqlite pool");
        sqlx::query(
            r#"CREATE TABLE stories (
                id TEXT PRIMARY KEY,
                name TEXT NOT NULL DEFAULT '',
                description TEXT NOT NULL DEFAULT '',
                genre TEXT NOT NULL DEFAULT '',
                tone TEXT NOT NULL DEFAULT '',
                language TEXT NOT NULL DEFAULT 'en',
                is_archived INTEGER NOT NULL DEFAULT 0,
                revision INTEGER NOT NULL DEFAULT 0,
                updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
            )"#,
        )
        .execute(&pool)
        .await
        .expect("create stories");
        for statement in [
            r#"CREATE TABLE characters (
                id TEXT PRIMARY KEY,
                story_id TEXT NOT NULL,
                name TEXT NOT NULL DEFAULT '',
                background TEXT NOT NULL DEFAULT '',
                stats_json TEXT NOT NULL DEFAULT '{}',
                traits_json TEXT NOT NULL DEFAULT '[]',
                skills_json TEXT NOT NULL DEFAULT '[]',
                inventory_json TEXT NOT NULL DEFAULT '[]',
                known_recipes_json TEXT NOT NULL DEFAULT '[]',
                updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
            )"#,
            r#"CREATE TABLE world_state (
                id TEXT PRIMARY KEY,
                story_id TEXT NOT NULL,
                current_location TEXT NOT NULL DEFAULT '',
                known_locations_json TEXT NOT NULL DEFAULT '[]',
                global_events_json TEXT NOT NULL DEFAULT '[]',
                faction_standings_json TEXT NOT NULL DEFAULT '{}',
                story_hooks_json TEXT NOT NULL DEFAULT '[]',
                world_reactions_json TEXT NOT NULL DEFAULT '[]',
                investigation_board_json TEXT NOT NULL DEFAULT '{}',
                project_clocks_json TEXT NOT NULL DEFAULT '{}',
                player_guidance_json TEXT NOT NULL DEFAULT '[]',
                fronts_json TEXT NOT NULL DEFAULT '[]',
                character_timeline_json TEXT NOT NULL DEFAULT '{}',
                scene_contract_json TEXT NOT NULL DEFAULT '{}',
                current_chapter INTEGER NOT NULL DEFAULT 1,
                current_turn INTEGER NOT NULL DEFAULT 3,
                updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
            )"#,
            r#"CREATE TABLE sessions (
                id TEXT PRIMARY KEY,
                story_id TEXT NOT NULL,
                started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                ended_at DATETIME,
                summary TEXT NOT NULL DEFAULT ''
            )"#,
            r#"CREATE TABLE chat_messages (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                session_id TEXT NOT NULL,
                story_id TEXT NOT NULL,
                turn INTEGER NOT NULL DEFAULT 0,
                role TEXT NOT NULL DEFAULT '',
                content TEXT NOT NULL DEFAULT '',
                message_type TEXT NOT NULL DEFAULT 'narrative',
                metadata_json TEXT NOT NULL DEFAULT '{}',
                created_at DATETIME DEFAULT CURRENT_TIMESTAMP
            )"#,
            r#"CREATE TABLE chapters (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                story_id TEXT NOT NULL,
                chapter_number INTEGER NOT NULL DEFAULT 1,
                title TEXT NOT NULL DEFAULT '',
                summary TEXT NOT NULL DEFAULT '',
                start_turn INTEGER NOT NULL DEFAULT 0,
                end_turn INTEGER NOT NULL DEFAULT 0,
                created_at DATETIME DEFAULT CURRENT_TIMESTAMP
            )"#,
            r#"CREATE TABLE achievements (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                story_id TEXT NOT NULL,
                name TEXT NOT NULL DEFAULT '',
                description TEXT NOT NULL DEFAULT '',
                category TEXT NOT NULL DEFAULT '',
                rarity TEXT NOT NULL DEFAULT '',
                context TEXT NOT NULL DEFAULT '',
                earned_at DATETIME DEFAULT CURRENT_TIMESTAMP
            )"#,
            r#"CREATE TABLE npcs (
                id TEXT PRIMARY KEY,
                story_id TEXT NOT NULL,
                name TEXT NOT NULL DEFAULT '',
                role TEXT NOT NULL DEFAULT '',
                appearance TEXT NOT NULL DEFAULT '',
                personality_json TEXT NOT NULL DEFAULT '{}',
                relationship_json TEXT NOT NULL DEFAULT '{}',
                disposition INTEGER NOT NULL DEFAULT 0,
                is_alive INTEGER NOT NULL DEFAULT 1,
                first_appeared_turn INTEGER NOT NULL DEFAULT 0,
                last_seen_turn INTEGER NOT NULL DEFAULT 0,
                can_help INTEGER NOT NULL DEFAULT 0,
                updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
            )"#,
            r#"CREATE TABLE saves (
                id TEXT PRIMARY KEY,
                story_id TEXT NOT NULL,
                name TEXT NOT NULL DEFAULT '',
                turn INTEGER NOT NULL DEFAULT 0,
                chapter INTEGER NOT NULL DEFAULT 0,
                location TEXT NOT NULL DEFAULT '',
                session_id TEXT NOT NULL DEFAULT '',
                metadata_json TEXT NOT NULL DEFAULT '{}',
                created_at DATETIME DEFAULT CURRENT_TIMESTAMP
            )"#,
            r#"CREATE TABLE story_visual_profiles (
                story_id TEXT PRIMARY KEY,
                world_style_prompt TEXT NOT NULL DEFAULT '',
                character_style_prompt TEXT NOT NULL DEFAULT '',
                negative_prompt TEXT NOT NULL DEFAULT '',
                palette TEXT NOT NULL DEFAULT '',
                updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
            )"#,
        ] {
            sqlx::query(statement)
                .execute(&pool)
                .await
                .expect("create snapshot fixture table");
        }
        sqlx::query(
            r#"CREATE TABLE visual_assets (
                id TEXT PRIMARY KEY,
                story_id TEXT NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
                kind TEXT NOT NULL,
                subject TEXT NOT NULL,
                entity_id TEXT NOT NULL DEFAULT '',
                prompt TEXT NOT NULL DEFAULT '',
                negative_prompt TEXT NOT NULL DEFAULT '',
                status TEXT NOT NULL DEFAULT 'pending',
                url TEXT NOT NULL DEFAULT '',
                file_path TEXT NOT NULL DEFAULT '',
                provider TEXT NOT NULL DEFAULT '',
                source TEXT NOT NULL DEFAULT '',
                error TEXT NOT NULL DEFAULT '',
                turn INTEGER NOT NULL DEFAULT 0,
                created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                UNIQUE(story_id, kind, subject)
            )"#,
        )
        .execute(&pool)
        .await
        .expect("create visual assets");
        sqlx::query("INSERT INTO stories (id, name, genre, tone) VALUES ('story', 'Story', 'test', 'focused')")
            .execute(&pool)
            .await
            .expect("insert story");
        sqlx::query(
            "INSERT INTO characters (id, story_id, name) VALUES ('character', 'story', 'Tester')",
        )
        .execute(&pool)
        .await
        .expect("insert character");
        sqlx::query("INSERT INTO world_state (id, story_id, current_location) VALUES ('world', 'story', 'Station')")
            .execute(&pool)
            .await
            .expect("insert world");
        sqlx::query("INSERT INTO sessions (id, story_id) VALUES ('session', 'story')")
            .execute(&pool)
            .await
            .expect("insert session");
        sqlx::query(
            r#"INSERT INTO visual_assets (
                id, story_id, kind, subject, entity_id, prompt, negative_prompt,
                status, provider, source, turn
            ) VALUES (
                'asset-location', 'story', 'location', 'Station', 'world',
                'wide station', 'no text', 'pending', '', 'auto-profile', 3
            )"#,
        )
        .execute(&pool)
        .await
        .expect("insert asset");
        ensure_visual_asset_version_schema(&pool)
            .await
            .expect("visual schemas");
        pool
    }

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
            model: "test-image-model".to_string(),
            provider: "test".to_string(),
            openclaw_bridge_url: "http://openclaw-imagegen:8099/generate".to_string(),
            default_size: "1024x1024".to_string(),
            location_size: "1536x1024".to_string(),
            character_size: "768x768".to_string(),
            default_resolution: String::new(),
            location_resolution: String::new(),
            character_resolution: String::new(),
            default_aspect_ratio: String::new(),
            location_aspect_ratio: String::new(),
            character_aspect_ratio: String::new(),
            quality: String::new(),
            output_format: "png".to_string(),
            background: String::new(),
            timeout_seconds: 10,
            auto_generate: true,
            append_negative_prompt: true,
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
        let config = ImageGenerationConfig {
            base_url: "http://example.test/v1".to_string(),
            api_key: "key".to_string(),
            model: "test-image-model".to_string(),
            provider: "test".to_string(),
            openclaw_bridge_url: "http://openclaw-imagegen:8099/generate".to_string(),
            default_size: "1024x1024".to_string(),
            location_size: "1536x1024".to_string(),
            character_size: "1024x1024".to_string(),
            default_resolution: String::new(),
            location_resolution: String::new(),
            character_resolution: String::new(),
            default_aspect_ratio: String::new(),
            location_aspect_ratio: String::new(),
            character_aspect_ratio: String::new(),
            quality: String::new(),
            output_format: "png".to_string(),
            background: String::new(),
            timeout_seconds: 10,
            auto_generate: true,
            append_negative_prompt: true,
        };
        assert!(final_prompt(&asset, &config).contains("portrait"));
        assert!(final_prompt(&asset, &config).contains("Avoid: no text"));
    }

    #[test]
    fn openclaw_bridge_does_not_require_api_key() {
        let config = ImageGenerationConfig {
            base_url: String::new(),
            api_key: String::new(),
            model: "test-image-model".to_string(),
            provider: "openclaw-bridge".to_string(),
            openclaw_bridge_url: "http://openclaw-imagegen:8099/generate".to_string(),
            default_size: "1024x1024".to_string(),
            location_size: "1536x1024".to_string(),
            character_size: "1024x1024".to_string(),
            default_resolution: String::new(),
            location_resolution: String::new(),
            character_resolution: String::new(),
            default_aspect_ratio: String::new(),
            location_aspect_ratio: String::new(),
            character_aspect_ratio: String::new(),
            quality: String::new(),
            output_format: "png".to_string(),
            background: String::new(),
            timeout_seconds: 10,
            auto_generate: true,
            append_negative_prompt: true,
        };
        assert!(is_openclaw_bridge(&config));
        assert!(image_generation_available(&config));
    }

    #[test]
    fn openclaw_bridge_payload_accepts_simple_model_names() {
        let config = ImageGenerationConfig {
            base_url: String::new(),
            api_key: String::new(),
            model: "gpt-image-2".to_string(),
            provider: "openclaw-bridge".to_string(),
            openclaw_bridge_url: "http://openclaw-imagegen:8099/generate".to_string(),
            default_size: "1024x1024".to_string(),
            location_size: "1536x1024".to_string(),
            character_size: "1024x1024".to_string(),
            default_resolution: String::new(),
            location_resolution: String::new(),
            character_resolution: String::new(),
            default_aspect_ratio: String::new(),
            location_aspect_ratio: String::new(),
            character_aspect_ratio: String::new(),
            quality: String::new(),
            output_format: "png".to_string(),
            background: String::new(),
            timeout_seconds: 10,
            auto_generate: true,
            append_negative_prompt: true,
        };

        assert_eq!(
            openclaw_payload_model(&config).as_deref(),
            Some("gpt-image-2")
        );
    }

    #[test]
    fn openai_compatible_provider_requires_api_key() {
        let config = ImageGenerationConfig {
            base_url: "http://example.test/v1".to_string(),
            api_key: String::new(),
            model: "test-image-model".to_string(),
            provider: "openai-compatible".to_string(),
            openclaw_bridge_url: "http://openclaw-imagegen:8099/generate".to_string(),
            default_size: "1024x1024".to_string(),
            location_size: "1536x1024".to_string(),
            character_size: "1024x1024".to_string(),
            default_resolution: String::new(),
            location_resolution: String::new(),
            character_resolution: String::new(),
            default_aspect_ratio: String::new(),
            location_aspect_ratio: String::new(),
            character_aspect_ratio: String::new(),
            quality: String::new(),
            output_format: "png".to_string(),
            background: String::new(),
            timeout_seconds: 10,
            auto_generate: true,
            append_negative_prompt: true,
        };
        assert!(!is_openclaw_bridge(&config));
        assert!(!image_generation_available(&config));
    }

    #[test]
    fn openai_compatible_provider_requires_base_url() {
        let config = ImageGenerationConfig {
            base_url: String::new(),
            api_key: "key".to_string(),
            model: "test-image-model".to_string(),
            provider: "openai-compatible".to_string(),
            openclaw_bridge_url: "http://openclaw-imagegen:8099/generate".to_string(),
            default_size: "1024x1024".to_string(),
            location_size: "1536x1024".to_string(),
            character_size: "1024x1024".to_string(),
            default_resolution: String::new(),
            location_resolution: String::new(),
            character_resolution: String::new(),
            default_aspect_ratio: String::new(),
            location_aspect_ratio: String::new(),
            character_aspect_ratio: String::new(),
            quality: String::new(),
            output_format: "png".to_string(),
            background: String::new(),
            timeout_seconds: 10,
            auto_generate: true,
            append_negative_prompt: true,
        };

        assert!(!image_generation_available(&config));
    }

    #[test]
    fn image_generation_requires_explicit_model() {
        let config = ImageGenerationConfig {
            base_url: String::new(),
            api_key: String::new(),
            model: String::new(),
            provider: "openclaw-bridge".to_string(),
            openclaw_bridge_url: "http://openclaw-imagegen:8099/generate".to_string(),
            default_size: "1024x1024".to_string(),
            location_size: "1536x1024".to_string(),
            character_size: "1024x1024".to_string(),
            default_resolution: String::new(),
            location_resolution: String::new(),
            character_resolution: String::new(),
            default_aspect_ratio: String::new(),
            location_aspect_ratio: String::new(),
            character_aspect_ratio: String::new(),
            quality: String::new(),
            output_format: "png".to_string(),
            background: String::new(),
            timeout_seconds: 10,
            auto_generate: true,
            append_negative_prompt: true,
        };
        assert!(!image_generation_available(&config));
    }

    #[tokio::test]
    async fn enqueue_visual_generation_jobs_marks_assets_queued_and_deduplicates() {
        let pool = visual_job_pool().await;
        let config = test_config();
        let asset = load_asset(&pool, "story", "asset-location")
            .await
            .expect("load asset")
            .expect("asset");
        let request = GenerateVisualAssetsRequest {
            asset_ids: vec!["asset-location".to_string()],
            force: false,
            limit: Some(1),
        };

        let first_queue =
            enqueue_visual_generation_jobs(&pool, std::slice::from_ref(&asset), &request, &config)
                .await
                .expect("first enqueue");
        let second_queue =
            enqueue_visual_generation_jobs(&pool, std::slice::from_ref(&asset), &request, &config)
                .await
                .expect("second enqueue");

        assert_eq!(first_queue.len(), 1);
        assert_eq!(first_queue[0].asset.id, "asset-location");
        assert!(first_queue[0].job_id > 0);
        assert!(second_queue.is_empty());

        let status: String = sqlx::query_scalar("SELECT status FROM visual_assets WHERE id = ?")
            .bind("asset-location")
            .fetch_one(&pool)
            .await
            .expect("asset status");
        let jobs: i64 = sqlx::query_scalar(
            "SELECT COUNT(*) FROM visual_generation_jobs WHERE asset_id = ? AND status = 'queued'",
        )
        .bind("asset-location")
        .fetch_one(&pool)
        .await
        .expect("job count");

        assert_eq!(status, "queued");
        assert_eq!(jobs, 1);

        let job_views = list_visual_generation_jobs(&pool, "story")
            .await
            .expect("list jobs");
        assert_eq!(job_views.len(), 1);
        assert_eq!(job_views[0].asset_id, "asset-location");
        assert_eq!(job_views[0].status, "queued");
        assert_eq!(job_views[0].attempts, 0);
        assert_eq!(job_views[0].max_attempts, 3);
    }

    #[tokio::test]
    async fn ensure_asset_rows_does_not_touch_updated_at_when_nothing_changes() {
        let pool = visual_job_pool().await;
        sqlx::query("UPDATE visual_assets SET updated_at = 'fixed', turn = 3 WHERE id = ?")
            .bind("asset-location")
            .execute(&pool)
            .await
            .expect("pin updated_at");
        let spec = VisualSpec {
            kind: "location".to_string(),
            subject: "Station".to_string(),
            entity_id: "world".to_string(),
            prompt: "wide station".to_string(),
            negative_prompt: "no text".to_string(),
            turn: 3,
        };

        ensure_asset_rows(&pool, "story", &[spec])
            .await
            .expect("ensure asset rows");

        let updated_at: String =
            sqlx::query_scalar("SELECT updated_at FROM visual_assets WHERE id = ?")
                .bind("asset-location")
                .fetch_one(&pool)
                .await
                .expect("updated_at");
        assert_eq!(updated_at, "fixed");
    }

    #[tokio::test]
    async fn claim_visual_generation_job_recovers_stale_running_jobs() {
        let pool = visual_job_pool().await;
        let config = test_config();
        let stale = (chrono::Utc::now() - chrono::Duration::seconds(5)).to_rfc3339();
        sqlx::query(
            r#"INSERT INTO visual_generation_jobs (
                asset_id, story_id, status, attempts, max_attempts, locked_until, provider
            ) VALUES (?, ?, 'running', 1, 3, ?, ?)"#,
        )
        .bind("asset-location")
        .bind("story")
        .bind(stale)
        .bind(provider_label(&config))
        .execute(&pool)
        .await
        .expect("insert stale job");

        let job = claim_visual_generation_job(&pool, &config)
            .await
            .expect("claim job")
            .expect("job");

        assert_eq!(job.asset.id, "asset-location");
        assert_eq!(job.attempts, 2);
        let status: String =
            sqlx::query_scalar("SELECT status FROM visual_generation_jobs WHERE id = ?")
                .bind(job.id)
                .fetch_one(&pool)
                .await
                .expect("job status");
        assert_eq!(status, "running");
    }

    #[test]
    fn retry_delay_uses_exponential_backoff_with_bounded_jitter() {
        let asset = VisualAsset {
            id: "asset-location".to_string(),
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
        let first = VisualGenerationJob {
            id: 5,
            asset: asset.clone(),
            attempts: 1,
            max_attempts: 3,
        };
        let second = VisualGenerationJob {
            id: 5,
            asset,
            attempts: 2,
            max_attempts: 3,
        };

        assert_eq!(retry_delay_seconds(&first), 35);
        assert_eq!(retry_delay_seconds(&second), 65);
    }

    #[tokio::test]
    async fn cancel_visual_generation_job_marks_job_cancelled_and_asset_pending() {
        let pool = visual_job_pool().await;
        let job_id: i64 = sqlx::query_scalar(
            r#"INSERT INTO visual_generation_jobs (
                asset_id, story_id, status, attempts, max_attempts, locked_until, provider
            ) VALUES (?, ?, 'running', 1, 3, '', ?)
            RETURNING id"#,
        )
        .bind("asset-location")
        .bind("story")
        .bind("test")
        .fetch_one(&pool)
        .await
        .expect("insert job");
        sqlx::query("UPDATE visual_assets SET status = 'running' WHERE id = ?")
            .bind("asset-location")
            .execute(&pool)
            .await
            .expect("mark asset running");

        let response = cancel_visual_generation_job(&pool, "story", job_id)
            .await
            .expect("cancel job");
        let job_status: String =
            sqlx::query_scalar("SELECT status FROM visual_generation_jobs WHERE id = ?")
                .bind(job_id)
                .fetch_one(&pool)
                .await
                .expect("job status");
        let asset = response
            .assets
            .iter()
            .find(|asset| asset.id == "asset-location")
            .expect("asset");

        assert_eq!(job_status, "cancelled");
        assert_eq!(asset.status, "pending");
        assert_eq!(asset.error, "Generation cancelled.");
    }

    #[tokio::test]
    async fn cleanup_visual_asset_files_removes_only_unreferenced_story_files() {
        let pool = visual_job_pool().await;
        let root = std::env::temp_dir().join(format!("oneday-assets-{}", uuid::Uuid::new_v4()));
        let story_dir = root.join(slug("story"));
        std::fs::create_dir_all(&story_dir).expect("create story dir");
        let kept = story_dir.join("kept.png");
        let stale = story_dir.join("stale.png");
        std::fs::write(&kept, b"kept").expect("write kept");
        std::fs::write(&stale, b"stale").expect("write stale");
        sqlx::query("UPDATE visual_assets SET file_path = ?, url = ? WHERE id = ?")
            .bind(kept.to_string_lossy().to_string())
            .bind("/generated/assets/story/kept.png")
            .bind("asset-location")
            .execute(&pool)
            .await
            .expect("set referenced file");

        let dry_run = cleanup_visual_asset_files(
            &pool,
            "story",
            &root,
            VisualAssetCleanupRequest { dry_run: true },
        )
        .await
        .expect("dry cleanup");
        assert_eq!(dry_run.deleted_files.len(), 1);
        assert!(stale.exists());

        let cleanup = cleanup_visual_asset_files(
            &pool,
            "story",
            &root,
            VisualAssetCleanupRequest { dry_run: false },
        )
        .await
        .expect("cleanup");
        assert_eq!(cleanup.deleted_files.len(), 1);
        assert!(kept.exists());
        assert!(!stale.exists());
        std::fs::remove_dir_all(&root).ok();
    }

    #[tokio::test]
    async fn cancel_story_visual_jobs_marks_active_jobs_cancelled() {
        let pool = visual_job_pool().await;
        sqlx::query(
            r#"INSERT INTO visual_generation_jobs (
                asset_id, story_id, status, attempts, max_attempts, locked_until, provider
            ) VALUES (?, ?, 'queued', 0, 3, '', ?)"#,
        )
        .bind("asset-location")
        .bind("story")
        .bind("test")
        .execute(&pool)
        .await
        .expect("insert job");

        let cancelled = cancel_story_visual_jobs(&pool, "story")
            .await
            .expect("cancel jobs");
        let status: String =
            sqlx::query_scalar("SELECT status FROM visual_generation_jobs WHERE story_id = ?")
                .bind("story")
                .fetch_one(&pool)
                .await
                .expect("job status");

        assert_eq!(cancelled, 1);
        assert_eq!(status, "cancelled");
    }
}
