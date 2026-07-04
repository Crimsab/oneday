use crate::db;
use anyhow::Context;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use sqlx::{Row, SqlitePool};

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
