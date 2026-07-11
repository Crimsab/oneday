use anyhow::Context;
use serde_json::json;
use sha2::{Digest, Sha256};
use sqlx::{Row, SqlitePool};
use std::time::Instant;
use uuid::Uuid;

pub struct ImageGenerationTrace {
    pub run_id: String,
    pub attempt_id: String,
    started: Instant,
}

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

    #[test]
    fn hashes_prompts_without_retaining_prompt_text() {
        let prompt = "private scene prompt bearer secret";
        let hash = sha256_label(prompt.as_bytes());
        assert!(hash.starts_with("sha256:"));
        assert!(!hash.contains("private"));
        assert_eq!(classify_image_error("request timeout"), "timeout");
        assert_eq!(classify_image_error("HTTP 429"), "rate_limit");
    }
}
