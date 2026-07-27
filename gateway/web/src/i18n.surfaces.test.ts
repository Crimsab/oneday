import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

describe("important localized surfaces", () => {
  it.each([
    ["components/TopBar.tsx", "Current story status"],
    ["components/LeftRail.tsx", "Story library"],
    ["components/Composer.tsx", "What do you want to try?"],
    ["components/settings/SettingsWorkspace.tsx", "Search options"],
    ["components/BranchNavigator.tsx", "Available story branches"],
    ["components/Transcript.tsx", "Choose a story to load the canonical transcript."],
    ["components/MessageBranchControls.tsx", "Story alternatives for turn"],
    ["components/HistoryReader.tsx", "Search branch history"],
    ["components/MiniGameHost.tsx", "Challenge Host"],
    ["components/SuggestedActions.tsx", "Suggested action"],
    ["components/Inspector.tsx", "Crafting conversation"],
    ["components/AudioControls.tsx", "Speech synthesis is in progress."],
    ["components/AudioLanguageTools.tsx", "Pronunciation lexicon"],
    ["components/PanelDrawer.tsx", "Automatic challenges"],
    ["components/PanelDrawer.tsx", "Model Routing"],
    ["components/PanelDrawer.tsx", "No saved snapshots yet."],
    ["components/LeftRail.tsx", "Select a story to see hooks, contacts, and next leads."],
    ["components/LeftRail.tsx", "Manage ${story.name || story.id}"],
  ])("routes %s presentation copy through catalogs", (file, untranslated) => {
    const source = readFileSync(new URL(file, import.meta.url), "utf8");
    expect(source).toContain("useTranslation");
    expect(source).not.toContain(`\"${untranslated}\"`);
  });

  it("keeps locale-specific API presentation in catalogs", () => {
    const source = readFileSync(new URL("api.ts", import.meta.url), "utf8");
    expect(source).not.toContain("i18n.language");
    expect(source).not.toContain("payload.error");
    expect(source).toContain("api_errors:");
  });

  it("resolves story-library time controls from the library namespace", () => {
    const source = readFileSync(new URL("features/story-library/StoryLibraryDrawer.tsx", import.meta.url), "utf8");
    expect(source).toContain('t("library:timeFormat")');
    expect(source).toContain('t("library:timeFormats.system")');
    expect(source).not.toContain('t("timeFormat")');
    expect(source).not.toContain('t("timeFormats.system")');
  });
});
