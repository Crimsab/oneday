package engine

import (
	"strings"

	"github.com/crimsab/oneday/internal/storage"
)

// validCategories is the set of allowed achievement categories.
var validCategories = map[string]bool{
	"story":       true,
	"combat":      true,
	"social":      true,
	"exploration": true,
	"skill":       true,
	"creative":    true,
	"meta":        true,
}

// validRarities is the set of allowed achievement rarities.
var validRarities = map[string]bool{
	"common":    true,
	"uncommon":  true,
	"rare":      true,
	"epic":      true,
	"legendary": true,
}

// ValidateAndPersistAchievement checks the AI-returned achievement data,
// validates category/rarity, checks for duplicates, and persists to DB.
// Returns the persisted Achievement or nil if invalid/duplicate.
func ValidateAndPersistAchievement(db *storage.DB, storyID string, data *AchievementData) *storage.Achievement {
	if data == nil {
		return nil
	}
	if strings.TrimSpace(data.Name) == "" {
		return nil
	}
	if !validCategories[strings.ToLower(data.Category)] {
		return nil
	}
	if !validRarities[strings.ToLower(data.Rarity)] {
		return nil
	}

	// Check for duplicates (case-insensitive).
	existing, err := db.ListAchievements(storyID)
	if err != nil {
		return nil
	}
	nameLower := strings.ToLower(strings.TrimSpace(data.Name))
	for _, a := range existing {
		if strings.ToLower(strings.TrimSpace(a.Name)) == nameLower {
			return nil // duplicate
		}
	}

	a := &storage.Achievement{
		StoryID:     storyID,
		Name:        strings.TrimSpace(data.Name),
		Description: data.Description,
		Category:    strings.ToLower(data.Category),
		Rarity:      strings.ToLower(data.Rarity),
		Context:     data.Context,
	}
	if err := db.CreateAchievement(a); err != nil {
		return nil
	}
	return a
}
