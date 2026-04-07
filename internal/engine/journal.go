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
			// Current in-progress chapter.
			sb.WriteString(fmt.Sprintf("--- Chapter %d: %s [In Progress] ---\n", ch.ChapterNumber, ch.Title))
			sb.WriteString(fmt.Sprintf("Started at turn %d (current: turn %d)\n", ch.StartTurn, currentTurn))
		} else {
			// Completed chapter.
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

// FormatAchievementsView builds a text representation of earned achievements.
func FormatAchievementsView(db *storage.DB, storyID string) string {
	achievements, err := db.ListAchievements(storyID)
	if err != nil || len(achievements) == 0 {
		return "No achievements earned yet. Keep playing — noteworthy moments will be recognized!"
	}

	var sb strings.Builder
	sb.WriteString("=== Achievements ===\n\n")

	// Count by rarity.
	rarityCount := map[string]int{
		"common":    0,
		"uncommon":  0,
		"rare":      0,
		"epic":      0,
		"legendary": 0,
	}

	for _, a := range achievements {
		rarity := strings.ToLower(a.Rarity)
		rarityLabel := strings.ToUpper(a.Rarity)
		if rarityLabel == "" {
			rarityLabel = "COMMON"
			rarity = "common"
		}
		if _, ok := rarityCount[rarity]; ok {
			rarityCount[rarity]++
		}

		sb.WriteString(fmt.Sprintf("[%s] %s\n", rarityLabel, a.Name))
		if a.Description != "" {
			sb.WriteString(fmt.Sprintf("  \"%s\"\n", a.Description))
		}
		details := ""
		if a.Category != "" {
			details += fmt.Sprintf("Category: %s", a.Category)
		}
		if a.Context != "" {
			if details != "" {
				details += " | "
			}
			details += fmt.Sprintf("Context: %s", a.Context)
		}
		if details != "" {
			sb.WriteString(fmt.Sprintf("  %s\n", details))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("Total: %d achievement(s) — %d common, %d uncommon, %d rare, %d epic, %d legendary\n",
		len(achievements),
		rarityCount["common"],
		rarityCount["uncommon"],
		rarityCount["rare"],
		rarityCount["epic"],
		rarityCount["legendary"],
	))

	return sb.String()
}
