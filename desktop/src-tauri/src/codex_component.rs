use flate2::read::GzDecoder;
use reqwest::redirect::Policy;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::HashMap;
#[cfg(any(windows, test))]
use std::ffi::OsStr;
use std::ffi::OsString;
use std::fs;
use std::io::{self, Write};
use std::path::{Path, PathBuf};
use std::process::Stdio;
use std::time::Duration;
use tauri::{AppHandle, Manager};
use tempfile::NamedTempFile;
use tokio::process::Command;

const MANIFEST: &str = include_str!("../codex-components.json");
const MAX_DOWNLOAD_OVERHEAD: u64 = 1024;
const MAX_BINARY_SIZE: u64 = 512 * 1024 * 1024;
const COMMAND_TIMEOUT: Duration = Duration::from_secs(20);
const LOGIN_TIMEOUT: Duration = Duration::from_secs(10 * 60);

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
struct ComponentManifest {
    version: String,
    release_tag: String,
    targets: HashMap<String, ComponentTarget>,
}

#[derive(Clone, Debug, Deserialize)]
struct ComponentTarget {
    url: String,
    sha256: String,
    size: u64,
    archive: String,
    entry: String,
}

#[derive(Clone, Debug)]
pub struct CodexRuntime {
    pub binary_dir: PathBuf,
    pub home: Option<PathBuf>,
    pub pathext: Option<OsString>,
}

#[derive(Clone, Debug)]
struct Installation {
    binary: PathBuf,
    source: Source,
    launcher: Launcher,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum Source {
    #[cfg_attr(not(windows), allow(dead_code))]
    Global,
    Managed,
    System,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum Launcher {
    Native,
    #[cfg(windows)]
    WindowsCommandShim,
}

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct CodexStatus {
    available: bool,
    state: String,
    source: String,
    version: Option<String>,
    authenticated: bool,
    desktop_app_detected: bool,
    legacy_cli_detected: bool,
    managed_version: String,
    message: String,
    launcher: Option<String>,
    diagnostic_shell: String,
    diagnostic_command: String,
    install_scope: String,
}

fn manifest() -> Result<ComponentManifest, String> {
    let manifest: ComponentManifest = serde_json::from_str(MANIFEST)
        .map_err(|error| format!("The bundled Codex component manifest is invalid: {error}"))?;
    if manifest.release_tag != format!("rust-v{}", manifest.version) {
        return Err("The bundled Codex component manifest has inconsistent versions.".into());
    }
    Ok(manifest)
}

fn target_triple() -> Option<&'static str> {
    match (std::env::consts::OS, std::env::consts::ARCH) {
        ("windows", "x86_64") => Some("x86_64-pc-windows-msvc"),
        ("macos", "aarch64") => Some("aarch64-apple-darwin"),
        ("macos", "x86_64") => Some("x86_64-apple-darwin"),
        ("linux", "x86_64") => Some("x86_64-unknown-linux-gnu"),
        _ => None,
    }
}

fn component_root(app: &AppHandle) -> Result<PathBuf, String> {
    app.path()
        .app_data_dir()
        .map(|path| path.join("components").join("codex"))
        .map_err(|error| format!("Could not locate the desktop component directory: {error}"))
}

fn managed_binary(app: &AppHandle, version: &str) -> Result<PathBuf, String> {
    let name = if cfg!(windows) { "codex.exe" } else { "codex" };
    Ok(component_root(app)?.join(version).join(name))
}

fn managed_home(app: &AppHandle) -> Result<PathBuf, String> {
    Ok(component_root(app)?.join("home"))
}

fn install_scope() -> &'static str {
    if cfg!(windows) {
        "global"
    } else {
        "managed"
    }
}

#[cfg(windows)]
fn global_binary_dir() -> Result<PathBuf, String> {
    let directory = if let Some(configured) = std::env::var_os("CODEX_INSTALL_DIR") {
        PathBuf::from(configured)
    } else {
        let local_app_data = std::env::var_os("LOCALAPPDATA").ok_or_else(|| {
            "Windows did not provide LOCALAPPDATA for the Codex installation.".to_string()
        })?;
        PathBuf::from(local_app_data)
            .join("Programs")
            .join("OpenAI")
            .join("Codex")
            .join("bin")
    };
    if !directory.is_absolute() {
        return Err("CODEX_INSTALL_DIR must be an absolute Windows path.".into());
    }
    Ok(directory)
}

#[cfg(windows)]
fn global_binary() -> Result<PathBuf, String> {
    Ok(global_binary_dir()?.join("codex.exe"))
}

#[cfg(any(windows, test))]
fn normalize_windows_path_entry(value: &OsStr) -> String {
    value
        .to_string_lossy()
        .trim()
        .trim_matches('"')
        .replace('/', "\\")
        .trim_end_matches('\\')
        .to_ascii_lowercase()
}

