use super::*;

#[test]
fn detects_supported_image_formats() {
    assert_eq!(detect_format(PNG).as_deref(), Some("png"));
    assert_eq!(
        detect_format(&[0xff, 0xd8, 0xff, 0x00]).as_deref(),
        Some("jpeg")
    );
    assert_eq!(detect_format(b"RIFF0000WEBP").as_deref(), Some("webp"));
    assert_eq!(detect_format(b"not-an-image"), None);
}

#[test]
fn bridge_model_drops_only_openai_namespace() {
    assert_eq!(normalize_bridge_model("openai/gpt-image-2"), "gpt-image-2");
    assert_eq!(normalize_bridge_model("custom/model"), "custom/model");
}

#[test]
fn validates_bridge_local_and_remote_transport_rules() {
    assert!(validate_bridge_endpoint("http://127.0.0.1:8787", "").is_ok());
    assert!(validate_bridge_endpoint("https://bridge.example.test", "token").is_ok());
    assert!(validate_bridge_endpoint("http://bridge.example.test", "token").is_err());
    assert!(validate_bridge_endpoint("https://bridge.example.test", "").is_err());
}

#[test]
fn text_only_mode_is_intentionally_unavailable_to_image_jobs() {
    let config = direct_config("text-only", String::new());
    assert_eq!(
        validation_error(&config, "", false).as_deref(),
        Some("text-only mode disables image generation")
    );
}

#[test]
fn compatible_no_auth_requires_a_literal_local_endpoint() {
    let local = ProviderConfig {
        base_url: "http://127.0.0.1:8080/v1".to_string(),
        api_key: String::new(),
        auth_mode: "none".to_string(),
        capability_probe_url: "http://127.0.0.1:8080/v1/images/capabilities".to_string(),
        api_version: String::new(),
    };
    assert!(validate_compatible_config(&local).is_ok());

    let private_v4 = ProviderConfig {
        base_url: "http://192.168.1.10:8080/v1".to_string(),
        capability_probe_url: "http://192.168.1.10:8080/v1/images/capabilities".to_string(),
        ..local.clone()
    };
    assert!(validate_compatible_config(&private_v4).is_ok());

    let ula_v6 = ProviderConfig {
        base_url: "http://[fd00::1]:8080/v1".to_string(),
        capability_probe_url: "http://[fd00::1]:8080/v1/images/capabilities".to_string(),
        ..local.clone()
    };
    assert!(validate_compatible_config(&ula_v6).is_ok());

    for remote in ["https://images.example.test/v1", "http://8.8.8.8/v1"] {
        let config = ProviderConfig {
            base_url: remote.to_string(),
            capability_probe_url: format!("{remote}/images/capabilities"),
            ..local.clone()
        };
        assert!(validate_compatible_config(&config)
            .unwrap_err()
            .contains("loopback or private IP-literal"));
    }
}

#[test]
fn fallback_route_supports_provider_and_optional_model() {
    assert_eq!(
        parse_fallback_route("codex-responses:gpt-image-2").unwrap(),
        json!({"provider": "codex-responses", "model": "gpt-image-2"})
    );
    assert_eq!(
        parse_fallback_route("codex-app-server").unwrap(),
        json!({"provider": "codex-app-server"})
    );
}

#[test]
fn validates_format_and_size() {
    assert!(validate_image(PNG, "png").is_ok());
    assert!(validate_image(PNG, "webp").is_err());
    assert!(validate_image(&[], "png").is_err());
}

#[test]
fn routing_keeps_codex_oauth_and_openai_platform_distinct() {
    assert_eq!(adapter_kind("codex-oauth"), Some(AdapterKind::CodexOAuth));
    assert_eq!(adapter_kind("openai"), Some(AdapterKind::OpenAi));
    assert_eq!(
        adapter_kind("openai-compatible"),
        Some(AdapterKind::OpenAiCompatible)
    );
    assert_eq!(adapter_kind("unknown-vendor"), None);
}
