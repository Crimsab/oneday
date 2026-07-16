import { expect, test, type Page, type Route } from "@playwright/test";
import { readFile } from "node:fs/promises";

const now = "2026-07-11T12:00:00Z";
const story = { id: "story-1", name: "The Glass Archive", description: "A branching test story", genre: "mystery", tone: "focused", language: "en", is_archived: false, updated_at: now };

async function openRail(page: Page) {
  if (!(await page.locator("#story-navigation").isVisible())) {
    await page.getByRole("button", { name: "Library" }).click();
  }
}

async function openStoryLibrary(page: Page) {
  await openRail(page);
  await page.getByRole("button", { name: /active stor(y|ies)/i }).click();
}

function snapshot(turn = 4, branchId = "branch-main") {
  const messages = [
    { id: 1, session_id: "session-1", story_id: story.id, turn: 3, role: "assistant", content: "The archive doors wait in silence.", message_type: "narrative", metadata: {}, created_at: now, branch_id: branchId, source_commit_id: "commit-3" },
    { id: 4, session_id: "session-1", story_id: story.id, turn: 4, role: "user", content: "[Choice 1] Inspect the fractured seal.", message_type: "action", metadata: {}, created_at: now, branch_id: branchId, source_commit_id: "commit-4" },
    { id: 2, session_id: "session-1", story_id: story.id, turn: 4, role: "assistant", content: "Mira studies the fractured seal.", message_type: "narrative", metadata: { provider: "codex", model: "gpt-5.5", latency_ms: 1250, streamed: true, usage: { total_tokens: 321 }, generation: { run_id: "run-2", trace_id: "trace-2", stage: "narrator" }, output: { dialogue_blocks: [{ speaker_id: "npc-mira", speaker: "Mira", role: "Archivist", text: "Choose carefully." }] } }, created_at: now, branch_id: branchId, source_commit_id: "commit-4" },
    { id: 3, session_id: "session-1", story_id: story.id, turn: 4, role: "assistant", content: "An older generation record remains readable.", message_type: "narrative", metadata: { model: "gpt-5.4-mini", latency_ms: 13250, streamed: true, usage: { total_tokens: 10311 } }, created_at: now, branch_id: branchId, source_commit_id: "commit-4" },
  ].filter((message) => message.turn <= turn);
  return {
    server_time: new Date().toISOString(),
    version: { turn, revision: branchId === "branch-main" ? 7 : 8, story_updated_at: now, active_session_id: "session-1", last_message_id: 4, world_updated_at: now, character_updated_at: now, npc_count: 1, npc_updated_at: now, chapter_count: 1, achievement_count: 0, latest_achievement_at: "", save_count: 0, latest_save_at: "", visual_asset_updated_at: "", visual_job_updated_at: "", active_visual_job_count: 0 },
    story,
    character: { id: "hero", name: "Iria", fields: { stats: { resolve: 42 }, condition: "Steady", inventory: [] } },
    world: { id: "world", current_location: "Glass Archive", current_location_id: "loc-archive", current_chapter: 1, current_turn: turn, spatial_locations: [{ id: "loc-archive", name: "Glass Archive", description: "The known archive", discovery_state: "visited" }, { id: "loc-court", name: "Outer Court", description: "A known courtyard", discovery_state: "known" }, { id: "loc-stair", name: "Mirror Stair", description: "A narrow stair lined with cracked mirrors", discovery_state: "known" }, { id: "loc-wharf", name: "Ash Wharf", description: "A loading platform above the black canal", discovery_state: "known" }], spatial_edges: [{ id: "edge-court", from_location_id: "loc-archive", to_location_id: "loc-court", direction: "south", travel_minutes: 5 }, { id: "edge-stair", from_location_id: "loc-court", to_location_id: "loc-stair", direction: "east", travel_minutes: 3 }, { id: "edge-wharf", from_location_id: "loc-stair", to_location_id: "loc-wharf", direction: "down", travel_minutes: 7 }, { id: "edge-return", from_location_id: "loc-wharf", to_location_id: "loc-archive", direction: "canal", travel_minutes: 12 }], world_time: { day: 2, minute_of_day: 780, display_text: "Day 2, 13:00" }, weather: { weather_kind: "clear", description: "Cold clear air" }, known_locations: [], global_events: [], faction_standings: {}, story_hooks: [], world_reactions: [], investigations: [], projects: [], guidance: [], fronts: [], timeline: [], scene_contract: {}, updated_at: now },
    active_session: { id: "session-1", story_id: story.id, started_at: now, ended_at: null, summary: "" },
    choices: [{ id: 1, text: "Inspect the seal", intent: "investigate", risk: "measured", scope: "local", certainty: "uncertain", related_stats: ["resolve"] }],
    messages,
    panels: { chapters: [{ id: 1, chapter_number: 1, title: "The Seal", summary: "Arrival at the archive.", start_turn: 1, end_turn: null, created_at: now, branch_id: branchId, source_commit_id: "commit-1" }], achievements: [], npcs: [{ id: "npc-mira", name: "Mira", fields: { disposition: -20, relationship: {} } }], sessions: [{ id: "session-1", story_id: story.id, started_at: now, ended_at: null, summary: "" }], saves: [] },
  };
}

const timeline = {
  active_branch_id: "branch-main",
  revision: 7,
  branches: [
    { id: "branch-main", story_id: story.id, name: "main", head_commit_id: "commit-4", head_turn: 4, created_at: now, updated_at: now },
    { id: "branch-alt", story_id: story.id, name: "quiet route", fork_commit_id: "commit-3", head_commit_id: "commit-alt", head_turn: 3, created_at: now, updated_at: now },
  ],
  head: { id: "commit-4", branch_id: "branch-main", parent_commit_id: "commit-3", canonical_turn: 4, kind: "turn", message: "Seal", created_at: now },
  commits: [
    { id: "commit-3", branch_id: "branch-main", parent_commit_id: "commit-2", canonical_turn: 3, kind: "turn", message: "Archive doors", created_at: now },
    { id: "commit-4", branch_id: "branch-main", parent_commit_id: "commit-3", canonical_turn: 4, kind: "turn", message: "Seal", created_at: now },
    { id: "commit-alt", branch_id: "branch-alt", parent_commit_id: "commit-3", canonical_turn: 4, kind: "turn", message: "Quiet route", created_at: "2026-07-11T12:00:01Z" },
  ],
};

function visualResponse(canUndo = true, canRedo = false) {
  return {
    profile: { id: "profile-1", story_id: story.id, revision: 3, fingerprint: "profile-fingerprint", branch_id: "branch-main", source_commit_id: "commit-4", world_style_prompt: "Glass and brass", character_style_prompt: "Grounded portrait", negative_prompt: "", palette: "amber", updated_at: now },
    assets: [
      { id: "asset-mira-new", story_id: story.id, kind: "character", subject: "Mira", entity_id: "npc-mira", canonical_entity_id: "npc-mira", canonical_location_id: "", form_id: "form-mira-restored", lineage_key: "npc-mira:form-mira-restored", appearance_fingerprint: "mira-restored", profile_revision_id: "profile-1", canon_status: "canonical", gate_state: "form_changed", gate_reason: "Mira's restored form has not been rendered on this branch.", generation_eligible: true, prompt: "Mira restored portrait", negative_prompt: "", status: "pending", url: "", provider: "", source: "", error: "", turn: 4, branch_id: "branch-main", source_commit_id: "commit-4", selected_version_id: 21, can_undo_selection: canUndo, can_redo_selection: canRedo, inherited: false, updated_at: now },
      { id: "map-background", story_id: story.id, kind: "map_background", subject: "Known world", entity_id: "", canonical_entity_id: "", canonical_location_id: "", form_id: "", lineage_key: "map:branch-main", appearance_fingerprint: "map-v1", profile_revision_id: "profile-1", canon_status: "canonical", gate_state: "established_canonical", gate_reason: "Known map topology", generation_eligible: true, prompt: "Decorative map", negative_prompt: "", status: "ready", url: "/assets/map.png", provider: "mock", source: "", error: "", turn: 4, branch_id: "branch-main", source_commit_id: "commit-4", selected_version_id: null, can_undo_selection: false, can_redo_selection: false, inherited: false, updated_at: now },
      { id: "archive-icon", story_id: story.id, kind: "map_icon", subject: "Glass Archive", entity_id: "", canonical_entity_id: "", canonical_location_id: "loc-archive", form_id: "", lineage_key: "map-icon:loc-archive", appearance_fingerprint: "archive-v1", profile_revision_id: "profile-1", canon_status: "canonical", gate_state: "established_canonical", gate_reason: "Known location", generation_eligible: true, prompt: "Archive icon", negative_prompt: "", status: "ready", url: "/assets/archive.png", provider: "mock", source: "", error: "", turn: 4, branch_id: "branch-main", source_commit_id: "commit-4", selected_version_id: null, can_undo_selection: false, can_redo_selection: false, inherited: false, updated_at: now },
      { id: "court-icon", story_id: story.id, kind: "map_icon", subject: "Outer Court", entity_id: "", canonical_entity_id: "", canonical_location_id: "loc-court", form_id: "", lineage_key: "map-icon:loc-court", appearance_fingerprint: "court-v1", profile_revision_id: "profile-1", canon_status: "canonical", gate_state: "established_canonical", gate_reason: "Known location", generation_eligible: true, prompt: "Court icon", negative_prompt: "", status: "ready", url: "/assets/court.png", provider: "mock", source: "", error: "", turn: 4, branch_id: "branch-main", source_commit_id: "commit-4", selected_version_id: null, can_undo_selection: false, can_redo_selection: false, inherited: false, updated_at: now },
      { id: "stair-icon", story_id: story.id, kind: "map_icon", subject: "Mirror Stair", entity_id: "", canonical_entity_id: "", canonical_location_id: "loc-stair", form_id: "", lineage_key: "map-icon:loc-stair", appearance_fingerprint: "stair-v1", profile_revision_id: "profile-1", canon_status: "canonical", gate_state: "established_canonical", gate_reason: "Known location", generation_eligible: true, prompt: "Stair icon", negative_prompt: "", status: "ready", url: "/assets/stair.png", provider: "mock", source: "", error: "", turn: 4, branch_id: "branch-main", source_commit_id: "commit-4", selected_version_id: null, can_undo_selection: false, can_redo_selection: false, inherited: false, updated_at: now },
      { id: "wharf-icon", story_id: story.id, kind: "map_icon", subject: "Ash Wharf", entity_id: "", canonical_entity_id: "", canonical_location_id: "loc-wharf", form_id: "", lineage_key: "map-icon:loc-wharf", appearance_fingerprint: "wharf-v1", profile_revision_id: "profile-1", canon_status: "canonical", gate_state: "established_canonical", gate_reason: "Known location", generation_eligible: true, prompt: "Wharf icon", negative_prompt: "", status: "ready", url: "/assets/wharf.png", provider: "mock", source: "", error: "", turn: 4, branch_id: "branch-main", source_commit_id: "commit-4", selected_version_id: null, can_undo_selection: false, can_redo_selection: false, inherited: false, updated_at: now },
    ],
    jobs: [],
  };
}