#[cfg(any(windows, test))]
fn windows_path_contains(current: &OsStr, directory: &Path) -> bool {
    let expected = normalize_windows_path_entry(directory.as_os_str());
    current
        .to_string_lossy()
        .split(';')
        .map(OsStr::new)
        .any(|entry| normalize_windows_path_entry(entry) == expected)
}

#[cfg(any(windows, test))]
fn append_windows_path(current: &OsStr, directory: &Path) -> Result<Option<OsString>, String> {
    if windows_path_contains(current, directory) {
        return Ok(None);
    }
    let mut updated = current
        .to_string_lossy()
        .trim()
        .trim_end_matches(';')
        .to_owned();
    if !updated.is_empty() {
        updated.push(';');
    }
    updated.push_str(&directory.to_string_lossy());
    if updated.encode_utf16().count() >= 32_767 {
        return Err("The Windows user PATH is too long to add Codex safely.".into());
    }
    Ok(Some(OsString::from(updated)))
}

#[cfg(windows)]
fn ensure_windows_user_path(directory: &Path) -> Result<bool, String> {
    use std::os::windows::ffi::OsStrExt;
    use windows_sys::Win32::UI::WindowsAndMessaging::{
        SendMessageTimeoutW, HWND_BROADCAST, SMTO_ABORTIFHUNG, WM_SETTINGCHANGE,
    };
    use winreg::enums::{HKEY_CURRENT_USER, REG_EXPAND_SZ, REG_SZ};
    use winreg::types::FromRegValue;
    use winreg::{RegKey, RegValue};

    let (environment, _) = RegKey::predef(HKEY_CURRENT_USER)
        .create_subkey("Environment")
        .map_err(|error| format!("Could not open the Windows user environment: {error}"))?;
    let existing_raw = environment.get_raw_value("Path").ok();
    let existing = existing_raw
        .as_ref()
        .map(OsString::from_reg_value)
        .transpose()
        .map_err(|error| format!("Could not read the Windows user PATH: {error}"))?
        .unwrap_or_default();
    let Some(updated) = append_windows_path(&existing, directory)? else {
        return Ok(false);
    };
    let value_type = existing_raw
        .as_ref()
        .map(|value| value.vtype.clone())
        .filter(|value_type| matches!(value_type, REG_SZ | REG_EXPAND_SZ))
        .unwrap_or(REG_EXPAND_SZ);
    let bytes = updated
        .encode_wide()
        .chain(std::iter::once(0))
        .flat_map(u16::to_le_bytes)
        .collect::<Vec<_>>();
    environment
        .set_raw_value(
            "Path",
            &RegValue {
                bytes,
                vtype: value_type,
            },
        )
        .map_err(|error| format!("Could not update the Windows user PATH: {error}"))?;

    let process_path = std::env::var_os("PATH").unwrap_or_default();
    if let Some(process_path) = append_windows_path(&process_path, directory)? {
        std::env::set_var("PATH", process_path);
    }

    let environment_name = OsStr::new("Environment")
        .encode_wide()
        .chain(std::iter::once(0))
        .collect::<Vec<_>>();
    let mut result = 0_usize;
    unsafe {
        SendMessageTimeoutW(
            HWND_BROADCAST,
            WM_SETTINGCHANGE,
            0,
            environment_name.as_ptr() as isize,
            SMTO_ABORTIFHUNG,
            5_000,
            &mut result,
        );
    }
    Ok(true)
}

fn launcher_for(binary: &Path) -> Launcher {
    #[cfg(windows)]
    {
        if binary
            .extension()
            .and_then(|extension| extension.to_str())
            .is_some_and(|extension| {
                extension.eq_ignore_ascii_case("cmd") || extension.eq_ignore_ascii_case("bat")
            })
        {
            return Launcher::WindowsCommandShim;
        }
    }
    let _ = binary;
    Launcher::Native
}

fn launcher_label(launcher: Launcher) -> &'static str {
    match launcher {
        Launcher::Native => "native executable",
        #[cfg(windows)]
        Launcher::WindowsCommandShim => "Windows command shim",
    }
}

fn diagnostic_command() -> String {
    if cfg!(windows) {
        "Get-Command codex,codex-cli -All -ErrorAction SilentlyContinue | Select-Object CommandType,Name,Source; Get-AppxPackage | Where-Object { $_.Name -match 'ChatGPT|Codex' } | Select-Object Name,Version; codex --version; codex login status".into()
    } else {
        "command -v codex; codex --version; codex login status".into()
    }
}

fn diagnostic_shell() -> &'static str {
    if cfg!(windows) {
        "powershell"
    } else {
        "terminal"
    }
}

fn candidate_names_for_extensions(extensions: &[String]) -> Vec<OsString> {
    candidate_names_for_command("codex", extensions)
}

