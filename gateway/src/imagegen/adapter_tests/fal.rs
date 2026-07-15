use super::*;

#[tokio::test]
async fn fal_adapter_accepts_queue_result_with_image_url() {
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let address = listener.local_addr().unwrap();
    let image_url = format!("http://{address}/image.png#oneday-test-local");
    let app = Router::new()
        .route(
            "/fal-ai/flux/schnell",
            post(move |headers: HeaderMap| {
                let image_url = image_url.clone();
                async move {
                    assert_eq!(headers.get(header::AUTHORIZATION).unwrap(), "Key test-key");
                    Json(json!({"images": [{"url": image_url, "content_type": "image/png"}]}))
                }
            }),
        )
        .route(
            "/image.png",
            get(|| async { ([(header::CONTENT_TYPE, "image/png")], PNG) }),
        );
    tokio::spawn(async move { axum::serve(listener, app).await.unwrap() });
    let generated = generate(
        &Client::new(),
        &direct_config("fal", format!("http://{address}")),
        &direct_request("fal-ai/flux/schnell"),
    )
    .await
    .unwrap();
    assert_eq!(generated.bytes, PNG);
}

#[tokio::test]
async fn fal_rejects_cross_origin_polling_urls_before_sending_credentials() {
    let app = Router::new().route(
        "/fal-ai/flux/schnell",
        post(|| async {
            Json(json!({
                "request_id": "request-1",
                "status_url": "https://attacker.invalid/status",
                "response_url": "https://attacker.invalid/result"
            }))
        }),
    );
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let address = listener.local_addr().unwrap();
    tokio::spawn(async move { axum::serve(listener, app).await.unwrap() });

    let error = generate(
        &Client::new(),
        &direct_config("fal", format!("http://{address}")),
        &direct_request("fal-ai/flux/schnell"),
    )
    .await
    .unwrap_err();
    assert!(error.to_string().contains("cross-origin polling URL"));
}
