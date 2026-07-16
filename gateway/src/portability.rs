use crate::{db::StorySummary, error::PublicError, AppState};
use anyhow::{anyhow, Context};
use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
use serde::{Deserialize, Serialize};
use serde_json::{json, Map, Value};
use sha2::{Digest, Sha256};
use sqlx::{Column, Row, Sqlite, SqlitePool, Transaction, TypeInfo, ValueRef};
use std::{
    collections::{BTreeMap, HashMap},
    io::{Cursor, Read, Write},
    path::{Component, Path, PathBuf},
    sync::Arc,
};
use uuid::Uuid;
use zip::{write::SimpleFileOptions, CompressionMethod, ZipArchive, ZipWriter};

const ARCHIVE_KIND: &str = "oneday-story-archive";
const ARCHIVE_VERSION: u32 = 1;
const TEMPLATE_KIND: &str = "oneday-world-template";
const TEMPLATE_VERSION: u32 = 1;
const MAX_ARCHIVE_BYTES: u64 = 512 * 1024 * 1024;
const MAX_ARCHIVE_ENTRIES: usize = 10_000;

#[derive(Clone, Debug, Deserialize)]
pub struct ArchiveOptions {
    #[serde(default = "yes")]
    pub history: bool,
    #[serde(default = "yes")]
    pub saves: bool,
    #[serde(default = "yes")]
    pub visual_assets: bool,
    #[serde(default = "yes")]
    pub audio: bool,
    #[serde(default = "yes")]
    pub translations: bool,
    #[serde(default = "yes")]
    pub world_detail: bool,
}
fn yes() -> bool {
    true
}
impl Default for ArchiveOptions {
    fn default() -> Self {
        Self {
            history: true,
            saves: true,
            visual_assets: true,
            audio: true,
            translations: true,
            world_detail: true,
        }
    }
}

#[derive(Debug, Serialize, Deserialize)]
struct ArchiveManifest {
    kind: String,
    version: u32,
    exported_at: String,
    source_story_id: String,
    data_sha256: String,
    file_sha256: BTreeMap<String, String>,
    file_map: BTreeMap<String, String>,
    options: ArchiveOptionsView,
}

#[derive(Debug, Serialize, Deserialize)]
struct ArchiveOptionsView {
    history: bool,
    saves: bool,
    visual_assets: bool,
    audio: bool,
    translations: bool,
    world_detail: bool,
}
impl From<&ArchiveOptions> for ArchiveOptionsView {
    fn from(v: &ArchiveOptions) -> Self {
        Self {
            history: v.history,
            saves: v.saves,
            visual_assets: v.visual_assets,
            audio: v.audio,
            translations: v.translations,
            world_detail: v.world_detail,
        }
    }
}

#[derive(Debug, Serialize, Deserialize)]
struct ArchiveData {
    tables: BTreeMap<String, Vec<Map<String, Value>>>,
}

