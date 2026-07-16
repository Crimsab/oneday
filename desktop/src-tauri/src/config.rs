use serde::{Deserialize, Serialize};
use std::fs;
use std::path::{Path, PathBuf};
use tauri::{AppHandle, Manager};
use url::Url;

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct ServerSettings {
    pub server_url: String,
}

pub fn validate_server_url(raw: &str) -> Result<Url, String> {
    let trimmed = raw.trim();
    let mut url = Url::parse(trimmed).map_err(|_| {
        "Enter a complete server URL, for example https://oneday.example.com.".to_string()
    })?;
    if !url.username().is_empty() || url.password().is_some() {
        return Err("Do not put credentials in the server URL.".into());
    }
    if url.query().is_some() || url.fragment().is_some() {
        return Err("The server URL cannot contain a query or fragment.".into());
    }
    let host = url
        .host_str()
        .ok_or_else(|| "The server URL must include a host.".to_string())?;
    let local = host.eq_ignore_ascii_case("localhost") || host == "127.0.0.1" || host == "::1";
    if url.scheme() != "https" && !(url.scheme() == "http" && local) {
        return Err("Use HTTPS. Plain HTTP is only allowed for localhost development.".into());
    }
    if url.path() != "/" && !url.path().is_empty() {
        return Err("Use the OneDay server origin without an extra path.".into());
    }
    url.set_path("/");
    Ok(url)
}

pub fn settings_path(app: &AppHandle) -> Result<PathBuf, String> {
    app.path()
        .app_config_dir()
        .map(|path| path.join("desktop.json"))
        .map_err(|error| format!("Could not locate the desktop configuration directory: {error}"))
}

pub fn load(app: &AppHandle) -> Result<Option<ServerSettings>, String> {
    let path = settings_path(app)?;
    if !path.exists() {
        return Ok(None);
    }
    let bytes =
        fs::read(&path).map_err(|error| format!("Could not read desktop settings: {error}"))?;
    let settings: ServerSettings = serde_json::from_slice(&bytes)
        .map_err(|_| "Desktop settings are invalid. Enter the server URL again.".to_string())?;
    validate_server_url(&settings.server_url)?;
    Ok(Some(settings))
}

pub fn save(app: &AppHandle, settings: &ServerSettings) -> Result<(), String> {
    let path = settings_path(app)?;
    let parent = path
        .parent()
        .ok_or_else(|| "Desktop settings path has no parent directory.".to_string())?;
    fs::create_dir_all(parent)
        .map_err(|error| format!("Could not create desktop settings directory: {error}"))?;
    let temporary = path.with_extension("json.tmp");
    let bytes = serde_json::to_vec_pretty(settings)
        .map_err(|error| format!("Could not serialize desktop settings: {error}"))?;
    fs::write(&temporary, bytes)
        .map_err(|error| format!("Could not stage desktop settings: {error}"))?;
    restrict_file(&temporary)?;
    fs::rename(&temporary, &path)
        .map_err(|error| format!("Could not save desktop settings: {error}"))?;
    Ok(())
}

#[cfg(unix)]
fn restrict_file(path: &Path) -> Result<(), String> {
    use std::os::unix::fs::PermissionsExt;
    fs::set_permissions(path, fs::Permissions::from_mode(0o600))
        .map_err(|error| format!("Could not secure desktop settings: {error}"))
}

#[cfg(not(unix))]
fn restrict_file(_path: &Path) -> Result<(), String> {
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn accepts_https_and_local_development() {
        assert_eq!(
            validate_server_url("https://oneday.example.com")
                .unwrap()
                .as_str(),
            "https://oneday.example.com/"
        );
        assert!(validate_server_url("http://localhost:8788").is_ok());
        assert!(validate_server_url("http://127.0.0.1:8788").is_ok());
    }

    #[test]
    fn rejects_insecure_remote_and_embedded_credentials() {
        assert!(validate_server_url("http://oneday.example.com").is_err());
        assert!(validate_server_url("https://user:pass@oneday.example.com").is_err());
        assert!(validate_server_url("https://oneday.example.com/app").is_err());
        assert!(validate_server_url("https://oneday.example.com?token=value").is_err());
    }
}