async function json(route: Route, body: unknown, status = 200) {
  await route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
}

function imageProvider(
  id: string,
  displayName: string,
  options: { default?: boolean; configured?: boolean; baseUrl?: string; models?: string[]; modelValidation?: string } = {},
) {
  return {
    id,
    display_name: displayName,
    auth_type: id === "codex-oauth" ? "codex_oauth" : id === "replicate" ? "api_token" : "api_key",
    default: Boolean(options.default),
    configured: Boolean(options.configured),
    api_key_configured: Boolean(options.configured && id !== "codex-oauth"),
    status: options.configured ? "configured" : "not configured",
    base_url: options.baseUrl ?? "https://images.example.test/v1",
    models: options.models ?? [],
    model_validation: options.modelValidation ?? "catalog",
    capabilities: {
      generate: true,
      edit: false,
      sizes: ["1024x1024", "1536x1024"],
      aspect_ratios: ["1:1", "3:2"],
      qualities: ["standard", "high"],
      output_formats: ["png", "webp"],
      supports_transparency: false,
    },
  };
}

function modelSettingsFixture() {
  return {
    config_path: "/test/config.yaml",
    config_revision: "revision-1",
    provider_priority: ["codex"],
    providers: [{ id: "codex", label: "Codex", enabled: true, model: "gpt-5.5", reasoning: "off", supports_model: true, supports_reasoning: true }],
    narrative_models: ["gpt-5.5"],
    utility_models: ["gpt-5.5"],
    repair_models: ["gpt-5.5"],
    image_models: ["gpt-image-2", "gpt-image-1"],
    ascii_models: ["gpt-5.5"],
    embedding_providers: ["auto"],
    image_providers: [
      imageProvider("codex-oauth", "Codex OAuth", { default: true, configured: true, baseUrl: "http://imagegen-bridge:8787", models: ["gpt-image-2", "gpt-image-1.5", "gpt-image-1", "gpt-image-1-mini"], modelValidation: "allowlist" }),
      imageProvider("openai", "OpenAI Platform", { models: ["gpt-image-2", "gpt-image-1.5", "gpt-image-1", "gpt-image-1-mini"], modelValidation: "allowlist" }),
      imageProvider("openai-compatible", "OpenAI-compatible / LiteLLM", { baseUrl: "", modelValidation: "configured" }),
      imageProvider("gemini", "Google Gemini", { baseUrl: "https://generativelanguage.googleapis.com", models: ["gemini-3.1-flash-image", "gemini-3-pro-image"], modelValidation: "allowlist_or_gemini_image_model" }),
      imageProvider("fal", "fal.ai", { models: ["fal-ai/flux/schnell", "fal-ai/nano-banana-2"], modelValidation: "vendor_slug" }),
      imageProvider("replicate", "Replicate", { models: ["black-forest-labs/flux-schnell"], modelValidation: "owner_model_slug" }),
      imageProvider("stability", "Stability AI", { models: ["stable-image-core"], modelValidation: "allowlist" }),
      imageProvider("azure-openai", "Azure OpenAI Images", { baseUrl: "", models: ["gpt-image-2", "gpt-image-1.5", "gpt-image-1", "gpt-image-1-mini"], modelValidation: "deployment_name" }),
    ],
    image_generation: {
      provider: "codex-oauth",
      map_icon_provider: "codex-oauth",
      base_url: "",
      api_key_configured: false,
      model: "gpt-image-2",
      map_icon_model: "gpt-image-1",
      openclaw_bridge_url: "",
      imagegen_bridge_url: "http://imagegen-bridge:8787",
      imagegen_bridge_token_configured: false,
      imagegen_bridge_provider: "codex-responses",
      imagegen_bridge_map_icon_provider: "codex-responses",
      imagegen_bridge_fallbacks: ["codex-app-server:gpt-image-2"],
      imagegen_bridge_fallback_policy: "on_unavailable",
      imagegen_bridge_compatibility: "normalize",
      default_size: "1024x1024",
      location_size: "1536x1024",
      character_size: "1024x1024",
      default_resolution: "",
      location_resolution: "",
      character_resolution: "",
      default_aspect_ratio: "1:1",
      location_aspect_ratio: "3:2",
      character_aspect_ratio: "1:1",
      quality: "standard",
      output_format: "png",
      background: "",
      timeout_seconds: 180,
      auto_generate: false,
      append_negative_prompt: true,
      available: true,
      status: "configured",
    },
    active: { provider: "codex", narrative_model: "gpt-5.5", utility_model: "gpt-5.5", repair_model: "gpt-5.5", repair_fallback_models: [], image_model: "gpt-image-2", ascii_model: "gpt-5.5", embedding_provider: "auto", embedding_model: "text-embedding-3-small", codex_reasoning: "off" },
    tts_status: "disabled",
  };
}

function automaticPatternMiniGame() {
  return { protocol_version: 1, id: "mini-auto-pattern", story_id: story.id, branch_id: "branch-main", turn: 4, seed: 7, definition: { id: "pattern-generic", kind: "pattern", prompt: "Decode the fractured seal by completing its pattern.", difficulty: 50, options: ["8", "9", "10"], rules: { selection_reason: "difficulty fit 50; 4 narrative tag matches; timing-free" } }, runtime: { phase: "active", revision: 1, state: {}, history: [{ action: "start" }] } };
}

