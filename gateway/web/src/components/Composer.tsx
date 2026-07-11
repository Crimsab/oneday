import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ChevronDown, CornerDownLeft } from "lucide-react";
import { commandSuggestions, type CommandSuggestionContext, type SlashCommandItem } from "../commands";
import { CommandPalette } from "./CommandPalette";
import type { CommandDescriptor } from "../types";

interface ComposerProps {
  draft: string;
  mode: string;
  disabled: boolean;
  notice: string;
  commandDescriptors: CommandDescriptor[];
  commandContext?: CommandSuggestionContext;
  onDraftChange: (value: string) => void;
  onModeChange: (value: string) => void;
  onSubmit: (draftOverride?: string) => void;
  onHistoryStep: (direction: -1 | 1) => string | null;
}

export function Composer({
  draft,
  mode,
  disabled,
  notice,
  commandDescriptors,
  commandContext,
  onDraftChange,
  onModeChange,
  onSubmit,
  onHistoryStep,
}: ComposerProps) {
  const [commandMenuOpen, setCommandMenuOpen] = useState(false);
  const [inlineSuppressed, setInlineSuppressed] = useState(false);
  const [activeIndex, setActiveIndex] = useState(0);
  const textAreaRef = useRef<HTMLTextAreaElement>(null);
  const suggestions = useMemo(
    () => commandSuggestions(draft, commandDescriptors, commandContext),
    [commandContext, commandDescriptors, draft],
  );
  const inlineOpen = draft.trimStart().startsWith("/") && suggestions.length > 0 && !inlineSuppressed;
  const paletteOpen = commandMenuOpen || inlineOpen;
  const paletteVariant = commandMenuOpen ? "full" : "inline";
  const visibleSuggestions = suggestions.slice(0, commandMenuOpen ? 18 : 9);

  const focusComposer = useCallback(() => {
    window.requestAnimationFrame(() => {
      textAreaRef.current?.focus();
      const length = textAreaRef.current?.value.length ?? 0;
      textAreaRef.current?.setSelectionRange(length, length);
    });
  }, []);

  const updateDraft = useCallback(
    (value: string) => {
      setInlineSuppressed(false);
      setActiveIndex(0);
      onDraftChange(value);
    },
    [onDraftChange],
  );

  const openCommandMenu = useCallback(() => {
    setInlineSuppressed(false);
    setCommandMenuOpen(true);
    setActiveIndex(0);
    if (!draft.trimStart().startsWith("/")) {
      onDraftChange("/");
    }
    focusComposer();
  }, [draft, focusComposer, onDraftChange]);

  const pickSuggestion = useCallback(
    (item: SlashCommandItem) => {
      setCommandMenuOpen(false);
      setInlineSuppressed(true);
      onDraftChange(item.value);
      focusComposer();
    },
    [focusComposer, onDraftChange],
  );

  useEffect(() => {
    setActiveIndex(0);
  }, [draft, suggestions.length, commandMenuOpen]);

  useEffect(() => {
    if (activeIndex < visibleSuggestions.length) return;
    setActiveIndex(Math.max(0, visibleSuggestions.length - 1));
  }, [activeIndex, visibleSuggestions.length]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (!(event.ctrlKey || event.metaKey) || event.key.toLowerCase() !== "k") return;
      event.preventDefault();
      openCommandMenu();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [openCommandMenu]);

  const moveActive = (direction: -1 | 1) => {
    if (visibleSuggestions.length === 0) return;
    setActiveIndex((index) => (index + direction + visibleSuggestions.length) % visibleSuggestions.length);
  };

  const submitCurrentDraft = () => {
    onSubmit(textAreaRef.current?.value ?? draft);
  };

  return (
    <form
      className="composer"
      onSubmit={(event) => {
        event.preventDefault();
        submitCurrentDraft();
      }}
    >
      <div className="composer-title"><span>Something else?</span><small>Describe any action in your own words.</small></div>
      <div className="composer-row">
        <span className="prompt-marker">&gt;</span>
        <textarea
          ref={textAreaRef}
          value={draft}
          onChange={(event) => updateDraft(event.target.value)}
          onBeforeInput={(event) => {
            const inputEvent = event.nativeEvent as InputEvent;
            if (paletteOpen && inputEvent.data === "\t") {
              event.preventDefault();
            }
          }}
          onKeyDown={(event) => {
            if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "k") {
              event.preventDefault();
              openCommandMenu();
              return;
            }
            if ((event.ctrlKey || event.metaKey) && event.key === "Enter") {
              event.preventDefault();
              submitCurrentDraft();
              return;
            }
            if ((event.key === "Enter" || event.key === "Tab") && paletteOpen && visibleSuggestions[activeIndex]) {
              event.preventDefault();
              pickSuggestion(visibleSuggestions[activeIndex]);
              return;
            }
            if (event.key === "Escape" && paletteOpen) {
              event.preventDefault();
              setCommandMenuOpen(false);
              setInlineSuppressed(true);
              return;
            }
            if (event.key === "ArrowUp") {
              if (paletteOpen && visibleSuggestions.length > 0) {
                event.preventDefault();
                moveActive(-1);
                return;
              }
              const next = onHistoryStep(-1);
              if (next !== null) {
                event.preventDefault();
                onDraftChange(next);
              }
            }
            if (event.key === "ArrowDown") {
              if (paletteOpen && visibleSuggestions.length > 0) {
                event.preventDefault();
                moveActive(1);
                return;
              }
              const next = onHistoryStep(1);
              if (next !== null) {
                event.preventDefault();
                onDraftChange(next);
              }
            }
          }}
          placeholder="What do you want to try?"
          rows={2}
        />
        <label className="select-wrap">
          <span className="sr-only">Action type</span>
          <select value={mode} onChange={(event) => onModeChange(event.target.value)}>
            <option value="action">Action</option>
            <option value="talk">Talk</option>
            <option value="advance">Advance</option>
            <option value="timeskip">Time Skip</option>
          </select>
          <ChevronDown size={14} />
        </label>
        <button type="submit" className="execute-button" disabled={disabled || !draft.trim()}>
          <CornerDownLeft size={16} />
          Send action
        </button>
      </div>
      <div className="composer-tip">
        <span>{notice || "Type / for commands or use Ctrl+K to explore them."}</span>
        <span>Ctrl+Enter to send</span>
      </div>
      {paletteOpen && (
        <CommandPalette
          items={visibleSuggestions}
          activeIndex={activeIndex}
          variant={paletteVariant}
          onActiveIndexChange={setActiveIndex}
          onPick={pickSuggestion}
        />
      )}
    </form>
  );
}
