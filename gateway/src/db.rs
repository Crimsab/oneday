use crate::error::PublicError;
use anyhow::Context;
use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use sqlx::{Row, SqliteConnection, SqlitePool};
use std::io::{Cursor, Write};
use zip::{write::SimpleFileOptions, CompressionMethod, ZipWriter};

#[derive(Clone, Debug, Serialize)]
pub struct StorySummary {
    pub id: String,
    pub name: String,
    pub description: String,
    pub genre: String,
    pub tone: String,
    pub language: String,
    pub is_archived: bool,
    pub updated_at: String,
}

#[derive(Debug, Serialize)]
pub struct StoryDeletePlan {
    pub story_id: String,
    pub story_name: String,
    pub counts: Vec<StoryDeleteCount>,
    pub total_rows: i64,
    pub retained_asset_files: Vec<String>,
}

#[derive(Debug, Serialize)]
pub struct StoryDeleteCount {
    pub table: String,
    pub rows: i64,
}

#[derive(Debug, Serialize)]
pub struct StoryOverview {
    pub story: StorySummary,
    pub active_branch_id: String,
    pub revision: i64,
    pub current_turn: i64,
    pub branch_count: i64,
    pub chapter_count: i64,
    pub save_count: i64,
    pub message_count: i64,
    pub asset_count: i64,
}

#[derive(Debug, Deserialize)]
pub struct StoryUpdate {
    pub name: Option<String>,
    pub description: Option<String>,
    pub genre: Option<String>,
    pub tone: Option<String>,
    pub language: Option<String>,
    pub is_archived: Option<bool>,
}

#[derive(Debug, Serialize)]
pub struct RecordView {
    pub id: String,
    pub name: String,
    pub fields: Value,
}

#[derive(Debug, Serialize)]
pub struct WorldView {
    pub id: String,
    pub current_location: String,
    pub current_location_id: String,
    pub spatial_regions: Value,
    pub spatial_locations: Value,
    pub spatial_edges: Value,
    pub world_time: Value,
    pub weather: Value,
    pub current_chapter: i64,
    pub current_turn: i64,
    pub known_locations: Value,
    pub global_events: Value,
    pub faction_standings: Value,
    pub story_hooks: Value,
    pub world_reactions: Value,
    pub investigations: Value,
    pub projects: Value,
    pub guidance: Value,
    pub fronts: Value,
    pub timeline: Value,
    pub scene_contract: Value,
    pub updated_at: String,
}

#[derive(Debug, Serialize)]
pub struct SessionView {
    pub id: String,
    pub story_id: String,
    pub started_at: String,
    pub ended_at: Option<String>,
    pub summary: String,
}

#[derive(Clone, Debug, Serialize)]
pub struct MessageView {
    pub id: i64,
    pub session_id: String,
    pub story_id: String,
    pub turn: i64,
    pub role: String,
    pub content: String,
    pub message_type: String,
    pub metadata: Value,
    pub created_at: String,
    pub branch_id: String,
    pub source_commit_id: String,
}

#[derive(Debug, Serialize)]
pub struct ChoiceView {
    pub id: i64,
    pub text: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub intent: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub risk: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub scope: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub certainty: Option<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    pub related_stats: Vec<String>,
}

#[derive(Clone, Debug, Serialize)]
pub struct ChapterView {
    pub id: i64,
    pub chapter_number: i64,
    pub title: String,
    pub summary: String,
    pub start_turn: i64,
    pub end_turn: Option<i64>,
    pub created_at: String,
    pub branch_id: String,
    pub source_commit_id: String,
}

#[derive(Debug, Serialize)]
pub struct HistoryPage {
    pub items: Vec<MessageView>,
    pub next_cursor: Option<i64>,
}

#[derive(Debug, Serialize)]
pub struct ChapterPage {
    pub items: Vec<ChapterView>,
    pub next_cursor: Option<i64>,
}

#[derive(Debug, Serialize)]
pub struct StoryExport {
    pub format: String,
    pub filename: String,
    pub content: String,
    pub encoding: String,
    pub content_type: String,
}

#[derive(Debug, Serialize)]
pub struct AgencyEventView {
    pub id: i64,
    pub story_id: String,
    pub branch_id: String,
    pub commit_id: String,
    pub canonical_turn: i64,
    pub entity_id: String,
    pub entity_name: String,
    pub action: String,
    pub summary: String,
    pub created_at: String,
}

