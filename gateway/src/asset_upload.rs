use crate::{assets, error::PublicError, AppState};
use anyhow::{anyhow, Context};
use axum::extract::Multipart;
use image::{DynamicImage, GenericImageView, ImageFormat, ImageReader, Limits};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use sqlx::Row;
use std::io::Cursor;
use std::path::{Path, PathBuf};
use std::sync::Arc;
use tokio::fs;
use tokio::io::AsyncWriteExt;
use uuid::Uuid;

const MAX_UPLOAD_BYTES: usize = 10 * 1024 * 1024;
const MAX_METADATA_BYTES: usize = 16 * 1024;
const MAX_IMAGE_PIXELS: u64 = 20_000_000;
const MAX_IMAGE_SIDE: u32 = 8_192;
const MAX_DECODE_ALLOC: u64 = 128 * 1024 * 1024;
const MAX_NORMALIZED_BYTES: usize = 64 * 1024 * 1024;
pub const MAX_UPLOAD_REQUEST_BYTES: usize = MAX_UPLOAD_BYTES + 64 * 1024;

#[derive(Debug, Default, Deserialize)]
struct UploadMetadata {
    #[serde(default, alias = "selectAfterUpload")]
    select_after_upload: bool,
    #[serde(default, alias = "displayName")]
    display_name: String,
    #[serde(default, alias = "assetKind")]
    asset_kind: String,
}

#[derive(Debug, Serialize)]
pub struct VisualAssetUploadResponse {
    pub asset_id: String,
    pub version_id: i64,
    pub selected: bool,
    pub visual_assets: assets::VisualAssetsResponse,
}

struct ReceivedUpload {
    temp_path: PathBuf,
    original_filename: String,
    declared_mime: String,
    byte_size: usize,
    metadata: UploadMetadata,
}

struct NormalizedUpload {
    png: Vec<u8>,
    detected_mime: &'static str,
    width: u32,
    height: u32,
    sha256: String,
}

struct UploadAsset {
    kind: String,
    subject: String,
    canonical_entity_id: String,
    canonical_location_id: String,
    form_id: String,
    appearance_fingerprint: String,
    profile_revision_id: Option<String>,
    canon_status: String,
    prompt: String,
    negative_prompt: String,
    turn: i64,
    branch_id: String,
    source_commit_id: String,
}

pub async fn upload_visual_asset_version(
    state: Arc<AppState>,
    story_id: &str,
    asset_id: &str,
    multipart: Multipart,
) -> anyhow::Result<VisualAssetUploadResponse> {
    let received = receive_multipart(&state.paths.visual_asset_dir, multipart).await?;
    let result = persist_received_upload(&state, story_id, asset_id, &received).await;
    if result.is_err() {
        remove_if_present(&received.temp_path).await;
    }
    result
}

async fn receive_multipart(
    visual_asset_dir: &Path,
    mut multipart: Multipart,
) -> anyhow::Result<ReceivedUpload> {
    let temp_dir = upload_temp_dir(visual_asset_dir);
    fs::create_dir_all(&temp_dir)
        .await
        .with_context(|| format!("creating upload temp directory {}", temp_dir.display()))?;
    let temp_path = temp_dir.join(format!("{}.part", Uuid::new_v4()));
    let mut wrote_file = false;

    let result = async {
        let mut original_filename = String::new();
        let mut declared_mime = String::new();
        let mut byte_size = 0_usize;
        let mut metadata = UploadMetadata::default();
        let mut saw_metadata = false;

        while let Some(mut field) = multipart.next_field().await.map_err(|_| {
            PublicError::bad_request("invalid_visual_upload", "invalid multipart upload")
        })? {
            match field.name().unwrap_or_default() {
                "file" => {
                    if wrote_file {
                        return Err(PublicError::bad_request(
                            "invalid_visual_upload",
                            "exactly one image file is required",
                        )
                        .into());
                    }
                    original_filename = sanitize_filename(field.file_name().unwrap_or_default());
                    declared_mime = field
                        .content_type()
                        .unwrap_or_default()
                        .to_ascii_lowercase();
                    let mut output = fs::File::create(&temp_path).await.with_context(|| {
                        format!("creating temporary upload {}", temp_path.display())
                    })?;
                    wrote_file = true;
                    while let Some(chunk) = field.chunk().await.map_err(|_| {
                        PublicError::bad_request(
                            "invalid_visual_upload",
                            "could not read multipart image data",
                        )
                    })? {
                        byte_size = byte_size.saturating_add(chunk.len());
                        if byte_size > MAX_UPLOAD_BYTES {
                            return Err(PublicError::payload_too_large(
                                "visual_upload_too_large",
                                "visual uploads must not exceed 10 MiB",
                            )
                            .into());
                        }
                        output.write_all(&chunk).await?;
                    }
                    output.flush().await?;
                    output.sync_all().await?;
                }
                "metadata" => {
                    if saw_metadata {
                        return Err(PublicError::bad_request(
                            "invalid_visual_upload",
                            "metadata must be provided at most once",
                        )
                        .into());
                    }
                    saw_metadata = true;
                    let raw = field.text().await.map_err(|_| {
                        PublicError::bad_request(
                            "invalid_visual_upload_metadata",
                            "upload metadata is not valid text",
                        )
                    })?;
                    if raw.len() > MAX_METADATA_BYTES {
                        return Err(PublicError::bad_request(
                            "invalid_visual_upload_metadata",
                            "upload metadata is too large",
                        )
                        .into());
                    }
                    metadata = serde_json::from_str(&raw).map_err(|_| {
                        PublicError::bad_request(
                            "invalid_visual_upload_metadata",
                            "upload metadata is not valid JSON",
                        )
                    })?;
                }
                _ => {
                    return Err(PublicError::bad_request(
                        "invalid_visual_upload",
                        "multipart upload contains an unsupported field",
                    )
                    .into())
                }
            }
        }
        if !wrote_file || byte_size == 0 {
            return Err(PublicError::bad_request(
                "visual_upload_file_required",
                "an image file is required",
            )
            .into());
        }
        Ok(ReceivedUpload {
            temp_path: temp_path.clone(),
            original_filename,
            declared_mime,
            byte_size,
            metadata,
        })
    }
    .await;

    if result.is_err() && wrote_file {
        remove_if_present(&temp_path).await;
    }
    result
}

