use crate::config;
use crate::secret::LaunchSecret;
use std::fs;
use std::net::TcpListener;
use std::path::{Path, PathBuf};
use std::process::{Child, Command, Stdio};
use std::thread;
use std::time::{Duration, Instant};
use tauri::{AppHandle, Manager};
use url::Url;

pub const MAX_START_ATTEMPTS: usize = 2;
const SHUTDOWN_WAIT: Duration = Duration::from_secs(5);

pub struct LocalProcess {
    child: Child,
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
}

impl LaunchPlan {
    fn create(profile_dir: PathBuf, resource_dir: PathBuf) -> Result<Self, String> {
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
        fs::create_dir_all(&self.profile_dir).map_err(|error| {
            format!("Could not create the standalone profile directory: {error}")
        })?;
        let config_path = self.profile_dir.join("config.yaml");
        fs::write(&config_path, "data_dir: ./data\n")
            .map_err(|error| format!("Could not prepare standalone configuration: {error}"))?;
        let mut command = Command::new(&self.gateway_bin);
        command
            .args(self.arguments(&config_path)?)
            // The launch token is intentionally an environment value, never a
            // command-line argument or persisted setting.
            .env("ONEDAY_GATEWAY_AUTH_TOKEN", secret.environment_value())
            .stdin(Stdio::null())
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .current_dir(&self.profile_dir);
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
    let plan = LaunchPlan::create(profile_dir, resource_dir)?;
    let child = plan
        .command(&secret)?
        .spawn()
        .map_err(|error| format!("Could not start the local OneDay gateway: {error}"))?;
    Ok(LocalProcess {
        child,
        endpoint: plan.endpoint,
        _secret: secret,
    })
}

pub fn stop(mut process: LocalProcess) -> Result<(), String> {
    request_graceful_stop(&mut process.child)?;
    let deadline = Instant::now() + SHUTDOWN_WAIT;
    while Instant::now() < deadline {
        if process
            .child
            .try_wait()
            .map_err(|error| format!("Could not inspect local gateway shutdown: {error}"))?
            .is_some()
        {
            return Ok(());
        }
        thread::sleep(Duration::from_millis(50));
    }
    process
        .child
        .kill()
        .map_err(|error| format!("Could not stop the local OneDay gateway: {error}"))?;
    process
        .child
        .wait()
        .map_err(|error| format!("Could not confirm local gateway shutdown: {error}"))?;
    Ok(())
}

#[cfg(unix)]
fn request_graceful_stop(child: &mut Child) -> Result<(), String> {
    // The gateway's existing shutdown signal is Ctrl-C (SIGINT).
    let result = unsafe { libc::kill(child.id() as libc::pid_t, libc::SIGINT) };
    if result == 0 {
        Ok(())
    } else {
        Err(format!(
            "Could not request local gateway shutdown: {}",
            std::io::Error::last_os_error()
        ))
    }
}

#[cfg(not(unix))]
fn request_graceful_stop(_child: &mut Child) -> Result<(), String> {
    // The current gateway exposes no cross-platform shutdown endpoint. The
    // bounded fallback below terminates only this child after Draining.
    Ok(())
}

fn bundled_bins(resource_dir: &Path) -> (PathBuf, PathBuf) {
    let extension = if cfg!(windows) { ".exe" } else { "" };
    (
        resource_dir.join(format!("binaries/oneday-gateway-v0.1.0{extension}")),
        resource_dir.join(format!("binaries/oneday-v0.1.0{extension}")),
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn launch_plan_uses_loopback_and_never_places_the_secret_in_arguments() {
        let root = std::env::current_dir()
            .expect("workspace")
            .join("target")
            .join(format!("oneday-desktop-{}", std::process::id()));
        let secret = LaunchSecret::generate().expect("secret");
        let plan = LaunchPlan::create(root.clone(), root.join("resources"));
        assert!(
            plan.is_err(),
            "missing sidecars should fail safely before launch"
        );

        let gateway = root.join("resources/binaries/oneday-gateway-v0.1.0");
        let engine = root.join("resources/binaries/oneday-v0.1.0");
        let static_index = root.join("resources/gateway/web/dist/index.html");
        fs::create_dir_all(gateway.parent().expect("gateway parent")).expect("bin dir");
        fs::create_dir_all(static_index.parent().expect("static parent")).expect("static dir");
        fs::write(&gateway, "").expect("gateway");
        fs::write(&engine, "").expect("engine");
        fs::write(&static_index, "").expect("index");
        let plan = LaunchPlan::create(root.clone(), root.join("resources")).expect("plan");
        assert_eq!(plan.endpoint.scheme(), "http");
        assert_eq!(plan.endpoint.host_str(), Some("127.0.0.1"));
        let arguments = plan
            .arguments(&root.join("config.yaml"))
            .expect("arguments");
        assert!(!arguments
            .iter()
            .any(|argument| argument == &secret.environment_value()));
        let _ = fs::remove_dir_all(root);
    }
}
