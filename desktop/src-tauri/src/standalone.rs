use crate::codex_component;
use crate::config;
use crate::containment::{self, ProcessContainment};
use crate::secret::LaunchSecret;
use std::fs;
use std::io::{Read, Write};
use std::net::TcpListener;
use std::path::{Path, PathBuf};
use std::process::{Child, Command, Stdio};
use std::sync::{Arc, Mutex};
use std::thread;
use std::time::{Duration, Instant};
use tauri::{AppHandle, Manager};
use url::Url;

pub const MAX_START_ATTEMPTS: usize = 2;
const SHUTDOWN_WAIT: Duration = Duration::from_secs(5);
const MAX_LOG_BYTES: u64 = 256 * 1024;

pub struct LocalProcess {
    child: Child,
    containment: ProcessContainment,
    logs: Vec<thread::JoinHandle<()>>,
    pub endpoint: Url,
    _secret: LaunchSecret,
}

#[derive(Debug)]
struct LaunchPlan {
    endpoint: Url,
    profile_dir: PathBuf,
    gateway_bin: PathBuf,
    engine_bin: PathBuf,
    static_dir: PathBuf,
    codex: Option<codex_component::CodexRuntime>,
}

impl LaunchPlan {
    fn create(
        profile_dir: PathBuf,
        resource_dir: PathBuf,
        codex: Option<codex_component::CodexRuntime>,
    ) -> Result<Self, String> {
        let listener = TcpListener::bind("127.0.0.1:0")
            .map_err(|error| format!("Could not reserve a loopback port: {error}"))?;
        let address = listener
            .local_addr()
            .map_err(|error| format!("Could not inspect the loopback port: {error}"))?;
        // The gateway only accepts an address rather than a pre-bound socket.
        // Releasing this immediately keeps the endpoint dynamically assigned;
        // startup retries if another local process wins the short race.
        drop(listener);
        let endpoint = Url::parse(&format!("http://127.0.0.1:{}/", address.port()))
            .map_err(|error| format!("Could not create the local endpoint: {error}"))?;
        let (gateway_bin, engine_bin) = bundled_bins(&resource_dir);
        let static_dir = resource_dir.join("gateway/web/dist");
        let plan = Self {
            endpoint,
            profile_dir,
            gateway_bin,
            engine_bin,
            static_dir,
            codex,
        };
        plan.validate()?;
        Ok(plan)
    }

    fn validate(&self) -> Result<(), String> {
        if !self.gateway_bin.is_file() || !self.engine_bin.is_file() {
            return Err(
                "This standalone build is missing its version-matched OneDay sidecars.".into(),
            );
        }
        if !self.static_dir.join("index.html").is_file() {
            return Err(
                "This standalone build is missing the bundled OneDay web interface.".into(),
            );
        }
        Ok(())
    }

    fn command(&self, secret: &LaunchSecret) -> Result<Command, String> {
        let config_path = config::create_initial_standalone_config(&self.profile_dir)?;
        let mut command = Command::new(&self.gateway_bin);
        command
            .args(self.arguments(&config_path)?)
            // The launch token is intentionally an environment value, never a
            // command-line argument or persisted setting.
            .env("ONEDAY_GATEWAY_AUTH_TOKEN", secret.environment_value())
            .env("ONEDAY_GATEWAY_URL", self.endpoint.as_str())
            .stdin(Stdio::null())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .current_dir(&self.profile_dir);
        if let Some(codex) = &self.codex {
            command.env("PATH", codex_component::prepend_path(&codex.binary_dir)?);
            if let Some(home) = &codex.home {
                command.env("CODEX_HOME", home);
                command.env_remove("OPENAI_API_KEY");
            }
        }
        containment::configure(&mut command);
        Ok(command)
    }

    fn arguments(&self, config_path: &Path) -> Result<Vec<String>, String> {
        let database = self.profile_dir.join("data").join("oneday.db");
        let address = self
            .endpoint
            .socket_addrs(|| None)
            .map_err(|error| format!("Could not resolve the loopback endpoint: {error}"))?
            .into_iter()
            .next()
            .ok_or_else(|| "Could not resolve the loopback endpoint.".to_string())?;
        Ok(vec![
            "--addr".into(),
            address.to_string(),
            "--oneday-root".into(),
            self.profile_dir.display().to_string(),
            "--config-path".into(),
            config_path.display().to_string(),
            "--db-path".into(),
            database.display().to_string(),
            "--oneday-bin".into(),
            self.engine_bin.display().to_string(),
            "--static-dir".into(),
            self.static_dir.display().to_string(),
        ])
    }
}