fn candidate_names_for_command(command: &str, extensions: &[String]) -> Vec<OsString> {
    let mut names = Vec::new();
    for extension in extensions {
        let extension = extension.trim();
        if extension.is_empty() {
            continue;
        }
        let extension = if extension.starts_with('.') {
            extension.to_owned()
        } else {
            format!(".{extension}")
        }
        .to_ascii_lowercase();
        let name = OsString::from(format!("{command}{extension}"));
        if !names.contains(&name) {
            names.push(name);
        }
    }
    let plain = OsString::from(command);
    if !names.contains(&plain) {
        names.push(plain);
    }
    names
}

#[cfg(any(windows, test))]
fn looks_like_codex_app_package(name: &str) -> bool {
    let name = name.to_ascii_lowercase();
    name.contains("openai") && (name.contains("chatgpt") || name.contains("codex"))
}

#[cfg(any(windows, test))]
fn looks_like_codex_app_path(path: &Path) -> bool {
    path.file_name()
        .and_then(|name| name.to_str())
        .is_some_and(|name| {
            let name = name.to_ascii_lowercase();
            name == "chatgpt.exe" || name == "codex.exe"
        })
}

#[cfg(windows)]
fn codex_desktop_app_detected() -> bool {
    use winreg::enums::HKEY_CURRENT_USER;
    use winreg::RegKey;

    if let Some(local_app_data) = std::env::var_os("LOCALAPPDATA") {
        let local_app_data = PathBuf::from(local_app_data);
        let packages = local_app_data.join("Packages");
        if fs::read_dir(packages).is_ok_and(|entries| {
            entries.flatten().any(|entry| {
                entry
                    .file_name()
                    .to_str()
                    .is_some_and(looks_like_codex_app_package)
            })
        }) {
            return true;
        }
        let aliases = local_app_data.join("Microsoft").join("WindowsApps");
        if fs::read_dir(aliases).is_ok_and(|entries| {
            entries
                .flatten()
                .any(|entry| looks_like_codex_app_path(&entry.path()))
        }) {
            return true;
        }
        for candidate in [
            local_app_data
                .join("Programs")
                .join("OpenAI")
                .join("ChatGPT")
                .join("ChatGPT.exe"),
            local_app_data
                .join("Programs")
                .join("ChatGPT")
                .join("ChatGPT.exe"),
        ] {
            if candidate.is_file() {
                return true;
            }
        }
    }
    RegKey::predef(HKEY_CURRENT_USER)
        .open_subkey(
            r"Software\Classes\Local Settings\Software\Microsoft\Windows\CurrentVersion\AppModel\Repository\Packages",
        )
        .ok()
        .is_some_and(|packages| {
            packages
                .enum_keys()
                .flatten()
                .any(|name| looks_like_codex_app_package(&name))
        })
}

#[cfg(not(windows))]
fn codex_desktop_app_detected() -> bool {
    false
}

#[cfg(windows)]
fn legacy_cli_detected() -> bool {
    let names = candidate_names_for_command("codex-cli", &windows_command_extensions());
    let mut directories = std::env::var_os("PATH")
        .map(|path| std::env::split_paths(&path).collect::<Vec<_>>())
        .unwrap_or_default();
    if let Some(app_data) = std::env::var_os("APPDATA") {
        directories.push(PathBuf::from(app_data).join("npm"));
    }
    if let Some(local_app_data) = std::env::var_os("LOCALAPPDATA") {
        let local_app_data = PathBuf::from(local_app_data);
        directories.push(local_app_data.join("npm"));
        directories.push(local_app_data.join("bun").join("bin"));
        directories.push(
            local_app_data
                .join("Microsoft")
                .join("WinGet")
                .join("Links"),
        );
    }
    if let Some(bun_install) = std::env::var_os("BUN_INSTALL") {
        directories.push(PathBuf::from(bun_install).join("bin"));
    }
    candidate_paths(&directories, &names)
        .into_iter()
        .any(|candidate| candidate.is_file())
}

#[cfg(not(windows))]
fn legacy_cli_detected() -> bool {
    false
}

#[cfg(windows)]
fn windows_command_extensions() -> Vec<String> {
    let mut extensions = std::env::var_os("PATHEXT")
        .map(|value| {
            value
                .to_string_lossy()
                .split(';')
                .map(str::trim)
                .filter(|extension| !extension.is_empty())
                .map(|extension| extension.to_ascii_lowercase())
                .collect::<Vec<_>>()
        })
        .unwrap_or_default();
    for extension in [".exe", ".cmd", ".bat", ".com"] {
        if !extensions
            .iter()
            .any(|configured| configured.eq_ignore_ascii_case(extension))
        {
            extensions.push(extension.into());
        }
    }
    extensions
}

fn candidate_names() -> Vec<OsString> {
    #[cfg(windows)]
    {
        return candidate_names_for_extensions(&windows_command_extensions());
    }
    #[cfg(not(windows))]
    {
        candidate_names_for_extensions(&[])
    }
}

