use super::*;

#[tokio::test]
async fn native_bridge_request_preserves_routing_policies_and_negative_prompt() {
    let (capture_tx, capture_rx) = oneshot::channel::<(HeaderMap, Value)>();
    let capture_tx = std::sync::Arc::new(std::sync::Mutex::new(Some(capture_tx)));
    let app = Router::new().route(
        "/v1/images",
        post({
            let capture_tx = capture_tx.clone();
            move |headers: HeaderMap, Json(payload): Json<Value>| {
                let capture_tx = capture_tx.clone();
                async move {
                    if let Some(tx) = capture_tx.lock().unwrap().take() {
                        let _ = tx.send((headers, payload));
                    }
                    Json(json!({
                        "id": "img-test",
                        "created": 1,
                        "provider": "codex-responses",
                        "model": "gpt-image-2",
                        "requested": {},
                        "effective": {},
                        "data": [{
                            "type": "b64_json",
                            "index": 0,
                            "b64_json": base64::engine::general_purpose::STANDARD.encode(PNG),
                            "format": "png",
                            "width": 1,
                            "height": 1,
                            "bytes": PNG.len(),
                            "sha256": "fixture"
                        }],
                        "revised_prompt": "revised",
                        "timings": {}
                    }))
                }
            }
        }),
    );
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let address = listener.local_addr().unwrap();
    tokio::spawn(async move { axum::serve(listener, app).await.unwrap() });

    let config = AdapterConfig {
        provider: "codex-oauth".into(),
        map_icon_provider: "codex-oauth".into(),
        providers: HashMap::new(),
        bridge_url: format!("http://{address}"),
        bridge_token: "test-token".into(),
        bridge_provider: "codex-app-server".into(),
        bridge_map_icon_provider: "codex-responses".into(),
        bridge_fallbacks: vec!["codex-app-server:gpt-image-2".into()],
        bridge_fallback_policy: "on_error".into(),
        bridge_compatibility: "normalize".into(),
        base_url: String::new(),
        api_key: String::new(),
        openclaw_url: String::new(),
    };
    let request = GenerateRequest {
        subject: "map icon".into(),
        prompt: "A clean landmark".into(),
        negative_prompt: "text".into(),
        model: "openai/gpt-image-2".into(),
        is_map_icon: true,
        size: "1024x1024".into(),
        resolution: Some("1k".into()),
        aspect_ratio: Some("1:1".into()),
        quality: "medium".into(),
        output_format: "png".into(),
        background: "transparent".into(),
        timeout_ms: 30_000,
        idempotency_key: "oneday-test".into(),
    };
    let generated = generate(&Client::new(), &config, &request).await.unwrap();
    let (headers, payload) = capture_rx.await.unwrap();

    assert_eq!(
        headers.get(header::AUTHORIZATION).unwrap(),
        "Bearer test-token"
    );
    assert_eq!(payload["negative_prompt"], "text");
    assert_eq!(payload["routing"]["provider"], "codex-responses");
    assert_eq!(payload["routing"]["model"], "gpt-image-2");
    assert_eq!(payload["routing"]["fallback_policy"], "on_error");
    assert_eq!(payload["parameters"]["action"], "auto");
    assert_eq!(
        payload["routing"]["fallbacks"][0],
        json!({"provider": "codex-app-server", "model": "gpt-image-2"})
    );
    assert_eq!(payload["session"]["mode"], "isolated");
    assert_eq!(payload["output"]["response_format"], "b64_json");
    assert_eq!(
        generated.provider_label,
        "codex-oauth/codex-responses:gpt-image-2"
    );
    assert_eq!(generated.revised_prompt, "revised");
    assert_eq!(generated.bytes, PNG);
}

#[tokio::test]
async fn native_edit_keeps_manual_compatibility_when_a_legacy_bridge_has_no_probe() {
    let app = Router::new()
        .route("/v1/providers/codex-responses/capabilities", get(|| async { axum::http::StatusCode::NOT_FOUND }))
        .route("/v1/images/edits", post(|| async { Json(json!({
            "data": [{ "b64_json": base64::engine::general_purpose::STANDARD.encode(PNG), "revised_prompt": "legacy" }],
            "imagegen_bridge": { "provider": "codex-responses", "model": "gpt-image-2" }
        })) }));
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let address = listener.local_addr().unwrap();
    tokio::spawn(async move { axum::serve(listener, app).await.unwrap() });
    let config = AdapterConfig {
        provider: "codex-oauth".into(),
        map_icon_provider: "codex-oauth".into(),
        providers: HashMap::new(),
        bridge_url: format!("http://{address}"),
        bridge_token: String::new(),
        bridge_provider: "codex-responses".into(),
        bridge_map_icon_provider: "codex-responses".into(),
        bridge_fallbacks: vec![],
        bridge_fallback_policy: "on_unavailable".into(),
        bridge_compatibility: "normalize".into(),
        base_url: String::new(),
        api_key: String::new(),
        openclaw_url: String::new(),
    };
    let request = NativeImageRequest {
        operation: ImageOperation::Edit,
        provider: "codex-oauth".into(),
        endpoint_id: "/v1/images/edits".into(),
        source: Some(super::CanonicalImage {
            png: PNG.to_vec(),
            width: 1,
            height: 1,
        }),
        prompt: "Adjust the lantern".into(),
        negative_prompt: String::new(),
        model: "gpt-image-2".into(),
        size: String::new(),
        quality: String::new(),
        output_format: "png".into(),
        idempotency_key: "legacy-edit".into(),
        mask: None,
    };
    let generated = crate::imagegen::codex_bridge::edit(&Client::new(), &config, &request)
        .await
        .unwrap();
    assert_eq!(generated.revised_prompt, "legacy");
}
