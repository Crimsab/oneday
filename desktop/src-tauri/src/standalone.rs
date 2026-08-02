use crate::claude_component;
use crate::codex_component;
use crate::config;
use crate::containment::{self, ProcessContainment};
use crate::secret::LaunchSecret;
use std::fs;
use std::io::{Read, Write};
use std::net::TcpListener;
use std::path::{Path, PathBuf};
use std::process::{Child, Command, ExitStatus, Stdio};
use std::sync::{Arc, Mutex};
use std::thread;
use std::time::{Duration, Instant};
use tauri::{AppHandle, Manager};
use url::Url;

pub const MAX_START_ATTEMPTS: usize = 2;
const SHUTDOWN_WAIT: Duration = Duration::from_secs(5);
const MAX_LOG_BYTES: u64 = 256 * 1024;
const GATEWAY_PORT_FILE: &str = "gateway-port";

pub struct LocalProcess {
    child: Child,
    containment: ProcessContainment,
    logs: Vec<thread::JoinHandle<()>>,
    imagegen_bridge: Option<ImagegenBridgeProcess>,
    gateway_log_path: PathBuf,
    pub endpoint: Url,
    pub browser_url: Url,
    _credentials: LaunchCredentials,
}

impl LocalProcess {
    pub fn pid(&self) -> u32 {
        self.child.id()
    }

    pub fn exit_status(&mut self) -> Result<Option<ExitStatus>, String> {
        self.child
            .try_wait()
            .map_err(|error| format!("Could not inspect the local OneDay gateway: {error}"))
    }

    pub fn gateway_log_path(&self) -> &Path {
        &self.gateway_log_path
    }
}

struct ImagegenBridgeProcess {
    child: Child,
    containment: ProcessContainment,
    logs: Vec<thread::JoinHandle<()>>,
}

struct LaunchCredentials {
    bearer: LaunchSecret,
    bootstrap: LaunchSecret,
    imagegen_bridge: LaunchSecret,
}

impl LaunchCredentials {
    fn generate() -> Result<Self, String> {
        Ok(Self {
            bearer: LaunchSecret::generate()?,
            bootstrap: LaunchSecret::generate()?,
            imagegen_bridge: LaunchSecret::generate()?,
        })
    }

    fn browser_url(&self, endpoint: &Url) -> Result<Url, String> {
        let mut url = endpoint
            .join("api/auth/bootstrap")
            .map_err(|error| format!("Could not create the local browser session URL: {error}"))?;
        url.query_pairs_mut()
            .append_pair("token", &self.bootstrap.environment_value());
        Ok(url)
    }

    fn redaction_values(&self) -> Vec<String> {
        vec![
            self.bearer.environment_value(),
            self.bootstrap.environment_value(),
            self.imagegen_bridge.environment_value(),
        ]
    }
}

#[derive(Debug)]
struct LaunchPlan {
    endpoint: Url,
    profile_dir: PathBuf,
    gateway_bin: PathBuf,
    engine_bin: PathBuf,
    imagegen_bridge_bin: Option<PathBuf>,
    imagegen_bridge_endpoint: Option<Url>,
    static_dir: PathBuf,
    codex: Option<codex_component::CodexRuntime>,
    claude: Option<claude_component::ClaudeRuntime>,
}

