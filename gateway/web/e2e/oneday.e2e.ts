import { expect, test, type Page, type Route } from "@playwright/test";

const now = "2026-07-11T12:00:00Z";
const story = { id: "story-1", name: "The Glass Archive", description: "A branching test story", genre: "mystery", tone: "focused", language: "en", is_archived: false, updated_at: now };

function snapshot(turn = 4, branchId = "branch-main") {
  const messages = [
    { id: 1, session_id: "session-1", story_id: story.id, turn: 3, role: "assistant", content: "The archive doors wait in silence.", message_type: "narrative", metadata: {}, created_at: now, branch_id: branchId, source_commit_id: "commit-3" },
    { id: 2, session_id: "session-1", story_id: story.id, turn: 4, role: "assistant", content: "Mira studies the fractured seal.", message_type: "narrative", metadata: { provider: "codex", model: "gpt-5.5", latency_ms: 1250, streamed: true, usage: { total_tokens: 321 }, generation: { run_id: "run-2", trace_id: "trace-2", stage: "narrator" }, output: { dialogue_blocks: [{ speaker_id: "npc-mira", speaker: "Mira", role: "Archivist", text: "Choose carefully." }] } }, created_at: now, branch_id: branchId, source_commit_id: "commit-4" },
    { id: 3, session_id: "session-1", story_id: story.id, turn: 4, role: "assistant", content: "An older generation record remains readable.", message_type: "narrative", metadata: { model: "gpt-5.4-mini", latency_ms: 13250, streamed: true, usage: { total_tokens: 10311 } }, created_at: now, branch_id: branchId, source_commit_id: "commit-4" },
  ];
  return {
    server_time: new Date().toISOString(),
    version: { turn, revision: branchId === "branch-main" ? 7 : 8, story_updated_at: now, active_session_id: "session-1", last_message_id: 3, world_updated_at: now, character_updated_at: now, npc_count: 1, npc_updated_at: now, chapter_count: 1, achievement_count: 0, latest_achievement_at: "", save_count: 0, latest_save_at: "", visual_asset_updated_at: "", visual_job_updated_at: "", active_visual_job_count: 0 },
    story,
    character: { id: "hero", name: "Iria", fields: { stats: { resolve: 42 }, condition: "Steady", inventory: [] } },
    world: { id: "world", current_location: "Glass Archive", current_location_id: "loc-archive", current_chapter: 1, current_turn: turn, spatial_locations: [{ id: "loc-archive", name: "Glass Archive", description: "The known archive", discovery_state: "visited" }, { id: "loc-court", name: "Outer Court", description: "A known courtyard", discovery_state: "known" }], spatial_edges: [{ id: "edge-court", from_location_id: "loc-archive", to_location_id: "loc-court", direction: "south", travel_minutes: 5 }], world_time: { day: 2, minute_of_day: 780, display_text: "Day 2, 13:00" }, weather: { weather_kind: "clear", description: "Cold clear air" }, known_locations: [], global_events: [], faction_standings: {}, story_hooks: [], world_reactions: [], investigations: [], projects: [], guidance: [], fronts: [], timeline: [], scene_contract: {}, updated_at: now },
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
  ],
};

