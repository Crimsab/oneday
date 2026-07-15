import { describe, expect, it } from "vitest";
import {
  buildProviderConfigUpdates,
  editableImageDraftFields,
} from "./imageGenerationDraft";
import type { ImageProviderCatalogEntry } from "../../types";

const provider = (
  id: string,
  extra: Partial<ImageProviderCatalogEntry> = {},
): ImageProviderCatalogEntry => ({
  id,
  display_name: id,
  auth_type: "api_key",
  default: id === "codex-oauth",
  configured: false,
  api_key_configured: false,
  status: "",
  base_url: "",
  models: [],
  model_validation: "allowlist",
  capabilities: {
    generate: true,
    edit: false,
    sizes: [],
    aspect_ratios: [],
    qualities: [],
    output_formats: [],
    supports_transparency: false,
  },
  ...extra,
});

describe("image provider settings payload", () => {
  it("emits only intentionally changed providers and fields", () => {
    const catalog = [
      provider("codex-oauth"),
      provider("openai", { base_url: "https://api.openai.com/v1" }),
      provider("azure-openai"),
    ];
    const updates = buildProviderConfigUpdates(
      catalog,
      {
        openai: {
          baseUrl: "https://api.openai.com/v1",
          apiVersion: "",
          models: "",
          apiKey: "",
          clearApiKey: false,
        },
        "azure-openai": {
          baseUrl: "https://example.openai.azure.com",
          apiVersion: "2025-04-01-preview",
          models: "deployment-a",
          apiKey: "secret",
          clearApiKey: false,
        },
      },
      new Set(["azure-openai"]),
    );
    expect(updates).toEqual([
      {
        id: "azure-openai",
        base_url: "https://example.openai.azure.com",
        api_version: "2025-04-01-preview",
        models: ["deployment-a"],
        api_key: "secret",
        clear_api_key: undefined,
      },
    ]);
  });
  it("supports a fresh configured-model compatible provider without rewriting others", () => {
    const updates = buildProviderConfigUpdates(
      [provider("openai-compatible", { model_validation: "configured" })],
      {
        "openai-compatible": {
          baseUrl: "http://lite.local/v1",
          apiVersion: "",
          models: "flux-custom, icon-custom",
          apiKey: "key",
          clearApiKey: false,
        },
      },
      new Set(["openai-compatible"]),
    );
    expect(updates[0]?.models).toEqual(["flux-custom", "icon-custom"]);
  });
  it("keeps every persisted image draft field represented", () => {
    expect(new Set(editableImageDraftFields).size).toBe(25);
    expect(editableImageDraftFields).toContain("imagegenBridgeFallbackPolicy");
    expect(editableImageDraftFields).toContain("characterResolution");
  });
});