impl LaunchPlan {
    fn create(
        profile_dir: PathBuf,
        resource_dir: PathBuf,
        codex: Option<codex_component::CodexRuntime>,
        claude: Option<claude_component::ClaudeRuntime>,
    ) -> Result<Self, String> {
        // Keep the standalone web origin stable across launches. Browser
        // preferences are origin-scoped, so a fresh random port on every
        // start made locale and UI settings appear to reset.
        let endpoint = reserve_profile_loopback_endpoint(&profile_dir)?;
        let (gateway_bin, engine_bin, bundled_imagegen_bridge) = bundled_bins(&resource_dir);
        let imagegen_bridge_bin = developer_imagegen_bridge_binary().or_else(|| {
            bundled_imagegen_bridge
                .is_file()
                .then_some(bundled_imagegen_bridge)
        });
        let imagegen_bridge_endpoint = if codex.is_some() && imagegen_bridge_bin.is_some() {
            Some(reserve_loopback_endpoint()?)
        } else {
            None
        };
        let static_dir = resource_dir.join("gateway/web/dist");
        let plan = Self {
            endpoint,
            profile_dir,
            gateway_bin,
            engine_bin,
            imagegen_bridge_bin,
            imagegen_bridge_endpoint,
            static_dir,
            codex,
            claude,
        };
        plan.validate()?;
        Ok(plan)
    }

    fn validate(&self) -> Result<(), String> {
        if !self.gateway_bin.is_file() || !self.engine_bin.is_file() {
            return Err(
                "This portable build is incomplete: keep oneday-desktop.exe together with the binaries folder and extract the complete OneDay package before starting it.".into(),
            );
        }
        if !self.static_dir.join("index.html").is_file() {
            return Err(
                "This standalone build is missing the bundled OneDay web interface.".into(),
            );
        }
        Ok(())
    }

    fn command(
        &self,
        credentials: &LaunchCredentials,
        imagegen_bridge_active: bool,
    ) -> Result<Command, String> {
        let config_path = config::create_initial_standalone_config(&self.profile_dir)?;
        let mut command = Command::new(&self.gateway_bin);
        command
            .args(self.arguments(&config_path)?)
            // The launch token is intentionally an environment value, never a
            // command-line argument or persisted setting.
            .env(
                "ONEDAY_GATEWAY_AUTH_TOKEN",
                credentials.bearer.environment_value(),
            )
            .env(
                "ONEDAY_GATEWAY_BOOTSTRAP_TOKEN",
                credentials.bootstrap.environment_value(),
            )
            .env("ONEDAY_GATEWAY_URL", self.endpoint.as_str())
            .stdin(Stdio::null())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .current_dir(&self.profile_dir);
        if imagegen_bridge_active {
            let endpoint = self.imagegen_bridge_endpoint.as_ref().ok_or_else(|| {
                "The local image component was marked active without an endpoint.".to_string()
            })?;
            command
                .env("ONEDAY_IMAGEGEN_BRIDGE_URL", endpoint.as_str())
                .env(
                    "ONEDAY_IMAGEGEN_BRIDGE_TOKEN",
                    credentials.imagegen_bridge.environment_value(),
                );
        } else {
            // An explicit blank value is intentional. It prevents a saved
            // remote bridge URL or the historical localhost default from
            // pretending that image generation is available in this local
            // profile when its managed component is not running.
            command
                .env("ONEDAY_IMAGEGEN_BRIDGE_URL", "")
                .env("ONEDAY_IMAGEGEN_BRIDGE_TOKEN", "");
        }
        self.configure_codex_environment(&mut command)?;
        containment::configure(&mut command);
        Ok(command)
    }

    fn imagegen_bridge_command(
        &self,
        credentials: &LaunchCredentials,
    ) -> Result<Option<Command>, String> {
        let (Some(binary), Some(endpoint), Some(_)) = (
            self.imagegen_bridge_bin.as_ref(),
            self.imagegen_bridge_endpoint.as_ref(),
            self.codex.as_ref(),
        ) else {
            return Ok(None);
        };
        let config_path = config::create_imagegen_bridge_config(&self.profile_dir)?;
        let address = endpoint
            .socket_addrs(|| None)
            .map_err(|error| {
                format!("Could not resolve the local image component endpoint: {error}")
            })?
            .into_iter()
            .next()
            .ok_or_else(|| "Could not resolve the local image component endpoint.".to_string())?;
        let mut command = Command::new(binary);
        command
            .arg("--config")
            .arg(config_path)
            .arg("serve")
            .arg("--bind")
            .arg(address.to_string())
            .env(
                "ONEDAY_IMAGEGEN_BRIDGE_TOKEN",
                credentials.imagegen_bridge.environment_value(),
            )
            .env("IMAGEGEN_BRIDGE_NO_UPDATE_CHECK", "1")
            .stdin(Stdio::null())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .current_dir(&self.profile_dir);
        self.configure_codex_environment(&mut command)?;
        containment::configure(&mut command);
        Ok(Some(command))
    }

