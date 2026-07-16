use crate::error::PublicError;
use anyhow::Context;
use serde::Serialize;
use serde_json::json;
use sha2::{Digest, Sha256};
use sqlx::{QueryBuilder, Row, Sqlite, SqlitePool};
use std::collections::HashMap;
use std::time::Instant;
use uuid::Uuid;

pub struct ImageGenerationTrace {
    pub run_id: String,
    pub attempt_id: String,
    started: Instant,
}

#[derive(Debug, Serialize)]
pub struct UsageView {
    pub input_tokens: i64,
    pub output_tokens: i64,
    pub reasoning_tokens: i64,
    pub cached_input_tokens: i64,
    pub total_tokens: i64,
    pub cost_usd: f64,
}

#[derive(Debug, Serialize)]
pub struct AttemptDiagnostics {
    pub sequence: i64,
    pub provider: String,
    pub requested_model: String,
    pub resolved_model: String,
    pub requested_streaming: bool,
    pub observed_streaming: bool,
    pub status: String,
    pub ttft_ms: i64,
    pub duration_ms: i64,
    pub usage: UsageView,
    pub retry_reason: String,
    pub error_class: String,
}

#[derive(Debug, Serialize)]
pub struct GenerationDiagnostics {
    pub run_id: String,
    pub trace_id: String,
    pub parent_run_id: String,
    pub story_id: String,
    pub branch_id: String,
    pub source_commit_id: String,
    pub message_id: Option<i64>,
    pub stage: String,
    pub status: String,
    pub prompt_profile: String,
    pub prompt_revision: i64,
    pub prompt_hash: String,
    pub requested_streaming: bool,
    pub observed_streaming: bool,
    pub ttft_ms: i64,
    pub duration_ms: i64,
    pub usage: UsageView,
    pub error_class: String,
    pub created_at: String,
    pub finished_at: String,
    pub attempts: Vec<AttemptDiagnostics>,
}

#[derive(Debug, Serialize)]
pub struct TelemetryExport {
    pub format: String,
    pub filename: String,
    pub content: String,
    pub count: usize,
    pub truncated: bool,
}

pub async fn message_diagnostics(
    pool: &SqlitePool,
    story_id: &str,
    message_id: i64,
) -> anyhow::Result<GenerationDiagnostics> {
    let run_id: String = sqlx::query_scalar(
        "SELECT id FROM generation_runs WHERE story_id=? AND message_id=? ORDER BY created_at DESC,id DESC LIMIT 1",
    )
    .bind(story_id)
    .bind(message_id)
    .fetch_optional(pool)
    .await?
    .ok_or_else(|| {
        PublicError::not_found(
            "generation_diagnostics_not_found",
            "generation diagnostics not found",
        )
    })?;
    run_diagnostics(pool, &run_id).await
}

