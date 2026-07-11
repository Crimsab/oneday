import { useEffect, useMemo, useRef } from "react";
import { compactText, messageClock, readableStructuredText } from "../format";
import { turnEventDetail, turnEventTitle } from "../turnEvents";
import { MarkdownText } from "./MarkdownText";
import type { MessageView, PendingTurnView, TurnStreamEvent } from "../types";

interface TranscriptProps {
  messages: MessageView[];
  hiddenBeforeId: number;
  pendingTurn?: PendingTurnView | null;
  liveEvents?: TurnStreamEvent[];
}

export function Transcript({ messages, hiddenBeforeId, pendingTurn, liveEvents = [] }: TranscriptProps) {
  const ref = useRef<HTMLDivElement>(null);
  const visibleMessages = useMemo(
    () => messages.filter((message) => message.id > hiddenBeforeId),
    [messages, hiddenBeforeId],
  );

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
        visibleMessages.map((message) => <TranscriptMessage key={message.id} message={message} />)
      )}
      {pendingTurn && <PendingTurnMessage pendingTurn={pendingTurn} />}
      {liveEvents.length > 0 && <TurnEventStream events={liveEvents} />}
    </div>
  );
}

function TranscriptMessage({ message }: { message: MessageView }) {
  const isSystem = message.role === "system" || message.message_type === "state";
  const isUser = message.role === "user";
  const content = readableStructuredText(message.content) || compactText(message.content || "(empty)", 160);

  return (
    <article className={`transcript-message ${message.role} ${isSystem ? "system-line" : ""}`}>
      <div className="message-stamp">
        <span>[{messageClock(message)}]</span>
        <small>
          {isUser ? "Command" : message.message_type || message.role} - Turn {message.turn}
        </small>
      </div>
      <div className="message-body">
        <MarkdownText className={contentLooksQuoted(content) ? "quoted" : undefined}>{content}</MarkdownText>
      </div>
    </article>
  );
}

function PendingTurnMessage({ pendingTurn }: { pendingTurn: PendingTurnView }) {
  return (
    <article className="transcript-message user pending-turn">
      <div className="message-stamp">
        <span>[now]</span>
        <small>Command - Turn {pendingTurn.turn}</small>
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
        <strong>Live engine</strong>
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