    fn configure_codex_environment(&self, command: &mut Command) -> Result<(), String> {
        if let Some(codex) = &self.codex {
            if let Some(home) = &codex.home {
                command.env("CODEX_HOME", home);
                command.env_remove("OPENAI_API_KEY");
            }
            if let Some(pathext) = &codex.pathext {
                command.env("PATHEXT", pathext);
            }
        }
        let component_paths = [
            self.codex
                .as_ref()
                .map(|runtime| runtime.binary_dir.as_path()),
            self.claude
                .as_ref()
                .map(|runtime| runtime.binary_dir.as_path()),
        ]
        .into_iter()
        .flatten()
        .collect::<Vec<_>>();
        if !component_paths.is_empty() {
            command.env("PATH", codex_component::prepend_paths(&component_paths)?);
        }
        Ok(())
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

fn reserve_loopback_endpoint() -> Result<Url, String> {
    let listener = TcpListener::bind("127.0.0.1:0")
        .map_err(|error| format!("Could not reserve a loopback port: {error}"))?;
    let address = listener
        .local_addr()
        .map_err(|error| format!("Could not inspect the loopback port: {error}"))?;
    // The gateway and image component accept an address rather than a
    // pre-bound socket. Releasing this immediately keeps the endpoint dynamic;
    // the caller retries startup if another local process wins the short race.
    drop(listener);
    Url::parse(&format!("http://127.0.0.1:{}/", address.port()))
        .map_err(|error| format!("Could not create the local endpoint: {error}"))
}

fn reserve_profile_loopback_endpoint(profile_dir: &Path) -> Result<Url, String> {
    fs::create_dir_all(profile_dir)
        .map_err(|error| format!("Could not create the standalone profile directory: {error}"))?;
    restrict_directory(profile_dir)?;
    let port_path = profile_dir.join(GATEWAY_PORT_FILE);

    if let Ok(saved) = fs::read_to_string(&port_path) {
        if let Ok(port) = saved.trim().parse::<u16>() {
            if port >= 1024 {
                if let Ok(listener) = TcpListener::bind(("127.0.0.1", port)) {
                    drop(listener);
                    return Url::parse(&format!("http://127.0.0.1:{port}/"))
                        .map_err(|error| format!("Could not restore the local endpoint: {error}"));
                }
            }
        }
    }

    let endpoint = reserve_loopback_endpoint()?;
    let port = endpoint
        .port()
        .ok_or_else(|| "Could not inspect the reserved loopback port.".to_string())?;
    let mut temporary = tempfile::NamedTempFile::new_in(profile_dir)
        .map_err(|error| format!("Could not stage the local endpoint: {error}"))?;
    restrict_file(temporary.path())?;
    temporary
        .write_all(format!("{port}\n").as_bytes())
        .and_then(|()| temporary.as_file().sync_all())
        .map_err(|error| format!("Could not save the local endpoint: {error}"))?;
    temporary
        .persist(&port_path)
        .map_err(|error| format!("Could not save the local endpoint: {}", error.error))?;
    Ok(endpoint)
}

#[cfg(debug_assertions)]
fn developer_imagegen_bridge_binary() -> Option<PathBuf> {
    std::env::var_os("ONEDAY_IMAGEGEN_BRIDGE_BIN")
        .map(PathBuf::from)
        .filter(|path| path.is_file())
}

#[cfg(not(debug_assertions))]
fn developer_imagegen_bridge_binary() -> Option<PathBuf> {
    None
}

pub async fn start(app: &AppHandle, profile_id: &str) -> Result<LocalProcess, String> {
    let profile_dir = config::standalone_profile_dir(app, profile_id)?;
    let resource_dir = app
        .path()
        .resource_dir()
        .map_err(|error| format!("Could not locate bundled standalone components: {error}"))?;
    let credentials = LaunchCredentials::generate()?;
    let plan = LaunchPlan::create(
        profile_dir,
        resource_dir,
        codex_component::runtime(app)?,
        claude_component::runtime(),
    )?;
    let secrets = credentials.redaction_values();
    let mut imagegen_bridge = match start_imagegen_bridge(&plan, &credentials, &secrets).await {
        Ok(process) => process,
        Err(error) => {
            record_imagegen_bridge_failure(&plan.profile_dir, &error);
            None
        }
    };
    let log_path = match prepare_log_path(&plan.profile_dir, "gateway") {
        Ok(path) => path,
        Err(error) => {
            if let Some(bridge) = imagegen_bridge.take() {
                let _ = stop_imagegen_bridge(bridge);
            }
            return Err(error);
        }
    };
    let browser_url = match credentials.browser_url(&plan.endpoint) {
        Ok(url) => url,
        Err(error) => {
            if let Some(bridge) = imagegen_bridge.take() {
                let _ = stop_imagegen_bridge(bridge);
            }
            return Err(error);
        }
    };
    let mut gateway_command = match plan.command(&credentials, imagegen_bridge.is_some()) {
        Ok(command) => command,
        Err(error) => {
            if let Some(bridge) = imagegen_bridge.take() {
                let _ = stop_imagegen_bridge(bridge);
            }
            return Err(error);
        }
    };
    let mut child = match gateway_command.spawn() {
        Ok(child) => child,
        Err(error) => {
            if let Some(bridge) = imagegen_bridge.take() {
                let _ = stop_imagegen_bridge(bridge);
            }
            return Err(format!("Could not start the local OneDay gateway: {error}"));
        }
    };
    let containment = match ProcessContainment::attach(&child) {
        Ok(containment) => containment,
        Err(error) => {
            let _ = child.kill();
            let _ = child.wait();
            if let Some(bridge) = imagegen_bridge.take() {
                let _ = stop_imagegen_bridge(bridge);
            }
            return Err(error);
        }
    };
    let logs = capture_logs(&mut child, log_path.clone(), secrets);
    Ok(LocalProcess {
        child,
        containment,
        logs,
        imagegen_bridge,
        gateway_log_path: log_path,
        endpoint: plan.endpoint,
        browser_url,
        _credentials: credentials,
    })
}

fn record_imagegen_bridge_failure(profile_dir: &Path, error: &str) {
    let Ok(log_path) = prepare_log_path(profile_dir, "imagegen-bridge") else {
        return;
    };
    append_bounded_log(
        &log_path,
        format!("[OneDay] Local image component is unavailable: {error}\n").as_bytes(),
    );
}

pub fn stop(mut process: LocalProcess) -> Result<(), String> {
    let gateway = stop_managed_process(
        &mut process.child,
        &process.containment,
        &mut process.logs,
        "local OneDay gateway",
    );
    let bridge = match process.imagegen_bridge {
        Some(bridge) => stop_imagegen_bridge(bridge),
        None => Ok(()),
    };
    gateway.and(bridge)
}

async fn start_imagegen_bridge(
    plan: &LaunchPlan,
    credentials: &LaunchCredentials,
    secrets: &[String],
) -> Result<Option<ImagegenBridgeProcess>, String> {
    let Some(mut command) = plan.imagegen_bridge_command(credentials)? else {
        return Ok(None);
    };
    let endpoint = plan
        .imagegen_bridge_endpoint
        .as_ref()
        .expect("image component command has an endpoint")
        .clone();
    let log_path = prepare_log_path(&plan.profile_dir, "imagegen-bridge")?;
    let mut child = command
        .spawn()
        .map_err(|error| format!("Could not start the local image component: {error}"))?;
    let containment = match ProcessContainment::attach(&child) {
        Ok(containment) => containment,
        Err(error) => {
            let _ = child.kill();
            let _ = child.wait();
            return Err(error);
        }
    };
    let logs = capture_logs(&mut child, log_path, secrets.to_vec());
    let mut process = ImagegenBridgeProcess {
        child,
        containment,
        logs,
    };
    if let Err(error) = wait_for_imagegen_bridge(&mut process.child, &endpoint).await {
        let _ = stop_managed_process(
            &mut process.child,
            &process.containment,
            &mut process.logs,
            "local image component",
        );
        return Err(error);
    }
    Ok(Some(process))
}

async fn wait_for_imagegen_bridge(child: &mut Child, endpoint: &Url) -> Result<(), String> {
    const RETRY_INTERVAL: Duration = Duration::from_millis(75);
    const STARTUP_WAIT: Duration = Duration::from_secs(12);
    let client = reqwest::Client::builder()
        .connect_timeout(Duration::from_millis(500))
        .timeout(Duration::from_millis(750))
        .build()
        .map_err(|error| format!("Could not prepare the local image component check: {error}"))?;
    let live = endpoint
        .join("health/live")
        .map_err(|error| format!("Could not create the local image component check: {error}"))?;
    let started = Instant::now();
    loop {
        if let Some(status) = child
            .try_wait()
            .map_err(|error| format!("Could not inspect the local image component: {error}"))?
        {
            return Err(format!(
                "The local image component stopped during startup ({status}). Check its private diagnostics log."
            ));
        }
        if let Ok(response) = client.get(live.clone()).send().await {
            if response.status().is_success() {
                return Ok(());
            }
        }
        if started.elapsed() >= STARTUP_WAIT {
            return Err(
                "The local image component did not become ready. Check the private diagnostics log and restart OneDay."
                    .into(),
            );
        }
        tokio::time::sleep(RETRY_INTERVAL).await;
    }
}

fn stop_imagegen_bridge(mut process: ImagegenBridgeProcess) -> Result<(), String> {
    stop_managed_process(
        &mut process.child,
        &process.containment,
        &mut process.logs,
        "local image component",
    )
}

fn stop_managed_process(
    child: &mut Child,
    containment: &ProcessContainment,
    logs: &mut Vec<thread::JoinHandle<()>>,
    label: &str,
) -> Result<(), String> {
    containment.request_graceful_stop()?;
    let deadline = Instant::now() + SHUTDOWN_WAIT;
    while Instant::now() < deadline {
        if child
            .try_wait()
            .map_err(|error| format!("Could not inspect {label} shutdown: {error}"))?
            .is_some()
        {
            // A process can exit before a descendant it started. The process
            // group/job remains ours, so clear any stragglers before returning.
            containment.force_stop()?;
            join_logs(std::mem::take(logs));
            return Ok(());
        }
        thread::sleep(Duration::from_millis(50));
    }
    containment.force_stop()?;
    child
        .wait()
        .map_err(|error| format!("Could not confirm {label} shutdown: {error}"))?;
    join_logs(std::mem::take(logs));
    Ok(())
}

fn prepare_log_path(profile_dir: &Path, component: &str) -> Result<PathBuf, String> {
    let logs_dir = profile_dir.join("logs");
    fs::create_dir_all(&logs_dir)
        .map_err(|error| format!("Could not create standalone diagnostics directory: {error}"))?;
    restrict_directory(&logs_dir)?;
    let log_path = logs_dir.join(format!("{component}.log"));
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
    secrets: Vec<String>,
) -> Vec<thread::JoinHandle<()>> {
    let mut drains = Vec::new();
    let write_lock = Arc::new(Mutex::new(()));
    if let Some(stdout) = child.stdout.take() {
        drains.push(capture_stream(
            stdout,
            log_path.clone(),
            secrets.clone(),
            Arc::clone(&write_lock),
        ));
    }
    if let Some(stderr) = child.stderr.take() {
        drains.push(capture_stream(stderr, log_path, secrets, write_lock));
    }
    drains
}

fn capture_stream<R>(
    mut stream: R,
    log_path: PathBuf,
    secrets: Vec<String>,
    write_lock: Arc<Mutex<()>>,
) -> thread::JoinHandle<()>
where
    R: Read + Send + 'static,
{
    thread::spawn(move || {
        let mut buffer = [0_u8; 4096];
        let mut redactor = SecretRedactor::new(secrets);
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
    secrets: Vec<Vec<u8>>,
    longest: usize,
    pending: Vec<u8>,
}

impl SecretRedactor {
    fn new(secrets: Vec<String>) -> Self {
        let secrets = secrets
            .into_iter()
            .map(String::into_bytes)
            .filter(|secret| !secret.is_empty())
            .collect::<Vec<_>>();
        let longest = secrets.iter().map(Vec::len).max().unwrap_or(0);
        Self {
            secrets,
            longest,
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
        if self.secrets.is_empty() {
            return std::mem::take(&mut self.pending);
        }

        let retained = if final_chunk {
            0
        } else {
            self.longest.saturating_sub(1)
        };
        if self.pending.len() <= retained {
            return Vec::new();
        }

        let mut safe_until = self.pending.len() - retained;
        if !final_chunk {
            let search_start = safe_until.saturating_sub(self.longest.saturating_sub(1));
            for start in search_start..safe_until {
                if self
                    .secrets
                    .iter()
                    .any(|secret| self.pending[start..].starts_with(secret))
                {
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
            if let Some(secret) = self
                .secrets
                .iter()
                .filter(|secret| self.pending[offset..safe_until].starts_with(secret))
                .max_by_key(|secret| secret.len())
            {
                redacted.extend_from_slice(b"[redacted]");
                offset += secret.len();
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

fn bundled_bins(resource_dir: &Path) -> (PathBuf, PathBuf, PathBuf) {
    let extension = if cfg!(windows) { ".exe" } else { "" };
    let version = env!("CARGO_PKG_VERSION");
    let target = env!("ONEDAY_TARGET_TRIPLE");
    (
        resource_dir.join(format!(
            "binaries/oneday-gateway-v{version}-{target}{extension}"
        )),
        resource_dir.join(format!("binaries/oneday-v{version}-{target}{extension}")),
        resource_dir.join(format!("binaries/imagegen-bridge-{target}{extension}")),
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn launch_plan_uses_loopback_and_never_places_the_secret_in_arguments() {
        let workspace = tempfile::tempdir().expect("isolated workspace");
        let root = workspace.path().join("runtime");
        let credentials = LaunchCredentials::generate().expect("credentials");
        let plan = LaunchPlan::create(root.clone(), root.join("resources"), None, None);
        assert!(
            plan.is_err(),
            "missing sidecars should fail safely before launch"
        );

        let (gateway, engine, _) = bundled_bins(&root.join("resources"));
        let static_index = root.join("resources/gateway/web/dist/index.html");
        fs::create_dir_all(gateway.parent().expect("gateway parent")).expect("bin dir");
        fs::create_dir_all(static_index.parent().expect("static parent")).expect("static dir");
        fs::write(&gateway, "").expect("gateway");
        fs::write(&engine, "").expect("engine");
        fs::write(&static_index, "").expect("index");
        let plan =
            LaunchPlan::create(root.clone(), root.join("resources"), None, None).expect("plan");
        assert_eq!(plan.endpoint.scheme(), "http");
        assert_eq!(plan.endpoint.host_str(), Some("127.0.0.1"));
        let arguments = plan
            .arguments(&root.join("config.yaml"))
            .expect("arguments");
        assert!(!arguments
            .iter()
            .any(|argument| argument == &credentials.bearer.environment_value()));
        assert!(!arguments
            .iter()
            .any(|argument| argument == &credentials.bootstrap.environment_value()));
        let command = plan
            .command(&credentials, false)
            .expect("standalone command");
        let gateway_url = command
            .get_envs()
            .find(|(name, _)| *name == "ONEDAY_GATEWAY_URL")
            .and_then(|(_, value)| value)
            .expect("gateway readiness URL");
        assert_eq!(gateway_url, plan.endpoint.as_str());
        let bootstrap = command
            .get_envs()
            .find(|(name, _)| *name == "ONEDAY_GATEWAY_BOOTSTRAP_TOKEN")
            .and_then(|(_, value)| value)
            .expect("browser bootstrap token");
        assert_eq!(
            bootstrap,
            std::ffi::OsStr::new(&credentials.bootstrap.environment_value())
        );
        let image_bridge_url = command
            .get_envs()
            .find(|(name, _)| *name == "ONEDAY_IMAGEGEN_BRIDGE_URL")
            .and_then(|(_, value)| value)
            .expect("explicit image bridge disable");
        assert_eq!(image_bridge_url, std::ffi::OsStr::new(""));
        assert_ne!(
            credentials.bearer.environment_value(),
            credentials.bootstrap.environment_value()
        );
        let browser_url = credentials
            .browser_url(&plan.endpoint)
            .expect("browser URL");
        assert_eq!(browser_url.path(), "/api/auth/bootstrap");
        assert_eq!(
            browser_url
                .query_pairs()
                .find(|(name, _)| name == "token")
                .map(|(_, value)| value.into_owned()),
            Some(credentials.bootstrap.environment_value())
        );
    }

    #[test]
    fn bundled_sidecars_follow_the_desktop_package_version() {
        let root = Path::new("/resources");
        let (gateway, engine, imagegen_bridge) = bundled_bins(root);
        let version = env!("CARGO_PKG_VERSION");
        let target = env!("ONEDAY_TARGET_TRIPLE");
        assert!(gateway
            .to_string_lossy()
            .contains(&format!("-v{version}-{target}")));
        assert!(engine
            .to_string_lossy()
            .contains(&format!("-v{version}-{target}")));
        assert!(imagegen_bridge
            .to_string_lossy()
            .contains(&format!("imagegen-bridge-{target}")));
    }

    #[test]
    fn standalone_profile_reuses_its_web_origin() {
        let temporary = tempfile::tempdir().expect("profile root");
        let first = reserve_profile_loopback_endpoint(temporary.path()).expect("first endpoint");
        let second = reserve_profile_loopback_endpoint(temporary.path()).expect("second endpoint");
        assert_eq!(first, second);
        assert_eq!(
            fs::read_to_string(temporary.path().join(GATEWAY_PORT_FILE))
                .expect("saved port")
                .trim(),
            first.port().expect("port").to_string()
        );
    }

    #[test]
    fn diagnostics_are_bounded_and_redact_the_launch_secret() {
        let temporary = tempfile::tempdir().expect("profile root");
        let log_path = prepare_log_path(temporary.path(), "gateway").expect("log path");
        let mut redactor = SecretRedactor::new(vec!["top-secret".into(), "other-secret".into()]);
        append_bounded_log(&log_path, &redactor.push(b"token=top-"));
        append_bounded_log(&log_path, &redactor.push(b"secret other-"));
        append_bounded_log(&log_path, &redactor.push(b"secret\n"));
        append_bounded_log(&log_path, &redactor.finish());
        let redacted = fs::read_to_string(&log_path).expect("redacted log contents");
        assert!(redacted.contains("[redacted]"));
        assert!(!redacted.contains("top-secret"));
        assert!(!redacted.contains("other-secret"));
        append_bounded_log(&log_path, &vec![b'x'; MAX_LOG_BYTES as usize]);
        let contents = fs::read_to_string(log_path).expect("log contents");
        assert!(!contents.contains("top-secret"));
        assert!(contents.len() <= MAX_LOG_BYTES as usize);
    }
}
