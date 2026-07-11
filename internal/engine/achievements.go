package engine

import (
	"database/sql"
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
	return validateAndPersistAchievement(db, nil, storyID, data)
}

func ValidateAndPersistAchievementTx(db *storage.DB, tx *sql.Tx, storyID string, data *AchievementData) *storage.Achievement {
	return validateAndPersistAchievement(db, tx, storyID, data)
}

func validateAndPersistAchievement(db *storage.DB, tx *sql.Tx, storyID string, data *AchievementData) *storage.Achievement {
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

	var exists bool
	var err error
	if tx != nil {
		exists, err = db.AchievementExistsByNameTx(tx, storyID, data.Name)
	} else {
		exists, err = db.AchievementExistsByName(storyID, data.Name)
	}
	if err != nil || exists {
		return nil
	}

	a := &storage.Achievement{
		StoryID:     storyID,
		Name:        strings.TrimSpace(data.Name),
		Description: data.Description,
		Category:    strings.ToLower(data.Category),
		Rarity:      strings.ToLower(data.Rarity),
		Context:     data.Context,
	}
	if tx != nil {
		err = db.CreateAchievementTx(tx, a)
	} else {
		err = db.CreateAchievement(a)
	}
	if err != nil {
		return nil
	}
	return a
}