fn push_unique_path(paths: &mut Vec<PathBuf>, candidate: PathBuf) {
    if !paths.iter().any(|path| path == &candidate) {
        paths.push(candidate);
    }
}

fn candidate_paths(directories: &[PathBuf], names: &[OsString]) -> Vec<PathBuf> {
    let mut candidates = Vec::new();
    for directory in directories {
        for name in names {
            push_unique_path(&mut candidates, directory.join(name));
        }
    }
    candidates
}

fn system_candidates() -> Vec<PathBuf> {
    let names = candidate_names();
    #[allow(unused_mut)]
    let mut directories = std::env::var_os("PATH")
        .map(|path| std::env::split_paths(&path).collect::<Vec<_>>())
        .unwrap_or_default();
    #[cfg(windows)]
    {
        if let Ok(global_dir) = global_binary_dir() {
            directories.insert(0, global_dir);
        }
        if let Some(app_data) = std::env::var_os("APPDATA") {
            directories.push(PathBuf::from(app_data).join("npm"));
        }
        if let Some(local_app_data) = std::env::var_os("LOCALAPPDATA") {
            let local_app_data = PathBuf::from(local_app_data);
            directories.push(local_app_data.join("npm"));
            directories.push(local_app_data.join("bun").join("bin"));
            directories.push(
                local_app_data
                    .join("Microsoft")
                    .join("WinGet")
                    .join("Links"),
            );
        }
        if let Some(bun_install) = std::env::var_os("BUN_INSTALL") {
            directories.push(PathBuf::from(bun_install).join("bin"));
        }
        if let Some(user_profile) = std::env::var_os("USERPROFILE") {
            directories.push(PathBuf::from(user_profile).join(".bun").join("bin"));
        }
    }
    let mut candidates = candidate_paths(&directories, &names);
    #[cfg(target_os = "macos")]
    {
        candidates.extend(candidate_paths(
            &[
                PathBuf::from("/opt/homebrew/bin"),
                PathBuf::from("/usr/local/bin"),
            ],
            &names,
        ));
    }
    if let Some(home) = std::env::var_os(if cfg!(windows) { "USERPROFILE" } else { "HOME" }) {
        candidates.extend(candidate_paths(
            &[PathBuf::from(home).join(".local").join("bin")],
            &names,
        ));
    }
    candidates
}

fn find_installation(app: &AppHandle) -> Result<Option<Installation>, String> {
    let manifest = manifest()?;
    if let Some(binary) = system_candidates()
        .into_iter()
        .find(|candidate| candidate.is_file())
    {
        #[cfg(windows)]
        let source = if normalize_windows_path_entry(binary.as_os_str())
            == normalize_windows_path_entry(global_binary()?.as_os_str())
        {
            Source::Global
        } else {
            Source::System
        };
        #[cfg(not(windows))]
        let source = Source::System;
        return Ok(Some(Installation {
            launcher: launcher_for(&binary),
            binary,
            source,
        }));
    }
    let managed = managed_binary(app, &manifest.version)?;
    if managed.is_file() {
        return Ok(Some(Installation {
            binary: managed,
            source: Source::Managed,
            launcher: Launcher::Native,
        }));
    }
    Ok(None)
}

fn configure_command(app: &AppHandle, installation: &Installation) -> Result<Command, String> {
    let mut command = Command::new(&installation.binary);
    if installation.source == Source::Managed {
        let home = managed_home(app)?;
        fs::create_dir_all(&home).map_err(|error| {
            format!("Could not create the private Codex login directory: {error}")
        })?;
        restrict_directory(&home)?;
        command.env("CODEX_HOME", home);
        command.env_remove("OPENAI_API_KEY");
    }
    command
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped());
    #[cfg(windows)]
    command.creation_flags(windows_sys::Win32::System::Threading::CREATE_NO_WINDOW);
    Ok(command)
}

async fn command_output(
    app: &AppHandle,
    installation: &Installation,
    arguments: &[&str],
    timeout: Duration,
) -> Result<std::process::Output, String> {
    let mut command = configure_command(app, installation)?;
    command.args(arguments);
    command.kill_on_drop(true);
    tokio::time::timeout(timeout, command.output())
        .await
        .map_err(|_| "Codex did not finish before the safety timeout.".to_string())?
        .map_err(|error| format!("Could not run Codex: {error}"))
}

fn clean_output(output: &[u8]) -> String {
    String::from_utf8_lossy(output)
        .lines()
        .next()
        .unwrap_or_default()
        .trim()
        .chars()
        .take(160)
        .collect()
}

