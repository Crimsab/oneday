import { describe, expect, it } from "vitest";
import type { StorySummary } from "../../types";
import { filterStoryLibrary } from "./storyLibraryState";

const stories = [
  { id: "a", name: "Active mystery", description: "", genre: "Mystery", tone: "", language: "en", is_archived: false, updated_at: "" },
  { id: "b", name: "Vecchia storia", description: "Roma", genre: "Drama", tone: "", language: "it", is_archived: true, updated_at: "" },
] satisfies StorySummary[];

describe("story library filtering", () => {
  it("separates active and archived stories", () => {
    expect(filterStoryLibrary(stories, "active", "").map((story) => story.id)).toEqual(["a"]);
    expect(filterStoryLibrary(stories, "archived", "").map((story) => story.id)).toEqual(["b"]);
  });

  it("searches summary metadata within the selected status", () => {
    expect(filterStoryLibrary(stories, "archived", "roma").map((story) => story.id)).toEqual(["b"]);
    expect(filterStoryLibrary(stories, "active", "roma")).toEqual([]);
  });
});
