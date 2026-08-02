import type { BrowserTranslationItem, TranslationEstimate, TranslationGlossaryEntry, TranslationJob, TranslationJobRequest } from "./batchTypes";

async function batchRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, { ...init, headers: { ...(init?.body ? { "Content-Type": "application/json" } : {}), ...init?.headers } });
  if (!response.ok) {
    const payload = await response.json().catch(() => ({})) as { error?: string; message?: string };
    throw new Error(payload.message || payload.error || `HTTP ${response.status}`);
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

const root = (storyId: string) => `/api/stories/${encodeURIComponent(storyId)}/translations`;

async function batchList<T>(path: string): Promise<T[]> {
  const payload = await batchRequest<unknown>(path);
  if (!Array.isArray(payload)) throw new Error("The translation service returned an unreadable response.");
  return payload as T[];
}

export const listTranslationJobs = (storyId: string) => batchList<TranslationJob>(`${root(storyId)}/jobs`);
export const estimateTranslationJob = (storyId: string, request: TranslationJobRequest, signal?: AbortSignal) => batchRequest<TranslationEstimate>(`${root(storyId)}/jobs/estimate`, { method: "POST", body: JSON.stringify(request), signal });
export const createTranslationJob = (storyId: string, request: TranslationJobRequest) => batchRequest<TranslationJob>(`${root(storyId)}/jobs`, { method: "POST", body: JSON.stringify(request) });
export const runTranslationJobAction = (storyId: string, jobId: string, action: "pause" | "resume" | "cancel" | "retry") => batchRequest<TranslationJob>(`${root(storyId)}/jobs/${encodeURIComponent(jobId)}/${action}`, { method: "POST" });
export const deleteTranslationJob = (storyId: string, jobId: string, deleteTranslations: boolean) => batchRequest<void>(`${root(storyId)}/jobs/${encodeURIComponent(jobId)}?delete_translations=${deleteTranslations}`, { method: "DELETE" });
export const nextBrowserTranslationItem = (storyId: string, jobId: string) => batchRequest<BrowserTranslationItem | null>(`${root(storyId)}/jobs/${encodeURIComponent(jobId)}/browser-next`);
export const completeBrowserTranslationItem = (storyId: string, jobId: string, itemId: string, translatedText: string) => batchRequest<TranslationJob>(`${root(storyId)}/jobs/${encodeURIComponent(jobId)}/items/${encodeURIComponent(itemId)}`, { method: "POST", body: JSON.stringify({ translated_text: translatedText }) });
export const listTranslationGlossary = (storyId: string) => batchList<TranslationGlossaryEntry>(`${root(storyId)}/glossary`);
export const createTranslationGlossary = (storyId: string, entry: Omit<TranslationGlossaryEntry, "id">) => batchRequest<TranslationGlossaryEntry>(`${root(storyId)}/glossary`, { method: "POST", body: JSON.stringify(entry) });
export const deleteTranslationGlossary = (storyId: string, entryId: string) => batchRequest<void>(`${root(storyId)}/glossary/${encodeURIComponent(entryId)}`, { method: "DELETE" });
export const translateTextWithAi = (storyId: string, request: { text: string; source_language: string; target_language: string; provider: string; model: string; style?: "faithful" | "natural" | "literary" }) => batchRequest<{ translated_text: string }>(`${root(storyId)}/text`, { method: "POST", body: JSON.stringify(request) });