function visualResponse(canUndo = true, canRedo = false) {
  return {
    profile: { id: "profile-1", story_id: story.id, revision: 3, fingerprint: "profile-fingerprint", branch_id: "branch-main", source_commit_id: "commit-4", world_style_prompt: "Glass and brass", character_style_prompt: "Grounded portrait", negative_prompt: "", palette: "amber", updated_at: now },
    assets: [
      { id: "asset-mira-new", story_id: story.id, kind: "character", subject: "Mira", entity_id: "npc-mira", canonical_entity_id: "npc-mira", canonical_location_id: "", form_id: "form-mira-restored", lineage_key: "npc-mira:form-mira-restored", appearance_fingerprint: "mira-restored", profile_revision_id: "profile-1", canon_status: "canonical", gate_state: "form_changed", gate_reason: "Mira's restored form has not been rendered on this branch.", generation_eligible: true, prompt: "Mira restored portrait", negative_prompt: "", status: "pending", url: "", provider: "", source: "", error: "", turn: 4, branch_id: "branch-main", source_commit_id: "commit-4", selected_version_id: 21, can_undo_selection: canUndo, can_redo_selection: canRedo, inherited: false, updated_at: now },
      { id: "map-background", story_id: story.id, kind: "map_background", subject: "Known world", entity_id: "", canonical_entity_id: "", canonical_location_id: "", form_id: "", lineage_key: "map:branch-main", appearance_fingerprint: "map-v1", profile_revision_id: "profile-1", canon_status: "canonical", gate_state: "established_canonical", gate_reason: "Known map topology", generation_eligible: true, prompt: "Decorative map", negative_prompt: "", status: "ready", url: "/assets/map.png", provider: "mock", source: "", error: "", turn: 4, branch_id: "branch-main", source_commit_id: "commit-4", selected_version_id: null, can_undo_selection: false, can_redo_selection: false, inherited: false, updated_at: now },
      { id: "archive-icon", story_id: story.id, kind: "map_icon", subject: "Glass Archive", entity_id: "", canonical_entity_id: "", canonical_location_id: "loc-archive", form_id: "", lineage_key: "map-icon:loc-archive", appearance_fingerprint: "archive-v1", profile_revision_id: "profile-1", canon_status: "canonical", gate_state: "established_canonical", gate_reason: "Known location", generation_eligible: true, prompt: "Archive icon", negative_prompt: "", status: "ready", url: "/assets/archive.png", provider: "mock", source: "", error: "", turn: 4, branch_id: "branch-main", source_commit_id: "commit-4", selected_version_id: null, can_undo_selection: false, can_redo_selection: false, inherited: false, updated_at: now },
    ],
    jobs: [],
  };
}

async function json(route: Route, body: unknown, status = 200) {
  await route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
}

function automaticPatternMiniGame() {
  return { protocol_version: 1, id: "mini-auto-pattern", story_id: story.id, branch_id: "branch-main", turn: 4, seed: 7, definition: { id: "pattern-generic", kind: "pattern", prompt: "Decode the fractured seal by completing its pattern.", difficulty: 50, options: ["8", "9", "10"], rules: { selection_reason: "difficulty fit 50; 4 narrative tag matches; timing-free" } }, runtime: { phase: "active", revision: 1, state: {}, history: [{ action: "start" }] } };
}

