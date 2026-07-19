use super::*;

#[tokio::test]
async fn openai_and_compatible_adapters_use_images_contract() {
    let app = Router::new()
        .route(
            "/v1/images/capabilities",
            axum::routing::get(|headers: HeaderMap| async move {
                assert_eq!(
                    headers.get(header::AUTHORIZATION).unwrap(),
                    "Bearer test-key"
                );
                Json(json!({"capabilities": {"images": true}}))
            }),
        )
        .route(
            "/v1/images/generations",
            post(
                |headers: HeaderMap, Json(payload): Json<Value>| async move {
                    assert_eq!(
                        headers.get(header::AUTHORIZATION).unwrap(),
                        "Bearer test-key"
                    );
                    assert_eq!(
                        headers.get("idempotency-key").unwrap(),
                        "oneday-contract-test"
                    );
                    assert_eq!(payload["prompt"], "A clean landmark\nAvoid: text");
                    Json(json!({"data": [{
                        "b64_json": base64::engine::general_purpose::STANDARD.encode(PNG),
                        "revised_prompt": "revised"
                    }]}))
                },
            ),
        );
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let address = listener.local_addr().unwrap();
    tokio::spawn(async move { axum::serve(listener, app).await.unwrap() });

    for provider in ["openai", "openai-compatible"] {
        let generated = generate(
            &Client::new(),
            &direct_config(provider, format!("http://{address}/v1")),
            &direct_request(if provider == "openai" {
                "gpt-image-1"
            } else {
                "custom-image-model"
            }),
        )
        .await
        .unwrap();
        assert!(generated.provider_label.starts_with(provider));
        assert_eq!(generated.bytes, PNG);
    }
}

#[tokio::test]
async fn compatible_adapter_supports_explicit_no_auth_only_after_capability_probe() {
    let app = Router::new()
        .route(
            "/v1/images/capabilities",
            axum::routing::get(|headers: HeaderMap| async move {
                assert!(headers.get(header::AUTHORIZATION).is_none());
                Json(json!({"images": true}))
            }),
        )
        .route(
            "/v1/images/generations",
            post(|headers: HeaderMap| async move {
                assert!(headers.get(header::AUTHORIZATION).is_none());
                Json(json!({"data": [{"b64_json": base64::engine::general_purpose::STANDARD.encode(PNG)}]}))
            }),
        );
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let address = listener.local_addr().unwrap();
    tokio::spawn(async move { axum::serve(listener, app).await.unwrap() });
    let base_url = format!("http://{address}/v1");
    let mut config = direct_config("openai-compatible", base_url);
    let compatible = config.providers.get_mut("openai-compatible").unwrap();
    compatible.api_key.clear();
    compatible.auth_mode = "none".into();
    let generated = generate(&Client::new(), &config, &direct_request("local-image"))
        .await
        .unwrap();
    assert_eq!(generated.bytes, PNG);
}

#[tokio::test]
async fn compatible_adapter_rejects_an_unadvertised_capability_before_dispatch() {
    let app = Router::new()
        .route(
            "/v1/images/capabilities",
            axum::routing::get(|| async { Json(json!({"images": false})) }),
        )
        .route(
            "/v1/images/generations",
            post(|| async { axum::http::StatusCode::INTERNAL_SERVER_ERROR }),
        );
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let address = listener.local_addr().unwrap();
    tokio::spawn(async move { axum::serve(listener, app).await.unwrap() });
    let config = direct_config("openai-compatible", format!("http://{address}/v1"));
    let error = generate(&Client::new(), &config, &direct_request("local-image"))
        .await
        .unwrap_err();
    assert_eq!(error_code(&error), "capability_unavailable");
    assert!(!is_retryable(&error));
}

#[tokio::test]
async fn timed_out_image_dispatch_is_an_unknown_outcome_and_is_not_retried() {
    let app = Router::new().route(
        "/v1/images/generations",
        post(|| async move {
            tokio::time::sleep(std::time::Duration::from_millis(100)).await;
            Json(json!({"data": []}))
        }),
    );
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let address = listener.local_addr().unwrap();
    tokio::spawn(async move { axum::serve(listener, app).await.unwrap() });
    let client = Client::builder()
        .timeout(std::time::Duration::from_millis(5))
        .build()
        .unwrap();
    let error = generate(
        &client,
        &direct_config("openai", format!("http://{address}/v1")),
        &direct_request("gpt-image-1"),
    )
    .await
    .unwrap_err();
    assert_eq!(error_code(&error), "unknown_outcome");
    assert!(!is_retryable(&error));
}

#[tokio::test]
async fn compatible_adapter_accepts_same_origin_private_image_url() {
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let address = listener.local_addr().unwrap();
    let app = Router::new()
        .route(
            "/v1/images/capabilities",
            axum::routing::get(|| async { Json(json!({"images": true})) }),
        )
        .route(
            "/v1/images/generations",
            post(move || async move {
                Json(json!({"data": [{"url": format!("http://{address}/image.png")}]}))
            }),
        )
        .route(
            "/image.png",
            get(|| async { ([(header::CONTENT_TYPE, "image/png")], PNG) }),
        );
    tokio::spawn(async move { axum::serve(listener, app).await.unwrap() });

    let generated = generate(
        &Client::new(),
        &direct_config("openai-compatible", format!("http://{address}/v1")),
        &direct_request("custom-image-model"),
    )
    .await
    .unwrap();
    assert_eq!(generated.bytes, PNG);
}
