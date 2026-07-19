import type { VisualAsset, VisualGenerationJobView } from "./types";

export type MediaStudioTab = "library" | "create" | "activity";
export type MediaAssetSort = "recent" | "turn" | "name";

export interface MediaAssetFilters {
  query: string;
  kind: string;
  status: string;
  location: string;
  entity: string;
  turn: string;
  canonical: "all" | "canonical" | "draft";
  sort: MediaAssetSort;
}

export const defaultMediaAssetFilters: MediaAssetFilters = {
  query: "",
  kind: "all",
  status: "all",
  location: "all",
  entity: "all",
  turn: "all",
  canonical: "all",
  sort: "recent",
};

export function filterMediaAssets(
  assets: VisualAsset[],
  filters: MediaAssetFilters,
): VisualAsset[] {
  const query = filters.query.trim().toLocaleLowerCase();
  return assets
    .filter((asset) => {
      const searchable = [
        asset.subject,
        asset.kind,
        asset.canonical_location_id,
        asset.canonical_entity_id,
        asset.provider,
        asset.canon_status,
        asset.status,
        String(asset.turn),
      ].join(" ").toLocaleLowerCase();
      return (!query || searchable.includes(query))
        && (filters.kind === "all" || asset.kind === filters.kind)
        && (filters.status === "all" || asset.status === filters.status)
        && (filters.location === "all" || asset.canonical_location_id === filters.location)
        && (filters.entity === "all" || asset.canonical_entity_id === filters.entity)
        && (filters.turn === "all" || String(asset.turn) === filters.turn)
        && (filters.canonical === "all" || asset.canon_status === filters.canonical);
    })
    .sort((left, right) => compareMediaAssets(left, right, filters.sort));
}

export function mediaActivity(
  jobs: VisualGenerationJobView[],
): VisualGenerationJobView[] {
  return [...jobs].sort((left, right) => {
    const active = (status: string) => status === "running" ? 0 : status === "queued" ? 1 : 2;
    return active(left.status) - active(right.status)
      || Date.parse(right.updated_at || right.created_at) - Date.parse(left.updated_at || left.created_at);
  });
}

function compareMediaAssets(left: VisualAsset, right: VisualAsset, sort: MediaAssetSort): number {
  if (sort === "name") return left.subject.localeCompare(right.subject);
  if (sort === "turn") return right.turn - left.turn || left.subject.localeCompare(right.subject);
  return Date.parse(right.updated_at) - Date.parse(left.updated_at) || right.turn - left.turn;
}
