export const library = {
  groups: { story: "Story", character: "Character", threads: "Active threads", library: "Library" },
  chapter: "Chapter {{number}}", untitled: "Untitled", turn: "Turn {{turn}}", emptyNotes: "Select a story to see hooks, contacts, and next leads.",
  name: "Name", description: "Description", genre: "Genre", tone: "Tone", cancel: "Cancel", save: "Save",
  storySummary: "Turn {{turn}} · {{genre}}", archived: "Archived", manage: "Manage {{name}}", edit: "Edit", restore: "Restore", archive: "Archive", delete: "Delete", storyFallback: "Story",
  note: { hook: "Hook: {{value}}", front: "Active front: {{value}}", contact: "Key contact: {{value}}", lead: "Next lead: {{value}}", updated: "Updated: {{value}}" },
  tabs: { history: "History", map: "Map", codex: "Codex", inventory: "Inventory", stats: "Stats", craft: "Craft", fronts: "Fronts", investigations: "Investigations", projects: "Projects", achievements: "Achievements", saves: "Saves" },
} as const;