pub fn start(app: &AppHandle, profile_id: &str) -> Result<LocalProcess, String> {
    let profile_dir = config::standalone_profile_dir(app, profile_id)?;
    let resource_dir = app
        .path()
        .resource_dir()
        .map_err(|error| format!("Could not locate bundled standalone components: {error}"))?;
    let secret = LaunchSecret::generate()?;
    let plan = LaunchPlan::create(profile_dir, resource_dir, codex_component::runtime(app)?)?;
    let log_path = prepare_log_path(&plan.profile_dir)?;
    let mut child = plan
        .command(&secret)?
        .spawn()
        .map_err(|error| format!("Could not start the local OneDay gateway: {error}"))?;
    let containment = match ProcessContainment::attach(&child) {
        Ok(containment) => containment,
        Err(error) => {
            let _ = child.kill();
            let _ = child.wait();
            return Err(error);
        }
    };
    let logs = capture_logs(&mut child, log_path, secret.environment_value());
    Ok(LocalProcess {
        child,
        containment,
        logs,
        endpoint: plan.endpoint,
        _secret: secret,
    })
}

pub fn stop(mut process: LocalProcess) -> Result<(), String> {
    process.containment.request_graceful_stop()?;
    let deadline = Instant::now() + SHUTDOWN_WAIT;
    while Instant::now() < deadline {
        if process
            .child
            .try_wait()
            .map_err(|error| format!("Could not inspect local gateway shutdown: {error}"))?
            .is_some()
        {
            // The gateway can exit before a descendant it started. The
            // process group/job remains ours until this value is dropped, so
            // clear any stragglers before returning a successful shutdown.
            process.containment.force_stop()?;
            join_logs(process.logs);
            return Ok(());
        }
        thread::sleep(Duration::from_millis(50));
    }
    process.containment.force_stop()?;
    process
        .child
        .wait()
        .map_err(|error| format!("Could not confirm local gateway shutdown: {error}"))?;
    join_logs(process.logs);
    Ok(())
}

fn prepare_log_path(profile_dir: &Path) -> Result<PathBuf, String> {
    let logs_dir = profile_dir.join("logs");
    fs::create_dir_all(&logs_dir)
        .map_err(|error| format!("Could not create standalone diagnostics directory: {error}"))?;
    restrict_directory(&logs_dir)?;
    let log_path = logs_dir.join("gateway.log");
    if !log_path.exists() {
        let file = create_private_file(&log_path, true)?;
        drop(file);
    }
    restrict_file(&log_path)?;
    Ok(log_path)
}

fn capture_logs(
    child: &mut Child,
    log_path: PathBuf,
    secret: String,
) -> Vec<thread::JoinHandle<()>> {
    let mut drains = Vec::new();
    let write_lock = Arc::new(Mutex::new(()));
    if let Some(stdout) = child.stdout.take() {
        drains.push(capture_stream(
            stdout,
            log_path.clone(),
            secret.clone(),
            Arc::clone(&write_lock),
        ));
    }
    if let Some(stderr) = child.stderr.take() {
        drains.push(capture_stream(stderr, log_path, secret, write_lock));
    }
    drains
}

fn capture_stream<R>(
    mut stream: R,
    log_path: PathBuf,
    secret: String,
    write_lock: Arc<Mutex<()>>,
) -> thread::JoinHandle<()>
where
    R: Read + Send + 'static,
{
    thread::spawn(move || {
        let mut buffer = [0_u8; 4096];
        let mut redactor = SecretRedactor::new(secret);
        loop {
            let read = match stream.read(&mut buffer) {
                Ok(0) | Err(_) => break,
                Ok(read) => read,
            };
            if let Ok(_guard) = write_lock.lock() {
                append_bounded_log(&log_path, &redactor.push(&buffer[..read]));
            }
        }
        if let Ok(_guard) = write_lock.lock() {
            append_bounded_log(&log_path, &redactor.finish());
        }
    })
}

