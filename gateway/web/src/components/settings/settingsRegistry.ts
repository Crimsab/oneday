export type SettingsSectionId = "general" | "gameplay" | "audio" | "visuals" | "models" | "advanced";

export interface SettingsCategory {
  id: SettingsSectionId;
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

export const settingsCategories: SettingsCategory[] = [
  { id: "general", title: "General", description: "Reading, layout, and interface preferences." },
  { id: "gameplay", title: "Gameplay", description: "Challenge behavior and accessibility policy." },
  { id: "audio", title: "Spoken audio", description: "Speech, voices, language, and pronunciation." },
  { id: "visuals", title: "Visuals and map", description: "Image direction, generated assets, and map art." },
  { id: "models", title: "AI and models", description: "Provider routing and model configuration." },
  { id: "advanced", title: "Advanced", description: "Runtime transport, capabilities, and diagnostics." },
];

export const settingsSearchEntries: SettingsSearchEntry[] = [
  entry("interface-language", "general", "Interface language", "Change controls and interface messages without changing story or audio language.", "locale italian english controls messages"),
  entry("density", "general", "Density", "Change spacing and information density.", "compact balanced comfortable spacing"),
  entry("font-size", "general", "Font size", "Change the transcript and interface text scale.", "text reading accessibility large small"),
  entry("typography", "general", "Typography", "Choose bundled, system, or imported fonts and customize reading style.", "font family system imported upload search preview weight italic color"),
  entry("accent", "general", "Accent", "Choose the interface accent color.", "amber green blue rose theme colour color"),
  entry("stories-sidebar", "general", "Stories sidebar", "Show or hide the stories and modules rail.", "left rail navigation"),
  entry("inspector", "general", "Inspector panel", "Show or hide the canonical inspector.", "right rail panel"),
  entry("transcript-wrap", "general", "Transcript wrap", "Wrap long narrative lines.", "reading prose line width"),
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
  entry("provider-order", "models", "Provider order", "Set AI provider priority and enablement.", "routing fallback codex"),
  entry("narrative-model", "models", "Narrative model", "Choose the primary story model.", "ai narrator"),
  entry("utility-model", "models", "Utility model", "Choose the model for supporting tasks.", "ai helper"),
  entry("repair-model", "models", "Repair model", "Configure structured-output repair and fallbacks.", "retry json"),
  entry("image-model", "models", "Image generation model", "Configure provider, model, sizes, and output format.", "imagegen codex responses openai litellm external openclaw"),
  entry("embedding-model", "models", "Embedding model", "Configure RAG embedding provider and model.", "vector memory rag"),
  entry("runtime-status", "advanced", "Runtime status", "Inspect transport, capabilities, and active theme.", "sse gateway turn diagnostics"),
  entry("configuration-revision", "advanced", "Configuration revision", "Inspect and reload the active model configuration.", "config version refresh"),
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
