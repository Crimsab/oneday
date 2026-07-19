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

## Native editing and inpainting

OneDay keeps generation, contextual editing, raster inpainting, global
image-to-image transforms, variations, reference generation, and outpainting
as distinct operations. The Settings catalog exposes structured descriptors
for the selected provider/model/endpoint; the gateway validates the same
contract again immediately before dispatch.

| Route | Contextual `edit` | Raster `inpaint` | Notes |
| --- | --- | --- | --- |
| Codex OAuth | Yes | No | `imagegen-bridge` attaches the source to the Codex turn. A mask is rejected with `CODEX_RASTER_MASK_UNSUPPORTED`; it is never converted into prompt text. |
| OpenAI Platform GPT Image | Yes | Yes | Uses `/images/edits`; the internal coverage mask is converted to transparent-is-edit alpha. Mask adherence is provider best-effort. |
| Azure OpenAI GPT Image deployment | Yes | Yes | Uses the deployment-specific Image Edit API and the same transparent-is-edit conversion. |
| Gemini image model | Yes | No | Source-image semantic editing through `v1beta/interactions`; Gemini semantic masking is not treated as a raster mask. |
| Stability AI | No | Yes | Uses `/stable-image/edit/inpaint`; the normalized black-preserve/white-edit mask is sent directly. |
| fal.ai, Replicate | Not yet advertised | Not yet advertised | Their capabilities are endpoint/model-version specific. Generation remains supported; OneDay does not infer edit support from the provider name. |
| OpenAI-compatible / LiteLLM | Disabled by default | Disabled by default | Transport compatibility alone does not prove equivalent image semantics or fallback fidelity. |

The browser editor stores a full-resolution `L8` coverage raster independently
from its red display overlay: `0` preserves a pixel, `255` allows editing, and
intermediate values provide feathering. The original version is immutable.
Every successful operation creates a child asset version with its parent,
operation, mask, branch, and selected-version history preserved.

Operations are asynchronous and never silently downgrade fidelity. The API
returns `202 Accepted`, then exposes `queued`, `running`, `succeeded`, or
`failed` in `operations` on the visual-assets response. Reusing an
idempotency key with a different source, prompt, route, mask hash, or output
contract returns `IDEMPOTENCY_CONFLICT`.

```json
{
  "operation": "inpaint",
  "source_version_id": 42,
  "mask_png_base64": "<opaque grayscale PNG>",
  "prompt": "Replace the lamp with a candle",
  "fallback": { "mode": "forbid" },
  "idempotency_key": "a-client-generated-uuid"
}
```

Submit it to
`POST /api/stories/{story_id}/visual-assets/{asset_id}/operations`. Provider
and model default to the active Settings route; callers may provide them only
when deliberately selecting another configured route. Status recovery is also
available at
`GET /api/stories/{story_id}/image-operations/{operation_id}`.

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

### Optional Docker profile

The public OneDay Compose stack does not include a Codex login or start a bridge
by default. The opt-in [`compose.imagegen-bridge.yaml`](../compose.imagegen-bridge.yaml)
profile connects OneDay to a private Compose-network bridge and exposes the
bridge dashboard only on host loopback. It pins the bridge release and stores
Codex OAuth, bridge state, and artifacts separately. Follow the
[Docker profile instructions](docker.md#optional-codex-oauth-imagegen-bridge-profile)
to generate a distinct bearer and copy only `auth.json` into the dedicated
OAuth volume.

The bridge bearer is required even for loopback use and is separate from Codex
OAuth. Never commit either value, mount an entire home directory, or publish an
unauthenticated bridge. If the bridge is operated remotely, use a trusted
private network or trusted TLS reverse proxy and give OneDay its private bridge
URL and bearer through the normal environment/configuration fields.

Codex OAuth is optional. The direct OpenAI, Gemini, fal.ai, Replicate,
Stability, Azure OpenAI, and OpenAI-compatible adapters do not use the bridge.
A local OpenAI-compatible endpoint (including a Z-Image-class deployment) must
implement the image-generation contract and explicit capability probe; being
compatible only with chat completions is insufficient. For text-only stories,
leave image auto-generation disabled and configure no visual provider.

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
