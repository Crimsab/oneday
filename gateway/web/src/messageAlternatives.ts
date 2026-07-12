import type { TimelineBranchView, TimelineResponse } from "./types";

export interface MessageAlternativeSet {
  decisionCommitId: string;
  branches: TimelineBranchView[];
  currentIndex: number;
}

export function messageAlternativesForCommit(
  sourceCommitId: string,
  timeline: TimelineResponse,
): MessageAlternativeSet {
  const outcomeCommit = timeline.commits.find((commit) => commit.id === sourceCommitId);
  const decisionCommitId = outcomeCommit?.parent_commit_id ?? "";
  const decisionCommit = timeline.commits.find((commit) => commit.id === decisionCommitId);
  if (!outcomeCommit || !decisionCommitId || !decisionCommit) {
    return { decisionCommitId, branches: [], currentIndex: -1 };
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
  };
}
