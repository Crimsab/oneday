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

// rarityStarsText returns a Unicode star string for a rarity level.
func rarityStarsText(rarity string) string {
	switch strings.ToLower(rarity) {
	case "uncommon":
		return "★★"
	case "rare":
		return "★★★"
	case "epic":
		return "★★★★"
	case "legendary":
		return "★★★★★"
	default: // common
		return "★"
	}
}

// categoryOrder defines the display order for achievement categories.
var categoryOrder = []string{"story", "combat", "social", "exploration", "skill", "creative", "meta"}

// FormatAchievementsView builds a text representation of earned achievements grouped by category,
// with rarity stars and a summary count.
func FormatAchievementsView(db *storage.DB, storyID string) string {
	achievements, err := db.ListAchievements(storyID)
	if err != nil || len(achievements) == 0 {
		return "No achievements earned yet. Keep playing — noteworthy moments will be recognized!"
	}

	// Group by category.
	byCategory := map[string][]storage.Achievement{}
	for _, a := range achievements {
		cat := strings.ToLower(a.Category)
		if cat == "" {
			cat = "story"
		}
		byCategory[cat] = append(byCategory[cat], a)
	}

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
		if rarity == "" {
			rarity = "common"
		}
		if _, ok := rarityCount[rarity]; ok {
			rarityCount[rarity]++
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== Achievements  (%d total) ===\n", len(achievements)))

	// Render categories in defined order, then any extras not in the list.
	seen := map[string]bool{}
	orderedCats := make([]string, 0, len(categoryOrder))
	for _, cat := range categoryOrder {
		if _, ok := byCategory[cat]; ok {
			orderedCats = append(orderedCats, cat)
			seen[cat] = true
		}
	}
	for cat := range byCategory {
		if !seen[cat] {
			orderedCats = append(orderedCats, cat)
		}
	}

	for _, cat := range orderedCats {
		achs := byCategory[cat]
		catLabel := strings.ToUpper(cat[:1]) + cat[1:]
		sb.WriteString(fmt.Sprintf("\n── %s ──\n", catLabel))

		for _, a := range achs {
			stars := rarityStarsText(a.Rarity)
			rarityLabel := strings.ToLower(a.Rarity)
			if rarityLabel == "" {
				rarityLabel = "common"
			}
			sb.WriteString(fmt.Sprintf("%s %s  [%s]\n", stars, a.Name, rarityLabel))
			if a.Description != "" {
				sb.WriteString(fmt.Sprintf("   %s\n", a.Description))
			}
			if a.Context != "" {
				sb.WriteString(fmt.Sprintf("   (%s)\n", a.Context))
			}
		}
	}

	sb.WriteString(fmt.Sprintf("\n%d common · %d uncommon · %d rare · %d epic · %d legendary\n",
		rarityCount["common"],
		rarityCount["uncommon"],
		rarityCount["rare"],
		rarityCount["epic"],
		rarityCount["legendary"],
	))

	return sb.String()
}
