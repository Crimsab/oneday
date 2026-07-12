import type { MessageView, TimelineBranchView, TimelineResponse } from "./types";

export interface MessageAlternativeSet {
  decisionCommitId: string;
  branches: TimelineBranchView[];
  currentIndex: number;
  atDecision: boolean;
}

export function messageAlternativesForCommit(
  sourceCommitId: string,
  timeline: TimelineResponse,
): MessageAlternativeSet {
  const activeBranch = timeline.branches.find((branch) => branch.id === timeline.active_branch_id);
  const atDecision =
    timeline.head?.id === sourceCommitId &&
    activeBranch?.head_commit_id === sourceCommitId &&
    activeBranch.fork_commit_id === sourceCommitId;
  if (atDecision) {
    const childBranchIds = new Set(
      timeline.commits
        .filter((commit) => commit.parent_commit_id === sourceCommitId)
        .map((commit) => commit.branch_id),
    );
    const branches = timeline.branches
      .filter((branch) => branch.head_commit_id !== sourceCommitId && childBranchIds.has(branch.id))
      .sort((left, right) => {
        const leftIsFork = left.fork_commit_id === sourceCommitId ? 1 : 0;
        const rightIsFork = right.fork_commit_id === sourceCommitId ? 1 : 0;
        return leftIsFork - rightIsFork || left.created_at.localeCompare(right.created_at) || left.id.localeCompare(right.id);
      });
    return { decisionCommitId: sourceCommitId, branches, currentIndex: -1, atDecision: true };
  }

  const outcomeCommit = timeline.commits.find((commit) => commit.id === sourceCommitId);
  const decisionCommitId = outcomeCommit?.parent_commit_id ?? "";
  const decisionCommit = timeline.commits.find((commit) => commit.id === decisionCommitId);
  if (!outcomeCommit || !decisionCommitId || !decisionCommit) {
    return { decisionCommitId, branches: [], currentIndex: -1, atDecision: false };
  }

  const originalBranch = timeline.branches.find((branch) => branch.id === decisionCommit.branch_id);
  const materializedForks = timeline.branches
    .filter(
      (branch) =>
        branch.id !== originalBranch?.id &&
        branch.fork_commit_id === decisionCommitId &&
        branch.head_commit_id !== decisionCommitId,
    )
    .sort((left, right) => left.created_at.localeCompare(right.created_at) || left.id.localeCompare(right.id));
  const branches = originalBranch ? [originalBranch, ...materializedForks] : materializedForks;

  return {
    decisionCommitId,
    branches,
    currentIndex: branches.findIndex((branch) => branch.id === outcomeCommit.branch_id),
    atDecision: false,
  };
}

export interface TimelineControlPlacement {
  restore: boolean;
  switcher: boolean;
}

export function timelineControlPlacements(
  messages: MessageView[],
  timeline: TimelineResponse | null,
): Map<number, TimelineControlPlacement> {
  if (!timeline) return new Map();

  const messagesByCommit = new Map<string, MessageView[]>();
  for (const message of messages) {
    if (!message.source_commit_id || message.role === "system") continue;
    const group = messagesByCommit.get(message.source_commit_id) ?? [];
    group.push(message);
    messagesByCommit.set(message.source_commit_id, group);
  }

  const placements = new Map<number, TimelineControlPlacement>();
  const place = (message: MessageView | undefined, kind: keyof TimelineControlPlacement) => {
    if (!message) return;
    const current = placements.get(message.id) ?? { restore: false, switcher: false };
    placements.set(message.id, { ...current, [kind]: true });
  };
  for (const [sourceCommitId, group] of messagesByCommit) {
    const alternatives = messageAlternativesForCommit(sourceCommitId, timeline);
    const commit = timeline.commits.find((item) => item.id === sourceCommitId);
    const canRestore = Boolean(commit?.parent_commit_id);
    const canSwitch = alternatives.atDecision
      ? alternatives.branches.length > 0
      : alternatives.branches.length > 1 && alternatives.currentIndex >= 0;

    const latest = (role: MessageView["role"]) => [...group].reverse().find((message) => message.role === role);
    if (canRestore) place(latest("user") ?? latest("assistant"), "restore");
    if (canSwitch) place(latest("assistant") ?? latest("user"), "switcher");
  }
  return placements;
}
