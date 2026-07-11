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
  const locationSubject = normalizeKey(snapshot?.world.current_location ?? "");
  const location =
    response.assets.find((asset) => asset.kind === "location" && asset.status === "ready" && normalizeKey(asset.subject) === locationSubject) ??
    response.assets.find((asset) => asset.kind === "location" && asset.status === "ready") ??
    response.assets.find((asset) => asset.kind === "location") ??
    null;

  const characters = new Map<string, VisualAsset>();
  for (const asset of response.assets) {
    if (asset.kind !== "character") continue;
    const key = normalizeKey(asset.entity_id || asset.subject);
    if (!key) continue;
    const existing = characters.get(key);
    if (!existing || (existing.status !== "ready" && asset.status === "ready")) {
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