async function mockGateway(page: Page, options: { failAction?: boolean; activeMiniGame?: boolean; ttsOff?: boolean } = {}) {
  let failAction = Boolean(options.failAction);
  let actionRequests = 0;
  let wizardRequests = 0;
  let activeMiniGame: any = options.activeMiniGame ? automaticPatternMiniGame() : null;
  let visualCanUndo = true;
  let visualCanRedo = false;
  let currentTimeline = structuredClone(timeline);
  let audioGenerated = false;
  let ttsSettings = { story_id: story.id, mode: options.ttsOff ? "off" : "all", autoplay: false, default_language_tag: "en", provider_policy: {} };
  let pronunciations: any[] = [];
  await page.route("**/api/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;
    if (path.endsWith("/events")) {
      return route.fulfill({ status: 200, contentType: "text/event-stream", headers: { "cache-control": "no-cache" }, body: `event: turn\ndata: ${JSON.stringify({ story_id: story.id, status: "event", client_turn: 4, event_type: "narrative.delta", event: { type: "narrative.delta", payload: { text: "A silver line appears." } }, message: "Narrative is streaming.", created_at: now })}\n\n` });
    }
    if (path === "/api/health") return json(route, { status: "ok", stories: 1 });
    if (path === "/api/stories" && request.method() === "GET") return json(route, [story]);
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
    if (path.endsWith("/audio/cleanup")) return json(route, { cleanup: { dry_run: request.postDataJSON().dry_run, files_scanned: 1, orphan_files: 0, files_removed: 0, invalid_cache_rows: 0 } });
    if (path.endsWith("/audio/export")) return json(route, { export: { format: "oneday-audio-manifest-v1", filename: "oneday-audio-story-1.json", generated_at: now, story_id: story.id, settings: ttsSettings, providers: [], voices: [], assignments: [], pronunciations, assets: [], jobs: [] } });
    if (/\/messages\/\d+\/audio$/.test(path) && request.method() === "GET") return json(route, { assets: audioGenerated ? [{ id: "audio-2", story_id: story.id, source_message_id: 2, segment_index: 0, segment_kind: "narrator", status: "ready", duration_ms: 300, language_tag: "en" }] : [], jobs: [] });
    if (/\/messages\/\d+\/audio$/.test(path) && request.method() === "POST") {
      audioGenerated = true;
      return json(route, { assets: [{ id: "audio-2", story_id: story.id, source_message_id: 2, segment_index: 0, segment_kind: "narrator", status: "ready", duration_ms: 300, language_tag: "en" }], jobs: [{ id: "job-2", audio_asset_id: "audio-2", status: "succeeded", attempts: 1, max_attempts: 3 }] });
    }
    if (path === "/api/audio/audio-2") return route.fulfill({ status: 200, contentType: "audio/wav", body: Buffer.from("RIFFtest") });
    if (path === "/api/config/models") return json(route, {
      config_path: "/test/config.yaml", config_revision: "revision-1", provider_priority: ["codex"],
      providers: [{ id: "codex", label: "Codex", enabled: true, model: "gpt-5.5", reasoning: "off", supports_model: true, supports_reasoning: true }],
      narrative_models: ["gpt-5.5"], utility_models: ["gpt-5.5"], repair_models: ["gpt-5.5"], image_models: ["gpt-image-1"], ascii_models: ["gpt-5.5"], embedding_providers: ["auto"],
      image_generation: { provider: "openclaw-bridge", base_url: "", api_key_configured: false, model: "gpt-image-1", openclaw_bridge_url: "http://image.test/generate", default_size: "1024x1024", location_size: "1536x1024", character_size: "1024x1024", default_resolution: "", location_resolution: "", character_resolution: "", default_aspect_ratio: "", location_aspect_ratio: "", character_aspect_ratio: "", quality: "", output_format: "png", background: "", timeout_seconds: 180, auto_generate: false, append_negative_prompt: true, available: true, status: "configured" },
      active: { provider: "codex", narrative_model: "gpt-5.5", utility_model: "gpt-5.5", repair_model: "gpt-5.5", repair_fallback_models: [], image_model: "gpt-image-1", ascii_model: "gpt-5.5", embedding_provider: "auto", embedding_model: "text-embedding-3-small", codex_reasoning: "off" },
      tts_status: "planned",
    });
    if (path === "/api/story-wizard") {
      wizardRequests += 1;
      const body = request.postDataJSON() as { action?: string; input?: string };
      if (body.action === "preset_dark_fantasy" || body.input?.includes("Italian dark fantasy")) {
        return json(route, { wizard: { state: { stage: "review_world" }, phase: "conversation", stage: "review_world", stage_label: "Review world draft", placeholder: "Type how to change the world draft...", message: "World Draft\n\nStory: Bells Under Salt\nGenre: Dark fantasy", actions: [{ key: "accept_world", label: "Accept world" }], definition: { name: "Bells Under Salt", description: "A dangerous pilgrimage through melancholy ruins.", genre: "dark fantasy", tone: "melancholy", language: "Italian", setting: { world_name: "The Salt Marches" }, stats_schema: { has_combat: true } }, last_model: "gpt-5.4-mini", last_latency_ms: 2100 } });
      }
      return json(route, { wizard: { state: { stage: "brief" }, phase: "conversation", stage: "brief", stage_label: "Choose the story brief", placeholder: "Describe the story you want...", message: "Story setup starts with one short brief.", actions: [{ key: "preset_dark_fantasy", label: "Dark fantasy", seed: "Italian dark fantasy with melancholy ruins, dangerous magic, elegant prose, and terse dialogue." }, { key: "preset_cyberpunk", label: "Cyberpunk noir", seed: "Italian cyberpunk noir with sharp dialogue." }, { key: "focus_input", label: "Write my own" }] } });
    }
    if (path.endsWith("/snapshot")) return json(route, snapshot());
    if (path.endsWith("/visual-assets/asset-mira-new/versions")) return json(route, [{ id: 21, asset_id: "asset-mira-new", story_id: story.id, kind: "character", subject: "Mira", canonical_entity_id: "npc-mira", canonical_location_id: "", form_id: "form-mira-restored", appearance_fingerprint: "mira-restored", profile_revision_id: "profile-1", canon_status: "canonical", url: "/assets/mira.png", prompt: "Mira restored portrait", revised_prompt: "", negative_prompt: "", provider: "mock", turn: 4, branch_id: "branch-main", source_commit_id: "commit-4", created_at: now }]);
    if (path.endsWith("/visual-assets/asset-mira-new/selection/undo")) {
      visualCanUndo = false;
      visualCanRedo = true;
      return json(route, visualResponse(visualCanUndo, visualCanRedo));
    }
    if (path.endsWith("/visual-assets/asset-mira-new/selection/redo")) {
      visualCanUndo = true;
      visualCanRedo = false;
      return json(route, visualResponse(visualCanUndo, visualCanRedo));
    }
    if (path.endsWith("/visual-assets")) return json(route, visualResponse(visualCanUndo, visualCanRedo));
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
      if (body.action === "fork") {
        const branch = { id: `branch-${currentTimeline.branches.length + 1}`, story_id: story.id, name: body.name || "alternate", fork_commit_id: body.from_commit_id || currentTimeline.head.parent_commit_id, head_commit_id: body.from_commit_id || currentTimeline.head.id, head_turn: Math.max(0, currentTimeline.head.canonical_turn - 1), created_at: now, updated_at: now };
        currentTimeline = { ...currentTimeline, revision: currentTimeline.revision + 1, branches: [...currentTimeline.branches, branch] };
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
    if (path.endsWith("/export")) {
      const format = url.searchParams.get("format") || "markdown";
      if (format === "epub") return json(route, { format, filename: "glass-archive.epub", content: btoa("PK EPUB"), encoding: "base64", content_type: "application/epub+zip" });
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
      await new Promise((resolve) => setTimeout(resolve, 350));
      return json(route, { events: [{ id: "challenge-start", type: "challenge.started", payload: { protocol_version: "challenge.v1" } }, { id: "challenge-end", type: "challenge.resolved", payload: { degree: "success" } }, { id: "commit", type: "turn.committed", payload: {} }], snapshot: snapshot(5) });
    }
    return json(route, {});
  });
  return { actionRequests: () => actionRequests, wizardRequests: () => wizardRequests };
}

