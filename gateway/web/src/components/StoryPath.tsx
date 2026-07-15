import { BookOpen, ImageOff, Moon, Pause, Play, SlidersHorizontal, Sun, Sunrise, Sunset } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { StorySnapshot, VisualAsset } from "../types";
import { displayClock } from "../format";
import { formatInterfaceNumber } from "../i18n";
import { readyAssetUrl } from "../visualAssets";

interface StoryPathProps {
  snapshot: StorySnapshot | null;
  locationAsset?: VisualAsset | null;
  paused?: boolean;
  onTogglePaused?: () => void;
  onClearTranscript?: () => void;
  onOpenVisualAsset?: (assetId: string) => void;
}

export function StoryPath({ snapshot, locationAsset, paused = false, onTogglePaused, onClearTranscript, onOpenVisualAsset }: StoryPathProps) {
  const { t } = useTranslation(["map", "format"]);
  if (!snapshot) {
    return (
      <div className="scene-header empty-scene">
        <BookOpen size={14} aria-hidden="true" />
        <span>{t("storyPath.selectStory")}</span>
      </div>
    );
  }

  const locationParts = snapshot.world.current_location
    .split(",")
    .map((part) => part.trim())
    .filter(Boolean)
    .slice(0, 4);
  const parts = locationParts.slice(1);

  const imageUrl = readyAssetUrl(locationAsset);
  const clock = displayClock(snapshot);
  const CycleIcon = cycleIcon(clock.cycle, {
    morning: t("format:morning"),
    afternoon: t("format:afternoon"),
    evening: t("format:evening"),
    night: t("format:night"),
  });

  return (
    <div className={`scene-header ${imageUrl ? "has-image" : ""}`} aria-label={t("storyPath.currentPath")}>
      {imageUrl && locationAsset && onOpenVisualAsset ? (
        <button
          type="button"
          className="scene-image-button"
          onClick={() => onOpenVisualAsset(locationAsset.id)}
          title={t("storyPath.editImagePrompt", { subject: locationAsset.subject })}
          aria-label={t("storyPath.editImagePrompt", { subject: locationAsset.subject })}
        >
          <img src={imageUrl} alt="" />
        </button>
      ) : (
        imageUrl && <img src={imageUrl} alt="" />
      )}
      <div className="scene-scrim" />
      <div className="scene-copy">
        <div className="scene-copy-head">
          <span className="scene-kicker">
            <BookOpen size={14} aria-hidden="true" />
            {t("storyPath.chapter", { number: formatInterfaceNumber(snapshot.world.current_chapter) })}
          </span>
          <span className="scene-cycle-chip">
            <CycleIcon size={13} aria-hidden="true" />
            {clock.cycle}
          </span>
        </div>
        <h2 title={snapshot.world.current_location}>{snapshot.world.current_location || t("storyPath.unknownLocation")}</h2>
        {parts.length > 0 && <div className="scene-path">
          {parts.map((part, index) => (
            <span key={`${part}-${index}`}>{part}</span>
          ))}
        </div>}
      </div>
      {(onTogglePaused || onClearTranscript) && (
        <details className="scene-reading-controls">
          <summary title={t("storyPath.readingControls")} aria-label={t("storyPath.readingControls")}><SlidersHorizontal size={15} aria-hidden="true" /></summary>
          <div>
            {onTogglePaused && <button type="button" onClick={onTogglePaused}>{paused ? <Play size={14} aria-hidden="true" /> : <Pause size={14} aria-hidden="true" />}{paused ? t("storyPath.resumeUpdates") : t("storyPath.pauseUpdates")}</button>}
            {onClearTranscript && <button type="button" onClick={onClearTranscript}>{t("storyPath.hideMessages")}</button>}
          </div>
        </details>
      )}
      {locationAsset && locationAsset.status !== "ready" && (
        <div className="scene-asset-state" title={locationAsset.prompt || t("storyPath.sceneNotReady")}>
          <ImageOff size={14} aria-hidden="true" />
          <span>{t("storyPath.sceneArtStatus", { status: assetStatusLabel(locationAsset.status, t) })}</span>
        </div>
      )}
    </div>
  );
}

function cycleIcon(cycle: string, labels: { morning: string; afternoon: string; evening: string; night: string }) {
  if (cycle === labels.morning) return Sunrise;
  if (cycle === labels.afternoon) return Sun;
  if (cycle === labels.evening) return Sunset;
  if (cycle === labels.night) return Moon;
  return Sun;
}

function assetStatusLabel(status: string, t: (key: string) => string): string {
  if (["queued", "running", "failed"].includes(status)) return t(`storyPath.assetStatus.${status}`);
  return t("storyPath.assetStatus.unknown");
}