pub async fn status(app: &AppHandle) -> Result<CodexStatus, String> {
    let managed_version = manifest()?.version;
    let desktop_app_detected = codex_desktop_app_detected();
    let legacy_cli_detected = legacy_cli_detected();
    let Some(installation) = find_installation(app)? else {
        let (state, message) = if desktop_app_detected {
            (
                "app_only",
                "The Codex app is installed, but Windows does not expose its internal agent as the codex command. Add the verified official CLI; OneDay will use the same %USERPROFILE%\\.codex sign-in when available.",
            )
        } else if legacy_cli_detected {
            (
                "legacy_cli",
                "A legacy codex-cli command was found, but the current official codex command is missing. Install the verified CLI to repair the setup without removing the older command.",
            )
        } else {
            (
                "missing",
                if cfg!(windows) {
                    "Neither the Codex app nor the official CLI was found. OneDay can install the verified CLI globally and add it to your user PATH."
                } else {
                    "Codex is optional and has not been installed for OneDay."
                },
            )
        };
        return Ok(CodexStatus {
            available: false,
            state: state.into(),
            source: "missing".into(),
            version: None,
            authenticated: false,
            desktop_app_detected,
            legacy_cli_detected,
            managed_version,
            message: message.into(),
            launcher: None,
            diagnostic_shell: diagnostic_shell().into(),
            diagnostic_command: diagnostic_command(),
            install_scope: install_scope().into(),
        });
    };
    let version_output = command_output(app, &installation, &["--version"], COMMAND_TIMEOUT).await;
    let version = version_output
        .as_ref()
        .ok()
        .filter(|output| output.status.success())
        .map(|output| clean_output(&output.stdout))
        .filter(|value| !value.is_empty());
    if version.is_none() {
        return Ok(CodexStatus {
            available: false,
            state: "unusable".into(),
            source: "missing".into(),
            version: None,
            authenticated: false,
            desktop_app_detected,
            legacy_cli_detected,
            managed_version,
            message: "The detected Codex executable is not usable. Install the verified component to repair this setup.".into(),
            launcher: Some(launcher_label(installation.launcher).into()),
            diagnostic_shell: diagnostic_shell().into(),
            diagnostic_command: diagnostic_command(),
            install_scope: install_scope().into(),
        });
    }
    #[cfg(windows)]
    if installation.source == Source::Global {
        ensure_windows_user_path(&global_binary_dir()?)?;
    }
    let login_output =
        command_output(app, &installation, &["login", "status"], COMMAND_TIMEOUT).await;
    let authenticated = login_output
        .as_ref()
        .map(|output| output.status.success())
        .unwrap_or(false);
    let source = match installation.source {
        Source::Global => "global",
        Source::Managed => "managed",
        Source::System => "system",
    };
    let location = match installation.source {
        Source::Global => "globally for this Windows user",
        Source::Managed => "in OneDay's private component directory",
        Source::System => "on this device",
    };
    Ok(CodexStatus {
        available: true,
        state: if authenticated { "ready" } else { "signed_out" }.into(),
        source: source.into(),
        version,
        authenticated,
        desktop_app_detected,
        legacy_cli_detected,
        managed_version,
        message: if authenticated {
            format!(
                "Codex is installed {location} through a {} and signed in. Choose it in OneDay Setup to use your subscription.",
                launcher_label(installation.launcher),
            )
        } else {
            format!(
                "Codex is installed {location} through a {}. Sign in before selecting it in OneDay Setup.",
                launcher_label(installation.launcher),
            )
        },
        launcher: Some(launcher_label(installation.launcher).into()),
        diagnostic_shell: diagnostic_shell().into(),
        diagnostic_command: diagnostic_command(),
        install_scope: install_scope().into(),
    })
}

#[tauri::command]
pub async fn codex_status(app: AppHandle) -> Result<CodexStatus, String> {
    status(&app).await
}

