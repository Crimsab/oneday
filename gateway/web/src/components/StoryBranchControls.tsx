import { ChevronLeft, ChevronRight, GitBranch, RotateCcw } from "lucide-react";
import type { TimelineResponse } from "../types";

interface StoryBranchControlsProps {
  timeline: TimelineResponse | null;
  busy: boolean;
  onCheckout: (branchId: string) => Promise<void>;
  onRevisitChoice: () => Promise<void>;
}

export function StoryBranchControls({ timeline, busy, onCheckout, onRevisitChoice }: StoryBranchControlsProps) {
  if (!timeline) return null;
  const activeIndex = timeline.branches.findIndex((branch) => branch.id === timeline.active_branch_id);
  const index = activeIndex < 0 ? 0 : activeIndex;
  const previous = timeline.branches[index - 1];
  const next = timeline.branches[index + 1];
  const canRevisit = Boolean(timeline.head?.parent_commit_id);

  return (
    <nav className="story-branch-controls" aria-label="Story branch navigation">
      <button
        type="button"
        className="revisit-choice-button"
        disabled={busy || !canRevisit}
        onClick={() => void onRevisitChoice()}
        title="Return to the previous decision on a new branch"
      >
        <RotateCcw size={14} />
        Try another choice
      </button>
      <span className="branch-position" title={timeline.branches[index]?.name || "Active branch"}>
        <GitBranch size={13} />
        {index + 1}/{Math.max(1, timeline.branches.length)}
      </span>
      <button
        type="button"
        className="branch-step-button"
        aria-label="Previous story branch"
        title={previous ? `Switch to ${previous.name}` : "No previous branch"}
        disabled={busy || !previous}
        onClick={() => previous && void onCheckout(previous.id)}
      >
        <ChevronLeft size={17} />
      </button>
      <button
        type="button"
        className="branch-step-button"
        aria-label="Next story branch"
        title={next ? `Switch to ${next.name}` : "No next branch"}
        disabled={busy || !next}
        onClick={() => next && void onCheckout(next.id)}
      >
        <ChevronRight size={17} />
      </button>
    </nav>
  );
}
