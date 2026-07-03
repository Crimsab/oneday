import { BookOpen, MapPin } from "lucide-react";
import type { StorySnapshot } from "../types";

interface StoryPathProps {
  snapshot: StorySnapshot | null;
}

export function StoryPath({ snapshot }: StoryPathProps) {
  if (!snapshot) {
    return (
      <div className="path-strip">
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

  return (
    <div className="path-strip" aria-label="Current story path">
      <MapPin size={14} />
      <div>
        {parts.map((part, index) => (
          <span key={`${part}-${index}`}>{part}</span>
        ))}
      </div>
    </div>
  );
}
