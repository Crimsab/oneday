use anyhow::Context;
use opentelemetry::global;
use opentelemetry::trace::TracerProvider as _;
use opentelemetry::KeyValue;
use opentelemetry_otlp::{Protocol, WithExportConfig};
use opentelemetry_sdk::trace::SdkTracerProvider;
use opentelemetry_sdk::Resource;
use std::env;
use tracing_subscriber::layer::SubscriberExt;
use tracing_subscriber::util::SubscriberInitExt;
use tracing_subscriber::EnvFilter;
use tracing_subscriber::Layer;

const DEFAULT_SERVICE_NAME: &str = "oneday-gateway";

#[derive(Clone, Debug)]
pub struct Status {
    pub otlp_traces: &'static str,
}

pub struct Runtime {
    provider: Option<SdkTracerProvider>,
    status: Status,
}

impl Runtime {
    pub fn init() -> anyhow::Result<Self> {
        let fmt_layer = tracing_subscriber::fmt::layer().with_filter(environment_filter());

        if !otlp_enabled()? {
            tracing_subscriber::registry()
                .with(fmt_layer)
                .try_init()
                .context("initializing local tracing subscriber")?;
            return Ok(Self {
                provider: None,
                status: Status {
                    otlp_traces: "disabled",
                },
            });
        }

        let exporter = opentelemetry_otlp::SpanExporter::builder()
            .with_http()
            .with_protocol(Protocol::HttpBinary)
            .build()
            .context("building OTLP/HTTP trace exporter")?;
        let service_name =
            non_empty_env("OTEL_SERVICE_NAME").unwrap_or_else(|| DEFAULT_SERVICE_NAME.to_string());
        let mut attributes = vec![KeyValue::new("service.version", env!("CARGO_PKG_VERSION"))];
        if let Some(environment) = non_empty_env("ONEDAY_DEPLOYMENT_ENVIRONMENT") {
            attributes.push(KeyValue::new("deployment.environment.name", environment));
        }
        let resource = Resource::builder()
            .with_service_name(service_name)
            .with_attributes(attributes)
            .build();
        let provider = SdkTracerProvider::builder()
            .with_resource(resource)
            .with_batch_exporter(exporter)
            .build();
        let tracer = provider.tracer(DEFAULT_SERVICE_NAME);

        tracing_subscriber::registry()
            .with(fmt_layer)
            .with(
                tracing_opentelemetry::layer()
                    .with_tracer(tracer)
                    .with_filter(environment_filter()),
            )
            .try_init()
            .context("initializing tracing subscriber with OTLP export")?;
        global::set_tracer_provider(provider.clone());

        tracing::info!(
            otlp_traces = "enabled",
            otlp_protocol = "http/protobuf",
            "optional OTLP trace export enabled"
        );
        Ok(Self {
            provider: Some(provider),
            status: Status {
                otlp_traces: "enabled",
            },
        })
    }

    pub fn status(&self) -> Status {
        self.status.clone()
    }
}

impl Drop for Runtime {
    fn drop(&mut self) {
        if let Some(provider) = self.provider.take() {
            let _ = provider.shutdown();
        }
    }
}

fn otlp_enabled() -> anyhow::Result<bool> {
    enabled_from(
        non_empty_env("ONEDAY_OTEL_ENABLED").as_deref(),
        non_empty_env("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT").as_deref(),
        non_empty_env("OTEL_EXPORTER_OTLP_ENDPOINT").as_deref(),
    )
}

fn enabled_from(
    explicit: Option<&str>,
    traces_endpoint: Option<&str>,
    shared_endpoint: Option<&str>,
) -> anyhow::Result<bool> {
    match explicit {
        Some(value) => parse_bool(value).with_context(|| {
            format!("ONEDAY_OTEL_ENABLED must be true or false, received {value:?}")
        }),
        None => Ok(traces_endpoint.is_some() || shared_endpoint.is_some()),
    }
}