async function mockGateway(page: Page, options: { failAction?: boolean; activeMiniGame?: boolean; ttsOff?: boolean; holdAction?: boolean } = {}) {
  let failAction = Boolean(options.failAction);
  let actionRequests = 0;
  let releaseAction = () => {};
  const actionGate = options.holdAction
    ? new Promise<void>((resolve) => { releaseAction = resolve; })
    : null;
  let wizardRequests = 0;
  const wizardBodies: Array<Record<string, unknown>> = [];
  let activeMiniGame: any = options.activeMiniGame ? automaticPatternMiniGame() : null;
  let visualCanUndo = true;
  let visualCanRedo = false;
  let visualPrompt = "Mira restored portrait";
  let visualStatus = "ready";
  let visualSelectedVersion = 21;
  let visualGenerationQueued = false;
  let imageOperations: any[] = [];
  let translationJobs: any[] = [];
  let visualVersions = [{ id: 21, asset_id: "asset-mira-new", story_id: story.id, kind: "character", subject: "Mira", canonical_entity_id: "npc-mira", canonical_location_id: "", form_id: "form-mira-restored", appearance_fingerprint: "mira-restored", profile_revision_id: "profile-1", canon_status: "canonical", url: "/assets/mira.png", prompt: "Mira restored portrait", revised_prompt: "", negative_prompt: "", provider: "mock", turn: 4, branch_id: "branch-main", source_commit_id: "commit-4", created_at: now }];
  const currentVisualResponse = () => {
    const response: any = visualResponse(visualCanUndo, visualCanRedo);
    response.operation_capabilities = [
      { operation: "edit", supported: true, availability: "available", controls: { negative_prompt: false } },
      { operation: "inpaint", supported: true, availability: "available", mask: { required: true, kind: "raster", soft_values: "supported", adherence: "best_effort" }, controls: { negative_prompt: false } },
      { operation: "image_transform", supported: true, availability: "available", controls: { negative_prompt: false } },
      { operation: "variation", supported: false, availability: "unavailable" },
    ];
    response.operations = imageOperations;
    response.assets[0] = {
      ...response.assets[0],
      prompt: visualPrompt,
      status: visualStatus,
      url: visualStatus === "ready" ? (visualSelectedVersion === 22 ? "/assets/mira-new.png" : "/assets/mira.png") : "",
      selected_version_id: visualSelectedVersion,
      updated_at: visualSelectedVersion === 22 ? "2026-07-11T12:01:00Z" : now,
    };
    response.jobs = visualGenerationQueued
      ? [{ id: 91, asset_id: "asset-mira-new", story_id: story.id, canonical_entity_id: "npc-mira", canonical_location_id: "", form_id: "form-mira-restored", appearance_fingerprint: "mira-restored", profile_revision_id: "profile-1", status: "queued", attempts: 0, max_attempts: 3, locked_until: "", error: "", provider: "", started_at: "", finished_at: "", branch_id: "branch-main", source_commit_id: "commit-4", created_at: now, updated_at: now }]
      : [];
    return response;
  };
  let currentTimeline = structuredClone(timeline);
  let audioGenerated = false;
  let ttsSettings = { story_id: story.id, mode: options.ttsOff ? "off" : "all", autoplay: false, default_language_tag: "en", provider_policy: {} };
  let pronunciations: any[] = [];
  await page.route("**/assets/*.png", async (route) => {
    const label = new URL(route.request().url()).pathname.split("/").at(-1)?.slice(0, 1).toUpperCase() || "M";
    await route.fulfill({ status: 200, contentType: "image/svg+xml", body: `<svg xmlns="http://www.w3.org/2000/svg" width="96" height="96"><rect width="96" height="96" fill="white"/><circle cx="48" cy="48" r="27" fill="#24292d"/><text x="48" y="57" text-anchor="middle" font-size="26" fill="#d09a48">${label}</text></svg>` });
  });
  await page.route("**/api/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;
    if (path.endsWith("/events")) {
      return route.fulfill({ status: 200, contentType: "text/event-stream", headers: { "cache-control": "no-cache" }, body: `event: turn\ndata: ${JSON.stringify({ story_id: story.id, status: "event", client_turn: 4, event_type: "narrative.delta", event: { type: "narrative.delta", payload: { text: "A silver line appears." } }, message: "Narrative is streaming.", created_at: now })}\n\n` });
    }
    if (path === "/api/health") return json(route, { status: "ok", stories: 1 });
    if (path === "/api/stories" && request.method() === "GET") return json(route, [story]);
    if (path.endsWith("/translations/jobs/estimate") && request.method() === "POST") return json(route, { total_items: 4, total_characters: 220, cache_hits: 0 });
    if (path.endsWith("/translations/jobs") && request.method() === "GET") return json(route, translationJobs);
    if (path.endsWith("/translations/jobs") && request.method() === "POST") {
      const body = request.postDataJSON();
      const job = { ...body, id: "translation-job-1", story_id: story.id, branch_id: "branch-main", status: "queued", total_items: 4, completed_items: 0, failed_items: 0, cached_items: 0, total_characters: 220, processed_characters: 0, error_code: "", error_summary: "", created_at: now, updated_at: now };
      translationJobs = [job];
      return json(route, job, 201);
    }
    if (path.endsWith("/translations/glossary") && request.method() === "GET") return json(route, []);
    if (path.endsWith("/agency-events")) return json(route, [{ id: 1, story_id: story.id, branch_id: "branch-main", commit_id: "commit-4", canonical_turn: 4, entity_id: "npc-mira", entity_name: "Mira", action: "pursues_goal", summary: "Mira advances an offscreen goal.", created_at: now }]);
    if (path === "/api/contracts/commands") return json(route, []);
    if (path === "/api/tts/voices" || path === "/api/tts/providers") return json(route, {
      providers: [{ id: "cloud", available: true }, { id: "local", available: false, reason: "disabled" }],
      voices: [{ id: "voice-alloy", provider: "cloud", model: "gpt-4o-mini-tts", provider_voice_id: "alloy", display_name: "Alloy", language_tags: ["en"], version: "1", style_family: "neutral", enabled: true }],
    });
    if (path.endsWith("/tts/settings") && request.method() === "GET") return json(route, { settings: ttsSettings });
    if (path.endsWith("/tts/settings") && request.method() === "PUT") {
      const body = request.postDataJSON();
      ttsSettings = { ...ttsSettings, ...body };
      return json(route, { settings: ttsSettings });
    }
    if (path.endsWith("/voice-assignments") && request.method() === "GET") return json(route, { assignments: [] });
    if (path.includes("/voice-assignments/") && request.method() === "PUT") return json(route, { assignment: request.postDataJSON() });
    if (path.endsWith("/pronunciations") && request.method() === "GET") return json(route, { pronunciations });
    if (path.includes("/pronunciations/") && request.method() === "PUT") {
      const entry = request.postDataJSON();
      pronunciations = [...pronunciations.filter((item) => item.id !== entry.id), entry];
      return json(route, { pronunciation: entry });
    }
    if (path.includes("/pronunciations/") && request.method() === "DELETE") {
      const id = decodeURIComponent(path.split("/").at(-1)!);
      pronunciations = pronunciations.filter((item) => item.id !== id);
      return json(route, { pronunciations });
    }
    if (path.endsWith("/audio/cleanup")) return json(route, { cleanup: { dry_run: request.postDataJSON().dry_run, files_scanned: 1, orphan_files: 0, files_removed: 0, invalid_cache_rows: 0, prunable_cache_rows: 0, cache_rows_removed: 0 } });
    if (path.endsWith("/audio/export")) return json(route, { export: { format: "oneday-audio-manifest-v1", filename: "oneday-audio-story-1.json", generated_at: now, story_id: story.id, settings: ttsSettings, providers: [], voices: [], assignments: [], pronunciations, assets: [], jobs: [] } });
    if (/\/messages\/\d+\/audio$/.test(path) && request.method() === "GET") return json(route, { assets: audioGenerated ? [{ id: "audio-2", story_id: story.id, source_message_id: 2, segment_index: 0, segment_kind: "narrator", status: "ready", duration_ms: 300, language_tag: "en" }] : [], jobs: [] });
    if (/\/messages\/\d+\/audio$/.test(path) && request.method() === "POST") {
      audioGenerated = true;
      return json(route, { assets: [{ id: "audio-2", story_id: story.id, source_message_id: 2, segment_index: 0, segment_kind: "narrator", status: "ready", duration_ms: 300, language_tag: "en" }], jobs: [{ id: "job-2", audio_asset_id: "audio-2", status: "succeeded", attempts: 1, max_attempts: 3 }] });
    }
    if (path === "/api/audio/audio-2") return route.fulfill({ status: 200, contentType: "audio/wav", body: Buffer.from("RIFFtest") });
    if (path === "/api/config/models") return json(route, modelSettingsFixture());
    if (path === "/api/story-wizard") {
      wizardRequests += 1;
      const body = request.postDataJSON() as Record<string, unknown> & { action?: string; input?: string };
      wizardBodies.push(body);
      if (body.action === "preset_dark_fantasy" || body.input?.includes("Italian dark fantasy")) {
        return json(route, { wizard: { state: { stage: "review_world" }, phase: "conversation", stage: "review_world", stage_label: "Review world draft", placeholder: "Type how to change the world draft...", message: "World Draft\n\nStory: Bells Under Salt\nGenre: Dark fantasy", actions: [{ key: "accept_world", label: "Accept world" }], definition: { name: "Bells Under Salt", description: "A dangerous pilgrimage through melancholy ruins.", genre: "dark fantasy", tone: "melancholy", language: "Italian", setting: { world_name: "The Salt Marches" }, stats_schema: { has_combat: true } }, last_model: "gpt-5.4-mini", last_latency_ms: 2100 } });
      }
      return json(route, { wizard: { state: { stage: "brief" }, phase: "conversation", stage: "brief", stage_label: "Choose the story brief", placeholder: "Describe the story you want...", message: "Story setup starts with one short brief.", actions: [{ key: "preset_dark_fantasy", label: "Dark fantasy", seed: "Italian dark fantasy with melancholy ruins, dangerous magic, elegant prose, and terse dialogue." }, { key: "preset_cyberpunk", label: "Cyberpunk noir", seed: "Italian cyberpunk noir with sharp dialogue." }, { key: "focus_input", label: "Write my own" }] } });
    }
    if (path.endsWith("/snapshot")) return json(route, snapshot());
    if (path.endsWith("/craft") && request.method() === "POST") {
      const body = request.postDataJSON() as { message: string; history: Array<{ role: string; content: string }> };
      return json(route, {
        crafting: {
          feasible: true,
          narrative: `You assemble ${body.message} without advancing the story turn.`,
          item: { name: "Prism key", description: "A keyed glass tool", effect: "Reveals sealed archive seams", materials: ["glass shard", "brass wire"] },
          alternatives: ["Reinforce it with wax"],
          choices: [{ id: 1, text: "Try a quieter version" }],
          resolved_outcome: { degree: "success" },
        },
        snapshot: snapshot(),
      });
    }
    if (path.endsWith("/visual-assets/asset-mira-new/versions")) return json(route, visualVersions);
    if (path.endsWith("/visual-assets/asset-mira-new") && request.method() === "PUT") {
      visualPrompt = request.postDataJSON().prompt;
      return json(route, currentVisualResponse());
    }
    if (path.endsWith("/visual-assets/asset-mira-new/operations") && request.method() === "POST") {
      const body = request.postDataJSON();
      imageOperations = [{ id: "image-operation-1", asset_id: "asset-mira-new", operation: body.operation, status: "queued", provider: "mock", model: "mock-image", endpoint_id: "/images/edits", source_version_id: body.source_version_id, mask_id: "mask-1", result_version_id: null, branch_id: "branch-main", error_code: "", error_summary: "", created_at: now, updated_at: now }];
      visualGenerationQueued = true;
      visualStatus = "queued";
      return json(route, currentVisualResponse());
    }
    if (path.endsWith("/visual-assets/generate") && request.method() === "POST") {
      visualGenerationQueued = true;
      visualStatus = "queued";
      return json(route, currentVisualResponse());
    }
    if (path.endsWith("/visual-assets/asset-mira-new/selection/undo")) {
      visualCanUndo = false;
      visualCanRedo = true;
      return json(route, currentVisualResponse());
    }
    if (path.endsWith("/visual-assets/asset-mira-new/selection/redo")) {
      visualCanUndo = true;
      visualCanRedo = false;
      return json(route, currentVisualResponse());
    }
    if (path.endsWith("/visual-assets")) {
      if (visualGenerationQueued) {
        visualGenerationQueued = false;
        visualStatus = "ready";
        visualSelectedVersion = 22;
        visualVersions = [{ ...visualVersions[0], id: 22, url: "/assets/mira-new.png", prompt: visualPrompt, created_at: "2026-07-11T12:01:00Z" }, ...visualVersions];
      }
      return json(route, currentVisualResponse());
    }
    if (path.endsWith("/minigames") && request.method() === "GET") return json(route, { instance: activeMiniGame });
    if (path.endsWith("/minigames") && request.method() === "POST") {
      const body = request.postDataJSON() as { definition: { kind: string } };
      const kind = body.definition.kind;
      activeMiniGame = { protocol_version: 1, id: `mini-${kind}`, story_id: story.id, branch_id: "branch-main", turn: 4, seed: 7, definition: { id: `${kind}-generic`, kind, prompt: kind === "pattern" ? "Complete the pattern: 2, 4, 6, ?" : "Resolve the challenge.", difficulty: 50, options: kind === "pattern" ? ["8", "9", "10"] : [], rules: { selection_reason: "player selected; timing-free" } }, runtime: { phase: "active", revision: 1, state: {}, history: [{ action: "start" }] } };
      return json(route, { instance: activeMiniGame });
    }
    if (path.includes("/minigames/") && path.endsWith("/input")) {
      const body = request.postDataJSON() as { input: { action: string; value?: string } };
      if (body.input.action === "submit") activeMiniGame = { ...activeMiniGame, runtime: { ...activeMiniGame.runtime, phase: "resolved", revision: 2, result: { passed: true, detail: `Pattern answer: ${body.input.value} → FULL SUCCESS`, outcome: { version: 1, degree: "full_success", difficulty: 50, seed: 7, roll: 0, total: 0, margin: 0 } } } };
      else activeMiniGame = { ...activeMiniGame, runtime: { ...activeMiniGame.runtime, phase: body.input.action === "pause" ? "paused" : "active", revision: activeMiniGame.runtime.revision + 1 } };
      return json(route, { instance: activeMiniGame });
    }
    if (path.endsWith("/timeline") && request.method() === "GET") return json(route, currentTimeline);
    if (path.endsWith("/timeline") && request.method() === "POST") {
      const body = request.postDataJSON() as { action: string; branch_id?: string; from_commit_id?: string; name?: string };
      if (body.action === "fork" || body.action === "fork_checkout") {
        const branch = { id: `branch-${currentTimeline.branches.length + 1}`, story_id: story.id, name: body.name || "alternate", fork_commit_id: body.from_commit_id || currentTimeline.head.parent_commit_id, head_commit_id: body.from_commit_id || currentTimeline.head.id, head_turn: Math.max(0, currentTimeline.head.canonical_turn - 1), created_at: now, updated_at: now };
        const nextTimeline = { ...currentTimeline, revision: currentTimeline.revision + 1, branches: [...currentTimeline.branches, branch] };
        const forkCommit = currentTimeline.commits.find((commit) => commit.id === branch.head_commit_id);
        currentTimeline = body.action === "fork_checkout"
          ? { ...nextTimeline, active_branch_id: branch.id, head: { ...(forkCommit || currentTimeline.head), branch_id: branch.id } }
          : nextTimeline;
      } else if (body.action === "checkout") {
        const branch = currentTimeline.branches.find((item) => item.id === body.branch_id) || currentTimeline.branches[0];
        currentTimeline = { ...currentTimeline, active_branch_id: branch.id, revision: currentTimeline.revision + 1, head: { ...currentTimeline.head, id: branch.head_commit_id, branch_id: branch.id, canonical_turn: branch.head_turn, parent_commit_id: branch.fork_commit_id || currentTimeline.head.parent_commit_id } };
      }
      const active = currentTimeline.branches.find((branch) => branch.id === currentTimeline.active_branch_id)!;
      return json(route, { timeline: currentTimeline, snapshot: snapshot(active.head_turn, active.id) });
    }
    if (path.endsWith("/history")) return json(route, { items: snapshot().messages, next_cursor: null });
    if (path.endsWith("/chapters")) return json(route, { items: snapshot().panels.chapters, next_cursor: null });
    if (path.endsWith("/messages/2/diagnostics")) return json(route, { run_id: "run-2", trace_id: "trace-2", parent_run_id: "", story_id: story.id, branch_id: "branch-main", source_commit_id: "commit-4", message_id: 2, stage: "narrator", status: "succeeded", prompt_profile: "narrator", prompt_revision: 3, prompt_hash: "sha256:redacted", requested_streaming: true, observed_streaming: true, ttft_ms: 180, duration_ms: 1250, usage: { input_tokens: 200, output_tokens: 121, reasoning_tokens: 40, cached_input_tokens: 0, total_tokens: 321, cost_usd: 0.003 }, error_class: "", created_at: now, finished_at: now, attempts: [{ sequence: 1, provider: "codex", requested_model: "gpt-5.5", resolved_model: "gpt-5.5", requested_streaming: true, observed_streaming: true, status: "succeeded", ttft_ms: 180, duration_ms: 1250, usage: { input_tokens: 200, output_tokens: 121, reasoning_tokens: 40, cached_input_tokens: 0, total_tokens: 321, cost_usd: 0.003 }, retry_reason: "", error_class: "" }] });
    if (path.endsWith("/telemetry/export")) return json(route, { format: "jsonl", filename: "glass-archive-generation-telemetry.jsonl", content: "{\"run_id\":\"run-2\"}\n", count: 1, truncated: false });
    if (path.endsWith("/archive") && request.method() === "POST") return route.fulfill({ status: 200, contentType: "application/zip", headers: { "content-disposition": 'attachment; filename="glass-archive-oneday.zip"' }, body: Buffer.from("PK ONEDAY") });
    if (path.endsWith("/world-template")) return route.fulfill({ status: 200, contentType: "application/json", headers: { "content-disposition": 'attachment; filename="glass-archive-world.oneday.json"' }, body: JSON.stringify({ kind: "oneday-world-template", version: 1, story: { name: story.name } }) });
    if (path.endsWith("/export")) {
      const format = url.searchParams.get("format") || "markdown";
      if (format === "epub") return route.fulfill({ status: 200, contentType: "application/epub+zip", headers: { "content-disposition": 'attachment; filename="glass-archive.epub"' }, body: Buffer.from("PK EPUB") });
      if (format === "replay") return json(route, { format, filename: "glass-archive-replay.json", content: JSON.stringify({ format: "oneday-replay-v1", visual_assets: [], audio_assets: [] }), encoding: "utf-8", content_type: "application/json" });
      return json(route, { format: format === "json" ? "json" : "markdown", filename: "glass-archive-history.md", content: "# The Glass Archive\n\nCanonical export", encoding: "utf-8" });
    }
    if (path.endsWith("/actions") && request.method() === "POST") {
      actionRequests += 1;
      if (failAction) {
        failAction = false;
        return json(route, { error: "simulated rollback" }, 500);
      }
      const body = request.postDataJSON() as { action?: { text?: string } };
      if (body.action?.text?.startsWith("[Challenge Result:")) activeMiniGame = null;
      if (actionGate) await actionGate;
      else await new Promise((resolve) => setTimeout(resolve, 350));
      return json(route, { events: [{ id: "challenge-start", type: "challenge.started", payload: { protocol_version: "challenge.v1" } }, { id: "challenge-end", type: "challenge.resolved", payload: { degree: "success" } }, { id: "commit", type: "turn.committed", payload: {} }], snapshot: snapshot(5) });
    }
    return json(route, {});
  });
  return {
    actionRequests: () => actionRequests,
    releaseAction,
    wizardRequests: () => wizardRequests,
    lastWizardRequest: () => wizardBodies.at(-1),
  };
}