#[tauri::command]
pub async fn install_codex_component(app: AppHandle) -> Result<CodexStatus, String> {
    let manifest = manifest()?;
    let current = status(&app).await?;
    if current.available {
        let installation = find_installation(&app)?
            .ok_or_else(|| "Codex disappeared while checking its installation.".to_string())?;
        #[cfg(windows)]
        {
            if installation.source == Source::Global {
                ensure_windows_user_path(&global_binary_dir()?)?;
                return status(&app).await;
            }
            if installation.source == Source::System {
                return Ok(current);
            }
            // Older OneDay previews used a private component. Continue below
            // to migrate it to the normal per-user Windows CLI location.
        }
        #[cfg(not(windows))]
        {
            let _ = installation;
            return Ok(current);
        }
    }
    let triple = target_triple().ok_or_else(|| {
        "This operating system or CPU does not have a managed Codex component yet.".to_string()
    })?;
    let target = manifest
        .targets
        .get(triple)
        .cloned()
        .ok_or_else(|| format!("The Codex component manifest has no entry for {triple}."))?;
    validate_target(&manifest, &target)?;
    let root = component_root(&app)?;
    fs::create_dir_all(&root)
        .map_err(|error| format!("Could not create the desktop component directory: {error}"))?;
    restrict_directory(&root)?;
    let mut archive = NamedTempFile::new_in(&root)
        .map_err(|error| format!("Could not stage the Codex download: {error}"))?;
    restrict_file(archive.path())?;
    download_verified(&target, archive.as_file_mut()).await?;
    archive
        .as_file_mut()
        .sync_all()
        .map_err(|error| format!("Could not finish the Codex download: {error}"))?;
    #[cfg(windows)]
    let version_dir = global_binary_dir()?;
    #[cfg(not(windows))]
    let version_dir = root.join(&manifest.version);
    fs::create_dir_all(&version_dir)
        .map_err(|error| format!("Could not create the Codex version directory: {error}"))?;
    restrict_directory(&version_dir)?;
    #[cfg(windows)]
    let final_binary = global_binary()?;
    #[cfg(not(windows))]
    let final_binary = managed_binary(&app, &manifest.version)?;
    let staged_binary = version_dir.join(if cfg!(windows) {
        "codex.new.exe"
    } else {
        "codex.new"
    });
    extract_binary(archive.path(), &target, &staged_binary)?;
    restrict_executable(&staged_binary)?;
    if final_binary.exists() {
        fs::remove_file(&final_binary)
            .map_err(|error| format!("Could not replace the unusable Codex component: {error}"))?;
    }
    fs::rename(&staged_binary, &final_binary)
        .map_err(|error| format!("Could not activate the Codex component: {error}"))?;
    #[cfg(windows)]
    ensure_windows_user_path(&version_dir)?;
    status(&app).await
}

#[tauri::command]
pub async fn login_codex(app: AppHandle) -> Result<CodexStatus, String> {
    let installation =
        find_installation(&app)?.ok_or_else(|| "Install Codex before signing in.".to_string())?;
    let output = command_output(&app, &installation, &["login"], LOGIN_TIMEOUT).await?;
    if !output.status.success() {
        let detail = clean_output(&output.stderr);
        return Err(if detail.is_empty() {
            "Codex sign-in was cancelled or did not complete.".into()
        } else {
            format!("Codex sign-in did not complete: {detail}")
        });
    }
    status(&app).await
}

pub fn runtime(app: &AppHandle) -> Result<Option<CodexRuntime>, String> {
    let Some(installation) = find_installation(app)? else {
        return Ok(None);
    };
    let binary_dir = installation
        .binary
        .parent()
        .ok_or_else(|| "The Codex executable has no parent directory.".to_string())?
        .to_path_buf();
    Ok(Some(CodexRuntime {
        binary_dir,
        home: (installation.source == Source::Managed)
            .then(|| managed_home(app))
            .transpose()?,
        pathext: runtime_pathext(&installation.binary),
    }))
}

#[cfg(windows)]
fn runtime_pathext(binary: &Path) -> Option<OsString> {
    let extension = binary
        .extension()
        .and_then(|extension| extension.to_str())
        .map(|extension| format!(".{}", extension.to_ascii_uppercase()))?;
    let mut values = windows_command_extensions();
    if !values
        .iter()
        .any(|configured| configured.eq_ignore_ascii_case(&extension))
    {
        values.push(extension);
    }
    Some(OsString::from(values.join(";")))
}

#[cfg(not(windows))]
fn runtime_pathext(_binary: &Path) -> Option<OsString> {
    None
}

pub fn prepend_paths(directories: &[&Path]) -> Result<OsString, String> {
    let mut paths = directories
        .iter()
        .map(|path| path.to_path_buf())
        .collect::<Vec<_>>();
    if let Some(current) = std::env::var_os("PATH") {
        paths.extend(std::env::split_paths(&current));
    }
    std::env::join_paths(paths)
        .map_err(|error| format!("Could not prepare the Codex PATH: {error}"))
}

fn validate_target(manifest: &ComponentManifest, target: &ComponentTarget) -> Result<(), String> {
    let expected_prefix = format!(
        "https://github.com/openai/codex/releases/download/{}/",
        manifest.release_tag
    );
    if !target.url.starts_with(&expected_prefix)
        || target.sha256.len() != 64
        || !target.sha256.bytes().all(|byte| byte.is_ascii_hexdigit())
        || target.size == 0
        || !matches!(target.archive.as_str(), "zip" | "tar.gz")
        || Path::new(&target.entry)
            .file_name()
            .and_then(|name| name.to_str())
            != Some(target.entry.as_str())
    {
        return Err("The bundled Codex download entry failed validation.".into());
    }
    Ok(())
}

