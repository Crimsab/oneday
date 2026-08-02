use std::fs;
use std::io::Write;
use std::path::PathBuf;
use std::sync::Once;
use std::time::{SystemTime, UNIX_EPOCH};

const MAX_BOOTSTRAP_LOG_BYTES: u64 = 128 * 1024;
fn log_path() -> PathBuf {
    let root = std::env::var_os("LOCALAPPDATA")
        .map(PathBuf::from)
        .or_else(|| {
            std::env::current_exe()
                .ok()
                .and_then(|path| path.parent().map(PathBuf::from))
        })
        .unwrap_or_else(std::env::temp_dir);
    root.join("dev.oneday.desktop")
        .join("logs")
        .join("desktop-bootstrap.log")
}

pub fn record(message: impl AsRef<str>) {
    let path = log_path();
    let Some(parent) = path.parent() else { return };
    if fs::create_dir_all(parent).is_err() {
        return;
    }
    if path
        .metadata()
        .is_ok_and(|metadata| metadata.len() >= MAX_BOOTSTRAP_LOG_BYTES)
    {
        let previous = path.with_extension("log.previous");
        let _ = fs::remove_file(&previous);
        let _ = fs::rename(&path, previous);
    }
    let timestamp = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|duration| duration.as_secs())
        .unwrap_or_default();
    if let Ok(mut log) = fs::OpenOptions::new().create(true).append(true).open(path) {
        let _ = writeln!(log, "[{timestamp}] {}", message.as_ref());
    }
}

#[cfg(windows)]
fn message_box(title: &str, message: &str, style: u32) -> i32 {
    use std::os::windows::ffi::OsStrExt;
    use windows_sys::Win32::UI::WindowsAndMessaging::MessageBoxW;

    let title = std::ffi::OsStr::new(title)
        .encode_wide()
        .chain(std::iter::once(0))
        .collect::<Vec<_>>();
    let message = std::ffi::OsStr::new(message)
        .encode_wide()
        .chain(std::iter::once(0))
        .collect::<Vec<_>>();
    unsafe {
        MessageBoxW(
            std::ptr::null_mut(),
            message.as_ptr(),
            title.as_ptr(),
            style,
        )
    }
}

#[cfg(not(windows))]
fn message_box(title: &str, message: &str, _style: u32) -> i32 {
    eprintln!("{title}: {message}");
    0
}

pub fn fatal(message: &str) {
    record(format!("fatal startup error: {message}"));
    let detail = if message.to_ascii_lowercase().contains("webview2") {
        "OneDay could not start because Microsoft Edge WebView2 Runtime is missing or damaged. Install or repair WebView2, then open OneDay again."
    } else {
        message
    };
    message_box(
        "OneDay could not start",
        &format!("{detail}\n\nDiagnostic log:\n{}", log_path().display()),
        0x0000_0010,
    );
}

pub fn install_panic_hook() {
    static INSTALL: Once = Once::new();
    INSTALL.call_once(|| {
        let previous = std::panic::take_hook();
        std::panic::set_hook(Box::new(move |info| {
            let message = format!("unexpected startup failure: {info}");
            record(&message);
            message_box(
                "OneDay stopped unexpectedly",
                &format!("{message}\n\nDiagnostic log:\n{}", log_path().display()),
                0x0000_0010,
            );
            previous(info);
        }));
    });
}