test("submits once, clears optimistically, and renders stream/challenge lifecycle", async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem("oneday-browser-preferences-v2", JSON.stringify({ showGenerationDiagnostics: true })));
  const errors: string[] = [];
  page.on("console", (message) => { if (message.type() === "error") errors.push(message.text()); });
  const requests = await mockGateway(page, { holdAction: true });
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "The Glass Archive" })).toBeVisible();
  await expect(page.getByRole("region", { name: "Challenge host" })).toHaveCount(0);
  await expect(page.getByRole("main").getByText("Mira studies the fractured seal.")).toBeVisible();
  await expect(page.getByLabel("Structured dialogue for turn 4")).toContainText("Choose carefully.");
  await page.locator(".generation-diagnostics summary").click();
  await expect(page.getByText("narrator rev 3")).toBeVisible();
  await expect(page.getByRole("list", { name: "Provider attempts" })).toContainText("TTFT 180 ms");
  await expect(page.getByLabel("Generation debug summary")).toHaveText("gpt-5.4-mini · 13 s · 10,311 tokens · streamed");

  const composer = page.getByPlaceholder("What do you want to try?");
  await composer.fill("Trace the silver fracture");
  const actionRequest = page.waitForRequest((request) => request.url().endsWith("/actions"));
  await page.getByRole("button", { name: "Send action" }).evaluate((button: HTMLButtonElement) => { button.click(); button.click(); });
  await actionRequest;
  const pendingNarrator = page.locator(".pending-narrator");
  await expect(pendingNarrator).toBeVisible();
  await expect(pendingNarrator.locator(".narrative-skeleton")).toBeVisible();
  await expect(page.getByText("Turn progress", { exact: true })).toHaveCount(0);
  await expect(composer).toHaveValue("");
  expect(requests.actionRequests()).toBe(1);
  requests.releaseAction();
  await expect(page.locator(".pending-narrator")).toHaveCount(0);
  expect(errors).toEqual([]);
});

test("creates a persistent AI translation job from the translation center", async ({ page }) => {
  await mockGateway(page);
  await page.goto("/");
  await page.getByRole("button", { name: "Open translation center" }).click();
  const dialog = page.getByRole("dialog", { name: "Translation center" });
  await expect(dialog).toBeVisible();
  await expect(dialog.locator("select")).toHaveCount(0);
  await expect(dialog.getByRole("combobox", { name: "Style" })).toHaveCount(0);
  await expect(dialog.getByRole("combobox", { name: "Target language" }).locator("img")).toBeVisible();
  await expect(dialog.getByRole("combobox", { name: "Target language" })).not.toContainText(/^EN\b/);
  await dialog.getByRole("combobox", { name: "Engine" }).click();
  await page.getByRole("option", { name: "AI" }).click();
  await expect(dialog.getByRole("combobox", { name: "Provider" })).toContainText("Codex");
  await dialog.getByRole("combobox", { name: "Style" }).click();
  await page.getByRole("option", { name: "Literary" }).click();
  await expect(dialog.getByRole("combobox", { name: "Style" })).toContainText("Literary");
  await expect(dialog.getByText(/4 items, 220 characters/)).toBeVisible();
  await dialog.getByRole("button", { name: "Start translation" }).click();
  await expect(dialog.getByText("0/4 items")).toBeVisible();
  const overflow = await dialog.evaluate((element) => ({ width: element.scrollWidth, viewport: element.clientWidth }));
  expect(overflow.width).toBeLessThanOrEqual(overflow.viewport + 1);
});

test("shows optional choice context including used attributes", async ({ page }) => {
  await page.addInitScript(() => {
    localStorage.setItem("oneday-browser-preferences-v2", JSON.stringify({ density: "balanced", fontSize: "base", accent: "amber", showLeftRail: false, showInspector: false, wrapTranscript: true, showChoiceDetails: true }));
  });
  await mockGateway(page);
  await page.goto("/");
  const choices = page.getByRole("region", { name: "Suggested actions" });
  await expect(choices).toContainText("USES resolve");
  await expect(choices).toContainText("RISK measured");
});

test("keeps story actions above the rail and closes them on outside click", async ({ page }) => {
  await mockGateway(page);
  await page.goto("/");
  await openStoryLibrary(page);
  const navigationBorderWidths = await page.locator(".module-nav button").evaluateAll((buttons) => buttons.map((button) => getComputedStyle(button).borderTopWidth));
  expect(new Set(navigationBorderWidths)).toEqual(new Set(["1px"]));
  const createBox = await page.getByRole("button", { name: "New Story" }).boundingBox();
  const searchBox = await page.getByPlaceholder("Filter stories").boundingBox();
  expect(createBox).not.toBeNull();
  expect(searchBox).not.toBeNull();
  const overlaps = createBox!.x < searchBox!.x + searchBox!.width
    && createBox!.x + createBox!.width > searchBox!.x
    && createBox!.y < searchBox!.y + searchBox!.height
    && createBox!.y + createBox!.height > searchBox!.y;
  expect(overlaps).toBe(false);
  const storyRows = page.locator(".story-library-row");
  await expect(storyRows).toHaveCount(1);
  const storyBox = await storyRows.first().boundingBox();
  expect(storyBox?.height).toBeGreaterThanOrEqual(80);
  await page.getByRole("button", { name: "Manage The Glass Archive" }).click();
  const menu = page.getByRole("menu");
  await expect(menu).toBeVisible();
  await expect(menu.getByRole("menuitem", { name: "Edit" })).toBeInViewport();
  await page.getByPlaceholder("Filter stories").click();
  await expect(menu).toHaveCount(0);
});

test("exposes complete story export directly from story management", async ({ page }) => {
  await mockGateway(page);
  await page.goto("/");
  await openStoryLibrary(page);

  const library = page.getByRole("dialog", { name: "Story library" });
  await expect(library.getByRole("button", { name: "Export", exact: true })).toBeVisible();
  await library.getByRole("button", { name: "Export", exact: true }).click();

  const exporter = page.getByRole("dialog", { name: "Export The Glass Archive" });
  await expect(exporter.getByText("Reading copy", { exact: true })).toBeVisible();
  await expect(exporter.getByText("Portable story archive", { exact: true })).toBeVisible();
  await expect(exporter.getByText("World template", { exact: true })).toBeVisible();
  await expect(exporter.locator("select")).toHaveCount(0);
  const [archiveDownload] = await Promise.all([
    page.waitForEvent("download"),
    exporter.getByRole("button", { name: "Download OneDay ZIP" }).click(),
  ]);
  expect(archiveDownload.suggestedFilename()).toBe("glass-archive-oneday.zip");

  await exporter.getByRole("button", { name: "Close" }).click();
  const reopenedLibrary = page.getByRole("dialog", { name: "Story library" });
  await reopenedLibrary.getByRole("button", { name: "Manage The Glass Archive" }).click();
  await page.getByRole("menuitem", { name: "Export" }).click();
  await expect(page.getByRole("dialog", { name: "Export The Glass Archive" })).toBeVisible();
});

