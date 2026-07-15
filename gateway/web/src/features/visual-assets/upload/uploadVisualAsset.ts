import type { VisualAssetUploadResponse } from "../../../types";

export interface VisualAssetUploadOptions {
  storyId: string;
  assetId: string;
  file: File;
  selectAfterUpload?: boolean;
  signal?: AbortSignal;
  onProgress?: (progress: number) => void;
}

export function uploadVisualAssetVersion({
  storyId,
  assetId,
  file,
  selectAfterUpload = false,
  signal,
  onProgress,
}: VisualAssetUploadOptions): Promise<VisualAssetUploadResponse> {
  return new Promise((resolve, reject) => {
    const request = new XMLHttpRequest();
    const endpoint = `/api/stories/${encodeURIComponent(storyId)}/visual-assets/${encodeURIComponent(assetId)}/versions/upload`;
    const data = new FormData();
    data.append("file", file, file.name);
    data.append("metadata", JSON.stringify({ selectAfterUpload }));

    const abort = () => request.abort();
    signal?.addEventListener("abort", abort, { once: true });
    request.upload.addEventListener("progress", (event) => {
      if (event.lengthComputable && event.total > 0) onProgress?.(Math.max(0, Math.min(1, event.loaded / event.total)));
    });
    request.addEventListener("load", () => {
      signal?.removeEventListener("abort", abort);
      let payload: unknown = null;
      try { payload = request.responseText ? JSON.parse(request.responseText) : null; } catch { /* handled below */ }
      if (request.status >= 200 && request.status < 300 && payload) {
        onProgress?.(1);
        resolve(payload as VisualAssetUploadResponse);
        return;
      }
      const message = typeof payload === "object" && payload && "error" in payload
        ? String((payload as { error: unknown }).error)
        : `Upload failed (${request.status || 0})`;
      reject(new Error(message));
    });
    request.addEventListener("error", () => reject(new Error("Upload failed because the server could not be reached.")));
    request.addEventListener("abort", () => reject(new DOMException("Upload cancelled", "AbortError")));
    request.open("POST", endpoint);
    request.send(data);
  });
}
