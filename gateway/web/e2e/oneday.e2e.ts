import { expect, test, type Page, type Route } from "@playwright/test";

const now = "2026-07-11T12:00:00Z";
const story = { id: "story-1", name: "The Glass Archive", description: "A branching test story", genre: "mystery", tone: "focused", language: "en", is_archived: false, updated_at: now };

function snapshot(turn = 4, branchId = "branch-main") {
  const messages = [
    { id: 1, session_id: "session-1", story_id: story.id, turn: 3, role: "assistant", content: "The archive doors wait in silence.", message_type: "narrative", metadata: {}, created_at: now, branch_id: branchId, source_commit_id: "commit-3" },
    { id: 2, session_id: "session-1", story_id: story.id, turn: 4, role: "assistant", content: "Mira studies the fractured seal.", message_type: "narrative", metadata: { provider: "codex", model: "gpt-5.5", latency_ms: 1250, streamed: true, usage: { total_tokens: 321 }, generation: { run_id: "run-2", trace_id: "trace-2", stage: "narrator" }, output: { dialogue_blocks: [{ speaker_id: "npc-mira", speaker: "Mira", role: "Archivist", text: "Choose carefully." }] } }, created_at: now, branch_id: branchId, source_commit_id: "commit-4" },
  ];
  return {
    server_time: new Date().toISOString(),
    version: { turn, revision: branchId === "branch-main" ? 7 : 8, story_updated_at: now, active_session_id: "session-1", last_message_id: 2, world_updated_at: now, character_updated_at: now, npc_count: 1, npc_updated_at: now, chapter_count: 1, achievement_count: 0, latest_achievement_at: "", save_count: 0, latest_save_at: "", visual_asset_updated_at: "", visual_job_updated_at: "", active_visual_job_count: 0 },
    story,
    character: { id: "hero", name: "Iria", fields: { stats: { resolve: 42 }, condition: "Steady", inventory: [] } },
    world: { id: "world", current_location: "Glass Archive", current_location_id: "loc-archive", current_chapter: 1, current_turn: turn, spatial_locations: [], spatial_edges: [], world_time: { day: 2, minute_of_day: 780, display_text: "Day 2, 13:00" }, weather: { weather_kind: "clear", description: "Cold clear air" }, known_locations: [], global_events: [], faction_standings: {}, story_hooks: [], world_reactions: [], investigations: [], projects: [], guidance: [], fronts: [], timeline: [], scene_contract: {}, updated_at: now },
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
};

function visualResponse(canUndo = true, canRedo = false) {
  return {
    profile: { id: "profile-1", story_id: story.id, revision: 3, fingerprint: "profile-fingerprint", branch_id: "branch-main", source_commit_id: "commit-4", world_style_prompt: "Glass and brass", character_style_prompt: "Grounded portrait", negative_prompt: "", palette: "amber", updated_at: now },
    assets: [{ id: "asset-mira-new", story_id: story.id, kind: "character", subject: "Mira", entity_id: "npc-mira", canonical_entity_id: "npc-mira", canonical_location_id: "", form_id: "form-mira-restored", lineage_key: "npc-mira:form-mira-restored", appearance_fingerprint: "mira-restored", profile_revision_id: "profile-1", canon_status: "canonical", gate_state: "form_changed", gate_reason: "Mira's restored form has not been rendered on this branch.", generation_eligible: true, prompt: "Mira restored portrait", negative_prompt: "", status: "pending", url: "", provider: "", source: "", error: "", turn: 4, branch_id: "branch-main", source_commit_id: "commit-4", selected_version_id: 21, can_undo_selection: canUndo, can_redo_selection: canRedo, inherited: false, updated_at: now }],
    jobs: [],
  };
}

async function json(route: Route, body: unknown, status = 200) {
  await route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
}

async function mockGateway(page: Page, options: { failAction?: boolean } = {}) {
  let failAction = Boolean(options.failAction);
  let actionRequests = 0;
  let activeMiniGame: any = null;
  let visualCanUndo = true;
  let visualCanRedo = false;
  await page.route("**/api/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;
    if (path.endsWith("/events")) {
      return route.fulfill({ status: 200, contentType: "text/event-stream", headers: { "cache-control": "no-cache" }, body: `event: turn\ndata: ${JSON.stringify({ story_id: story.id, status: "event", client_turn: 4, event_type: "narrative.delta", event: { type: "narrative.delta", payload: { text: "A silver line appears." } }, message: "Narrative is streaming.", created_at: now })}\n\n` });
    }
    if (path === "/api/health") return json(route, { status: "ok", stories: 1 });
    if (path === "/api/stories" && request.method() === "GET") return json(route, [story]);
    if (path === "/api/contracts/commands") return json(route, []);
    if (path === "/api/config/models") return json(route, {
      config_path: "/test/config.yaml", config_revision: "revision-1", provider_priority: ["codex"],
      providers: [{ id: "codex", label: "Codex", enabled: true, model: "gpt-5.5", reasoning: "off", supports_model: true, supports_reasoning: true }],
      narrative_models: ["gpt-5.5"], utility_models: ["gpt-5.5"], repair_models: ["gpt-5.5"], image_models: ["gpt-image-1"], ascii_models: ["gpt-5.5"], embedding_providers: ["auto"],
      image_generation: { provider: "openclaw-bridge", base_url: "", api_key_configured: false, model: "gpt-image-1", openclaw_bridge_url: "http://image.test/generate", default_size: "1024x1024", location_size: "1536x1024", character_size: "1024x1024", default_resolution: "", location_resolution: "", character_resolution: "", default_aspect_ratio: "", location_aspect_ratio: "", character_aspect_ratio: "", quality: "", output_format: "png", background: "", timeout_seconds: 180, auto_generate: false, append_negative_prompt: true, available: true, status: "configured" },
      active: { provider: "codex", narrative_model: "gpt-5.5", utility_model: "gpt-5.5", repair_model: "gpt-5.5", repair_fallback_models: [], image_model: "gpt-image-1", ascii_model: "gpt-5.5", embedding_provider: "auto", embedding_model: "text-embedding-3-small", codex_reasoning: "off" },
      tts_status: "planned",
    });
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
    if (path.endsWith("/timeline") && request.method() === "GET") return json(route, timeline);
    if (path.endsWith("/timeline") && request.method() === "POST") {
      const nextTimeline = { ...timeline, active_branch_id: "branch-alt", revision: 8, head: { ...timeline.head, id: "commit-alt", branch_id: "branch-alt", canonical_turn: 3 } };
      return json(route, { timeline: nextTimeline, snapshot: snapshot(3, "branch-alt") });
    }
    if (path.endsWith("/history")) return json(route, { items: snapshot().messages, next_cursor: null });
    if (path.endsWith("/chapters")) return json(route, { items: snapshot().panels.chapters, next_cursor: null });
    if (path.endsWith("/messages/2/diagnostics")) return json(route, { run_id: "run-2", trace_id: "trace-2", parent_run_id: "", story_id: story.id, branch_id: "branch-main", source_commit_id: "commit-4", message_id: 2, stage: "narrator", status: "succeeded", prompt_profile: "narrator", prompt_revision: 3, prompt_hash: "sha256:redacted", requested_streaming: true, observed_streaming: true, ttft_ms: 180, duration_ms: 1250, usage: { input_tokens: 200, output_tokens: 121, reasoning_tokens: 40, cached_input_tokens: 0, total_tokens: 321, cost_usd: 0.003 }, error_class: "", created_at: now, finished_at: now, attempts: [{ sequence: 1, provider: "codex", requested_model: "gpt-5.5", resolved_model: "gpt-5.5", requested_streaming: true, observed_streaming: true, status: "succeeded", ttft_ms: 180, duration_ms: 1250, usage: { input_tokens: 200, output_tokens: 121, reasoning_tokens: 40, cached_input_tokens: 0, total_tokens: 321, cost_usd: 0.003 }, retry_reason: "", error_class: "" }] });
    if (path.endsWith("/telemetry/export")) return json(route, { format: "jsonl", filename: "glass-archive-generation-telemetry.jsonl", content: "{\"run_id\":\"run-2\"}\n", count: 1, truncated: false });
    if (path.endsWith("/export")) return json(route, { format: url.searchParams.get("format") === "json" ? "json" : "markdown", filename: "glass-archive-history.md", content: "# The Glass Archive\n\nCanonical export" });
    if (path.endsWith("/actions") && request.method() === "POST") {
      actionRequests += 1;
      if (failAction) {
        failAction = false;
        return json(route, { error: "simulated rollback" }, 500);
      }
      return json(route, { events: [{ id: "challenge-start", type: "challenge.started", payload: { protocol_version: "challenge.v1" } }, { id: "challenge-end", type: "challenge.resolved", payload: { degree: "success" } }, { id: "commit", type: "turn.committed", payload: {} }], snapshot: snapshot(5) });
    }
    return json(route, {});
  });
  return { actionRequests: () => actionRequests };
}

