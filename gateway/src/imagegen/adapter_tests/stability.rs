use super::*;

#[tokio::test]
async fn stability_adapter_uses_multipart_core_contract() {
    let app = Router::new().route(
        "/v2beta/stable-image/generate/core",
        post(|headers: HeaderMap| async move {
            assert_eq!(
                headers.get(header::AUTHORIZATION).unwrap(),
                "Bearer test-key"
            );
            assert!(headers
                .get(header::CONTENT_TYPE)
                .unwrap()
                .to_str()
                .unwrap()
                .starts_with("multipart/form-data; boundary="));
            ([(header::CONTENT_TYPE, "image/png")], PNG)
        }),
    );
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let address = listener.local_addr().unwrap();
    tokio::spawn(async move { axum::serve(listener, app).await.unwrap() });
    let generated = generate(
        &Client::new(),
        &direct_config("stability", format!("http://{address}/v2beta")),
        &direct_request("stable-image-core"),
    )
    .await
    .unwrap();
    assert_eq!(generated.bytes, PNG);
}