pub async fn agency_events(
    pool: &SqlitePool,
    story_id: &str,
    limit: i64,
) -> anyhow::Result<Vec<AgencyEventView>> {
    let limit = limit.clamp(1, 100);
    let rows = sqlx::query(r#"SELECT ce.id,ce.story_id,ce.branch_id,ce.commit_id,tc.canonical_turn,
        COALESCE(json_extract(ce.payload_json,'$.entity_id'),''),COALESCE(json_extract(ce.payload_json,'$.entity_name'),''),
        COALESCE(json_extract(ce.payload_json,'$.action'),''),COALESCE(json_extract(ce.payload_json,'$.summary'),''),CAST(ce.created_at AS TEXT)
        FROM canonical_events ce JOIN turn_commits tc ON tc.id=ce.commit_id JOIN stories s ON s.id=ce.story_id
        WHERE ce.story_id=? AND ce.branch_id=s.active_branch_id AND ce.event_type='npc.agency'
        ORDER BY tc.canonical_turn DESC,ce.sequence DESC LIMIT ?"#).bind(story_id).bind(limit).fetch_all(pool).await?;
    rows.into_iter()
        .map(|row| {
            Ok(AgencyEventView {
                id: row.try_get(0)?,
                story_id: row.try_get(1)?,
                branch_id: row.try_get(2)?,
                commit_id: row.try_get(3)?,
                canonical_turn: row.try_get(4)?,
                entity_id: row.try_get(5)?,
                entity_name: row.try_get(6)?,
                action: row.try_get(7)?,
                summary: row.try_get(8)?,
                created_at: row.try_get(9)?,
            })
        })
        .collect()
}

#[derive(Debug, Serialize)]
pub struct AchievementView {
    pub id: i64,
    pub name: String,
    pub description: String,
    pub category: String,
    pub rarity: String,
    pub context: String,
    pub earned_at: String,
}

#[derive(Debug, Serialize)]
pub struct SaveView {
    pub id: String,
    pub name: String,
    pub turn: i64,
    pub chapter: i64,
    pub location: String,
    pub session_id: Option<String>,
    pub metadata: Value,
    pub created_at: String,
}

#[derive(Debug, Serialize)]
pub struct PanelsView {
    pub chapters: Vec<ChapterView>,
    pub achievements: Vec<AchievementView>,
    pub npcs: Vec<RecordView>,
    pub sessions: Vec<SessionView>,
    pub saves: Vec<SaveView>,
}

#[derive(Debug, Serialize)]
pub struct StorySnapshot {
    pub server_time: String,
    pub version: StoryVersion,
    pub story: StorySummary,
    pub character: RecordView,
    pub world: WorldView,
    pub active_session: SessionView,
    pub choices: Vec<ChoiceView>,
    pub messages: Vec<MessageView>,
    pub panels: PanelsView,
}

#[derive(Clone, Debug, Eq, PartialEq, Serialize)]
pub struct StoryVersion {
    pub turn: i64,
    pub revision: i64,
    pub story_updated_at: String,
    pub active_session_id: String,
    pub last_message_id: i64,
    pub world_updated_at: String,
    pub character_updated_at: String,
    pub npc_count: i64,
    pub npc_updated_at: String,
    pub chapter_count: i64,
    pub achievement_count: i64,
    pub latest_achievement_at: String,
    pub save_count: i64,
    pub latest_save_at: String,
    pub visual_asset_updated_at: String,
    pub visual_job_updated_at: String,
    pub active_visual_job_count: i64,
}

pub async fn list_stories(pool: &SqlitePool) -> anyhow::Result<Vec<StorySummary>> {
    let rows = sqlx::query(r#"SELECT id, name, description, genre, tone, language, is_archived, CAST(updated_at AS TEXT) AS updated_at
         FROM stories
         ORDER BY updated_at DESC"#,
    )
    .fetch_all(pool)
    .await?;
    Ok(rows.into_iter().map(story_summary_from_row).collect())
}

pub async fn update_story(
    pool: &SqlitePool,
    story_id: &str,
    update: StoryUpdate,
) -> anyhow::Result<StorySummary> {
    let prompt_affecting = update.name.is_some()
        || update.description.is_some()
        || update.genre.is_some()
        || update.tone.is_some()
        || update.language.is_some();
    let name = normalize_story_name(update.name)?;
    let description = update.description.map(|value| value.trim().to_string());
    let genre = update.genre.map(|value| value.trim().to_string());
    let tone = update.tone.map(|value| value.trim().to_string());
    let language = update
        .language
        .map(|value| value.trim().to_string())
        .filter(|value| !value.is_empty());
    let archived = update
        .is_archived
        .map(|value| if value { 1_i64 } else { 0_i64 });
    let now = chrono::Utc::now().to_rfc3339();
    let result = sqlx::query(
        r#"UPDATE stories
           SET name = COALESCE(?, name),
               description = COALESCE(?, description),
               genre = COALESCE(?, genre),
               tone = COALESCE(?, tone),
               language = COALESCE(?, language),
               is_archived = COALESCE(?, is_archived),
               revision = CASE WHEN ? = 1 THEN revision + 1 ELSE revision END,
               updated_at = ?
           WHERE id = ?"#,
    )
    .bind(name)
    .bind(description)
    .bind(genre)
    .bind(tone)
    .bind(language)
    .bind(archived)
    .bind(if prompt_affecting { 1_i64 } else { 0_i64 })
    .bind(now)
    .bind(story_id)
    .execute(pool)
    .await?;
    if result.rows_affected() == 0 {
        return Err(PublicError::not_found(
            "story_not_found",
            format!("story not found: {story_id}"),
        )
        .into());
    }
    load_story(pool, story_id).await
}

pub async fn delete_story(pool: &SqlitePool, story_id: &str) -> anyhow::Result<()> {
    let result = sqlx::query("DELETE FROM stories WHERE id = ?")
        .bind(story_id)
        .execute(pool)
        .await?;
    if result.rows_affected() == 0 {
        return Err(PublicError::not_found(
            "story_not_found",
            format!("story not found: {story_id}"),
        )
        .into());
    }
    Ok(())
}

pub async fn story_delete_plan(
    pool: &SqlitePool,
    story_id: &str,
) -> anyhow::Result<StoryDeletePlan> {
    let story = load_story(pool, story_id).await?;
    let mut counts = Vec::new();
    for table in [
        "characters",
        "world_state",
        "sessions",
        "chat_messages",
        "chapters",
        "achievements",
        "saves",
        "npcs",
        "story_visual_profiles",
        "visual_assets",
        "visual_asset_versions",
        "visual_generation_jobs",
        "turn_idempotency",
    ] {
        counts.push(StoryDeleteCount {
            table: table.to_string(),
            rows: count_story_rows(pool, table, story_id).await?,
        });
    }
    counts.push(StoryDeleteCount {
        table: "stories".to_string(),
        rows: 1,
    });
    let total_rows = counts.iter().map(|count| count.rows).sum();
    let retained_asset_files = retained_asset_files(pool, story_id).await?;
    Ok(StoryDeletePlan {
        story_id: story.id,
        story_name: story.name,
        counts,
        total_rows,
        retained_asset_files,
    })
}

pub async fn snapshot(pool: &SqlitePool, story_id: &str) -> anyhow::Result<StorySnapshot> {
    let mut tx = pool.begin().await?;
    let (story, branch_id) = load_snapshot_story(&mut tx, story_id).await?;
    let character = load_character(&mut tx, story_id).await?;
    let world = load_world(&mut tx, story_id, &branch_id).await?;
    let active_session = load_active_session(&mut tx, story_id, &branch_id).await?;
    let messages = load_messages(&mut tx, story_id, &branch_id, 120).await?;
    let choices = latest_choices(&messages, &active_session.id, world.current_turn);
    let panels = PanelsView {
        chapters: load_chapters(&mut tx, story_id, &branch_id).await?,
        achievements: load_achievements(&mut tx, story_id).await?,
        npcs: load_npcs(&mut tx, story_id, &branch_id).await?,
        sessions: load_sessions(&mut tx, story_id, &branch_id).await?,
        saves: load_saves(&mut tx, story_id, &branch_id).await?,
    };
    let version = story_version_for_branch(&mut *tx, story_id, &branch_id).await?;
    let snapshot = StorySnapshot {
        server_time: chrono::Utc::now().to_rfc3339(),
        version,
        story,
        character,
        world,
        active_session,
        choices,
        messages,
        panels,
    };
    tx.commit().await?;
    Ok(snapshot)
}

pub async fn story_version(pool: &SqlitePool, story_id: &str) -> anyhow::Result<StoryVersion> {
    let mut tx = pool.begin().await?;
    let branch_id: String = sqlx::query_scalar("SELECT active_branch_id FROM stories WHERE id = ?")
        .bind(story_id)
        .fetch_one(&mut *tx)
        .await?;
    let version = story_version_for_branch(&mut *tx, story_id, &branch_id).await?;
    tx.commit().await?;
    Ok(version)
}

pub async fn story_overview(pool: &SqlitePool, story_id: &str) -> anyhow::Result<StoryOverview> {
    let story = load_story(pool, story_id).await?;
    let row = sqlx::query(
        r#"SELECT s.active_branch_id, s.revision,
                  COALESCE(w.current_turn, 0) AS current_turn,
                  (SELECT COUNT(*) FROM story_branches b WHERE b.story_id=s.id) AS branch_count,
                  (SELECT COUNT(*) FROM chapters c WHERE c.story_id=s.id AND c.branch_id=s.active_branch_id) AS chapter_count,
                  (SELECT COUNT(*) FROM saves v WHERE v.story_id=s.id AND v.branch_id=s.active_branch_id) AS save_count,
                  (SELECT COUNT(*) FROM chat_messages m WHERE m.story_id=s.id AND m.branch_id=s.active_branch_id) AS message_count,
                  (SELECT COUNT(*) FROM visual_assets a WHERE a.story_id=s.id AND a.branch_id=s.active_branch_id) AS asset_count
           FROM stories s LEFT JOIN world_state w ON w.story_id=s.id WHERE s.id=?"#,
    )
    .bind(story_id)
    .fetch_one(pool)
    .await?;
    Ok(StoryOverview {
        story,
        active_branch_id: row.try_get("active_branch_id")?,
        revision: row.try_get("revision")?,
        current_turn: row.try_get("current_turn")?,
        branch_count: row.try_get("branch_count")?,
        chapter_count: row.try_get("chapter_count")?,
        save_count: row.try_get("save_count")?,
        message_count: row.try_get("message_count")?,
        asset_count: row.try_get("asset_count")?,
    })
}

async fn story_version_for_branch<'e, E>(
    executor: E,
    story_id: &str,
    branch_id: &str,
) -> anyhow::Result<StoryVersion>
where
    E: sqlx::Executor<'e, Database = sqlx::Sqlite>,
{
    let row = sqlx::query(r#"WITH target AS (SELECT ? AS story_id, ? AS branch_id) SELECT
           COALESCE((SELECT current_turn FROM world_state WHERE story_id=(SELECT story_id FROM target)), 0) AS turn,
           COALESCE((SELECT revision FROM stories WHERE id=(SELECT story_id FROM target)), 0) AS revision,
           COALESCE((SELECT CAST(updated_at AS TEXT) FROM stories WHERE id=(SELECT story_id FROM target)), '') AS story_updated_at,
           COALESCE((SELECT id FROM sessions WHERE story_id=(SELECT story_id FROM target) AND branch_id=(SELECT branch_id FROM target) AND ended_at IS NULL ORDER BY started_at DESC LIMIT 1),
                    (SELECT id FROM sessions WHERE story_id=(SELECT story_id FROM target) AND branch_id=(SELECT branch_id FROM target) ORDER BY started_at DESC LIMIT 1),
                    '') AS active_session_id,
           COALESCE((SELECT MAX(id) FROM chat_messages WHERE story_id=(SELECT story_id FROM target) AND branch_id=(SELECT branch_id FROM target)), 0) AS last_message_id,
           COALESCE((SELECT CAST(updated_at AS TEXT) FROM world_state WHERE story_id=(SELECT story_id FROM target)), '') AS world_updated_at,
           COALESCE((SELECT CAST(MAX(updated_at) AS TEXT) FROM characters WHERE story_id=(SELECT story_id FROM target)), '') AS character_updated_at,
           COALESCE((SELECT COUNT(*) FROM npcs WHERE story_id=(SELECT story_id FROM target)), 0) AS npc_count,
           COALESCE((SELECT CAST(MAX(updated_at) AS TEXT) FROM npcs WHERE story_id=(SELECT story_id FROM target)), '') AS npc_updated_at,
           COALESCE((SELECT COUNT(*) FROM chapters WHERE story_id=(SELECT story_id FROM target) AND branch_id=(SELECT branch_id FROM target)), 0) AS chapter_count,
           COALESCE((SELECT COUNT(*) FROM achievements WHERE story_id=(SELECT story_id FROM target)), 0) AS achievement_count,
           COALESCE((SELECT CAST(MAX(earned_at) AS TEXT) FROM achievements WHERE story_id=(SELECT story_id FROM target)), '') AS latest_achievement_at,
           COALESCE((SELECT COUNT(*) FROM saves WHERE story_id=(SELECT story_id FROM target) AND branch_id=(SELECT branch_id FROM target)), 0) AS save_count,
           COALESCE((SELECT CAST(MAX(created_at) AS TEXT) FROM saves WHERE story_id=(SELECT story_id FROM target) AND branch_id=(SELECT branch_id FROM target)), '') AS latest_save_at,
           COALESCE((SELECT CAST(MAX(updated_at) AS TEXT) FROM visual_assets WHERE story_id=(SELECT story_id FROM target) AND branch_id=(SELECT branch_id FROM target)), '') AS visual_asset_updated_at,
           COALESCE((SELECT CAST(MAX(updated_at) AS TEXT) FROM visual_generation_jobs WHERE story_id=(SELECT story_id FROM target) AND branch_id=(SELECT branch_id FROM target)), '') AS visual_job_updated_at,
           COALESCE((SELECT COUNT(*) FROM visual_generation_jobs WHERE story_id=(SELECT story_id FROM target) AND branch_id=(SELECT branch_id FROM target) AND status IN ('queued', 'running')), 0) AS active_visual_job_count"#,
    )
    .bind(story_id)
    .bind(branch_id)
    .fetch_one(executor)
    .await?;
    Ok(StoryVersion {
        turn: row.try_get("turn")?,
        revision: row.try_get("revision")?,
        story_updated_at: row.try_get("story_updated_at")?,
        active_session_id: row.try_get("active_session_id")?,
        last_message_id: row.try_get("last_message_id")?,
        world_updated_at: row.try_get("world_updated_at")?,
        character_updated_at: row.try_get("character_updated_at")?,
        npc_count: row.try_get("npc_count")?,
        npc_updated_at: row.try_get("npc_updated_at")?,
        chapter_count: row.try_get("chapter_count")?,
        achievement_count: row.try_get("achievement_count")?,
        latest_achievement_at: row.try_get("latest_achievement_at")?,
        save_count: row.try_get("save_count")?,
        latest_save_at: row.try_get("latest_save_at")?,
        visual_asset_updated_at: row.try_get("visual_asset_updated_at")?,
        visual_job_updated_at: row.try_get("visual_job_updated_at")?,
        active_visual_job_count: row.try_get("active_visual_job_count")?,
    })
}

async fn load_story(pool: &SqlitePool, story_id: &str) -> anyhow::Result<StorySummary> {
    let row = sqlx::query(r#"SELECT id, name, description, genre, tone, language, is_archived, CAST(updated_at AS TEXT) AS updated_at
         FROM stories WHERE id = ?"#,
    )
    .bind(story_id)
    .fetch_optional(pool)
    .await
    .with_context(|| format!("loading story {story_id}"))?
    .ok_or_else(|| story_not_found(story_id))?;
    Ok(story_summary_from_row(row))
}

async fn load_snapshot_story(
    conn: &mut SqliteConnection,
    story_id: &str,
) -> anyhow::Result<(StorySummary, String)> {
    let row = sqlx::query(
        r#"SELECT id, name, description, genre, tone, language, is_archived,
                  active_branch_id, CAST(updated_at AS TEXT) AS updated_at
           FROM stories WHERE id = ?"#,
    )
    .bind(story_id)
    .fetch_optional(&mut *conn)
    .await
    .with_context(|| format!("loading snapshot story {story_id}"))?
    .ok_or_else(|| story_not_found(story_id))?;
    let branch_id = row.try_get("active_branch_id")?;
    Ok((story_summary_from_row(row), branch_id))
}

fn story_not_found(story_id: &str) -> anyhow::Error {
    PublicError::not_found("story_not_found", format!("story not found: {story_id}")).into()
}

async fn count_story_rows(pool: &SqlitePool, table: &str, story_id: &str) -> anyhow::Result<i64> {
    let query = format!("SELECT COUNT(*) FROM {table} WHERE story_id = ?");
    match sqlx::query_scalar::<_, i64>(&query)
        .bind(story_id)
        .fetch_one(pool)
        .await
    {
        Ok(count) => Ok(count),
        Err(err) if err.to_string().contains("no such table") => Ok(0),
        Err(err) => Err(err.into()),
    }
}