test("submits once, clears optimistically, and renders stream/challenge lifecycle", async ({ page }) => {
  const errors: string[] = [];
  page.on("console", (message) => { if (message.type() === "error") errors.push(message.text()); });
  const requests = await mockGateway(page);
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Narrative Transcript" })).toBeVisible();
  await expect(page.getByRole("main").getByText("Mira studies the fractured seal.")).toBeVisible();
  await expect(page.getByLabel("Structured dialogue for turn 4")).toContainText("Choose carefully.");
  await expect(page.getByText("codex · gpt-5.5 · 1.3 s · 321 tokens · streamed")).toBeVisible();
  await page.getByText("Operator diagnostics").click();
  await expect(page.getByText("narrator rev 3")).toBeVisible();
  await expect(page.getByRole("list", { name: "Provider attempts" })).toContainText("TTFT 180 ms");

  const composer = page.getByPlaceholder("Enter command or action...");
  await composer.fill("Trace the silver fracture");
  const actionRequest = page.waitForRequest((request) => request.url().endsWith("/actions"));
  await page.getByRole("button", { name: "Execute" }).evaluate((button: HTMLButtonElement) => { button.click(); button.click(); });
  await actionRequest;
  await expect(composer).toHaveValue("");
  expect(requests.actionRequests()).toBe(1);
  await expect(page.getByText("challenge.started")).toBeVisible();
  await expect(page.getByText("challenge.resolved")).toBeVisible();
  expect(errors).toEqual([]);
});

