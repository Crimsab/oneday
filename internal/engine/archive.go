package engine

import (
	"fmt"
	"sort"
	"strings"

	"github.com/crimsab/oneday/internal/storage"
)

// StoryArchiveSummary is the player-facing home/archive summary for a story.
type StoryArchiveSummary struct {
	Story            storage.Story
	ProtagonistName  string
	CurrentLocation  string
	CurrentTurn      int
	AchievementCount int
	Achievements     []storage.Achievement
}

// BuildStoryArchiveSummaries loads player-facing archive summaries for all stories.
func BuildStoryArchiveSummaries(db *storage.DB) ([]StoryArchiveSummary, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}

	stories, err := db.ListStories()
	if err != nil {
		return nil, err
	}

	summaries := make([]StoryArchiveSummary, 0, len(stories))
	for _, story := range stories {
		summary, err := BuildStoryArchiveSummary(db, story.ID)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, *summary)
	}

	sort.SliceStable(summaries, func(i, j int) bool {
		return summaries[i].Story.UpdatedAt.After(summaries[j].Story.UpdatedAt)
	})
	return summaries, nil
}

// BuildStoryArchiveSummary loads the player-facing archive summary for a single story.
func BuildStoryArchiveSummary(db *storage.DB, storyID string) (*StoryArchiveSummary, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}

	story, err := db.GetStory(storyID)
	if err != nil {
		return nil, err
	}

	summary := &StoryArchiveSummary{Story: *story}

	if char, err := db.GetCharacterByStory(storyID); err == nil && char != nil {
		summary.ProtagonistName = strings.TrimSpace(char.Name)
	}

	if world, err := db.GetWorldState(storyID); err == nil && world != nil {
		summary.CurrentLocation = strings.TrimSpace(world.CurrentLocation)
		summary.CurrentTurn = world.CurrentTurn
	}

	achievements, err := db.ListAchievements(storyID)
	if err != nil {
		return nil, err
	}
	summary.Achievements = achievements
	summary.AchievementCount = len(achievements)

	return summary, nil
}
