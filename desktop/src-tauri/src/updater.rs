use crate::{stop_local, AppRuntime};
use serde::Serialize;
use tauri::plugin::TauriPlugin;
use tauri::{AppHandle, Manager, Runtime, Wry};
use tauri_plugin_updater::{Update, UpdaterExt};
use url::Url;

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
        .ok_or_else(|| {
            "Signed updates are unavailable in this development build. Release builds check the stable OneDay feed."
                .to_string()
        })?;
    let pubkey = option_env!("ONEDAY_UPDATER_PUBKEY")
        .filter(|value| !value.trim().is_empty())
        .ok_or_else(|| {
            "Signed updates are unavailable in this development build. Release builds check the stable OneDay feed."
                .to_string()
        })?;
    let endpoint =
        Url::parse(endpoint).map_err(|_| "The signed update endpoint is invalid.".to_string())?;
    if endpoint.scheme() != "https" {
        return Err("The signed update endpoint must use HTTPS.".into());
    }
    Ok((endpoint, pubkey.to_owned()))
}

pub fn status() -> UpdaterStatus {
    match configuration() {
        Ok(_) => UpdaterStatus {
            enabled: true,
            current_version: env!("CARGO_PKG_VERSION").into(),
            channel: "Stable".into(),
            reason: "Updates are verified with the OneDay release signing key before installation."
                .into(),
        },
        Err(reason) => UpdaterStatus {
            enabled: false,
            current_version: env!("CARGO_PKG_VERSION").into(),
            channel: "Development".into(),
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
    fn updater_is_disabled_without_signed_build_configuration() {
        let state = status();
        assert!(!state.current_version.is_empty());
        if option_env!("ONEDAY_UPDATER_ENDPOINT").is_none() {
            assert!(!state.enabled);
            assert!(state.reason.contains("development build"));
        }
    }

    #[test]
    fn no_update_result_is_explicit() {
        let result = describe(None);
        assert!(!result.available);
        assert_eq!(result.message, "OneDay is up to date.");
    }
}