test("restores a failed draft, checks out a branch, and exposes searchable history/export", async ({ page, browserName }) => {
  await mockGateway(page, { failAction: true });
  await page.goto("/");
  const composer = page.getByPlaceholder("Enter command or action...");
  await composer.fill("Open the forbidden door");
  await page.getByRole("button", { name: "Execute" }).click();
  await expect(composer).toHaveValue("Open the forbidden door");
  await expect(page.getByText("simulated rollback")).toBeVisible();

  await composer.fill("/checkout quiet route");
  await page.getByRole("button", { name: "Execute" }).click();
  await expect(page.getByText("Active branch: quiet route.")).toBeVisible();

  await composer.fill("/history");
  await page.getByRole("button", { name: "Execute" }).click();
  const history = page.viewportSize()!.width <= 1240 ? page.getByRole("dialog") : page.locator(".right-inspector");
  await expect(history.getByPlaceholder("Search this branch")).toBeVisible();
  await history.getByPlaceholder("Search this branch").fill("seal");
  await expect(history.getByRole("heading", { name: "Transcript", exact: true })).toBeVisible();
  await expect(history.getByText("Arrival at the archive.")).toBeVisible();
  const [download] = await Promise.all([
    page.waitForEvent("download"),
    history.getByRole("button", { name: "Export Markdown" }).click(),
  ]);
  expect(download.suggestedFilename()).toBe("glass-archive-history.md");

  const overflow = await page.evaluate(() => ({ width: document.documentElement.scrollWidth, viewport: innerWidth, fontSize: Number.parseFloat(getComputedStyle(document.querySelector("textarea")!).fontSize) }));
  expect(overflow.width).toBeLessThanOrEqual(overflow.viewport + 1);
  if (browserName === "chromium" && page.viewportSize()!.width < 860) expect(overflow.fontSize).toBeGreaterThanOrEqual(16);
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
  await mockGateway(page);
  await page.goto("/");
  const host = page.getByRole("region", { name: "Challenge host" });
  await expect(host.getByRole("button", { name: "Auto-fit" })).toBeVisible();
  await expect(host.getByRole("button", { name: "Courtroom" })).toBeVisible();
  await expect(host.getByRole("button", { name: "Comedy" })).toBeVisible();
  await expect(host.getByRole("button", { name: "Pattern" })).toBeVisible();
  await host.getByRole("button", { name: "Pattern" }).click();
  await expect(host.getByText("Complete the pattern: 2, 4, 6, ?")).toBeVisible();
  await expect(host.getByText("Selected because: player selected; timing-free")).toBeVisible();
  await host.getByLabel("Pattern answer").selectOption("8");
  await host.getByRole("button", { name: "Resolve challenge" }).click();
  await expect(host.getByText("full success", { exact: true })).toBeVisible();
  await expect(host.getByText("Pattern answer: 8 → FULL SUCCESS")).toBeVisible();
  expect(errors).toEqual([]);
});
