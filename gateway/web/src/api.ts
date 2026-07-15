import type {
  ActionEnvelope,
  ActionResponse,
  CommandDescriptor,
  CraftConversationMessage,
  CraftResponseEnvelope,
  DeleteSaveEnvelope,
  DeleteSaveResponse,
  Health,
	HistoryPage,
	ChapterPage,
  GenerationDiagnostics,
  GenerateVisualAssetsRequest,
  LoadEnvelope,
  LoadResponse,
  MetaEnvelope,
  MetaResponse,
  ModelSettings,
  ModelSettingsUpdate,
  MiniGameInput,
  MiniGameKind,
  MiniGameResponse,
  SaveEnvelope,
  SaveResponse,
  StoryCreateEnvelope,
  StoryCreateResponse,
  StoryDeletePlan,
  StoryDeleteResponse,
  StoryEnhanceEnvelope,
  StoryEnhanceResponse,
  StorySnapshot,
	StoryExport,
	TelemetryExport,
	TimelineEnvelope,
	TimelineMutationResponse,
	TimelineResponse,
  StoryWizardEnvelope,
  StoryWizardResponse,
  StorySummary,
  StoryUpdatePayload,
  VisualAssetsResponse,
  VisualAssetCleanupRequest,
  VisualAssetCleanupResponse,
  VisualAssetPromptUpdate,
  VisualAssetVersion,
  VisualAssetOperationRequest,
  VisualProfileUpdate,
  TTSCatalogResponse,
  TTSSettingsResponse,
  VoiceAssignmentsResponse,
  MessageAudioResponse,
  StoryTTSSettings,
  VoiceAssignment,
  PronunciationEntry,
  PronunciationsResponse,
  AudioCleanupResponse,
  AudioExportResponse,
  AgencyEventView,
} from "./types";
import i18n from "./i18n";
import { recordApiSupportEvent } from "./supportDiagnostics";

interface ErrorPayload {
  error?: string;
  code?: string;
}

interface RequestOptions extends RequestInit {
	timeoutMs?: number;
}

const READ_TIMEOUT_MS = 30_000;
const MUTATION_TIMEOUT_MS = 360_000;

export class ApiRequestError extends Error {
  status: number;
  code: string;
  payload: unknown;

  constructor(message: string, status: number, payload: ErrorPayload) {
    super(message);
    this.name = "ApiRequestError";
    this.status = status;
    this.code = typeof payload?.code === "string" ? payload.code : "";
    this.payload = payload;
  }
}

async function fetchWithTimeout(path: string, options: RequestOptions = {}): Promise<Response> {
	const startedAt = performance.now();
	const method = options.method ?? "GET";
	const { timeoutMs, signal: callerSignal, ...fetchOptions } = options;
	const controller = new AbortController();
	const timeout = globalThis.setTimeout(
		() => controller.abort(new DOMException("Request timed out", "TimeoutError")),
		timeoutMs ?? (fetchOptions.method && fetchOptions.method !== "GET" ? MUTATION_TIMEOUT_MS : READ_TIMEOUT_MS),
	);
	const abortFromCaller = () => controller.abort(callerSignal?.reason);
	if (callerSignal?.aborted) {
		abortFromCaller();
	} else {
		callerSignal?.addEventListener("abort", abortFromCaller, { once: true });
	}
	let response: Response;
	try {
		response = await fetch(path, { ...fetchOptions, signal: controller.signal });
		recordApiSupportEvent(method, path, response.status, performance.now() - startedAt);
	} catch (error) {
		recordApiSupportEvent(method, path, 0, performance.now() - startedAt, error instanceof Error ? error.message : String(error));
		if (controller.signal.aborted && !callerSignal?.aborted) {
			throw new ApiRequestError(i18n.t("common:requestTimedOut"), 408, { code: "request_timeout" });
		}
		throw error;
	} finally {
		globalThis.clearTimeout(timeout);
		callerSignal?.removeEventListener("abort", abortFromCaller);
	}
	return response;
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
	const response = await fetchWithTimeout(path, options);
  const payload = (await response.json().catch(() => {
    if (response.ok) {
      throw new ApiRequestError(i18n.t("common:nonJson"), response.status, {});
    }
    return {};
  })) as ErrorPayload;
  if (!response.ok) {
    const message = localizedError(payload, response.status);
    throw new ApiRequestError(message, response.status, payload);
  }
  return payload as T;
}

