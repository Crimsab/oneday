import { afterEach, describe, expect, it, vi } from "vitest";
import { setInterfaceLocale } from "./i18n";
import {
  ApiRequestError,
  AUTHENTICATION_REQUIRED_EVENT,
  bootstrapBrowserSession,
  cancelVisualGenerationJob,
  cleanupVisualAssetFiles,
  createStory,
  deleteStory,
  generateVisualAssets,
  getChapters,
  getCommandDescriptors,
  getHistory,
  getAuthSession,
  getMessageAudio,
  getMessageDiagnostics,
  getModelDiscovery,
  getActiveMiniGame,
  getStories,
  getSetupReadiness,
  getStoryExport,
	getStoryEpub,
  getTelemetryExport,
  getStoryDeletePlan,
  getTimeline,
  stepVisualAssetSelection,
  startMiniGame,
  inputMiniGame,
  runVisualAssetOperation,
  updateTimeline,
  updateStory,
} from "./api";

const originalFetch = globalThis.fetch;

describe("api request handling", () => {
  afterEach(async () => {
	vi.useRealTimers();
    await setInterfaceLocale("en");
    globalThis.fetch = originalFetch;
  });

  it("returns JSON payloads from the gateway", async () => {
    mockFetch(new Response(JSON.stringify([{ id: "story", name: "Story" }]), { status: 200 }));
    await expect(getStories()).resolves.toMatchObject([{ id: "story", name: "Story" }]);
  });

  it("checks the browser session before protected application requests", async () => {
    mockFetch(new Response(JSON.stringify({ authenticated: false, bootstrap_available: true }), { status: 200 }));

    await expect(getAuthSession()).resolves.toEqual({
      authenticated: false,
      bootstrap_available: true,
    });
    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/auth/session",
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
  });

  it("exchanges a bootstrap token without storing it in the client", async () => {
    mockFetch(new Response(JSON.stringify({ session_token: "signed", token_type: "Bearer", expires_in: 43_200 }), { status: 200 }));

    await bootstrapBrowserSession("browser-bootstrap");
    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/auth/bootstrap",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ token: "browser-bootstrap" }),
      }),
    );
  });

  it("announces authentication loss when a protected request returns 401", async () => {
    const listener = vi.fn();
    window.addEventListener(AUTHENTICATION_REQUIRED_EVENT, listener);
    mockFetch(new Response(JSON.stringify({ code: "authentication_required" }), { status: 401 }));

    await expect(getStories()).rejects.toMatchObject({ status: 401 });
    expect(listener).toHaveBeenCalledOnce();
    window.removeEventListener(AUTHENTICATION_REQUIRED_EVENT, listener);
  });

  it("reads the protected installation readiness report", async () => {
    mockFetch(new Response(JSON.stringify({ probes: [{ name: "narrative", status: "ready", required: true }] }), { status: 200 }));

    await expect(getSetupReadiness()).resolves.toMatchObject({ probes: [{ name: "narrative", required: true }] });
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/setup/readiness", expect.objectContaining({ signal: expect.any(AbortSignal) }));
  });

  it("reads server-side model discovery without sending endpoint details", async () => {
    mockFetch(new Response(JSON.stringify({ sources: [{ id: "litellm", status: "ready", models: ["story-model"], checked_at: "2026-07-19T00:00:00Z" }] }), { status: 200 }));
    await expect(getModelDiscovery()).resolves.toMatchObject({ sources: [{ id: "litellm", models: ["story-model"] }] });
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/config/model-discovery", expect.objectContaining({ signal: expect.any(AbortSignal) }));
  });

  it("requests a fresh server-side discovery only when explicitly refreshed", async () => {
    mockFetch(new Response(JSON.stringify({ sources: [] }), { status: 200 }));
    await getModelDiscovery(true);
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/config/model-discovery?refresh=true", expect.objectContaining({ signal: expect.any(AbortSignal) }));
  });

  it("requests localized command presentation while preserving stable command tokens", async () => {
    globalThis.fetch = vi.fn((input: RequestInfo | URL) => {
      const locale = new URL(String(input), "http://oneday.test").searchParams.get("locale");
      return Promise.resolve(new Response(JSON.stringify([{ id: "save", canonical: "save", aliases: ["s"], title: locale === "it" ? "Salva" : "Save", description: locale === "it" ? "Crea un salvataggio." : "Create a save." }] ), { status: 200 }));
    }) as typeof fetch;
    const english = await getCommandDescriptors("en");
    const italian = await getCommandDescriptors("it");
    expect([english[0].title, italian[0].title]).toEqual(["Save", "Salva"]);
    expect([english[0].description, italian[0].description]).toEqual(["Create a save.", "Crea un salvataggio."]);
    expect({ canonical: italian[0].canonical, aliases: italian[0].aliases }).toEqual({ canonical: english[0].canonical, aliases: english[0].aliases });
  });

	it("aborts stalled reads with a typed timeout error", async () => {
		vi.useFakeTimers();
		globalThis.fetch = vi.fn((_path: RequestInfo | URL, options?: RequestInit) => new Promise<Response>((_resolve, reject) => {
			options?.signal?.addEventListener("abort", () => reject(options.signal?.reason), { once: true });
		})) as typeof fetch;
		const pending = getStories();
		vi.advanceTimersByTime(30_000);
		await expect(pending).rejects.toMatchObject({ code: "request_timeout", status: 408 });
	});

  it("normalizes omitted empty audio collections", async () => {
    mockFetch(new Response(JSON.stringify({}), { status: 200 }));

    await expect(getMessageAudio("story-1", 42)).resolves.toEqual({ assets: [], jobs: [] });
  });

  it("posts browser story creation requests to the gateway", async () => {
    mockFetch(
      new Response(
        JSON.stringify({
          story: { story_id: "story-1", character_id: "char-1", started: true },
          snapshot: { story: { id: "story-1" } },
        }),
        { status: 200 },
      ),
    );

    await expect(
      createStory({
        brief: "short test",
        character_name: "Tester",
        character_background: "",
        start: true,
      }),
    ).resolves.toMatchObject({ story: { story_id: "story-1" } });

    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/stories",
      expect.objectContaining({
        method: "POST",
        body: expect.stringContaining("\"brief\":\"short test\""),
      }),
    );
  });

  it("posts visual image generation requests to the story gateway", async () => {
    mockFetch(
      new Response(
        JSON.stringify({
          profile: { story_id: "story-1" },
          assets: [{ id: "asset-1", status: "running" }],
        }),
        { status: 200 },
      ),
    );

    await expect(generateVisualAssets("story-1", { asset_ids: ["asset-1"], force: true, limit: 1 })).resolves.toMatchObject({
      assets: [{ id: "asset-1", status: "running" }],
    });

    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/stories/story-1/visual-assets/generate",
      expect.objectContaining({
        method: "POST",
        body: expect.stringContaining("\"asset_ids\":[\"asset-1\"]"),
      }),
    );
  });

  it("cancels visual generation jobs through the story gateway", async () => {
    mockFetch(
      new Response(
        JSON.stringify({
          profile: { story_id: "story-1" },
          assets: [{ id: "asset-1", status: "pending" }],
          jobs: [{ id: 12, status: "cancelled" }],
        }),
        { status: 200 },
      ),
    );

    await expect(cancelVisualGenerationJob("story-1", 12)).resolves.toMatchObject({
      jobs: [{ id: 12, status: "cancelled" }],
    });

    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/stories/story-1/visual-assets/jobs/12/cancel",
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("submits exact visual operations with fallback forbidden", async () => {
    mockFetch(new Response(JSON.stringify({ profile: {}, assets: [], jobs: [] }), { status: 200 }));

    await runVisualAssetOperation("story/one", "asset one", {
      operation: "inpaint",
      source_version_id: 17,
      prompt: "Replace the lantern",
      mask_png_base64: "cG5n",
      fallback: { mode: "forbid" },
      idempotency_key: "operation-1",
    });

    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/stories/story%2Fone/visual-assets/asset%20one/operations",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          operation: "inpaint",
          source_version_id: 17,
          prompt: "Replace the lantern",
          mask_png_base64: "cG5n",
          fallback: { mode: "forbid" },
          idempotency_key: "operation-1",
        }),
      }),
    );
  });

  it("requests visual asset cleanup without exposing file contents", async () => {
    mockFetch(
      new Response(
        JSON.stringify({
          story_id: "story-1",
          dry_run: true,
          deleted_files: ["/tmp/stale.png"],
          kept_files: ["/tmp/kept.png"],
        }),
        { status: 200 },
      ),
    );

    await expect(cleanupVisualAssetFiles("story-1", { dry_run: true })).resolves.toMatchObject({
      story_id: "story-1",
      dry_run: true,
      deleted_files: ["/tmp/stale.png"],
    });

    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/stories/story-1/visual-assets/cleanup",
      expect.objectContaining({
        method: "POST",
        body: expect.stringContaining("\"dry_run\":true"),
      }),
    );
  });

  it("patches story metadata and archive state", async () => {
    mockFetch(new Response(JSON.stringify({ id: "story-1", name: "Renamed", is_archived: true }), { status: 200 }));

    await expect(updateStory("story-1", { name: "Renamed", is_archived: true })).resolves.toMatchObject({
      id: "story-1",
      name: "Renamed",
      is_archived: true,
    });

    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/stories/story-1",
      expect.objectContaining({
        method: "PATCH",
        body: expect.stringContaining("\"is_archived\":true"),
      }),
    );
  });

  it("deletes stories through the gateway", async () => {
    mockFetch(new Response(JSON.stringify({ story_id: "story-1" }), { status: 200 }));

    await expect(deleteStory("story-1")).resolves.toEqual({ story_id: "story-1" });
    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/stories/story-1",
      expect.objectContaining({
        method: "DELETE",
      }),
    );
  });

  it("loads story delete plans before destructive deletion", async () => {
    mockFetch(new Response(JSON.stringify({ story_id: "story-1", total_rows: 42, counts: [], retained_asset_files: [] }), { status: 200 }));

    await expect(getStoryDeletePlan("story-1")).resolves.toMatchObject({ story_id: "story-1", total_rows: 42 });
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/stories/story-1/delete-plan", expect.objectContaining({ signal: expect.any(AbortSignal) }));
  });

  it("uses the branch timeline contract for reads and guarded mutations", async () => {
    mockFetch(new Response(JSON.stringify({ active_branch_id: "branch-main", revision: 7, branches: [] }), { status: 200 }));
    await expect(getTimeline("story/one")).resolves.toMatchObject({ active_branch_id: "branch-main", revision: 7 });
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/stories/story%2Fone/timeline", expect.objectContaining({ signal: expect.any(AbortSignal) }));

    mockFetch(new Response(JSON.stringify({ timeline: { active_branch_id: "branch-alt" }, snapshot: { story: { id: "story/one" } } }), { status: 200 }));
    await updateTimeline("story/one", { action: "checkout", client_revision: 7, branch_id: "branch-alt" });
    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/stories/story%2Fone/timeline",
      expect.objectContaining({ method: "POST", body: expect.stringContaining('"client_revision":7') }),
    );

    mockFetch(new Response(JSON.stringify({ timeline: { active_branch_id: "branch-fork" }, snapshot: { story: { id: "story/one" } } }), { status: 200 }));
    await updateTimeline("story/one", { action: "fork_checkout", client_revision: 8, from_commit_id: "commit/one", name: "alternate" });
    expect(globalThis.fetch).toHaveBeenLastCalledWith(
      "/api/stories/story%2Fone/timeline",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ action: "fork_checkout", client_revision: 8, from_commit_id: "commit/one", name: "alternate" }),
      }),
    );
  });

  it("recovers timeline reads from transient server failures", async () => {
    mockFetchSequence([
      new Response(JSON.stringify({ error: "temporary timeline failure" }), { status: 500 }),
      new Response(JSON.stringify({ active_branch_id: "branch-main", revision: 9, branches: [], head: null, commits: [] }), { status: 200 }),
    ]);

    await expect(getTimeline("story-1")).resolves.toMatchObject({ revision: 9 });
    expect(globalThis.fetch).toHaveBeenCalledTimes(2);
  });

  it("requests branch-scoped paginated history, chapters, and exports", async () => {
    mockFetch(new Response(JSON.stringify({ items: [], next_cursor: null }), { status: 200 }));
    await getHistory("story-1", 41, "glass seal");
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/stories/story-1/history?limit=40&q=glass+seal&cursor=41", expect.objectContaining({ signal: expect.any(AbortSignal) }));

    mockFetch(new Response(JSON.stringify({ items: [], next_cursor: null }), { status: 200 }));
    await getChapters("story-1", 3, "arrival");
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/stories/story-1/chapters?limit=30&q=arrival&cursor=3", expect.objectContaining({ signal: expect.any(AbortSignal) }));

    mockFetch(new Response(JSON.stringify({ format: "json", filename: "history.json", content: "{}" }), { status: 200 }));
    await expect(getStoryExport("story-1", "json")).resolves.toMatchObject({ filename: "history.json" });
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/stories/story-1/export?format=json", expect.objectContaining({ signal: expect.any(AbortSignal) }));
  });

	it("downloads EPUB as binary without base64 JSON expansion", async () => {
		mockFetch(new Response(new Uint8Array([80, 75, 3, 4]), {
			status: 200,
			headers: {
				"content-type": "application/epub+zip",
				"content-disposition": 'attachment; filename="story.epub"',
			},
		}));
		const result = await getStoryEpub("story-1");
		expect(result.filename).toBe("story.epub");
		expect(result.blob.size).toBe(4);
		expect(globalThis.fetch).toHaveBeenCalledWith(
			"/api/stories/story-1/export?format=epub",
			expect.objectContaining({ signal: expect.any(AbortSignal) }),
		);
	});

  it("loads message diagnostics and redacted telemetry exports", async () => {
    mockFetch(new Response(JSON.stringify({ run_id: "run-1", attempts: [] }), { status: 200 }));
    await expect(getMessageDiagnostics("story/one", 42)).resolves.toMatchObject({ run_id: "run-1" });
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/stories/story%2Fone/messages/42/diagnostics", expect.objectContaining({ signal: expect.any(AbortSignal) }));

    mockFetch(new Response(JSON.stringify({ format: "jsonl", filename: "telemetry.jsonl", content: "", count: 0, truncated: false }), { status: 200 }));
    await expect(getTelemetryExport("story/one", 250)).resolves.toMatchObject({ format: "jsonl", count: 0 });
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/stories/story%2Fone/telemetry/export?limit=250", expect.objectContaining({ signal: expect.any(AbortSignal) }));
  });

  it("steps branch-local visual selection history", async () => {
    mockFetch(new Response(JSON.stringify({ profile: {}, assets: [], jobs: [] }), { status: 200 }));
    await stepVisualAssetSelection("story/one", "asset one", "undo");
    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/stories/story%2Fone/visual-assets/asset%20one/selection/undo",
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("uses the branch-scoped minigame host endpoints", async () => {
    mockFetch(new Response(JSON.stringify({ instance: null }), { status: 200 }));
    await getActiveMiniGame("story/one");
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/stories/story%2Fone/minigames", expect.objectContaining({ signal: expect.any(AbortSignal) }));

    mockFetch(new Response(JSON.stringify({ instance: { id: "mini-1" } }), { status: 200 }));
    await startMiniGame("story/one", "pattern");
    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/stories/story%2Fone/minigames",
      expect.objectContaining({ method: "POST", body: expect.stringContaining('"kind":"pattern"') }),
    );

    mockFetch(new Response(JSON.stringify({ instance: { id: "mini-1", runtime: { phase: "resolved" } } }), { status: 200 }));
    await inputMiniGame("story/one", "mini/1", { action: "submit", value: "8" });
    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/stories/story%2Fone/minigames/mini%2F1/input",
      expect.objectContaining({ method: "POST", body: expect.stringContaining('"value":"8"') }),
    );
  });

  it("rejects successful non-JSON responses before they crash React state", async () => {
    mockFetch(new Response("<html>vite fallback</html>", { status: 200, headers: { "content-type": "text/html" } }));

    await expect(getStories()).rejects.toMatchObject({
      name: "ApiRequestError",
      status: 200,
      message: "The gateway returned an unreadable response.",
    });
  });

  it("keeps non-JSON error responses controlled", async () => {
    mockFetch(new Response("", { status: 502, statusText: "Bad Gateway" }));
    await expect(getStories()).rejects.toBeInstanceOf(ApiRequestError);
    await expect(getStories()).rejects.toMatchObject({ status: 502, message: "Something went wrong. Try again." });
  });

  it("localizes stable API error codes without exposing server prose", async () => {
    await setInterfaceLocale("it");
    mockFetch(new Response(JSON.stringify({ code: "stale_config", error: "raw server detail" }), { status: 409 }));
    await expect(getStories()).rejects.toMatchObject({ code: "stale_config", message: "La configurazione è cambiata. Aggiorna e riprova." });
  });

  it("maps public audio and challenge validation codes", async () => {
    await setInterfaceLocale("it");
    mockFetch(new Response(JSON.stringify({ code: "invalid_audio_request" }), { status: 400 }));
    await expect(getStories()).rejects.toMatchObject({ message: "Controlla le impostazioni dell’audio parlato e riprova." });
    mockFetch(new Response(JSON.stringify({ code: "invalid_minigame_request" }), { status: 400 }));
    await expect(getStories()).rejects.toMatchObject({ message: "L’azione della sfida non è valida. Controlla la scelta e riprova." });
  });

  it("uses a safe localized fallback for unknown client errors", async () => {
    await setInterfaceLocale("it");
    mockFetch(new Response(JSON.stringify({ code: "future_error", error: "sensitive internal detail" }), { status: 422 }));
    await expect(getStories()).rejects.toMatchObject({ code: "future_error", message: "Impossibile completare la richiesta. Controlla i dati e riprova." });
  });
});

function mockFetch(response: Response) {
  globalThis.fetch = vi.fn(async () => response.clone()) as typeof fetch;
}

function mockFetchSequence(responses: Response[]) {
  let index = 0;
  globalThis.fetch = vi.fn(async () => responses[Math.min(index++, responses.length - 1)].clone()) as typeof fetch;
}