async fn persist_received_upload(
    state: &AppState,
    story_id: &str,
    asset_id: &str,
    received: &ReceivedUpload,
) -> anyhow::Result<VisualAssetUploadResponse> {
    let bytes = fs::read(&received.temp_path).await?;
    let normalized = normalize_upload(&bytes, &received.declared_mime)?;
    if normalized.png.len() > MAX_NORMALIZED_BYTES {
        return Err(PublicError::unprocessable_entity(
            "visual_upload_normalized_too_large",
            "normalized image exceeds the 64 MiB storage limit",
        )
        .into());
    }
    fs::write(&received.temp_path, &normalized.png).await?;

    let story_slug = slug(story_id);
    let filename = format!(
        "{}-upload-{}-{}.png",
        slug(asset_id),
        &normalized.sha256[..16],
        Uuid::new_v4()
    );
    let final_dir = state.paths.visual_asset_dir.join(&story_slug);
    fs::create_dir_all(&final_dir).await?;
    let final_path = final_dir.join(&filename);
    fs::rename(&received.temp_path, &final_path)
        .await
        .with_context(|| format!("publishing uploaded image {}", final_path.display()))?;
    let url = format!("/generated/assets/{story_slug}/{filename}");

    let database_result = persist_upload_rows(
        &state.pool,
        story_id,
        asset_id,
        received,
        &normalized,
        &final_path,
        &url,
    )
    .await;
    let (version_id, selected) = match database_result {
        Ok(value) => value,
        Err(error) => {
            remove_if_present(&final_path).await;
            return Err(error);
        }
    };

    Ok(VisualAssetUploadResponse {
        asset_id: asset_id.to_string(),
        version_id,
        selected,
        visual_assets: assets::visual_assets(&state.pool, story_id).await?,
    })
}

pub async fn upload_new_visual_asset(
    state: Arc<AppState>,
    story_id: &str,
    multipart: Multipart,
) -> anyhow::Result<VisualAssetUploadResponse> {
    let received = receive_multipart(&state.paths.visual_asset_dir, multipart).await?;
    let result = persist_new_visual_asset(&state, story_id, &received).await;
    if result.is_err() {
        remove_if_present(&received.temp_path).await;
    }
    result
}