function localizedError(payload: ErrorPayload, status: number): string {
  const knownCodes = new Set(["invalid_request", "invalid_audio_request", "invalid_minigame_request", "stale_request", "conflict", "not_found", "validation_failed", "stale_config", "config_locked", "internal_error", "request_timeout"]);
  if (payload.code && knownCodes.has(payload.code)) return i18n.t(`api_errors:${payload.code}`);
  if (status >= 400 && status < 500) return i18n.t("api_errors:client_error");
  return i18n.t("api_errors:internal_error");
}

export function getHealth(): Promise<Health> {
  return request<Health>("/api/health");
}

export function getAgencyEvents(storyId: string, limit = 20): Promise<AgencyEventView[]> {
  return request<AgencyEventView[]>(`/api/stories/${encodeURIComponent(storyId)}/agency-events?limit=${limit}`);
}

export function getTTSCatalog(language = ""): Promise<TTSCatalogResponse> {
  const query = new URLSearchParams();
  if (language) query.set("language", language);
  return request<TTSCatalogResponse>(`/api/tts/voices${query.size ? `?${query}` : ""}`);
}

export function getTTSSettings(storyId: string): Promise<TTSSettingsResponse> {
  return request<TTSSettingsResponse>(`/api/stories/${encodeURIComponent(storyId)}/tts/settings`);
}

export function updateTTSSettings(storyId: string, settings: StoryTTSSettings, clientRevision: number): Promise<TTSSettingsResponse> {
  return request<TTSSettingsResponse>(`/api/stories/${encodeURIComponent(storyId)}/tts/settings`, {
    method: "PUT", headers: { "content-type": "application/json" },
    body: JSON.stringify({ ...settings, client_revision: clientRevision }),
  });
}

export function getVoiceAssignments(storyId: string): Promise<VoiceAssignmentsResponse> {
  return request<VoiceAssignmentsResponse>(`/api/stories/${encodeURIComponent(storyId)}/voice-assignments`);
}

export function updateVoiceAssignment(storyId: string, assignment: VoiceAssignment, clientRevision: number): Promise<VoiceAssignmentsResponse> {
  return request<VoiceAssignmentsResponse>(`/api/stories/${encodeURIComponent(storyId)}/voice-assignments/${encodeURIComponent(assignment.id)}`, {
    method: "PUT", headers: { "content-type": "application/json" },
    body: JSON.stringify({ ...assignment, client_revision: clientRevision }),
  });
}

export function getMessageAudio(storyId: string, messageId: number): Promise<MessageAudioResponse> {
  return request<Partial<MessageAudioResponse>>(`/api/stories/${encodeURIComponent(storyId)}/messages/${messageId}/audio`)
    .then(normalizeMessageAudioResponse);
}

export function createMessageAudio(storyId: string, messageId: number): Promise<MessageAudioResponse> {
  return request<Partial<MessageAudioResponse>>(`/api/stories/${encodeURIComponent(storyId)}/messages/${messageId}/audio`, { method: "POST" })
    .then(normalizeMessageAudioResponse);
}

export function retryAudioJob(storyId: string, jobId: string): Promise<MessageAudioResponse> {
  return request<Partial<MessageAudioResponse>>(`/api/stories/${encodeURIComponent(storyId)}/audio/jobs/${encodeURIComponent(jobId)}/retry`, { method: "POST" })
    .then(normalizeMessageAudioResponse);
}

export function cancelAudioJob(storyId: string, jobId: string): Promise<MessageAudioResponse> {
  return request<Partial<MessageAudioResponse>>(`/api/stories/${encodeURIComponent(storyId)}/audio/jobs/${encodeURIComponent(jobId)}/cancel`, { method: "POST" })
    .then(normalizeMessageAudioResponse);
}

export function normalizeMessageAudioResponse(response: Partial<MessageAudioResponse>): MessageAudioResponse {
  return {
    ...response,
    assets: Array.isArray(response.assets) ? response.assets : [],
    jobs: Array.isArray(response.jobs) ? response.jobs : [],
  };
}