test("submits once, clears optimistically, and renders stream/challenge lifecycle", async ({ page }) => {
  const errors: string[] = [];
  page.on("console", (message) => { if (message.type() === "error") errors.push(message.text()); });
  const requests = await mockGateway(page);
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
  await expect(page.locator(".pending-narrator")).toBeVisible();
  await expect(page.locator(".narrative-skeleton")).toBeVisible();
  await expect(page.getByText("Turn progress", { exact: true })).toHaveCount(0);
  await expect(composer).toHaveValue("");
  expect(requests.actionRequests()).toBe(1);
  await expect(page.locator(".pending-narrator")).toHaveCount(0);
  expect(errors).toEqual([]);
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
  await page.getByRole("button", { name: /library/i }).click();
  const createBox = await page.getByRole("button", { name: "New Story" }).boundingBox();
  const searchBox = await page.getByPlaceholder("Filter stories").boundingBox();
  expect(createBox).not.toBeNull();
  expect(searchBox).not.toBeNull();
  expect(searchBox!.y - (createBox!.y + createBox!.height)).toBeGreaterThanOrEqual(6);
  const storyRows = page.locator(".story-row");
  await expect(storyRows).toHaveCount(1);
  const storyBox = await storyRows.first().boundingBox();
  expect(storyBox?.height).toBeGreaterThanOrEqual(80);
  await page.getByRole("button", { name: "Manage The Glass Archive" }).click();
  const menu = page.getByRole("menu");
  await expect(menu).toBeVisible();
  await expect(menu.getByRole("menuitem", { name: "Edit" })).toBeInViewport();
  await page.getByRole("button", { name: "New Story" }).click({ position: { x: 4, y: 4 } });
  await expect(menu).toHaveCount(0);
});