test("reviews a story preset before starting structured generation", async ({ page }) => {
  const requests = await mockGateway(page);
  await page.goto("/");
  await openStoryLibrary(page);
  await page.getByRole("button", { name: "New Story" }).click();
  const dialog = page.getByRole("dialog");
  await expect(dialog.getByRole("button", { name: "Dark fantasy" })).toBeVisible();
  await expect(dialog.getByRole("combobox", { name: "Visual style" })).toContainText("Photorealistic");
  expect(requests.wizardRequests()).toBe(1);
  await dialog.getByRole("button", { name: "Dark fantasy" }).click();
  await expect(dialog.getByRole("region", { name: "Confirm story preset" })).toContainText("Nothing has been generated or created yet");
  await expect(dialog.getByLabel("Story brief")).toHaveValue(/Italian dark fantasy/);
  expect(requests.wizardRequests()).toBe(1);
  await dialog.getByRole("button", { name: "Generate draft" }).click();
  await expect(dialog.getByText("Review world draft", { exact: true }).first()).toBeVisible();
  expect(requests.wizardRequests()).toBe(2);
  expect(requests.lastWizardRequest()).toMatchObject({
    world_style_prompt: expect.stringContaining("physically real"),
    character_style_prompt: expect.stringContaining("Photorealistic real-human"),
    negative_prompt: expect.stringContaining("cosplay"),
  });
});

test("sends a custom visual direction with the story draft", async ({ page }) => {
  const requests = await mockGateway(page);
  await page.goto("/");
  await openStoryLibrary(page);
  await page.getByRole("button", { name: "New Story" }).click();
  const dialog = page.getByRole("dialog");

  await dialog.getByRole("combobox", { name: "Visual style" }).click();
  await page.getByRole("option", { name: "Custom prompt" }).click();
  await dialog.getByLabel("World direction").fill("Charcoal diorama environments");
  await dialog.getByLabel("Character direction").fill("Paper-cut portrait silhouettes");
  await dialog.getByLabel("Avoid").fill("Glossy plastic");
  await dialog.getByLabel("Palette").fill("Ash and copper");
  await dialog.getByRole("button", { name: "Dark fantasy" }).click();
  await dialog.getByRole("button", { name: "Generate draft" }).click();

  expect(requests.lastWizardRequest()).toMatchObject({
    world_style_prompt: "Charcoal diorama environments",
    character_style_prompt: "Paper-cut portrait silhouettes",
    negative_prompt: "Glossy plastic",
    palette: "Ash and copper",
  });
});

test("restores a failed draft, checks out a branch, and exposes searchable history/export", async ({ page, browserName }) => {
  await mockGateway(page, { failAction: true });
  await page.goto("/");
  const composer = page.getByPlaceholder("What do you want to try?");
  await composer.fill("Open the forbidden door");
  await page.getByRole("button", { name: "Send action" }).click();
  await expect(composer).toHaveValue("Open the forbidden door");
  await expect(page.getByText("Something went wrong. Try again.")).toBeVisible();

  await composer.fill("/checkout quiet route");
  await page.getByRole("button", { name: "Send action" }).click();
  await expect(page.getByText("Active branch: quiet route.")).toBeVisible();

  await composer.fill("/history");
  await page.getByRole("button", { name: "Send action" }).click();
  const history = page.viewportSize()!.width <= 1240 ? page.getByRole("dialog") : page.locator(".right-inspector");
  await expect(history.getByPlaceholder("Search this branch")).toBeVisible();
  await history.getByPlaceholder("Search this branch").fill("seal");
  await expect(history.getByRole("heading", { name: "Transcript", exact: true })).toBeVisible();
  await expect(history.getByText("Arrival at the archive.")).toBeVisible();
  const messageCard = history.locator(".history-message").first();
  await expect(messageCard.locator(".history-message-header")).toContainText(/You|Narrator/);
  await expect(messageCard.locator(".history-message-turn")).toContainText(/Turn \d+/);
  await expect(messageCard.locator(".history-message-body")).toBeVisible();
  await history.getByText("Export this branch", { exact: true }).click();
  const [download] = await Promise.all([
    page.waitForEvent("download"),
    history.getByRole("button", { name: "Download", exact: true }).click(),
  ]);
  expect(download.suggestedFilename()).toBe("glass-archive-history.md");
  await expect(history.locator(".story-export-workspace select")).toHaveCount(0);
  await history.getByRole("combobox", { name: "Format" }).click();
  await page.getByRole("option", { name: "EPUB" }).click();
  const [epubDownload] = await Promise.all([page.waitForEvent("download"), history.getByRole("button", { name: "Download", exact: true }).click()]);
  expect(epubDownload.suggestedFilename()).toBe("glass-archive.epub");
  await history.getByText("Technical exports", { exact: true }).click();
  const [replayDownload] = await Promise.all([page.waitForEvent("download"), history.getByRole("button", { name: "Export media replay" }).click()]);
  expect(replayDownload.suggestedFilename()).toBe("glass-archive-replay.json");

  const overflow = await page.evaluate(() => ({ width: document.documentElement.scrollWidth, viewport: innerWidth, fontSize: Number.parseFloat(getComputedStyle(document.querySelector("textarea")!).fontSize) }));
  expect(overflow.width).toBeLessThanOrEqual(overflow.viewport + 1);
  if (browserName === "chromium" && page.viewportSize()!.width < 860) expect(overflow.fontSize).toBeGreaterThanOrEqual(16);
});

test("restores a message decision and only exposes branch navigation when alternatives exist", async ({ page }) => {
  await mockGateway(page);
  await page.goto("/");
  const playerPrompt = page.locator("article.transcript-message.user").filter({ hasText: "Inspect the fractured seal." });
  const narratorResponse = page.locator("article.transcript-message.assistant").filter({ hasText: "Mira studies the fractured seal." });
  const narratorOutcomeTail = page.locator("article.transcript-message.assistant").filter({ hasText: "An older generation record remains readable." });
  await expect(playerPrompt.getByRole("button", { name: "Try another choice" })).toBeVisible();
  await expect(playerPrompt.getByLabel("Available story alternatives")).toHaveCount(0);
  await expect(narratorResponse.getByLabel("Available story alternatives")).toHaveCount(0);
  await expect(narratorOutcomeTail.getByLabel("Available story alternatives")).toContainText("1/2");
  await expect(narratorResponse.getByRole("button", { name: "Try another choice" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Previous alternative" })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Next alternative" })).toBeEnabled();
  await page.getByRole("button", { name: "Try another choice", description: "Create a new branch from before turn 4" }).click();
  await expect(page.getByText("Back at the previous decision on Turn 4 alternative 2.")).toBeVisible();
  const restoredNarrator = page.locator("article.transcript-message.assistant").filter({ hasText: "The archive doors wait in silence." });
  await expect(restoredNarrator.getByLabel("Available story alternatives")).toContainText("2 saved");
  await expect(page.getByRole("button", { name: "Next alternative" })).toBeEnabled();
  const checkout = page.waitForResponse((response) => response.url().endsWith("/timeline") && response.request().method() === "POST");
  await page.getByRole("button", { name: "Next alternative" }).click();
  await checkout;
  await expect(page.getByText("Mira studies the fractured seal.")).toBeVisible();
});

test("keeps story branches inline and closes the menu outside or with escape", async ({ page }) => {
  await mockGateway(page);
  await page.goto("/");
  await openRail(page);
  const toggle = page.getByRole("button", { name: /Story branches/ });
  await toggle.click();
  await expect(page.locator("#branch-menu")).toBeVisible();
  await page.locator(".rail-brand").click();
  await expect(page.locator("#branch-menu")).toHaveCount(0);
  await toggle.click();
  await page.keyboard.press("Escape");
  await expect(page.locator("#branch-menu")).toHaveCount(0);
});

test("keeps the collapsed rail controls inside a short desktop viewport", async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 560 });
  await mockGateway(page);
  await page.goto("/");
  await openRail(page);

  await expect(page.locator(".rail-brand strong")).toHaveText("OneDay");
  await page.getByRole("button", { name: "Collapse library" }).click();

  const rail = page.locator("#story-navigation");
  await expect(rail).toHaveClass(/rail-collapsed/);
  const geometry = await rail.evaluate((element) => {
    const bounds = element.getBoundingClientRect();
    const toggle = element.querySelector<HTMLElement>(".rail-collapse-toggle")!.getBoundingClientRect();
    const stories = element.querySelector<HTMLElement>(".rail-stories-button")!.getBoundingClientRect();
    const count = element.querySelector<HTMLElement>(".rail-stories-count")!.getBoundingClientRect();
    const navigation = element.querySelector<HTMLElement>(".module-nav")!;
    return {
      railBottom: bounds.bottom,
      toggleBottom: toggle.bottom,
      toggleHeight: toggle.height,
      navigationHeight: navigation.getBoundingClientRect().height,
      storiesCenterDelta: Math.abs((stories.left + stories.width / 2) - (bounds.left + bounds.width / 2)),
      countInsideRail: count.left >= bounds.left && count.right <= bounds.right,
    };
  });
  expect(geometry.toggleBottom).toBeLessThanOrEqual(geometry.railBottom + 1);
  expect(geometry.toggleHeight).toBeLessThanOrEqual(56);
  expect(geometry.navigationHeight).toBeGreaterThan(100);
  expect(geometry.storiesCenterDelta).toBeLessThanOrEqual(1);
  expect(geometry.countInsideRail).toBe(true);
});

test("keeps modal focus contained and restores it after escape", async ({ page }) => {
  await mockGateway(page);
  await page.goto("/");
  const trigger = page.getByRole("button", { name: "Options" });
  await trigger.focus();
  await trigger.click();
  const dialog = page.getByRole("dialog", { name: "Options" });
  const close = dialog.getByRole("button", { name: "Close" });
  await expect(dialog).toBeVisible();
  await expect(close).toBeFocused();
  await page.keyboard.press("Shift+Tab");
  expect(await dialog.evaluate((element) => element.contains(document.activeElement))).toBe(true);
  await page.keyboard.press("Escape");
  await expect(dialog).toHaveCount(0);
  await expect(trigger).toBeFocused();
});

