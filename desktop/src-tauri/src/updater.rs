use crate::{stop_local, AppRuntime};
use base64::Engine;
use serde::Serialize;
use tauri::plugin::TauriPlugin;
use tauri::{AppHandle, Manager, Runtime, Wry};
use tauri_plugin_updater::{Update, UpdaterExt};
use url::Url;

// The updater trust root is public by design. It lets every desktop build check
// the official release feed; only the private signing key can create an update
// that this key accepts.
const DEFAULT_UPDATER_ENDPOINT: &str =
    "https://github.com/Crimsab/oneday/releases/latest/download/latest.json";
const DEFAULT_UPDATER_PUBKEY: &str = "dW50cnVzdGVkIGNvbW1lbnQ6IG1pbmlzaWduIHB1YmxpYyBrZXk6IDgzQzU4RUI4NjMyMTlGQ0YKUldUUG55Rmp1STdGZ3lqMExJZnBIVmtCQ3BzazhKWDVmWkhWeVFHalhDY1lJNHlrbkhQaFk3UHYK";

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdaterStatus {
    pub enabled: bool,
    pub current_version: String,
    pub channel: String,
    pub reason: String,
}

fn configuration() -> Result<(Url, String), String> {
    let endpoint = option_env!("ONEDAY_UPDATER_ENDPOINT")
        .filter(|value| !value.trim().is_empty())
        .unwrap_or(DEFAULT_UPDATER_ENDPOINT);
    let pubkey = option_env!("ONEDAY_UPDATER_PUBKEY")
        .filter(|value| !value.trim().is_empty())
        .unwrap_or(DEFAULT_UPDATER_PUBKEY);
    let endpoint =
        Url::parse(endpoint).map_err(|_| "The OneDay update endpoint is invalid.".to_string())?;
    if endpoint.scheme() != "https" {
        return Err("The OneDay update endpoint must use HTTPS.".into());
    }
    validate_public_key(pubkey)?;
    Ok((endpoint, pubkey.to_owned()))
}

fn validate_public_key(value: &str) -> Result<(), String> {
    let decoded = base64::engine::general_purpose::STANDARD
        .decode(value.trim())
        .map_err(|_| "The OneDay updater public key is not valid base64.".to_string())?;
    let decoded = std::str::from_utf8(&decoded)
        .map_err(|_| "The OneDay updater public key is not valid UTF-8.".to_string())?;
    let mut lines = decoded.lines().filter(|line| !line.trim().is_empty());
    let comment = lines.next().unwrap_or_default();
    let key = lines.next().unwrap_or_default();
    if !comment.starts_with("untrusted comment: minisign public key:")
        || key.len() != 56
        || !key
            .bytes()
            .all(|byte| byte.is_ascii_alphanumeric() || byte == b'+' || byte == b'/')
        || lines.next().is_some()
    {
        return Err("The OneDay updater public key has an invalid Minisign format.".into());
    }
    Ok(())
}

pub fn status() -> UpdaterStatus {
    match configuration() {
        Ok(_) => UpdaterStatus {
            enabled: true,
            current_version: env!("CARGO_PKG_VERSION").into(),
            channel: "Stable".into(),
            reason: "OneDay checks the official release feed. It verifies every downloaded update with the embedded public key before installation.".into(),
        },
        Err(reason) => UpdaterStatus {
            enabled: false,
            current_version: env!("CARGO_PKG_VERSION").into(),
            channel: "Manual updates".into(),
            reason,
        },
    }
}

