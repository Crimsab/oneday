import { useEffect, useMemo, useRef } from "react";
import { compactText, messageClock } from "../format";
import { MarkdownText } from "./MarkdownText";
import type { MessageView, PendingTurnView } from "../types";

interface TranscriptProps {
  messages: MessageView[];
  hiddenBeforeId: number;
  pendingTurn?: PendingTurnView | null;
}

export function Transcript({ messages, hiddenBeforeId, pendingTurn }: TranscriptProps) {
  const ref = useRef<HTMLDivElement>(null);
  const visibleMessages = useMemo(
    () => messages.filter((message) => message.id > hiddenBeforeId),
    [messages, hiddenBeforeId],
  );

  useEffect(() => {
    const node = ref.current;
    if (node) node.scrollTop = node.scrollHeight;
  }, [pendingTurn?.id, visibleMessages.length]);

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
    </div>
  );
}

function TranscriptMessage({ message }: { message: MessageView }) {
  const isSystem = message.role === "system" || message.message_type === "state";
  const isUser = message.role === "user";
  const content = message.content.trim() || compactText(message.content || "(empty)", 160);

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
      </div>
    </article>
  );
}

function contentLooksQuoted(content: string): boolean {
  return content.startsWith("\"") || content.startsWith("'") || content.includes(" reads:");
}
