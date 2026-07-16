use serde::Serialize;
use tauri::plugin::TauriPlugin;
use tauri::{AppHandle, Runtime, Wry};
use tauri_plugin_updater::UpdaterExt;
use url::Url;

#[derive(Clone, Debug, Serialize)]
pub struct UpdaterStatus {
    pub enabled: bool,
    pub reason: String,
}

fn configuration() -> Result<(Url, String), String> {
    let endpoint = option_env!("ONEDAY_UPDATER_ENDPOINT")
        .filter(|value| !value.trim().is_empty())
        .ok_or_else(|| {
            "Signed updates are disabled until a release endpoint and public key are configured."
                .to_string()
        })?;
    let pubkey = option_env!("ONEDAY_UPDATER_PUBKEY")
        .filter(|value| !value.trim().is_empty())
        .ok_or_else(|| {
            "Signed updates are disabled until a release endpoint and public key are configured."
                .to_string()
        })?;
    let endpoint =
        Url::parse(endpoint).map_err(|_| "The updater endpoint is invalid.".to_string())?;
    if endpoint.scheme() != "https" {
        return Err("The updater endpoint must use HTTPS.".into());
    }
    Ok((endpoint, pubkey.to_owned()))
}

pub fn status() -> UpdaterStatus {
    match configuration() {
        Ok(_) => UpdaterStatus {
            enabled: true,
            reason: "Signed updates are configured for this build.".into(),
        },
        Err(reason) => UpdaterStatus {
            enabled: false,
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
    message: String,
}

#[tauri::command]
pub fn updater_status() -> UpdaterStatus {
    status()
}

#[tauri::command]
pub async fn check_and_install_update<R: Runtime>(
    app: AppHandle<R>,
) -> Result<UpdateCheck, String> {
    if !status().enabled {
        return Err(status().reason);
    }
    let (endpoint, pubkey) = configuration()?;
    let update = app
        .updater_builder()
        .endpoints(vec![endpoint])
        .map_err(|error| format!("Could not configure signed updates: {error}"))?
        .pubkey(pubkey)
        .build()
        .map_err(|error| format!("Could not initialize signed updates: {error}"))?
        .check()
        .await
        .map_err(|error| format!("Could not check for updates: {error}"))?;
    let Some(update) = update else {
        return Ok(UpdateCheck {
            available: false,
            version: None,
            message: "OneDay is up to date.".into(),
        });
    };
    let version = update.version.clone();
    update
        .download_and_install(|_, _| {}, || {})
        .await
        .map_err(|error| format!("Could not install the signed update: {error}"))?;
    Ok(UpdateCheck {
        available: true,
        version: Some(version),
        message: "The signed update was installed. Restart OneDay to use it.".into(),
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn updater_is_disabled_without_signed_build_configuration() {
        let state = status();
        if option_env!("ONEDAY_UPDATER_ENDPOINT").is_none() {
            assert!(!state.enabled);
            assert!(state.reason.contains("disabled"));
        }
    }
}