test("reviews a story preset before starting structured generation", async ({ page }) => {
  const requests = await mockGateway(page);
  await page.goto("/");
  await page.getByRole("button", { name: /library/i }).click();
  await page.getByRole("button", { name: "New Story" }).click();
  const dialog = page.getByRole("dialog");
  await expect(dialog.getByRole("button", { name: "Dark fantasy" })).toBeVisible();
  expect(requests.wizardRequests()).toBe(1);
  await dialog.getByRole("button", { name: "Dark fantasy" }).click();
  await expect(dialog.getByRole("region", { name: "Confirm story preset" })).toContainText("Nothing has been generated or created yet");
  await expect(dialog.getByLabel("Story brief")).toHaveValue(/Italian dark fantasy/);
  expect(requests.wizardRequests()).toBe(1);
  await dialog.getByRole("button", { name: "Generate draft" }).click();
  await expect(dialog.getByText("Review world draft", { exact: true }).first()).toBeVisible();
  expect(requests.wizardRequests()).toBe(2);
});

test("restores a failed draft, checks out a branch, and exposes searchable history/export", async ({ page, browserName }) => {
  await mockGateway(page, { failAction: true });
  await page.goto("/");
  const composer = page.getByPlaceholder("What do you want to try?");
  await composer.fill("Open the forbidden door");
  await page.getByRole("button", { name: "Send action" }).click();
  await expect(composer).toHaveValue("Open the forbidden door");
  await expect(page.getByText("simulated rollback")).toBeVisible();

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
  await history.getByText("Export this branch", { exact: true }).click();
  const [download] = await Promise.all([
    page.waitForEvent("download"),
    history.getByRole("button", { name: "Export Markdown" }).click(),
  ]);
  expect(download.suggestedFilename()).toBe("glass-archive-history.md");
  const [epubDownload] = await Promise.all([page.waitForEvent("download"), history.getByRole("button", { name: "Export EPUB" }).click()]);
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
  await expect(page.getByRole("button", { name: "Previous story branch" })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Next story branch" })).toBeEnabled();
  await page.getByRole("button", { name: "Try another path from here", description: "Create a new branch from before turn 4" }).click();
  await expect(page.getByText("Back at the previous decision on Turn 4 alternative 2.")).toBeVisible();
});

test("does not render spoken audio controls while story speech is off", async ({ page }) => {
  await mockGateway(page, { ttsOff: true });
  const settingsLoaded = page.waitForResponse((response) => response.url().endsWith("/tts/settings"));
  await page.goto("/");
  await settingsLoaded;
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
  const map = page.locator('.canonical-map:visible svg[role="img"]');
  await expect(map).toHaveAttribute("aria-label", "Canonical map with 2 known locations and 1 known routes");
  await expect(map).toBeVisible();
  await expect(map.getByText("Glass Archive", { exact: true })).toBeVisible();
  await expect(map.getByText("Outer Court", { exact: true })).toBeVisible();
  await expect(page.locator(".canonical-map:visible")).toHaveClass(/illustrated/);
  await expect(page.locator('.canonical-map:visible > img.canonical-map-art')).toHaveAttribute("src", "/assets/map.png");
  await expect(map.locator('image[href="/assets/archive.png"]')).toHaveCount(1);
});

test("generates committed audio and exposes per-story and per-character voice controls", async ({ page }) => {
  const errors: string[] = [];
  page.on("console", (message) => { if (message.type() === "error") errors.push(message.text()); });
  page.on("pageerror", (error) => errors.push(error.message));
  await mockGateway(page);
  await page.goto("/");
  const message = page.locator("article.transcript-message").filter({ hasText: "Mira studies the fractured seal." });
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
  await expect(dialog.getByText("Audit: 1 audio files, 0 orphaned, 0 invalid cache rows.")).toBeVisible();
  const [audioDownload] = await Promise.all([
    page.waitForEvent("download"),
    dialog.getByRole("button", { name: "Export audio manifest" }).click(),
  ]);
  expect(audioDownload.suggestedFilename()).toBe("oneday-audio-story-1.json");
  const overflow = await page.evaluate(() => ({ width: document.documentElement.scrollWidth, viewport: innerWidth }));
  expect(overflow.width).toBeLessThanOrEqual(overflow.viewport + 1);
  expect(errors).toEqual([]);
});