fn append_bounded_log(path: &Path, bytes: &[u8]) {
    if bytes.is_empty() {
        return;
    }
    let mut entry = bytes.to_vec();
    if entry.len() as u64 > MAX_LOG_BYTES {
        entry = entry[entry.len() - MAX_LOG_BYTES as usize..].to_vec();
    }
    let Ok(metadata) = fs::metadata(path) else {
        return;
    };
    if metadata.len().saturating_add(entry.len() as u64) > MAX_LOG_BYTES {
        let rotated = path.with_extension("log.previous");
        let _ = fs::remove_file(&rotated);
        let _ = fs::rename(path, rotated);
        if create_private_file(path, false).is_err() {
            return;
        }
        let _ = restrict_file(path);
    }
    if let Ok(mut log) = fs::OpenOptions::new().append(true).open(path) {
        let _ = log.write_all(&entry);
    }
}

/// Redacts the launch secret even when it is split between two pipe reads.
/// Keeping a small suffix pending trades a few bytes of diagnostic latency for
/// making the token impossible to write verbatim to the diagnostic file.
struct SecretRedactor {
    secret: Vec<u8>,
    pending: Vec<u8>,
}

impl SecretRedactor {
    fn new(secret: String) -> Self {
        Self {
            secret: secret.into_bytes(),
            pending: Vec::new(),
        }
    }

    fn push(&mut self, bytes: &[u8]) -> Vec<u8> {
        self.pending.extend_from_slice(bytes);
        self.drain_ready(false)
    }

    fn finish(&mut self) -> Vec<u8> {
        self.drain_ready(true)
    }

    fn drain_ready(&mut self, final_chunk: bool) -> Vec<u8> {
        if self.secret.is_empty() {
            return std::mem::take(&mut self.pending);
        }

        let retained = if final_chunk {
            0
        } else {
            self.secret.len().saturating_sub(1)
        };
        if self.pending.len() <= retained {
            return Vec::new();
        }

        let mut safe_until = self.pending.len() - retained;
        if !final_chunk {
            let search_start = safe_until.saturating_sub(self.secret.len().saturating_sub(1));
            for start in search_start..safe_until {
                if self.pending[start..].starts_with(&self.secret) {
                    safe_until = start;
                    break;
                }
            }
        }
        if safe_until == 0 {
            return Vec::new();
        }

        let mut redacted = Vec::with_capacity(safe_until);
        let mut offset = 0;
        while offset < safe_until {
            if self.pending[offset..safe_until].starts_with(&self.secret) {
                redacted.extend_from_slice(b"[redacted]");
                offset += self.secret.len();
            } else {
                redacted.push(self.pending[offset]);
                offset += 1;
            }
        }
        self.pending.drain(..safe_until);
        redacted
    }
}

fn join_logs(logs: Vec<thread::JoinHandle<()>>) {
    for log in logs {
        let _ = log.join();
    }
}

#[cfg(unix)]
fn create_private_file(path: &Path, create_new: bool) -> Result<fs::File, String> {
    use std::os::unix::fs::OpenOptionsExt;
    let mut options = fs::OpenOptions::new();
    options.write(true).create(true).mode(0o600);
    if create_new {
        options.create_new(true);
    } else {
        options.truncate(true);
    }
    options
        .open(path)
        .map_err(|error| format!("Could not create standalone diagnostics log: {error}"))
}

#[cfg(not(unix))]
fn create_private_file(path: &Path, create_new: bool) -> Result<fs::File, String> {
    let mut options = fs::OpenOptions::new();
    options.write(true).create(true);
    if create_new {
        options.create_new(true);
    } else {
        options.truncate(true);
    }
    options
        .open(path)
        .map_err(|error| format!("Could not create standalone diagnostics log: {error}"))
}

#[cfg(unix)]
fn restrict_file(path: &Path) -> Result<(), String> {
    use std::os::unix::fs::PermissionsExt;
    fs::set_permissions(path, fs::Permissions::from_mode(0o600))
        .map_err(|error| format!("Could not secure standalone diagnostics: {error}"))
}

