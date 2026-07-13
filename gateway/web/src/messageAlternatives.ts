import type { MessageView, TimelineBranchView, TimelineResponse } from "./types";

export interface MessageAlternativeSet {
  decisionCommitId: string;
  branches: TimelineBranchView[];
  currentIndex: number;
  atDecision: boolean;
}

interface TimelineIndexes {
  branchesById: Map<string, TimelineBranchView>;
  commitsById: Map<string, TimelineResponse["commits"][number]>;
  forwardBranchesByDecision: Map<string, TimelineBranchView[]>;
  materializedForksByDecision: Map<string, TimelineBranchView[]>;
}

export function messageAlternativesForCommit(
  sourceCommitId: string,
  timeline: TimelineResponse,
): MessageAlternativeSet {
  return messageAlternativesForCommitWithIndexes(sourceCommitId, timeline, buildTimelineIndexes(timeline));
}

function messageAlternativesForCommitWithIndexes(
  sourceCommitId: string,
  timeline: TimelineResponse,
  indexes: TimelineIndexes,
): MessageAlternativeSet {
  const activeBranch = indexes.branchesById.get(timeline.active_branch_id);
  const atDecision =
    timeline.head?.id === sourceCommitId &&
    activeBranch?.head_commit_id === sourceCommitId &&
    activeBranch.fork_commit_id === sourceCommitId;
  if (atDecision) {
    const branches = indexes.forwardBranchesByDecision.get(sourceCommitId) ?? [];
    return { decisionCommitId: sourceCommitId, branches, currentIndex: -1, atDecision: true };
  }

  const outcomeCommit = indexes.commitsById.get(sourceCommitId);
  const decisionCommitId = outcomeCommit?.parent_commit_id ?? "";
  const decisionCommit = indexes.commitsById.get(decisionCommitId);
  if (!outcomeCommit || !decisionCommitId || !decisionCommit) {
    return { decisionCommitId, branches: [], currentIndex: -1, atDecision: false };
  }

  const originalBranch = indexes.branchesById.get(decisionCommit.branch_id);
  const materializedForks = (indexes.materializedForksByDecision.get(decisionCommitId) ?? [])
    .filter((branch) => branch.id !== originalBranch?.id);
  const branches = originalBranch ? [originalBranch, ...materializedForks] : materializedForks;

  return {
    decisionCommitId,
    branches,
    currentIndex: branches.findIndex((branch) => branch.id === outcomeCommit.branch_id),
    atDecision: false,
  };
}

function buildTimelineIndexes(timeline: TimelineResponse): TimelineIndexes {
  const branchesById = new Map(timeline.branches.map((branch) => [branch.id, branch]));
  const commitsById = new Map(timeline.commits.map((commit) => [commit.id, commit]));
  const childBranchIdsByDecision = new Map<string, Set<string>>();
  for (const commit of timeline.commits) {
    if (!commit.parent_commit_id) continue;
    const branchIds = childBranchIdsByDecision.get(commit.parent_commit_id) ?? new Set<string>();
    branchIds.add(commit.branch_id);
    childBranchIdsByDecision.set(commit.parent_commit_id, branchIds);
  }

  const forwardBranchesByDecision = new Map<string, TimelineBranchView[]>();
  for (const [decisionCommitId, branchIds] of childBranchIdsByDecision) {
    const branches = [...branchIds]
      .map((branchId) => branchesById.get(branchId))
      .filter((branch): branch is TimelineBranchView => branch !== undefined && branch.head_commit_id !== decisionCommitId)
      .sort((left, right) => {
        const leftIsFork = left.fork_commit_id === decisionCommitId ? 1 : 0;
        const rightIsFork = right.fork_commit_id === decisionCommitId ? 1 : 0;
        return leftIsFork - rightIsFork || left.created_at.localeCompare(right.created_at) || left.id.localeCompare(right.id);
      });
    forwardBranchesByDecision.set(decisionCommitId, branches);
  }

  const materializedForksByDecision = new Map<string, TimelineBranchView[]>();
  for (const branch of timeline.branches) {
    if (!branch.fork_commit_id || branch.head_commit_id === branch.fork_commit_id) continue;
    const forks = materializedForksByDecision.get(branch.fork_commit_id) ?? [];
    forks.push(branch);
    materializedForksByDecision.set(branch.fork_commit_id, forks);
  }
  for (const forks of materializedForksByDecision.values()) {
    forks.sort((left, right) => left.created_at.localeCompare(right.created_at) || left.id.localeCompare(right.id));
  }

  return { branchesById, commitsById, forwardBranchesByDecision, materializedForksByDecision };
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
  const indexes = buildTimelineIndexes(timeline);

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
    const alternatives = messageAlternativesForCommitWithIndexes(sourceCommitId, timeline, indexes);
    const commit = indexes.commitsById.get(sourceCommitId);
    const canRestore = Boolean(commit?.parent_commit_id);
    const canSwitch = alternatives.atDecision
      ? alternatives.branches.length > 0
      : alternatives.branches.length > 1 && alternatives.currentIndex >= 0;

    let latestUser: MessageView | undefined;
    let latestAssistant: MessageView | undefined;
    for (let index = group.length - 1; index >= 0 && (!latestUser || !latestAssistant); index -= 1) {
      const message = group[index];
      if (!latestUser && message.role === "user") latestUser = message;
      if (!latestAssistant && message.role === "assistant") latestAssistant = message;
    }
    if (canRestore) place(latestUser ?? latestAssistant, "restore");
    if (canSwitch) place(latestAssistant ?? latestUser, "switcher");
  }
  return placements;
}