#[derive(Debug, Serialize)]
pub struct ImportResult {
    pub story_id: String,
    pub story_name: String,
    pub imported_tables: usize,
    pub imported_files: usize,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct WorldTemplate {
    kind: String,
    version: u32,
    exported_at: String,
    data: ArchiveData,
}

pub async fn cleanup_orphan_imports(state: &AppState) -> anyhow::Result<()> {
    let mut entries = match tokio::fs::read_dir(&state.paths.visual_asset_dir).await {
        Ok(entries) => entries,
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => return Ok(()),
        Err(error) => return Err(error.into()),
    };
    while let Some(entry) = entries.next_entry().await? {
        let name = entry.file_name().to_string_lossy().to_string();
        let story_id = match name.strip_prefix("import-") {
            Some(id) => id,
            None => continue,
        };
        if !entry.file_type().await?.is_dir() {
            continue;
        }
        let exists: i64 = sqlx::query_scalar("SELECT COUNT(*) FROM stories WHERE id=?")
            .bind(story_id)
            .fetch_one(&state.pool)
            .await?;
        if exists > 0 {
            continue;
        }
        let old = entry
            .metadata()
            .await?
            .modified()
            .ok()
            .and_then(|time| time.elapsed().ok())
            .map(|age| age.as_secs() > 3600)
            .unwrap_or(false);
        if old {
            let _ = tokio::fs::remove_dir_all(entry.path()).await;
        }
    }
    Ok(())
}

pub async fn export_story_archive(
    state: Arc<AppState>,
    story_id: &str,
    options: ArchiveOptions,
) -> anyhow::Result<(String, Vec<u8>)> {
    let story: StorySummary = sqlx::query_as::<_,(String,String,String,String,String,String,i64,String)>("SELECT id,name,description,genre,tone,language,is_archived,CAST(updated_at AS TEXT) FROM stories WHERE id=?")
        .bind(story_id).fetch_optional(&state.pool).await?.map(|r| StorySummary{id:r.0,name:r.1,description:r.2,genre:r.3,tone:r.4,language:r.5,is_archived:r.6!=0,updated_at:r.7})
        .ok_or_else(|| PublicError::not_found("story_not_found","Story not found."))?;
    let table_names: Vec<String> = sqlx::query_scalar("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name").fetch_all(&state.pool).await?;
    let mut tables: BTreeMap<String, Vec<Map<String, Value>>> = BTreeMap::new();
    for table in table_names {
        if !safe_identifier(&table) || !include_table(&table, &options) {
            continue;
        }
        let columns = table_columns(&state.pool, &table).await?;
        let filter = if table == "stories" {
            Some("id=?")
        } else if columns.iter().any(|column| column.name == "story_id") {
            Some("story_id=?")
        } else {
            None
        };
        let Some(filter) = filter else {
            continue;
        };
        let sql = format!("SELECT * FROM \"{table}\" WHERE {filter}");
        let rows = sqlx::query(&sql)
            .bind(story_id)
            .fetch_all(&state.pool)
            .await?;
        if !rows.is_empty() {
            tables.insert(
                table,
                rows.iter().map(row_json).collect::<anyhow::Result<_>>()?,
            );
        }
    }
    let data = serde_json::to_vec(&ArchiveData { tables })?;
    let data_sha256 = sha256(&data);
    let mut file_map = BTreeMap::new();
    let mut file_sha256 = BTreeMap::new();
    let mut files = Vec::new();
    let data_root = state
        .paths
        .db_path
        .parent()
        .unwrap_or(Path::new("/nonexistent"))
        .canonicalize()
        .unwrap_or_else(|_| {
            state
                .paths
                .db_path
                .parent()
                .unwrap_or(Path::new("/nonexistent"))
                .to_path_buf()
        });
    let archive_data: ArchiveData = serde_json::from_slice(&data)?;
    for rows in archive_data.tables.values() {
        for row in rows {
            if let Some(Value::String(raw)) = row.get("file_path") {
                if raw.is_empty() || file_map.contains_key(raw) {
                    continue;
                }
                let path = PathBuf::from(raw);
                let canonical = match path.canonicalize() {
                    Ok(v) => v,
                    Err(_) => continue,
                };
                if !canonical.starts_with(&data_root) {
                    continue;
                }
                let bytes = tokio::fs::read(&canonical).await?;
                if bytes.len() as u64 > MAX_ARCHIVE_BYTES {
                    return Err(PublicError::payload_too_large(
                        "archive_asset_too_large",
                        "An asset is too large for a portable archive.",
                    )
                    .into());
                }
                let digest = sha256(&bytes);
                let name = safe_filename(
                    canonical
                        .file_name()
                        .and_then(|v| v.to_str())
                        .unwrap_or("asset.bin"),
                );
                let entry = format!("assets/{}-{name}", &digest[..16]);
                file_map.insert(raw.clone(), entry.clone());
                file_sha256.insert(entry.clone(), digest);
                files.push((entry, bytes));
            }
        }
    }
    let manifest = ArchiveManifest {
        kind: ARCHIVE_KIND.into(),
        version: ARCHIVE_VERSION,
        exported_at: chrono::Utc::now().to_rfc3339(),
        source_story_id: story_id.into(),
        data_sha256,
        file_sha256,
        file_map,
        options: (&options).into(),
    };
    let bytes = tokio::task::spawn_blocking(move || build_zip(&manifest, &data, files))
        .await
        .context("joining story archive writer")??;
    Ok((format!("{}-oneday.zip", safe_filename(&story.name)), bytes))
}

pub async fn export_world_template(
    pool: &SqlitePool,
    story_id: &str,
) -> anyhow::Result<(String, Vec<u8>)> {
    let mut tables: BTreeMap<String, Vec<Map<String, Value>>> = BTreeMap::new();
    for table in [
        "stories",
        "characters",
        "world_state",
        "story_visual_profiles",
        "world_calendars",
        "world_clocks",
    ] {
        let columns = table_columns(pool, table).await?;
        if columns.is_empty() {
            continue;
        }
        let filter = if table == "stories" {
            "id=?"
        } else {
            "story_id=?"
        };
        let rows = sqlx::query(&format!("SELECT * FROM \"{table}\" WHERE {filter}"))
            .bind(story_id)
            .fetch_all(pool)
            .await?;
        if !rows.is_empty() {
            tables.insert(
                table.to_string(),
                rows.iter().map(row_json).collect::<anyhow::Result<_>>()?,
            );
        }
    }
    let story = tables
        .get_mut("stories")
        .and_then(|rows| rows.first_mut())
        .ok_or_else(|| PublicError::not_found("story_not_found", "Story not found."))?;
    let story_name = story
        .get("name")
        .and_then(Value::as_str)
        .unwrap_or("story")
        .to_string();
    let source_story_id = story
        .get("id")
        .and_then(Value::as_str)
        .unwrap_or(story_id)
        .to_string();
    let source_branch_id = format!("template-branch-{source_story_id}");
    let source_commit_id = format!("template-root-{source_story_id}");
    let source_session_id = format!("template-session-{source_story_id}");
    story.insert(
        "active_branch_id".into(),
        Value::String(source_branch_id.clone()),
    );
    story.insert("revision".into(), Value::Number(0.into()));
    story.insert("is_archived".into(), Value::Number(0.into()));
    if let Some(world) = tables
        .get_mut("world_state")
        .and_then(|rows| rows.first_mut())
    {
        world.insert("current_turn".into(), Value::Number(0.into()));
        world.insert("current_chapter".into(), Value::Number(1.into()));
        for field in [
            "global_events_json",
            "world_reactions_json",
            "player_guidance_json",
            "fronts_json",
        ] {
            world.insert(field.into(), Value::String("[]".into()));
        }
        for field in [
            "investigation_board_json",
            "project_clocks_json",
            "character_timeline_json",
            "scene_contract_json",
        ] {
            world.insert(field.into(), Value::String("{}".into()));
        }
    }
    if let Some(characters) = tables.get_mut("characters") {
        for character in characters {
            character.insert("inventory_json".into(), Value::String("[]".into()));
            character.insert("known_recipes_json".into(), Value::String("[]".into()));
        }
    }
    tables.entry("world_calendars".into()).or_insert_with(|| {
        vec![json!({
            "story_id": source_story_id.clone(),
            "name": "Default calendar",
            "config_json": "{\"hours_per_day\":24,\"minutes_per_hour\":60}"
        })
        .as_object()
        .expect("template calendar object")
        .clone()]
    });
    let clock = tables.entry("world_clocks".into()).or_insert_with(|| {
        vec![json!({
            "story_id": source_story_id.clone(),
            "calendar_story_id": source_story_id.clone(),
            "branch_id": source_branch_id.clone(),
            "source_commit_id": source_commit_id.clone()
        })
        .as_object()
        .expect("template clock object")
        .clone()]
    });
    if let Some(clock) = clock.first_mut() {
        clock.insert("day".into(), Value::Number(0.into()));
        clock.insert("minute_of_day".into(), Value::Number(0.into()));
        clock.insert("display_text".into(), Value::String("Day 0, 00:00".into()));
        clock.insert("branch_id".into(), Value::String(source_branch_id.clone()));
        clock.insert(
            "source_commit_id".into(),
            Value::String(source_commit_id.clone()),
        );
    }
    tables.insert(
        "story_branches".into(),
        vec![json!({
            "id": source_branch_id.clone(),
            "story_id": source_story_id.clone(),
            "name": "main",
            "fork_commit_id": null,
            "head_commit_id": source_commit_id.clone()
        })
        .as_object()
        .expect("template branch object")
        .clone()],
    );
    tables.insert(
        "turn_commits".into(),
        vec![json!({
            "id": source_commit_id.clone(),
            "story_id": source_story_id.clone(),
            "branch_id": source_branch_id.clone(),
            "parent_commit_id": null,
            "canonical_turn": 0,
            "story_revision": 0,
            "payload_hash": "",
            "kind": "root",
            "message": "Imported world template"
        })
        .as_object()
        .expect("template commit object")
        .clone()],
    );
    tables.insert(
        "sessions".into(),
        vec![json!({
            "id": source_session_id,
            "story_id": source_story_id,
            "summary": "",
            "branch_id": source_branch_id,
            "source_commit_id": source_commit_id
        })
        .as_object()
        .expect("template session object")
        .clone()],
    );
    let template = WorldTemplate {
        kind: TEMPLATE_KIND.into(),
        version: TEMPLATE_VERSION,
        exported_at: chrono::Utc::now().to_rfc3339(),
        data: ArchiveData { tables },
    };
    Ok((
        format!("{}-world.oneday.json", safe_filename(&story_name)),
        serde_json::to_vec_pretty(&template)?,
    ))
}

pub async fn import_world_template(
    state: Arc<AppState>,
    bytes: &[u8],
) -> anyhow::Result<ImportResult> {
    if bytes.len() > 4 * 1024 * 1024 {
        return Err(PublicError::payload_too_large(
            "template_too_large",
            "World template exceeds the 4 MiB limit.",
        )
        .into());
    }
    let template: WorldTemplate = serde_json::from_slice(bytes).map_err(|_| {
        PublicError::bad_request(
            "invalid_world_template",
            "The selected file is not a valid OneDay world template.",
        )
    })?;
    if template.kind != TEMPLATE_KIND || template.version != TEMPLATE_VERSION {
        return Err(PublicError::bad_request(
            "unsupported_template",
            "Unsupported OneDay world template.",
        )
        .into());
    }
    let imported_tables = template.data.tables.len();
    let story_id = Uuid::new_v4().to_string();
    let story_name = import_tables(
        &state.pool,
        template.data,
        &story_id,
        &HashMap::new(),
        &state.paths.visual_asset_dir,
    )
    .await?;
    Ok(ImportResult {
        story_id,
        story_name,
        imported_tables,
        imported_files: 0,
    })
}

fn include_table(table: &str, options: &ArchiveOptions) -> bool {
    if matches!(
        table,
        "schema_version"
            | "turn_idempotency"
            | "story_turn_locks"
            | "visual_generation_jobs"
            | "image_operations"
            | "translation_jobs"
            | "translation_job_items"
            | "tts_jobs"
            | "generation_runs"
            | "generation_attempts"
            | "generation_events"
    ) {
        return false;
    }
    if table.contains("telemetry") || table == "generation_traces" {
        return false;
    }
    if !options.history
        && matches!(
            table,
            "chat_messages" | "chapters" | "canonical_events" | "save_bookmarks"
        )
    {
        return false;
    }
    if !options.saves && matches!(table, "saves" | "save_bookmarks") {
        return false;
    }
    if !options.visual_assets && (table.contains("visual_") || table.starts_with("image_")) {
        return false;
    }
    if !options.audio
        && (table.contains("audio")
            || table.starts_with("tts_")
            || table.contains("voice_")
            || table.contains("pronunciation"))
    {
        return false;
    }
    if !options.translations
        && (table.starts_with("translation_") || table == "content_translations")
    {
        return false;
    }
    if !options.world_detail
        && matches!(
            table,
            "regions"
                | "locations"
                | "location_aliases"
                | "location_edges"
                | "entity_position_events"
                | "world_time_events"
                | "weather_states"
                | "canonical_world_events"
                | "world_thread_events"
        )
    {
        return false;
    }
    true
}

fn build_zip(
    manifest: &ArchiveManifest,
    data: &[u8],
    files: Vec<(String, Vec<u8>)>,
) -> anyhow::Result<Vec<u8>> {
    let mut zip = ZipWriter::new(Cursor::new(Vec::new()));
    let options = SimpleFileOptions::default().compression_method(CompressionMethod::Deflated);
    zip.start_file("manifest.json", options)?;
    zip.write_all(&serde_json::to_vec_pretty(manifest)?)?;
    zip.start_file("data.json", options)?;
    zip.write_all(data)?;
    for (name, bytes) in files {
        zip.start_file(name, options)?;
        zip.write_all(&bytes)?;
    }
    Ok(zip.finish()?.into_inner())
}

pub async fn import_story_archive(
    state: Arc<AppState>,
    bytes: Vec<u8>,
) -> anyhow::Result<ImportResult> {
    if bytes.len() as u64 > MAX_ARCHIVE_BYTES {
        return Err(PublicError::payload_too_large(
            "archive_too_large",
            "Story archive exceeds the import limit.",
        )
        .into());
    }
    let parsed = tokio::task::spawn_blocking(move || parse_zip(bytes))
        .await
        .context("joining story archive reader")?
        .map_err(|_| {
            PublicError::bad_request(
                "invalid_story_archive",
                "The selected file is not a valid OneDay story archive.",
            )
        })?;
    let new_story_id = Uuid::new_v4().to_string();
    let import_id = Uuid::new_v4().to_string();
    let stage = state
        .paths
        .visual_asset_dir
        .join(format!(".import-{import_id}"));
    let final_dir = state
        .paths
        .visual_asset_dir
        .join(format!("import-{new_story_id}"));
    tokio::fs::create_dir_all(&stage).await?;
    let mut path_map = HashMap::new();
    for (old, entry) in &parsed.manifest.file_map {
        let file = safe_filename(
            Path::new(entry)
                .file_name()
                .and_then(|v| v.to_str())
                .unwrap_or("asset.bin"),
        );
        let staged = stage.join(&file);
        let data = parsed
            .files
            .get(entry)
            .ok_or_else(|| anyhow!("archive asset {entry} is missing"))?;
        tokio::fs::write(&staged, data).await?;
        path_map.insert(old.clone(), final_dir.join(file));
    }
    let imported_tables = parsed.data.tables.len();
    if !path_map.is_empty() {
        if let Err(error) = tokio::fs::rename(&stage, &final_dir).await {
            let _ = tokio::fs::remove_dir_all(&stage).await;
            return Err(error.into());
        }
    } else {
        let _ = tokio::fs::remove_dir_all(&stage).await;
    }
    let result = import_tables(
        &state.pool,
        parsed.data,
        &new_story_id,
        &path_map,
        &state.paths.visual_asset_dir,
    )
    .await;
    let story_name = match result {
        Ok(name) => name,
        Err(error) => {
            let _ = tokio::fs::remove_dir_all(&final_dir).await;
            return Err(error);
        }
    };
    Ok(ImportResult {
        story_id: new_story_id,
        story_name,
        imported_tables,
        imported_files: path_map.len(),
    })
}

struct ParsedArchive {
    manifest: ArchiveManifest,
    data: ArchiveData,
    files: BTreeMap<String, Vec<u8>>,
}
fn parse_zip(bytes: Vec<u8>) -> anyhow::Result<ParsedArchive> {
    let mut zip = ZipArchive::new(Cursor::new(bytes))?;
    if zip.len() > MAX_ARCHIVE_ENTRIES {
        return Err(anyhow!("archive contains too many entries"));
    }
    let mut total = 0u64;
    let mut entries = BTreeMap::new();
    for index in 0..zip.len() {
        let mut file = zip.by_index(index)?;
        let name = file.name().to_string();
        validate_archive_path(&name)?;
        total = total.saturating_add(file.size());
        if total > MAX_ARCHIVE_BYTES {
            return Err(anyhow!("archive expands beyond the import limit"));
        }
        let mut data = Vec::new();
        file.read_to_end(&mut data)?;
        entries.insert(name, data);
    }
    let manifest: ArchiveManifest = serde_json::from_slice(
        entries
            .get("manifest.json")
            .ok_or_else(|| anyhow!("manifest.json is missing"))?,
    )?;
    if manifest.kind != ARCHIVE_KIND || manifest.version != ARCHIVE_VERSION {
        return Err(anyhow!("unsupported OneDay archive format"));
    }
    let data_bytes = entries
        .get("data.json")
        .ok_or_else(|| anyhow!("data.json is missing"))?;
    if sha256(data_bytes) != manifest.data_sha256 {
        return Err(anyhow!("story data checksum mismatch"));
    }
    for (name, expected) in &manifest.file_sha256 {
        let data = entries
            .get(name)
            .ok_or_else(|| anyhow!("archive asset {name} is missing"))?;
        if sha256(data) != *expected {
            return Err(anyhow!("archive asset checksum mismatch: {name}"));
        }
    }
    let data = serde_json::from_slice(data_bytes)?;
    Ok(ParsedArchive {
        manifest,
        data,
        files: entries,
    })
}

fn validate_archive_path(value: &str) -> anyhow::Result<()> {
    let path = Path::new(value);
    if path.is_absolute()
        || value.contains('\\')
        || value.contains('\0')
        || path.components().any(|c| {
            matches!(
                c,
                Component::ParentDir | Component::RootDir | Component::Prefix(_)
            )
        })
    {
        return Err(anyhow!("unsafe archive path"));
    }
    Ok(())
}

#[derive(Clone)]
struct ColumnMeta {
    name: String,
    kind: String,
    pk: i64,
}
#[derive(Clone)]
struct TableMeta {
    columns: Vec<ColumnMeta>,
    foreign: HashMap<String, (String, String)>,
}
async fn table_columns(pool: &SqlitePool, table: &str) -> anyhow::Result<Vec<ColumnMeta>> {
    let rows = sqlx::query(&format!("PRAGMA table_info(\"{table}\")"))
        .fetch_all(pool)
        .await?;
    Ok(rows
        .into_iter()
        .map(|r| ColumnMeta {
            name: r.get(1),
            kind: r.get::<String, _>(2).to_uppercase(),
            pk: r.get(5),
        })
        .collect())
}
async fn table_meta(pool: &SqlitePool, table: &str) -> anyhow::Result<TableMeta> {
    let columns = table_columns(pool, table).await?;
    let rows = sqlx::query(&format!("PRAGMA foreign_key_list(\"{table}\")"))
        .fetch_all(pool)
        .await?;
    let foreign = rows
        .into_iter()
        .map(|r| {
            (
                r.get::<String, _>(3),
                (r.get::<String, _>(2), r.get::<String, _>(4)),
            )
        })
        .collect();
    Ok(TableMeta { columns, foreign })
}

async fn import_tables(
    pool: &SqlitePool,
    mut data: ArchiveData,
    new_story_id: &str,
    path_map: &HashMap<String, PathBuf>,
    visual_root: &Path,
) -> anyhow::Result<String> {
    if !data.tables.contains_key("stories") {
        return Err(anyhow!("archive has no story record"));
    }
    let mut metas = HashMap::new();
    for table in data.tables.keys() {
        if !safe_identifier(table) {
            return Err(anyhow!("unsafe table name"));
        }
        metas.insert(table.clone(), table_meta(pool, table).await?);
    }
    let mut mappings: HashMap<(String, String), Value> = HashMap::new();
    let mut global_strings = HashMap::new();
    for (table, rows) in &data.tables {
        let meta = &metas[table];
        let pks = meta.columns.iter().filter(|c| c.pk > 0).collect::<Vec<_>>();
        if pks.len() != 1 {
            continue;
        }
        let pk = pks[0];
        for row in rows {
            if let Some(old) = row.get(&pk.name).filter(|v| !v.is_null()) {
                let next = if (table == "stories" && pk.name == "id") || pk.name == "story_id" {
                    Value::String(new_story_id.into())
                } else if pk.kind.contains("INT") {
                    Value::Number(random_positive_i64().into())
                } else {
                    Value::String(Uuid::new_v4().to_string())
                };
                mappings.insert((table.clone(), value_key(old)), next.clone());
                if let (Value::String(a), Value::String(b)) = (old, &next) {
                    global_strings.insert(a.clone(), b.clone());
                }
            }
        }
    }
    let special = |column: &str| -> Option<&'static str> {
        match column {
            "active_branch_id" | "branch_id" => Some("story_branches"),
            "head_commit_id" | "fork_commit_id" | "parent_commit_id" | "source_commit_id"
            | "commit_id" => Some("turn_commits"),
            "session_id" => Some("sessions"),
            "save_id" => Some("saves"),
            "asset_id" => Some("visual_assets"),
            "selected_version_id"
            | "source_version_id"
            | "parent_version_id"
            | "result_version_id"
            | "version_id" => Some("visual_asset_versions"),
            "source_message_id" | "message_id" => Some("chat_messages"),
            "translation_id" => Some("content_translations"),
            _ => None,
        }
    };
    for (table, rows) in &mut data.tables {
        let meta = &metas[table];
        for row in rows {
            let old_file = row
                .get("file_path")
                .and_then(Value::as_str)
                .map(str::to_owned);
            for column in &meta.columns {
                let Some(value) = row.get_mut(&column.name) else {
                    continue;
                };
                if column.name == "story_id" {
                    *value = Value::String(new_story_id.into());
                    continue;
                }
                let reference = meta
                    .foreign
                    .get(&column.name)
                    .map(|v| v.0.as_str())
                    .or_else(|| special(&column.name));
                if let Some(reference) = reference {
                    if let Some(next) = mappings.get(&(reference.to_string(), value_key(value))) {
                        *value = next.clone();
                        continue;
                    }
                }
                if let Some(next) = mappings.get(&(table.clone(), value_key(value))) {
                    *value = next.clone();
                    continue;
                }
                remap_json_value(value, &global_strings);
            }
            if let Some(old) = old_file {
                if let Some(final_path) = path_map.get(&old) {
                    row.insert(
                        "file_path".into(),
                        Value::String(final_path.to_string_lossy().into()),
                    );
                    if row.contains_key("url") {
                        let relative = final_path.strip_prefix(visual_root).unwrap_or(final_path);
                        row.insert(
                            "url".into(),
                            Value::String(format!(
                                "/generated/assets/{}",
                                relative.to_string_lossy().replace('\\', "/")
                            )),
                        );
                    }
                }
            }
        }
    }
    let story_name = data
        .tables
        .get("stories")
        .and_then(|rows| rows.first())
        .and_then(|row| row.get("name"))
        .and_then(Value::as_str)
        .unwrap_or("Imported story")
        .to_string();
    let mut tx = pool.begin().await?;
    sqlx::query("PRAGMA defer_foreign_keys=ON")
        .execute(&mut *tx)
        .await?;
    let mut order = data.tables.keys().cloned().collect::<Vec<_>>();
    order.sort_by_key(|name| if name == "stories" { 0 } else { 1 });
    for table in order {
        let meta = &metas[&table];
        for row in &data.tables[&table] {
            insert_row(&mut tx, &table, meta, row)
                .await
                .with_context(|| format!("importing table {table}"))?;
        }
    }
    tx.commit().await?;
    Ok(story_name)
}

