# Image provider compatibility

OneDay exposes image providers as explicit, persisted choices. **Codex OAuth**
is the first and default catalog entry. It uses imagegen-bridge as its Codex-only
transport; imagegen-bridge is not a provider-neutral gateway and OneDay never
routes vendor API keys through it.

## Supported providers

| OneDay provider ID | Display name | Authentication | Adapter contract |
| --- | --- | --- | --- |
| `codex-oauth` | Codex OAuth | Existing Codex/ChatGPT OAuth in imagegen-bridge | Native imagegen-bridge `/v1/images`; `codex-responses` is recommended and `codex-app-server` is the Codex compatibility fallback. |
| `openai` | OpenAI Platform | OpenAI Platform API key | Direct `/v1/images/generations`. This is separate from Codex OAuth. |
| `openai-compatible` | OpenAI-compatible / LiteLLM | Endpoint API key | Direct `/images/generations`, only for gateways that implement the image request and response schema—not merely chat completions. |
| `gemini` | Google Gemini | Gemini API key | Direct Gemini `v1beta/interactions` image response. |
| `fal` | fal.ai | fal key | Direct queue API submission, bounded polling, and result download. |
| `replicate` | Replicate | API token | Direct official-model prediction API with bounded polling. Models use `owner/model`. |
| `stability` | Stability AI | Stability API key | Direct Stable Image Core v2beta multipart API. |
| `azure-openai` | Azure OpenAI Images | Azure resource API key | Direct Azure OpenAI `/openai/v1/images/generations`; the model field is the deployment name. |

Legacy `openclaw-bridge` configuration remains readable for compatibility but
is not a catalog provider for new setups. OneDay does not advertise Imagen's
separate Vertex AI `predict` surface: the current Google adapter implements the
stable Gemini Developer API image contract without adding a heavyweight cloud
SDK or pretending that Vertex credentials are interchangeable.

## Codex OAuth configuration

```yaml
ai:
  image_generation:
    provider: codex-oauth
    map_icon_provider: codex-oauth
    model: gpt-image-2
    map_icon_model: gpt-image-1
    imagegen_bridge_url: http://127.0.0.1:8787
    imagegen_bridge_token: ${ONEDAY_IMAGEGEN_BRIDGE_TOKEN}
    imagegen_bridge_provider: codex-responses
    imagegen_bridge_map_icon_provider: codex-responses
    imagegen_bridge_fallbacks:
      - codex-app-server:gpt-image-2
    imagegen_bridge_fallback_policy: on_error
    imagegen_bridge_compatibility: normalize
```

Only `codex-responses` and `codex-app-server` routes are accepted in bridge
configuration. A third-party provider route is a validation error.

## Direct provider configuration

Scene art and map icons may use different providers and models. Credentials are
stored server-side under `providers`; they are accepted as write-only Settings
updates and are never returned by the API.

```yaml
ai:
  image_generation:
    provider: gemini
    model: gemini-3.1-flash-image
    map_icon_provider: openai
    map_icon_model: gpt-image-1
    providers:
      openai:
        base_url: https://api.openai.com/v1
        api_key: ${ONEDAY_IMAGEGEN_OPENAI_API_KEY}
        models: [gpt-image-1]
      gemini:
        base_url: https://generativelanguage.googleapis.com/v1beta
        api_key: ${ONEDAY_IMAGEGEN_GEMINI_API_KEY}
        models: [gemini-3.1-flash-image]
```

Official base URLs are defaults for OpenAI, Gemini, fal.ai, Replicate, and
Stability. `openai-compatible` and `azure-openai` require an explicit endpoint.
Azure also accepts `api_version` (default `preview`).

## Settings contract and secret handling

`GET /api/config/models` returns `image_providers` in stable display order and
the selected `image_generation` route. Catalog entries contain ID, display
name, authentication kind, redacted configuration status, safe endpoint and
API version, configured models, model-validation mode, and capabilities.

`PUT /api/config/models` accepts provider-specific write-only updates under
`image_generation.provider_configs`:

```json
{
  "base_revision": "...",
  "image_generation": {
    "provider": "gemini",
    "model": "gemini-3.1-flash-image",
    "provider_configs": [{
      "id": "gemini",
      "api_key": "write-only",
      "models": ["gemini-3.1-flash-image"]
    }]
  }
}
```

Use `clear_api_key: true` to remove a vendor key. Codex bridge tokens use the
separate write-only `imagegen_bridge_token` and
`clear_imagegen_bridge_token`. Responses expose only `configured` and
`api_key_configured`; keys and bridge tokens are never echoed. There is no
connection-test endpoint: saving performs local contract validation, while an
actual provider request occurs only as an explicit asynchronous generation job.

Legacy `imagegen-bridge`, `imagegen_bridge`, and `bridge-native` provider IDs
are normalized to `codex-oauth`. A legacy Codex bridge configuration with no
image model defaults at runtime to the recommended `gpt-image-2` for scene art
and map icons; new Settings writes persist the user's explicit selections.

## Failure and retry behavior

Provider/model routing is explicit. OneDay does not silently fall back between
providers with different credentials or costs. Codex OAuth may use only its
configured Codex-internal bridge fallback policy. Jobs have bounded attempts;
authentication, validation, unsupported-parameter, invalid-response, and
post-accept timeout errors are terminal. Only failures known to be safe to
retry, such as connection failures, rate limits, and upstream 5xx responses,
are requeued. The persisted asset metadata records the provider and model that
actually produced the image.
