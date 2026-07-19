import { ChevronLeft, ChevronRight, RotateCcw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { messageAlternativesForCommit } from "../messageAlternatives";
import type { MessageView, TimelineResponse } from "../types";

interface MessageBranchControlsProps {
  message: MessageView;
  timeline: TimelineResponse | null;
  showRestore: boolean;
  showSwitcher: boolean;
  busy: boolean;
  onCheckout: (branchId: string) => Promise<void>;
  onRestoreDecision: (fromCommitId: string, turn: number) => Promise<void>;
}

export function MessageBranchControls({
  message,
  timeline,
  showRestore,
  showSwitcher,
  busy,
  onCheckout,
  onRestoreDecision,
}: MessageBranchControlsProps) {
  const { t } = useTranslation(["branches", "flow"]);
  if (!timeline || !message.source_commit_id) return null;

  const commit = timeline.commits.find((item) => item.id === message.source_commit_id);
  const restoreFrom = commit?.parent_commit_id || "";
  const alternatives = messageAlternativesForCommit(message.source_commit_id, timeline);
  const index = alternatives.currentIndex;
  const previous = alternatives.atDecision
    ? alternatives.branches.at(-1)
    : alternatives.branches[index - 1];
  const next = alternatives.atDecision
    ? alternatives.branches[0]
    : alternatives.branches[index + 1];
  const showBranchSwitcher = showSwitcher && (alternatives.atDecision
    ? alternatives.branches.length > 0
    : alternatives.branches.length > 1 && index >= 0);

  if (!(showRestore && restoreFrom) && !showBranchSwitcher) return null;

  return (
    <div className="message-branch-controls" role="group" aria-label={t("branches:forTurn", { turn: message.turn })}>
      {showRestore && restoreFrom && (
        <button
          type="button"
          className="restore-decision-button"
          disabled={busy}
          onClick={() => void onRestoreDecision(restoreFrom, message.turn)}
          title={t("flow:createBranch", { turn: message.turn })}
        >
          <RotateCcw size={13} aria-hidden="true" />
          {t("flow:tryAnother")}
        </button>
      )}
      {showBranchSwitcher && (
        <div className="message-branch-switcher" aria-label={t("branches:alternatives")}>
          <button
            type="button"
            aria-label={t("branches:previous")}
            title={previous ? `${t("branches:switch")} ${previous.name}` : t("branches:noPrevious")}
            disabled={busy || !previous}
            onClick={() => previous && void onCheckout(previous.id)}
          >
            <ChevronLeft size={16} aria-hidden="true" />
          </button>
          <span title={alternatives.atDecision ? "Open a saved path from this restored decision" : alternatives.branches[index]?.name || "Displayed branch"}>
            {alternatives.atDecision ? `${alternatives.branches.length} saved` : `${index + 1}/${alternatives.branches.length}`}
          </span>
          <button
            type="button"
            aria-label={t("branches:next")}
            title={next ? `${t("branches:switch")} ${next.name}` : t("branches:noNext")}
            disabled={busy || !next}
            onClick={() => next && void onCheckout(next.id)}
          >
            <ChevronRight size={16} aria-hidden="true" />
          </button>
        </div>
      )}
    </div>
  );
}
