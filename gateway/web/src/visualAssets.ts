import type { RecordView, StorySnapshot, VisualAsset, VisualAssetsResponse, VisualProfile } from "./types";

export interface VisualCatalog {
  profile: VisualProfile | null;
  assets: VisualAsset[];
  location: VisualAsset | null;
  characters: Map<string, VisualAsset>;
}

export const emptyVisualCatalog: VisualCatalog = {
  profile: null,
  assets: [],
  location: null,
  characters: new Map(),
};

export function visualCatalog(response: VisualAssetsResponse | null, snapshot: StorySnapshot | null): VisualCatalog {
  if (!response) return emptyVisualCatalog;
  const locationId = normalizeKey(snapshot?.world.current_location_id ?? "");
  const locationSubject = normalizeKey(snapshot?.world.current_location ?? "");
  const locationCandidates = response.assets.filter((asset) => asset.kind === "location");
  const exactLocations = locationCandidates.filter(
    (asset) =>
      (locationId && normalizeKey(asset.canonical_location_id) === locationId) ||
      (locationSubject && normalizeKey(asset.subject) === locationSubject),
  );
  const location = bestCanonicalAsset(exactLocations.length ? exactLocations : locationCandidates);

  const characters = new Map<string, VisualAsset>();
  for (const asset of response.assets) {
    if (asset.kind !== "character") continue;
    const key = normalizeKey(asset.canonical_entity_id || asset.entity_id || asset.subject);
    if (!key) continue;
    const existing = characters.get(key);
    if (!existing || canonicalAssetScore(asset) > canonicalAssetScore(existing)) {
      characters.set(key, asset);
    }
  }

  for (const npc of snapshot?.panels.npcs ?? []) {
    const byId = characters.get(normalizeKey(npc.id));
    const byName = characters.get(normalizeKey(npc.name));
    if (!byId && byName) characters.set(normalizeKey(npc.id), byName);
  }

  return {
    profile: response.profile,
    assets: response.assets,
    location,
    characters,
  };
}

export function characterAsset(catalog: VisualCatalog, npc: RecordView): VisualAsset | null {
  return catalog.characters.get(normalizeKey(npc.id)) ?? catalog.characters.get(normalizeKey(npc.name)) ?? null;
}

export function readyAssetUrl(asset: VisualAsset | null | undefined): string {
  if (!asset || asset.status !== "ready" || !asset.url) return "";
  return asset.url;
}

export function normalizeKey(value: string): string {
  return value.trim().toLowerCase().replace(/[^a-z0-9]+/g, " ").trim();
}

function bestCanonicalAsset(assets: VisualAsset[]): VisualAsset | null {
  return assets.reduce<VisualAsset | null>(
    (best, asset) => (!best || canonicalAssetScore(asset) > canonicalAssetScore(best) ? asset : best),
    null,
  );
}

function canonicalAssetScore(asset: VisualAsset): number {
  const gatePriority: Record<string, number> = {
    identity_contradiction: 120,
    form_changed: 110,
    chapter_milestone: 100,
    meaningful_stay: 95,
    narrative_significance: 90,
    established_canonical: 85,
    explicit_request_available: 80,
    identified_draft: 70,
    silhouette_available: 60,
    insufficient_canon: 50,
    insufficient_observation: 40,
  };
  return (gatePriority[asset.gate_state] ?? 0) * 100 + (asset.generation_eligible ? 10 : 0) + (asset.status === "ready" ? 1 : 0);
}
