use crate::{engine, error::PublicError, AppState};
use anyhow::anyhow;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use sqlx::{Row, Sqlite, SqlitePool, Transaction};
use std::{collections::BTreeMap, sync::Arc, time::Duration};
use uuid::Uuid;

const CODEC_VERSION: &str = "protected-v1";

#[derive(Clone, Debug, Deserialize)]
pub struct CreateJobRequest {
    pub scope_kind: String,
    #[serde(default)]
    pub scope_id: String,
    #[serde(default)]
    pub message_ids: Vec<i64>,
    pub target_language: String,
    #[serde(default)]
    pub source_language: String,
    pub engine: String,
    #[serde(default)]
    pub provider: String,
    #[serde(default)]
    pub model: String,
    #[serde(default = "default_style")]
    pub style: String,
}

fn default_style() -> String {
    "faithful".into()
}

#[derive(Clone, Debug, Serialize)]
pub struct JobView {
    pub id: String,
    pub story_id: String,
    pub branch_id: String,
    pub scope_kind: String,
    pub scope_id: String,
    pub source_language: String,
    pub target_language: String,
    pub engine: String,
    pub provider: String,
    pub model: String,
    pub style: String,
    pub status: String,
    pub total_items: i64,
    pub completed_items: i64,
    pub failed_items: i64,
    pub cached_items: i64,
    pub total_characters: i64,
    pub processed_characters: i64,
    pub error_code: String,
    pub error_summary: String,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Clone, Debug, Serialize)]
pub struct EstimateView {
    pub total_items: usize,
    pub total_characters: usize,
    pub cache_hits: usize,
}

#[derive(Clone, Debug, Serialize)]
pub struct BrowserItemView {
    pub id: String,
    pub content_kind: String,
    pub content_id: String,
    pub source_text: String,
    pub source_language: String,
    pub target_language: String,
}

