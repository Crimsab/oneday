use super::*;

#[tokio::test]
async fn generated_image_download_blocks_local_destinations_and_local_redirects() {
    let local = reqwest::Url::parse("http://127.0.0.1:1/image.png").unwrap();
    assert!(validate_public_destination(&local, false).await.is_err());

    let app = Router::new()
        .route(
            "/start",
            get(|| async {
                (
                    [(header::LOCATION, "/image.png")],
                    axum::http::StatusCode::FOUND,
                )
            }),
        )
        .route(
            "/image.png",
            get(|| async { ([(header::CONTENT_TYPE, "image/png")], PNG) }),
        );
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let address = listener.local_addr().unwrap();
    tokio::spawn(async move { axum::serve(listener, app).await.unwrap() });

    let error = download_image(
        &Client::new(),
        &format!("http://{address}/start#oneday-test-local"),
        "png",
        "redirect test",
        None,
    )
    .await
    .unwrap_err();
    assert!(error.to_string().contains("non-public"));
}

#[tokio::test]
async fn generated_image_download_blocks_non_public_ipv4_ranges() {
    for address in [
        "100.64.0.1",
        "198.18.0.1",
        "192.0.0.1",
        "192.0.2.1",
        "198.51.100.1",
        "203.0.113.1",
        "240.0.0.1",
    ] {
        let url = reqwest::Url::parse(&format!("http://{address}/image.png")).unwrap();
        assert!(
            validate_public_destination(&url, false).await.is_err(),
            "{address} must not be considered public"
        );
    }
}
