import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { BookOpen, CircleHelp, Clock3, Hash, PanelLeftClose, PanelLeftOpen, PanelRightClose, PanelRightOpen, SlidersHorizontal, Wrench } from "lucide-react";
import { displayClock } from "../format";
import type { OverlayKind, StorySnapshot, SyncState } from "../types";
import type { ModelSettings } from "../types";
import { TranslationCenter } from "../features/translation/TranslationCenter";

interface TopBarProps {
  snapshot: StorySnapshot | null;
  sync: SyncState;
  syncLabel: string;
  syncTitle: string;
  leftRailVisible: boolean;
  showInspector: boolean;
  onToggleLeftRail: () => void;
  onToggleInspector: () => void;
  onOpen: (overlay: OverlayKind) => void;
  onOpenSetup: () => void;
  modelSettings: ModelSettings | null;
  translationCenterOpen: boolean;
  onTranslationCenterOpenChange: (open: boolean) => void;
}

export function TopBar({ snapshot, sync, syncLabel, syncTitle, leftRailVisible, showInspector, onToggleLeftRail, onToggleInspector, onOpen, onOpenSetup, modelSettings, translationCenterOpen, onTranslationCenterOpenChange }: TopBarProps) {
  const { t } = useTranslation("chrome");
  const clock = displayClock(snapshot);
  return (
    <header className="top-bar">
      <div className="top-status" aria-label={t("status")}>
        <StatusCell icon={<Hash size={14} />} label={t("turn")} value={snapshot ? String(snapshot.world.current_turn) : "-"} />
        <StatusCell icon={<BookOpen size={14} />} label={t("story")} value={snapshot?.story.name || t("chooseStory")} strong />
        <StatusCell icon={<Clock3 size={14} />} label={t("storyTime")} value={snapshot ? clock.time : "-"} />
        <div className={`status-cell sync-cell ${sync.toLowerCase()}`}>
          <i aria-hidden="true" />
          <strong title={syncTitle}>{syncLabel}</strong>
        </div>
      </div>
      <div className="top-actions">
        {snapshot && <TranslationCenter storyId={snapshot.story.id} storyLanguage={snapshot.story.language} modelSettings={modelSettings} open={translationCenterOpen} onOpenChange={onTranslationCenterOpenChange} />}
        <button
          className="square-button"
          type="button"
          onClick={onToggleLeftRail}
          title={`${leftRailVisible ? t("hideLibrary") : t("showLibrary")} ([)`}
          aria-label={leftRailVisible ? t("hideLibrary") : t("showLibrary")}
          aria-expanded={leftRailVisible}
          aria-controls="story-navigation"
        >
          {leftRailVisible ? <PanelLeftClose size={16} /> : <PanelLeftOpen size={16} />}
          <span>{t("libraryLabel")}</span>
        </button>
        <button
          className="square-button"
          type="button"
          onClick={onToggleInspector}
          title={`${showInspector ? t("hideDetails") : t("showDetails")} (])`}
          aria-label={showInspector ? t("hideDetails") : t("showDetails")}
          aria-expanded={showInspector}
          aria-controls="story-details"
        >
          {showInspector ? <PanelRightClose size={16} /> : <PanelRightOpen size={16} />}
          <span>{t("details")}</span>
        </button>
        <button className="chrome-button" type="button" onClick={() => onOpen("options")} aria-label={t("options")} title={t("options")}>
          <SlidersHorizontal size={15} />
          {t("options")}
        </button>
        <button className="chrome-button" type="button" onClick={onOpenSetup} aria-label={t("setup")} title={t("setup")}>
          <Wrench size={15} />
          {t("setup")}
        </button>
        <button
          className="chrome-button"
          type="button"
          onClick={() => onOpen("help")}
          aria-label={t("help")}
          title={t("help")}
        >
          <CircleHelp size={16} />
          <span className="sr-only">{t("help")}</span>
        </button>
      </div>
    </header>
  );
}

function StatusCell({
  icon,
  label,
  value,
  strong = false,
}: {
  icon?: ReactNode;
  label: string;
  value: string;
  strong?: boolean;
}) {
  return (
    <div className="status-cell" aria-label={`${label}: ${value}`}>
      {icon && <span className="status-icon" aria-hidden="true">{icon}</span>}
      <span>{label}</span>
      <strong className={strong ? "status-strong" : undefined} title={value}>
        {value}
      </strong>
    </div>
  );
}
