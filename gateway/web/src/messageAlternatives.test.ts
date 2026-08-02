import { describe, expect, it } from "vitest";
import { messageAlternativesForCommit, timelineControlPlacements } from "./messageAlternatives";
import type { MessageView, TimelineResponse } from "./types";

describe("messageAlternativesForCommit", () => {
  it("groups alternatives by their specific decision and ignores empty rollback branches", () => {
    const timeline = abcTimeline();

    const cAlternatives = messageAlternativesForCommit("commit-c", timeline);
    expect(cAlternatives.decisionCommitId).toBe("commit-b");
    expect(cAlternatives.branches.map((branch) => branch.id)).toEqual(["branch-main", "branch-c-alternate"]);
    expect(cAlternatives.currentIndex).toBe(0);

    const bAlternatives = messageAlternativesForCommit("commit-b", timeline);
    expect(bAlternatives.decisionCommitId).toBe("commit-a");
    expect(bAlternatives.branches.map((branch) => branch.id)).toEqual(["branch-main"]);
    expect(bAlternatives.currentIndex).toBe(0);
  });

  it("uses the branch that produced the displayed message instead of the global active branch", () => {
    const timeline = abcTimeline();
    timeline.commits = [
      timeline.commits[0],
      timeline.commits[1],
      { ...timeline.commits[2], id: "commit-c2", branch_id: "branch-c-alternate" },
    ];
    timeline.head = timeline.commits[2];
    timeline.active_branch_id = "branch-c-alternate";

    const alternatives = messageAlternativesForCommit("commit-c2", timeline);
    expect(alternatives.branches.map((branch) => branch.id)).toEqual(["branch-main", "branch-c-alternate"]);
    expect(alternatives.currentIndex).toBe(1);
  });

  it("exposes materialized forward paths on the exact restored decision message", () => {
    const timeline = abcTimeline();
    timeline.commits.push(commit("commit-c2", "branch-c-alternate", "commit-b", 3));
    timeline.active_branch_id = "branch-c-empty";
    timeline.head = commit("commit-b", "branch-main", "commit-a", 2);

    const alternatives = messageAlternativesForCommit("commit-b", timeline);

    expect(alternatives.atDecision).toBe(true);
    expect(alternatives.branches.map((branch) => branch.id)).toEqual(["branch-main", "branch-c-alternate"]);
    expect(alternatives.currentIndex).toBe(-1);
  });

  it("places rollback and branch versions beside the narrator result", () => {
    const timeline = abcTimeline();
    const messages = [
      message(10, "user", "commit-c"),
      message(11, "assistant", "commit-c"),
    ];

    expect([...timelineControlPlacements(messages, timeline)]).toEqual([
      [11, { restore: true, switcher: true }],
    ]);
  });

  it("places the path pager after the final narrator block for an outcome", () => {
    const timeline = abcTimeline();
    const messages = [
      message(10, "user", "commit-c"),
      message(11, "assistant", "commit-c"),
      message(12, "assistant", "commit-c"),
    ];

    expect([...timelineControlPlacements(messages, timeline)]).toEqual([
      [12, { restore: true, switcher: true }],
    ]);
  });

  it("anchors restored forward paths to the narrator when no new prompt exists", () => {
    const timeline = abcTimeline();
    timeline.commits.push(commit("commit-c2", "branch-c-alternate", "commit-b", 3));
    timeline.active_branch_id = "branch-c-empty";
    timeline.head = commit("commit-b", "branch-main", "commit-a", 2);
    const messages = [
      message(20, "user", "commit-b"),
      message(21, "assistant", "commit-b"),
    ];

    expect([...timelineControlPlacements(messages, timeline)]).toEqual([
      [21, { restore: true, switcher: true }],
    ]);
  });

  it("falls back to the narrator for response-only alternatives", () => {
    const timeline = abcTimeline();
    const messages = [message(31, "assistant", "commit-c")];

    expect([...timelineControlPlacements(messages, timeline)]).toEqual([
      [31, { restore: true, switcher: true }],
    ]);
  });

  it("keeps a restored decision switchable when only one saved path exists", () => {
    const timeline = abcTimeline();
    timeline.branches = timeline.branches.filter((branch) => branch.id !== "branch-c-alternate");
    timeline.active_branch_id = "branch-c-empty";
    timeline.head = commit("commit-b", "branch-main", "commit-a", 2);
    const messages = [message(41, "assistant", "commit-b")];

    expect(messageAlternativesForCommit("commit-b", timeline).branches).toHaveLength(1);
    expect([...timelineControlPlacements(messages, timeline)]).toEqual([
      [41, { restore: true, switcher: true }],
    ]);
  });
});

function abcTimeline(): TimelineResponse {
  return {
    active_branch_id: "branch-main",
    revision: 4,
    branches: [
      branch("branch-main", "main", "", "commit-c", "2026-01-01T00:00:00Z"),
      branch("branch-c-alternate", "C alternate", "commit-b", "commit-c2", "2026-01-01T00:01:00Z"),
      branch("branch-c-empty", "C rollback", "commit-b", "commit-b", "2026-01-01T00:02:00Z"),
      branch("branch-b-empty", "B rollback", "commit-a", "commit-a", "2026-01-01T00:03:00Z"),
    ],
    head: commit("commit-c", "branch-main", "commit-b", 3),
    commits: [
      commit("commit-a", "branch-main", "", 1),
      commit("commit-b", "branch-main", "commit-a", 2),
      commit("commit-c", "branch-main", "commit-b", 3),
    ],
  };
}

function branch(id: string, name: string, forkCommitId: string, headCommitId: string, createdAt: string) {
  return {
    id,
    story_id: "story-1",
    name,
    fork_commit_id: forkCommitId,
    head_commit_id: headCommitId,
    head_turn: 3,
    created_at: createdAt,
    updated_at: createdAt,
  };
}

function commit(id: string, branchId: string, parentCommitId: string, turn: number) {
  return {
    id,
    branch_id: branchId,
    parent_commit_id: parentCommitId,
    canonical_turn: turn,
    kind: "turn",
    message: id,
    created_at: `2026-01-01T00:00:0${turn}Z`,
  };
}

function message(id: number, role: "user" | "assistant", sourceCommitId: string): MessageView {
  return {
    id,
    session_id: "session-1",
    story_id: "story-1",
    turn: 3,
    role,
    content: `${role}-${id}`,
    message_type: "narrative",
    metadata: {},
    created_at: "2026-01-01T00:00:03Z",
    branch_id: "branch-main",
    source_commit_id: sourceCommitId,
  };
}
