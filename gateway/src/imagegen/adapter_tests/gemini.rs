use super::*;

#[tokio::test]
async fn gemini_adapter_uses_interactions_contract() {
    let app = Router::new().route(
        "/v1beta/interactions",
        post(
            |headers: HeaderMap, Json(payload): Json<Value>| async move {
                assert_eq!(headers.get("x-goog-api-key").unwrap(), "test-key");
                assert_eq!(payload["model"], "gemini-3.1-flash-image");
                assert_eq!(payload["response_format"]["aspect_ratio"], "1:1");
                Json(json!({"output_image": {
                    "data": base64::engine::general_purpose::STANDARD.encode(PNG),
                    "mime_type": "image/png"
                }}))
            },
        ),
    );
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let address = listener.local_addr().unwrap();
    tokio::spawn(async move { axum::serve(listener, app).await.unwrap() });
    let generated = generate(
        &Client::new(),
        &direct_config("gemini", format!("http://{address}/v1beta")),
        &direct_request("gemini-3.1-flash-image"),
    )
    .await
    .unwrap();
    assert_eq!(generated.bytes, PNG);
    assert_eq!(generated.provider_label, "gemini:gemini-3.1-flash-image");
}

#[tokio::test]
async fn gemini_adapter_accepts_image_content_blocks() {
    let app = Router::new().route(
        "/v1beta/interactions",
        post(|| async move {
            Json(json!({"steps": [{"content": [{
                "type": "image",
                "data": base64::engine::general_purpose::STANDARD.encode(PNG),
                "mime_type": "image/png"
            }]}]}))
        }),
    );
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let address = listener.local_addr().unwrap();
    tokio::spawn(async move { axum::serve(listener, app).await.unwrap() });

    let generated = generate(
        &Client::new(),
        &direct_config("gemini", format!("http://{address}/v1beta")),
        &direct_request("gemini-3.1-flash-image"),
    )
    .await
    .unwrap();
    assert_eq!(generated.bytes, PNG);
}