#[cfg(not(unix))]
fn restrict_file(_path: &Path) -> Result<(), String> {
    Ok(())
}

#[cfg(unix)]
fn restrict_directory(path: &Path) -> Result<(), String> {
    use std::os::unix::fs::PermissionsExt;
    fs::set_permissions(path, fs::Permissions::from_mode(0o700))
        .map_err(|error| format!("Could not secure standalone diagnostics directory: {error}"))
}

#[cfg(not(unix))]
fn restrict_directory(_path: &Path) -> Result<(), String> {
    Ok(())
}

fn bundled_bins(resource_dir: &Path) -> (PathBuf, PathBuf) {
    let extension = if cfg!(windows) { ".exe" } else { "" };
    let version = env!("CARGO_PKG_VERSION");
    let target = env!("ONEDAY_TARGET_TRIPLE");
    (
        resource_dir.join(format!(
            "binaries/oneday-gateway-v{version}-{target}{extension}"
        )),
        resource_dir.join(format!("binaries/oneday-v{version}-{target}{extension}")),
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn launch_plan_uses_loopback_and_never_places_the_secret_in_arguments() {
        let workspace = tempfile::tempdir().expect("isolated workspace");
        let root = workspace.path().join("runtime");
        let secret = LaunchSecret::generate().expect("secret");
        let plan = LaunchPlan::create(root.clone(), root.join("resources"), None);
        assert!(
            plan.is_err(),
            "missing sidecars should fail safely before launch"
        );

        let (gateway, engine) = bundled_bins(&root.join("resources"));
        let static_index = root.join("resources/gateway/web/dist/index.html");
        fs::create_dir_all(gateway.parent().expect("gateway parent")).expect("bin dir");
        fs::create_dir_all(static_index.parent().expect("static parent")).expect("static dir");
        fs::write(&gateway, "").expect("gateway");
        fs::write(&engine, "").expect("engine");
        fs::write(&static_index, "").expect("index");
        let plan = LaunchPlan::create(root.clone(), root.join("resources"), None).expect("plan");
        assert_eq!(plan.endpoint.scheme(), "http");
        assert_eq!(plan.endpoint.host_str(), Some("127.0.0.1"));
        let arguments = plan
            .arguments(&root.join("config.yaml"))
            .expect("arguments");
        assert!(!arguments
            .iter()
            .any(|argument| argument == &secret.environment_value()));
        let command = plan.command(&secret).expect("standalone command");
        let gateway_url = command
            .get_envs()
            .find(|(name, _)| *name == "ONEDAY_GATEWAY_URL")
            .and_then(|(_, value)| value)
            .expect("gateway readiness URL");
        assert_eq!(gateway_url, plan.endpoint.as_str());
    }

    #[test]
    fn bundled_sidecars_follow_the_desktop_package_version() {
        let root = Path::new("/resources");
        let (gateway, engine) = bundled_bins(root);
        let version = env!("CARGO_PKG_VERSION");
        let target = env!("ONEDAY_TARGET_TRIPLE");
        assert!(gateway
            .to_string_lossy()
            .contains(&format!("-v{version}-{target}")));
        assert!(engine
            .to_string_lossy()
            .contains(&format!("-v{version}-{target}")));
    }

    #[test]
    fn diagnostics_are_bounded_and_redact_the_launch_secret() {
        let temporary = tempfile::tempdir().expect("profile root");
        let log_path = prepare_log_path(temporary.path()).expect("log path");
        let mut redactor = SecretRedactor::new("top-secret".into());
        append_bounded_log(&log_path, &redactor.push(b"token=top-"));
        append_bounded_log(&log_path, &redactor.push(b"secret\n"));
        append_bounded_log(&log_path, &redactor.finish());
        let redacted = fs::read_to_string(&log_path).expect("redacted log contents");
        assert!(redacted.contains("[redacted]"));
        assert!(!redacted.contains("top-secret"));
        append_bounded_log(&log_path, &vec![b'x'; MAX_LOG_BYTES as usize]);
        let contents = fs::read_to_string(log_path).expect("log contents");
        assert!(!contents.contains("top-secret"));
        assert!(contents.len() <= MAX_LOG_BYTES as usize);
    }
}