export function getPronunciations(storyId: string, language = ""): Promise<PronunciationsResponse> {
  const query = new URLSearchParams();
  if (language) query.set("language", language);
  return request<PronunciationsResponse>(`/api/stories/${encodeURIComponent(storyId)}/pronunciations${query.size ? `?${query}` : ""}`);
}

export function updatePronunciation(storyId: string, entry: PronunciationEntry, clientRevision: number): Promise<PronunciationsResponse> {
  return request<PronunciationsResponse>(`/api/stories/${encodeURIComponent(storyId)}/pronunciations/${encodeURIComponent(entry.id)}`, {
    method: "PUT", headers: { "content-type": "application/json" }, body: JSON.stringify({ ...entry, client_revision: clientRevision }),
  });
}

export function deletePronunciation(storyId: string, id: string, clientRevision: number): Promise<PronunciationsResponse> {
  return request<PronunciationsResponse>(`/api/stories/${encodeURIComponent(storyId)}/pronunciations/${encodeURIComponent(id)}?client_revision=${clientRevision}`, { method: "DELETE" });
}

export function cleanupAudio(storyId: string, dryRun = true): Promise<AudioCleanupResponse> {
  return request<AudioCleanupResponse>(`/api/stories/${encodeURIComponent(storyId)}/audio/cleanup`, {
    method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify({ dry_run: dryRun }),
  });
}

export function getAudioExport(storyId: string): Promise<AudioExportResponse> {
  return request<AudioExportResponse>(`/api/stories/${encodeURIComponent(storyId)}/audio/export`);
}

export function getStories(): Promise<StorySummary[]> {
  return request<StorySummary[]>("/api/stories");
}

export function createStory(envelope: StoryCreateEnvelope): Promise<StoryCreateResponse> {
  return request<StoryCreateResponse>("/api/stories", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(envelope),
  });
}

export function updateStory(storyId: string, payload: StoryUpdatePayload): Promise<StorySummary> {
  return request<StorySummary>(`/api/stories/${encodeURIComponent(storyId)}`, {
    method: "PATCH",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export function deleteStory(storyId: string): Promise<StoryDeleteResponse> {
  return request<StoryDeleteResponse>(`/api/stories/${encodeURIComponent(storyId)}`, {
    method: "DELETE",
  });
}

export function getStoryDeletePlan(storyId: string): Promise<StoryDeletePlan> {
  return request<StoryDeletePlan>(`/api/stories/${encodeURIComponent(storyId)}/delete-plan`);
}

export function runStoryWizard(envelope: StoryWizardEnvelope): Promise<StoryWizardResponse> {
  return request<StoryWizardResponse>("/api/story-wizard", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(envelope),
  });
}

export function enhanceStoryText(envelope: StoryEnhanceEnvelope): Promise<StoryEnhanceResponse> {
  return request<StoryEnhanceResponse>("/api/story-enhance", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(envelope),
  });
}

export function getSnapshot(storyId: string): Promise<StorySnapshot> {
  return request<StorySnapshot>(`/api/stories/${encodeURIComponent(storyId)}/snapshot`);
}

export function sendCraftMessage(
  storyId: string,
  message: string,
  history: CraftConversationMessage[],
): Promise<CraftResponseEnvelope> {
  return request<CraftResponseEnvelope>(`/api/stories/${encodeURIComponent(storyId)}/craft`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ message, history }),
  });
}

