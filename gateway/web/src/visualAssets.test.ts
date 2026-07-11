import { describe, expect, it } from "vitest";
import { characterAsset, readyAssetUrl, visualCatalog } from "./visualAssets";
import type { RecordView, StorySnapshot, VisualAsset, VisualAssetsResponse } from "./types";

describe("visual canon selection", () => {
  it("prefers the current canonical form over an older ready portrait", () => {
    const oldReady = asset({ id: "old", form_id: "form-old", gate_state: "established_canonical", status: "ready", url: "/old.png" });
    const currentForm = asset({ id: "current", form_id: "form-new", gate_state: "form_changed", status: "pending", url: "" });
    const catalog = visualCatalog(response([oldReady, currentForm]), snapshot());

    expect(characterAsset(catalog, npc())).toMatchObject({ id: "current", form_id: "form-new" });
    expect(readyAssetUrl(characterAsset(catalog, npc()))).toBe("");
  });

  it("suppresses a stale portrait while canonical identity is contradictory", () => {
    const oldReady = asset({ id: "old", gate_state: "established_canonical", status: "ready", url: "/old.png" });
    const contradiction = asset({ id: "blocked", gate_state: "identity_contradiction", status: "blocked", url: "" });
    const selected = characterAsset(visualCatalog(response([oldReady, contradiction]), snapshot()), npc());

    expect(selected?.id).toBe("blocked");
    expect(readyAssetUrl(selected)).toBe("");
  });

  it("uses a ready legacy image while a non-invalidating canonical replacement is still gated", () => {
    const legacyReady = asset({ id: "legacy", canonical_entity_id: "", entity_id: "", gate_state: "legacy", status: "ready", url: "/legacy.png" });
    const observing = asset({ id: "observing", gate_state: "insufficient_observation", generation_eligible: false, status: "pending", url: "" });
    const catalog = visualCatalog(response([legacyReady, observing]), snapshot());

    expect(characterAsset(catalog, npc())?.id).toBe("legacy");
    expect(readyAssetUrl(characterAsset(catalog, npc()))).toBe("/legacy.png");
    expect(catalog.assets).toHaveLength(1);
  });

  it("catalogs the generated map art by canonical location", () => {
    const background = asset({ id: "map", kind: "map_background", entity_id: "", canonical_entity_id: "", status: "ready", url: "/map.png" });
    const icon = asset({ id: "harbor-icon", kind: "map_icon", subject: "Harbor", entity_id: "", canonical_entity_id: "", canonical_location_id: "loc-harbor", status: "ready", url: "/harbor.png" });
    const pendingIcon = asset({ id: "court-icon", kind: "map_icon", subject: "Court", entity_id: "", canonical_entity_id: "", canonical_location_id: "loc-court", status: "pending", url: "" });
    const catalog = visualCatalog(response([background, icon, pendingIcon]), snapshot());

    expect(catalog.mapBackground?.id).toBe("map");
    expect(catalog.mapIcons.get("loc harbor")?.id).toBe("harbor-icon");
    expect(catalog.mapIcons.has("loc court")).toBe(false);
  });
});

function asset(overrides: Partial<VisualAsset>): VisualAsset {
  return {
    id: "asset",
    story_id: "story",
    kind: "character",
    subject: "Mara",
    entity_id: "npc-mara",
    canonical_entity_id: "npc-mara",
    canonical_location_id: "",
    form_id: "form-current",
    lineage_key: "npc-mara:form-current",
    appearance_fingerprint: "appearance",
    profile_revision_id: "profile-1",
    canon_status: "canonical",
    gate_state: "established_canonical",
    gate_reason: "Known identity",
    generation_eligible: true,
    prompt: "portrait",
    negative_prompt: "",
    status: "pending",
    url: "",
    provider: "",
    source: "",
    error: "",
    turn: 2,
    branch_id: "main",
    source_commit_id: "commit-2",
    selected_version_id: null,
    can_undo_selection: false,
    can_redo_selection: false,
    inherited: false,
    updated_at: "2026-07-11T00:00:00Z",
    ...overrides,
  };
}

function response(assets: VisualAsset[]): VisualAssetsResponse {
  return {
    profile: {
      id: "profile-1",
      story_id: "story",
      revision: 1,
      fingerprint: "profile-fingerprint",
      branch_id: "main",
      source_commit_id: "commit-2",
      world_style_prompt: "",
      character_style_prompt: "",
      negative_prompt: "",
      palette: "",
      updated_at: "2026-07-11T00:00:00Z",
    },
    assets,
    jobs: [],
  };
}

function snapshot(): StorySnapshot {
  return { world: { current_location: "Harbor", current_location_id: "loc-harbor" }, panels: { npcs: [npc()] } } as unknown as StorySnapshot;
}

function npc(): RecordView {
  return { id: "npc-mara", name: "Mara", fields: {} };
}
