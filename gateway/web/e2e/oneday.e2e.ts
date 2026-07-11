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

async function json(route: Route, body: unknown, status = 200) {
  await route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
}

async function mockGateway(page: Page, options: { failAction?: boolean } = {}) {
  let failAction = Boolean(options.failAction);
  let actionRequests = 0;
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
    if (path === "/api/config/models") return json(route, { providers: [], active: { provider: "", model: "" }, generation: {} });
    if (path.endsWith("/snapshot")) return json(route, snapshot());
    if (path.endsWith("/visual-assets")) return json(route, { profile: { story_id: story.id, world_style_prompt: "", character_style_prompt: "", negative_prompt: "", palette: "", updated_at: now }, assets: [], jobs: [] });
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
  await expect(page.getByText("Mira studies the fractured seal.")).toBeVisible();
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
