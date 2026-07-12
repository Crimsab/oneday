import type { TimelineBranchView, TimelineResponse } from "./types";

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
