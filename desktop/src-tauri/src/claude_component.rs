use serde::Serialize;
use std::path::PathBuf;
use std::process::Stdio;
use std::time::Duration;
use tauri::AppHandle;
use tokio::process::Command;

const COMMAND_TIMEOUT: Duration = Duration::from_secs(20);
const LOGIN_TIMEOUT: Duration = Duration::from_secs(10 * 60);
const INSTALL_TIMEOUT: Duration = Duration::from_secs(15 * 60);
const INSTALL_GUIDE: &str = "https://docs.anthropic.com/en/docs/claude-code/setup";

#[derive(Clone, Debug)]
pub struct ClaudeRuntime {
    pub binary_dir: PathBuf,
}

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ClaudeStatus {
    available: bool,
    version: Option<String>,
    authenticated: bool,
    install_supported: bool,
    install_method: Option<String>,
    message: String,
}

#[cfg(any(windows, target_os = "macos"))]
fn find_command(name: &str) -> Option<PathBuf> {
    std::env::var_os("PATH").and_then(|path| {
        std::env::split_paths(&path)
            .map(|directory| directory.join(name))
            .find(|candidate| candidate.is_file())
    })
}

fn installer() -> Option<(PathBuf, &'static [&'static str], &'static str)> {
    #[cfg(windows)]
    if let Some(winget) = find_command("winget.exe") {
        return Some((
            winget,
            &[
                "install",
                "--id",
                "Anthropic.ClaudeCode",
                "--exact",
                "--silent",
                "--accept-package-agreements",
                "--accept-source-agreements",
            ],
            "WinGet",
        ));
    }
    #[cfg(target_os = "macos")]
    if let Some(brew) = find_command("brew") {
        return Some((brew, &["install", "--cask", "claude-code"], "Homebrew"));
    }
    None
}

fn candidates() -> Vec<PathBuf> {
    let name = if cfg!(windows) {
        "claude.exe"
    } else {
        "claude"
    };
    let mut values: Vec<PathBuf> = std::env::var_os("PATH")
        .map(|path| {
            std::env::split_paths(&path)
                .map(|directory| directory.join(name))
                .collect()
        })
        .unwrap_or_default();
    #[cfg(target_os = "macos")]
    {
        values.push(PathBuf::from("/opt/homebrew/bin/claude"));
        values.push(PathBuf::from("/usr/local/bin/claude"));
    }
    #[cfg(windows)]
    if let Some(local) = std::env::var_os("LOCALAPPDATA") {
        values.push(
            PathBuf::from(local)
                .join("Microsoft")
                .join("WinGet")
                .join("Links")
                .join(name),
        );
    }
    if let Some(home) = std::env::var_os(if cfg!(windows) { "USERPROFILE" } else { "HOME" }) {
        let home = PathBuf::from(home);
        values.push(home.join(".local").join("bin").join(name));
        values.push(home.join(".claude").join("local").join(name));
    }
    values
}

fn find_binary() -> Option<PathBuf> {
    candidates()
        .into_iter()
        .find(|candidate| candidate.is_file())
}

fn command(binary: &PathBuf) -> Command {
    let mut command = Command::new(binary);
    command
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped());
    #[cfg(windows)]
    command.creation_flags(windows_sys::Win32::System::Threading::CREATE_NO_WINDOW);
    command
}

async fn output(
    binary: &PathBuf,
    arguments: &[&str],
    timeout: Duration,
) -> Result<std::process::Output, String> {
    let mut command = command(binary);
    command.args(arguments);
    command.kill_on_drop(true);
    tokio::time::timeout(timeout, command.output())
        .await
        .map_err(|_| "Claude Code did not finish before the safety timeout.".to_string())?
        .map_err(|error| format!("Could not run Claude Code: {error}"))
}

fn first_line(bytes: &[u8]) -> Option<String> {
    let value = String::from_utf8_lossy(bytes)
        .lines()
        .next()
        .unwrap_or_default()
        .trim()
        .chars()
        .take(160)
        .collect::<String>();
    (!value.is_empty()).then_some(value)
}