async fn download_verified(target: &ComponentTarget, output: &mut fs::File) -> Result<(), String> {
    let client = reqwest::Client::builder()
        .connect_timeout(Duration::from_secs(15))
        .timeout(Duration::from_secs(15 * 60))
        .redirect(Policy::limited(5))
        .user_agent(concat!("OneDay-Desktop/", env!("CARGO_PKG_VERSION")))
        .build()
        .map_err(|error| format!("Could not prepare the Codex download: {error}"))?;
    let mut response = client
        .get(&target.url)
        .send()
        .await
        .map_err(|error| format!("Could not download Codex: {error}"))?
        .error_for_status()
        .map_err(|error| format!("The Codex download failed: {error}"))?;
    if response
        .content_length()
        .is_some_and(|size| size != target.size)
    {
        return Err("The Codex download size does not match the pinned release manifest.".into());
    }
    let mut digest = Sha256::new();
    let mut received = 0_u64;
    while let Some(chunk) = response
        .chunk()
        .await
        .map_err(|error| format!("The Codex download was interrupted: {error}"))?
    {
        received = received.saturating_add(chunk.len() as u64);
        if received > target.size.saturating_add(MAX_DOWNLOAD_OVERHEAD) {
            return Err("The Codex download exceeded the pinned size limit.".into());
        }
        digest.update(&chunk);
        output
            .write_all(&chunk)
            .map_err(|error| format!("Could not store the Codex download: {error}"))?;
    }
    if received != target.size || format!("{:x}", digest.finalize()) != target.sha256 {
        return Err("The Codex download failed its SHA-256 integrity check.".into());
    }
    Ok(())
}

fn extract_binary(archive: &Path, target: &ComponentTarget, output: &Path) -> Result<(), String> {
    let mut destination = fs::File::create(output)
        .map_err(|error| format!("Could not stage the Codex executable: {error}"))?;
    match target.archive.as_str() {
        "zip" => {
            let file = fs::File::open(archive)
                .map_err(|error| format!("Could not read the Codex archive: {error}"))?;
            let mut zip = zip::ZipArchive::new(file)
                .map_err(|error| format!("The Codex ZIP archive is invalid: {error}"))?;
            let mut entry = zip.by_name(&target.entry).map_err(|_| {
                "The Codex ZIP archive is missing its expected executable.".to_string()
            })?;
            if entry.size() > MAX_BINARY_SIZE {
                return Err("The Codex executable exceeds its extraction limit.".into());
            }
            io::copy(&mut entry, &mut destination)
                .map_err(|error| format!("Could not extract the Codex executable: {error}"))?;
        }
        "tar.gz" => {
            let file = fs::File::open(archive)
                .map_err(|error| format!("Could not read the Codex archive: {error}"))?;
            let mut tar = tar::Archive::new(GzDecoder::new(file));
            let mut found = false;
            for entry in tar
                .entries()
                .map_err(|error| format!("The Codex tar archive is invalid: {error}"))?
            {
                let mut entry =
                    entry.map_err(|error| format!("The Codex tar archive is invalid: {error}"))?;
                let path = entry
                    .path()
                    .map_err(|error| format!("The Codex archive entry is invalid: {error}"))?;
                if path.file_name().and_then(|name| name.to_str()) == Some(target.entry.as_str()) {
                    if entry.size() > MAX_BINARY_SIZE {
                        return Err("The Codex executable exceeds its extraction limit.".into());
                    }
                    io::copy(&mut entry, &mut destination).map_err(|error| {
                        format!("Could not extract the Codex executable: {error}")
                    })?;
                    found = true;
                    break;
                }
            }
            if !found {
                return Err("The Codex tar archive is missing its expected executable.".into());
            }
        }
        _ => return Err("The Codex archive format is not supported.".into()),
    }
    destination
        .sync_all()
        .map_err(|error| format!("Could not finish the Codex executable: {error}"))
}

#[cfg(unix)]
fn restrict_executable(path: &Path) -> Result<(), String> {
    use std::os::unix::fs::PermissionsExt;
    fs::set_permissions(path, fs::Permissions::from_mode(0o700))
        .map_err(|error| format!("Could not secure the Codex executable: {error}"))
}

#[cfg(not(unix))]
fn restrict_executable(_path: &Path) -> Result<(), String> {
    Ok(())
}

#[cfg(unix)]
fn restrict_file(path: &Path) -> Result<(), String> {
    use std::os::unix::fs::PermissionsExt;
    fs::set_permissions(path, fs::Permissions::from_mode(0o600))
        .map_err(|error| format!("Could not secure the Codex download: {error}"))
}

#[cfg(not(unix))]
fn restrict_file(_path: &Path) -> Result<(), String> {
    Ok(())
}

#[cfg(unix)]
fn restrict_directory(path: &Path) -> Result<(), String> {
    use std::os::unix::fs::PermissionsExt;
    fs::set_permissions(path, fs::Permissions::from_mode(0o700))
        .map_err(|error| format!("Could not secure the Codex directory: {error}"))
}