export async function getTimeline(storyId:string):Promise<TimelineResponse> {
  const path = `/api/stories/${encodeURIComponent(storyId)}/timeline`;
  let lastError: unknown;
  for (let attempt = 0; attempt < 3; attempt += 1) {
    try {
      return await request<TimelineResponse>(path);
    } catch (error) {
      lastError = error;
      const retryable = error instanceof ApiRequestError ? error.status >= 500 : error instanceof TypeError;
      if (!retryable || attempt === 2) throw error;
      await new Promise((resolve) => globalThis.setTimeout(resolve, 120 * (attempt + 1)));
    }
  }
  throw lastError;
}
export function updateTimeline(storyId:string,payload:TimelineEnvelope):Promise<TimelineMutationResponse> { return request<TimelineMutationResponse>(`/api/stories/${encodeURIComponent(storyId)}/timeline`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(payload)}); }
export function getHistory(storyId:string,cursor?:number,search="",signal?:AbortSignal):Promise<HistoryPage> { const query=new URLSearchParams({limit:"40",q:search}); if(cursor)query.set("cursor",String(cursor)); return request<HistoryPage>(`/api/stories/${encodeURIComponent(storyId)}/history?${query}`,{signal}); }
export function getChapters(storyId:string,cursor?:number,search="",signal?:AbortSignal):Promise<ChapterPage> { const query=new URLSearchParams({limit:"30",q:search}); if(cursor)query.set("cursor",String(cursor)); return request<ChapterPage>(`/api/stories/${encodeURIComponent(storyId)}/chapters?${query}`,{signal}); }
export function getStoryExport(storyId:string,format:"markdown"|"json"|"epub"|"replay"):Promise<StoryExport> { return request<StoryExport>(`/api/stories/${encodeURIComponent(storyId)}/export?format=${format}`); }
export async function getStoryEpub(storyId:string):Promise<{filename:string;blob:Blob}> {
	const response = await fetchWithTimeout(`/api/stories/${encodeURIComponent(storyId)}/export?format=epub`);
	if (!response.ok) {
		const payload = await response.json().catch(() => ({})) as ErrorPayload;
		throw new ApiRequestError(localizedError(payload, response.status), response.status, payload);
	}
	const disposition = response.headers.get("content-disposition") ?? "";
	const filename = disposition.match(/filename="?([^";]+)"?/i)?.[1] ?? `${storyId}-history.epub`;
	return { filename, blob: await response.blob() };
}
export function getMessageDiagnostics(storyId:string,messageId:number):Promise<GenerationDiagnostics> { return request<GenerationDiagnostics>(`/api/stories/${encodeURIComponent(storyId)}/messages/${encodeURIComponent(String(messageId))}/diagnostics`); }
export function getTelemetryExport(storyId:string,limit=1000):Promise<TelemetryExport> { return request<TelemetryExport>(`/api/stories/${encodeURIComponent(storyId)}/telemetry/export?limit=${encodeURIComponent(String(limit))}`); }

export function getVisualAssets(storyId: string): Promise<VisualAssetsResponse> {
  return request<VisualAssetsResponse>(`/api/stories/${encodeURIComponent(storyId)}/visual-assets`);
}

export function getActiveMiniGame(storyId: string): Promise<MiniGameResponse> {
  return request<MiniGameResponse>(`/api/stories/${encodeURIComponent(storyId)}/minigames`);
}

export function startMiniGame(storyId: string, kind: MiniGameKind | null, narrativeTags: string[] = []): Promise<MiniGameResponse> {
  return request<MiniGameResponse>(`/api/stories/${encodeURIComponent(storyId)}/minigames`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      definition: { kind: kind ?? "" },
      selection: { narrative_tags: narrativeTags, difficulty: 50, timing_free_only: true },
    }),
  });
}

export function inputMiniGame(storyId: string, instanceId: string, input: MiniGameInput): Promise<MiniGameResponse> {
  return request<MiniGameResponse>(`/api/stories/${encodeURIComponent(storyId)}/minigames/${encodeURIComponent(instanceId)}/input`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ input }),
  });
}