test("configures catalog-driven image providers without exposing saved secrets", async ({ page }) => {
  await mockGateway(page);
  await page.goto("/");
  await page.getByRole("button", { name: "Options" }).click();
  const dialog = page.getByRole("dialog", { name: "Options" });
  await dialog.getByRole("button", { name: /AI and models/ }).click();

  const providerChoices = dialog.getByRole("radiogroup", { name: "Image provider" });
  await expect(providerChoices.getByRole("radio").first()).toContainText("Codex OAuth");
  await expect(providerChoices.getByRole("radio").first()).toHaveAttribute("aria-checked", "true");
  await expect(dialog.getByText("Uses your Codex subscription through imagegen-bridge. No OPENAI_API_KEY is required.")).toBeVisible();
  await expect(dialog.getByLabel("Bridge token (optional)")).toHaveAttribute("type", "password");
  await expect(dialog.getByText("bridge auth not configured")).toHaveCount(0);
  const imageSettings = dialog.locator(".image-provider-settings");
  const imageOverflow = await imageSettings.evaluate((element) => ({
    scrollWidth: element.scrollWidth,
    clientWidth: element.clientWidth,
  }));
  expect(imageOverflow.scrollWidth).toBeLessThanOrEqual(imageOverflow.clientWidth + 1);

  const layout = page.viewportSize()!.width <= 1240 ? "mobile" : "desktop";
  if (process.env.ONEDAY_QA_SCREENSHOTS) {
    await imageSettings.scrollIntoViewIfNeeded();
    await imageSettings.screenshot({
      path: `/tmp/oneday-settings-final-${layout}-en-codex.png`,
    });
  }

  await providerChoices.getByRole("radio", { name: /Google Gemini/ }).click();
  await expect(dialog.getByText("Connect Google Gemini directly through OneDay’s dedicated adapter.")).toBeVisible();
  const geminiConfig = dialog.getByRole("group", { name: "Image provider: Google Gemini" });
  await expect(geminiConfig.getByLabel("API key")).toHaveAttribute("type", "password");

  await dialog.getByLabel("Map icon provider").selectOption("openai-compatible");
  await dialog.getByLabel("Map icon model").fill("custom-map-model");
  const compatibleConfig = dialog.getByRole("group", { name: "Image provider: OpenAI-compatible / LiteLLM" });
  await compatibleConfig.getByLabel("Provider URL").fill("http://lite.homelab.local/v1");
  await compatibleConfig.getByLabel("Model").fill("custom-map-model");
  if (process.env.ONEDAY_QA_SCREENSHOTS) {
    await compatibleConfig.scrollIntoViewIfNeeded();
    await imageSettings.screenshot({
      path: `/tmp/oneday-settings-final-${layout}-en-dual-provider.png`,
    });
  }
  await geminiConfig.getByLabel("API key").fill("test-gemini-secret");
  await compatibleConfig.getByLabel("API key").fill("test-compatible-secret");
  await dialog.getByLabel("Auto-generate visuals").check();

  const saveRequest = page.waitForRequest(
    (request) => request.url().endsWith("/api/config/models") && request.method() === "PUT",
  );
  await dialog.getByRole("button", { name: "Save model routing" }).click();
  const payload = await (await saveRequest).postDataJSON();
  expect(payload.image_generation.provider_configs).toEqual([
    expect.objectContaining({ id: "openai-compatible", base_url: "http://lite.homelab.local/v1", api_key: "test-compatible-secret", models: ["custom-map-model"] }),
    expect.objectContaining({ id: "gemini", api_key: "test-gemini-secret" }),
  ]);
  expect(payload.image_generation.provider_configs).not.toEqual(
    expect.arrayContaining([expect.objectContaining({ id: "codex-oauth" })]),
  );

  await page.evaluate(() => localStorage.setItem("oneday-browser-preferences-v2", JSON.stringify({ locale: "it" })));
  await page.reload();
  await page.getByRole("button", { name: "Opzioni" }).click();
  const italianDialog = page.getByRole("dialog", { name: "Opzioni" });
  await italianDialog.getByRole("button", { name: /IA e modelli/ }).click();
  await expect(italianDialog.getByText("Usa l’abbonamento Codex tramite imagegen-bridge. Non richiede OPENAI_API_KEY.")).toBeVisible();
  await expect(italianDialog.getByLabel("Token del bridge (facoltativo)")).toHaveAttribute("type", "password");
  if (process.env.ONEDAY_QA_SCREENSHOTS) {
    await italianDialog.locator(".image-provider-settings").scrollIntoViewIfNeeded();
    await italianDialog.locator(".image-provider-settings").screenshot({
      path: `/tmp/oneday-settings-final-${layout}-it-codex.png`,
    });
  }
});

test("does not render spoken audio controls while story speech is off", async ({ page }) => {
  await mockGateway(page, { ttsOff: true });
  const settingsLoaded = page.waitForResponse((response) => response.url().endsWith("/tts/settings"));
  await page.goto("/");
  await settingsLoaded;
  await expect(page.locator(".transcript")).toHaveAttribute("data-speech-mode", "off");
  await expect(page.getByRole("region", { name: "Spoken audio" })).toHaveCount(0);
});

test("shows canonical visual lineage and branch-local selection controls", async ({ page }) => {
  const errors: string[] = [];
  page.on("console", (message) => { if (message.type() === "error") errors.push(message.text()); });
  page.on("pageerror", (error) => errors.push(error.message));
  await mockGateway(page);
  const visualsLoaded = page.waitForResponse((response) => response.url().endsWith("/visual-assets"));
  await page.goto("/");
  await visualsLoaded;
  await expect(page.getByRole("main").getByText("Mira studies the fractured seal.")).toBeVisible();
  await page.getByRole("button", { name: "Options" }).click();

  const dialog = page.getByRole("dialog");
  const search = dialog.getByPlaceholder("Search options");
  await expect(search).toBeVisible();
  await search.fill("known location icons");
  await expect(dialog.getByRole("button", { name: /Map art/ })).toBeVisible();
  await dialog.getByRole("button", { name: /Map art/ }).click();
  await expect(dialog.getByRole("heading", { name: "Visuals and map" })).toBeVisible();
  expect(errors).toEqual([]);
  await expect(dialog.getByText("New canonical form")).toBeVisible();
  await expect(dialog.getByText("Mira's restored form has not been rendered on this branch.")).toBeVisible();
  await expect(dialog.getByText(/Profile rev 3.*form form-mira-restored.*current branch/)).toBeVisible();
  await expect(dialog.getByRole("button", { name: "Undo selection" })).toBeEnabled();
  const undoResponse = page.waitForResponse((response) => response.url().endsWith("/visual-assets/asset-mira-new/selection/undo"));
  await dialog.getByRole("button", { name: "Undo selection" }).click();
  await undoResponse;
  await expect(page.getByText("Visual selection undone.")).toBeVisible();
  await expect(dialog.getByRole("button", { name: "Redo selection" })).toBeEnabled();
  expect(errors).toEqual([]);
});

test("keeps asset prompt edits stable and reveals completed image versions", async ({ page }) => {
  await mockGateway(page);
  await page.goto("/");
  await page.getByRole("button", { name: "Options" }).click();
  const dialog = page.getByRole("dialog");
  await dialog.getByPlaceholder("Search options").fill("known location icons");
  await dialog.getByRole("button", { name: /Map art/ }).click();

  const prompt = dialog.getByLabel("Asset prompt");
  await prompt.fill("Mira beneath rain-dark archive glass");
  const reload = page.waitForResponse((response) => response.url().endsWith("/visual-assets"));
  await dialog.getByRole("button", { name: "Reload assets" }).click();
  await reload;
  await expect(prompt).toHaveValue("Mira beneath rain-dark archive glass");

  const queued = page.waitForResponse((response) => response.url().endsWith("/visual-assets/generate"));
  await dialog.getByRole("button", { name: "Regenerate", exact: true }).click();
  await queued;
  await expect(prompt).toHaveValue("Mira beneath rain-dark archive glass");
  await expect(dialog.getByRole("button", { name: "Generating…" })).toBeDisabled();

  const completed = page.waitForResponse((response) => response.url().endsWith("/visual-assets"));
  await dialog.getByRole("button", { name: "Reload assets" }).click();
  await completed;
  await expect(dialog.getByText("2 / 2 · selected")).toBeVisible();
  await dialog.getByRole("button", { name: "Older →" }).click();
  await expect(dialog.getByText("1 / 2 · preview")).toBeVisible();
  await dialog.getByRole("button", { name: "← Newer" }).click();
  await expect(dialog.getByText("2 / 2 · selected")).toBeVisible();
});

test("opens a codex image directly in its editable visual asset workspace", async ({ page }) => {
  await mockGateway(page);
  await page.goto("/");
  await openRail(page);
  await page.getByRole("button", { name: "Codex" }).click();
  const moduleSurface = page.viewportSize()!.width <= 1240 ? page.getByRole("dialog") : page.locator(".right-inspector");
  await moduleSurface.getByRole("button", { name: "Open image for Mira" }).click();
  const dialog = page.getByRole("dialog");
  await expect(dialog.getByRole("heading", { name: "Options" })).toBeVisible();
  await expect(dialog.getByLabel("Asset prompt")).toHaveValue("Mira restored portrait");
  await expect(dialog.getByText("Mira", { exact: true }).first()).toBeVisible();
});

test("paints a full-resolution mask and submits inpainting without fallback", async ({ page }) => {
  await mockGateway(page);
  await page.goto("/");
  await page.getByRole("button", { name: "Options" }).click();
  const dialog = page.getByRole("dialog");
  await dialog.getByPlaceholder("Search options").fill("known location icons");
  await dialog.getByRole("button", { name: /Map art/ }).click();

  await expect(dialog.getByRole("radio", { name: /Directed edit/ })).toBeVisible();
  await expect(dialog.getByRole("radio", { name: /Paint an area/ })).toBeVisible();
  await expect(dialog.getByRole("radio", { name: /Transform/ })).toBeVisible();
  await expect(dialog.getByRole("radio", { name: /Variation/ })).toHaveCount(0);
  await dialog.getByRole("radio", { name: /Paint an area/ }).click();

  const canvas = dialog.getByRole("application", { name: "Paint the image area that may change" });
  await expect(canvas).toBeVisible();
  await canvas.scrollIntoViewIfNeeded();
  const box = await canvas.boundingBox();
  expect(box).not.toBeNull();
  await page.mouse.move(box!.x + box!.width * 0.35, box!.y + box!.height * 0.45);
  await page.mouse.down();
  await page.mouse.move(box!.x + box!.width * 0.58, box!.y + box!.height * 0.55, { steps: 8 });
  await page.mouse.up();

  await expect(dialog.getByRole("button", { name: "Undo stroke" })).toBeEnabled();
  await dialog.getByRole("button", { name: "Undo stroke" }).click();
  await expect(dialog.getByRole("button", { name: "Redo stroke" })).toBeEnabled();
  await dialog.getByRole("button", { name: "Redo stroke" }).click();

  const operationRequest = page.waitForRequest((request) => request.url().endsWith("/visual-assets/asset-mira-new/operations"));
  await dialog.getByRole("button", { name: "Create inpainted version" }).click();
  const payload = (await operationRequest).postDataJSON();
  expect(payload).toMatchObject({
    operation: "inpaint",
    source_version_id: 21,
    prompt: "Mira restored portrait",
    fallback: { mode: "forbid" },
  });
  expect(payload.idempotency_key).toEqual(expect.any(String));
  expect(payload.mask_png_base64).toMatch(/^[A-Za-z0-9+/]+=*$/);
  await expect(page.getByText("The image operation was queued as a new version.")).toBeVisible();
  await expect(dialog.getByText("Queued", { exact: true }).last()).toBeVisible();
  await expect(dialog.getByText("mock · mock-image")).toBeVisible();
});