async fn insert_row(
    tx: &mut Transaction<'_, Sqlite>,
    table: &str,
    meta: &TableMeta,
    row: &Map<String, Value>,
) -> anyhow::Result<()> {
    let columns = meta
        .columns
        .iter()
        .filter(|c| row.contains_key(&c.name))
        .collect::<Vec<_>>();
    let names = columns
        .iter()
        .map(|c| format!("\"{}\"", c.name))
        .collect::<Vec<_>>()
        .join(",");
    let placeholders = std::iter::repeat_n("?", columns.len())
        .collect::<Vec<_>>()
        .join(",");
    let sql = format!("INSERT INTO \"{table}\" ({names}) VALUES ({placeholders})");
    let mut query = sqlx::query(&sql);
    for column in columns {
        let value = &row[&column.name];
        query = match value {
            Value::Null => query.bind(Option::<String>::None),
            Value::Bool(v) => query.bind(if *v { 1i64 } else { 0 }),
            Value::Number(v) if v.is_i64() => query.bind(v.as_i64().unwrap()),
            Value::Number(v) => query.bind(v.as_f64().unwrap_or_default()),
            Value::String(v) => query.bind(v),
            Value::Object(v) if v.contains_key("$base64") => {
                query.bind(BASE64.decode(v["$base64"].as_str().unwrap_or_default())?)
            }
            other => query.bind(serde_json::to_string(other)?),
        };
    }
    query.execute(&mut **tx).await?;
    Ok(())
}