async fn retained_asset_files(pool: &SqlitePool, story_id: &str) -> anyhow::Result<Vec<String>> {
    let mut files = Vec::new();
    for query in [
        "SELECT file_path FROM visual_assets WHERE story_id = ? AND file_path != ''",
        "SELECT file_path FROM visual_asset_versions WHERE story_id = ? AND file_path != ''",
    ] {
        match sqlx::query_scalar::<_, String>(query)
            .bind(story_id)
            .fetch_all(pool)
            .await
        {
            Ok(paths) => files.extend(paths),
            Err(err) if err.to_string().contains("no such table") => {}
            Err(err) => return Err(err.into()),
        }
    }
    files.sort();
    files.dedup();
    Ok(files)
}

async fn load_character(conn: &mut SqliteConnection, story_id: &str) -> anyhow::Result<RecordView> {
    let row = sqlx::query(
        r#"SELECT id, name, background, stats_json, traits_json, skills_json,
                inventory_json, known_recipes_json, CAST(updated_at AS TEXT) AS updated_at
         FROM characters WHERE story_id = ?"#,
    )
    .bind(story_id)
    .fetch_one(&mut *conn)
    .await?;
    Ok(RecordView {
        id: row.try_get("id")?,
        name: row.try_get("name")?,
        fields: json!({
            "background": row_string(&row, "background"),
            "stats": json_field(&row, "stats_json", json!({})),
            "traits": json_field(&row, "traits_json", json!([])),
            "skills": json_field(&row, "skills_json", json!([])),
            "inventory": json_field(&row, "inventory_json", json!([])),
            "known_recipes": json_field(&row, "known_recipes_json", json!([])),
            "updated_at": row_string(&row, "updated_at"),
        }),
    })
}

