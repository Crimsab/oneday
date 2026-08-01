use tauri::{AppHandle, Manager, WebviewUrl, WebviewWindowBuilder};
use url::Url;

pub fn same_origin(allowed: &Url, candidate: &Url) -> bool {
    allowed.scheme() == candidate.scheme()
        && allowed.host_str() == candidate.host_str()
        && allowed.port_or_known_default() == candidate.port_or_known_default()
}

fn navigation_origin(url: &Url) -> Url {
    let mut origin = url.clone();
    origin.set_path("/");
    origin.set_query(None);
    origin.set_fragment(None);
    origin
}

pub fn open(app: &AppHandle, server_url: &Url) -> Result<(), String> {
    let allowed = navigation_origin(server_url);
    if let Some(window) = app.get_webview_window("main") {
        let current = window
            .url()
            .map_err(|error| format!("Could not inspect the OneDay window: {error}"))?;
        if same_origin(&allowed, &current) {
            window
                .navigate(server_url.clone())
                .map_err(|error| format!("Could not navigate to the OneDay server: {error}"))?;
            window.show().map_err(|error| error.to_string())?;
            window.unminimize().map_err(|error| error.to_string())?;
            window.set_focus().map_err(|error| error.to_string())?;
            return Ok(());
        }

        // A standalone restart normally selects a fresh loopback port. The
        // existing webview's navigation guard intentionally rejects that new
        // origin, so replace the webview instead of leaving it on a dead port.
        window
            .destroy()
            .map_err(|error| format!("Could not replace the previous OneDay window: {error}"))?;
    }

    WebviewWindowBuilder::new(app, "main", WebviewUrl::External(server_url.clone()))
        .title("OneDay")
        .inner_size(1440.0, 900.0)
        .min_inner_size(900.0, 620.0)
        .center()
        .devtools(cfg!(debug_assertions))
        .on_navigation(move |candidate| same_origin(&allowed, candidate))
        .build()
        .map_err(|error| format!("Could not open the OneDay window: {error}"))?;
    Ok(())
}

pub fn show(app: &AppHandle) -> Result<(), String> {
    let window = app
        .get_webview_window("main")
        .ok_or_else(|| "Connect to a OneDay server first.".to_string())?;
    window.show().map_err(|error| error.to_string())?;
    window.unminimize().map_err(|error| error.to_string())?;
    window.set_focus().map_err(|error| error.to_string())
}

pub fn close(app: &AppHandle) {
    if let Some(window) = app.get_webview_window("main") {
        let _ = window.close();
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn navigation_stays_on_the_configured_origin() {
        let allowed = Url::parse("https://oneday.example.com/").unwrap();
        assert!(same_origin(
            &allowed,
            &Url::parse("https://oneday.example.com/stories/one/map").unwrap()
        ));
        assert!(!same_origin(
            &allowed,
            &Url::parse("https://attacker.example/stories/one").unwrap()
        ));
        assert!(!same_origin(
            &allowed,
            &Url::parse("http://oneday.example.com/").unwrap()
        ));
    }

    #[test]
    fn navigation_origin_drops_paths_queries_and_credentials() {
        let origin = navigation_origin(
            &Url::parse("http://127.0.0.1:48788/api/auth/bootstrap?token=secret").unwrap(),
        );
        assert_eq!(origin.as_str(), "http://127.0.0.1:48788/");
    }
}