fn row_json(row: &sqlx::sqlite::SqliteRow) -> anyhow::Result<Map<String, Value>> {
    let mut out = Map::new();
    for (index, column) in row.columns().iter().enumerate() {
        let raw = row.try_get_raw(index)?;
        let value = if raw.is_null() {
            Value::Null
        } else {
            match raw.type_info().name() {
                "INTEGER" => Value::Number(row.try_get::<i64, _>(index)?.into()),
                "REAL" => serde_json::Number::from_f64(row.try_get::<f64, _>(index)?)
                    .map(Value::Number)
                    .unwrap_or(Value::Null),
                "BLOB" => {
                    serde_json::json!({"$base64":BASE64.encode(row.try_get::<Vec<u8>,_>(index)?)})
                }
                _ => Value::String(row.try_get::<String, _>(index)?),
            }
        };
        out.insert(column.name().into(), value);
    }
    Ok(out)
}
fn remap_json_value(value: &mut Value, map: &HashMap<String, String>) {
    if let Value::String(raw) = value {
        if let Some(next) = map.get(raw) {
            *raw = next.clone();
            return;
        }
        if (raw.starts_with('{') || raw.starts_with('[')) && raw.len() < 16 * 1024 * 1024 {
            if let Ok(mut nested) = serde_json::from_str::<Value>(raw) {
                remap_nested(&mut nested, map);
                if let Ok(encoded) = serde_json::to_string(&nested) {
                    *raw = encoded;
                }
            }
        }
    }
}
fn remap_nested(value: &mut Value, map: &HashMap<String, String>) {
    match value {
        Value::String(v) => {
            if let Some(next) = map.get(v) {
                *v = next.clone();
            }
        }
        Value::Array(v) => v.iter_mut().for_each(|item| remap_nested(item, map)),
        Value::Object(v) => v.values_mut().for_each(|item| remap_nested(item, map)),
        _ => {}
    }
}
fn value_key(value: &Value) -> String {
    match value {
        Value::String(v) => format!("s:{v}"),
        Value::Number(v) => format!("n:{v}"),
        other => format!("j:{other}"),
    }
}
fn random_positive_i64() -> i64 {
    let bytes = *Uuid::new_v4().as_bytes();
    (i64::from_be_bytes(bytes[..8].try_into().unwrap()) & i64::MAX).max(1_000_000)
}
fn sha256(bytes: &[u8]) -> String {
    format!("{:x}", Sha256::digest(bytes))
}
fn safe_identifier(value: &str) -> bool {
    !value.is_empty() && value.chars().all(|c| c.is_ascii_alphanumeric() || c == '_')
}
fn safe_filename(value: &str) -> String {
    let clean = value
        .chars()
        .map(|c| {
            if c.is_ascii_alphanumeric() || matches!(c, '.' | '-' | '_') {
                c
            } else {
                '-'
            }
        })
        .collect::<String>();
    if clean.is_empty() {
        "story".into()
    } else {
        clean
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use sqlx::sqlite::SqlitePoolOptions;
    #[test]
    fn archive_paths_reject_traversal() {
        assert!(validate_archive_path("assets/a.png").is_ok());
        assert!(validate_archive_path("../a.png").is_err());
        assert!(validate_archive_path("assets\\a.png").is_err());
    }
    #[test]
    fn filenames_are_sanitized() {
        assert_eq!(safe_filename("A story: one"), "A-story--one");
    }

    #[test]
    fn custom_archives_keep_minimum_runtime_timeline_and_clock() {
        let options = ArchiveOptions {
            history: false,
            saves: false,
            visual_assets: false,
            audio: false,
            translations: false,
            world_detail: false,
        };
        for table in [
            "stories",
            "characters",
            "world_state",
            "sessions",
            "story_branches",
            "turn_commits",
            "world_calendars",
            "world_clocks",
        ] {
            assert!(
                include_table(table, &options),
                "missing runtime table {table}"
            );
        }
        assert!(!include_table("chat_messages", &options));
        assert!(!include_table("locations", &options));
    }

    #[tokio::test]
    async fn world_template_round_trip_resets_progress_and_bootstraps_timeline() {
        let pool = SqlitePoolOptions::new()
            .max_connections(1)
            .connect("sqlite::memory:")
            .await
            .expect("memory sqlite");
        for statement in [
            "CREATE TABLE stories(id TEXT PRIMARY KEY,name TEXT NOT NULL,active_branch_id TEXT,revision INTEGER DEFAULT 0,is_archived INTEGER DEFAULT 0)",
            "CREATE TABLE characters(id TEXT PRIMARY KEY,story_id TEXT,name TEXT,inventory_json TEXT,known_recipes_json TEXT)",
            "CREATE TABLE world_state(id TEXT PRIMARY KEY,story_id TEXT UNIQUE,current_turn INTEGER,current_chapter INTEGER,global_events_json TEXT,world_reactions_json TEXT,player_guidance_json TEXT,fronts_json TEXT,investigation_board_json TEXT,project_clocks_json TEXT,character_timeline_json TEXT,scene_contract_json TEXT)",
            "CREATE TABLE story_visual_profiles(story_id TEXT PRIMARY KEY,world_style_prompt TEXT)",
            "CREATE TABLE world_calendars(story_id TEXT PRIMARY KEY,name TEXT,config_json TEXT)",
            "CREATE TABLE world_clocks(story_id TEXT PRIMARY KEY,calendar_story_id TEXT,day INTEGER,minute_of_day INTEGER,display_text TEXT,branch_id TEXT,source_commit_id TEXT)",
            "CREATE TABLE story_branches(id TEXT PRIMARY KEY,story_id TEXT,name TEXT,fork_commit_id TEXT,head_commit_id TEXT)",
            "CREATE TABLE turn_commits(id TEXT PRIMARY KEY,story_id TEXT,branch_id TEXT,parent_commit_id TEXT,canonical_turn INTEGER,story_revision INTEGER,payload_hash TEXT,kind TEXT,message TEXT)",
            "CREATE TABLE sessions(id TEXT PRIMARY KEY,story_id TEXT,summary TEXT,branch_id TEXT,source_commit_id TEXT)",
        ] {
            sqlx::query(statement).execute(&pool).await.expect(statement);
        }
        sqlx::query("INSERT INTO stories VALUES('source','Harbor','old-branch',9,1)")
            .execute(&pool)
            .await
            .expect("story");
        sqlx::query(
            "INSERT INTO characters VALUES('hero','source','Mara','[\"key\"]','[\"tea\"]')",
        )
        .execute(&pool)
        .await
        .expect("character");
        sqlx::query("INSERT INTO world_state VALUES('world','source',18,4,'[1]','[1]','[1]','[1]','{\"a\":1}','{\"a\":1}','{\"a\":1}','{\"a\":1}')")
            .execute(&pool).await.expect("world");
        sqlx::query("INSERT INTO story_visual_profiles VALUES('source','ink')")
            .execute(&pool)
            .await
            .expect("profile");

        let (_, bytes) = export_world_template(&pool, "source")
            .await
            .expect("export");
        let template: WorldTemplate = serde_json::from_slice(&bytes).expect("template json");
        let name = import_tables(
            &pool,
            template.data,
            "clone",
            &HashMap::new(),
            Path::new("/tmp"),
        )
        .await
        .expect("import");
        assert_eq!(name, "Harbor");
        let (turn, chapter): (i64, i64) = sqlx::query_as(
            "SELECT current_turn,current_chapter FROM world_state WHERE story_id='clone'",
        )
        .fetch_one(&pool)
        .await
        .expect("cloned world");
        assert_eq!((turn, chapter), (0, 1));
        let inventory: String =
            sqlx::query_scalar("SELECT inventory_json FROM characters WHERE story_id='clone'")
                .fetch_one(&pool)
                .await
                .expect("cloned character");
        assert_eq!(inventory, "[]");
        let active: String =
            sqlx::query_scalar("SELECT active_branch_id FROM stories WHERE id='clone'")
                .fetch_one(&pool)
                .await
                .expect("active branch");
        let branch: String =
            sqlx::query_scalar("SELECT id FROM story_branches WHERE story_id='clone'")
                .fetch_one(&pool)
                .await
                .expect("branch");
        assert_eq!(active, branch);
    }
}
