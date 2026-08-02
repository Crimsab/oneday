#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod bootstrap;
mod claude_component;
mod codex_component;
mod config;
mod containment;
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
use std::time::{Duration, Instant};
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
    startup_warning: Mutex<Option<String>>,
    local_operation: tokio::sync::Mutex<()>,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct DesktopStateView {
    profile: Option<Profile>,
    server_url: Option<String>,
    lifecycle: Lifecycle,
    started_minimized: bool,
    updater: updater::UpdaterStatus,
    startup_warning: Option<String>,
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

fn unexpected_local_exit(status: impl std::fmt::Display, log_path: &std::path::Path) -> String {
    format!(
        "The local OneDay service stopped unexpectedly ({status}). Choose Open OneDay to restart it. Diagnostics: {}",
        log_path.display()
    )
}

fn refresh_local_lifecycle(state: &AppRuntime) -> Result<(), String> {
    if !matches!(read_lifecycle(state)?, Lifecycle::Ready { .. }) {
        return Ok(());
    }
    let stopped = {
        let mut local = state
            .local
            .lock()
            .map_err(|_| "Desktop lifecycle state is unavailable.".to_string())?;
        match local.as_mut() {
            Some(process) => process
                .exit_status()?
                .map(|status| (status, process.gateway_log_path().to_path_buf())),
            None => None,
        }
    };
    if let Some((status, log_path)) = stopped {
        bootstrap::record(format!(
            "local gateway exited unexpectedly: status={status}; log={}",
            log_path.display()
        ));
        *state
            .server_url
            .lock()
            .map_err(|_| "Desktop connection state is unavailable.".to_string())? = None;
        write_lifecycle(
            state,
            Lifecycle::Failed {
                message: unexpected_local_exit(status, &log_path),
            },
        )?;
    }
    Ok(())
}

#[tauri::command]
fn desktop_state(state: State<'_, AppRuntime>) -> Result<DesktopStateView, String> {
    refresh_local_lifecycle(&state)?;
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
        startup_warning: state
            .startup_warning
            .lock()
            .map_err(|_| "Desktop startup state is unavailable.".to_string())?
            .clone(),
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

async fn wait_for_server(
    client: &reqwest::Client,
    server: &Url,
    max_wait: Duration,
) -> Result<(), String> {
    const RETRY_INTERVAL: Duration = Duration::from_millis(75);
    let started = Instant::now();
    loop {
        match probe_server(client, server).await {
            Ok(()) => return Ok(()),
            Err(error) if started.elapsed() >= max_wait => return Err(error),
            Err(_) => {
                let remaining = max_wait.saturating_sub(started.elapsed());
                tokio::time::sleep(RETRY_INTERVAL.min(remaining)).await;
            }
        }
    }
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
    bootstrap::record(format!(
        "stopping local gateway: pid={}; endpoint={}",
        process.pid(),
        process.endpoint
    ));
    write_lifecycle(state, Lifecycle::Draining)?;
    let result = standalone::stop(process);
    *state
        .server_url
        .lock()
        .map_err(|_| "Desktop connection state is unavailable.".to_string())? = None;
    write_lifecycle(
        state,
        match &result {
            Ok(()) => Lifecycle::Stopped,
            Err(error) => Lifecycle::Failed {
                message: error.clone(),
            },
        },
    )?;
    result
}

#[tauri::command]
async fn show_provider_setup(app: AppHandle, state: State<'_, AppRuntime>) -> Result<(), String> {
    let profile = state
        .profile
        .lock()
        .map_err(|_| "Desktop profile state is unavailable.".to_string())?
        .clone()
        .ok_or_else(|| "Choose where OneDay should run first.".to_string())?;
    if let Profile::Standalone { profile_id } = profile {
        start_local(&app, &state, &profile_id, false).await?;
    }
    let server = state
        .server_url
        .lock()
        .map_err(|_| "Desktop connection state is unavailable.".to_string())?
        .clone()
        .ok_or_else(|| "Start the local OneDay gateway first.".to_string())?;
    probe_server(&state.client, &server).await?;
    let configuration = server
        .join("?overlay=options&section=operator")
        .map_err(|error| format!("Could not build the provider configuration URL: {error}"))?;
    remote::open(&app, &configuration)
}

async fn start_local(
    app: &AppHandle,
    state: &AppRuntime,
    profile_id: &str,
    show_window: bool,
) -> Result<(), String> {
    let _operation = state.local_operation.lock().await;
    start_local_locked(app, state, profile_id, show_window).await
}

async fn start_local_locked(
    app: &AppHandle,
    state: &AppRuntime,
    profile_id: &str,
    show_window: bool,
) -> Result<(), String> {
    bootstrap::record(format!(
        "local gateway start requested: profile={profile_id}; show_window={show_window}"
    ));
    let reusable_endpoint = if matches!(read_lifecycle(state)?, Lifecycle::Ready { .. }) {
        let mut local = state
            .local
            .lock()
            .map_err(|_| "Desktop lifecycle state is unavailable.".to_string())?;
        match local.as_mut() {
            Some(process) => match process.exit_status()? {
                None => Some(process.endpoint.clone()),
                Some(_) => None,
            },
            None => None,
        }
    } else {
        None
    };
    if let Some(endpoint) = reusable_endpoint {
        if probe_server(&state.client, &endpoint).await.is_ok()
            && (!show_window || remote::show(app).is_ok())
        {
            bootstrap::record(format!(
                "reusing healthy local gateway: endpoint={endpoint}"
            ));
            return Ok(());
        }
    }
    stop_local(state)?;
    write_lifecycle(state, Lifecycle::Starting)?;
    let mut latest_error = "The local OneDay gateway did not start.".to_string();
    let mut ready_process = None;
    for attempt in 0..standalone::MAX_START_ATTEMPTS {
        match standalone::start(app, profile_id).await {
            Ok(mut process) => {
                let endpoint = process.endpoint.clone();
                bootstrap::record(format!(
                    "spawned local gateway: attempt={}; pid={}; endpoint={}",
                    attempt + 1,
                    process.pid(),
                    endpoint
                ));
                match wait_for_server(&state.client, &endpoint, Duration::from_secs(12)).await {
                    Ok(()) => {
                        // A process can briefly answer its first health check and
                        // then stop while completing initialization. Do not expose
                        // that dead endpoint to the webview.
                        tokio::time::sleep(Duration::from_millis(300)).await;
                        match process.exit_status() {
                            Ok(Some(status)) => {
                                latest_error =
                                    unexpected_local_exit(status, process.gateway_log_path());
                                let _ = standalone::stop(process);
                            }
                            Ok(None) => match probe_server(&state.client, &endpoint).await {
                                Ok(()) => {
                                    bootstrap::record(format!(
                                        "local gateway passed readiness: pid={}; endpoint={}",
                                        process.pid(),
                                        endpoint
                                    ));
                                    ready_process = Some(process);
                                    break;
                                }
                                Err(error) => {
                                    latest_error = error;
                                    let _ = standalone::stop(process);
                                }
                            },
                            Err(error) => {
                                latest_error = error;
                                let _ = standalone::stop(process);
                            }
                        }
                    }
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
    let Some(process) = ready_process else {
        write_lifecycle(
            state,
            Lifecycle::Failed {
                message: latest_error.clone(),
            },
        )?;
        return Err(latest_error);
    };
    let endpoint = process.endpoint.clone();
    let browser_url = process.browser_url.clone();
    if let Err(error) = remote::open(app, &browser_url) {
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
    bootstrap::record(format!("local gateway committed: endpoint={endpoint}"));
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
    let _operation = state.local_operation.lock().await;
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
    let _operation = state.local_operation.lock().await;
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
    start_local_locked(&app, &state, &profile_id, true).await
}

#[tauri::command]
async fn restart_standalone(app: AppHandle, state: State<'_, AppRuntime>) -> Result<(), String> {
    let _operation = state.local_operation.lock().await;
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
    start_local_locked(&app, &state, &profile_id, true).await
}

#[tauri::command]
async fn stop_standalone(app: AppHandle, state: State<'_, AppRuntime>) -> Result<(), String> {
    let _operation = state.local_operation.lock().await;
    stop_local(&state)?;
    remote::close(&app);
    Ok(())
}

async fn show_story_window_inner(app: &AppHandle, state: &AppRuntime) -> Result<(), String> {
    let profile = state
        .profile
        .lock()
        .map_err(|_| "Desktop profile state is unavailable.".to_string())?
        .clone()
        .ok_or_else(|| "Choose where OneDay should run first.".to_string())?;
    match profile {
        Profile::Standalone { profile_id } => start_local(app, state, &profile_id, true).await,
        Profile::Remote { server_url } => {
            let server = config::validate_server_url(&server_url)?;
            probe_server(&state.client, &server).await?;
            remote::open(app, &server)
        }
    }
}

#[tauri::command]
async fn show_story_window(app: AppHandle, state: State<'_, AppRuntime>) -> Result<(), String> {
    show_story_window_inner(&app, &state).await
}

fn show_settings(app: &AppHandle) -> Result<(), String> {
    if let Some(window) = app.get_webview_window("settings") {
        window.show().map_err(|error| error.to_string())?;
        window.unminimize().map_err(|error| error.to_string())?;
        window.set_focus().map_err(|error| error.to_string())?;
        return Ok(());
    }
    Err("The OneDay setup window was not created.".into())
}

fn show_story_from_tray(app: &AppHandle) {
    let handle = app.clone();
    tauri::async_runtime::spawn(async move {
        let result = {
            let state = handle.state::<AppRuntime>();
            show_story_window_inner(&handle, &state).await
        };
        if let Err(error) = result {
            bootstrap::record(format!("could not open story window from tray: {error}"));
            let _ = show_settings(&handle);
        }
    });
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
            "show" => show_story_from_tray(app),
            "settings" => {
                let _ = show_settings(app);
            }
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
                show_story_from_tray(tray.app_handle());
            }
        });
    if let Some(icon) = app.default_window_icon() {
        tray = tray.icon(icon.clone());
    }
    tray.build(app)?;
    Ok(())
}

fn run() -> Result<(), String> {
    let started_minimized = std::env::args().any(|argument| argument == "--minimized");
    let client = reqwest::Client::builder()
        .connect_timeout(Duration::from_secs(6))
        .timeout(Duration::from_secs(120))
        .redirect(Policy::none())
        .user_agent(concat!("OneDay-Desktop/", env!("CARGO_PKG_VERSION")))
        .build()
        .map_err(|error| format!("Could not create the desktop HTTP client: {error}"))?;
    let builder = tauri::Builder::default()
        .plugin(tauri_plugin_single_instance::init(|app, _args, _cwd| {
            bootstrap::record("a second desktop launch was redirected to the running instance");
            if remote::show(app).is_err() {
                let _ = show_settings(app);
            }
        }))
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_autostart::init(
            MacosLauncher::LaunchAgent,
            Some(vec!["--minimized"]),
        ));
    let builder = match updater::plugin() {
        Ok(Some(plugin)) => builder.plugin(plugin),
        Ok(None) => builder,
        Err(error) => return Err(format!("Invalid signed updater configuration: {error}")),
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
            startup_warning: Mutex::new(None),
            local_operation: tokio::sync::Mutex::new(()),
        })
        .invoke_handler(tauri::generate_handler![
            desktop_state,
            connect_server,
            start_standalone,
            restart_standalone,
            stop_standalone,
            show_story_window,
            show_provider_setup,
            portability::list_remote_stories,
            portability::choose_and_import_story,
            portability::choose_and_export_story,
            updater::updater_status,
            updater::check_update,
            updater::install_update,
            codex_component::codex_status,
            codex_component::install_codex_component,
            codex_component::login_codex,
            claude_component::claude_status,
            claude_component::install_claude,
            claude_component::login_claude,
            claude_component::open_claude_install_guide,
        ])
        .setup(move |app| {
            let settings = match config::load(app.handle()) {
                Ok(settings) => settings,
                Err(error) => {
                    bootstrap::record(format!("desktop settings recovery: {error}"));
                    if let Err(quarantine_error) =
                        config::quarantine_invalid_settings(app.handle())
                    {
                        bootstrap::record(format!(
                            "could not quarantine desktop settings: {quarantine_error}"
                        ));
                    }
                    *app.state::<AppRuntime>()
                        .startup_warning
                        .lock()
                        .map_err(|_| {
                            std::io::Error::other("desktop startup state is unavailable")
                        })? = Some(
                        "OneDay ignored invalid desktop connection settings. Story data was not changed."
                            .into(),
                    );
                    None
                }
            };
            let standalone_id =
                if let Some(settings) = settings {
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
                    }
                    *app.state::<AppRuntime>().profile.lock().map_err(|_| {
                        std::io::Error::other("desktop profile state is unavailable")
                    })? = Some(profile);
                    standalone_id
                } else {
                    None
                };
            if let Err(error) = create_tray(app.handle()) {
                bootstrap::record(format!("system tray unavailable: {error}"));
            }
            if started_minimized {
                if let Some(window) = app.get_webview_window("settings") {
                    let _ = window.hide();
                }
                if let Some(profile_id) = standalone_id {
                    let handle = app.handle().clone();
                    tauri::async_runtime::spawn(async move {
                        let state = handle.state::<AppRuntime>();
                        let _ = start_local(&handle, &state, &profile_id, false).await;
                    });
                }
            }
            if !started_minimized {
                show_settings(app.handle()).map_err(std::io::Error::other)?;
            }
            bootstrap::record("OneDay desktop setup completed");
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
        .map_err(|error| format!("Could not build the OneDay desktop application: {error}"))?;

    app.run(|app, event| {
        if let RunEvent::ExitRequested { api, .. } = event {
            if !app.state::<AppRuntime>().quitting.load(Ordering::SeqCst) {
                api.prevent_exit();
            }
        }
    });
    Ok(())
}

fn main() {
    bootstrap::install_panic_hook();
    bootstrap::record(format!(
        "OneDay desktop {} starting",
        env!("CARGO_PKG_VERSION")
    ));
    if let Err(error) = run() {
        bootstrap::fatal(&error);
        std::process::exit(1);
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::{Read, Write};
    use std::net::TcpListener;
    use std::thread;

    #[test]
    fn standalone_readiness_retries_until_a_delayed_gateway_is_listening() {
        tauri::async_runtime::block_on(async {
            let reservation = TcpListener::bind("127.0.0.1:0").expect("reserve port");
            let address = reservation.local_addr().expect("reserved address");
            drop(reservation);
            let server = Url::parse(&format!("http://{address}/")).expect("server URL");
            let responder = thread::spawn(move || {
                thread::sleep(Duration::from_millis(175));
                let listener = TcpListener::bind(address).expect("delayed listener");
                let (mut stream, _) = listener.accept().expect("health request");
                let mut request = [0_u8; 1024];
                let _ = stream.read(&mut request).expect("read health request");
                stream
                    .write_all(
                        b"HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 15\r\nConnection: close\r\n\r\n{\"status\":\"ok\"}",
                    )
                    .expect("write health response");
            });
            let client = reqwest::Client::builder()
                .timeout(Duration::from_secs(1))
                .build()
                .expect("client");

            wait_for_server(&client, &server, Duration::from_secs(2))
                .await
                .expect("delayed gateway becomes ready");
            responder.join().expect("responder");
        });
    }

    #[test]
    fn unexpected_gateway_exit_points_to_recovery_and_diagnostics() {
        let message =
            unexpected_local_exit("exit code 7", std::path::Path::new("C:/OneDay/gateway.log"));
        assert!(message.contains("Choose Open OneDay to restart it"));
        assert!(message.contains("exit code 7"));
        assert!(message.contains("gateway.log"));
    }
}
