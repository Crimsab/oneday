use super::*;

#[tokio::test]
async fn azure_openai_adapter_uses_api_key_and_preview_endpoint() {
    let app = Router::new().route(
        "/openai/v1/images/generations",
        post(
            |headers: HeaderMap, Json(payload): Json<Value>| async move {
                assert_eq!(headers.get("api-key").unwrap(), "test-key");
                assert_eq!(payload["model"], "my-image-deployment");
                Json(json!({"data": [{
                    "b64_json": base64::engine::general_purpose::STANDARD.encode(PNG)
                }]}))
            },
        ),
    );
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let address = listener.local_addr().unwrap();
    tokio::spawn(async move { axum::serve(listener, app).await.unwrap() });
    let generated = generate(
        &Client::new(),
        &direct_config("azure-openai", format!("http://{address}")),
        &direct_request("my-image-deployment"),
    )
    .await
    .unwrap();
    assert_eq!(generated.bytes, PNG);
}