pub async fn export_story_telemetry(
    pool: &SqlitePool,
    story_id: &str,
    limit: i64,
) -> anyhow::Result<TelemetryExport> {
    let limit = limit.clamp(1, 5000);
    let run_sql =
        format!("{DIAGNOSTICS_SELECT} WHERE r.story_id=? ORDER BY r.created_at,r.id LIMIT ?");
    let mut rows = sqlx::query(&run_sql)
        .bind(story_id)
        .bind(limit + 1)
        .fetch_all(pool)
        .await?;
    let truncated = rows.len() as i64 > limit;
    rows.truncate(limit as usize);
    let run_ids = rows
        .iter()
        .map(|row| row.try_get::<String, _>("id"))
        .collect::<Result<Vec<_>, _>>()?;
    let mut attempts_by_run: HashMap<String, Vec<AttemptDiagnostics>> = HashMap::new();
    for ids in run_ids.chunks(500) {
        let mut query = QueryBuilder::<Sqlite>::new(
            r#"SELECT run_id,sequence,provider,requested_model,resolved_model,requested_streaming,observed_streaming,status,
                  ttft_ms,duration_ms,input_tokens,output_tokens,reasoning_tokens,cached_input_tokens,total_tokens,cost_usd,
                  retry_reason,error_class FROM generation_attempts WHERE run_id IN ("#,
        );
        let mut separated = query.separated(",");
        for id in ids {
            separated.push_bind(id);
        }
        separated.push_unseparated(") ORDER BY run_id,sequence");
        for attempt in query.build().fetch_all(pool).await? {
            let run_id: String = attempt.try_get("run_id")?;
            attempts_by_run
                .entry(run_id)
                .or_default()
                .push(attempt_from_row(&attempt));
        }
    }
    let mut lines = Vec::with_capacity(rows.len());
    for row in rows {
        let run_id: String = row.try_get("id")?;
        let attempts = attempts_by_run.remove(&run_id).unwrap_or_default();
        lines.push(serde_json::to_string(&diagnostics_from_row(
            &row, attempts,
        )?)?);
    }
    let safe_story: String = story_id
        .chars()
        .map(|value| if value.is_alphanumeric() { value } else { '-' })
        .collect();
    Ok(TelemetryExport {
        format: "jsonl".into(),
        filename: format!("{safe_story}-generation-telemetry.jsonl"),
        content: lines.join("\n") + if lines.is_empty() { "" } else { "\n" },
        count: lines.len(),
        truncated,
    })
}

const DIAGNOSTICS_SELECT: &str = r#"SELECT r.id,r.trace_id,COALESCE(r.parent_run_id,'') AS parent_run_id,
                  r.story_id,r.branch_id,r.source_commit_id,r.message_id,r.stage,r.status,
                  COALESCE(p.name,'') AS prompt_profile,COALESCE(pr.version,0) AS prompt_revision,
                  r.prompt_hash,r.requested_streaming,r.observed_streaming,r.ttft_ms,r.duration_ms,
                  r.input_tokens,r.output_tokens,r.reasoning_tokens,r.cached_input_tokens,r.total_tokens,r.cost_usd,
                  r.error_class,CAST(r.created_at AS TEXT) AS created_at,COALESCE(CAST(r.finished_at AS TEXT),'') AS finished_at
           FROM generation_runs r
           LEFT JOIN prompt_profile_revisions pr ON pr.id=r.prompt_revision_id
           LEFT JOIN prompt_profiles p ON p.id=pr.profile_id"#;

pub async fn prune_expired(pool: &SqlitePool) -> anyhow::Result<u64> {
    let result = sqlx::query(
        r#"DELETE FROM generation_runs
           WHERE id IN (
             SELECT r.id FROM generation_runs r
             JOIN prompt_profile_revisions pr ON pr.id=r.prompt_revision_id
             JOIN prompt_profiles p ON p.id=pr.profile_id
             WHERE r.finished_at IS NOT NULL
               AND datetime(r.finished_at) < datetime('now', '-' || p.retention_days || ' days')
           )"#,
    )
    .execute(pool)
    .await?;
    Ok(result.rows_affected())
}

async fn run_diagnostics(pool: &SqlitePool, run_id: &str) -> anyhow::Result<GenerationDiagnostics> {
    let run_sql = format!("{DIAGNOSTICS_SELECT} WHERE r.id=?");
    let row = sqlx::query(&run_sql).bind(run_id).fetch_one(pool).await?;
    let attempt_rows = sqlx::query(
        r#"SELECT sequence,provider,requested_model,resolved_model,requested_streaming,observed_streaming,status,
                  ttft_ms,duration_ms,input_tokens,output_tokens,reasoning_tokens,cached_input_tokens,total_tokens,cost_usd,
                  retry_reason,error_class
           FROM generation_attempts WHERE run_id=? ORDER BY sequence"#,
    )
    .bind(run_id)
    .fetch_all(pool)
    .await?;
    let attempts = attempt_rows.iter().map(attempt_from_row).collect();
    diagnostics_from_row(&row, attempts)
}

fn attempt_from_row(attempt: &sqlx::sqlite::SqliteRow) -> AttemptDiagnostics {
    AttemptDiagnostics {
        sequence: attempt.try_get("sequence").unwrap_or_default(),
        provider: attempt.try_get("provider").unwrap_or_default(),
        requested_model: attempt.try_get("requested_model").unwrap_or_default(),
        resolved_model: attempt.try_get("resolved_model").unwrap_or_default(),
        requested_streaming: attempt
            .try_get::<i64, _>("requested_streaming")
            .unwrap_or_default()
            != 0,
        observed_streaming: attempt
            .try_get::<i64, _>("observed_streaming")
            .unwrap_or_default()
            != 0,
        status: attempt.try_get("status").unwrap_or_default(),
        ttft_ms: attempt.try_get("ttft_ms").unwrap_or_default(),
        duration_ms: attempt.try_get("duration_ms").unwrap_or_default(),
        usage: usage_from_row(attempt),
        retry_reason: attempt.try_get("retry_reason").unwrap_or_default(),
        error_class: attempt.try_get("error_class").unwrap_or_default(),
    }
}

fn diagnostics_from_row(
    row: &sqlx::sqlite::SqliteRow,
    attempts: Vec<AttemptDiagnostics>,
) -> anyhow::Result<GenerationDiagnostics> {
    Ok(GenerationDiagnostics {
        run_id: row.try_get("id")?,
        trace_id: row.try_get("trace_id")?,
        parent_run_id: row.try_get("parent_run_id")?,
        story_id: row.try_get("story_id")?,
        branch_id: row.try_get("branch_id")?,
        source_commit_id: row.try_get("source_commit_id")?,
        message_id: row.try_get("message_id")?,
        stage: row.try_get("stage")?,
        status: row.try_get("status")?,
        prompt_profile: row.try_get("prompt_profile")?,
        prompt_revision: row.try_get("prompt_revision")?,
        prompt_hash: row.try_get("prompt_hash")?,
        requested_streaming: row.try_get::<i64, _>("requested_streaming")? != 0,
        observed_streaming: row.try_get::<i64, _>("observed_streaming")? != 0,
        ttft_ms: row.try_get("ttft_ms")?,
        duration_ms: row.try_get("duration_ms")?,
        usage: usage_from_row(row),
        error_class: row.try_get("error_class")?,
        created_at: row.try_get("created_at")?,
        finished_at: row.try_get("finished_at")?,
        attempts,
    })
}

fn usage_from_row(row: &sqlx::sqlite::SqliteRow) -> UsageView {
    UsageView {
        input_tokens: row.try_get("input_tokens").unwrap_or_default(),
        output_tokens: row.try_get("output_tokens").unwrap_or_default(),
        reasoning_tokens: row.try_get("reasoning_tokens").unwrap_or_default(),
        cached_input_tokens: row.try_get("cached_input_tokens").unwrap_or_default(),
        total_tokens: row.try_get("total_tokens").unwrap_or_default(),
        cost_usd: row.try_get("cost_usd").unwrap_or_default(),
    }
}

#[allow(clippy::too_many_arguments)]
pub async fn start_image_generation(
    pool: &SqlitePool,
    story_id: &str,
    job_id: i64,
    attempt_number: i64,
    asset_id: &str,
    prompt: &str,
    provider: &str,
    model: &str,
) -> anyhow::Result<ImageGenerationTrace> {
    let prompt_hash = sha256_label(prompt.as_bytes());
    let profile_id = ensure_prompt_revision(pool, &prompt_hash, provider, model).await?;
    let lineage = sqlx::query(
        r#"SELECT COALESCE(NULLIF(j.branch_id,''),s.active_branch_id) AS branch_id,
                  COALESCE(NULLIF(j.source_commit_id,''),b.head_commit_id,'') AS source_commit_id
           FROM visual_generation_jobs j
           JOIN stories s ON s.id=j.story_id
           LEFT JOIN story_branches b ON b.id=s.active_branch_id
           WHERE j.id=?"#,
    )
    .bind(job_id)
    .fetch_one(pool)
    .await
    .context("loading image generation lineage")?;
    let branch_id: String = lineage.try_get("branch_id").unwrap_or_default();
    let source_commit_id: String = lineage.try_get("source_commit_id").unwrap_or_default();
    let trace_id = format!("image-job-{job_id}");
    let parent_run_id: Option<String> = sqlx::query_scalar(
        "SELECT id FROM generation_runs WHERE trace_id=? ORDER BY created_at DESC,id DESC LIMIT 1",
    )
    .bind(&trace_id)
    .fetch_optional(pool)
    .await?;
    let run_id = Uuid::new_v4().to_string();
    let attempt_id = Uuid::new_v4().to_string();
    let request_config = json!({
        "provider": provider,
        "requested_model": model,
        "job_id": job_id,
        "asset_id": asset_id,
    })
    .to_string();
    let metadata =
        json!({"job_id":job_id,"asset_id":asset_id,"attempt_number":attempt_number}).to_string();
    let mut tx = pool.begin().await?;
    sqlx::query(
        r#"INSERT INTO generation_runs
           (id,trace_id,parent_run_id,story_id,branch_id,source_commit_id,stage,status,prompt_revision_id,prompt_hash,request_config_json,requested_streaming,metadata_json)
           VALUES (?,?,?,?,?,?,'image_generation','running',?,?,?,0,?)"#,
    )
    .bind(&run_id)
    .bind(&trace_id)
    .bind(parent_run_id)
    .bind(story_id)
    .bind(branch_id)
    .bind(source_commit_id)
    .bind(profile_id)
    .bind(prompt_hash)
    .bind(request_config)
    .bind(metadata)
    .execute(&mut *tx)
    .await?;
    sqlx::query(
        r#"INSERT INTO generation_attempts
           (id,run_id,sequence,provider,requested_model,reasoning_config_json,requested_streaming,status,retry_reason)
           VALUES (?,?,1,?,?,'{"effort":"not_applicable","summary":"not_persisted"}',0,'running',?)"#,
    )
    .bind(&attempt_id)
    .bind(&run_id)
    .bind(provider)
    .bind(model)
    .bind(if attempt_number > 1 { "job_retry" } else { "" })
    .execute(&mut *tx)
    .await?;
    sqlx::query("INSERT INTO generation_events (run_id,attempt_id,event_type,payload_json) VALUES (?,?, 'request_started','{}')")
        .bind(&run_id)
        .bind(&attempt_id)
        .execute(&mut *tx)
        .await?;
    tx.commit().await?;
    Ok(ImageGenerationTrace {
        run_id,
        attempt_id,
        started: Instant::now(),
    })
}

impl ImageGenerationTrace {
    pub async fn succeed(self, pool: &SqlitePool, resolved_model: &str) -> anyhow::Result<()> {
        self.finish(pool, "succeeded", resolved_model, "", "").await
    }

    pub async fn fail(self, pool: &SqlitePool, error_class: &str) -> anyhow::Result<()> {
        self.finish(
            pool,
            "failed",
            "",
            error_class,
            "image provider request failed",
        )
        .await
    }

    pub async fn cancel(self, pool: &SqlitePool) -> anyhow::Result<()> {
        self.finish(
            pool,
            "cancelled",
            "",
            "cancelled",
            "image generation cancelled",
        )
        .await
    }

    async fn finish(
        self,
        pool: &SqlitePool,
        status: &str,
        resolved_model: &str,
        error_class: &str,
        error_summary: &str,
    ) -> anyhow::Result<()> {
        let duration = self.started.elapsed().as_millis() as i64;
        let mut tx = pool.begin().await?;
        sqlx::query("UPDATE generation_attempts SET status=?,resolved_model=?,duration_ms=?,error_class=?,error_summary=?,finished_at=CURRENT_TIMESTAMP WHERE id=? AND status='running'")
            .bind(status)
            .bind(resolved_model)
            .bind(duration)
            .bind(error_class)
            .bind(error_summary)
            .bind(&self.attempt_id)
            .execute(&mut *tx)
            .await?;
        sqlx::query("UPDATE generation_runs SET status=?,duration_ms=?,error_class=?,finished_at=CURRENT_TIMESTAMP WHERE id=? AND status='running'")
            .bind(status)
            .bind(duration)
            .bind(error_class)
            .bind(&self.run_id)
            .execute(&mut *tx)
            .await?;
        sqlx::query("INSERT INTO generation_events (run_id,attempt_id,event_type,payload_json) VALUES (?,?,?,?)")
            .bind(&self.run_id)
            .bind(&self.attempt_id)
            .bind(format!("request_{status}"))
            .bind(json!({"duration_ms":duration,"error_class":error_class}).to_string())
            .execute(&mut *tx)
            .await?;
        tx.commit().await?;
        Ok(())
    }
}

async fn ensure_prompt_revision(
    pool: &SqlitePool,
    prompt_hash: &str,
    provider: &str,
    model: &str,
) -> anyhow::Result<String> {
    let profile_id = "prompt-profile-image-generation";
    sqlx::query("INSERT OR IGNORE INTO prompt_profiles (id,name,description,redaction_policy,retention_days) VALUES (?,'image_generation','Image generation prompt fingerprint','secrets_and_reasoning',30)")
        .bind(profile_id)
        .execute(pool)
        .await?;
    if let Some(id) = sqlx::query_scalar::<_, String>("SELECT id FROM prompt_profile_revisions WHERE profile_id=? AND prompt_hash=? AND response_schema_hash='' LIMIT 1")
        .bind(profile_id)
        .bind(prompt_hash)
        .fetch_optional(pool)
        .await?
    {
        return Ok(id);
    }
    let version: i64 = sqlx::query_scalar(
        "SELECT COALESCE(MAX(version),0)+1 FROM prompt_profile_revisions WHERE profile_id=?",
    )
    .bind(profile_id)
    .fetch_one(pool)
    .await?;
    let revision_id = Uuid::new_v4().to_string();
    let config = json!({"provider":provider,"requested_model":model}).to_string();
    sqlx::query("INSERT INTO prompt_profile_revisions (id,profile_id,version,template_version,prompt_hash,response_schema_hash,config_json) VALUES (?, ?, ?, 'v1', ?, '', ?)")
        .bind(&revision_id)
        .bind(profile_id)
        .bind(version)
        .bind(prompt_hash)
        .bind(config)
        .execute(pool)
        .await?;
    Ok(revision_id)
}

fn sha256_label(value: &[u8]) -> String {
    let digest = Sha256::digest(value);
    let encoded: String = digest.iter().map(|byte| format!("{byte:02x}")).collect();
    format!("sha256:{encoded}")
}

pub fn classify_image_error(message: &str) -> &'static str {
    let lower = message.to_ascii_lowercase();
    if lower.contains("timeout") || lower.contains("deadline") {
        "timeout"
    } else if lower.contains("401") || lower.contains("403") || lower.contains("unauthorized") {
        "authentication"
    } else if lower.contains("429") || lower.contains("rate limit") {
        "rate_limit"
    } else if lower.contains("cancel") {
        "cancelled"
    } else if lower.contains("connect") || lower.contains("dns") || lower.contains("network") {
        "transport"
    } else {
        "provider_error"
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use sqlx::sqlite::SqlitePoolOptions;

    #[test]
    fn hashes_prompts_without_retaining_prompt_text() {
        let prompt = "private scene prompt bearer secret";
        let hash = sha256_label(prompt.as_bytes());
        assert!(hash.starts_with("sha256:"));
        assert!(!hash.contains("private"));
        assert_eq!(classify_image_error("request timeout"), "timeout");
        assert_eq!(classify_image_error("HTTP 429"), "rate_limit");
    }

    #[tokio::test]
    async fn diagnostics_export_and_retention_are_redacted_and_bounded() {
        let pool = SqlitePoolOptions::new()
            .max_connections(1)
            .connect("sqlite::memory:")
            .await
            .expect("memory pool");
        for statement in [
            "CREATE TABLE prompt_profiles (id TEXT PRIMARY KEY,name TEXT,retention_days INTEGER)",
            "CREATE TABLE prompt_profile_revisions (id TEXT PRIMARY KEY,profile_id TEXT,version INTEGER,prompt_hash TEXT,response_schema_hash TEXT)",
            "CREATE TABLE generation_runs (id TEXT PRIMARY KEY,trace_id TEXT,parent_run_id TEXT,story_id TEXT,branch_id TEXT,source_commit_id TEXT,message_id INTEGER,stage TEXT,status TEXT,prompt_revision_id TEXT,prompt_hash TEXT,request_config_json TEXT,requested_streaming INTEGER,observed_streaming INTEGER,input_tokens INTEGER,output_tokens INTEGER,reasoning_tokens INTEGER,cached_input_tokens INTEGER,total_tokens INTEGER,cost_usd REAL,ttft_ms INTEGER,duration_ms INTEGER,error_class TEXT,metadata_json TEXT,created_at TEXT,finished_at TEXT)",
            "CREATE TABLE generation_attempts (id TEXT PRIMARY KEY,run_id TEXT,sequence INTEGER,provider TEXT,requested_model TEXT,resolved_model TEXT,requested_streaming INTEGER,observed_streaming INTEGER,status TEXT,ttft_ms INTEGER,duration_ms INTEGER,input_tokens INTEGER,output_tokens INTEGER,reasoning_tokens INTEGER,cached_input_tokens INTEGER,total_tokens INTEGER,cost_usd REAL,retry_reason TEXT,error_class TEXT)",
            "CREATE TABLE generation_events (id INTEGER PRIMARY KEY,run_id TEXT,attempt_id TEXT,event_type TEXT,payload_json TEXT)",
        ] {
            sqlx::query(statement).execute(&pool).await.expect("schema");
        }
        sqlx::query("INSERT INTO prompt_profiles VALUES ('profile','narrator',1)")
            .execute(&pool)
            .await
            .unwrap();
        sqlx::query(
            "INSERT INTO prompt_profile_revisions VALUES ('revision','profile',2,'sha256:safe','')",
        )
        .execute(&pool)
        .await
        .unwrap();
        sqlx::query("INSERT INTO generation_runs VALUES ('run','trace','', 'story','branch','commit',42,'narrator','succeeded','revision','sha256:safe','{}',1,1,10,5,2,1,15,0.01,12,30,'','{}','2020-01-01','2020-01-01')")
            .execute(&pool)
            .await
            .unwrap();
        sqlx::query("INSERT INTO generation_attempts VALUES ('attempt','run',1,'provider','requested','resolved',1,1,'succeeded',12,30,10,5,2,1,15,0.01,'','')")
            .execute(&pool)
            .await
            .unwrap();
        sqlx::query("INSERT INTO generation_runs VALUES ('run-2','trace-2','', 'story','branch','commit',43,'narrator','succeeded','revision','sha256:safe','{}',0,0,1,2,0,0,3,0.0,0,8,'','{}','2020-01-02','2020-01-02')")
			.execute(&pool).await.unwrap();
        sqlx::query("INSERT INTO generation_attempts VALUES ('attempt-2','run-2',1,'provider-2','requested','resolved',0,0,'succeeded',0,8,1,2,0,0,3,0.0,'','')")
			.execute(&pool).await.unwrap();

        let diagnostics = message_diagnostics(&pool, "story", 42).await.unwrap();
        assert_eq!(diagnostics.prompt_profile, "narrator");
        assert_eq!(diagnostics.attempts.len(), 1);
        assert_eq!(diagnostics.usage.reasoning_tokens, 2);
        let export = export_story_telemetry(&pool, "story", 2).await.unwrap();
        assert_eq!(export.count, 2);
        assert!(!export.content.contains("request_config_json"));
        assert!(!export.content.contains("metadata_json"));
        let exported = export
            .content
            .lines()
            .map(|line| serde_json::from_str::<serde_json::Value>(line).unwrap())
            .collect::<Vec<_>>();
        assert_eq!(exported[0]["attempts"][0]["provider"], "provider");
        assert_eq!(exported[1]["attempts"][0]["provider"], "provider-2");

        assert_eq!(prune_expired(&pool).await.unwrap(), 2);
        let remaining: i64 = sqlx::query_scalar("SELECT COUNT(*) FROM generation_runs")
            .fetch_one(&pool)
            .await
            .unwrap();
        assert_eq!(remaining, 0);
    }
}
