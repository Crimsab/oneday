import { ChevronLeft, ChevronRight, GitBranch, RotateCcw } from "lucide-react";
import type { MessageView, TimelineResponse } from "../types";

interface MessageBranchControlsProps {
  message: MessageView;
  timeline: TimelineResponse | null;
  busy: boolean;
  onCheckout: (branchId: string) => Promise<void>;
  onRestoreDecision: (fromCommitId: string, turn: number) => Promise<void>;
}

export function MessageBranchControls({
  message,
  timeline,
  busy,
  onCheckout,
  onRestoreDecision,
}: MessageBranchControlsProps) {
  if (!timeline || !message.source_commit_id) return null;

  const commit = timeline.commits.find((item) => item.id === message.source_commit_id);
  const restoreFrom = commit?.parent_commit_id || "";
  const activeIndex = timeline.branches.findIndex((branch) => branch.id === timeline.active_branch_id);
  const index = Math.max(0, activeIndex);
  const previous = timeline.branches[index - 1];
  const next = timeline.branches[index + 1];
  const showBranchSwitcher = timeline.branches.length > 1 && timeline.head?.id === message.source_commit_id;

  if (!restoreFrom && !showBranchSwitcher) return null;

  return (
    <nav className="message-branch-controls" aria-label={`Story alternatives for turn ${message.turn}`}>
      {restoreFrom && (
        <button
          type="button"
          className="restore-decision-button"
          disabled={busy}
          onClick={() => void onRestoreDecision(restoreFrom, message.turn)}
          title={`Create a new branch from before turn ${message.turn}`}
        >
          <RotateCcw size={13} aria-hidden="true" />
          Try another path from here
        </button>
      )}
      {showBranchSwitcher && (
        <div className="message-branch-switcher" aria-label="Available story branches">
          <GitBranch size={13} aria-hidden="true" />
          <span title={timeline.branches[index]?.name || "Active branch"}>{index + 1}/{timeline.branches.length}</span>
          <button
            type="button"
            aria-label="Previous story branch"
            title={previous ? `Switch to ${previous.name}` : "No previous branch"}
            disabled={busy || !previous}
            onClick={() => previous && void onCheckout(previous.id)}
          >
            <ChevronLeft size={16} aria-hidden="true" />
          </button>
          <button
            type="button"
            aria-label="Next story branch"
            title={next ? `Switch to ${next.name}` : "No next branch"}
            disabled={busy || !next}
            onClick={() => next && void onCheckout(next.id)}
          >
            <ChevronRight size={16} aria-hidden="true" />
          </button>
        </div>
      )}
    </nav>
  );
}
