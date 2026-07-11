import type { VisualAssetsResponse, VisualGenerationJobView } from "./types";

const ACTIVE_VISUAL_JOB_STATUSES = new Set(["queued", "running"]);
const ACTIVE_VISUAL_ASSET_STATUSES = new Set(["queued", "running"]);

export function isActiveVisualJob(job: VisualGenerationJobView): boolean {
  return ACTIVE_VISUAL_JOB_STATUSES.has(job.status);
}

export function hasActiveVisualGeneration(
  visualAssets: VisualAssetsResponse | null,
): boolean {
  if (!visualAssets) return false;
  return (
    visualAssets.jobs.some(isActiveVisualJob) ||
    visualAssets.assets.some((asset) =>
      ACTIVE_VISUAL_ASSET_STATUSES.has(asset.status),
    )
  );
}

export function visualPollingDelayMs(
  visualAssets: VisualAssetsResponse | null,
): number {
  return hasActiveVisualGeneration(visualAssets) ? 2500 : 0;
}