pub fn plugin() -> Result<Option<TauriPlugin<Wry, tauri_plugin_updater::Config>>, String> {
    let Ok((_, pubkey)) = configuration() else {
        return Ok(None);
    };
    let plugin = tauri_plugin_updater::Builder::new().pubkey(pubkey).build();
    Ok(Some(plugin))
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdateCheck {
    available: bool,
    version: Option<String>,
    notes: Option<String>,
    published_at: Option<String>,
    message: String,
}

fn describe(update: Option<&Update>) -> UpdateCheck {
    let Some(update) = update else {
        return UpdateCheck {
            available: false,
            version: None,
            notes: None,
            published_at: None,
            message: "OneDay is up to date.".into(),
        };
    };
    UpdateCheck {
        available: true,
        version: Some(update.version.clone()),
        notes: update.body.clone(),
        published_at: update.date.map(|date| date.to_string()),
        message: format!(
            "OneDay {} is available. Review it before installing.",
            update.version
        ),
    }
}

async fn available_update<R: Runtime>(app: &AppHandle<R>) -> Result<Option<Update>, String> {
    if !status().enabled {
        return Err(status().reason);
    }
    let (endpoint, pubkey) = configuration()?;
    app.updater_builder()
        .endpoints(vec![endpoint])
        .map_err(|error| format!("Could not configure signed updates: {error}"))?
        .pubkey(pubkey)
        .build()
        .map_err(|error| format!("Could not initialize signed updates: {error}"))?
        .check()
        .await
        .map_err(|error| format!("Could not check for updates: {error}"))
}

#[tauri::command]
pub fn updater_status() -> UpdaterStatus {
    status()
}

#[tauri::command]
pub async fn check_update<R: Runtime>(app: AppHandle<R>) -> Result<UpdateCheck, String> {
    let update = available_update(&app).await?;
    Ok(describe(update.as_ref()))
}

#[tauri::command]
pub async fn install_update<R: Runtime>(app: AppHandle<R>) -> Result<(), String> {
    let Some(update) = available_update(&app).await? else {
        return Err("OneDay is already up to date.".into());
    };
    let bytes = update
        .download(|_, _| {}, || {})
        .await
        .map_err(|error| format!("Could not download or verify the signed update: {error}"))?;

    // The gateway owns the local story database. Stop it cleanly only after the
    // complete update has been downloaded and its signature has been verified.
    stop_local(&app.state::<AppRuntime>()).map_err(|error| {
        format!("The update is ready, but OneDay could not stop safely: {error}")
    })?;
    update
        .install(bytes)
        .map_err(|error| format!("Could not install the signed update: {error}"))?;
    app.restart()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn updater_uses_the_embedded_public_release_configuration() {
        let state = status();
        assert!(!state.current_version.is_empty());
        assert!(
            state.enabled,
            "default public updater configuration is valid"
        );
        assert_eq!(state.channel, "Stable");
        let (endpoint, public_key) = configuration().expect("valid updater configuration");
        if option_env!("ONEDAY_UPDATER_ENDPOINT").is_none() {
            assert_eq!(endpoint.as_str(), DEFAULT_UPDATER_ENDPOINT);
        } else {
            assert_eq!(endpoint.scheme(), "https");
        }
        validate_public_key(&public_key).expect("valid updater public key");
    }

    #[test]
    fn tauri_config_contains_a_deserializable_updater_object() {
        let config: serde_json::Value =
            serde_json::from_str(include_str!("../tauri.conf.json")).expect("valid Tauri config");
        let raw = config
            .pointer("/plugins/updater")
            .cloned()
            .expect("plugins.updater must be present");
        assert!(raw.is_object(), "plugins.updater must never be null");

        let plugin: tauri_plugin_updater::Config =
            serde_json::from_value(raw).expect("updater config must deserialize at startup");
        assert_eq!(plugin.pubkey, DEFAULT_UPDATER_PUBKEY);
        assert_eq!(plugin.endpoints.len(), 1);
        assert_eq!(plugin.endpoints[0].as_str(), DEFAULT_UPDATER_ENDPOINT);
    }

    #[test]
    fn no_update_result_is_explicit() {
        let result = describe(None);
        assert!(!result.available);
        assert_eq!(result.message, "OneDay is up to date.");
    }
}
