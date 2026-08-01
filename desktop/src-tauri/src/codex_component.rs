use flate2::read::GzDecoder;
use reqwest::redirect::Policy;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::HashMap;
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
}

#[derive(Clone, Debug)]
struct Installation {
    binary: PathBuf,
    source: Source,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum Source {
    Managed,
    System,
}

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct CodexStatus {
    available: bool,
    source: String,
    version: Option<String>,
    authenticated: bool,
    managed_version: String,
    message: String,
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

fn system_candidates() -> Vec<PathBuf> {
    let name = if cfg!(windows) { "codex.exe" } else { "codex" };
    let mut candidates = std::env::var_os("PATH")
        .map(|path| {
            std::env::split_paths(&path)
                .map(|directory| directory.join(name))
                .collect::<Vec<_>>()
        })
        .unwrap_or_default();
    #[cfg(target_os = "macos")]
    {
        candidates.push(PathBuf::from("/opt/homebrew/bin/codex"));
        candidates.push(PathBuf::from("/usr/local/bin/codex"));
    }
    if let Some(home) = std::env::var_os(if cfg!(windows) { "USERPROFILE" } else { "HOME" }) {
        candidates.push(PathBuf::from(home).join(".local").join("bin").join(name));
    }
    candidates
}

fn find_installation(app: &AppHandle) -> Result<Option<Installation>, String> {
    let manifest = manifest()?;
    let managed = managed_binary(app, &manifest.version)?;
    if managed.is_file() {
        return Ok(Some(Installation {
            binary: managed,
            source: Source::Managed,
        }));
    }
    Ok(system_candidates()
        .into_iter()
        .find(|candidate| candidate.is_file())
        .map(|binary| Installation {
            binary,
            source: Source::System,
        }))
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
    let Some(installation) = find_installation(app)? else {
        return Ok(CodexStatus {
            available: false,
            source: "missing".into(),
            version: None,
            authenticated: false,
            managed_version,
            message: "Codex is optional and has not been installed for OneDay.".into(),
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
            source: "missing".into(),
            version: None,
            authenticated: false,
            managed_version,
            message: "The detected Codex executable is not usable. Install the verified component to repair this setup.".into(),
        });
    }
    let login_output =
        command_output(app, &installation, &["login", "status"], COMMAND_TIMEOUT).await;
    let authenticated = login_output
        .as_ref()
        .map(|output| output.status.success())
        .unwrap_or(false);
    let source = match installation.source {
        Source::Managed => "managed",
        Source::System => "system",
    };
    Ok(CodexStatus {
        available: true,
        source: source.into(),
        version,
        authenticated,
        managed_version,
        message: if authenticated {
            "Codex is installed and signed in. Choose it in OneDay Setup to use your subscription."
                .into()
        } else {
            "Codex is installed. Sign in before selecting it in OneDay Setup.".into()
        },
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
        return Ok(current);
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
    let version_dir = root.join(&manifest.version);
    fs::create_dir_all(&version_dir)
        .map_err(|error| format!("Could not create the Codex version directory: {error}"))?;
    restrict_directory(&version_dir)?;
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
    }))
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
}
