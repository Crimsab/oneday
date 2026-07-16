use crate::AppRuntime;
use reqwest::multipart::{Form, Part};
use serde::{Deserialize, Serialize};
use std::path::{Path, PathBuf};
use tauri::{AppHandle, State};
use tauri_plugin_dialog::{DialogExt, FilePath};
use tokio::fs::File;
use tokio_util::io::ReaderStream;
use url::Url;

const MAX_ARCHIVE_BYTES: u64 = 512 * 1024 * 1024;
const MAX_TEMPLATE_BYTES: u64 = 4 * 1024 * 1024;

#[derive(Debug, Deserialize, Serialize)]
pub struct StorySummary {
    id: String,
    name: String,
}

#[derive(Debug, Deserialize)]
struct ImportResult {
    story_id: String,
    story_name: String,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct TransferResult {
    cancelled: bool,
    message: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    story_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    path: Option<String>,
}

fn configured_server(state: &State<'_, AppRuntime>) -> Result<Url, String> {
    state
        .server_url
        .lock()
        .map_err(|_| "Desktop connection state is unavailable.".to_string())?
        .clone()
        .ok_or_else(|| "Connect to a OneDay server first.".to_string())
}

async fn checked(response: reqwest::Response) -> Result<reqwest::Response, String> {
    let status = response.status();
    if status.is_success() {
        return Ok(response);
    }
    let detail = response
        .json::<serde_json::Value>()
        .await
        .ok()
        .and_then(|value| {
            value
                .get("message")
                .or_else(|| value.get("error"))
                .and_then(|value| value.as_str())
                .map(str::to_owned)
        })
        .unwrap_or_else(|| format!("The OneDay server returned {status}."));
    Err(detail)
}

fn selected_path(value: Option<FilePath>) -> Result<Option<PathBuf>, String> {
    match value {
        None => Ok(None),
        Some(FilePath::Path(path)) => Ok(Some(path)),
        Some(FilePath::Url(_)) => Err("Choose a local file.".into()),
    }
}

#[tauri::command]
pub async fn list_remote_stories(
    state: State<'_, AppRuntime>,
) -> Result<Vec<StorySummary>, String> {
    let server = configured_server(&state)?;
    let response = checked(
        state
            .client
            .get(
                server
                    .join("api/stories")
                    .map_err(|error| error.to_string())?,
            )
            .send()
            .await
            .map_err(|error| format!("Could not load stories: {error}"))?,
    )
    .await?;
    response
        .json()
        .await
        .map_err(|error| format!("The server returned an invalid story list: {error}"))
}

#[tauri::command]
pub async fn choose_and_import_story(
    app: AppHandle,
    state: State<'_, AppRuntime>,
) -> Result<TransferResult, String> {
    let selection = app
        .dialog()
        .file()
        .add_filter("OneDay story", &["zip", "json"])
        .blocking_pick_file();
    let Some(path) = selected_path(selection)? else {
        return Ok(TransferResult {
            cancelled: true,
            message: "Import cancelled.".into(),
            story_id: None,
            path: None,
        });
    };
    let extension = path
        .extension()
        .and_then(|value| value.to_str())
        .unwrap_or_default()
        .to_ascii_lowercase();
    let metadata = tokio::fs::metadata(&path)
        .await
        .map_err(|error| format!("Could not inspect the selected file: {error}"))?;
    if !metadata.is_file() {
        return Err("Choose a regular OneDay ZIP or world template file.".into());
    }
    let server = configured_server(&state)?;
    let response = match extension.as_str() {
        "zip" => {
            if metadata.len() > MAX_ARCHIVE_BYTES {
                return Err("The archive is larger than the 512 MiB import limit.".into());
            }
            let file = File::open(&path)
                .await
                .map_err(|error| format!("Could not open the selected archive: {error}"))?;
            let body = reqwest::Body::wrap_stream(ReaderStream::new(file));
            let part = Part::stream_with_length(body, metadata.len())
                .file_name(
                    path.file_name()
                        .and_then(|v| v.to_str())
                        .unwrap_or("story.zip")
                        .to_owned(),
                )
                .mime_str("application/zip")
                .map_err(|error| error.to_string())?;
            checked(
                state
                    .client
                    .post(
                        server
                            .join("api/stories/import")
                            .map_err(|error| error.to_string())?,
                    )
                    .multipart(Form::new().part("file", part))
                    .send()
                    .await
                    .map_err(|error| format!("Could not upload the story archive: {error}"))?,
            )
            .await?
        }
        "json" => {
            if metadata.len() > MAX_TEMPLATE_BYTES {
                return Err("The world template is larger than the 4 MiB import limit.".into());
            }
            let body = tokio::fs::read(&path)
                .await
                .map_err(|error| format!("Could not read the selected template: {error}"))?;
            checked(
                state
                    .client
                    .post(
                        server
                            .join("api/stories/import-template")
                            .map_err(|error| error.to_string())?,
                    )
                    .header("content-type", "application/json")
                    .body(body)
                    .send()
                    .await
                    .map_err(|error| format!("Could not upload the world template: {error}"))?,
            )
            .await?
        }
        _ => return Err("Choose a .zip OneDay archive or .json world template.".into()),
    };
    let result: ImportResult = response
        .json()
        .await
        .map_err(|error| format!("The server returned an invalid import result: {error}"))?;
    Ok(TransferResult {
        cancelled: false,
        message: format!("Imported “{}”.", result.story_name),
        story_id: Some(result.story_id),
        path: None,
    })
}

fn safe_story_id(value: &str) -> Result<&str, String> {
    let trimmed = value.trim();
    if trimmed.is_empty()
        || trimmed.len() > 128
        || !trimmed
            .bytes()
            .all(|value| value.is_ascii_alphanumeric() || matches!(value, b'-' | b'_'))
    {
        return Err("Choose a valid story.".into());
    }
    Ok(trimmed)
}

fn add_extension(path: PathBuf, extension: &str) -> PathBuf {
    if path
        .extension()
        .and_then(|value| value.to_str())
        .is_some_and(|value| value.eq_ignore_ascii_case(extension))
    {
        path
    } else {
        PathBuf::from(format!("{}.{}", path.display(), extension))
    }
}

#[tauri::command]
pub async fn choose_and_export_story(
    app: AppHandle,
    state: State<'_, AppRuntime>,
    story_id: String,
    kind: String,
) -> Result<TransferResult, String> {
    let story_id = safe_story_id(&story_id)?;
    let server = configured_server(&state)?;
    let (response, extension, fallback) =
        match kind.as_str() {
            "archive" => {
                let url = server
                    .join(&format!("api/stories/{story_id}/archive"))
                    .map_err(|error| error.to_string())?;
                let response = state
                    .client
                    .post(url)
                    .json(&serde_json::json!({
                        "history": true,
                        "saves": true,
                        "visual_assets": true,
                        "audio": true,
                        "translations": true,
                        "world_detail": true
                    }))
                    .send()
                    .await
                    .map_err(|error| format!("Could not prepare the story archive: {error}"))?;
                (checked(response).await?, "zip", "oneday-story.zip")
            }
            "world" => {
                let url = server
                    .join(&format!("api/stories/{story_id}/world-template"))
                    .map_err(|error| error.to_string())?;
                let response =
                    state.client.get(url).send().await.map_err(|error| {
                        format!("Could not prepare the world template: {error}")
                    })?;
                (checked(response).await?, "json", "oneday-world.json")
            }
            _ => return Err("Unsupported export kind.".into()),
        };
    let filename = response
        .headers()
        .get(reqwest::header::CONTENT_DISPOSITION)
        .and_then(|value| value.to_str().ok())
        .and_then(filename_from_disposition)
        .unwrap_or_else(|| fallback.to_string());
    let bytes = response
        .bytes()
        .await
        .map_err(|error| format!("Could not download the export: {error}"))?;
    let selection = app
        .dialog()
        .file()
        .set_file_name(&filename)
        .add_filter("OneDay export", &[extension])
        .blocking_save_file();
    let Some(path) = selected_path(selection)? else {
        return Ok(TransferResult {
            cancelled: true,
            message: "Export cancelled.".into(),
            story_id: None,
            path: None,
        });
    };
    let path = add_extension(path, extension);
    tokio::fs::write(&path, bytes)
        .await
        .map_err(|error| format!("Could not save the export: {error}"))?;
    Ok(TransferResult {
        cancelled: false,
        message: format!("Saved {}.", path.display()),
        story_id: Some(story_id.to_owned()),
        path: Some(path.to_string_lossy().into_owned()),
    })
}

fn filename_from_disposition(value: &str) -> Option<String> {
    let value = value
        .split(';')
        .map(str::trim)
        .find_map(|part| part.strip_prefix("filename="))?
        .trim_matches('"');
    let name = Path::new(value).file_name()?.to_str()?.trim();
    if name.is_empty() {
        None
    } else {
        Some(name.to_owned())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn sanitizes_server_filename_and_extension() {
        assert_eq!(
            filename_from_disposition("attachment; filename=\"story.zip\""),
            Some("story.zip".into())
        );
        assert_eq!(
            filename_from_disposition("attachment; filename=\"../story.zip\""),
            Some("story.zip".into())
        );
        assert_eq!(
            add_extension(PathBuf::from("story"), "zip"),
            PathBuf::from("story.zip")
        );
        assert_eq!(
            add_extension(PathBuf::from("story.ZIP"), "zip"),
            PathBuf::from("story.ZIP")
        );
    }

    #[test]
    fn rejects_path_like_story_ids() {
        assert!(safe_story_id("../secret").is_err());
        assert!(safe_story_id("story/child").is_err());
        assert_eq!(safe_story_id("story-id").unwrap(), "story-id");
    }
}
