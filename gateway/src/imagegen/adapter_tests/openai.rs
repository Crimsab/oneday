use super::*;

#[tokio::test]
async fn openai_and_compatible_adapters_use_images_contract() {
    let app = Router::new().route(
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
async fn compatible_adapter_accepts_same_origin_private_image_url() {
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let address = listener.local_addr().unwrap();
    let app = Router::new()
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