async fn persist_new_visual_asset(
    state: &AppState,
    story_id: &str,
    received: &ReceivedUpload,
) -> anyhow::Result<VisualAssetUploadResponse> {
    let display_name = received.metadata.display_name.trim();
    if display_name.is_empty() || display_name.chars().count() > 100 {
        return Err(PublicError::bad_request(
            "invalid_visual_asset_name",
            "displayName must contain between 1 and 100 characters",
        )
        .into());
    }
    let kind = received.metadata.asset_kind.trim().to_ascii_lowercase();
    if !matches!(kind.as_str(), "custom" | "world" | "location" | "character") {
        return Err(PublicError::bad_request(
            "invalid_visual_asset_kind",
            "assetKind must be custom, world, location, or character",
        )
        .into());
    }
    let bytes = fs::read(&received.temp_path).await?;
    let normalized = normalize_upload(&bytes, &received.declared_mime)?;
    if normalized.png.len() > MAX_NORMALIZED_BYTES {
        return Err(PublicError::unprocessable_entity(
            "visual_upload_normalized_too_large",
            "normalized image exceeds the 64 MiB storage limit",
        )
        .into());
    }
    fs::write(&received.temp_path, &normalized.png).await?;
    let asset_id = format!("custom-{}", Uuid::new_v4());
    let story_slug = slug(story_id);
    let filename = format!(
        "{}-upload-{}-{}.png",
        slug(&asset_id),
        &normalized.sha256[..16],
        Uuid::new_v4()
    );
    let final_dir = state.paths.visual_asset_dir.join(&story_slug);
    fs::create_dir_all(&final_dir).await?;
    let final_path = final_dir.join(&filename);
    fs::rename(&received.temp_path, &final_path).await?;
    let url = format!("/generated/assets/{story_slug}/{filename}");
    let db_result = persist_new_asset_rows(
        &state.pool,
        story_id,
        &asset_id,
        display_name,
        &kind,
        received,
        &normalized,
        &final_path,
        &url,
    )
    .await;
    let version_id = match db_result {
        Ok(id) => id,
        Err(error) => {
            remove_if_present(&final_path).await;
            return Err(error);
        }
    };
    Ok(VisualAssetUploadResponse {
        asset_id,
        version_id,
        selected: true,
        visual_assets: assets::visual_assets(&state.pool, story_id).await?,
    })
}

#[allow(clippy::too_many_arguments)]
async fn persist_new_asset_rows(
    pool: &sqlx::SqlitePool,
    story_id: &str,
    asset_id: &str,
    display_name: &str,
    kind: &str,
    received: &ReceivedUpload,
    normalized: &NormalizedUpload,
    final_path: &Path,
    url: &str,
) -> anyhow::Result<i64> {
    let mut tx = pool.begin().await?;
    let row = sqlx::query(
        r#"SELECT s.active_branch_id,b.head_commit_id,COALESCE(w.current_turn,0) AS current_turn
           FROM stories s JOIN story_branches b ON b.id=s.active_branch_id
           LEFT JOIN world_state w ON w.story_id=s.id WHERE s.id=?"#,
    )
    .bind(story_id)
    .fetch_optional(&mut *tx)
    .await?
    .ok_or_else(|| PublicError::not_found("story_not_found", "story not found"))?;
    let branch_id: String = row.try_get("active_branch_id")?;
    let source_commit_id: String = row.try_get("head_commit_id")?;
    let turn: i64 = row.try_get("current_turn")?;
    let lineage_key = format!("custom:{asset_id}");
    let fingerprint = format!("upload:{}", normalized.sha256);
    sqlx::query(
        r#"INSERT INTO visual_assets
           (id,story_id,kind,subject,lineage_key,appearance_fingerprint,canon_status,gate_state,gate_reason,
            generation_eligible,prompt,negative_prompt,status,url,file_path,provider,source,turn,branch_id,source_commit_id)
           VALUES (?,?,?,?,?,?,'draft','manual_upload','User-provided image',0,'','','ready',?,?,'manual-upload','upload',?,?,?)"#,
    ).bind(asset_id).bind(story_id).bind(kind).bind(display_name).bind(&lineage_key).bind(&fingerprint)
      .bind(url).bind(final_path.to_string_lossy().as_ref()).bind(turn).bind(&branch_id).bind(&source_commit_id)
      .execute(&mut *tx).await?;
    let version_id: i64 = sqlx::query_scalar(
        r#"INSERT INTO visual_asset_versions
           (asset_id,story_id,kind,subject,url,file_path,provider,turn,branch_id,source_commit_id,
            appearance_fingerprint,canon_status,source_kind)
           VALUES (?,?,?,?,?,?,'manual-upload',?,?,?,?,?,'upload') RETURNING id"#,
    )
    .bind(asset_id)
    .bind(story_id)
    .bind(kind)
    .bind(display_name)
    .bind(url)
    .bind(final_path.to_string_lossy().as_ref())
    .bind(turn)
    .bind(&branch_id)
    .bind(&source_commit_id)
    .bind(&fingerprint)
    .bind("draft")
    .fetch_one(&mut *tx)
    .await?;
    sqlx::query(
        r#"INSERT INTO visual_asset_uploads
           (version_id,story_id,asset_id,branch_id,original_filename_display,declared_mime,detected_mime,byte_size,width,height,sha256)
           VALUES (?,?,?,?,?,?,?,?,?,?,?)"#,
    ).bind(version_id).bind(story_id).bind(asset_id).bind(&branch_id).bind(&received.original_filename)
      .bind(&received.declared_mime).bind(normalized.detected_mime).bind(received.byte_size as i64)
      .bind(i64::from(normalized.width)).bind(i64::from(normalized.height)).bind(&normalized.sha256).execute(&mut *tx).await?;
    assets::select_new_version_on(
        &mut tx,
        story_id,
        asset_id,
        &branch_id,
        &source_commit_id,
        version_id,
    )
    .await?;
    tx.commit().await?;
    Ok(version_id)
}

