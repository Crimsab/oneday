import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { compactText, readableStructuredText } from "../format";
import { MarkdownText } from "./MarkdownText";
import { MessageDiagnostics } from "./MessageDiagnostics";
import { AudioControls } from "./AudioControls";
import { MessageBranchControls } from "./MessageBranchControls";
import { timelineControlPlacements } from "../messageAlternatives";
import type { MessageView, PendingTurnView, TimelineResponse } from "../types";

interface TranscriptProps {
  storyId: string;
  messages: MessageView[];
  hiddenBeforeId: number;
  pendingTurn?: PendingTurnView | null;
  timeline: TimelineResponse | null;
  timelineBusy: boolean;
  onCheckoutBranch: (branchId: string) => Promise<void>;
  onRestoreDecision: (fromCommitId: string, turn: number) => Promise<void>;
}

export function Transcript({ storyId, messages, hiddenBeforeId, pendingTurn, timeline, timelineBusy, onCheckoutBranch, onRestoreDecision }: TranscriptProps) {
  const ref = useRef<HTMLDivElement>(null);
  const followLatest = useRef(true);
  const activeStoryId = useRef(storyId);
  const checkoutBranchRef = useRef(onCheckoutBranch);
  const restoreDecisionRef = useRef(onRestoreDecision);
  checkoutBranchRef.current = onCheckoutBranch;
  restoreDecisionRef.current = onRestoreDecision;
  const checkoutBranch = useCallback((branchId: string) => checkoutBranchRef.current(branchId), []);
  const restoreDecision = useCallback((fromCommitId: string, turn: number) => restoreDecisionRef.current(fromCommitId, turn), []);
  const visibleMessages = useMemo(
    () => messages.filter((message) => message.id > hiddenBeforeId),
    [messages, hiddenBeforeId],
  );
  const timelineControls = useMemo(
    () => timelineControlPlacements(visibleMessages, timeline),
    [timeline, visibleMessages],
  );
  const autoplayMessageId = latestAutoplayMessageId(visibleMessages);

  useEffect(() => {
    const node = ref.current;
    if (!node) return;
    if (activeStoryId.current !== storyId) {
      activeStoryId.current = storyId;
      followLatest.current = true;
    }
    if (followLatest.current) node.scrollTop = node.scrollHeight;
  }, [pendingTurn?.id, pendingTurn?.streamingText?.length, storyId, visibleMessages.length]);

  return (
    <div
      ref={ref}
      className="transcript"
      aria-live="polite"
      onScroll={(event) => {
        followLatest.current = isTranscriptNearBottom(event.currentTarget);
      }}
    >
      {visibleMessages.length === 0 ? (
        <div className="empty-copy transcript-empty">
          {messages.length ? "Transcript cleared locally. New canonical messages will appear here." : "Choose a story to load the canonical transcript."}
        </div>
      ) : (
        visibleMessages.map((message) => <TranscriptMessage key={message.id} storyId={storyId} message={message} autoplay={message.id === autoplayMessageId} timelineControls={timelineControls.get(message.id)} timeline={timeline} timelineBusy={timelineBusy} onCheckoutBranch={checkoutBranch} onRestoreDecision={restoreDecision} />)
      )}
      {pendingTurn && <PendingTurnMessage pendingTurn={pendingTurn} />}
    </div>
  );
}

export function latestAutoplayMessageId(messages: MessageView[]): number | null {
  for (let index = messages.length - 1; index >= 0; index -= 1) {
    const message = messages[index];
    if (message.role === "assistant" && Boolean(message.source_commit_id)) return message.id;
  }
  return null;
}

export function isTranscriptNearBottom(
  viewport: Pick<HTMLElement, "scrollTop" | "scrollHeight" | "clientHeight">,
  threshold = 96,
): boolean {
  return viewport.scrollHeight - viewport.clientHeight - viewport.scrollTop <= threshold;
}

