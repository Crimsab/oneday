import type {
  ActionEnvelope,
  ActionResponse,
  CommandDescriptor,
  DeleteSaveEnvelope,
  DeleteSaveResponse,
  Health,
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
  StorySnapshot,
  StorySummary,
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

export function getSnapshot(storyId: string): Promise<StorySnapshot> {
  return request<StorySnapshot>(`/api/stories/${encodeURIComponent(storyId)}/snapshot`);
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