export function updateVisualProfile(storyId: string, payload: VisualProfileUpdate): Promise<VisualAssetsResponse> {
  return request<VisualAssetsResponse>(`/api/stories/${encodeURIComponent(storyId)}/visual-profile`, {
    method: "PUT",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export function generateVisualAssets(storyId: string, payload: GenerateVisualAssetsRequest = {}): Promise<VisualAssetsResponse> {
  return request<VisualAssetsResponse>(`/api/stories/${encodeURIComponent(storyId)}/visual-assets/generate`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export function cancelVisualGenerationJob(storyId: string, jobId: number): Promise<VisualAssetsResponse> {
  return request<VisualAssetsResponse>(
    `/api/stories/${encodeURIComponent(storyId)}/visual-assets/jobs/${encodeURIComponent(String(jobId))}/cancel`,
    { method: "POST" },
  );
}

export function cleanupVisualAssetFiles(
  storyId: string,
  payload: VisualAssetCleanupRequest = {},
): Promise<VisualAssetCleanupResponse> {
  return request<VisualAssetCleanupResponse>(
    `/api/stories/${encodeURIComponent(storyId)}/visual-assets/cleanup`,
    {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify(payload),
    },
  );
}

export function getVisualAssetVersions(storyId: string, assetId: string): Promise<VisualAssetVersion[]> {
  return request<VisualAssetVersion[]>(
    `/api/stories/${encodeURIComponent(storyId)}/visual-assets/${encodeURIComponent(assetId)}/versions`,
  );
}

export function updateVisualAssetPrompt(
  storyId: string,
  assetId: string,
  payload: VisualAssetPromptUpdate,
): Promise<VisualAssetsResponse> {
  return request<VisualAssetsResponse>(
    `/api/stories/${encodeURIComponent(storyId)}/visual-assets/${encodeURIComponent(assetId)}`,
    {
      method: "PUT",
      headers: { "content-type": "application/json" },
      body: JSON.stringify(payload),
    },
  );
}

export function selectVisualAssetVersion(storyId: string, assetId: string, versionId: number): Promise<VisualAssetsResponse> {
  return request<VisualAssetsResponse>(
    `/api/stories/${encodeURIComponent(storyId)}/visual-assets/${encodeURIComponent(assetId)}/versions/${encodeURIComponent(String(versionId))}/select`,
    { method: "POST" },
  );
}

export function stepVisualAssetSelection(storyId: string, assetId: string, action: "undo" | "redo"): Promise<VisualAssetsResponse> {
  return request<VisualAssetsResponse>(
    `/api/stories/${encodeURIComponent(storyId)}/visual-assets/${encodeURIComponent(assetId)}/selection/${action}`,
    { method: "POST" },
  );
}

export function runVisualAssetOperation(
  storyId: string,
  assetId: string,
  payload: VisualAssetOperationRequest,
): Promise<VisualAssetsResponse> {
  return request<VisualAssetsResponse>(
    `/api/stories/${encodeURIComponent(storyId)}/visual-assets/${encodeURIComponent(assetId)}/operations`,
    {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify(payload),
    },
  );
}

export function getCommandDescriptors(locale = i18n.resolvedLanguage ?? "en"): Promise<CommandDescriptor[]> {
  return request<CommandDescriptor[]>(`/api/contracts/commands?locale=${encodeURIComponent(locale)}`);
}

export function getModelSettings(): Promise<ModelSettings> {
  return request<ModelSettings>("/api/config/models");
}

export function updateModelSettings(payload: ModelSettingsUpdate): Promise<ModelSettings> {
  return request<ModelSettings>("/api/config/models", {
    method: "PUT",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export function submitAction(storyId: string, envelope: ActionEnvelope): Promise<ActionResponse> {
  return request<ActionResponse>(`/api/stories/${encodeURIComponent(storyId)}/actions`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(envelope),
  });
}

export function submitMeta(storyId: string, envelope: MetaEnvelope): Promise<MetaResponse> {
  return request<MetaResponse>(`/api/stories/${encodeURIComponent(storyId)}/meta`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(envelope),
  });
}

export function createSave(storyId: string, envelope: SaveEnvelope): Promise<SaveResponse> {
  return request<SaveResponse>(`/api/stories/${encodeURIComponent(storyId)}/saves`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(envelope),
  });
}

export function loadSave(storyId: string, envelope: LoadEnvelope): Promise<LoadResponse> {
  return request<LoadResponse>(
    `/api/stories/${encodeURIComponent(storyId)}/saves/${encodeURIComponent(envelope.save_id)}/load`,
    {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify(envelope),
    },
  );
}

export function deleteSave(storyId: string, envelope: DeleteSaveEnvelope): Promise<DeleteSaveResponse> {
  return request<DeleteSaveResponse>(`/api/stories/${encodeURIComponent(storyId)}/saves/${encodeURIComponent(envelope.save_id)}`, {
    method: "DELETE",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(envelope),
  });
}
