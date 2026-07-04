import { BookOpen, ImageOff, MapPin, Moon, Sun, Sunrise, Sunset } from "lucide-react";
import type { StorySnapshot, VisualAsset } from "../types";
import { displayClock } from "../format";
import { readyAssetUrl } from "../visualAssets";

interface StoryPathProps {
  snapshot: StorySnapshot | null;
  locationAsset?: VisualAsset | null;
}

const cycleIcon = {
  Morning: Sunrise,
  Afternoon: Sun,
  Evening: Sunset,
  Night: Moon,
};

export function StoryPath({ snapshot, locationAsset }: StoryPathProps) {
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
  const parts = [snapshot.story.name || "Story", `Chapter ${snapshot.world.current_chapter}`, ...locationParts];

  const imageUrl = readyAssetUrl(locationAsset);
  const clock = displayClock(snapshot.world.current_turn);
  const CycleIcon = cycleIcon[clock.cycle as keyof typeof cycleIcon] ?? Sun;

  return (
    <div className={`scene-header ${imageUrl ? "has-image" : ""}`} aria-label="Current story path">
      {imageUrl && <img src={imageUrl} alt="" />}
      <div className="scene-scrim" />
      <div className="scene-copy">
        <div className="scene-copy-head">
          <span className="scene-kicker">
            <MapPin size={14} />
            {snapshot.story.name || "Story"}
          </span>
          <span className="scene-cycle-chip">
            <CycleIcon size={13} />
            {clock.cycle}
          </span>
        </div>
        <h2 title={snapshot.world.current_location}>{snapshot.world.current_location || "Unknown location"}</h2>
        <div className="scene-path">
          {parts.slice(1).map((part, index) => (
            <span key={`${part}-${index}`}>{part}</span>
          ))}
        </div>
      </div>
      <div className="scene-asset-state" title={locationAsset?.prompt || "No visual prompt available yet"}>
        {locationAsset?.status === "ready" ? (
          <span>Image ready</span>
        ) : (
          <>
            <ImageOff size={14} />
            <span>{locationAsset ? `Image ${locationAsset.status}` : "Image pending"}</span>
          </>
        )}
      </div>
    </div>
  );
}
