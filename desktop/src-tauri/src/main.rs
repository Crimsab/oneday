#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod config;
mod lifecycle;
mod portability;
mod remote;
mod secret;
mod standalone;
mod updater;

use config::{DesktopSettings, Profile};
use lifecycle::Lifecycle;
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
    profile: Mutex<Option<Profile>>,
    server_url: Mutex<Option<Url>>,
    local: Mutex<Option<standalone::LocalProcess>>,
    lifecycle: Mutex<Lifecycle>,
    client: reqwest::Client,
    started_minimized: bool,
    quitting: AtomicBool,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct DesktopStateView {
    profile: Option<Profile>,
    server_url: Option<String>,
    lifecycle: Lifecycle,
    started_minimized: bool,
    updater: updater::UpdaterStatus,
}

fn read_lifecycle(state: &AppRuntime) -> Result<Lifecycle, String> {
    state
        .lifecycle
        .lock()
        .map_err(|_| "Desktop lifecycle state is unavailable.".to_string())
        .map(|value| value.clone())
}

fn write_lifecycle(state: &AppRuntime, lifecycle: Lifecycle) -> Result<(), String> {
    *state
        .lifecycle
        .lock()
        .map_err(|_| "Desktop lifecycle state is unavailable.".to_string())? = lifecycle;
    Ok(())
}

