import type {
  ActionEnvelope,
  ActionResponse,
  CommandDescriptor,
  DeleteSaveEnvelope,
  DeleteSaveResponse,
  Health,
  GenerateVisualAssetsRequest,
  LoadEnvelope,
  LoadResponse,
  MetaEnvelope,
  MetaResponse,
  ModelSettings,
  ModelSettingsUpdate,
  SaveEnvelope,
  SaveResponse,
  StoryCreateEnvelope,
  StoryCreateResponse,
  StoryDeletePlan,
  StoryDeleteResponse,
  StoryEnhanceEnvelope,
  StoryEnhanceResponse,
  StorySnapshot,
  StoryWizardEnvelope,
  StoryWizardResponse,
  StorySummary,
  StoryUpdatePayload,
  VisualAssetsResponse,
  VisualAssetCleanupRequest,
  VisualAssetCleanupResponse,
  VisualAssetPromptUpdate,
  VisualAssetVersion,
  VisualProfileUpdate,
} from "./types";

interface ErrorPayload {
  error?: string;
  code?: string;
}

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

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(path, options);
  const payload = (await response.json().catch(() => {
    if (response.ok) {
      throw new ApiRequestError("Gateway returned a non-JSON response.", response.status, {});
    }
    return {};
  })) as ErrorPayload;
  if (!response.ok) {
    const message = typeof payload?.error === "string" ? payload.error : response.statusText;
    throw new ApiRequestError(message, response.status, payload);
  }
  return payload as T;
}

export function getHealth(): Promise<Health> {
  return request<Health>("/api/health");
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

export function getVisualAssets(storyId: string): Promise<VisualAssetsResponse> {
  return request<VisualAssetsResponse>(`/api/stories/${encodeURIComponent(storyId)}/visual-assets`);
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

export function getCommandDescriptors(): Promise<CommandDescriptor[]> {
  return request<CommandDescriptor[]>("/api/contracts/commands");
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
