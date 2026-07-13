package engine

import (
	"fmt"
	"strings"

	"github.com/crimsab/oneday/internal/storage"
)

// FormatJournalView builds a text representation of the story journal using chapter summaries.
func FormatJournalView(db *storage.DB, storyID string, currentChapter int, currentTurn int) string {
	chapters, err := db.ListChapters(storyID)
	if err != nil || len(chapters) == 0 {
		return "Your story has just begun. The journal will fill as your adventure unfolds."
	}

	var sb strings.Builder
	sb.WriteString("=== Story Journal ===\n")

	for _, ch := range chapters {
		sb.WriteString("\n")
		if ch.EndTurn == nil {
			sb.WriteString(fmt.Sprintf("--- Chapter %d: %s [In Progress] ---\n", ch.ChapterNumber, ch.Title))
			sb.WriteString(fmt.Sprintf("Started at turn %d (current: turn %d)\n", ch.StartTurn, currentTurn))
		} else {
			sb.WriteString(fmt.Sprintf("--- Chapter %d: %s ---\n", ch.ChapterNumber, ch.Title))
			sb.WriteString(fmt.Sprintf("Turns %d-%d\n", ch.StartTurn, *ch.EndTurn))
			if ch.Summary != "" {
				sb.WriteString("\n")
				sb.WriteString(ch.Summary)
				sb.WriteString("\n")
			}
		}
	}

	return sb.String()
}
