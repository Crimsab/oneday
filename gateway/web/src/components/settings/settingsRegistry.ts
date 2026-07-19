export type SettingsSectionId = "appearance" | "typography" | "gameplay" | "audio" | "visuals" | "preferences" | "operator";
export type SettingsScope = "player" | "operator";
export type SettingsNavigationGroupId = "player" | "operator";

export interface SettingsCategory {
  id: SettingsSectionId;
  scope: SettingsScope;
  title: string;
  description: string;
}

export interface SettingsSearchEntry {
  id: string;
  section: SettingsSectionId;
  label: string;
  description: string;
  keywords: string[];
}

export interface SettingsNavigationGroup {
  id: SettingsNavigationGroupId;
  sections: SettingsSectionId[];
}

export const settingsCategories: SettingsCategory[] = [
  { id: "appearance", scope: "player", title: "Appearance", description: "Interface color, density, language, and workspace layout." },
  { id: "typography", scope: "player", title: "Typography", description: "Fonts, reading scale, style, color, and local library." },
  { id: "gameplay", scope: "player", title: "Gameplay", description: "Challenge behavior and accessibility policy." },
  { id: "audio", scope: "player", title: "Spoken audio", description: "Speech, voices, language, and pronunciation." },
  { id: "visuals", scope: "player", title: "Visuals and map", description: "Story art direction, generated assets, and map art." },
  { id: "preferences", scope: "player", title: "Preferences and portability", description: "Message details, local preferences, themes, and reset." },
  { id: "operator", scope: "operator", title: "Operator configuration", description: "Protected provider, endpoint, readiness, and support configuration." },
];

export const settingsNavigationGroups: SettingsNavigationGroup[] = [
  { id: "player", sections: ["appearance", "typography", "gameplay", "audio", "visuals", "preferences"] },
  { id: "operator", sections: ["operator"] },
];

export const settingsSearchEntries: SettingsSearchEntry[] = [
  entry("interface-language", "appearance", "Interface language", "Change controls and interface messages without changing story or audio language.", "locale italian english controls messages"),
  entry("density", "appearance", "Density", "Change spacing and information density.", "compact balanced comfortable spacing"),
  entry("font-size", "appearance", "Interface scale", "Change the scale of controls and interface text.", "interface accessibility large small"),
  entry("accent", "appearance", "Accent", "Choose the interface accent color.", "amber green blue rose theme colour color scrollbar"),
  entry("stories-sidebar", "appearance", "Stories sidebar", "Show or hide the stories and modules rail.", "left rail navigation"),
  entry("inspector", "appearance", "Inspector panel", "Show or hide the canonical inspector.", "right rail panel"),
  entry("transcript-wrap", "appearance", "Transcript wrap", "Wrap long narrative lines.", "reading prose line width"),
  entry("typography", "typography", "Typography", "Choose bundled, system, local, or online fonts and customize reading style.", "font family system imported online url download upload search preview weight italic color"),
  entry("automatic-challenges", "gameplay", "Automatic challenges", "NPC situations trigger challenges without a player chooser.", "minigame automatic npc challenge host"),
  entry("choice-details", "gameplay", "Choice context", "Show used attributes, risk, certainty, scope, and outcome hints below choices.", "choices tags requirements attributes stats risk outcomes metadata"),
  entry("timing-free", "gameplay", "Timing-free selection", "Prefer accessible challenges that do not require reflex timing.", "accessibility cooldown quick time reflex"),
  entry("challenge-cooldown", "gameplay", "Challenge cooldown", "Avoid repeating the same challenge family too soon.", "repetition variety selector"),
  entry("speech-mode", "audio", "Speech mode", "Choose narration, dialogue, both, or off.", "tts spoken voice narrator dialogue"),
  entry("autoplay", "audio", "Audio autoplay", "Play newly generated committed audio automatically.", "tts playback"),
  entry("language", "audio", "Default language", "Set the speech language tag.", "locale pronunciation tts"),
  entry("voice-assignments", "audio", "Voice assignments", "Assign voices to narrator, protagonist, and NPCs.", "character speaker registry"),
  entry("pronunciation", "audio", "Pronunciation lexicon", "Override how names and terms are spoken.", "dictionary alphabet ipa"),
  entry("audio-maintenance", "audio", "Audio export and cleanup", "Export the manifest or preview safe cache cleanup.", "retry jobs cache"),
  entry("visual-profile", "visuals", "Visual direction", "Set the story visual style, palette, and negative prompt.", "image profile style bible"),
  entry("map-art", "visuals", "Map art", "Generate a thematic background and location symbols.", "known location imagegen icons svg"),
  entry("asset-generation", "visuals", "Asset generation", "Generate eligible character and location imagery.", "image jobs portraits locations"),
  entry("asset-versions", "visuals", "Asset versions", "Inspect, select, regenerate, undo, and redo visual versions.", "history prompt canonical"),
  entry("visual-cleanup", "visuals", "Visual cleanup", "Preview and remove unreferenced generated files.", "image maintenance files"),
  entry("generation-diagnostics", "preferences", "Message diagnostics", "Choose whether to show redacted generation details below messages.", "trace telemetry provider debugging"),
  entry("preferences-portability", "preferences", "Preference portability", "Export, import, or reset browser preferences.", "json backup restore defaults"),
  entry("theme-portability", "preferences", "Theme portability", "Export or import a personal theme and optional locally stored fonts.", "theme zip font browser local"),
  entry("provider-order", "operator", "Provider routing", "Set AI provider priority and enablement.", "routing fallback codex"),
  entry("narrative-model", "operator", "Narrative model", "Choose the primary story model.", "ai narrator"),
  entry("utility-model", "operator", "Utility model", "Choose the model for supporting tasks.", "ai helper"),
  entry("repair-model", "operator", "Repair model", "Configure structured-output repair and fallbacks.", "retry json"),
  entry("image-model", "operator", "Image generation model", "Configure provider, model, sizes, endpoint, and write-only credentials.", "imagegen endpoint api key credential secret codex responses openai litellm"),
  entry("embedding-model", "operator", "Embedding model", "Configure RAG embedding provider and model.", "vector memory rag"),
  entry("runtime-status", "operator", "Runtime readiness", "Inspect transport, capabilities, and active configuration.", "sse gateway turn diagnostics readiness"),
  entry("configuration-revision", "operator", "Configuration revision", "Inspect and reload the active model configuration.", "config version refresh"),
  entry("support-bundle", "operator", "Support bundle", "Create a redacted technical report for support.", "diagnostics logs telemetry redacted issue"),
];

export function searchSettings(query: string): SettingsSearchEntry[] {
  const terms = query.toLowerCase().trim().split(/\s+/).filter(Boolean);
  if (terms.length === 0) return [];
  return settingsSearchEntries.filter((item) => {
    const haystack = [item.label, item.description, ...item.keywords].join(" ").toLowerCase();
    return terms.every((term) => haystack.includes(term));
  });
}

function entry(id: string, section: SettingsSectionId, label: string, description: string, keywords: string): SettingsSearchEntry {
  return { id, section, label, description, keywords: keywords.split(" ") };
}