#[cfg(not(unix))]
fn restrict_directory(_path: &Path) -> Result<(), String> {
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn manifest_uses_pinned_official_assets_for_every_desktop_target() {
        let manifest = manifest().expect("valid manifest");
        for triple in [
            "x86_64-pc-windows-msvc",
            "aarch64-apple-darwin",
            "x86_64-apple-darwin",
            "x86_64-unknown-linux-gnu",
        ] {
            validate_target(&manifest, manifest.targets.get(triple).expect("target"))
                .expect("valid target");
        }
    }

    #[test]
    fn rejects_an_untrusted_or_unpinned_download() {
        let manifest = manifest().expect("valid manifest");
        let mut target = manifest.targets.values().next().expect("target").clone();
        target.url = "https://example.com/codex.zip".into();
        assert!(validate_target(&manifest, &target).is_err());
        target.url = format!(
            "https://github.com/openai/codex/releases/download/{}/codex.zip",
            manifest.release_tag
        );
        target.sha256 = "not-a-digest".into();
        assert!(validate_target(&manifest, &target).is_err());
    }

    #[test]
    fn prepends_component_without_dropping_the_existing_path() {
        let joined = prepend_paths(&[Path::new("/component/bin")]).expect("joined PATH");
        let paths = std::env::split_paths(&joined).collect::<Vec<_>>();
        assert_eq!(paths.first(), Some(&PathBuf::from("/component/bin")));
    }

    #[test]
    fn windows_candidate_names_cover_executables_and_common_command_shims() {
        let names = candidate_names_for_extensions(&[
            ".EXE".to_string(),
            ".cmd".to_string(),
            "bat".to_string(),
            ".CMD".to_string(),
        ]);
        let names = names
            .iter()
            .map(|name| name.to_string_lossy().to_ascii_lowercase())
            .collect::<Vec<_>>();
        assert!(names.contains(&"codex.exe".to_string()));
        assert!(names.contains(&"codex.cmd".to_string()));
        assert!(names.contains(&"codex.bat".to_string()));
        assert!(names.contains(&"codex".to_string()));
        assert_eq!(names.iter().filter(|name| *name == "codex.cmd").count(), 1);
    }

    #[test]
    fn legacy_cli_candidates_keep_the_legacy_name_separate() {
        let names =
            candidate_names_for_command("codex-cli", &[".exe".to_string(), ".cmd".to_string()])
                .into_iter()
                .map(|name| name.to_string_lossy().to_ascii_lowercase())
                .collect::<Vec<_>>();
        assert!(names.contains(&"codex-cli.exe".to_string()));
        assert!(names.contains(&"codex-cli.cmd".to_string()));
        assert!(!names.contains(&"codex.exe".to_string()));
    }

    #[test]
    fn recognizes_only_openai_chatgpt_or_codex_app_packages() {
        assert!(looks_like_codex_app_package(
            "OpenAI.ChatGPT-Desktop_1.2.3_x64__publisher"
        ));
        assert!(looks_like_codex_app_package(
            "OpenAI.Codex_1.2.3_x64__publisher"
        ));
        assert!(!looks_like_codex_app_package(
            "Example.ChatGPT-Notes_1.0_x64__publisher"
        ));
        assert!(!looks_like_codex_app_package("OpenAI.Images_1.0"));
        assert!(looks_like_codex_app_path(Path::new("ChatGPT.exe")));
        assert!(!looks_like_codex_app_path(Path::new("chatgpt-helper.exe")));
    }

    #[test]
    fn windows_path_registration_is_case_insensitive_and_idempotent() {
        let directory = Path::new(r"C:\Users\Ada\AppData\Local\Programs\OpenAI\Codex\bin");
        let existing = OsStr::new(
            r"C:\Windows\System32;c:/users/ada/appdata/local/programs/openai/codex/bin\",
        );
        assert!(windows_path_contains(existing, directory));
        assert_eq!(
            append_windows_path(existing, directory).expect("valid PATH"),
            None
        );
    }

    #[test]
    fn windows_path_registration_preserves_existing_entries() {
        let directory = Path::new(r"C:\Users\Ada\AppData\Local\Programs\OpenAI\Codex\bin");
        let updated = append_windows_path(OsStr::new(r"C:\Windows;C:\Tools"), directory)
            .expect("valid PATH")
            .expect("new entry");
        assert_eq!(
            updated,
            OsString::from(
                r"C:\Windows;C:\Tools;C:\Users\Ada\AppData\Local\Programs\OpenAI\Codex\bin"
            )
        );
    }

    #[test]
    fn diagnostic_command_is_static_and_never_contains_a_user_path() {
        let command = diagnostic_command();
        assert!(command.contains("codex --version"));
        assert!(!command.contains("APPDATA"));
        assert!(!command.contains("CODEX_HOME"));
        if cfg!(windows) {
            assert_eq!(diagnostic_shell(), "powershell");
            assert!(command.contains("ChatGPT|Codex"));
            assert!(command.contains("Get-Command"));
        } else {
            assert_eq!(diagnostic_shell(), "terminal");
            assert!(command.contains("command -v codex"));
            assert!(!command.contains("Get-AppxPackage"));
        }
    }
}