#[derive(Debug, Deserialize)]
pub struct CompleteBrowserItemRequest {
    pub translated_text: String,
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct GlossaryEntry {
    pub id: String,
    pub source_language: String,
    pub target_language: String,
    pub source_term: String,
    pub target_term: String,
    pub mode: String,
}

#[derive(Debug, Deserialize)]
pub struct GlossaryEntryRequest {
    #[serde(default)]
    pub source_language: String,
    #[serde(default)]
    pub target_language: String,
    pub source_term: String,
    #[serde(default)]
    pub target_term: String,
    #[serde(default = "default_glossary_mode")]
    pub mode: String,
}

fn default_glossary_mode() -> String {
    "translate".into()
}

#[derive(Clone)]
struct SourceItem {
    kind: String,
    id: String,
    text: String,
}

pub async fn estimate(
    pool: &SqlitePool,
    story_id: &str,
    request: &CreateJobRequest,
) -> anyhow::Result<EstimateView> {
    validate_request(request)?;
    let (branch, source) = story_scope(pool, story_id).await?;
    let items = source_items(pool, story_id, &branch, request).await?;
    let mut hits = 0;
    for item in &items {
        let hash = content_hash(&item.text);
        if cached_translation_id(pool, story_id, &branch, item, &hash, &source, request)
            .await?
            .is_some()
        {
            hits += 1;
        }
    }
    Ok(EstimateView {
        total_items: items.len(),
        total_characters: items.iter().map(|item| item.text.chars().count()).sum(),
        cache_hits: hits,
    })
}

pub async fn create_job(
    pool: &SqlitePool,
    story_id: &str,
    request: CreateJobRequest,
) -> anyhow::Result<JobView> {
    validate_request(&request)?;
    let (branch, story_language) = story_scope(pool, story_id).await?;
    let source_language = if request.source_language.trim().is_empty() {
        story_language
    } else {
        request.source_language.trim().to_string()
    };
    let items = source_items(pool, story_id, &branch, &request).await?;
    if items.is_empty() {
        return Err(PublicError::bad_request(
            "empty_translation_scope",
            "The selected scope has no translatable content.",
        )
        .into());
    }
    let job_id = Uuid::new_v4().to_string();
    let total_chars: usize = items.iter().map(|item| item.text.chars().count()).sum();
    let mut tx = pool.begin().await?;
    sqlx::query(r#"INSERT INTO translation_jobs(id,story_id,branch_id,scope_kind,scope_id,source_language,target_language,engine,provider,model,style,status,total_items,total_characters)
        VALUES(?,?,?,?,?,?,?,?,?,?,?,'queued',?,?)"#)
        .bind(&job_id).bind(story_id).bind(&branch).bind(&request.scope_kind).bind(&request.scope_id)
        .bind(&source_language).bind(request.target_language.trim()).bind(&request.engine).bind(request.provider.trim()).bind(request.model.trim()).bind(&request.style)
        .bind(items.len() as i64).bind(total_chars as i64).execute(&mut *tx).await?;
    for (ordinal, item) in items.iter().enumerate() {
        let hash = content_hash(&item.text);
        let cached = cached_translation_id_tx(
            &mut tx,
            story_id,
            &branch,
            item,
            &hash,
            &source_language,
            &request,
        )
        .await?;
        let (status, translation_id) = cached
            .as_ref()
            .map(|id| ("cache_hit", Some(id.as_str())))
            .unwrap_or(("pending", None));
        sqlx::query("INSERT INTO translation_job_items(id,job_id,ordinal,content_kind,content_id,content_hash,source_text,status,translation_id) VALUES(?,?,?,?,?,?,?,?,?)")
            .bind(Uuid::new_v4().to_string()).bind(&job_id).bind(ordinal as i64).bind(&item.kind).bind(&item.id).bind(hash).bind(&item.text).bind(status).bind(translation_id).execute(&mut *tx).await?;
    }
    refresh_job_tx(&mut tx, &job_id).await?;
    tx.commit().await?;
    get_job(pool, story_id, &job_id).await
}

fn validate_request(request: &CreateJobRequest) -> anyhow::Result<()> {
    if !matches!(
        request.scope_kind.as_str(),
        "chapter" | "story" | "selection"
    ) {
        return Err(PublicError::bad_request(
            "invalid_translation_scope",
            "Translation scope must be chapter, story or selection.",
        )
        .into());
    }
    if !matches!(request.engine.as_str(), "browser" | "ai") {
        return Err(PublicError::bad_request(
            "invalid_translation_engine",
            "Translation engine must be browser or ai.",
        )
        .into());
    }
    if !matches!(request.style.as_str(), "faithful" | "natural" | "literary") {
        return Err(PublicError::bad_request(
            "invalid_translation_style",
            "Translation style is invalid.",
        )
        .into());
    }
    if request.target_language.trim().is_empty() {
        return Err(PublicError::bad_request(
            "missing_target_language",
            "A target language is required.",
        )
        .into());
    }
    if request.engine == "ai" && request.provider.trim().is_empty() {
        return Err(PublicError::bad_request(
            "missing_translation_provider",
            "Choose an AI provider explicitly.",
        )
        .into());
    }
    Ok(())
}

async fn story_scope(pool: &SqlitePool, story_id: &str) -> anyhow::Result<(String, String)> {
    let row = sqlx::query("SELECT active_branch_id,language FROM stories WHERE id=?")
        .bind(story_id)
        .fetch_optional(pool)
        .await?
        .ok_or_else(|| PublicError::not_found("story_not_found", "Story not found."))?;
    Ok((row.get(0), row.get(1)))
}

async fn source_items(
    pool: &SqlitePool,
    story_id: &str,
    branch: &str,
    request: &CreateJobRequest,
) -> anyhow::Result<Vec<SourceItem>> {
    let mut items = Vec::new();
    if request.scope_kind == "chapter" {
        let chapter_id: i64 = request
            .scope_id
            .parse()
            .map_err(|_| PublicError::bad_request("invalid_chapter", "Choose a valid chapter."))?;
        let chapter = sqlx::query("SELECT id,title,summary,start_turn,COALESCE(end_turn,9223372036854775807) FROM chapters WHERE id=? AND story_id=? AND branch_id=?")
            .bind(chapter_id).bind(story_id).bind(branch).fetch_optional(pool).await?.ok_or_else(|| PublicError::not_found("chapter_not_found", "Chapter not found."))?;
        items.push(SourceItem {
            kind: "chapter_title".into(),
            id: chapter.get::<i64, _>(0).to_string(),
            text: chapter.get(1),
        });
        items.push(SourceItem {
            kind: "chapter_summary".into(),
            id: chapter.get::<i64, _>(0).to_string(),
            text: chapter.get(2),
        });
        let rows = sqlx::query("SELECT id,content FROM chat_messages WHERE story_id=? AND branch_id=? AND turn>=? AND turn<=? ORDER BY id")
            .bind(story_id).bind(branch).bind(chapter.get::<i64,_>(3)).bind(chapter.get::<i64,_>(4)).fetch_all(pool).await?;
        items.extend(rows.into_iter().map(|row| SourceItem {
            kind: "message".into(),
            id: row.get::<i64, _>(0).to_string(),
            text: row.get(1),
        }));
    } else {
        if request.scope_kind == "story" {
            let chapters = sqlx::query(
                "SELECT id,title,summary FROM chapters WHERE story_id=? AND branch_id=? ORDER BY chapter_number",
            )
            .bind(story_id)
            .bind(branch)
            .fetch_all(pool)
            .await?;
            for chapter in chapters {
                let id = chapter.get::<i64, _>(0).to_string();
                items.push(SourceItem {
                    kind: "chapter_title".into(),
                    id: id.clone(),
                    text: chapter.get(1),
                });
                items.push(SourceItem {
                    kind: "chapter_summary".into(),
                    id,
                    text: chapter.get(2),
                });
            }
        }
        let rows = if request.scope_kind == "selection" {
            if request.message_ids.is_empty() {
                return Ok(items);
            }
            let placeholders = std::iter::repeat_n("?", request.message_ids.len())
                .collect::<Vec<_>>()
                .join(",");
            let sql = format!("SELECT id,content FROM chat_messages WHERE story_id=? AND branch_id=? AND id IN ({placeholders}) ORDER BY id");
            let mut query = sqlx::query(&sql).bind(story_id).bind(branch);
            for id in &request.message_ids {
                query = query.bind(id);
            }
            query.fetch_all(pool).await?
        } else {
            sqlx::query(
                "SELECT id,content FROM chat_messages WHERE story_id=? AND branch_id=? ORDER BY id",
            )
            .bind(story_id)
            .bind(branch)
            .fetch_all(pool)
            .await?
        };
        items.extend(rows.into_iter().map(|row| SourceItem {
            kind: "message".into(),
            id: row.get::<i64, _>(0).to_string(),
            text: row.get(1),
        }));
    }
    items.retain(|item| !item.text.trim().is_empty());
    Ok(items)
}

fn content_hash(text: &str) -> String {
    format!("{:x}", Sha256::digest(text.as_bytes()))
}

async fn cached_translation_id(
    pool: &SqlitePool,
    story_id: &str,
    branch: &str,
    item: &SourceItem,
    hash: &str,
    source: &str,
    request: &CreateJobRequest,
) -> anyhow::Result<Option<String>> {
    Ok(sqlx::query_scalar(CACHE_QUERY)
        .bind(story_id)
        .bind(branch)
        .bind(&item.kind)
        .bind(&item.id)
        .bind(hash)
        .bind(source)
        .bind(request.target_language.trim())
        .bind(&request.engine)
        .bind(request.provider.trim())
        .bind(request.model.trim())
        .bind(&request.style)
        .bind(CODEC_VERSION)
        .fetch_optional(pool)
        .await?)
}

async fn cached_translation_id_tx(
    tx: &mut Transaction<'_, Sqlite>,
    story_id: &str,
    branch: &str,
    item: &SourceItem,
    hash: &str,
    source: &str,
    request: &CreateJobRequest,
) -> anyhow::Result<Option<String>> {
    Ok(sqlx::query_scalar(CACHE_QUERY)
        .bind(story_id)
        .bind(branch)
        .bind(&item.kind)
        .bind(&item.id)
        .bind(hash)
        .bind(source)
        .bind(request.target_language.trim())
        .bind(&request.engine)
        .bind(request.provider.trim())
        .bind(request.model.trim())
        .bind(&request.style)
        .bind(CODEC_VERSION)
        .fetch_optional(&mut **tx)
        .await?)
}

const CACHE_QUERY: &str = "SELECT id FROM content_translations WHERE story_id=? AND branch_id=? AND content_kind=? AND content_id=? AND content_hash=? AND source_language=? AND target_language=? AND engine=? AND provider=? AND model=? AND style=? AND codec_version=? ORDER BY created_at DESC LIMIT 1";

pub async fn list_jobs(pool: &SqlitePool, story_id: &str) -> anyhow::Result<Vec<JobView>> {
    let rows = sqlx::query(JOB_SELECT)
        .bind(story_id)
        .fetch_all(pool)
        .await?;
    rows.iter().map(job_from_row).collect()
}

pub async fn get_job(pool: &SqlitePool, story_id: &str, job_id: &str) -> anyhow::Result<JobView> {
    let row = sqlx::query(&format!(
        "{} AND id=?",
        JOB_SELECT.trim_end_matches(" ORDER BY created_at DESC")
    ))
    .bind(story_id)
    .bind(job_id)
    .fetch_optional(pool)
    .await?
    .ok_or_else(|| {
        PublicError::not_found("translation_job_not_found", "Translation job not found.")
    })?;
    job_from_row(&row)
}

const JOB_SELECT: &str = "SELECT id,story_id,branch_id,scope_kind,scope_id,source_language,target_language,engine,provider,model,style,status,total_items,completed_items,failed_items,cached_items,total_characters,processed_characters,error_code,error_summary,CAST(created_at AS TEXT),CAST(updated_at AS TEXT) FROM translation_jobs WHERE story_id=? ORDER BY created_at DESC";

fn job_from_row(row: &sqlx::sqlite::SqliteRow) -> anyhow::Result<JobView> {
    Ok(JobView {
        id: row.try_get(0)?,
        story_id: row.try_get(1)?,
        branch_id: row.try_get(2)?,
        scope_kind: row.try_get(3)?,
        scope_id: row.try_get(4)?,
        source_language: row.try_get(5)?,
        target_language: row.try_get(6)?,
        engine: row.try_get(7)?,
        provider: row.try_get(8)?,
        model: row.try_get(9)?,
        style: row.try_get(10)?,
        status: row.try_get(11)?,
        total_items: row.try_get(12)?,
        completed_items: row.try_get(13)?,
        failed_items: row.try_get(14)?,
        cached_items: row.try_get(15)?,
        total_characters: row.try_get(16)?,
        processed_characters: row.try_get(17)?,
        error_code: row.try_get(18)?,
        error_summary: row.try_get(19)?,
        created_at: row.try_get(20)?,
        updated_at: row.try_get(21)?,
    })
}

pub async fn job_action(
    pool: &SqlitePool,
    story_id: &str,
    job_id: &str,
    action: &str,
) -> anyhow::Result<JobView> {
    let mut tx = pool.begin().await?;
    let belongs: i64 =
        sqlx::query_scalar("SELECT COUNT(*) FROM translation_jobs WHERE id=? AND story_id=?")
            .bind(job_id)
            .bind(story_id)
            .fetch_one(&mut *tx)
            .await?;
    if belongs == 0 {
        return Err(PublicError::not_found(
            "translation_job_not_found",
            "Translation job not found.",
        )
        .into());
    }
    match action {
        "pause" => {
            sqlx::query("UPDATE translation_jobs SET status='paused',updated_at=CURRENT_TIMESTAMP WHERE id=? AND status IN ('queued','running')").bind(job_id).execute(&mut *tx).await?;
        }
        "resume" => {
            sqlx::query("UPDATE translation_job_items SET status='pending',updated_at=CURRENT_TIMESTAMP WHERE job_id=? AND status IN ('running','cancelled')").bind(job_id).execute(&mut *tx).await?;
            sqlx::query("UPDATE translation_jobs SET status='queued',completed_at=NULL,error_code='',error_summary='',updated_at=CURRENT_TIMESTAMP WHERE id=? AND status IN ('paused','cancelled','partial','failed')").bind(job_id).execute(&mut *tx).await?;
        }
        "cancel" => {
            sqlx::query("UPDATE translation_job_items SET status='cancelled',updated_at=CURRENT_TIMESTAMP WHERE job_id=? AND status IN ('pending','running')").bind(job_id).execute(&mut *tx).await?;
            sqlx::query("UPDATE translation_jobs SET status='cancelled',completed_at=CURRENT_TIMESTAMP,updated_at=CURRENT_TIMESTAMP WHERE id=? AND status NOT IN ('completed','cancelled')").bind(job_id).execute(&mut *tx).await?;
        }
        "retry" => {
            sqlx::query("UPDATE translation_job_items SET status='pending',error_code='',error_summary='',updated_at=CURRENT_TIMESTAMP WHERE job_id=? AND status='failed'").bind(job_id).execute(&mut *tx).await?;
            sqlx::query("UPDATE translation_jobs SET status='queued',completed_at=NULL,error_code='',error_summary='',updated_at=CURRENT_TIMESTAMP WHERE id=?").bind(job_id).execute(&mut *tx).await?;
        }
        _ => {
            return Err(PublicError::bad_request(
                "invalid_translation_action",
                "Unknown translation job action.",
            )
            .into())
        }
    }
    refresh_job_tx(&mut tx, job_id).await?;
    tx.commit().await?;
    get_job(pool, story_id, job_id).await
}

pub async fn delete_job(
    pool: &SqlitePool,
    story_id: &str,
    job_id: &str,
    delete_translations: bool,
) -> anyhow::Result<()> {
    let mut tx = pool.begin().await?;
    if delete_translations {
        sqlx::query("DELETE FROM content_translations WHERE id IN (SELECT translation_id FROM translation_job_items WHERE job_id=?)").bind(job_id).execute(&mut *tx).await?;
    }
    let result = sqlx::query("DELETE FROM translation_jobs WHERE id=? AND story_id=?")
        .bind(job_id)
        .bind(story_id)
        .execute(&mut *tx)
        .await?;
    if result.rows_affected() == 0 {
        return Err(PublicError::not_found(
            "translation_job_not_found",
            "Translation job not found.",
        )
        .into());
    }
    tx.commit().await?;
    Ok(())
}

pub async fn next_browser_item(
    pool: &SqlitePool,
    story_id: &str,
    job_id: &str,
) -> anyhow::Result<Option<BrowserItemView>> {
    let mut tx = pool.begin().await?;
    let job = sqlx::query("SELECT source_language,target_language,status,engine FROM translation_jobs WHERE id=? AND story_id=?").bind(job_id).bind(story_id).fetch_optional(&mut *tx).await?
        .ok_or_else(|| PublicError::not_found("translation_job_not_found", "Translation job not found."))?;
    if job.get::<String, _>(3) != "browser"
        || !matches!(job.get::<String, _>(2).as_str(), "queued" | "running")
    {
        tx.commit().await?;
        return Ok(None);
    }
    let row = sqlx::query("SELECT id,content_kind,content_id,source_text FROM translation_job_items WHERE job_id=? AND status='pending' ORDER BY ordinal LIMIT 1").bind(job_id).fetch_optional(&mut *tx).await?;
    let Some(row) = row else {
        refresh_job_tx(&mut tx, job_id).await?;
        tx.commit().await?;
        return Ok(None);
    };
    let id: String = row.get(0);
    sqlx::query("UPDATE translation_job_items SET status='running',attempt_count=attempt_count+1,updated_at=CURRENT_TIMESTAMP WHERE id=?").bind(&id).execute(&mut *tx).await?;
    sqlx::query("UPDATE translation_jobs SET status='running',started_at=COALESCE(started_at,CURRENT_TIMESTAMP),updated_at=CURRENT_TIMESTAMP WHERE id=?").bind(job_id).execute(&mut *tx).await?;
    let source_text: String = row.get(3);
    let (protected_text, _) = protect_text(&source_text);
    let result = BrowserItemView {
        id,
        content_kind: row.get(1),
        content_id: row.get(2),
        source_text: protected_text,
        source_language: job.get(0),
        target_language: job.get(1),
    };
    tx.commit().await?;
    Ok(Some(result))
}

pub async fn complete_browser_item(
    pool: &SqlitePool,
    story_id: &str,
    job_id: &str,
    item_id: &str,
    translated_text: String,
) -> anyhow::Result<JobView> {
    if translated_text.trim().is_empty() {
        return Err(
            PublicError::bad_request("empty_translation", "Translation cannot be empty.").into(),
        );
    }
    let source_text: String =
        sqlx::query_scalar("SELECT source_text FROM translation_job_items WHERE id=? AND job_id=?")
            .bind(item_id)
            .bind(job_id)
            .fetch_optional(pool)
            .await?
            .ok_or_else(|| {
                PublicError::not_found("translation_item_not_found", "Translation item not found.")
            })?;
    let (_, tokens) = protect_text(&source_text);
    let translated_text = restore_text(translated_text, &tokens)?;
    let mut tx = pool.begin().await?;
    store_item_translation_tx(&mut tx, story_id, job_id, item_id, translated_text).await?;
    refresh_job_tx(&mut tx, job_id).await?;
    tx.commit().await?;
    get_job(pool, story_id, job_id).await
}

async fn store_item_translation_tx(
    tx: &mut Transaction<'_, Sqlite>,
    story_id: &str,
    job_id: &str,
    item_id: &str,
    translated_text: String,
) -> anyhow::Result<()> {
    let row = sqlx::query(r#"SELECT j.branch_id,j.source_language,j.target_language,j.engine,j.provider,j.model,j.style,i.content_kind,i.content_id,i.content_hash,LENGTH(i.source_text),i.status
        FROM translation_job_items i JOIN translation_jobs j ON j.id=i.job_id WHERE i.id=? AND i.job_id=? AND j.story_id=?"#)
        .bind(item_id).bind(job_id).bind(story_id).fetch_optional(&mut **tx).await?.ok_or_else(|| PublicError::not_found("translation_item_not_found", "Translation item not found."))?;
    if row.get::<String, _>(11) != "running" {
        return Err(PublicError::conflict(
            "translation_item_not_running",
            "Translation item is not running.",
        )
        .into());
    }
    let translation_id = Uuid::new_v4().to_string();
    sqlx::query(r#"INSERT INTO content_translations(id,story_id,branch_id,content_kind,content_id,content_hash,source_language,target_language,engine,provider,model,style,codec_version,translated_text,character_count)
        VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(story_id,branch_id,content_kind,content_id,content_hash,source_language,target_language,engine,provider,model,style,codec_version)
        DO UPDATE SET translated_text=excluded.translated_text,character_count=excluded.character_count,created_at=CURRENT_TIMESTAMP"#)
        .bind(&translation_id).bind(story_id).bind(row.get::<String,_>(0)).bind(row.get::<String,_>(7)).bind(row.get::<String,_>(8)).bind(row.get::<String,_>(9)).bind(row.get::<String,_>(1)).bind(row.get::<String,_>(2)).bind(row.get::<String,_>(3)).bind(row.get::<String,_>(4)).bind(row.get::<String,_>(5)).bind(row.get::<String,_>(6)).bind(CODEC_VERSION).bind(&translated_text).bind(translated_text.chars().count() as i64).execute(&mut **tx).await?;
    let actual_id: String = sqlx::query_scalar(CACHE_QUERY)
        .bind(story_id)
        .bind(row.get::<String, _>(0))
        .bind(row.get::<String, _>(7))
        .bind(row.get::<String, _>(8))
        .bind(row.get::<String, _>(9))
        .bind(row.get::<String, _>(1))
        .bind(row.get::<String, _>(2))
        .bind(row.get::<String, _>(3))
        .bind(row.get::<String, _>(4))
        .bind(row.get::<String, _>(5))
        .bind(row.get::<String, _>(6))
        .bind(CODEC_VERSION)
        .fetch_one(&mut **tx)
        .await?;
    sqlx::query("UPDATE translation_job_items SET status='completed',translation_id=?,error_code='',error_summary='',updated_at=CURRENT_TIMESTAMP WHERE id=?").bind(actual_id).bind(item_id).execute(&mut **tx).await?;
    Ok(())
}

async fn refresh_job_tx(tx: &mut Transaction<'_, Sqlite>, job_id: &str) -> anyhow::Result<()> {
    let row = sqlx::query(r#"SELECT COUNT(*),SUM(CASE WHEN status IN ('completed','cache_hit') THEN 1 ELSE 0 END),SUM(CASE WHEN status='failed' THEN 1 ELSE 0 END),SUM(CASE WHEN status='cache_hit' THEN 1 ELSE 0 END),COALESCE(SUM(CASE WHEN status IN ('completed','cache_hit') THEN LENGTH(source_text) ELSE 0 END),0),SUM(CASE WHEN status IN ('pending','running') THEN 1 ELSE 0 END) FROM translation_job_items WHERE job_id=?"#).bind(job_id).fetch_one(&mut **tx).await?;
    let total: i64 = row.get(0);
    let completed: i64 = row.get(1);
    let failed: i64 = row.get(2);
    let cached: i64 = row.get(3);
    let chars: i64 = row.get(4);
    let remaining: i64 = row.get(5);
    let current: String = sqlx::query_scalar("SELECT status FROM translation_jobs WHERE id=?")
        .bind(job_id)
        .fetch_one(&mut **tx)
        .await?;
    let status = if matches!(current.as_str(), "paused" | "cancelled") || remaining > 0 {
        current
    } else if failed > 0 && completed > 0 {
        "partial".into()
    } else if failed > 0 {
        "failed".into()
    } else {
        "completed".into()
    };
    sqlx::query("UPDATE translation_jobs SET completed_items=?,failed_items=?,cached_items=?,processed_characters=?,status=?,completed_at=CASE WHEN ? IN ('completed','partial','failed','cancelled') THEN CURRENT_TIMESTAMP ELSE completed_at END,updated_at=CURRENT_TIMESTAMP WHERE id=?")
        .bind(completed).bind(failed).bind(cached).bind(chars).bind(&status).bind(&status).bind(job_id).execute(&mut **tx).await?;
    let _ = total;
    Ok(())
}

pub async fn list_glossary(
    pool: &SqlitePool,
    story_id: &str,
) -> anyhow::Result<Vec<GlossaryEntry>> {
    let rows = sqlx::query("SELECT id,source_language,target_language,source_term,target_term,mode FROM translation_glossary_entries WHERE story_id=? ORDER BY source_term COLLATE NOCASE").bind(story_id).fetch_all(pool).await?;
    Ok(rows
        .into_iter()
        .map(|row| GlossaryEntry {
            id: row.get(0),
            source_language: row.get(1),
            target_language: row.get(2),
            source_term: row.get(3),
            target_term: row.get(4),
            mode: row.get(5),
        })
        .collect())
}

pub async fn upsert_glossary(
    pool: &SqlitePool,
    story_id: &str,
    id: Option<&str>,
    request: GlossaryEntryRequest,
) -> anyhow::Result<GlossaryEntry> {
    let source_term = request.source_term.trim();
    if source_term.is_empty() {
        return Err(PublicError::bad_request(
            "missing_glossary_term",
            "A glossary source term is required.",
        )
        .into());
    }
    if !matches!(request.mode.as_str(), "translate" | "preserve") {
        return Err(
            PublicError::bad_request("invalid_glossary_mode", "Glossary mode is invalid.").into(),
        );
    }
    let id = id
        .map(str::to_owned)
        .unwrap_or_else(|| Uuid::new_v4().to_string());
    sqlx::query(r#"INSERT INTO translation_glossary_entries(id,story_id,source_language,target_language,source_term,target_term,mode) VALUES(?,?,?,?,?,?,?)
        ON CONFLICT(story_id,source_language,target_language,source_term) DO UPDATE SET target_term=excluded.target_term,mode=excluded.mode,updated_at=CURRENT_TIMESTAMP"#)
        .bind(&id).bind(story_id).bind(request.source_language.trim()).bind(request.target_language.trim()).bind(source_term).bind(request.target_term.trim()).bind(&request.mode).execute(pool).await?;
    let row = sqlx::query("SELECT id,source_language,target_language,source_term,target_term,mode FROM translation_glossary_entries WHERE story_id=? AND source_language=? AND target_language=? AND source_term=?")
        .bind(story_id).bind(request.source_language.trim()).bind(request.target_language.trim()).bind(source_term).fetch_one(pool).await?;
    Ok(GlossaryEntry {
        id: row.get(0),
        source_language: row.get(1),
        target_language: row.get(2),
        source_term: row.get(3),
        target_term: row.get(4),
        mode: row.get(5),
    })
}

pub async fn delete_glossary(pool: &SqlitePool, story_id: &str, id: &str) -> anyhow::Result<()> {
    let changed = sqlx::query("DELETE FROM translation_glossary_entries WHERE id=? AND story_id=?")
        .bind(id)
        .bind(story_id)
        .execute(pool)
        .await?
        .rows_affected();
    if changed == 0 {
        return Err(PublicError::not_found(
            "glossary_entry_not_found",
            "Glossary entry not found.",
        )
        .into());
    }
    Ok(())
}

pub fn spawn_worker(state: Arc<AppState>) {
    tokio::spawn(async move {
        if let Err(error) = sqlx::query("UPDATE translation_job_items SET status='pending' WHERE status='running' AND job_id IN (SELECT id FROM translation_jobs WHERE engine='ai' AND status='running')").execute(&state.pool).await { tracing::warn!(%error, "translation recovery failed"); }
        let _ = sqlx::query("UPDATE translation_jobs SET status='queued',updated_at=CURRENT_TIMESTAMP WHERE engine='ai' AND status='running'").execute(&state.pool).await;
        loop {
            match claim_ai_item(&state.pool).await {
                Ok(Some(item)) => process_ai_item(state.clone(), item).await,
                Ok(None) => tokio::time::sleep(Duration::from_millis(750)).await,
                Err(error) => {
                    tracing::warn!(%error, "translation worker claim failed");
                    tokio::time::sleep(Duration::from_secs(2)).await;
                }
            }
        }
    });
}

struct ClaimedItem {
    story_id: String,
    job_id: String,
    item_id: String,
    text: String,
    source: String,
    target: String,
    provider: String,
    model: String,
    style: String,
}

async fn claim_ai_item(pool: &SqlitePool) -> anyhow::Result<Option<ClaimedItem>> {
    let mut tx = pool.begin().await?;
    let row = sqlx::query(r#"SELECT j.story_id,j.id,i.id,i.source_text,j.source_language,j.target_language,j.provider,j.model,j.style
        FROM translation_jobs j JOIN translation_job_items i ON i.job_id=j.id WHERE j.engine='ai' AND j.status IN ('queued','running') AND i.status='pending' ORDER BY j.created_at,i.ordinal LIMIT 1"#).fetch_optional(&mut *tx).await?;
    let Some(row) = row else {
        tx.commit().await?;
        return Ok(None);
    };
    let item = ClaimedItem {
        story_id: row.get(0),
        job_id: row.get(1),
        item_id: row.get(2),
        text: row.get(3),
        source: row.get(4),
        target: row.get(5),
        provider: row.get(6),
        model: row.get(7),
        style: row.get(8),
    };
    sqlx::query("UPDATE translation_job_items SET status='running',attempt_count=attempt_count+1,updated_at=CURRENT_TIMESTAMP WHERE id=? AND status='pending'").bind(&item.item_id).execute(&mut *tx).await?;
    sqlx::query("UPDATE translation_jobs SET status='running',started_at=COALESCE(started_at,CURRENT_TIMESTAMP),updated_at=CURRENT_TIMESTAMP WHERE id=? AND status='queued'").bind(&item.job_id).execute(&mut *tx).await?;
    tx.commit().await?;
    Ok(Some(item))
}

async fn process_ai_item(state: Arc<AppState>, item: ClaimedItem) {
    let result = async {
        let (protected, tokens) = protect_text(&item.text);
        let glossary =
            glossary_map(&state.pool, &item.story_id, &item.source, &item.target).await?;
        let response = engine::translate(
            state.clone(),
            engine::TranslationRequest {
                text: protected,
                source_language: item.source.clone(),
                target_language: item.target.clone(),
                provider: item.provider.clone(),
                model: item.model.clone(),
                style: item.style.clone(),
                glossary,
            },
        )
        .await?;
        restore_text(response.text.unwrap_or_default(), &tokens)
    }
    .await;
    let mut tx = match state.pool.begin().await {
        Ok(tx) => tx,
        Err(error) => {
            tracing::warn!(%error, "translation result transaction failed");
            return;
        }
    };
    match result {
        Ok(text) => {
            if let Err(error) = store_item_translation_tx(
                &mut tx,
                &item.story_id,
                &item.job_id,
                &item.item_id,
                text,
            )
            .await
            {
                let _ =
                    mark_item_failed(&mut tx, &item.item_id, "store_failed", &error.to_string())
                        .await;
            }
        }
        Err(error) => {
            let _ = mark_item_failed(
                &mut tx,
                &item.item_id,
                "translation_failed",
                &error.to_string(),
            )
            .await;
        }
    }
    let _ = refresh_job_tx(&mut tx, &item.job_id).await;
    let _ = tx.commit().await;
}

async fn mark_item_failed(
    tx: &mut Transaction<'_, Sqlite>,
    item_id: &str,
    code: &str,
    summary: &str,
) -> anyhow::Result<()> {
    let summary = summary.chars().take(300).collect::<String>();
    sqlx::query("UPDATE translation_job_items SET status='failed',error_code=?,error_summary=?,updated_at=CURRENT_TIMESTAMP WHERE id=?").bind(code).bind(summary).bind(item_id).execute(&mut **tx).await?;
    Ok(())
}

async fn glossary_map(
    pool: &SqlitePool,
    story_id: &str,
    source: &str,
    target: &str,
) -> anyhow::Result<BTreeMap<String, String>> {
    let rows = sqlx::query("SELECT source_term,target_term,mode FROM translation_glossary_entries WHERE story_id=? AND (source_language='' OR source_language=?) AND (target_language='' OR target_language=?)").bind(story_id).bind(source).bind(target).fetch_all(pool).await?;
    Ok(rows
        .into_iter()
        .map(|row| {
            let source: String = row.get(0);
            let mode: String = row.get(2);
            let target: String = row.get(1);
            (
                source.clone(),
                if mode == "preserve" { source } else { target },
            )
        })
        .collect())
}

fn protect_text(text: &str) -> (String, Vec<(String, String)>) {
    let mut output = text.to_string();
    let mut protected = Vec::new();
    for (start, end) in protected_ranges(text).into_iter().rev() {
        let original = output[start..end].to_string();
        let token = format!("[[ODP_{:04}]]", protected.len());
        output.replace_range(start..end, &token);
        protected.push((token, original));
    }
    (output, protected)
}

fn protected_ranges(text: &str) -> Vec<(usize, usize)> {
    let bytes = text.as_bytes();
    let mut ranges = Vec::new();
    let mut i = 0;
    while i < bytes.len() {
        let (open, close) = if text[i..].starts_with("```") {
            (3, "```")
        } else if bytes[i] == b'`' {
            (1, "`")
        } else if text[i..].starts_with("{{") {
            (2, "}}")
        } else if text[i..].starts_with("${") {
            (2, "}")
        } else if text[i..].starts_with("http://") || text[i..].starts_with("https://") {
            let end = text[i..]
                .find(char::is_whitespace)
                .map(|n| i + n)
                .unwrap_or(bytes.len());
            ranges.push((i, end));
            i = end;
            continue;
        } else {
            i += text[i..].chars().next().map(char::len_utf8).unwrap_or(1);
            continue;
        };
        if let Some(relative) = text[i + open..].find(close) {
            let end = i + open + relative + close.len();
            ranges.push((i, end));
            i = end;
        } else {
            i += open;
        }
    }
    ranges
}

fn restore_text(mut text: String, tokens: &[(String, String)]) -> anyhow::Result<String> {
    for (token, original) in tokens {
        if text.matches(token).count() != 1 {
            return Err(anyhow!("protected structure token {token} was changed"));
        }
        text = text.replace(token, original);
    }
    Ok(text)
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn protected_codec_round_trips_markdown_and_placeholders() {
        let source = "Read `x = 1`, keep {{hero}}, then https://example.test/a and:\n```rs\nfn main() {}\n```";
        let (encoded, tokens) = protect_text(source);
        assert!(!encoded.contains("example.test"));
        assert_eq!(restore_text(encoded, &tokens).unwrap(), source);
    }
    #[test]
    fn protected_codec_rejects_missing_tokens() {
        let (_, tokens) = protect_text("`secret`");
        assert!(restore_text("translated".into(), &tokens).is_err());
    }
}
