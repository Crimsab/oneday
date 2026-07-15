import type { StorySummary } from "../../types";

export type StoryLibraryStatus = "active" | "archived";

export function filterStoryLibrary(
  stories: readonly StorySummary[],
  status: StoryLibraryStatus,
  query: string,
): StorySummary[] {
  const normalizedQuery = query.trim().toLocaleLowerCase();
  return stories.filter((story) => {
    if (story.is_archived !== (status === "archived")) return false;
    if (!normalizedQuery) return true;
    return `${story.name} ${story.description} ${story.genre} ${story.tone} ${story.language}`
      .toLocaleLowerCase()
      .includes(normalizedQuery);
  });
}
