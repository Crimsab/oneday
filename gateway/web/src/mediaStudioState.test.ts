import { describe, expect, it } from "vitest";
import { defaultMediaAssetFilters, filterMediaAssets, mediaActivity } from "./mediaStudioState";
import type { VisualAsset, VisualGenerationJobView } from "./types";

describe("media studio filters", () => {
  it("filters available metadata and keeps an explicit canonical fallback", () => {
    const assets = [asset({ id: "mara", subject: "Mara", kind: "character", canon_status: "canonical", canonical_entity_id: "mara" }), asset({ id: "map", subject: "Harbor map", kind: "map_background", canon_status: "draft", canonical_location_id: "harbor" })];
    expect(filterMediaAssets(assets, { ...defaultMediaAssetFilters, query: "harbor", kind: "map_background" }).map(({ id }) => id)).toEqual(["map"]);
    expect(filterMediaAssets(assets, { ...defaultMediaAssetFilters, canonical: "canonical" }).map(({ id }) => id)).toEqual(["mara"]);
  });

  it("puts recoverable activity ahead of completed history", () => {
    expect(mediaActivity([job({ id: 3, status: "failed" }), job({ id: 2, status: "queued" }), job({ id: 1, status: "running" })]).map(({ id }) => id)).toEqual([1, 2, 3]);
  });
});

function asset(overrides: Partial<VisualAsset>): VisualAsset {
  return { id: "asset", story_id: "story", kind: "location", subject: "Market", entity_id: "", canonical_entity_id: "", canonical_location_id: "", form_id: "", lineage_key: "", appearance_fingerprint: "", profile_revision_id: "", canon_status: "draft", gate_state: "", gate_reason: "", generation_eligible: true, prompt: "", negative_prompt: "", status: "ready", url: "", provider: "", source: "", error: "", turn: 1, branch_id: "", source_commit_id: "", can_undo_selection: false, can_redo_selection: false, inherited: false, updated_at: "2026-07-19T12:00:00Z", ...overrides };
}

function job(overrides: Partial<VisualGenerationJobView>): VisualGenerationJobView {
  return { id: 0, asset_id: "asset", story_id: "story", canonical_entity_id: "", canonical_location_id: "", form_id: "", appearance_fingerprint: "", profile_revision_id: "", status: "succeeded", attempts: 1, max_attempts: 1, locked_until: "", error: "", provider: "", started_at: "", finished_at: "", created_at: "2026-07-19T12:00:00Z", updated_at: "2026-07-19T12:00:00Z", branch_id: "", source_commit_id: "", ...overrides };
}