async fn persist_upload_rows(
    pool: &sqlx::SqlitePool,
    story_id: &str,
    asset_id: &str,
    received: &ReceivedUpload,
    normalized: &NormalizedUpload,
    final_path: &Path,
    url: &str,
) -> anyhow::Result<(i64, bool)> {
    let mut transaction = pool.begin().await?;
    let asset = load_visible_asset(&mut transaction, story_id, asset_id).await?;
    let version_id: i64 = sqlx::query_scalar(
        r#"INSERT INTO visual_asset_versions
           (asset_id,story_id,kind,subject,url,file_path,prompt,revised_prompt,negative_prompt,
            provider,turn,branch_id,source_commit_id,canonical_entity_id,canonical_location_id,
            form_id,appearance_fingerprint,profile_revision_id,canon_status,source_kind)
           VALUES (?,?,?,?,?,?,?,'',?,'manual-upload',?,?,?,?,?,?,?,?,?,'upload') RETURNING id"#,
    )
    .bind(asset_id)
    .bind(story_id)
    .bind(&asset.kind)
    .bind(&asset.subject)
    .bind(url)
    .bind(final_path.to_string_lossy().as_ref())
    .bind(&asset.prompt)
    .bind(&asset.negative_prompt)
    .bind(asset.turn)
    .bind(&asset.branch_id)
    .bind(&asset.source_commit_id)
    .bind(&asset.canonical_entity_id)
    .bind(&asset.canonical_location_id)
    .bind(&asset.form_id)
    .bind(&asset.appearance_fingerprint)
    .bind(&asset.profile_revision_id)
    .bind(&asset.canon_status)
    .fetch_one(&mut *transaction)
    .await?;

    sqlx::query(
        r#"INSERT INTO visual_asset_uploads
           (version_id,story_id,asset_id,branch_id,original_filename_display,declared_mime,
            detected_mime,byte_size,width,height,sha256)
           VALUES (?,?,?,?,?,?,?,?,?,?,?)"#,
    )
    .bind(version_id)
    .bind(story_id)
    .bind(asset_id)
    .bind(&asset.branch_id)
    .bind(&received.original_filename)
    .bind(&received.declared_mime)
    .bind(normalized.detected_mime)
    .bind(received.byte_size as i64)
    .bind(i64::from(normalized.width))
    .bind(i64::from(normalized.height))
    .bind(&normalized.sha256)
    .execute(&mut *transaction)
    .await?;

    if received.metadata.select_after_upload {
        assets::select_new_version_on(
            &mut transaction,
            story_id,
            asset_id,
            &asset.branch_id,
            &asset.source_commit_id,
            version_id,
        )
        .await?;
    }
    transaction.commit().await?;
    Ok((version_id, received.metadata.select_after_upload))
}

async fn load_visible_asset(
    connection: &mut sqlx::SqliteConnection,
    story_id: &str,
    asset_id: &str,
) -> anyhow::Result<UploadAsset> {
    let row = sqlx::query(
        r#"WITH RECURSIVE active AS (
             SELECT s.active_branch_id AS branch_id,b.head_commit_id,b.fork_commit_id,b.created_at AS branch_created
             FROM stories s JOIN story_branches b ON b.id=s.active_branch_id WHERE s.id=?
           ), ancestors(id) AS (
             SELECT head_commit_id FROM active
             UNION ALL SELECT c.parent_commit_id FROM turn_commits c JOIN ancestors a ON c.id=a.id
             WHERE c.parent_commit_id IS NOT NULL
           )
           SELECT v.kind,v.subject,v.canonical_entity_id,v.canonical_location_id,v.form_id,
                  v.appearance_fingerprint,v.profile_revision_id,v.canon_status,v.turn,
                  x.branch_id,x.head_commit_id AS source_commit_id,
                  CASE WHEN o.asset_id IS NULL THEN v.prompt ELSE o.prompt_override END AS prompt,
                  CASE WHEN o.asset_id IS NULL THEN v.negative_prompt ELSE o.negative_prompt_override END AS negative_prompt
           FROM visual_assets v CROSS JOIN active x
           LEFT JOIN visual_asset_branch_overrides o ON o.asset_id=v.id AND o.branch_id=x.branch_id
           WHERE v.story_id=? AND v.id=? AND v.source_commit_id IN (SELECT id FROM ancestors)
             AND (v.branch_id=x.branch_id OR v.source_commit_id!=COALESCE(x.fork_commit_id,'') OR v.created_at<=x.branch_created)"#,
    )
    .bind(story_id)
    .bind(story_id)
    .bind(asset_id)
    .fetch_optional(&mut *connection)
    .await?
    .ok_or_else(|| {
        PublicError::not_found(
            "visual_asset_not_found",
            "visual asset not found on the active branch",
        )
    })?;
    Ok(UploadAsset {
        kind: row.try_get("kind")?,
        subject: row.try_get("subject")?,
        canonical_entity_id: row.try_get("canonical_entity_id")?,
        canonical_location_id: row.try_get("canonical_location_id")?,
        form_id: row.try_get("form_id")?,
        appearance_fingerprint: row.try_get("appearance_fingerprint")?,
        profile_revision_id: row.try_get("profile_revision_id")?,
        canon_status: row.try_get("canon_status")?,
        prompt: row.try_get("prompt")?,
        negative_prompt: row.try_get("negative_prompt")?,
        turn: row.try_get("turn")?,
        branch_id: row.try_get("branch_id")?,
        source_commit_id: row.try_get("source_commit_id")?,
    })
}

