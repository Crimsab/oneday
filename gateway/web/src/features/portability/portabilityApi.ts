export type ReadableFormat = "markdown" | "html" | "txt" | "json" | "epub";
export type ReadingMode = "original" | "translated" | "bilingual";

export interface ArchiveOptions {
  history: boolean;
  saves: boolean;
  visual_assets: boolean;
  audio: boolean;
  translations: boolean;
  world_detail: boolean;
}

export interface ImportResult {
  story_id: string;
  story_name: string;
  imported_tables: number;
  imported_files: number;
}

function filename(response: Response, fallback: string): string {
  return response.headers.get("content-disposition")?.match(/filename="?([^";]+)"?/i)?.[1] ?? fallback;
}

async function checked(response: Response): Promise<Response> {
  if (response.ok) return response;
  const payload = await response.json().catch(() => ({})) as { error?: string };
  throw new Error(payload.error || `Request failed (${response.status})`);
}

export async function exportArchive(storyId: string, options: ArchiveOptions): Promise<{ blob: Blob; filename: string }> {
  const response = await checked(await fetch(`/api/stories/${encodeURIComponent(storyId)}/archive`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(options),
  }));
  return { blob: await response.blob(), filename: filename(response, `${storyId}-oneday.zip`) };
}

export async function importArchive(file: File): Promise<ImportResult> {
  const body = new FormData();
  body.append("file", file);
  const response = await checked(await fetch("/api/stories/import", { method: "POST", body }));
  return response.json() as Promise<ImportResult>;
}

export async function exportTemplate(storyId: string): Promise<{ text: string; filename: string }> {
  const response = await checked(await fetch(`/api/stories/${encodeURIComponent(storyId)}/world-template`));
  return { text: await response.text(), filename: filename(response, `${storyId}-world.oneday.json`) };
}

export async function importTemplate(text: string): Promise<ImportResult> {
  const response = await checked(await fetch("/api/stories/import-template", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: text,
  }));
  return response.json() as Promise<ImportResult>;
}
