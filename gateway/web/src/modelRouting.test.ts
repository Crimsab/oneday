import { describe, expect, it } from "vitest";
import { draftFromModelSettings, hasModelRoutingChanges, modelRoutingIssues, promoteProvider, splitModelList, updateFromDraft } from "./modelRouting";
import type { ModelSettings } from "./types";

describe("model routing helpers", () => {
  it("builds editable drafts from backend settings", () => {
    const draft = draftFromModelSettings(settings);

    expect(draft.providerPriority).toEqual(["codex", "litellm", "openrouter", "claude-code"]);
    expect(draft.providers.codex).toMatchObject({ enabled: true, model: "gpt-5.4-mini", reasoning: "off" });
    expect(draft.repairFallbackModels).toBe("gpt-5.4-mini");
  });

  it("promotes providers while preserving a complete priority chain", () => {
    expect(promoteProvider(["codex", "litellm"], ["codex", "litellm", "openrouter"], "openrouter")).toEqual([
      "openrouter",
      "codex",
      "litellm",
    ]);
  });

  it("deduplicates comma and newline fallback model lists", () => {
    expect(splitModelList("gpt-5.4-mini, gpt-5.5\ngpt-5.4-mini")).toEqual(["gpt-5.4-mini", "gpt-5.5"]);
  });

  it("creates a shared config update payload", () => {
    const draft = draftFromModelSettings(settings);
    draft.providerPriority = promoteProvider(draft.providerPriority, settings.providers.map((provider) => provider.id), "litellm");
    draft.providers.litellm.enabled = true;
    draft.providers.litellm.model = "gpt-5.4-mini-updated";
    draft.utilityModel = "gpt-5.4-mini-fast";
    draft.repairFallbackModels = "gpt-5.4-mini, gpt-5.5";

    expect(updateFromDraft(settings, draft)).toMatchObject({
      base_revision: "revision-1",
      provider_priority: ["litellm", "codex", "openrouter", "claude-code"],
      utility_model: "gpt-5.4-mini-fast",
      repair_fallback_models: ["gpt-5.4-mini", "gpt-5.5"],
      providers: expect.arrayContaining([
        expect.objectContaining({ id: "litellm", enabled: true, model: "gpt-5.4-mini-updated" }),
      ]),
    });
  });

  it("detects dirty and invalid drafts before save", () => {
    const draft = draftFromModelSettings(settings);
    expect(hasModelRoutingChanges(settings, draft)).toBe(false);
    draft.providers.codex.enabled = false;
    draft.providers.litellm.enabled = false;
    draft.providers.openrouter.enabled = false;
    draft.providers["claude-code"].enabled = false;
    expect(hasModelRoutingChanges(settings, draft)).toBe(true);
    expect(modelRoutingIssues(settings, draft)).toContain("Enable at least one provider.");
  });
});

const settings: ModelSettings = {
  config_path: "/opt/oneday/config.yaml",
  config_revision: "revision-1",
  provider_priority: ["codex", "litellm"],
  providers: [
    {
      id: "codex",
      label: "Codex OAuth",
      enabled: true,
      model: "gpt-5.4-mini",
      reasoning: "off",
      supports_model: true,
      supports_reasoning: true,
    },
    {
      id: "litellm",
      label: "LiteLLM",
      enabled: false,
      model: "gpt-5.4-mini",
      supports_model: true,
      supports_reasoning: false,
    },
    {
      id: "openrouter",
      label: "OpenRouter",
      enabled: false,
      model: "google/gemini-2.5-flash-lite",
      supports_model: true,
      supports_reasoning: false,
    },
    {
      id: "claude-code",
      label: "Claude Code",
      enabled: false,
      supports_model: false,
      supports_reasoning: false,
    },
  ],
  narrative_models: ["gpt-5.4-mini", "google/gemini-2.5-flash-lite"],
  utility_models: ["gpt-5.4-mini"],
  repair_models: ["gemini-3.1-flash-lite-preview", "gpt-5.4-mini"],
  image_models: ["ascii-ambient"],
  embedding_providers: ["auto", "litellm", "openrouter", "local"],
  active: {
    provider: "codex",
    narrative_model: "gpt-5.4-mini",
    utility_model: "gpt-5.4-mini",
    repair_model: "gemini-3.1-flash-lite-preview",
    repair_fallback_models: ["gpt-5.4-mini"],
    image_model: "ascii-ambient",
    embedding_provider: "auto",
    embedding_model: "text-embedding-3-small",
    codex_reasoning: "off",
  },
  tts_status: "planned",
};
