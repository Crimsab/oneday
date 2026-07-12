import { ChevronLeft, ChevronRight, GitBranch, RotateCcw } from "lucide-react";
import { messageAlternativesForCommit } from "../messageAlternatives";
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
  const alternatives = messageAlternativesForCommit(message.source_commit_id, timeline);
  const index = alternatives.currentIndex;
  const previous = alternatives.branches[index - 1];
  const next = alternatives.branches[index + 1];
  const showBranchSwitcher = alternatives.branches.length > 1 && index >= 0;

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
        <div className="message-branch-switcher" aria-label="Available story alternatives">
          <GitBranch size={13} aria-hidden="true" />
          <span title={alternatives.branches[index]?.name || "Displayed branch"}>{index + 1}/{alternatives.branches.length}</span>
          <button
            type="button"
            aria-label="Previous alternative"
            title={previous ? `Switch to ${previous.name}` : "No previous branch"}
            disabled={busy || !previous}
            onClick={() => previous && void onCheckout(previous.id)}
          >
            <ChevronLeft size={16} aria-hidden="true" />
          </button>
          <button
            type="button"
            aria-label="Next alternative"
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