test("uses the dedicated inventory-aware crafting conversation and separates achievements", async ({ page }) => {
  await mockGateway(page);
  await page.goto("/");
  await openRail(page);
  await page.getByRole("button", { name: "Craft" }).click();
  const compact = page.viewportSize()!.width <= 1240;
  const craftSurface = compact ? page.getByRole("dialog") : page.locator(".right-inspector");
  await expect(craftSurface.getByText("Dedicated AI workbench")).toBeVisible();
  const craftRequest = page.waitForRequest((request) => request.url().endsWith("/craft"));
  await craftSurface.getByPlaceholder("What do you want to craft or improvise?").fill("a prism key");
  await craftSurface.getByRole("button", { name: "Evaluate" }).click();
  const request = await craftRequest;
  expect(request.postDataJSON()).toMatchObject({ message: "a prism key", history: [] });
  await expect(craftSurface.getByText("You assemble a prism key without advancing the story turn.")).toBeVisible();
  await expect(craftSurface.getByText("Prism key", { exact: true })).toBeVisible();

  if (compact) await craftSurface.getByRole("button", { name: "Close" }).click();
  await openRail(page);
  await page.getByRole("button", { name: "Achievements" }).click();
  const achievementSurface = compact ? page.getByRole("dialog") : page.locator(".right-inspector");
  await expect(achievementSurface.getByRole("heading", { name: "Achievements" })).toBeVisible();
  await expect(achievementSurface.getByText("Achievements are permanent milestones recorded by the engine.")).toBeVisible();
  await expect(achievementSurface.getByText("Saved snapshots", { exact: true })).toHaveCount(0);
});

test("plays a timing-free minigame through the shared browser host", async ({ page }) => {
  const errors: string[] = [];
  page.on("console", (message) => { if (message.type() === "error") errors.push(message.text()); });
  page.on("pageerror", (error) => errors.push(error.message));
  const requests = await mockGateway(page, { activeMiniGame: true });
  await page.goto("/");
  const host = page.getByRole("region", { name: "Challenge host" });
  await expect(host).toBeVisible();
  await expect(host.getByRole("button", { name: "Auto-fit" })).toHaveCount(0);
  await expect(host.getByRole("button", { name: "Courtroom" })).toHaveCount(0);
  await expect(host.getByText("Decode the fractured seal by completing its pattern.")).toBeVisible();
  await expect(host.getByText(/Selected because:.*narrative tag matches.*timing-free/)).toBeVisible();
  await host.getByLabel("Pattern answer").click();
  await page.getByRole("option", { name: "8", exact: true }).click();
  const continuation = page.waitForRequest((request) => request.url().endsWith("/actions") && request.postData()?.includes("[Challenge Result:"));
  await host.getByRole("button", { name: "Resolve challenge" }).click();
  await continuation;
  await expect(host).toHaveCount(0);
  expect(requests.actionRequests()).toBe(1);
  expect(errors).toEqual([]);
});