fn normalize_upload(bytes: &[u8], declared_mime: &str) -> anyhow::Result<NormalizedUpload> {
    let format = image::guess_format(bytes).map_err(|_| {
        PublicError::unsupported_media_type(
            "unsupported_visual_upload_format",
            "visual uploads must be PNG, JPEG, or static WebP",
        )
    })?;
    let detected_mime = match format {
        ImageFormat::Png => {
            if png_is_animated(bytes)? {
                return Err(PublicError::unsupported_media_type(
                    "animated_visual_upload_not_supported",
                    "animated PNG uploads are not supported",
                )
                .into());
            }
            "image/png"
        }
        ImageFormat::Jpeg => "image/jpeg",
        ImageFormat::WebP => {
            if webp_is_animated(bytes)? {
                return Err(PublicError::unsupported_media_type(
                    "animated_visual_upload_not_supported",
                    "animated WebP uploads are not supported",
                )
                .into());
            }
            "image/webp"
        }
        _ => {
            return Err(PublicError::unsupported_media_type(
                "unsupported_visual_upload_format",
                "visual uploads must be PNG, JPEG, or static WebP",
            )
            .into())
        }
    };
    if !declared_mime.is_empty() && declared_mime != detected_mime {
        return Err(PublicError::unsupported_media_type(
            "visual_upload_mime_mismatch",
            "declared image type does not match the uploaded bytes",
        )
        .into());
    }

    let dimensions = ImageReader::with_format(Cursor::new(bytes), format)
        .into_dimensions()
        .map_err(invalid_image)?;
    validate_dimensions(dimensions.0, dimensions.1)?;
    let mut limits = Limits::default();
    limits.max_image_width = Some(MAX_IMAGE_SIDE);
    limits.max_image_height = Some(MAX_IMAGE_SIDE);
    limits.max_alloc = Some(MAX_DECODE_ALLOC);
    let mut reader = ImageReader::with_format(Cursor::new(bytes), format);
    reader.limits(limits);
    let image = reader.decode().map_err(invalid_image)?;
    let (width, height) = image.dimensions();
    validate_dimensions(width, height)?;
    let mut output = Cursor::new(Vec::new());
    DynamicImage::ImageRgba8(image.to_rgba8())
        .write_to(&mut output, ImageFormat::Png)
        .map_err(invalid_image)?;
    let png = output.into_inner();
    let sha256 = sha256_hex(&png);
    Ok(NormalizedUpload {
        png,
        detected_mime,
        width,
        height,
        sha256,
    })
}

fn validate_dimensions(width: u32, height: u32) -> anyhow::Result<()> {
    if width == 0
        || height == 0
        || width > MAX_IMAGE_SIDE
        || height > MAX_IMAGE_SIDE
        || u64::from(width) * u64::from(height) > MAX_IMAGE_PIXELS
    {
        return Err(PublicError::unprocessable_entity(
            "visual_upload_dimensions_invalid",
            "image dimensions exceed the 8192 pixel side or 20 megapixel limit",
        )
        .into());
    }
    Ok(())
}

fn invalid_image(error: image::ImageError) -> anyhow::Error {
    PublicError::unprocessable_entity(
        "invalid_visual_upload_image",
        format!("uploaded image could not be decoded: {error}"),
    )
    .into()
}

