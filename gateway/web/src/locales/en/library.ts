export const library = {
  filterStatus: "Story status", activeStories: "Active ({{count}})", archivedStories: "Archived ({{count}})",
  storiesCount: "Stories ({{count}})", activeStoryCount_one: "1 active story", activeStoryCount_other: "{{count}} active stories",
  noArchivedStories: "No archived stories.", current: "Current", storyMeta: "{{genre}} · {{language}}", updated: "Updated {{value}}",
  groups: { story: "Story", character: "Character", threads: "Active threads", library: "Library" },
  chapter: "Chapter {{number}}", untitled: "Untitled", turn: "Turn {{turn}}", emptyNotes: "Select a story to see hooks, contacts, and next leads.",
  name: "Name", description: "Description", genre: "Genre", tone: "Tone", language: "Story language", languageChangeConfirm: "Change the story language? This affects future output only. Existing turns remain unchanged.", cancel: "Cancel", save: "Save",
  storySummary: "Turn {{turn}} · {{genre}}", archived: "Archived", manage: "Manage {{name}}", details: "Details", edit: "Edit", restore: "Restore", archive: "Archive", delete: "Delete", storyFallback: "Story",
  note: { hook: "Hook: {{value}}", front: "Active front: {{value}}", contact: "Key contact: {{value}}", lead: "Next lead: {{value}}", updated: "Updated: {{value}}" },
  detail: { back: "Back", open: "Open story", loading: "Loading…", empty: "Nothing here yet.", noDescription: "No description.", active: "active", branch: "branch", branchMeta: "{{count}} turns · {{state}}", turnRange: "Turns {{start}}–{{end}}", tabs: { overview: "Overview", branches: "Branches", chapters: "Chapters", saves: "Saves", timeline: "Timeline", assets: "Assets" }, stats: { turn: "Turn", branches: "Branches", chapters: "Chapters", saves: "Saves", messages: "Messages", assets: "Assets" } },
  tabs: { history: "History", map: "Map", codex: "Codex", inventory: "Inventory", stats: "Stats", craft: "Craft", fronts: "Fronts", investigations: "Investigations", projects: "Projects", achievements: "Achievements", saves: "Saves" },
} as const;
