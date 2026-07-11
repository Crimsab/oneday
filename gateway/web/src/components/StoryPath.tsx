import { BookOpen, ImageOff, Moon, Pause, Play, SlidersHorizontal, Sun, Sunrise, Sunset } from "lucide-react";
import type { StorySnapshot, VisualAsset } from "../types";
import { displayClock } from "../format";
import { readyAssetUrl } from "../visualAssets";

interface StoryPathProps {
  snapshot: StorySnapshot | null;
  locationAsset?: VisualAsset | null;
  paused?: boolean;
  onTogglePaused?: () => void;
  onClearTranscript?: () => void;
  onOpenVisualAsset?: (assetId: string) => void;
}

const cycleIcon = {
  Morning: Sunrise,
  Afternoon: Sun,
  Evening: Sunset,
  Night: Moon,
};

export function StoryPath({ snapshot, locationAsset, paused = false, onTogglePaused, onClearTranscript, onOpenVisualAsset }: StoryPathProps) {
  if (!snapshot) {
    return (
      <div className="scene-header empty-scene">
        <BookOpen size={14} />
        <span>Select a story</span>
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
  const CycleIcon = cycleIcon[clock.cycle as keyof typeof cycleIcon] ?? Sun;

  return (
    <div className={`scene-header ${imageUrl ? "has-image" : ""}`} aria-label="Current story path">
      {imageUrl && locationAsset && onOpenVisualAsset ? (
        <button
          type="button"
          className="scene-image-button"
          onClick={() => onOpenVisualAsset(locationAsset.id)}
          title={`Edit image prompt for ${locationAsset.subject}`}
          aria-label={`Edit image prompt for ${locationAsset.subject}`}
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
            <BookOpen size={14} />
            Chapter {snapshot.world.current_chapter}
          </span>
          <span className="scene-cycle-chip">
            <CycleIcon size={13} />
            {clock.cycle}
          </span>
        </div>
        <h2 title={snapshot.world.current_location}>{snapshot.world.current_location || "Unknown location"}</h2>
        {parts.length > 0 && <div className="scene-path">
          {parts.map((part, index) => (
            <span key={`${part}-${index}`}>{part}</span>
          ))}
        </div>}
      </div>
      {(onTogglePaused || onClearTranscript) && (
        <details className="scene-reading-controls">
          <summary title="Reading controls" aria-label="Reading controls"><SlidersHorizontal size={15} /></summary>
          <div>
            {onTogglePaused && <button type="button" onClick={onTogglePaused}>{paused ? <Play size={14} /> : <Pause size={14} />}{paused ? "Resume updates" : "Pause updates"}</button>}
            {onClearTranscript && <button type="button" onClick={onClearTranscript}>Hide current messages</button>}
          </div>
        </details>
      )}
      {locationAsset && locationAsset.status !== "ready" && (
        <div className="scene-asset-state" title={locationAsset.prompt || "Scene art is not ready yet"}>
          <ImageOff size={14} />
          <span>Scene art {locationAsset.status}</span>
        </div>
      )}
    </div>
  );
}