fn png_is_animated(bytes: &[u8]) -> anyhow::Result<bool> {
    if !bytes.starts_with(b"\x89PNG\r\n\x1a\n") {
        return Ok(false);
    }
    let mut offset = 8_usize;
    while offset + 12 <= bytes.len() {
        let length = u32::from_be_bytes(bytes[offset..offset + 4].try_into()?) as usize;
        let chunk_end = offset
            .checked_add(12)
            .and_then(|value| value.checked_add(length))
            .ok_or_else(|| anyhow!("invalid PNG chunk length"))?;
        if chunk_end > bytes.len() {
            return Err(PublicError::unprocessable_entity(
                "invalid_visual_upload_image",
                "uploaded PNG is truncated",
            )
            .into());
        }
        if &bytes[offset + 4..offset + 8] == b"acTL" {
            return Ok(true);
        }
        offset = chunk_end;
    }
    Ok(false)
}

fn webp_is_animated(bytes: &[u8]) -> anyhow::Result<bool> {
    if bytes.len() < 12 || &bytes[..4] != b"RIFF" || &bytes[8..12] != b"WEBP" {
        return Ok(false);
    }
    let mut offset = 12_usize;
    while offset + 8 <= bytes.len() {
        let kind = &bytes[offset..offset + 4];
        let length = u32::from_le_bytes(bytes[offset + 4..offset + 8].try_into()?) as usize;
        let data_start = offset + 8;
        let data_end = data_start
            .checked_add(length)
            .ok_or_else(|| anyhow!("invalid WebP chunk length"))?;
        if data_end > bytes.len() {
            return Err(PublicError::unprocessable_entity(
                "invalid_visual_upload_image",
                "uploaded WebP is truncated",
            )
            .into());
        }
        if kind == b"ANIM" || kind == b"ANMF" {
            return Ok(true);
        }
        if kind == b"VP8X" && length > 0 && bytes[data_start] & 0x02 != 0 {
            return Ok(true);
        }
        offset = data_end + (length & 1);
    }
    Ok(false)
}

fn sanitize_filename(value: &str) -> String {
    let basename = value.rsplit(['/', '\\']).next().unwrap_or_default();
    basename
        .chars()
        .filter(|character| !character.is_control())
        .take(160)
        .collect()
}

fn slug(value: &str) -> String {
    let mut result = String::new();
    let mut last_dash = false;
    for character in value.chars() {
        if character.is_ascii_alphanumeric() {
            result.push(character.to_ascii_lowercase());
            last_dash = false;
        } else if !last_dash {
            result.push('-');
            last_dash = true;
        }
    }
    let result = result.trim_matches('-');
    if result.is_empty() {
        "asset".to_string()
    } else {
        result.to_string()
    }
}

fn sha256_hex(bytes: &[u8]) -> String {
    Sha256::digest(bytes)
        .iter()
        .map(|byte| format!("{byte:02x}"))
        .collect()
}

fn upload_temp_dir(visual_asset_dir: &Path) -> PathBuf {
    visual_asset_dir
        .parent()
        .unwrap_or(visual_asset_dir)
        .join("visual_asset_upload_tmp")
}

pub async fn cleanup_stale_upload_parts(visual_asset_dir: &Path) -> anyhow::Result<()> {
    let temp_dir = upload_temp_dir(visual_asset_dir);
    let mut entries = match fs::read_dir(&temp_dir).await {
        Ok(entries) => entries,
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => return Ok(()),
        Err(error) => return Err(error.into()),
    };
    while let Some(entry) = entries.next_entry().await? {
        if entry.file_type().await?.is_file()
            && entry
                .path()
                .extension()
                .is_some_and(|extension| extension == "part")
        {
            fs::remove_file(entry.path()).await?;
        }
    }
    Ok(())
}

