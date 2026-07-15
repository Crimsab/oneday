# Observability and traces

OneDay keeps useful diagnostics locally by default and can optionally export
OpenTelemetry traces. The export path uses OTLP over HTTP with protobuf, so it
works with Langfuse, an OpenTelemetry Collector, Grafana Tempo, Jaeger, and
other receivers that accept that protocol.

Export is opt-in. OneDay continues to work without an observability backend.

## Signals available without an external service

- Structured gateway logs describe request and background-job decisions.
- Redacted generation runs, attempts, timings, token usage, provider/model
  lineage, bounded error classes, and retry reasons are retained in SQLite.
- The browser exposes per-message diagnostics and a bounded story telemetry
  export in JSONL format.
- `/api/health` reports whether the OTLP exporter was initialized, without
  returning endpoints or authorization headers.

These local signals remain available when external export is disabled or the
receiver is temporarily unavailable.

## Enable a generic OTLP receiver

Set a collector base endpoint when the receiver follows the standard OTLP HTTP
layout:

```dotenv
ONEDAY_OTEL_ENABLED=true
OTEL_SERVICE_NAME=oneday-gateway
ONEDAY_DEPLOYMENT_ENVIRONMENT=production
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
```

The exporter appends `/v1/traces` to the base endpoint. If the receiver gives
you a complete signal URL instead, use:

```dotenv
OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=https://telemetry.example.com/v1/traces
```

Use `ONEDAY_OTEL_ENABLED=false` as an explicit kill switch when a shared host
already defines `OTEL_EXPORTER_OTLP_*` variables.

Optional standard controls include:

```dotenv
OTEL_RESOURCE_ATTRIBUTES=service.namespace=oneday,service.instance.id=example
OTEL_TRACES_SAMPLER=parentbased_traceidratio
OTEL_TRACES_SAMPLER_ARG=0.25
```

The ratio is between `0.0` and `1.0`. Start with full sampling during setup,
then reduce it only when trace volume or ingestion cost justifies the loss of
detail.

## Connect Langfuse

Langfuse accepts OTLP HTTP/protobuf at its public OTLP endpoint. For the EU
cloud region:

```dotenv
ONEDAY_OTEL_ENABLED=true
OTEL_SERVICE_NAME=oneday-gateway
OTEL_EXPORTER_OTLP_ENDPOINT=https://cloud.langfuse.com/api/public/otel
OTEL_EXPORTER_OTLP_HEADERS=Authorization=Basic%20<base64-public-key-colon-secret-key>,x-langfuse-ingestion-version=4
```

Choose the endpoint for the Langfuse region that owns the project. A
self-hosted installation normally uses
`https://langfuse.example.com/api/public/otel`. The signal-specific endpoint is
the same base path followed by `/v1/traces`.

Create the Basic authorization value outside application logs and store it in
the deployment secret mechanism. Never commit the public/secret key pair or its
encoded value. See the
[Langfuse OpenTelemetry integration](https://langfuse.com/integrations/native/opentelemetry)
for current regions and ingestion requirements.

## Verify the connection

Restart OneDay after changing exporter variables, then check the redacted
health surface:

```bash
curl -fsS http://localhost:8788/api/health | jq '.observability'
```

Expected initialization result:

```json
{
  "otlp_traces": "enabled"
}
```

This status proves that the exporter initialized. It does not prove that a
remote receiver accepted a span. Generate one test image or request, then find
the `oneday-gateway` service in the receiver. If it does not appear, inspect
gateway logs for OTLP export errors and verify the endpoint, HTTP/protobuf
support, authorization header, network route, and TLS trust.

For a local Collector, its debug exporter is a useful first destination before
adding a production backend.

## Privacy boundary

Exported image spans may contain the job and asset kind, requested and resolved
model, actual provider, duration, status, and bounded error class. They do not
contain prompts, revised prompts, story text, image bytes, bearer tokens, API
keys, or configured OTLP headers.

SQLite diagnostics are richer and remain local to the OneDay data directory.
Treat telemetry JSONL exports as user data because identifiers and operational
metadata can still reveal how a story was used.

## Failure behavior

- Invalid required exporter configuration fails startup when
  `ONEDAY_OTEL_ENABLED=true`.
- Disabling OTLP never disables local structured logs or persisted generation
  diagnostics.
- A receiver outage does not roll back canonical story state or a successful
  image result.
- `/api/health` deliberately reports only `enabled` or `disabled`; credentials
  and endpoints stay redacted.

See [Configuration](configuration.md) for the environment variable reference
and [Generated media](media.md) for image-job failure and lineage behavior.
