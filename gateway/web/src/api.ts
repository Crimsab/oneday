import type { ActionEnvelope, ActionResponse, Health, StorySnapshot, StorySummary } from "./types";

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(path, options);
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    const message = typeof payload?.error === "string" ? payload.error : response.statusText;
    throw new Error(message);
  }
  return payload as T;
}

export function getHealth(): Promise<Health> {
  return request<Health>("/api/health");
}

export function getStories(): Promise<StorySummary[]> {
  return request<StorySummary[]>("/api/stories");
}

export function getSnapshot(storyId: string): Promise<StorySnapshot> {
  return request<StorySnapshot>(`/api/stories/${encodeURIComponent(storyId)}/snapshot`);
}

export function submitAction(storyId: string, envelope: ActionEnvelope): Promise<ActionResponse> {
  return request<ActionResponse>(`/api/stories/${encodeURIComponent(storyId)}/actions`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(envelope),
  });
}