async fn remove_if_present(path: &Path) {
    if let Err(error) = fs::remove_file(path).await {
        if error.kind() != std::io::ErrorKind::NotFound {
            tracing::warn!(path = %path.display(), error = %error, "could not remove visual upload temporary file");
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use image::{ImageBuffer, Rgba};
    use sqlx::sqlite::SqlitePoolOptions;

    fn png_fixture() -> Vec<u8> {
        let image = ImageBuffer::from_pixel(2, 3, Rgba([10_u8, 20, 30, 255]));
        let mut output = Cursor::new(Vec::new());
        DynamicImage::ImageRgba8(image)
            .write_to(&mut output, ImageFormat::Png)
            .unwrap();
        output.into_inner()
    }

    #[test]
    fn normalizes_static_png_and_rejects_mime_spoofing() {
        let normalized = normalize_upload(&png_fixture(), "image/png").unwrap();
        assert_eq!((normalized.width, normalized.height), (2, 3));
        assert_eq!(normalized.detected_mime, "image/png");
        assert!(normalized.png.starts_with(b"\x89PNG\r\n\x1a\n"));
        assert!(normalize_upload(&png_fixture(), "image/jpeg").is_err());
    }

    #[test]
    fn detects_apng_control_chunk() {
        let mut bytes = png_fixture();
        let iend = bytes
            .windows(4)
            .position(|window| window == b"IEND")
            .unwrap()
            - 4;
        let mut chunk = Vec::new();
        chunk.extend_from_slice(&8_u32.to_be_bytes());
        chunk.extend_from_slice(b"acTL");
        chunk.extend_from_slice(&[0; 8]);
        chunk.extend_from_slice(&[0; 4]);
        bytes.splice(iend..iend, chunk);
        assert!(png_is_animated(&bytes).unwrap());
    }

    #[test]
    fn strips_paths_and_controls_from_display_filename() {
        assert_eq!(sanitize_filename("../folder\\evil\n.png"), "evil.png");
    }

    #[tokio::test]
    async fn persists_upload_without_selection_unless_explicitly_requested() {
        let pool = SqlitePoolOptions::new()
            .max_connections(1)
            .connect("sqlite::memory:")
            .await
            .unwrap();
        for statement in [
            r#"CREATE TABLE stories (id TEXT PRIMARY KEY,active_branch_id TEXT NOT NULL)"#,
            r#"CREATE TABLE story_branches (id TEXT PRIMARY KEY,story_id TEXT NOT NULL,head_commit_id TEXT NOT NULL,fork_commit_id TEXT,created_at DATETIME DEFAULT CURRENT_TIMESTAMP)"#,
            r#"CREATE TABLE turn_commits (id TEXT PRIMARY KEY,parent_commit_id TEXT)"#,
            r#"CREATE TABLE visual_assets (
                id TEXT PRIMARY KEY,story_id TEXT NOT NULL,kind TEXT NOT NULL,subject TEXT NOT NULL,
                canonical_entity_id TEXT NOT NULL DEFAULT '',canonical_location_id TEXT NOT NULL DEFAULT '',
                form_id TEXT NOT NULL DEFAULT '',appearance_fingerprint TEXT NOT NULL DEFAULT '',
                lineage_key TEXT NOT NULL DEFAULT '',gate_state TEXT NOT NULL DEFAULT 'eligible',gate_reason TEXT NOT NULL DEFAULT '',
                generation_eligible INTEGER NOT NULL DEFAULT 1,status TEXT NOT NULL DEFAULT 'pending',url TEXT NOT NULL DEFAULT '',
                file_path TEXT NOT NULL DEFAULT '',provider TEXT NOT NULL DEFAULT '',source TEXT NOT NULL DEFAULT '',
                profile_revision_id TEXT,canon_status TEXT NOT NULL DEFAULT 'draft',prompt TEXT NOT NULL DEFAULT '',
                negative_prompt TEXT NOT NULL DEFAULT '',turn INTEGER NOT NULL DEFAULT 0,branch_id TEXT NOT NULL,
                source_commit_id TEXT NOT NULL,created_at DATETIME DEFAULT CURRENT_TIMESTAMP)"#,
            r#"CREATE TABLE visual_asset_versions (
                id INTEGER PRIMARY KEY AUTOINCREMENT,asset_id TEXT NOT NULL,story_id TEXT NOT NULL,kind TEXT NOT NULL,
                subject TEXT NOT NULL,url TEXT NOT NULL,file_path TEXT NOT NULL,prompt TEXT NOT NULL DEFAULT '',
                revised_prompt TEXT NOT NULL DEFAULT '',negative_prompt TEXT NOT NULL DEFAULT '',provider TEXT NOT NULL DEFAULT '',
                turn INTEGER NOT NULL DEFAULT 0,branch_id TEXT NOT NULL,source_commit_id TEXT NOT NULL,
                canonical_entity_id TEXT NOT NULL DEFAULT '',canonical_location_id TEXT NOT NULL DEFAULT '',
                form_id TEXT NOT NULL DEFAULT '',appearance_fingerprint TEXT NOT NULL DEFAULT '',profile_revision_id TEXT,
                canon_status TEXT NOT NULL DEFAULT 'draft',source_kind TEXT NOT NULL DEFAULT 'generated')"#,
            r#"CREATE TABLE visual_asset_uploads (
                version_id INTEGER PRIMARY KEY,story_id TEXT NOT NULL,asset_id TEXT NOT NULL,branch_id TEXT NOT NULL,
                original_filename_display TEXT NOT NULL,declared_mime TEXT NOT NULL,detected_mime TEXT NOT NULL,
                byte_size INTEGER NOT NULL,width INTEGER NOT NULL,height INTEGER NOT NULL,sha256 TEXT NOT NULL)"#,
            r#"CREATE TABLE visual_asset_selection_states (
                asset_id TEXT NOT NULL,story_id TEXT NOT NULL,branch_id TEXT NOT NULL,source_commit_id TEXT NOT NULL,
                selected_version_id INTEGER,history_json TEXT NOT NULL DEFAULT '[]',cursor INTEGER NOT NULL DEFAULT -1,
                updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,PRIMARY KEY(asset_id,branch_id))"#,
            r#"CREATE TABLE visual_asset_branch_overrides (
                asset_id TEXT NOT NULL,story_id TEXT NOT NULL,branch_id TEXT NOT NULL,source_commit_id TEXT NOT NULL,
                prompt_override TEXT NOT NULL DEFAULT '',negative_prompt_override TEXT NOT NULL DEFAULT '',
                status_override TEXT NOT NULL DEFAULT '',error_override TEXT NOT NULL DEFAULT '',
                updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,PRIMARY KEY(asset_id,branch_id))"#,
            r#"CREATE TABLE world_state (story_id TEXT PRIMARY KEY,current_turn INTEGER NOT NULL DEFAULT 0)"#,
            r#"INSERT INTO stories VALUES ('story','branch-main')"#,
            r#"INSERT INTO story_branches (id,story_id,head_commit_id) VALUES ('branch-main','story','commit-main')"#,
            r#"INSERT INTO turn_commits VALUES ('commit-main',NULL)"#,
            r#"INSERT INTO world_state VALUES ('story',7)"#,
            r#"INSERT INTO visual_assets (id,story_id,kind,subject,appearance_fingerprint,branch_id,source_commit_id)
               VALUES ('asset','story','location','Dock','dock','branch-main','commit-main')"#,
        ] {
            sqlx::query(statement).execute(&pool).await.unwrap();
        }
        let normalized = NormalizedUpload {
            png: png_fixture(),
            detected_mime: "image/png",
            width: 2,
            height: 3,
            sha256: "abc".into(),
        };
        let received = |select_after_upload| ReceivedUpload {
            temp_path: PathBuf::from("/tmp/unused.part"),
            original_filename: "dock.png".into(),
            declared_mime: "image/png".into(),
            byte_size: 42,
            metadata: UploadMetadata {
                select_after_upload,
                ..UploadMetadata::default()
            },
        };
        let (first, selected) = persist_upload_rows(
            &pool,
            "story",
            "asset",
            &received(false),
            &normalized,
            Path::new("/tmp/first.png"),
            "/generated/first.png",
        )
        .await
        .unwrap();
        assert!(!selected);
        let selection_count: i64 = sqlx::query_scalar(
            "SELECT COUNT(*) FROM visual_asset_selection_states WHERE asset_id='asset'",
        )
        .fetch_one(&pool)
        .await
        .unwrap();
        assert_eq!(selection_count, 0);

        let (second, selected) = persist_upload_rows(
            &pool,
            "story",
            "asset",
            &received(true),
            &normalized,
            Path::new("/tmp/second.png"),
            "/generated/second.png",
        )
        .await
        .unwrap();
        assert!(selected);
        assert_ne!(first, second);
        let selected_version: i64 = sqlx::query_scalar(
            "SELECT selected_version_id FROM visual_asset_selection_states WHERE asset_id='asset'",
        )
        .fetch_one(&pool)
        .await
        .unwrap();
        assert_eq!(selected_version, second);
        let source_kind: String =
            sqlx::query_scalar("SELECT source_kind FROM visual_asset_versions WHERE id=?")
                .bind(second)
                .fetch_one(&pool)
                .await
                .unwrap();
        assert_eq!(source_kind, "upload");

        let custom_received = ReceivedUpload {
            temp_path: PathBuf::from("/tmp/unused-custom.part"),
            original_filename: "custom.png".into(),
            declared_mime: "image/png".into(),
            byte_size: 42,
            metadata: UploadMetadata {
                display_name: "Custom cover".into(),
                asset_kind: "custom".into(),
                select_after_upload: true,
            },
        };
        let custom_version = persist_new_asset_rows(
            &pool,
            "story",
            "custom-fixture",
            "Custom cover",
            "custom",
            &custom_received,
            &normalized,
            Path::new("/tmp/custom.png"),
            "/generated/assets/story/custom.png",
        )
        .await
        .unwrap();
        let custom_source: String =
            sqlx::query_scalar("SELECT source_kind FROM visual_asset_versions WHERE id=?")
                .bind(custom_version)
                .fetch_one(&pool)
                .await
                .unwrap();
        let selected_custom: i64 = sqlx::query_scalar("SELECT selected_version_id FROM visual_asset_selection_states WHERE asset_id='custom-fixture'")
            .fetch_one(&pool).await.unwrap();
        assert_eq!(custom_source, "upload");
        assert_eq!(selected_custom, custom_version);
    }
}
