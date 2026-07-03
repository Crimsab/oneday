use anyhow::Context;
use serde::Serialize;
use serde_json::{json, Value};
use sqlx::{Row, SqlitePool};

#[derive(Debug, Serialize)]
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
pub struct RecordView {
    pub id: String,
    pub name: String,
    pub fields: Value,
}

#[derive(Debug, Serialize)]
pub struct WorldView {
    pub id: String,
    pub current_location: String,
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

#[derive(Debug, Serialize)]
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

#[derive(Debug, Serialize)]
pub struct ChapterView {
    pub id: i64,
    pub chapter_number: i64,
    pub title: String,
    pub summary: String,
    pub start_turn: i64,
    pub end_turn: Option<i64>,
    pub created_at: String,
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
    pub last_message_id: i64,
    pub world_updated_at: String,
    pub achievement_count: i64,
    pub save_count: i64,
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

pub async fn snapshot(pool: &SqlitePool, story_id: &str) -> anyhow::Result<StorySnapshot> {
    let story = load_story(pool, story_id).await?;
    let character = load_character(pool, story_id).await?;
    let world = load_world(pool, story_id).await?;
    let active_session = load_active_session(pool, story_id).await?;
    let messages = load_messages(pool, story_id, 500).await?;
    let choices = latest_choices(&messages);
    let panels = PanelsView {
        chapters: load_chapters(pool, story_id).await?,
        achievements: load_achievements(pool, story_id).await?,
        npcs: load_npcs(pool, story_id).await?,
        sessions: load_sessions(pool, story_id).await?,
        saves: load_saves(pool, story_id).await?,
    };
    let version = story_version(pool, story_id).await?;
    Ok(StorySnapshot {
        server_time: chrono::Utc::now().to_rfc3339(),
        version,
        story,
        character,
        world,
        active_session,
        choices,
        messages,
        panels,
    })
}

pub async fn story_version(pool: &SqlitePool, story_id: &str) -> anyhow::Result<StoryVersion> {
    let row = sqlx::query(r#"SELECT
           COALESCE((SELECT current_turn FROM world_state WHERE story_id = ?), 0) AS turn,
           COALESCE((SELECT revision FROM stories WHERE id = ?), 0) AS revision,
           COALESCE((SELECT MAX(id) FROM chat_messages WHERE story_id = ?), 0) AS last_message_id,
           COALESCE((SELECT CAST(updated_at AS TEXT) FROM world_state WHERE story_id = ?), '') AS world_updated_at,
           COALESCE((SELECT COUNT(*) FROM achievements WHERE story_id = ?), 0) AS achievement_count,
           COALESCE((SELECT COUNT(*) FROM saves WHERE story_id = ?), 0) AS save_count"#,
    )
    .bind(story_id)
    .bind(story_id)
    .bind(story_id)
    .bind(story_id)
    .bind(story_id)
    .bind(story_id)
    .fetch_one(pool)
    .await?;
    Ok(StoryVersion {
        turn: row.try_get("turn")?,
        revision: row.try_get("revision")?,
        last_message_id: row.try_get("last_message_id")?,
        world_updated_at: row.try_get("world_updated_at")?,
        achievement_count: row.try_get("achievement_count")?,
        save_count: row.try_get("save_count")?,
    })
}

async fn load_story(pool: &SqlitePool, story_id: &str) -> anyhow::Result<StorySummary> {
    let row = sqlx::query(r#"SELECT id, name, description, genre, tone, language, is_archived, CAST(updated_at AS TEXT) AS updated_at
         FROM stories WHERE id = ?"#,
    )
    .bind(story_id)
    .fetch_one(pool)
    .await
    .with_context(|| format!("loading story {story_id}"))?;
    Ok(story_summary_from_row(row))
}

async fn load_character(pool: &SqlitePool, story_id: &str) -> anyhow::Result<RecordView> {
    let row = sqlx::query(
        r#"SELECT id, name, background, stats_json, traits_json, skills_json,
                inventory_json, known_recipes_json, CAST(updated_at AS TEXT) AS updated_at
         FROM characters WHERE story_id = ?"#,
    )
    .bind(story_id)
    .fetch_one(pool)
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

async fn load_world(pool: &SqlitePool, story_id: &str) -> anyhow::Result<WorldView> {
    let row = sqlx::query(
        r#"SELECT id, current_location, known_locations_json, global_events_json,
                faction_standings_json, story_hooks_json, world_reactions_json,
                investigation_board_json, project_clocks_json, player_guidance_json,
                fronts_json, character_timeline_json, scene_contract_json,
                current_chapter, current_turn, CAST(updated_at AS TEXT) AS updated_at
         FROM world_state WHERE story_id = ?"#,
    )
    .bind(story_id)
    .fetch_one(pool)
    .await?;
    Ok(WorldView {
        id: row.try_get("id")?,
        current_location: row.try_get("current_location")?,
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

async fn load_active_session(pool: &SqlitePool, story_id: &str) -> anyhow::Result<SessionView> {
    let row = sqlx::query(
        r#"SELECT id, story_id, CAST(started_at AS TEXT) AS started_at,
                CAST(ended_at AS TEXT) AS ended_at, summary
         FROM sessions
         WHERE story_id = ? AND ended_at IS NULL
         ORDER BY started_at DESC
         LIMIT 1"#,
    )
    .bind(story_id)
    .fetch_optional(pool)
    .await?;
    if let Some(row) = row {
        return Ok(session_from_row(row));
    }

    let row = sqlx::query(
        r#"SELECT id, story_id, CAST(started_at AS TEXT) AS started_at,
                CAST(ended_at AS TEXT) AS ended_at, summary
         FROM sessions
         WHERE story_id = ?
         ORDER BY started_at DESC
         LIMIT 1"#,
    )
    .bind(story_id)
    .fetch_one(pool)
    .await?;
    Ok(session_from_row(row))
}

async fn load_messages(
    pool: &SqlitePool,
    story_id: &str,
    limit: i64,
) -> anyhow::Result<Vec<MessageView>> {
    let rows = sqlx::query(
        r#"SELECT id, session_id, story_id, turn, role, content, message_type,
                metadata_json, CAST(created_at AS TEXT) AS created_at
         FROM (
           SELECT id, session_id, story_id, turn, role, content, message_type,
                  metadata_json, created_at
           FROM chat_messages
           WHERE story_id = ?
           ORDER BY turn DESC, id DESC
           LIMIT ?
         )
         ORDER BY turn ASC, id ASC"#,
    )
    .bind(story_id)
    .bind(limit)
    .fetch_all(pool)
    .await?;
    rows.into_iter().map(message_from_row).collect()
}

async fn load_chapters(pool: &SqlitePool, story_id: &str) -> anyhow::Result<Vec<ChapterView>> {
    let rows = sqlx::query(
        r#"SELECT id, chapter_number, title, summary, start_turn, end_turn,
                CAST(created_at AS TEXT) AS created_at
         FROM chapters WHERE story_id = ? ORDER BY chapter_number ASC"#,
    )
    .bind(story_id)
    .fetch_all(pool)
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
            })
        })
        .collect()
}

async fn load_achievements(
    pool: &SqlitePool,
    story_id: &str,
) -> anyhow::Result<Vec<AchievementView>> {
    let rows = sqlx::query(
        r#"SELECT id, name, description, category, rarity, context,
                CAST(earned_at AS TEXT) AS earned_at
         FROM achievements WHERE story_id = ? ORDER BY earned_at ASC"#,
    )
    .bind(story_id)
    .fetch_all(pool)
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

async fn load_npcs(pool: &SqlitePool, story_id: &str) -> anyhow::Result<Vec<RecordView>> {
    let rows = sqlx::query(r#"SELECT id, name, role, appearance, personality_json, relationship_json, nemesis_json,
                private_thoughts, notes_on_protagonist, desires, disposition, is_alive,
                first_appeared_turn, last_seen_turn, can_help, CAST(updated_at AS TEXT) AS updated_at
         FROM npcs WHERE story_id = ? ORDER BY last_seen_turn DESC, name ASC"#,
    )
    .bind(story_id)
    .fetch_all(pool)
    .await?;
    Ok(rows
        .into_iter()
        .map(|row| RecordView {
            id: row_string(&row, "id"),
            name: row_string(&row, "name"),
            fields: json!({
                "role": row_string(&row, "role"),
                "appearance": row_string(&row, "appearance"),
                "personality": json_field(&row, "personality_json", json!({})),
                "relationship": json_field(&row, "relationship_json", json!({})),
                "nemesis": json_field(&row, "nemesis_json", json!({})),
                "private_thoughts": row_string(&row, "private_thoughts"),
                "notes_on_protagonist": row_string(&row, "notes_on_protagonist"),
                "desires": row_string(&row, "desires"),
                "disposition": row.try_get::<i64, _>("disposition").unwrap_or_default(),
                "is_alive": row.try_get::<i64, _>("is_alive").unwrap_or(1) != 0,
                "first_appeared_turn": row.try_get::<i64, _>("first_appeared_turn").unwrap_or_default(),
                "last_seen_turn": row.try_get::<i64, _>("last_seen_turn").unwrap_or_default(),
                "can_help": row.try_get::<i64, _>("can_help").unwrap_or_default() != 0,
                "updated_at": row_string(&row, "updated_at"),
            }),
        })
        .collect())
}

async fn load_sessions(pool: &SqlitePool, story_id: &str) -> anyhow::Result<Vec<SessionView>> {
    let rows = sqlx::query(
        r#"SELECT id, story_id, CAST(started_at AS TEXT) AS started_at,
                CAST(ended_at AS TEXT) AS ended_at, summary
         FROM sessions WHERE story_id = ? ORDER BY started_at DESC"#,
    )
    .bind(story_id)
    .fetch_all(pool)
    .await?;
    Ok(rows.into_iter().map(session_from_row).collect())
}

async fn load_saves(pool: &SqlitePool, story_id: &str) -> anyhow::Result<Vec<SaveView>> {
    let rows = sqlx::query(
        r#"SELECT id, name, turn, chapter, location, session_id, metadata_json,
                CAST(created_at AS TEXT) AS created_at
         FROM saves WHERE story_id = ? ORDER BY created_at DESC"#,
    )
    .bind(story_id)
    .fetch_all(pool)
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

fn latest_choices(messages: &[MessageView]) -> Vec<ChoiceView> {
    for message in messages.iter().rev() {
        if message.role != "assistant" {
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