test("renders only canonical known map topology and bounded agency events", async ({ page }) => {
  await mockGateway(page);
  await page.goto("/");
  await page.getByPlaceholder("What do you want to try?").fill("/map");
  await page.getByRole("button", { name: "Send action" }).click();
  await expect(page.locator(".agency-feed:visible").filter({ hasText: "Mira advances an offscreen goal." })).toBeVisible();
  if (page.viewportSize()!.width > 1240) {
    const compactMap = page.locator(".right-inspector .canonical-map");
    await compactMap.scrollIntoViewIfNeeded();
    const toolbarIcons = compactMap.locator(".canonical-map-toolbar button svg");
    await expect(toolbarIcons).toHaveCount(4);
    expect(await toolbarIcons.evaluateAll((icons) => icons.every((icon) => {
      const box = icon.getBoundingClientRect();
      return box.width === 15 && box.height === 15 && getComputedStyle(icon).stroke !== "none";
    }))).toBe(true);

    const inspector = page.locator(".inspector-body");
    const scrollBeforeZoom = await inspector.evaluate((node) => node.scrollTop);
    const stageBox = await compactMap.locator(".canonical-map-stage").boundingBox();
    expect(stageBox).not.toBeNull();
    await page.mouse.move(stageBox!.x + stageBox!.width * 0.5, stageBox!.y + stageBox!.height * 0.5);
    await page.mouse.wheel(0, -140);
    await expect(compactMap.getByText("114%", { exact: true })).toBeVisible();
    await expect.poll(() => inspector.evaluate((node) => node.scrollTop)).toBe(scrollBeforeZoom);

    await page.getByRole("button", { name: "Open Map in a larger view" }).click();
  }
  const mapWorkspace = page.getByRole("dialog");
  const mapShell = mapWorkspace.locator(".canonical-map");
  const map = mapShell.locator('svg[role="img"]');
  await expect(map).toHaveAttribute("aria-label", "Interactive map of World with 4 known places and 4 known routes");
  await expect(map).toBeVisible();
  await expect(map.getByText("Glass Archive", { exact: true })).toBeVisible();
  await expect(map.getByText("Outer Court", { exact: true })).toBeVisible();
  await expect(map.getByText("Mirror Stair", { exact: true })).toBeVisible();
  await expect(map.getByText("Ash Wharf", { exact: true })).toBeVisible();
  await expect(mapShell).toHaveClass(/illustrated/);
  await expect(mapShell.locator(".canonical-map-stage > img.canonical-map-art")).toHaveAttribute("src", "/assets/map.png");
  await expect(map.locator("image[clip-path]")).toHaveCount(4);
  const clipPathIds = await page.locator(".canonical-map clipPath[id]").evaluateAll((nodes) => nodes.map((node) => node.id));
  expect(new Set(clipPathIds).size).toBe(clipPathIds.length);
  const expandedClipTargetsExist = await map.locator("image[clip-path]").evaluateAll((nodes) => nodes.every((node) => {
    const reference = node.getAttribute("clip-path")?.match(/^url\(#(.+)\)$/)?.[1];
    return Boolean(reference && node.ownerSVGElement?.querySelector(`clipPath[id="${CSS.escape(reference)}"]`));
  }));
  expect(expandedClipTargetsExist).toBe(true);
  await mapShell.getByRole("button", { name: "Zoom in" }).click();
  await expect(mapShell.getByText("120%", { exact: true })).toBeVisible();
  await map.getByRole("button", { name: "Outer Court" }).click();
  await expect(mapShell.locator(".canonical-map-selection")).toContainText("A known courtyard");
  const beforeDrag = await map.locator(".canonical-map-viewport").getAttribute("transform");
  const box = await map.boundingBox();
  expect(box).not.toBeNull();
  await page.mouse.move(box!.x + box!.width * 0.55, box!.y + box!.height * 0.55);
  await page.mouse.down();
  await page.mouse.move(box!.x + box!.width * 0.68, box!.y + box!.height * 0.64, { steps: 4 });
  await page.mouse.up();
  await expect.poll(() => map.locator(".canonical-map-viewport").getAttribute("transform")).not.toBe(beforeDrag);
});

test("drills through canonical region and sub-location map scopes", async ({ page }) => {
  await mockGateway(page);
  const hierarchical = snapshot() as any;
  hierarchical.world.current_location = "Dock 7";
  hierarchical.world.current_location_id = "dock";
  hierarchical.world.spatial_regions = [
    { id: "vharrow", name: "Vharrow", kind: "macroregion", parent_region_id: "" },
    { id: "port", name: "Port District", kind: "district", parent_region_id: "vharrow" },
  ];
  hierarchical.world.spatial_locations = [
    { id: "dock", name: "Dock 7", kind: "site", region_id: "port", parent_location_id: "", description: "Cargo piers and warehouses" },
    { id: "pump", name: "Pump House", kind: "landmark", region_id: "port", parent_location_id: "", description: "An old pumping station" },
    { id: "lane", name: "Access Lane", kind: "subzone", region_id: "port", parent_location_id: "dock", description: "A narrow service lane" },
  ];
  hierarchical.world.spatial_edges = [
    { id: "dock-pump", from_location_id: "dock", to_location_id: "pump", direction: "east", travel_minutes: 8, travel_mode: "walk", bidirectional: true },
  ];
  await page.route("**/api/stories/story-1/snapshot", (route) => json(route, hierarchical));
  await page.goto("/");
  await page.getByPlaceholder("What do you want to try?").fill("/map");
  await page.getByRole("button", { name: "Send action" }).click();
  if (page.viewportSize()!.width > 1240) {
    await page.getByRole("button", { name: "Open Map in a larger view" }).click();
  }
  await expect(page.getByRole("dialog")).toBeVisible();
  const mapShell = page.getByRole("dialog").locator(".canonical-map");
  await expect(mapShell.locator(".canonical-map-breadcrumbs")).toContainText("WorldVharrowPort District");
  const map = mapShell.locator('svg[role="img"]');
  await expect(map).toHaveAttribute("aria-label", "Interactive map of Port District with 2 known places and 1 known route");
  await map.getByRole("button", { name: /Dock 7/ }).dblclick();
  await expect(mapShell.locator(".canonical-map-breadcrumbs")).toContainText("WorldVharrowPort DistrictDock 7");
  await expect(mapShell.locator('svg[role="img"]')).toHaveAttribute("aria-label", "Interactive map of Dock 7 with 1 known place and 0 known routes");
  await expect(mapShell.getByRole("button", { name: "Access Lane, subzone" })).toBeVisible();
  await mapShell.getByRole("button", { name: "Port District", exact: true }).click();
  await expect(mapShell.locator('svg[role="img"]')).toHaveAttribute("aria-label", "Interactive map of Port District with 2 known places and 1 known route");
});

test("generates committed audio and exposes per-story and per-character voice controls", async ({ page }) => {
  const errors: string[] = [];
  page.on("console", (message) => { if (message.type() === "error") errors.push(message.text()); });
  page.on("pageerror", (error) => errors.push(error.message));
  await mockGateway(page);
  await page.goto("/");
  const message = page.locator("article.transcript-message").filter({ hasText: "Mira studies the fractured seal." });
  await expect(page.locator(".transcript")).toHaveAttribute("data-speech-mode", "all");
  await message.getByRole("button", { name: "Load spoken audio" }).click();
  await expect(message.getByText("Spoken audio")).toBeVisible();
  await message.getByRole("button", { name: "Generate" }).click();
  await expect(message.locator("audio")).toHaveCount(1);

  await page.getByRole("button", { name: "Options" }).click();
  const dialog = page.getByRole("dialog");
  await dialog.getByRole("button", { name: /Spoken audio/ }).click();
  await expect(dialog.getByRole("heading", { name: "Spoken audio" })).toBeVisible();
  await expect(dialog.getByText("cloud: available")).toBeVisible();
  await expect(dialog.getByText("local: disabled")).toBeVisible();
  await expect(dialog.getByText("Narrator", { exact: true })).toBeVisible();
  await expect(dialog.locator(".voice-assignment-row").filter({ hasText: "Mira" }).getByText("Mira", { exact: true })).toBeVisible();
  await dialog.getByLabel("Speech mode").click();
  await page.getByRole("option", { name: "Narrator only", exact: true }).click();
  await dialog.getByRole("button", { name: "Save audio settings" }).click();
  await expect(dialog.getByRole("button", { name: "Save audio settings" })).toBeEnabled();
  await dialog.getByLabel("Written text").fill("Mira");
  await dialog.getByLabel("Spoken form").fill("Mee-ra");
  await dialog.getByRole("button", { name: "Add pronunciation" }).click();
  await expect(dialog.getByLabel("Pronunciation entries")).toContainText("Mira");
  await dialog.getByRole("button", { name: "Audit cache" }).click();
  await expect(dialog.getByText("Audit: 1 audio files, 0 orphaned, 0 invalid and 0 expired cache rows.")).toBeVisible();
  const [audioDownload] = await Promise.all([
    page.waitForEvent("download"),
    dialog.getByRole("button", { name: "Export audio manifest" }).click(),
  ]);
  expect(audioDownload.suggestedFilename()).toBe("oneday-audio-story-1.json");
  const overflow = await page.evaluate(() => ({ width: document.documentElement.scrollWidth, viewport: innerWidth }));
  expect(overflow.width).toBeLessThanOrEqual(overflow.viewport + 1);
  expect(errors).toEqual([]);
});

test("personalizes scrollbar and fonts across reading, interface, and portals", async ({ page, isMobile }) => {
  const fontBytes = await readFile(new URL("../node_modules/@fontsource-variable/ibm-plex-sans/files/ibm-plex-sans-latin-wght-normal.woff2", import.meta.url));
  await page.route("https://fonts.example.test/qa-reader.woff2", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "font/woff2",
      headers: { "access-control-allow-origin": "*", "cache-control": "no-store" },
      body: fontBytes,
    });
  });
  await mockGateway(page);
  await page.goto("/");
  await openRail(page);
  await page.getByRole("button", { name: "Options" }).click();

  for (const group of ["Personalization", "Experience", "System"]) {
    const heading = page.getByText(group, { exact: true });
    await expect(heading).toBeAttached();
    if (isMobile) await expect(heading).toBeHidden();
    else await expect(heading).toBeVisible();
  }
  await expect(page.locator(".settings-sidebar button.active")).toHaveCSS("border-top-width", "1px");
  await expect(page.locator(".settings-sidebar button.active")).toHaveCSS("border-left-width", "1px");

  const themeBefore = await page.evaluate(() => ({
    accent: getComputedStyle(document.documentElement).getPropertyValue("--accent").trim(),
    scrollbar: getComputedStyle(document.documentElement).scrollbarColor,
  }));
  await page.getByRole("button", { name: /Accent/ }).click();
  await page.getByLabel("Hex value").fill("#4f9cff");
  await expect.poll(() => page.evaluate(() => getComputedStyle(document.documentElement).getPropertyValue("--accent").trim())).toBe(themeBefore.accent);
  await page.getByRole("button", { name: "Apply" }).click();
  await expect.poll(() => page.evaluate(() => getComputedStyle(document.documentElement).getPropertyValue("--accent").trim())).toBe("#4f9cff");
  expect(await page.evaluate(() => getComputedStyle(document.documentElement).scrollbarColor)).not.toBe(themeBefore.scrollbar);

  await page.getByRole("button", { name: /Gameplay/ }).click();
  await page.getByRole("button", { name: /Typography/ }).click();
  await page.getByRole("button", { name: "From URL" }).click();
  const onlineDialog = page.getByRole("dialog", { name: "Add an online font" });
  await expect(onlineDialog).toBeVisible();
  const dialogBox = await onlineDialog.boundingBox();
  if (!dialogBox) throw new Error("online font dialog has no bounding box");
  await page.mouse.click(Math.max(1, dialogBox.x - 8), Math.max(1, dialogBox.y - 8));
  await expect(onlineDialog).toBeHidden();

  await page.getByRole("button", { name: "From URL" }).click();
  await page.getByLabel("Direct font URL").fill("https://fonts.example.test/qa-reader.woff2");
  await page.getByLabel("Library name").fill("QA Online Font");
  await page.getByRole("button", { name: "Download and save" }).click();
  await expect(page.getByRole("option", { name: /QA Online Font/ })).toBeVisible();

  const selectedFamily = await page.evaluate(() => JSON.parse(localStorage.getItem("oneday-browser-preferences-v2") || "{}").readingFontFamily as string);
  const assistant = page.locator(".transcript-message.assistant").first();
  await expect.poll(() => assistant.evaluate((node) => getComputedStyle(node).fontFamily)).toContain(selectedFamily);
  await expect.poll(() => page.evaluate(() => getComputedStyle(document.body).fontFamily)).not.toContain(selectedFamily);

  await page.locator(".font-target-switcher").getByRole("button", { name: /Interface/ }).click();
  await page.getByRole("option", { name: /QA Online Font/ }).click();
  await expect.poll(() => page.evaluate(() => getComputedStyle(document.body).fontFamily)).toContain(selectedFamily);
  await expect.poll(() => assistant.evaluate((node) => getComputedStyle(node).fontFamily)).toContain(selectedFamily);

  const leftRailFontBefore = await page.locator(".left-rail").evaluate((node) => getComputedStyle(node).fontSize);
  await page.locator("label.font-size-control", { hasText: "Interface size" }).locator('input[type="range"]').fill("125");
  await expect.poll(() => page.locator(".left-rail").evaluate((node) => getComputedStyle(node).fontSize)).not.toBe(leftRailFontBefore);
  await expect.poll(() => page.evaluate(() => getComputedStyle(document.documentElement).fontSize)).toBe("20px");

  await page.locator(".font-target-switcher").getByRole("button", { name: /Story text/ }).click();
  await page.locator("label.font-size-control", { hasText: "Reading size" }).locator('input[type="range"]').fill("23");
  await expect.poll(() => assistant.evaluate((node) => getComputedStyle(node).fontSize)).toBe("23px");
  expect(await page.evaluate(() => JSON.parse(localStorage.getItem("oneday-browser-preferences-v2") || "{}").interfaceFontScale)).toBe(125);

  await page.getByRole("button", { name: "Edit QA Online Font" }).click();
  await page.getByLabel("Library name").fill("QA Updated Font");
  await page.getByRole("button", { name: "Download update" }).click();
  await expect(page.getByRole("option", { name: /QA Updated Font/ })).toBeVisible();
  await page.getByRole("button", { name: "Delete QA Updated Font" }).click();
  await expect(page.getByRole("option", { name: /QA Updated Font/ })).toHaveCount(0);
  expect(await page.evaluate(() => {
    const preferences = JSON.parse(localStorage.getItem("oneday-browser-preferences-v2") || "{}");
    return [preferences.interfaceFontSource, preferences.readingFontSource];
  })).toEqual(["bundled", "bundled"]);

  await page.getByRole("button", { name: /Advanced/ }).click();
  await expect(page.getByText("Support bundle", { exact: true })).toBeVisible();
  await page.getByText("Recent technical log", { exact: true }).click();
  await expect(page.locator(".support-log-list")).toBeVisible();
  await expect(page.locator(".generation-diagnostics, .generation-diagnostics-inline")).toHaveCount(0);
  await page.getByText("Show diagnostics in messages", { exact: true }).click();
  await expect(page.locator(".generation-diagnostics, .generation-diagnostics-inline")).not.toHaveCount(0);

  await page.getByRole("button", { name: /Gameplay/ }).click();
  await page.locator("label.minigame-preference", { hasText: "Deduction" }).locator('input[type="checkbox"]').uncheck();
  await page.locator(".options-overlay").getByRole("button", { name: "Close" }).click();
  const hideLibrary = page.getByRole("button", { name: "Hide library" });
  if (await hideLibrary.isVisible()) await hideLibrary.click();
  const policyRequest = page.waitForRequest((request) => request.url().endsWith("/actions") && request.method() === "POST");
  await page.getByPlaceholder("What do you want to try?").fill("Test an allowed challenge fallback");
  await page.getByRole("button", { name: "Send action" }).click();
  const policyBody = await (await policyRequest).postDataJSON() as { capabilities: { excluded_minigames: string[] } };
  expect(policyBody.capabilities.excluded_minigames).toContain("deduction");
});

test("switches and persists interface locale without changing story or speech language", async ({ page }) => {
  await mockGateway(page);
  await page.goto("/");
  await page.getByRole("button", { name: "Options" }).click();
  const language = page.getByRole("combobox", { name: "Interface language" });
  await language.click();
  await page.getByRole("option", { name: "Italiano" }).click();

  await expect(page.getByRole("heading", { name: "Aspetto" })).toBeVisible();
  await expect(page.getByText("Cambia i controlli e i messaggi di OneDay.")).toBeVisible();
  await expect(page.getByText("Mira studies the fractured seal.")).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("lang", "it");
  expect(await page.evaluate(() => JSON.parse(localStorage.getItem("oneday-browser-preferences-v2") || "{}").locale)).toBe("it");

  await page.reload();
  await expect(page.getByRole("button", { name: "Opzioni" })).toBeVisible();
  await expect(page.getByText("Mira studies the fractured seal.")).toBeVisible();
  await page.getByRole("button", { name: "Opzioni" }).click();
  await page.getByRole("button", { name: /Audio parlato/ }).click();
  await expect(page.getByPlaceholder("en-US")).toHaveValue("en");
});

test("localizes fresh Italian onboarding from stable wizard keys", async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem("oneday-browser-preferences-v2", JSON.stringify({ locale: "it" })));
  await mockGateway(page);
  await page.goto("/?overlay=new-story");
  await expect(page.getByText("Scegli la traccia della storia")).toBeVisible();
  await expect(page.getByRole("button", { name: /Fantasy oscuro/ })).toBeVisible();
  await expect(page.locator(".story-wizard-message").getByText("La creazione della storia inizia con una breve traccia.")).toBeVisible();
});
