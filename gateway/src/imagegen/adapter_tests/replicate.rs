use super::*;

#[tokio::test]
async fn replicate_adapter_uses_official_model_prediction_contract() {
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let address = listener.local_addr().unwrap();
    let image_url = format!("http://{address}/replicate.png#oneday-test-local");
    let app = Router::new()
        .route(
            "/v1/models/black-forest-labs/flux-schnell/predictions",
            post(move |headers: HeaderMap, Json(payload): Json<Value>| {
                let image_url = image_url.clone();
                async move {
                    assert_eq!(
                        headers.get(header::AUTHORIZATION).unwrap(),
                        "Bearer test-key"
                    );
                    assert_eq!(payload["input"]["prompt"], "A clean landmark\nAvoid: text");
                    Json(json!({"status": "succeeded", "output": [image_url]}))
                }
            }),
        )
        .route(
            "/replicate.png",
            get(|| async { ([(header::CONTENT_TYPE, "image/png")], PNG) }),
        );
    tokio::spawn(async move { axum::serve(listener, app).await.unwrap() });
    let generated = generate(
        &Client::new(),
        &direct_config("replicate", format!("http://{address}/v1")),
        &direct_request("black-forest-labs/flux-schnell"),
    )
    .await
    .unwrap();
    assert_eq!(generated.bytes, PNG);
}

#[tokio::test]
async fn replicate_rejects_cross_origin_polling_urls_before_sending_credentials() {
    let app = Router::new().route(
        "/v1/models/black-forest-labs/flux-schnell/predictions",
        post(|| async {
            Json(json!({
                "status": "processing",
                "urls": {"get": "https://attacker.invalid/predictions/1"}
            }))
        }),
    );
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let address = listener.local_addr().unwrap();
    tokio::spawn(async move { axum::serve(listener, app).await.unwrap() });

    let error = generate(
        &Client::new(),
        &direct_config("replicate", format!("http://{address}/v1")),
        &direct_request("black-forest-labs/flux-schnell"),
    )
    .await
    .unwrap_err();
    assert!(error.to_string().contains("cross-origin polling URL"));
}