pub async fn status() -> Result<ClaudeStatus, String> {
    let install_method = installer().map(|(_, _, method)| method.to_string());
    let Some(binary) = find_binary() else {
        return Ok(ClaudeStatus {
            available: false,
            version: None,
            authenticated: false,
            install_supported: install_method.is_some(),
            install_method: install_method.clone(),
            message: match install_method {
                Some(method) => format!(
                    "Claude Code was not found. OneDay can install the official package with {method}."
                ),
                None => "Claude Code was not found. Use Anthropic's official installation guide, then refresh this connection.".into(),
            },
        });
    };
    let version_output = output(&binary, &["--version"], COMMAND_TIMEOUT).await?;
    let version = version_output
        .status
        .success()
        .then(|| first_line(&version_output.stdout))
        .flatten();
    if version.is_none() {
        return Ok(ClaudeStatus {
            available: false,
            version: None,
            authenticated: false,
            install_supported: install_method.is_some(),
            install_method,
            message: "The detected Claude Code executable is not usable.".into(),
        });
    }
    let authenticated = output(&binary, &["auth", "status"], COMMAND_TIMEOUT)
        .await
        .is_ok_and(|result| result.status.success());
    Ok(ClaudeStatus {
        available: true,
        version,
        authenticated,
        install_supported: false,
        install_method: None,
        message: if authenticated {
            "Claude Code is installed and signed in with a local subscription session.".into()
        } else {
            "Claude Code is installed. Sign in before enabling it in OneDay Setup.".into()
        },
    })
}

#[tauri::command]
pub async fn claude_status(_app: AppHandle) -> Result<ClaudeStatus, String> {
    status().await
}

#[tauri::command]
pub async fn login_claude(_app: AppHandle) -> Result<ClaudeStatus, String> {
    let binary = find_binary().ok_or_else(|| {
        "Install Claude Code from claude.ai or WinGet before signing in.".to_string()
    })?;
    let login = output(&binary, &["auth", "login"], LOGIN_TIMEOUT).await?;
    if !login.status.success() {
        return Err("Claude Code sign-in was cancelled or did not complete.".into());
    }
    status().await
}

#[tauri::command]
pub async fn install_claude() -> Result<ClaudeStatus, String> {
    let (binary, arguments, method) = installer().ok_or_else(|| {
        "No supported system package manager was found. Open the official Claude Code installation guide instead."
            .to_string()
    })?;
    let result = output(&binary, arguments, INSTALL_TIMEOUT).await?;
    if !result.status.success() {
        return Err(format!(
            "{method} could not install Claude Code. No provider setting was changed."
        ));
    }
    status().await
}

#[tauri::command]
pub fn open_claude_install_guide() -> Result<(), String> {
    #[cfg(windows)]
    let (program, arguments) = (
        PathBuf::from("rundll32.exe"),
        vec!["url.dll,FileProtocolHandler", INSTALL_GUIDE],
    );
    #[cfg(target_os = "macos")]
    let (program, arguments) = (PathBuf::from("open"), vec![INSTALL_GUIDE]);
    #[cfg(all(unix, not(target_os = "macos")))]
    let (program, arguments) = (PathBuf::from("xdg-open"), vec![INSTALL_GUIDE]);

    let mut process = command(&program);
    process
        .args(arguments)
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .spawn()
        .map(|_| ())
        .map_err(|error| format!("Could not open the Claude Code installation guide: {error}"))
}

pub fn runtime() -> Option<ClaudeRuntime> {
    find_binary().and_then(|binary| {
        binary.parent().map(|parent| ClaudeRuntime {
            binary_dir: parent.to_path_buf(),
        })
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn candidates_never_include_a_shell_wrapper_name() {
        assert!(candidates().iter().all(|candidate| {
            let name = candidate
                .file_name()
                .and_then(|value| value.to_str())
                .unwrap_or_default();
            name == "claude" || name == "claude.exe"
        }));
    }
}
