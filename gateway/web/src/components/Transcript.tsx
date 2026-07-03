import { useEffect, useMemo, useRef } from "react";
import { compactText, messageClock } from "../format";
import type { MessageView } from "../types";

interface TranscriptProps {
  messages: MessageView[];
  hiddenBeforeId: number;
}

export function Transcript({ messages, hiddenBeforeId }: TranscriptProps) {
  const ref = useRef<HTMLDivElement>(null);
  const visibleMessages = useMemo(
    () => messages.filter((message) => message.id > hiddenBeforeId),
    [messages, hiddenBeforeId],
  );

  useEffect(() => {
    const node = ref.current;
    if (node) node.scrollTop = node.scrollHeight;
  }, [visibleMessages.length]);

  return (
    <div ref={ref} className="transcript" aria-live="polite">
      {visibleMessages.length === 0 ? (
        <div className="empty-copy transcript-empty">
          {messages.length ? "Transcript cleared locally. New canonical messages will appear here." : "Choose a story to load the canonical transcript."}
        </div>
      ) : (
        visibleMessages.map((message) => <TranscriptMessage key={message.id} message={message} />)
      )}
    </div>
  );
}

function TranscriptMessage({ message }: { message: MessageView }) {
  const lines = message.content
    .split(/\n{2,}/)
    .map((line) => line.trim())
    .filter(Boolean);
  const isSystem = message.role === "system" || message.message_type === "state";
  const isUser = message.role === "user";

  return (
    <article className={`transcript-message ${message.role} ${isSystem ? "system-line" : ""}`}>
      <div className="message-stamp">
        <span>[{messageClock(message)}]</span>
        <small>
          {isUser ? "Command" : message.message_type || message.role} - Turn {message.turn}
        </small>
      </div>
      <div className="message-body">
        {lines.length ? (
          lines.map((line, index) => (
            <p key={`${message.id}-${index}`} className={lineLooksQuoted(line) ? "quoted" : undefined}>
              {line}
            </p>
          ))
        ) : (
          <p>{compactText(message.content || "(empty)", 160)}</p>
        )}
      </div>
    </article>
  );
}

function lineLooksQuoted(line: string): boolean {
  return line.startsWith("\"") || line.startsWith("'") || line.includes(" reads:");
}