#[tauri::command]
fn desktop_state(state: State<'_, AppRuntime>) -> Result<DesktopStateView, String> {
    Ok(DesktopStateView {
        profile: state
            .profile
            .lock()
            .map_err(|_| "Desktop profile state is unavailable.".to_string())?
            .clone(),
        server_url: state
            .server_url
            .lock()
            .map_err(|_| "Desktop connection state is unavailable.".to_string())?
            .as_ref()
            .map(Url::to_string),
        lifecycle: read_lifecycle(&state)?,
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

fn stop_local(state: &AppRuntime) -> Result<(), String> {
    let process = state
        .local
        .lock()
        .map_err(|_| "Desktop lifecycle state is unavailable.".to_string())?
        .take();
    let Some(process) = process else {
        return Ok(());
    };
    write_lifecycle(state, Lifecycle::Draining)?;
    let result = standalone::stop(process);
    *state
        .server_url
        .lock()
        .map_err(|_| "Desktop connection state is unavailable.".to_string())? = None;
    write_lifecycle(
        state,
        match result {
            Ok(()) => Lifecycle::Stopped,
            Err(error) => Lifecycle::Failed {
                message: error.clone(),
            },
        },
    )?;
    result
}

async fn start_local(
    app: &AppHandle,
    state: &AppRuntime,
    profile_id: &str,
    show_window: bool,
) -> Result<(), String> {
    let has_local_process = state
        .local
        .lock()
        .map_err(|_| "Desktop lifecycle state is unavailable.".to_string())?
        .is_some();
    if has_local_process && matches!(read_lifecycle(state)?, Lifecycle::Ready { .. }) {
        return Ok(());
    }
    stop_local(state)?;
    write_lifecycle(state, Lifecycle::Starting)?;
    let mut latest_error = "The local OneDay gateway did not start.".to_string();
    let process = 'launch: loop {
        for attempt in 0..standalone::MAX_START_ATTEMPTS {
            match standalone::start(app, profile_id) {
                Ok(process) => {
                    let endpoint = process.endpoint.clone();
                    match tokio::time::timeout(
                        Duration::from_secs(12),
                        probe_server(&state.client, &endpoint),
                    )
                    .await
                    .map_err(|_| {
                        "The local OneDay gateway did not become ready in time.".to_string()
                    })
                    .and_then(|result| result)
                    {
                        Ok(()) => break 'launch process,
                        Err(error) => {
                            latest_error = error;
                            let _ = standalone::stop(process);
                        }
                    }
                }
                Err(error) => latest_error = error,
            }
            if attempt + 1 < standalone::MAX_START_ATTEMPTS {
                tokio::time::sleep(Duration::from_millis(150)).await;
            }
        }
        write_lifecycle(
            state,
            Lifecycle::Failed {
                message: latest_error.clone(),
            },
        )?;
        return Err(latest_error);
    };
    let endpoint = process.endpoint.clone();
    if let Err(error) = remote::open(app, &endpoint) {
        let _ = standalone::stop(process);
        write_lifecycle(
            state,
            Lifecycle::Failed {
                message: error.clone(),
            },
        )?;
        return Err(error);
    }
    if !show_window {
        if let Some(window) = app.get_webview_window("main") {
            let _ = window.hide();
        }
    }
    *state
        .server_url
        .lock()
        .map_err(|_| "Desktop connection state is unavailable.".to_string())? =
        Some(endpoint.clone());
    *state
        .local
        .lock()
        .map_err(|_| "Desktop lifecycle state is unavailable.".to_string())? = Some(process);
    write_lifecycle(
        state,
        Lifecycle::Ready {
            endpoint: endpoint.to_string(),
        },
    )
}

#[tauri::command]
async fn connect_server(
    app: AppHandle,
    state: State<'_, AppRuntime>,
    server_url: String,
) -> Result<(), String> {
    let server = config::validate_server_url(&server_url)?;
    probe_server(&state.client, &server).await?;
    stop_local(&state)?;
    config::save(
        &app,
        &DesktopSettings {
            profile: Profile::Remote {
                server_url: server.to_string(),
            },
        },
    )?;
    *state
        .profile
        .lock()
        .map_err(|_| "Desktop profile state is unavailable.".to_string())? =
        Some(Profile::Remote {
            server_url: server.to_string(),
        });
    *state
        .server_url
        .lock()
        .map_err(|_| "Desktop connection state is unavailable.".to_string())? =
        Some(server.clone());
    write_lifecycle(
        &state,
        Lifecycle::Ready {
            endpoint: server.to_string(),
        },
    )?;
    remote::open(&app, &server)
}

#[tauri::command]
async fn start_standalone(app: AppHandle, state: State<'_, AppRuntime>) -> Result<(), String> {
    let existing = state
        .profile
        .lock()
        .map_err(|_| "Desktop profile state is unavailable.".to_string())?
        .clone();
    let profile_id = match &existing {
        Some(Profile::Standalone { profile_id }) => profile_id.clone(),
        _ => secret::profile_id()?,
    };
    if !matches!(existing, Some(Profile::Standalone { .. })) {
        stop_local(&state)?;
        config::save(
            &app,
            &DesktopSettings {
                profile: Profile::Standalone {
                    profile_id: profile_id.clone(),
                },
            },
        )?;
        *state
            .profile
            .lock()
            .map_err(|_| "Desktop profile state is unavailable.".to_string())? =
            Some(Profile::Standalone {
                profile_id: profile_id.clone(),
            });
    }
    start_local(&app, &state, &profile_id, true).await
}

#[tauri::command]
async fn restart_standalone(app: AppHandle, state: State<'_, AppRuntime>) -> Result<(), String> {
    let profile_id = match state
        .profile
        .lock()
        .map_err(|_| "Desktop profile state is unavailable.".to_string())?
        .clone()
    {
        Some(Profile::Standalone { profile_id }) => profile_id,
        _ => return Err("Choose standalone mode before restarting the local gateway.".into()),
    };
    stop_local(&state)?;
    start_local(&app, &state, &profile_id, true).await
}

#[tauri::command]
fn stop_standalone(app: AppHandle, state: State<'_, AppRuntime>) -> Result<(), String> {
    stop_local(&state)?;
    remote::close(&app);
    Ok(())
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
                let _ = stop_local(&app.state::<AppRuntime>());
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
            profile: Mutex::new(None),
            server_url: Mutex::new(None),
            local: Mutex::new(None),
            lifecycle: Mutex::new(Lifecycle::Stopped),
            client,
            started_minimized,
            quitting: AtomicBool::new(false),
        })
        .invoke_handler(tauri::generate_handler![
            desktop_state,
            connect_server,
            start_standalone,
            restart_standalone,
            stop_standalone,
            show_story_window,
            portability::list_remote_stories,
            portability::choose_and_import_story,
            portability::choose_and_export_story,
            updater::updater_status,
            updater::check_and_install_update,
        ])
        .setup(move |app| {
            let standalone_id =
                if let Some(settings) = config::load(app.handle())? {
                    let profile = settings.profile;
                    let standalone_id = match &profile {
                        Profile::Standalone { profile_id } => Some(profile_id.clone()),
                        Profile::Remote { .. } => None,
                    };
                    if let Profile::Remote { server_url } = &profile {
                        let server = config::validate_server_url(server_url)?;
                        *app.state::<AppRuntime>().server_url.lock().map_err(|_| {
                            std::io::Error::other("desktop connection state is unavailable")
                        })? = Some(server.clone());
                        *app.state::<AppRuntime>().lifecycle.lock().map_err(|_| {
                            std::io::Error::other("desktop lifecycle state is unavailable")
                        })? = Lifecycle::Ready {
                            endpoint: server.to_string(),
                        };
                    }
                    *app.state::<AppRuntime>().profile.lock().map_err(|_| {
                        std::io::Error::other("desktop profile state is unavailable")
                    })? = Some(profile);
                    standalone_id
                } else {
                    None
                };
            create_tray(app.handle())?;
            if started_minimized {
                if let Some(profile_id) = standalone_id {
                    let handle = app.handle().clone();
                    tauri::async_runtime::spawn(async move {
                        let state = handle.state::<AppRuntime>();
                        let _ = start_local(&handle, &state, &profile_id, false).await;
                    });
                }
            }
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
