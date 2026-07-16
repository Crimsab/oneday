#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod config;
mod portability;
mod remote;
mod updater;

use config::ServerSettings;
use reqwest::redirect::Policy;
use serde::Serialize;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Mutex;
use std::time::Duration;
use tauri::menu::{Menu, MenuItem};
use tauri::tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent};
use tauri::{AppHandle, Manager, RunEvent, State, WindowEvent};
use tauri_plugin_autostart::MacosLauncher;
use url::Url;

pub struct AppRuntime {
    server_url: Mutex<Option<Url>>,
    client: reqwest::Client,
    started_minimized: bool,
    quitting: AtomicBool,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct DesktopStateView {
    server_url: Option<String>,
    started_minimized: bool,
    updater: updater::UpdaterStatus,
}

#[tauri::command]
fn desktop_state(state: State<'_, AppRuntime>) -> Result<DesktopStateView, String> {
    Ok(DesktopStateView {
        server_url: state
            .server_url
            .lock()
            .map_err(|_| "Desktop connection state is unavailable.".to_string())?
            .as_ref()
            .map(Url::to_string),
        started_minimized: state.started_minimized,
        updater: updater::status(),
    })
}

async fn probe_server(client: &reqwest::Client, server: &Url) -> Result<(), String> {
    let url = server
        .join("api/health")
        .map_err(|error| format!("Could not build the health-check URL: {error}"))?;
    let response = client
        .get(url)
        .send()
        .await
        .map_err(|error| format!("Could not reach the OneDay server: {error}"))?;
    if !response.status().is_success() {
        return Err(format!(
            "The server health check returned {}. Verify the URL and reverse-proxy access.",
            response.status()
        ));
    }
    let payload: serde_json::Value = response
        .json()
        .await
        .map_err(|_| "The URL responded, but it is not a compatible OneDay gateway.".to_string())?;
    if payload.get("status").and_then(|value| value.as_str()) != Some("ok") {
        return Err("The URL responded, but the OneDay gateway is not healthy.".into());
    }
    Ok(())
}

#[tauri::command]
async fn connect_server(
    app: AppHandle,
    state: State<'_, AppRuntime>,
    server_url: String,
) -> Result<(), String> {
    let server = config::validate_server_url(&server_url)?;
    probe_server(&state.client, &server).await?;
    config::save(
        &app,
        &ServerSettings {
            server_url: server.to_string(),
        },
    )?;
    *state
        .server_url
        .lock()
        .map_err(|_| "Desktop connection state is unavailable.".to_string())? =
        Some(server.clone());
    remote::open(&app, &server)
}

#[tauri::command]
fn show_story_window(app: AppHandle) -> Result<(), String> {
    remote::show(&app)
}

fn show_settings(app: &AppHandle) {
    if let Some(window) = app.get_webview_window("settings") {
        let _ = window.show();
        let _ = window.unminimize();
        let _ = window.set_focus();
    }
}

fn create_tray(app: &AppHandle) -> tauri::Result<()> {
    let show = MenuItem::with_id(app, "show", "Open OneDay", true, None::<&str>)?;
    let settings = MenuItem::with_id(app, "settings", "Desktop settings", true, None::<&str>)?;
    let quit = MenuItem::with_id(app, "quit", "Quit", true, None::<&str>)?;
    let menu = Menu::with_items(app, &[&show, &settings, &quit])?;
    let mut tray = TrayIconBuilder::new()
        .tooltip("OneDay")
        .menu(&menu)
        .show_menu_on_left_click(false)
        .on_menu_event(|app, event| match event.id().as_ref() {
            "show" => {
                if remote::show(app).is_err() {
                    show_settings(app);
                }
            }
            "settings" => show_settings(app),
            "quit" => {
                app.state::<AppRuntime>()
                    .quitting
                    .store(true, Ordering::SeqCst);
                app.exit(0);
            }
            _ => {}
        })
        .on_tray_icon_event(|tray, event| {
            if matches!(
                event,
                TrayIconEvent::Click {
                    button: MouseButton::Left,
                    button_state: MouseButtonState::Up,
                    ..
                }
            ) {
                let app = tray.app_handle();
                if remote::show(app).is_err() {
                    show_settings(app);
                }
            }
        });
    if let Some(icon) = app.default_window_icon() {
        tray = tray.icon(icon.clone());
    }
    tray.build(app)?;
    Ok(())
}

fn main() {
    let started_minimized = std::env::args().any(|argument| argument == "--minimized");
    let client = reqwest::Client::builder()
        .connect_timeout(Duration::from_secs(6))
        .timeout(Duration::from_secs(120))
        .redirect(Policy::none())
        .user_agent(concat!("OneDay-Desktop/", env!("CARGO_PKG_VERSION")))
        .build()
        .expect("desktop HTTP client");
    let builder = tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_autostart::init(
            MacosLauncher::LaunchAgent,
            Some(vec!["--minimized"]),
        ));
    let builder = match updater::plugin() {
        Ok(Some(plugin)) => builder.plugin(plugin),
        Ok(None) => builder,
        Err(error) => panic!("invalid signed updater configuration: {error}"),
    };
    let app = builder
        .manage(AppRuntime {
            server_url: Mutex::new(None),
            client,
            started_minimized,
            quitting: AtomicBool::new(false),
        })
        .invoke_handler(tauri::generate_handler![
            desktop_state,
            connect_server,
            show_story_window,
            portability::list_remote_stories,
            portability::choose_and_import_story,
            portability::choose_and_export_story,
            updater::updater_status,
            updater::check_and_install_update,
        ])
        .setup(move |app| {
            if let Some(settings) = config::load(app.handle())? {
                let server = config::validate_server_url(&settings.server_url)?;
                *app.state::<AppRuntime>().server_url.lock().map_err(|_| {
                    std::io::Error::other("desktop connection state is unavailable")
                })? = Some(server);
            }
            create_tray(app.handle())?;
            if !started_minimized {
                show_settings(app.handle());
            }
            Ok(())
        })
        .on_window_event(|window, event| {
            if let WindowEvent::CloseRequested { api, .. } = event {
                let state = window.state::<AppRuntime>();
                if !state.quitting.load(Ordering::SeqCst)
                    && matches!(window.label(), "main" | "settings")
                {
                    api.prevent_close();
                    let _ = window.hide();
                }
            }
        })
        .build(tauri::generate_context!())
        .expect("build OneDay desktop application");

    app.run(|app, event| {
        if let RunEvent::ExitRequested { api, .. } = event {
            if !app.state::<AppRuntime>().quitting.load(Ordering::SeqCst) {
                api.prevent_exit();
            }
        }
    });
}
