import { describe, expect, it } from "vitest";
import { hasActiveVisualGeneration, visualPollingDelayMs } from "./visualJobs";
import type { VisualAssetsResponse } from "./types";

describe("visual job helpers", () => {
  it("activates polling when jobs are queued or running", () => {
    const response = visualResponse({
      jobs: [{ id: 1, status: "queued" }],
    });

    expect(hasActiveVisualGeneration(response)).toBe(true);
    expect(visualPollingDelayMs(response)).toBe(2500);
  });

  it("activates polling when assets are still queued or running", () => {
    const response = visualResponse({
      assets: [{ id: "asset-1", status: "running" }],
    });

    expect(hasActiveVisualGeneration(response)).toBe(true);
  });

  it("stops polling when visual generation is terminal", () => {
    const response = visualResponse({
      assets: [{ id: "asset-1", status: "ready" }],
      jobs: [{ id: 1, status: "succeeded" }],
    });

    expect(hasActiveVisualGeneration(response)).toBe(false);
    expect(visualPollingDelayMs(response)).toBe(0);
  });

  it("keeps polling while a native image operation is queued", () => {
    const response = visualResponse({});
    response.operations = [{
      id: "op-1",
      asset_id: "asset-1",
      operation: "inpaint",
      status: "queued",
      provider: "openai",
      model: "gpt-image-2",
      endpoint_id: "/images/edits",
      source_version_id: 1,
      mask_id: "mask-1",
      result_version_id: null,
      branch_id: "main",
      error_code: "",
      error_summary: "",
      created_at: "",
      updated_at: "",
    }];
    expect(visualPollingDelayMs(response)).toBe(2500);
  });
});

function visualResponse({
  assets = [],
  jobs = [],
}: {
  assets?: Array<{ id: string; status: string }>;
  jobs?: Array<{ id: number; status: string }>;
}): VisualAssetsResponse {
  return {
    profile: {
      id: "profile",
      story_id: "story",
      revision: 1,
      fingerprint: "fingerprint",
      branch_id: "main",
      source_commit_id: "commit",
      world_style_prompt: "",
      character_style_prompt: "",
      negative_prompt: "",
      palette: "",
      updated_at: "",
    },
    assets: assets.map((asset) => ({
      id: asset.id,
      story_id: "story",
      kind: "location",
      subject: "Station",
      entity_id: "",
      canonical_entity_id: "",
      canonical_location_id: "location",
      form_id: "",
      lineage_key: "location",
      appearance_fingerprint: "fingerprint",
      profile_revision_id: "profile",
      canon_status: "canonical",
      gate_state: "meaningful_stay",
      gate_reason: "Eligible",
      generation_eligible: true,
      prompt: "",
      negative_prompt: "",
      status: asset.status,
      url: "",
      provider: "",
      source: "",
      error: "",
      turn: 1,
      branch_id: "main",
      source_commit_id: "commit",
      selected_version_id: null,
      can_undo_selection: false,
      can_redo_selection: false,
      inherited: false,
      updated_at: "",
    })),
    jobs: jobs.map((job) => ({
      id: job.id,
      asset_id: "asset-1",
      story_id: "story",
      canonical_entity_id: "",
      canonical_location_id: "location",
      form_id: "",
      appearance_fingerprint: "fingerprint",
      profile_revision_id: "profile",
      status: job.status,
      attempts: 0,
      max_attempts: 3,
      locked_until: "",
      error: "",
      provider: "",
      started_at: "",
      finished_at: "",
      created_at: "",
      updated_at: "",
      branch_id: "main",
      source_commit_id: "commit",
    })),
  };
}
