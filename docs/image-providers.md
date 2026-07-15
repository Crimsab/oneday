# Image provider compatibility

OneDay has three image adapter families. The preferred path is the native
imagegen-bridge contract because it keeps provider discovery, capabilities,
fallbacks, revised prompts, and verified output metadata outside the story
engine.

## Current matrix

| Route | Status in OneDay | Authentication | Notes |
| --- | --- | --- | --- |
| imagegen-bridge → `codex-responses` | Default | Codex/ChatGPT OAuth | First-class bridge provider for `gpt-image-2`, `gpt-image-1.5`, `gpt-image-1`, and `gpt-image-1-mini`. |
| imagegen-bridge → `codex-app-server` | Supported fallback | Codex/ChatGPT OAuth | Supports the Codex app-server lifecycle and a smaller parameter set. |
| Future or third-party imagegen-bridge provider | Contract-ready | Provider-defined | OneDay forwards the registered provider/model route without requiring a new client adapter. The provider must exist in the running bridge. |
| OpenAI Platform `/images/generations` | Supported directly | API key | Configure an OpenAI-compatible `base_url`, key, and model. This path is separate from Codex OAuth. |
| LiteLLM or another OpenAI-compatible gateway | Supported conditionally | Gateway-defined | Works when the gateway implements the same `/images/generations` path and response schema. |
| OpenClaw image bridge | Legacy compatibility | Bridge-defined | Uses the configured OpenClaw generation URL. It is not required by the preferred imagegen-bridge path. |
| Provider-specific Gemini, fal, Replicate, Stability, Azure, or similar API | No direct adapter | Provider-defined | Use an imagegen-bridge provider or a compatible gateway. OneDay does not guess vendor-specific paths, headers, or payloads. |

The current imagegen-bridge release ships two built-in providers:
`codex-responses` and `codex-app-server`. Its official OpenAI Platform provider
is reserved but not implemented there; OneDay can still call the official
OpenAI-compatible image endpoint through its direct adapter.

## Preferred configuration

```yaml
ai:
  image_generation:
    provider: imagegen-bridge
    imagegen_bridge_url: http://127.0.0.1:8787
    imagegen_bridge_token: ${ONEDAY_IMAGEGEN_BRIDGE_TOKEN}
    imagegen_bridge_provider: codex-responses
    imagegen_bridge_map_icon_provider: codex-responses
    imagegen_bridge_fallbacks:
      - codex-app-server:gpt-image-2
    imagegen_bridge_fallback_policy: on_error
    imagegen_bridge_compatibility: normalize
    auto_generate: true
```

From Docker, replace the loopback host with
`http://host.docker.internal:8787` when the bridge runs on the host.

## Direct OpenAI-compatible configuration

```yaml
ai:
  image_generation:
    provider: openai
    base_url: https://api.openai.com/v1
    api_key: ${ONEDAY_IMAGEGEN_API_KEY}
    model: gpt-image-1
    map_icon_model: gpt-image-1
    auto_generate: true
```

Use this adapter only for endpoints that deliberately implement the familiar
OpenAI image-generation contract. A text-completions-compatible endpoint is not
automatically image-compatible.

## Capability and failure behavior

The native bridge validates provider/model capabilities before generation and
returns the actual provider used for every fallback attempt. OneDay keeps image
work asynchronous: an unavailable provider records a bounded failure without
rolling back or blocking canonical story text.

General scene art and map symbols may use different routes. Safety refusals,
cancellation, and invalid requests are never silently rerouted as technical
fallbacks. See [Generated media](media.md) and
[Configuration](configuration.md) for all controls.