const TranscriptMessage = memo(function TranscriptMessage({ storyId, message, autoplay, timelineControls, timeline, timelineBusy, onCheckoutBranch, onRestoreDecision }: { storyId: string; message: MessageView; autoplay: boolean; timelineControls?: { restore: boolean; switcher: boolean }; timeline: TimelineResponse | null; timelineBusy: boolean; onCheckoutBranch: (branchId: string) => Promise<void>; onRestoreDecision: (fromCommitId: string, turn: number) => Promise<void> }) {
  const [audioOpen, setAudioOpen] = useState(false);
  const isSystem = message.role === "system" || message.message_type === "state";
  const isUser = message.role === "user";
  const content = readableStructuredText(message.content) || compactText(message.content || "(empty)", 160);
	const dialogue = dialogueBlocksFromMessage(message);

  return (
    <article className={`transcript-message ${message.role} ${isSystem ? "system-line" : ""}`}>
      <div className="message-stamp">
        <span>{isUser ? "You" : isSystem ? "System" : "Narrator"}</span>
        <small>Turn {message.turn}</small>
      </div>
      <div className="message-body">
        <MarkdownText className={contentLooksQuoted(content) ? "quoted" : undefined}>{content}</MarkdownText>
		{dialogue.length > 0 && <div className="dialogue-blocks" aria-label={`Structured dialogue for turn ${message.turn}`}>{dialogue.map((block,index)=><blockquote key={`${block.speakerId || block.speaker}-${index}`}><strong>{block.speaker || "Unknown speaker"}</strong><span>{block.role}</span><p>{block.text}</p></blockquote>)}</div>}
        <MessageDiagnostics message={message} />
        {message.role === "assistant" && Boolean(message.source_commit_id) && (
          autoplay || audioOpen
            ? <AudioControls storyId={storyId} messageId={message.id} autoplay={autoplay} />
            : <section className="message-audio" aria-label="Spoken audio"><div className="message-audio-head"><button type="button" onClick={() => setAudioOpen(true)}>Load spoken audio</button></div></section>
        )}
        {timelineControls && (
          <MessageBranchControls message={message} timeline={timeline} showRestore={timelineControls.restore} showSwitcher={timelineControls.switcher} busy={timelineBusy} onCheckout={onCheckoutBranch} onRestoreDecision={onRestoreDecision} />
        )}
      </div>
    </article>
  );
});

export interface DialogueView { speakerId:string; speaker:string; role:string; text:string }
export function dialogueBlocksFromMessage(message:MessageView):DialogueView[] {
	const metadata = message.metadata && typeof message.metadata === "object" && !Array.isArray(message.metadata) ? message.metadata : {};
	const output = metadata.output && typeof metadata.output === "object" && !Array.isArray(metadata.output) ? metadata.output : {};
	const blocks = Array.isArray(output.dialogue_blocks) ? output.dialogue_blocks : [];
	return blocks.flatMap((value) => {
		if (!value || typeof value !== "object" || Array.isArray(value)) return [];
		const text = typeof value.text === "string" ? value.text.trim() : "";
		if (!text) return [];
		return [{ speakerId: typeof value.speaker_id === "string" ? value.speaker_id : "", speaker: typeof value.speaker === "string" ? value.speaker : "", role: typeof value.role === "string" ? value.role : "speaker", text }];
	});
}

function PendingTurnMessage({ pendingTurn }: { pendingTurn: PendingTurnView }) {
  return (
    <>
      <article className="transcript-message user pending-turn-user">
        <div className="message-stamp">
          <span>You</span>
          <small>Turn {pendingTurn.turn}</small>
        </div>
        <div className="message-body">
          <MarkdownText>{pendingTurn.source}</MarkdownText>
        </div>
      </article>
      <article className="transcript-message assistant pending-narrator" aria-busy="true" aria-label={pendingTurn.detail}>
        <div className="message-stamp">
          <span>Narrator</span>
          <small>Writing</small>
        </div>
        <div className="message-body">
          {pendingTurn.streamingText ? (
            <div className="pending-streaming-text">
              <MarkdownText>{pendingTurn.streamingText}</MarkdownText>
            </div>
          ) : (
            <div className="narrative-skeleton" aria-hidden="true">
              <span />
              <span />
              <span />
              <span />
            </div>
          )}
          <span className="sr-only">{pendingTurn.detail}</span>
        </div>
      </article>
    </>
  );
}

function contentLooksQuoted(content: string): boolean {
  return content.startsWith("\"") || content.startsWith("'") || content.includes(" reads:");
}
