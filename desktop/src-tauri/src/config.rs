use serde::{Deserialize, Serialize};
use std::fs;
use std::io::Write;
use std::path::{Path, PathBuf};
use tauri::{AppHandle, Manager};
use url::Url;

#[derive(Clone, Debug, Deserialize, Serialize, PartialEq, Eq)]
#[serde(
    tag = "mode",
    rename_all = "snake_case",
    rename_all_fields = "camelCase"
)]
pub enum Profile {
    Remote { server_url: String },
    Standalone { profile_id: String },
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct DesktopSettings {
    pub profile: Profile,
}

// Reads the remote-only v1 settings once, then writes the explicit profile
// shape on the next successful profile switch.
#[derive(Deserialize)]
struct LegacyServerSettings {
    server_url: String,
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

pub fn standalone_profile_dir(app: &AppHandle, profile_id: &str) -> Result<PathBuf, String> {
    if profile_id.len() != 32 || !profile_id.bytes().all(|byte| byte.is_ascii_hexdigit()) {
        return Err("The standalone profile ID is invalid.".into());
    }
    app.path()
        .app_data_dir()
        .map(|path| path.join("profiles").join(profile_id))
        .map_err(|error| {
            format!("Could not locate the standalone application data directory: {error}")
        })
}

pub fn create_initial_standalone_config(profile_dir: &Path) -> Result<PathBuf, String> {
    fs::create_dir_all(profile_dir)
        .map_err(|error| format!("Could not create the standalone profile directory: {error}"))?;
    restrict_directory(profile_dir)?;
    let config_path = profile_dir.join("config.yaml");
    if config_path.exists() {
        return Ok(config_path);
    }
    let data_dir = profile_dir.join("data");
    let contents = initial_standalone_config(&data_dir)?;
    let mut temporary = tempfile::NamedTempFile::new_in(profile_dir)
        .map_err(|error| format!("Could not stage standalone configuration: {error}"))?;
    restrict_file(temporary.path())?;
    temporary
        .write_all(contents.as_bytes())
        .and_then(|()| temporary.as_file().sync_all())
        .map_err(|error| format!("Could not write standalone configuration: {error}"))?;
    match temporary.persist_noclobber(&config_path) {
        Ok(_) => Ok(config_path),
        Err(error) if error.error.kind() == std::io::ErrorKind::AlreadyExists => Ok(config_path),
        Err(error) => Err(format!(
            "Could not save standalone configuration: {}",
            error.error
        )),
    }
}

fn initial_standalone_config(data_dir: &Path) -> Result<String, String> {
    let absolute = if data_dir.is_absolute() {
        data_dir.to_path_buf()
    } else {
        std::env::current_dir()
            .map_err(|error| format!("Could not resolve the standalone data directory: {error}"))?
            .join(data_dir)
    };
    let quoted = serde_json::to_string(&absolute.to_string_lossy())
        .map_err(|error| format!("Could not serialize standalone configuration: {error}"))?;
    Ok(format!("data_dir: {quoted}\n"))
}

pub fn load(app: &AppHandle) -> Result<Option<DesktopSettings>, String> {
    let path = settings_path(app)?;
    if !path.exists() {
        return Ok(None);
    }
    let bytes =
        fs::read(&path).map_err(|error| format!("Could not read desktop settings: {error}"))?;
    if let Ok(settings) = serde_json::from_slice::<DesktopSettings>(&bytes) {
        validate_profile(&settings.profile)?;
        return Ok(Some(settings));
    }
    let legacy: LegacyServerSettings = serde_json::from_slice(&bytes).map_err(|_| {
        "Desktop settings are invalid. Choose a connection profile again.".to_string()
    })?;
    validate_server_url(&legacy.server_url)?;
    Ok(Some(DesktopSettings {
        profile: Profile::Remote {
            server_url: legacy.server_url,
        },
    }))
}

pub fn save(app: &AppHandle, settings: &DesktopSettings) -> Result<(), String> {
    validate_profile(&settings.profile)?;
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

fn validate_profile(profile: &Profile) -> Result<(), String> {
    match profile {
        Profile::Remote { server_url } => validate_server_url(server_url).map(|_| ()),
        Profile::Standalone { profile_id }
            if profile_id.len() == 32
                && profile_id.bytes().all(|byte| byte.is_ascii_hexdigit()) =>
        {
            Ok(())
        }
        Profile::Standalone { .. } => Err("The standalone profile ID is invalid.".into()),
    }
}

#[cfg(unix)]
fn restrict_file(path: &Path) -> Result<(), String> {
    use std::os::unix::fs::PermissionsExt;
    fs::set_permissions(path, fs::Permissions::from_mode(0o600))
        .map_err(|error| format!("Could not secure desktop settings: {error}"))
}

#[cfg(unix)]
fn restrict_directory(path: &Path) -> Result<(), String> {
    use std::os::unix::fs::PermissionsExt;
    fs::set_permissions(path, fs::Permissions::from_mode(0o700))
        .map_err(|error| format!("Could not secure desktop profile directory: {error}"))
}

#[cfg(not(unix))]
fn restrict_file(_path: &Path) -> Result<(), String> {
    Ok(())
}

#[cfg(not(unix))]
fn restrict_directory(_path: &Path) -> Result<(), String> {
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

    #[test]
    fn standalone_profiles_are_isolated_and_remote_profiles_hold_no_data_path() {
        let standalone = Profile::Standalone {
            profile_id: "a".repeat(32),
        };
        assert!(validate_profile(&standalone).is_ok());
        assert!(validate_profile(&Profile::Remote {
            server_url: "https://oneday.example.com".into(),
        })
        .is_ok());
        assert!(validate_profile(&Profile::Standalone {
            profile_id: "../not-a-profile".into(),
        })
        .is_err());
    }

    #[test]
    fn initial_standalone_config_keeps_existing_settings() {
        let temporary = tempfile::tempdir().expect("profile root");
        let config_path = temporary.path().join("config.yaml");
        fs::write(
            &config_path,
            "data_dir: /already-configured\nprovider: local\n",
        )
        .expect("existing config");

        let returned = create_initial_standalone_config(temporary.path()).expect("config path");

        assert_eq!(returned, config_path);
        assert_eq!(
            fs::read_to_string(config_path).expect("preserved config"),
            "data_dir: /already-configured\nprovider: local\n"
        );
    }

    #[test]
    fn initial_standalone_config_uses_an_absolute_data_dir() {
        let temporary = tempfile::tempdir().expect("profile root");
        let config_path = create_initial_standalone_config(temporary.path()).expect("config path");
        let contents = fs::read_to_string(config_path).expect("config");
        let expected = temporary.path().join("data").to_string_lossy().to_string();
        assert_eq!(
            contents,
            format!("data_dir: {}\n", serde_json::to_string(&expected).unwrap())
        );
    }

    #[cfg(unix)]
    #[test]
    fn initial_standalone_config_is_private() {
        use std::os::unix::fs::PermissionsExt;

        let temporary = tempfile::tempdir().expect("profile root");
        let config_path = create_initial_standalone_config(temporary.path()).expect("config path");
        assert_eq!(
            fs::metadata(config_path).unwrap().permissions().mode() & 0o777,
            0o600
        );
        assert_eq!(
            fs::metadata(temporary.path()).unwrap().permissions().mode() & 0o777,
            0o700
        );
    }
}
