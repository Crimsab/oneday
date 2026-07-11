import { useEffect, useMemo, useRef } from "react";
import { compactText, readableStructuredText } from "../format";
import { turnEventDetail, turnEventTitle } from "../turnEvents";
import { MarkdownText } from "./MarkdownText";
import { MessageDiagnostics } from "./MessageDiagnostics";
import { AudioControls } from "./AudioControls";
import { MessageBranchControls } from "./MessageBranchControls";
import type { MessageView, PendingTurnView, TimelineResponse, TurnStreamEvent } from "../types";

interface TranscriptProps {
  storyId: string;
  messages: MessageView[];
  hiddenBeforeId: number;
  pendingTurn?: PendingTurnView | null;
  liveEvents?: TurnStreamEvent[];
  timeline: TimelineResponse | null;
  timelineBusy: boolean;
  onCheckoutBranch: (branchId: string) => Promise<void>;
  onRestoreDecision: (fromCommitId: string, turn: number) => Promise<void>;
}

export function Transcript({ storyId, messages, hiddenBeforeId, pendingTurn, liveEvents = [], timeline, timelineBusy, onCheckoutBranch, onRestoreDecision }: TranscriptProps) {
  const ref = useRef<HTMLDivElement>(null);
  const visibleMessages = useMemo(
    () => messages.filter((message) => message.id > hiddenBeforeId),
    [messages, hiddenBeforeId],
  );
  const latestMessageByCommit = useMemo(() => {
    const latest = new Map<string, number>();
    for (const message of visibleMessages) {
      if (message.role === "assistant" && message.source_commit_id) latest.set(message.source_commit_id, message.id);
    }
    return latest;
  }, [visibleMessages]);

  useEffect(() => {
    const node = ref.current;
    if (node) node.scrollTop = node.scrollHeight;
  }, [liveEvents.length, pendingTurn?.id, visibleMessages.length]);

  return (
    <div ref={ref} className="transcript" aria-live="polite">
      {visibleMessages.length === 0 ? (
        <div className="empty-copy transcript-empty">
          {messages.length ? "Transcript cleared locally. New canonical messages will appear here." : "Choose a story to load the canonical transcript."}
        </div>
      ) : (
        visibleMessages.map((message) => <TranscriptMessage key={message.id} storyId={storyId} message={message} showTimelineControls={latestMessageByCommit.get(message.source_commit_id) === message.id} timeline={timeline} timelineBusy={timelineBusy} onCheckoutBranch={onCheckoutBranch} onRestoreDecision={onRestoreDecision} />)
      )}
      {pendingTurn && <PendingTurnMessage pendingTurn={pendingTurn} />}
      {liveEvents.length > 0 && <TurnEventStream events={liveEvents} />}
    </div>
  );
}

function TranscriptMessage({ storyId, message, showTimelineControls, timeline, timelineBusy, onCheckoutBranch, onRestoreDecision }: { storyId: string; message: MessageView; showTimelineControls: boolean; timeline: TimelineResponse | null; timelineBusy: boolean; onCheckoutBranch: (branchId: string) => Promise<void>; onRestoreDecision: (fromCommitId: string, turn: number) => Promise<void> }) {
  const isSystem = message.role === "system" || message.message_type === "state";
  const isUser = message.role === "user";
  const content = readableStructuredText(message.content) || compactText(message.content || "(empty)", 160);
	const dialogue = dialogueBlocksFromMessage(message);

  return (
    <article className={`transcript-message ${message.role} ${isSystem ? "system-line" : ""}`}>
      <div className="message-stamp">
        <span>Turn {message.turn}</span>
        <small>{isUser ? "Your action" : isSystem ? "Story state" : "Narration"}</small>
      </div>
      <div className="message-body">
        <MarkdownText className={contentLooksQuoted(content) ? "quoted" : undefined}>{content}</MarkdownText>
		{dialogue.length > 0 && <div className="dialogue-blocks" aria-label={`Structured dialogue for turn ${message.turn}`}>{dialogue.map((block,index)=><blockquote key={`${block.speakerId || block.speaker}-${index}`}><strong>{block.speaker || "Unknown speaker"}</strong><span>{block.role}</span><p>{block.text}</p></blockquote>)}</div>}
        <MessageDiagnostics message={message} />
        {message.role === "assistant" && Boolean(message.source_commit_id) && <AudioControls storyId={storyId} messageId={message.id} />}
        {message.role === "assistant" && showTimelineControls && (
          <MessageBranchControls message={message} timeline={timeline} busy={timelineBusy} onCheckout={onCheckoutBranch} onRestoreDecision={onRestoreDecision} />
        )}
      </div>
    </article>
  );
}

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
    <article className="transcript-message user pending-turn">
      <div className="message-stamp">
        <span>Turn {pendingTurn.turn}</span>
        <small>Your action · sending</small>
      </div>
      <div className="message-body">
        <MarkdownText>{pendingTurn.source}</MarkdownText>
        <div className="pending-assistant">
          <span className="pending-pulse" aria-hidden="true" />
          <span>{pendingTurn.detail}</span>
        </div>
        {pendingTurn.streamingText && (
          <div className="pending-streaming-draft">
            <small>Provisional assistant draft</small>
            <MarkdownText>{pendingTurn.streamingText}</MarkdownText>
          </div>
        )}
      </div>
    </article>
  );
}

function TurnEventStream({ events }: { events: TurnStreamEvent[] }) {
  return (
    <aside className="turn-event-stream" aria-label="Live turn events">
      <div className="turn-event-stream-head">
        <span className="pending-pulse" aria-hidden="true" />
		<strong>Turn progress</strong>
      </div>
      {events.map((event) => (
        <div className={`turn-event-row ${event.status}`} key={`${event.created_at}-${event.event_type ?? event.status}`}>
          <span>{turnEventTitle(event)}</span>
          <small>{turnEventDetail(event)}</small>
        </div>
      ))}
    </aside>
  );
}

function contentLooksQuoted(content: string): boolean {
  return content.startsWith("\"") || content.startsWith("'") || content.includes(" reads:");
}