async fn load_world(
    conn: &mut SqliteConnection,
    story_id: &str,
    branch_id: &str,
) -> anyhow::Result<WorldView> {
    let row = sqlx::query(
        r#"SELECT id, current_location, current_location_id, known_locations_json, global_events_json,
                faction_standings_json, story_hooks_json, world_reactions_json,
                investigation_board_json, project_clocks_json, player_guidance_json,
                fronts_json, character_timeline_json, scene_contract_json,
                current_chapter, current_turn, CAST(updated_at AS TEXT) AS updated_at
         FROM world_state WHERE story_id = ?"#,
    )
    .bind(story_id)
    .fetch_one(&mut *conn)
    .await?;
    let spatial_regions: String=sqlx::query_scalar(r#"SELECT COALESCE(json_group_array(json_object('id',id,'name',name,'kind',region_kind,'parent_region_id',parent_region_id,'visibility',visibility)),'[]') FROM (SELECT id,name,region_kind,parent_region_id,visibility FROM regions WHERE story_id=? AND branch_id=? AND visibility IN ('public','player') ORDER BY lower(name),id)"#).bind(story_id).bind(branch_id).fetch_one(&mut *conn).await?;
    let spatial_locations: String=sqlx::query_scalar(r#"SELECT COALESCE(json_group_array(json_object('id',id,'name',canonical_name,'kind',location_kind,'region_id',region_id,'parent_location_id',parent_location_id,'description',description,'discovery_state',discovery_state)),'[]') FROM (SELECT id,canonical_name,location_kind,region_id,parent_location_id,description,discovery_state FROM locations WHERE story_id=? AND branch_id=? AND visibility IN ('public','player') AND discovery_state!='unknown' ORDER BY lower(canonical_name),id)"#).bind(story_id).bind(branch_id).fetch_one(&mut *conn).await?;
    let spatial_edges: String=sqlx::query_scalar(r#"SELECT COALESCE(json_group_array(json_object('id',id,'from_location_id',from_location_id,'to_location_id',to_location_id,'direction',direction,'travel_minutes',travel_minutes,'travel_mode',travel_mode,'bidirectional',json(CASE WHEN bidirectional=1 THEN 'true' ELSE 'false' END),'conditions',json(conditions_json))),'[]') FROM (SELECT id,from_location_id,to_location_id,direction,travel_minutes,travel_mode,bidirectional,conditions_json FROM location_edges WHERE story_id=? AND branch_id=? AND visibility IN ('public','player') ORDER BY from_location_id,to_location_id,direction,travel_mode,id)"#).bind(story_id).bind(branch_id).fetch_one(&mut *conn).await?;
    let world_time: String=sqlx::query_scalar(r#"SELECT json_object('day',day,'minute_of_day',minute_of_day,'display_text',display_text) FROM world_clocks WHERE story_id=? AND branch_id=?"#).bind(story_id).bind(branch_id).fetch_one(&mut *conn).await?;
    let weather: Option<String>=sqlx::query_scalar(r#"SELECT json_object('tracked',json('true'),'label',weather_kind,'intensity',intensity,'description',description) FROM weather_states WHERE story_id=? AND branch_id=? AND (location_id=? OR location_id IS NULL) AND visibility IN ('public','player') ORDER BY valid_from_day DESC,valid_from_minute DESC LIMIT 1"#).bind(story_id).bind(branch_id).bind(row_string(&row,"current_location_id")).fetch_optional(&mut *conn).await?;
    Ok(WorldView {
        id: row.try_get("id")?,
        current_location: row.try_get("current_location")?,
        current_location_id: row_string(&row, "current_location_id"),
        spatial_regions: serde_json::from_str(&spatial_regions).unwrap_or_else(|_| json!([])),
        spatial_locations: serde_json::from_str(&spatial_locations).unwrap_or_else(|_| json!([])),
        spatial_edges: serde_json::from_str(&spatial_edges).unwrap_or_else(|_| json!([])),
        world_time: serde_json::from_str(&world_time)
            .unwrap_or_else(|_| json!({"display_text":"Not tracked"})),
        weather: weather
            .and_then(|v| serde_json::from_str(&v).ok())
            .unwrap_or_else(|| json!({"tracked":false,"label":"Not tracked"})),
        current_chapter: row.try_get("current_chapter")?,
        current_turn: row.try_get("current_turn")?,
        known_locations: json_field(&row, "known_locations_json", json!([])),
        global_events: json_field(&row, "global_events_json", json!([])),
        faction_standings: json_field(&row, "faction_standings_json", json!({})),
        story_hooks: json_field(&row, "story_hooks_json", json!([])),
        world_reactions: json_field(&row, "world_reactions_json", json!([])),
        investigations: json_field(&row, "investigation_board_json", json!({})),
        projects: json_field(&row, "project_clocks_json", json!({})),
        guidance: json_field(&row, "player_guidance_json", json!([])),
        fronts: json_field(&row, "fronts_json", json!([])),
        timeline: json_field(&row, "character_timeline_json", json!({})),
        scene_contract: json_field(&row, "scene_contract_json", json!({})),
        updated_at: row.try_get("updated_at")?,
    })
}

async fn load_active_session(
    conn: &mut SqliteConnection,
    story_id: &str,
    branch_id: &str,
) -> anyhow::Result<SessionView> {
    let row = sqlx::query(
        r#"SELECT id, story_id, CAST(started_at AS TEXT) AS started_at,
                CAST(ended_at AS TEXT) AS ended_at, summary
         FROM sessions
         WHERE story_id = ? AND branch_id = ? AND ended_at IS NULL
         ORDER BY started_at DESC
         LIMIT 1"#,
    )
    .bind(story_id)
    .bind(branch_id)
    .fetch_optional(&mut *conn)
    .await?;
    if let Some(row) = row {
        return Ok(session_from_row(row));
    }

    let row = sqlx::query(
        r#"SELECT id, story_id, CAST(started_at AS TEXT) AS started_at,
                CAST(ended_at AS TEXT) AS ended_at, summary
         FROM sessions
         WHERE story_id = ? AND branch_id = ?
         ORDER BY started_at DESC
         LIMIT 1"#,
    )
    .bind(story_id)
    .bind(branch_id)
    .fetch_one(&mut *conn)
    .await?;
    Ok(session_from_row(row))
}

async fn load_messages(
    conn: &mut SqliteConnection,
    story_id: &str,
    branch_id: &str,
    limit: i64,
) -> anyhow::Result<Vec<MessageView>> {
    let rows = sqlx::query(
        r#"SELECT id, session_id, story_id, turn, role, content, message_type,
                metadata_json, CAST(created_at AS TEXT) AS created_at, branch_id, source_commit_id
         FROM (
           SELECT id, session_id, story_id, turn, role, content, message_type,
                  metadata_json, created_at, branch_id, source_commit_id
           FROM chat_messages
           WHERE story_id = ? AND branch_id = ?
           ORDER BY turn DESC, id DESC
           LIMIT ?
         )
         ORDER BY turn ASC, id ASC"#,
    )
    .bind(story_id)
    .bind(branch_id)
    .bind(limit)
    .fetch_all(&mut *conn)
    .await?;
    rows.into_iter().map(message_from_row).collect()
}

async fn load_chapters(
    conn: &mut SqliteConnection,
    story_id: &str,
    branch_id: &str,
) -> anyhow::Result<Vec<ChapterView>> {
    let rows = sqlx::query(
        r#"SELECT id, chapter_number, title, summary, start_turn, end_turn,
                CAST(created_at AS TEXT) AS created_at, branch_id, source_commit_id
         FROM chapters WHERE story_id = ? AND branch_id = ? ORDER BY chapter_number ASC"#,
    )
    .bind(story_id)
    .bind(branch_id)
    .fetch_all(&mut *conn)
    .await?;
    rows.into_iter()
        .map(|row| {
            Ok(ChapterView {
                id: row.try_get("id")?,
                chapter_number: row.try_get("chapter_number")?,
                title: row.try_get("title")?,
                summary: row.try_get("summary")?,
                start_turn: row.try_get("start_turn")?,
                end_turn: row.try_get("end_turn")?,
                created_at: row.try_get("created_at")?,
                branch_id: row.try_get("branch_id")?,
                source_commit_id: row.try_get("source_commit_id")?,
            })
        })
        .collect()
}

pub async fn history_page(
    pool: &SqlitePool,
    story_id: &str,
    cursor: Option<i64>,
    limit: i64,
    search: &str,
) -> anyhow::Result<HistoryPage> {
    let limit = limit.clamp(1, 100);
    let cursor = cursor.unwrap_or(i64::MAX);
    let search = search.trim();
    let rows = if search.is_empty() {
        sqlx::query(r#"SELECT id,session_id,story_id,turn,role,content,message_type,metadata_json,CAST(created_at AS TEXT) AS created_at,branch_id,source_commit_id FROM chat_messages WHERE story_id=? AND branch_id=(SELECT active_branch_id FROM stories WHERE id=?) AND id<? ORDER BY id DESC LIMIT ?"#)
            .bind(story_id).bind(story_id).bind(cursor).bind(limit + 1).fetch_all(pool).await?
    } else {
        let pattern = format!("%{search}%");
        sqlx::query(r#"SELECT m.id,m.session_id,m.story_id,m.turn,m.role,m.content,m.message_type,m.metadata_json,CAST(m.created_at AS TEXT) AS created_at,m.branch_id,m.source_commit_id FROM chat_messages m JOIN chat_messages_fts ON chat_messages_fts.rowid=m.id WHERE m.story_id=? AND m.branch_id=(SELECT active_branch_id FROM stories WHERE id=?) AND m.id<? AND chat_messages_fts.content LIKE ? ORDER BY m.id DESC LIMIT ?"#)
            .bind(story_id).bind(story_id).bind(cursor).bind(pattern).bind(limit + 1).fetch_all(pool).await?
    };
    let has_more = rows.len() as i64 > limit;
    let mut items: Vec<MessageView> = rows
        .into_iter()
        .take(limit as usize)
        .map(message_from_row)
        .collect::<anyhow::Result<_>>()?;
    let next_cursor = if has_more {
        items.last().map(|item| item.id)
    } else {
        None
    };
    items.reverse();
    Ok(HistoryPage { items, next_cursor })
}

pub async fn chapter_page(
    pool: &SqlitePool,
    story_id: &str,
    cursor: Option<i64>,
    limit: i64,
    search: &str,
) -> anyhow::Result<ChapterPage> {
    let limit = limit.clamp(1, 100);
    let cursor = cursor.unwrap_or(i64::MAX);
    let search = search.trim();
    let rows = if search.is_empty() {
        sqlx::query(r#"SELECT id,chapter_number,title,summary,start_turn,end_turn,CAST(created_at AS TEXT) AS created_at,branch_id,source_commit_id FROM chapters WHERE story_id=? AND branch_id=(SELECT active_branch_id FROM stories WHERE id=?) AND chapter_number<? ORDER BY chapter_number DESC LIMIT ?"#)
            .bind(story_id).bind(story_id).bind(cursor).bind(limit + 1).fetch_all(pool).await?
    } else {
        let pattern = format!("%{search}%");
        sqlx::query(r#"SELECT c.id,c.chapter_number,c.title,c.summary,c.start_turn,c.end_turn,CAST(c.created_at AS TEXT) AS created_at,c.branch_id,c.source_commit_id FROM chapters c JOIN chapters_fts ON chapters_fts.rowid=c.id WHERE c.story_id=? AND c.branch_id=(SELECT active_branch_id FROM stories WHERE id=?) AND c.chapter_number<? AND (chapters_fts.title LIKE ? OR chapters_fts.summary LIKE ?) ORDER BY c.chapter_number DESC LIMIT ?"#)
            .bind(story_id).bind(story_id).bind(cursor).bind(&pattern).bind(&pattern).bind(limit + 1).fetch_all(pool).await?
    };
    let has_more = rows.len() as i64 > limit;
    let mut items = Vec::new();
    for row in rows.into_iter().take(limit as usize) {
        items.push(ChapterView {
            id: row.try_get("id")?,
            chapter_number: row.try_get("chapter_number")?,
            title: row.try_get("title")?,
            summary: row.try_get("summary")?,
            start_turn: row.try_get("start_turn")?,
            end_turn: row.try_get("end_turn")?,
            created_at: row.try_get("created_at")?,
            branch_id: row.try_get("branch_id")?,
            source_commit_id: row.try_get("source_commit_id")?,
        });
    }
    let next_cursor = if has_more {
        items.last().map(|item| item.chapter_number)
    } else {
        None
    };
    items.reverse();
    Ok(ChapterPage { items, next_cursor })
}

pub async fn export_story(
    pool: &SqlitePool,
    story_id: &str,
    format: &str,
) -> anyhow::Result<StoryExport> {
    let StoryExportSource {
        story,
        active_branch_id,
        messages,
        chapters,
        safe_name,
    } = load_story_export_source(pool, story_id).await?;
    if format.eq_ignore_ascii_case("json") {
        let payload = json!({"story": story, "active_branch_id": active_branch_id, "chapters": chapters, "messages": messages});
        return Ok(StoryExport {
            format: "json".into(),
            filename: format!("{safe_name}-history.json"),
            content: serde_json::to_string_pretty(&payload)?,
            encoding: "utf-8".into(),
            content_type: "application/json".into(),
        });
    }
    if format.eq_ignore_ascii_case("replay") {
        let visuals = sqlx::query(r#"SELECT id,kind,subject,url,branch_id,source_commit_id FROM visual_assets WHERE story_id=? AND branch_id=? AND status='ready' ORDER BY kind,subject,id"#).bind(story_id).bind(&active_branch_id).fetch_all(pool).await?.into_iter().map(|row| json!({"id":row.get::<String,_>(0),"kind":row.get::<String,_>(1),"subject":row.get::<String,_>(2),"url":row.get::<String,_>(3),"branch_id":row.get::<String,_>(4),"source_commit_id":row.get::<String,_>(5)})).collect::<Vec<_>>();
        let audio = sqlx::query(r#"SELECT id,source_message_id,segment_index,segment_kind,language_tag,duration_ms,branch_id,source_commit_id FROM audio_assets WHERE story_id=? AND branch_id=? AND status='ready' ORDER BY source_message_id,segment_index"#).bind(story_id).bind(&active_branch_id).fetch_all(pool).await?.into_iter().map(|row| { let id:String=row.get(0); json!({"id":id,"url":format!("/api/audio/{id}"),"source_message_id":row.get::<i64,_>(1),"segment_index":row.get::<i64,_>(2),"segment_kind":row.get::<String,_>(3),"language_tag":row.get::<String,_>(4),"duration_ms":row.get::<i64,_>(5),"branch_id":row.get::<String,_>(6),"source_commit_id":row.get::<String,_>(7)}) }).collect::<Vec<_>>();
        let payload = json!({"format":"oneday-replay-v1","story":story,"active_branch_id":active_branch_id,"chapters":chapters,"messages":messages,"visual_assets":visuals,"audio_assets":audio});
        return Ok(StoryExport {
            format: "replay".into(),
            filename: format!("{safe_name}-replay.json"),
            content: serde_json::to_string_pretty(&payload)?,
            encoding: "utf-8".into(),
            content_type: "application/json".into(),
        });
    }
    if format.eq_ignore_ascii_case("epub") {
        return tokio::task::spawn_blocking(move || {
            let bytes = build_epub(&story, &active_branch_id, &chapters, &messages)?;
            Ok(StoryExport {
                format: "epub".into(),
                filename: format!("{safe_name}.epub"),
                content: BASE64.encode(bytes),
                encoding: "base64".into(),
                content_type: "application/epub+zip".into(),
            })
        })
        .await
        .context("joining EPUB export renderer")?;
    }
    tokio::task::spawn_blocking(move || {
        build_markdown_export(story, active_branch_id, chapters, messages, safe_name)
    })
    .await
    .context("joining Markdown export renderer")
}

struct StoryExportSource {
    story: StorySummary,
    active_branch_id: String,
    messages: Vec<MessageView>,
    chapters: Vec<ChapterView>,
    safe_name: String,
}

async fn load_story_export_source(
    pool: &SqlitePool,
    story_id: &str,
) -> anyhow::Result<StoryExportSource> {
    let story = load_story(pool, story_id).await?;
    let active_branch_id: String =
        sqlx::query_scalar("SELECT active_branch_id FROM stories WHERE id=?")
            .bind(story_id)
            .fetch_one(pool)
            .await?;
    let mut messages = Vec::new();
    let mut cursor = None;
    loop {
        let page = history_page(pool, story_id, cursor, 100, "").await?;
        messages.extend(page.items);
        if page.next_cursor.is_none() {
            break;
        }
        cursor = page.next_cursor;
    }
    messages.sort_by_key(|message| message.id);
    let mut chapters = Vec::new();
    let mut chapter_cursor = None;
    loop {
        let page = chapter_page(pool, story_id, chapter_cursor, 100, "").await?;
        chapters.extend(page.items);
        if page.next_cursor.is_none() {
            break;
        }
        chapter_cursor = page.next_cursor;
    }
    chapters.sort_by_key(|chapter| chapter.chapter_number);
    let safe_name: String = story
        .name
        .chars()
        .map(|c| if c.is_alphanumeric() { c } else { '-' })
        .collect::<String>()
        .to_lowercase();
    Ok(StoryExportSource {
        story,
        active_branch_id,
        messages,
        chapters,
        safe_name,
    })
}

pub async fn export_story_epub(
    pool: &SqlitePool,
    story_id: &str,
) -> anyhow::Result<(String, Vec<u8>)> {
    let source = load_story_export_source(pool, story_id).await?;
    tokio::task::spawn_blocking(move || {
        let bytes = build_epub(
            &source.story,
            &source.active_branch_id,
            &source.chapters,
            &source.messages,
        )?;
        Ok((format!("{}.epub", source.safe_name), bytes))
    })
    .await
    .context("joining binary EPUB renderer")?
}

fn build_markdown_export(
    story: StorySummary,
    active_branch_id: String,
    chapters: Vec<ChapterView>,
    messages: Vec<MessageView>,
    safe_name: String,
) -> StoryExport {
    let labels = export_labels(&story.language);
    let mut content = format!(
        "# {}\n\n{}: `{}`\n\n",
        story.name, labels.branch, active_branch_id
    );
    if !chapters.is_empty() {
        content.push_str(&format!("## {}\n\n", labels.chapters));
        for chapter in chapters {
            let turn_range = export_turn_range(labels, chapter.start_turn, chapter.end_turn);
            content.push_str(&format!(
                "### {}\n\n{}\n\n{}\n\n",
                chapter.title, turn_range, chapter.summary
            ));
        }
        content.push_str(&format!("## {}\n\n", labels.transcript));
    }
    for message in messages {
        content.push_str(&format!(
            "## {} {} — {}\n\n{}\n\n",
            labels.turn,
            message.turn,
            export_role(labels.locale, &message.role),
            message.content
        ));
    }
    StoryExport {
        format: "markdown".into(),
        filename: format!("{safe_name}-history.md"),
        content,
        encoding: "utf-8".into(),
        content_type: "text/markdown".into(),
    }
}

fn build_epub(
    story: &StorySummary,
    branch_id: &str,
    chapters: &[ChapterView],
    messages: &[MessageView],
) -> anyhow::Result<Vec<u8>> {
    let labels = export_labels(&story.language);
    let document_language = export_document_language(&story.language, labels.locale);
    let mut body = format!(
        "<h1>{}</h1><p>{}: <code>{}</code></p>",
        xml_escape(&story.name),
        labels.branch,
        xml_escape(branch_id)
    );
    for chapter in chapters {
        body.push_str(&format!(
            "<section><h2>{}</h2><p>{}</p></section>",
            xml_escape(&chapter.title),
            xml_escape(&chapter.summary)
        ));
    }
    body.push_str(&format!("<section><h2>{}</h2>", labels.transcript));
    for message in messages {
        body.push_str(&format!(
            "<article><h3>{} {} — {}</h3><p>{}</p></article>",
            labels.turn,
            message.turn,
            xml_escape(export_role(labels.locale, &message.role)),
            xml_escape(&message.content)
        ));
    }
    body.push_str("</section>");
    let xhtml = format!(
        r#"<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE html><html xmlns="http://www.w3.org/1999/xhtml" lang="{}"><head><meta charset="utf-8"/><title>{}</title></head><body>{}</body></html>"#,
        document_language,
        xml_escape(&story.name),
        body
    );
    let nav = format!(
        r#"<?xml version="1.0" encoding="UTF-8"?><html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops" lang="{}"><head><title>{}</title></head><body><nav epub:type="toc"><ol><li><a href="content.xhtml">{}</a></li></ol></nav></body></html>"#,
        document_language,
        labels.navigation,
        xml_escape(&story.name)
    );
    let opf = format!(
        r#"<?xml version="1.0" encoding="UTF-8"?><package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="book-id"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:identifier id="book-id">urn:oneday:{}</dc:identifier><dc:title>{}</dc:title><dc:language>{}</dc:language><meta property="dcterms:modified">{}</meta></metadata><manifest><item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/><item id="content" href="content.xhtml" media-type="application/xhtml+xml"/></manifest><spine><itemref idref="content"/></spine></package>"#,
        xml_escape(&story.id),
        xml_escape(&story.name),
        document_language,
        chrono::Utc::now().format("%Y-%m-%dT%H:%M:%SZ")
    );
    let cursor = Cursor::new(Vec::new());
    let mut zip = ZipWriter::new(cursor);
    zip.start_file(
        "mimetype",
        SimpleFileOptions::default().compression_method(CompressionMethod::Stored),
    )?;
    zip.write_all(b"application/epub+zip")?;
    let compressed = SimpleFileOptions::default().compression_method(CompressionMethod::Deflated);
    zip.start_file("META-INF/container.xml", compressed)?;
    zip.write_all(br#"<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>"#)?;
    zip.start_file("OEBPS/content.opf", compressed)?;
    zip.write_all(opf.as_bytes())?;
    zip.start_file("OEBPS/nav.xhtml", compressed)?;
    zip.write_all(nav.as_bytes())?;
    zip.start_file("OEBPS/content.xhtml", compressed)?;
    zip.write_all(xhtml.as_bytes())?;
    Ok(zip.finish()?.into_inner())
}

#[derive(Clone, Copy)]
struct ExportLabels {
    locale: &'static str,
    branch: &'static str,
    chapters: &'static str,
    turns: &'static str,
    current: &'static str,
    transcript: &'static str,
    turn: &'static str,
    navigation: &'static str,
}

fn export_labels(language: &str) -> ExportLabels {
    let language = language.trim().to_ascii_lowercase();
    let italian = language == "it"
        || language.starts_with("it-")
        || language == "italian"
        || language.starts_with("italiano");
    if italian {
        return ExportLabels {
            locale: "it",
            branch: "Ramo",
            chapters: "Capitoli",
            turns: "Turni",
            current: "in corso",
            transcript: "Trascrizione",
            turn: "Turno",
            navigation: "Navigazione",
        };
    }
    ExportLabels {
        locale: "en",
        branch: "Branch",
        chapters: "Chapters",
        turns: "Turns",
        current: "current",
        transcript: "Transcript",
        turn: "Turn",
        navigation: "Navigation",
    }
}

fn export_document_language<'a>(language: &'a str, fallback: &'static str) -> &'a str {
    let language = language.trim();
    let mut subtags = language.split('-');
    let Some(primary) = subtags.next() else {
        return fallback;
    };
    let primary_valid = (2..=3).contains(&primary.len())
        && primary
            .bytes()
            .all(|character| character.is_ascii_alphabetic());
    let remaining_valid = subtags.all(|subtag| {
        (1..=8).contains(&subtag.len())
            && subtag
                .bytes()
                .all(|character| character.is_ascii_alphanumeric())
    });
    if primary_valid && remaining_valid {
        language
    } else {
        fallback
    }
}

fn export_role<'a>(locale: &str, role: &'a str) -> &'a str {
    match (locale, role) {
        ("it", "assistant") => "Narratore",
        ("it", "user") => "Giocatore",
        ("it", "system") => "Sistema",
        ("en", "assistant") => "Narrator",
        ("en", "user") => "Player",
        ("en", "system") => "System",
        _ => role,
    }
}

fn export_turn_range(labels: ExportLabels, start_turn: i64, end_turn: Option<i64>) -> String {
    match (labels.locale, end_turn) {
        ("it", Some(end_turn)) => format!("{} {start_turn}–{end_turn}", labels.turns),
        ("it", None) => format!("Dal turno {start_turn} ({})", labels.current),
        (_, Some(end_turn)) => format!("{} {start_turn}–{end_turn}", labels.turns),
        (_, None) => format!("{} {start_turn}–{}", labels.turns, labels.current),
    }
}

fn xml_escape(value: &str) -> String {
    value
        .replace('&', "&amp;")
        .replace('<', "&lt;")
        .replace('>', "&gt;")
        .replace('"', "&quot;")
        .replace('\'', "&apos;")
}

async fn load_achievements(
    conn: &mut SqliteConnection,
    story_id: &str,
) -> anyhow::Result<Vec<AchievementView>> {
    let rows = sqlx::query(
        r#"SELECT id, name, description, category, rarity, context,
                CAST(earned_at AS TEXT) AS earned_at
         FROM achievements WHERE story_id = ? ORDER BY earned_at ASC"#,
    )
    .bind(story_id)
    .fetch_all(&mut *conn)
    .await?;
    rows.into_iter()
        .map(|row| {
            Ok(AchievementView {
                id: row.try_get("id")?,
                name: row.try_get("name")?,
                description: row.try_get("description")?,
                category: row.try_get("category")?,
                rarity: row.try_get("rarity")?,
                context: row.try_get("context")?,
                earned_at: row.try_get("earned_at")?,
            })
        })
        .collect()
}

async fn load_npcs(
    conn: &mut SqliteConnection,
    story_id: &str,
    branch_id: &str,
) -> anyhow::Result<Vec<RecordView>> {
    let rows = sqlx::query(r#"SELECT id, COALESCE(NULLIF(canonical_entity_id,''),id) AS canonical_entity_id, name, role, appearance, personality_json, relationship_json,
                discovery_json,
                disposition, is_alive,
                first_appeared_turn, last_seen_turn, can_help, CAST(updated_at AS TEXT) AS updated_at,
                COALESCE((SELECT json_group_array(json_object('predicate',f.predicate,'value',json(f.object_json),'confidence',f.confidence))
                  FROM character_facts f WHERE f.story_id=npcs.story_id AND f.branch_id=?
                    AND f.subject_entity_id=COALESCE(NULLIF(npcs.canonical_entity_id,''),npcs.id)
                    AND f.visibility IN ('public','player') AND f.retracts_fact_id IS NULL
                    AND NOT EXISTS (SELECT 1 FROM character_facts newer
                      WHERE newer.story_id=f.story_id AND newer.branch_id=f.branch_id
                        AND (newer.retracts_fact_id=f.id OR newer.supersedes_fact_id=f.id))),'[]') AS known_facts_json
         FROM npcs WHERE story_id = ? ORDER BY last_seen_turn DESC, name ASC"#,
    )
    .bind(branch_id)
    .bind(story_id)
    .fetch_all(&mut *conn)
    .await?;
    Ok(rows
        .into_iter()
        .map(|row| {
            let discovery = json_field(&row, "discovery_json", json!({}));
            let discovery_stage = value_string(&discovery, "stage");
            let profile_completeness = value_i64(&discovery, "profile_completeness");
            let visual_completeness = value_i64(&discovery, "visual_completeness");
            let visual_readiness = value_string(&discovery, "visual_readiness");
            let public_label = value_string(&discovery, "public_label");
            RecordView {
                id: row_string(&row, "canonical_entity_id"),
                name: row_string(&row, "name"),
                fields: json!({
                "role": row_string(&row, "role"),
                "compatibility_npc_id": row_string(&row, "id"),
				"known_facts": json_field(&row, "known_facts_json", json!([])),
                "appearance": row_string(&row, "appearance"),
                "personality": json_field(&row, "personality_json", json!({})),
                "relationship": json_field(&row, "relationship_json", json!({})),
                "discovery": discovery,
                "discovery_stage": discovery_stage,
                "profile_completeness": profile_completeness,
                "visual_completeness": visual_completeness,
                "visual_readiness": visual_readiness,
                "public_label": public_label,
                "disposition": row.try_get::<i64, _>("disposition").unwrap_or_default(),
                "is_alive": row.try_get::<i64, _>("is_alive").unwrap_or(1) != 0,
                "first_appeared_turn": row.try_get::<i64, _>("first_appeared_turn").unwrap_or_default(),
                "last_seen_turn": row.try_get::<i64, _>("last_seen_turn").unwrap_or_default(),
                "can_help": row.try_get::<i64, _>("can_help").unwrap_or_default() != 0,
                "updated_at": row_string(&row, "updated_at"),
                }),
            }
        })
        .collect())
}

async fn load_sessions(
    conn: &mut SqliteConnection,
    story_id: &str,
    branch_id: &str,
) -> anyhow::Result<Vec<SessionView>> {
    let rows = sqlx::query(
        r#"SELECT id, story_id, CAST(started_at AS TEXT) AS started_at,
                CAST(ended_at AS TEXT) AS ended_at, summary
         FROM sessions WHERE story_id = ? AND branch_id = ? ORDER BY started_at DESC"#,
    )
    .bind(story_id)
    .bind(branch_id)
    .fetch_all(&mut *conn)
    .await?;
    Ok(rows.into_iter().map(session_from_row).collect())
}

async fn load_saves(
    conn: &mut SqliteConnection,
    story_id: &str,
    branch_id: &str,
) -> anyhow::Result<Vec<SaveView>> {
    let rows = sqlx::query(
        r#"SELECT id, name, turn, chapter, location, session_id, metadata_json,
                CAST(created_at AS TEXT) AS created_at
         FROM saves WHERE story_id = ? AND branch_id = ? ORDER BY created_at DESC"#,
    )
    .bind(story_id)
    .bind(branch_id)
    .fetch_all(&mut *conn)
    .await?;
    rows.into_iter()
        .map(|row| {
            Ok(SaveView {
                id: row.try_get("id")?,
                name: row.try_get("name")?,
                turn: row.try_get("turn")?,
                chapter: row.try_get("chapter")?,
                location: row.try_get("location")?,
                session_id: row.try_get("session_id")?,
                metadata: json_field(&row, "metadata_json", json!({})),
                created_at: row.try_get("created_at")?,
            })
        })
        .collect()
}

pub async fn story_saves(pool: &SqlitePool, story_id: &str) -> anyhow::Result<Vec<SaveView>> {
    let mut conn = pool.acquire().await?;
    let branch_id: String = sqlx::query_scalar("SELECT active_branch_id FROM stories WHERE id = ?")
        .bind(story_id)
        .fetch_one(&mut *conn)
        .await?;
    load_saves(&mut conn, story_id, &branch_id).await
}

fn latest_choices(
    messages: &[MessageView],
    active_session_id: &str,
    current_turn: i64,
) -> Vec<ChoiceView> {
    for message in messages.iter().rev() {
        if message.role != "assistant"
            || message.session_id != active_session_id
            || message.turn > current_turn
            || current_turn.saturating_sub(message.turn) > 1
        {
            continue;
        }
        let output = message.metadata.get("output").unwrap_or(&Value::Null);
        if let Some(items) = output.get("choices_data").and_then(Value::as_array) {
            let choices: Vec<_> = items
                .iter()
                .enumerate()
                .filter_map(|(idx, item)| choice_from_value(idx, item))
                .collect();
            if !choices.is_empty() {
                return choices;
            }
        }
        let simple = output
            .get("choices")
            .or_else(|| message.metadata.get("choices"))
            .and_then(Value::as_array);
        if let Some(items) = simple {
            let choices: Vec<_> = items
                .iter()
                .enumerate()
                .filter_map(|(idx, item)| {
                    item.as_str().map(|text| ChoiceView {
                        id: (idx + 1) as i64,
                        text: text.to_string(),
                        intent: None,
                        risk: None,
                        scope: None,
                        certainty: None,
                        related_stats: vec![],
                    })
                })
                .collect();
            if !choices.is_empty() {
                return choices;
            }
        }
    }
    vec![]
}

fn choice_from_value(idx: usize, item: &Value) -> Option<ChoiceView> {
    let text = item.get("text").and_then(Value::as_str)?.trim();
    if text.is_empty() {
        return None;
    }
    Some(ChoiceView {
        id: item
            .get("id")
            .and_then(Value::as_i64)
            .filter(|id| *id > 0)
            .unwrap_or((idx + 1) as i64),
        text: text.to_string(),
        intent: optional_string(item, "intent"),
        risk: optional_string(item, "risk"),
        scope: optional_string(item, "scope"),
        certainty: optional_string(item, "certainty"),
        related_stats: item
            .get("related_stats")
            .and_then(Value::as_array)
            .map(|items| {
                items
                    .iter()
                    .filter_map(Value::as_str)
                    .map(ToString::to_string)
                    .collect()
            })
            .unwrap_or_default(),
    })
}

fn optional_string(value: &Value, key: &str) -> Option<String> {
    value
        .get(key)
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|s| !s.is_empty())
        .map(ToString::to_string)
}

fn story_summary_from_row(row: sqlx::sqlite::SqliteRow) -> StorySummary {
    StorySummary {
        id: row_string(&row, "id"),
        name: row_string(&row, "name"),
        description: row_string(&row, "description"),
        genre: row_string(&row, "genre"),
        tone: row_string(&row, "tone"),
        language: row_string(&row, "language"),
        is_archived: row.try_get::<i64, _>("is_archived").unwrap_or_default() != 0,
        updated_at: row_string(&row, "updated_at"),
    }
}

fn normalize_story_name(value: Option<String>) -> anyhow::Result<Option<String>> {
    match value {
        Some(raw) => {
            let name = raw.trim().to_string();
            if name.is_empty() {
                return Err(PublicError::bad_request(
                    "story_name_required",
                    "story name cannot be empty",
                )
                .into());
            }
            Ok(Some(name))
        }
        None => Ok(None),
    }
}

fn session_from_row(row: sqlx::sqlite::SqliteRow) -> SessionView {
    SessionView {
        id: row_string(&row, "id"),
        story_id: row_string(&row, "story_id"),
        started_at: row_string(&row, "started_at"),
        ended_at: row.try_get("ended_at").ok().flatten(),
        summary: row_string(&row, "summary"),
    }
}

fn message_from_row(row: sqlx::sqlite::SqliteRow) -> anyhow::Result<MessageView> {
    Ok(MessageView {
        id: row.try_get("id")?,
        session_id: row.try_get("session_id")?,
        story_id: row.try_get("story_id")?,
        turn: row.try_get("turn")?,
        role: row.try_get("role")?,
        content: row.try_get("content")?,
        message_type: row.try_get("message_type")?,
        metadata: json_field(&row, "metadata_json", json!({})),
        created_at: row.try_get("created_at")?,
        branch_id: row.try_get("branch_id")?,
        source_commit_id: row.try_get("source_commit_id")?,
    })
}

fn row_string(row: &sqlx::sqlite::SqliteRow, key: &str) -> String {
    row.try_get::<Option<String>, _>(key)
        .ok()
        .flatten()
        .unwrap_or_default()
}

fn json_field(row: &sqlx::sqlite::SqliteRow, key: &str, fallback: Value) -> Value {
    let raw = row_string(row, key);
    if raw.trim().is_empty() || raw.trim() == "null" {
        return fallback;
    }
    serde_json::from_str(&raw).unwrap_or(fallback)
}

fn value_string(value: &Value, key: &str) -> String {
    value
        .get(key)
        .and_then(Value::as_str)
        .unwrap_or_default()
        .to_string()
}

fn value_i64(value: &Value, key: &str) -> i64 {
    value.get(key).and_then(Value::as_i64).unwrap_or_default()
}

#[cfg(test)]
mod tests {
    use super::*;
    use sqlx::sqlite::SqlitePoolOptions;

    #[test]
    fn export_chrome_follows_story_language_without_changing_machine_fields() {
        let story = StorySummary {
            id: "story-it".into(),
            name: "La soglia".into(),
            description: String::new(),
            genre: String::new(),
            tone: String::new(),
            language: "it-IT".into(),
            is_archived: false,
            updated_at: String::new(),
        };
        let chapters = vec![ChapterView {
            id: 1,
            chapter_number: 1,
            title: "Il porto".into(),
            summary: "Una notte di nebbia.".into(),
            start_turn: 1,
            end_turn: None,
            created_at: String::new(),
            branch_id: "branch-main".into(),
            source_commit_id: "commit-main".into(),
        }];
        let messages = vec![MessageView {
            id: 1,
            session_id: "session-main".into(),
            story_id: story.id.clone(),
            turn: 1,
            role: "assistant".into(),
            content: "La campana suona.".into(),
            message_type: "narrative".into(),
            metadata: json!({}),
            created_at: String::new(),
            branch_id: "branch-main".into(),
            source_commit_id: "commit-main".into(),
        }];

        let markdown = build_markdown_export(
            story.clone(),
            "branch-main".into(),
            chapters.clone(),
            messages.clone(),
            "la-soglia".into(),
        );
        assert!(markdown.content.contains("Ramo: `branch-main`"));
        assert!(markdown.content.contains("## Capitoli"));
        assert!(markdown.content.contains("Dal turno 1 (in corso)"));
        assert!(markdown.content.contains("## Turno 1 — Narratore"));

        let bytes = build_epub(&story, "branch-main", &chapters, &messages).expect("EPUB");
        let mut archive = zip::ZipArchive::new(Cursor::new(bytes)).expect("valid EPUB");
        let mut content = String::new();
        std::io::Read::read_to_string(
            &mut archive.by_name("OEBPS/content.xhtml").expect("content"),
            &mut content,
        )
        .expect("read content");
        assert!(content.contains("lang=\"it-IT\""));
        assert!(content.contains("<h2>Trascrizione</h2>"));
        assert!(content.contains("Turno 1 — Narratore"));
    }

    #[test]
    fn export_locale_normalizes_supported_variants_and_falls_back_to_english() {
        assert_eq!(export_labels("italiano").locale, "it");
        assert_eq!(export_labels("it-CH").locale, "it");
        assert_eq!(export_labels("en-US").locale, "en");
        assert_eq!(export_labels("fr-FR").locale, "en");
        assert_eq!(export_document_language("fr-FR", "en"), "fr-FR");
        assert_eq!(export_document_language("it-IT", "it"), "it-IT");
        assert_eq!(export_document_language("italiano", "it"), "it");
    }

    #[test]
    fn epub_preserves_valid_non_interface_story_language_metadata() {
        let story = StorySummary {
            id: "story-fr".into(),
            name: "Le seuil".into(),
            description: String::new(),
            genre: String::new(),
            tone: String::new(),
            language: "fr-FR".into(),
            is_archived: false,
            updated_at: String::new(),
        };

        let bytes = build_epub(&story, "branch-main", &[], &[]).expect("EPUB");
        let mut archive = zip::ZipArchive::new(Cursor::new(bytes)).expect("valid EPUB");
        for (name, marker) in [
            ("OEBPS/content.xhtml", "lang=\"fr-FR\""),
            ("OEBPS/nav.xhtml", "lang=\"fr-FR\""),
            ("OEBPS/content.opf", "<dc:language>fr-FR</dc:language>"),
        ] {
            let mut document = String::new();
            std::io::Read::read_to_string(
                &mut archive.by_name(name).expect("EPUB document"),
                &mut document,
            )
            .expect("read EPUB document");
            assert!(document.contains(marker), "{name} did not contain {marker}");
        }
    }

    async fn story_pool() -> SqlitePool {
        let pool = SqlitePoolOptions::new()
            .max_connections(1)
            .connect("sqlite::memory:")
            .await
            .expect("memory sqlite pool");
        sqlx::query(
            r#"CREATE TABLE stories (
                id TEXT PRIMARY KEY,
                name TEXT NOT NULL,
                description TEXT NOT NULL DEFAULT '',
                genre TEXT NOT NULL DEFAULT '',
                tone TEXT NOT NULL DEFAULT '',
                language TEXT NOT NULL DEFAULT '',
                is_archived INTEGER NOT NULL DEFAULT 0,
                revision INTEGER NOT NULL DEFAULT 0,
				active_branch_id TEXT NOT NULL DEFAULT 'branch-main',
                updated_at TEXT NOT NULL
            )"#,
        )
        .execute(&pool)
        .await
        .expect("create stories table");
        sqlx::query(
            r#"INSERT INTO stories (
                id, name, description, genre, tone, language, is_archived, revision, updated_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)"#,
        )
        .bind("story-1")
        .bind("Old Name")
        .bind("Old description")
        .bind("mystery")
        .bind("tense")
        .bind("en")
        .bind(0_i64)
        .bind(7_i64)
        .bind("2026-01-01T00:00:00Z")
        .execute(&pool)
        .await
        .expect("insert story");
        create_story_version_tables(&pool).await;
        pool
    }

    #[tokio::test]
    async fn npc_facts_are_scoped_to_story_and_active_branch() {
        let pool = SqlitePoolOptions::new()
            .max_connections(1)
            .connect("sqlite::memory:")
            .await
            .expect("memory sqlite pool");
        sqlx::query(
            r#"CREATE TABLE npcs (
                id TEXT PRIMARY KEY, story_id TEXT NOT NULL, canonical_entity_id TEXT NOT NULL,
                name TEXT NOT NULL, role TEXT NOT NULL DEFAULT '', appearance TEXT NOT NULL DEFAULT '',
                personality_json TEXT NOT NULL DEFAULT '{}', relationship_json TEXT NOT NULL DEFAULT '{}',
                discovery_json TEXT NOT NULL DEFAULT '{}', disposition INTEGER NOT NULL DEFAULT 0,
                is_alive INTEGER NOT NULL DEFAULT 1, first_appeared_turn INTEGER NOT NULL DEFAULT 0,
                last_seen_turn INTEGER NOT NULL DEFAULT 0, can_help INTEGER NOT NULL DEFAULT 0,
                updated_at TEXT NOT NULL DEFAULT ''
            );
            CREATE TABLE character_facts (
                id TEXT PRIMARY KEY, story_id TEXT NOT NULL, branch_id TEXT NOT NULL,
                subject_entity_id TEXT NOT NULL, predicate TEXT NOT NULL, object_json TEXT NOT NULL,
                confidence REAL NOT NULL, visibility TEXT NOT NULL, retracts_fact_id TEXT,
                supersedes_fact_id TEXT
            )"#,
        )
        .execute(&pool)
        .await
        .expect("create NPC fact fixtures");
        sqlx::query("INSERT INTO npcs (id,story_id,canonical_entity_id,name) VALUES ('npc-1','story-1','entity-1','Mara')")
            .execute(&pool).await.expect("insert NPC");
        sqlx::query("INSERT INTO character_facts VALUES ('fact-main','story-1','branch-main','entity-1','role','\"guide\"',1.0,'player',NULL,NULL)")
            .execute(&pool).await.expect("insert main fact");
        sqlx::query("INSERT INTO character_facts VALUES ('retract-other','story-2','branch-main','entity-1','role','\"other\"',1.0,'player','fact-main',NULL)")
            .execute(&pool).await.expect("insert cross-story retraction");
        sqlx::query("INSERT INTO character_facts VALUES ('supersede-alt','story-1','branch-alt','entity-1','role','\"alternate\"',1.0,'player',NULL,'fact-main')")
            .execute(&pool).await.expect("insert alternate-branch supersession");

        let mut conn = pool.acquire().await.expect("acquire connection");
        let records = load_npcs(&mut conn, "story-1", "branch-main")
            .await
            .expect("load NPCs");
        let facts = records[0].fields["known_facts"]
            .as_array()
            .expect("known facts array");
        assert_eq!(facts.len(), 1);
        assert_eq!(facts[0]["predicate"], "role");
        assert_eq!(facts[0]["value"], "guide");
    }

    async fn create_story_version_tables(pool: &SqlitePool) {
        for statement in [
            "CREATE TABLE world_state (story_id TEXT NOT NULL, current_turn INTEGER NOT NULL DEFAULT 0, updated_at TEXT NOT NULL DEFAULT '')",
            "CREATE TABLE sessions (id TEXT NOT NULL, story_id TEXT NOT NULL, started_at TEXT NOT NULL DEFAULT '', ended_at TEXT, branch_id TEXT NOT NULL DEFAULT 'branch-main')",
            "CREATE TABLE chat_messages (id INTEGER PRIMARY KEY AUTOINCREMENT, story_id TEXT NOT NULL, session_id TEXT NOT NULL, turn INTEGER NOT NULL DEFAULT 0, role TEXT NOT NULL, content TEXT NOT NULL DEFAULT '', message_type TEXT NOT NULL DEFAULT 'narrative', metadata_json TEXT NOT NULL DEFAULT '{}', created_at TEXT NOT NULL DEFAULT '', branch_id TEXT NOT NULL DEFAULT 'branch-main', source_commit_id TEXT NOT NULL DEFAULT 'commit-main')",
            "CREATE TABLE characters (story_id TEXT NOT NULL, updated_at TEXT NOT NULL DEFAULT '')",
            "CREATE TABLE npcs (story_id TEXT NOT NULL, updated_at TEXT NOT NULL DEFAULT '')",
            "CREATE TABLE chapters (story_id TEXT NOT NULL, branch_id TEXT NOT NULL DEFAULT 'branch-main', source_commit_id TEXT NOT NULL DEFAULT 'commit-main')",
            "CREATE TABLE achievements (story_id TEXT NOT NULL, earned_at TEXT NOT NULL DEFAULT '')",
            "CREATE TABLE saves (id TEXT NOT NULL DEFAULT 'save-default', story_id TEXT NOT NULL, name TEXT NOT NULL DEFAULT '', turn INTEGER NOT NULL DEFAULT 0, chapter INTEGER NOT NULL DEFAULT 0, location TEXT NOT NULL DEFAULT '', session_id TEXT NOT NULL DEFAULT '', metadata_json TEXT NOT NULL DEFAULT '{}', created_at TEXT NOT NULL DEFAULT '', branch_id TEXT NOT NULL DEFAULT 'branch-main')",
            "CREATE TABLE visual_assets (story_id TEXT NOT NULL, updated_at TEXT NOT NULL DEFAULT '', file_path TEXT NOT NULL DEFAULT '', branch_id TEXT NOT NULL DEFAULT 'branch-main')",
            "CREATE TABLE visual_generation_jobs (story_id TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'queued', updated_at TEXT NOT NULL DEFAULT '', branch_id TEXT NOT NULL DEFAULT 'branch-main')",
        ] {
            sqlx::query(statement)
                .execute(pool)
                .await
                .expect("create story version table");
        }
    }

    fn story_update() -> StoryUpdate {
        StoryUpdate {
            name: None,
            description: None,
            genre: None,
            tone: None,
            language: None,
            is_archived: None,
        }
    }

    async fn revision(pool: &SqlitePool, story_id: &str) -> i64 {
        sqlx::query_scalar("SELECT revision FROM stories WHERE id = ?")
            .bind(story_id)
            .fetch_one(pool)
            .await
            .expect("story revision")
    }

    #[tokio::test]
    async fn snapshot_transaction_keeps_branch_and_revision_consistent() {
        let path =
            std::env::temp_dir().join(format!("oneday-snapshot-{}.db", uuid::Uuid::new_v4()));
        let database_url = format!("sqlite://{}?mode=rwc", path.display());
        let pool = SqlitePoolOptions::new()
            .max_connections(2)
            .connect(&database_url)
            .await
            .expect("file sqlite pool");
        sqlx::query("PRAGMA journal_mode=WAL")
            .execute(&pool)
            .await
            .expect("enable WAL");
        sqlx::query(
            r#"CREATE TABLE stories (
                id TEXT PRIMARY KEY,
                name TEXT NOT NULL,
                description TEXT NOT NULL DEFAULT '',
                genre TEXT NOT NULL DEFAULT '',
                tone TEXT NOT NULL DEFAULT '',
                language TEXT NOT NULL DEFAULT '',
                is_archived INTEGER NOT NULL DEFAULT 0,
                revision INTEGER NOT NULL DEFAULT 0,
                active_branch_id TEXT NOT NULL,
                updated_at TEXT NOT NULL
            )"#,
        )
        .execute(&pool)
        .await
        .expect("create stories table");
        create_story_version_tables(&pool).await;
        sqlx::query("INSERT INTO stories (id,name,revision,active_branch_id,updated_at) VALUES ('story-1','Story',7,'branch-main','v7')")
            .execute(&pool)
            .await
            .expect("insert story");
        sqlx::query("INSERT INTO sessions (id,story_id,started_at,branch_id) VALUES ('session-main','story-1','1','branch-main')")
            .execute(&pool)
            .await
            .expect("insert main session");
        sqlx::query("INSERT INTO chat_messages (story_id,session_id,turn,role,branch_id) VALUES ('story-1','session-main',7,'assistant','branch-main')")
            .execute(&pool)
            .await
            .expect("insert main message");
        sqlx::query("INSERT INTO saves (id,story_id,name,branch_id) VALUES ('save-main','story-1','Main save','branch-main')")
            .execute(&pool)
            .await
            .expect("insert main save");

        let mut read_tx = pool.begin().await.expect("begin snapshot transaction");
        let (_, branch_id) = load_snapshot_story(&mut read_tx, "story-1")
            .await
            .expect("load snapshot context");
        assert_eq!(branch_id, "branch-main");

        let mut write_tx = pool.begin().await.expect("begin branch switch");
        sqlx::query("UPDATE stories SET revision=8,active_branch_id='branch-alt',updated_at='v8' WHERE id='story-1'")
            .execute(&mut *write_tx)
            .await
            .expect("switch active branch");
        sqlx::query("INSERT INTO sessions (id,story_id,started_at,branch_id) VALUES ('session-alt','story-1','2','branch-alt')")
            .execute(&mut *write_tx)
            .await
            .expect("insert alt session");
        sqlx::query("INSERT INTO chat_messages (story_id,session_id,turn,role,branch_id) VALUES ('story-1','session-alt',8,'assistant','branch-alt')")
            .execute(&mut *write_tx)
            .await
            .expect("insert alt message");
        sqlx::query("INSERT INTO saves (id,story_id,name,branch_id) VALUES ('save-alt','story-1','Alt save','branch-alt')")
            .execute(&mut *write_tx)
            .await
            .expect("insert alt save");
        write_tx.commit().await.expect("commit branch switch");

        let during = story_version_for_branch(&mut *read_tx, "story-1", &branch_id)
            .await
            .expect("version inside snapshot transaction");
        assert_eq!(during.revision, 7);
        assert_eq!(during.active_session_id, "session-main");
        assert_eq!(during.last_message_id, 1);
        assert_eq!(during.save_count, 1);
        let saves = load_saves(&mut read_tx, "story-1", &branch_id)
            .await
            .expect("branch-scoped saves inside snapshot transaction");
        assert_eq!(saves.len(), 1);
        assert_eq!(saves[0].id, "save-main");
        read_tx.commit().await.expect("commit snapshot transaction");

        let after = story_version(&pool, "story-1")
            .await
            .expect("version after branch switch");
        assert_eq!(after.revision, 8);
        assert_eq!(after.active_session_id, "session-alt");
        assert_eq!(after.last_message_id, 2);
        assert_eq!(after.save_count, 1);
        pool.close().await;
        let _ = std::fs::remove_file(path);
    }

    #[tokio::test]
    async fn update_story_bumps_revision_for_prompt_metadata() {
        let pool = story_pool().await;
        let mut update = story_update();
        update.name = Some(" New Name ".to_string());
        update.description = Some(" sharper premise ".to_string());

        let story = update_story(&pool, "story-1", update)
            .await
            .expect("update story");

        assert_eq!(story.name, "New Name");
        assert_eq!(story.description, "sharper premise");
        assert_eq!(revision(&pool, "story-1").await, 8);
    }

    #[tokio::test]
    async fn update_story_language_applies_to_future_prompt_revision() {
        let pool = story_pool().await;
        let mut update = story_update();
        update.language = Some(" it-IT ".to_string());

        let story = update_story(&pool, "story-1", update)
            .await
            .expect("update story language");

        assert_eq!(story.language, "it-IT");
        assert_eq!(revision(&pool, "story-1").await, 8);
    }

    #[tokio::test]
    async fn update_story_archive_only_does_not_bump_revision() {
        let pool = story_pool().await;
        let mut update = story_update();
        update.is_archived = Some(true);

        let story = update_story(&pool, "story-1", update)
            .await
            .expect("archive story");

        assert!(story.is_archived);
        assert_eq!(revision(&pool, "story-1").await, 7);
    }

    #[tokio::test]
    async fn update_story_missing_story_is_not_found_error() {
        let pool = story_pool().await;
        let err = update_story(&pool, "missing-story", story_update())
            .await
            .expect_err("missing story should fail");

        assert!(err.to_string().contains("story not found: missing-story"));
    }

    #[tokio::test]
    async fn story_delete_plan_counts_story_and_tolerates_missing_optional_tables() {
        let pool = story_pool().await;

        let plan = story_delete_plan(&pool, "story-1")
            .await
            .expect("delete plan");

        assert_eq!(plan.story_id, "story-1");
        assert_eq!(plan.story_name, "Old Name");
        assert!(plan.total_rows >= 1);
        assert!(plan
            .counts
            .iter()
            .any(|count| count.table == "stories" && count.rows == 1));
        assert!(plan.retained_asset_files.is_empty());
    }

    #[tokio::test]
    async fn story_version_includes_session_panels_and_visual_job_state() {
        let pool = story_pool().await;
        for statement in [
            "INSERT INTO world_state (story_id, current_turn, updated_at) VALUES ('story-1', 3, 'world-v3')",
            "INSERT INTO sessions (id, story_id, started_at, ended_at) VALUES ('old-session', 'story-1', '2026-01-01T00:00:00Z', '2026-01-01T01:00:00Z')",
            "INSERT INTO sessions (id, story_id, started_at, ended_at) VALUES ('active-session', 'story-1', '2026-01-01T02:00:00Z', NULL)",
            "INSERT INTO chat_messages (story_id, session_id, turn, role, metadata_json, created_at) VALUES ('story-1', 'active-session', 3, 'assistant', '{}', 'm1')",
            "INSERT INTO characters (story_id, updated_at) VALUES ('story-1', 'character-v2')",
            "INSERT INTO npcs (story_id, updated_at) VALUES ('story-1', 'npc-v1')",
            "INSERT INTO npcs (story_id, updated_at) VALUES ('story-1', 'npc-v2')",
            "INSERT INTO chapters (story_id) VALUES ('story-1')",
            "INSERT INTO achievements (story_id, earned_at) VALUES ('story-1', 'achievement-v1')",
            "INSERT INTO saves (story_id, created_at) VALUES ('story-1', 'save-v1')",
            "INSERT INTO visual_assets (story_id, updated_at, file_path) VALUES ('story-1', 'asset-v1', '')",
            "INSERT INTO visual_generation_jobs (story_id, status, updated_at) VALUES ('story-1', 'queued', 'job-v1')",
        ] {
            sqlx::query(statement)
                .execute(&pool)
                .await
                .expect("insert version fixture");
        }

        let version = story_version(&pool, "story-1")
            .await
            .expect("story version");

        assert_eq!(version.turn, 3);
        assert_eq!(version.active_session_id, "active-session");
        assert_eq!(version.story_updated_at, "2026-01-01T00:00:00Z");
        assert_eq!(version.world_updated_at, "world-v3");
        assert_eq!(version.character_updated_at, "character-v2");
        assert_eq!(version.npc_count, 2);
        assert_eq!(version.npc_updated_at, "npc-v2");
        assert_eq!(version.chapter_count, 1);
        assert_eq!(version.latest_achievement_at, "achievement-v1");
        assert_eq!(version.latest_save_at, "save-v1");
        assert_eq!(version.visual_asset_updated_at, "asset-v1");
        assert_eq!(version.visual_job_updated_at, "job-v1");
        assert_eq!(version.active_visual_job_count, 1);
    }

    #[tokio::test]
    async fn history_and_export_are_paginated_searchable_and_branch_scoped() {
        let pool = story_pool().await;
        sqlx::query("DROP TABLE chapters")
            .execute(&pool)
            .await
            .expect("drop minimal chapters fixture");
        sqlx::query(
            r#"CREATE TABLE chapters (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                story_id TEXT NOT NULL,
                chapter_number INTEGER NOT NULL,
                title TEXT NOT NULL DEFAULT '',
                summary TEXT NOT NULL DEFAULT '',
                start_turn INTEGER NOT NULL DEFAULT 0,
                end_turn INTEGER,
                created_at TEXT NOT NULL DEFAULT '',
                branch_id TEXT NOT NULL,
                source_commit_id TEXT NOT NULL
            )"#,
        )
        .execute(&pool)
        .await
        .expect("create full chapters fixture");
        for statement in [
            "CREATE VIRTUAL TABLE chat_messages_fts USING fts5(content,content='chat_messages',content_rowid='id',tokenize='trigram')",
            "CREATE TRIGGER chat_messages_fts_insert AFTER INSERT ON chat_messages BEGIN INSERT INTO chat_messages_fts(rowid,content) VALUES (new.id,new.content); END",
            "CREATE VIRTUAL TABLE chapters_fts USING fts5(title,summary,content='chapters',content_rowid='id',tokenize='trigram')",
            "CREATE TRIGGER chapters_fts_insert AFTER INSERT ON chapters BEGIN INSERT INTO chapters_fts(rowid,title,summary) VALUES (new.id,new.title,new.summary); END",
        ] {
            sqlx::query(statement)
                .execute(&pool)
                .await
                .expect("create search fixture");
        }
        for (turn, content, branch) in [
            (1_i64, "first main message", "branch-main"),
            (2_i64, "needle in the archive", "branch-main"),
            (3_i64, "latest main message", "branch-main"),
            (9_i64, "alternate secret", "branch-alt"),
        ] {
            sqlx::query("INSERT INTO chat_messages (story_id,session_id,turn,role,content,created_at,branch_id,source_commit_id) VALUES ('story-1','session-1',?,'assistant',?,'2026-01-01',?,'commit')")
                .bind(turn)
                .bind(content)
                .bind(branch)
                .execute(&pool)
                .await
                .expect("insert history fixture");
        }
        for (number, title, branch) in [
            (1_i64, "Main Chapter", "branch-main"),
            (2_i64, "Alternate Chapter", "branch-alt"),
        ] {
            sqlx::query("INSERT INTO chapters (story_id,chapter_number,title,summary,start_turn,created_at,branch_id,source_commit_id) VALUES ('story-1',?,?,'summary',1,'2026-01-01',?,'commit')")
                .bind(number)
                .bind(title)
                .bind(branch)
                .execute(&pool)
                .await
                .expect("insert chapter fixture");
        }

        let first_page = history_page(&pool, "story-1", None, 2, "")
            .await
            .expect("first history page");
        assert_eq!(first_page.items.len(), 2);
        assert!(first_page.next_cursor.is_some());
        assert!(first_page
            .items
            .iter()
            .all(|item| item.branch_id == "branch-main"));

        let search = history_page(&pool, "story-1", None, 40, "needle")
            .await
            .expect("search history");
        assert_eq!(search.items.len(), 1);
        assert_eq!(search.items[0].content, "needle in the archive");

        let export = export_story(&pool, "story-1", "json")
            .await
            .expect("export active branch");
        let payload: Value = serde_json::from_str(&export.content).expect("valid export JSON");
        assert_eq!(payload["active_branch_id"], "branch-main");
        assert_eq!(payload["messages"].as_array().map(Vec::len), Some(3));
        assert_eq!(payload["chapters"].as_array().map(Vec::len), Some(1));
        assert_eq!(payload["chapters"][0]["title"], "Main Chapter");

        let epub = export_story(&pool, "story-1", "epub")
            .await
            .expect("EPUB export");
        assert_eq!(epub.encoding, "base64");
        let bytes = BASE64.decode(&epub.content).expect("base64 EPUB");
        let mut archive = zip::ZipArchive::new(Cursor::new(bytes)).expect("valid EPUB zip");
        let mut mimetype = String::new();
        std::io::Read::read_to_string(
            &mut archive
                .by_name("mimetype")
                .expect("mimetype first-class entry"),
            &mut mimetype,
        )
        .expect("read mimetype");
        assert_eq!(mimetype, "application/epub+zip");
        assert!(archive.by_name("OEBPS/content.opf").is_ok());
        assert!(archive.by_name("OEBPS/content.xhtml").is_ok());

        let (binary_name, binary_epub) = export_story_epub(&pool, "story-1")
            .await
            .expect("binary EPUB export");
        assert!(binary_name.ends_with(".epub"));
        assert!(binary_epub.starts_with(b"PK"));
    }

    #[test]
    fn latest_choices_uses_current_or_immediately_prior_active_session_turn() {
        let messages = vec![
            assistant_choice("old-session", 3, "Old session choice"),
            assistant_choice("active-session", 2, "Old turn choice"),
            assistant_choice("active-session", 3, "Current choice"),
        ];

        let choices = latest_choices(&messages, "active-session", 3);

        assert_eq!(choices.len(), 1);
        assert_eq!(choices[0].text, "Current choice");
        let next_turn_choices = latest_choices(&messages, "active-session", 4);
        assert_eq!(next_turn_choices.len(), 1);
        assert_eq!(next_turn_choices[0].text, "Current choice");
        assert!(latest_choices(&messages, "active-session", 5).is_empty());
    }

    #[test]
    fn latest_choices_preserves_first_turn_choices_after_world_advances() {
        let messages = vec![assistant_choice(
            "active-session",
            0,
            "Inspect the marked note",
        )];

        let choices = latest_choices(&messages, "active-session", 1);

        assert_eq!(choices.len(), 1);
        assert_eq!(choices[0].text, "Inspect the marked note");
    }

    fn assistant_choice(session_id: &str, turn: i64, text: &str) -> MessageView {
        MessageView {
            id: turn,
            session_id: session_id.to_string(),
            story_id: "story-1".to_string(),
            turn,
            role: "assistant".to_string(),
            content: text.to_string(),
            message_type: "narrative".to_string(),
            metadata: serde_json::json!({
                "output": {
                    "choices_data": [{ "id": 1, "text": text, "risk": "Low" }]
                }
            }),
            created_at: "2026-01-01T00:00:00Z".to_string(),
            branch_id: "branch-main".to_string(),
            source_commit_id: "commit-main".to_string(),
        }
    }
}