fn non_empty_env(key: &str) -> Option<String> {
    env::var(key)
        .ok()
        .map(|value| value.trim().to_string())
        .filter(|value| !value.is_empty())
}

fn environment_filter() -> EnvFilter {
    EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new("info"))
}

fn parse_bool(value: &str) -> Option<bool> {
    match value.trim().to_ascii_lowercase().as_str() {
        "1" | "true" | "yes" | "on" => Some(true),
        "0" | "false" | "no" | "off" => Some(false),
        _ => None,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use axum::body::Bytes;
    use axum::extract::State;
    use axum::http::{HeaderMap, StatusCode};
    use axum::routing::post;
    use axum::Router;
    use std::sync::{Arc, Mutex};
    use tokio::sync::oneshot;

    #[test]
    fn parses_explicit_boolean_values() {
        for value in ["1", "true", "YES", "on"] {
            assert_eq!(parse_bool(value), Some(true));
        }
        for value in ["0", "false", "No", "off"] {
            assert_eq!(parse_bool(value), Some(false));
        }
        assert_eq!(parse_bool("sometimes"), None);
    }

    #[test]
    fn explicit_switch_overrides_endpoint_autodetection() {
        assert!(enabled_from(None, None, Some("http://collector:4318")).unwrap());
        assert!(!enabled_from(Some("false"), None, Some("http://collector:4318")).unwrap());
        assert!(enabled_from(Some("true"), None, None).unwrap());
        assert!(enabled_from(Some("sometimes"), None, None).is_err());
    }

    #[tokio::test(flavor = "multi_thread", worker_threads = 2)]
    async fn exports_tracing_spans_as_otlp_http_protobuf() {
        type Received = (String, usize);
        let (sender, receiver) = oneshot::channel::<Received>();
        let sender = Arc::new(Mutex::new(Some(sender)));
        let app = Router::new()
            .route(
                "/v1/traces",
                post(
                    |State(sender): State<Arc<Mutex<Option<oneshot::Sender<Received>>>>>,
                     headers: HeaderMap,
                     body: Bytes| async move {
                        let content_type = headers
                            .get("content-type")
                            .and_then(|value| value.to_str().ok())
                            .unwrap_or_default()
                            .to_string();
                        if let Some(sender) = sender.lock().expect("sender lock").take() {
                            let _ = sender.send((content_type, body.len()));
                        }
                        StatusCode::OK
                    },
                ),
            )
            .with_state(sender);
        let listener = tokio::net::TcpListener::bind("127.0.0.1:0")
            .await
            .expect("mock OTLP listener");
        let address = listener.local_addr().expect("mock OTLP address");
        let server = tokio::spawn(async move { axum::serve(listener, app).await });

        let exporter = opentelemetry_otlp::SpanExporter::builder()
            .with_http()
            .with_protocol(Protocol::HttpBinary)
            .with_endpoint(format!("http://{address}/v1/traces"))
            .build()
            .expect("mock OTLP exporter");
        let provider = SdkTracerProvider::builder()
            .with_simple_exporter(exporter)
            .build();
        let tracer = provider.tracer("oneday-observability-test");
        let subscriber =
            tracing_subscriber::registry().with(tracing_opentelemetry::layer().with_tracer(tracer));
        tracing::subscriber::with_default(subscriber, || {
            let span = tracing::info_span!(
                "image_generation",
                gen_ai.operation.name = "image_generation",
                oneday.image.job.id = 42_i64,
            );
            let _entered = span.enter();
        });

        let (content_type, body_len) =
            tokio::time::timeout(std::time::Duration::from_secs(5), receiver)
                .await
                .expect("OTLP request timeout")
                .expect("OTLP request receiver");
        assert_eq!(content_type, "application/x-protobuf");
        assert!(body_len > 0);
        provider.shutdown().expect("OTLP provider shutdown");
        server.abort();
    }
}
