package storage

import (
	"fmt"
	"time"
)

// CreateStory inserts a new story.
func (db *DB) CreateStory(s *Story) error {
	_, err := db.conn.Exec(
		`INSERT INTO stories (id, name, setting_json, stats_schema_json, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?)`,
		s.ID, s.Name, s.SettingJSON, s.StatsSchemaJSON, s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting story: %w", err)
	}
	return nil
}

// GetStory retrieves a story by ID.
func (db *DB) GetStory(id string) (*Story, error) {
	s := &Story{}
	err := db.conn.QueryRow(
		`SELECT id, name, setting_json, stats_schema_json, created_at, updated_at
         FROM stories WHERE id = ?`, id,
	).Scan(&s.ID, &s.Name, &s.SettingJSON, &s.StatsSchemaJSON, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting story %s: %w", id, err)
	}
	return s, nil
}

// ListStories returns all stories ordered by most recent.
func (db *DB) ListStories() ([]Story, error) {
	rows, err := db.conn.Query(
		`SELECT id, name, setting_json, stats_schema_json, created_at, updated_at
         FROM stories ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing stories: %w", err)
	}
	defer rows.Close()

	var stories []Story
	for rows.Next() {
		var s Story
		if err := rows.Scan(&s.ID, &s.Name, &s.SettingJSON, &s.StatsSchemaJSON,
			&s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning story: %w", err)
		}
		stories = append(stories, s)
	}
	return stories, rows.Err()
}

// CreateCharacter inserts a new character.
func (db *DB) CreateCharacter(c *Character) error {
	_, err := db.conn.Exec(
		`INSERT INTO characters (id, story_id, name, background, stats_json, traits_json,
         skills_json, inventory_json, known_recipes_json, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.StoryID, c.Name, c.Background, c.StatsJSON, c.TraitsJSON,
		c.SkillsJSON, c.InventoryJSON, c.KnownRecipesJSON, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting character: %w", err)
	}
	return nil
}

// GetCharacterByStory retrieves the protagonist for a story.
func (db *DB) GetCharacterByStory(storyID string) (*Character, error) {
	c := &Character{}
	err := db.conn.QueryRow(
		`SELECT id, story_id, name, background, stats_json, traits_json,
         skills_json, inventory_json, known_recipes_json, created_at, updated_at
         FROM characters WHERE story_id = ?`, storyID,
	).Scan(&c.ID, &c.StoryID, &c.Name, &c.Background, &c.StatsJSON, &c.TraitsJSON,
		&c.SkillsJSON, &c.InventoryJSON, &c.KnownRecipesJSON, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting character for story %s: %w", storyID, err)
	}
	return c, nil
}

// CreateWorldState inserts a new world state.
func (db *DB) CreateWorldState(ws *WorldState) error {
	_, err := db.conn.Exec(
		`INSERT INTO world_state (id, story_id, current_location, known_locations_json,
         global_events_json, faction_standings_json, current_chapter, current_turn, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ws.ID, ws.StoryID, ws.CurrentLocation, ws.KnownLocationsJSON,
		ws.GlobalEventsJSON, ws.FactionStandingsJSON, ws.CurrentChapter, ws.CurrentTurn, ws.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting world state: %w", err)
	}
	return nil
}

// GetWorldState retrieves the world state for a story.
func (db *DB) GetWorldState(storyID string) (*WorldState, error) {
	ws := &WorldState{}
	err := db.conn.QueryRow(
		`SELECT id, story_id, current_location, known_locations_json,
         global_events_json, faction_standings_json, current_chapter, current_turn, updated_at
         FROM world_state WHERE story_id = ?`, storyID,
	).Scan(&ws.ID, &ws.StoryID, &ws.CurrentLocation, &ws.KnownLocationsJSON,
		&ws.GlobalEventsJSON, &ws.FactionStandingsJSON, &ws.CurrentChapter, &ws.CurrentTurn, &ws.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting world state for story %s: %w", storyID, err)
	}
	return ws, nil
}

// UpdateCharacterStats updates only the stats_json and updated_at for a character.
func (db *DB) UpdateCharacterStats(c *Character) error {
	_, err := db.conn.Exec(
		`UPDATE characters SET stats_json = ?, updated_at = ? WHERE id = ?`,
		c.StatsJSON, time.Now(), c.ID,
	)
	if err != nil {
		return fmt.Errorf("updating character stats: %w", err)
	}
	return nil
}

// UpdateCharacterFull updates all mutable character fields: stats, traits, skills,
// inventory, and known recipes. Use this after any state change that may affect
// traits, skills, or inventory (not just stats).
func (db *DB) UpdateCharacterFull(c *Character) error {
	_, err := db.conn.Exec(
		`UPDATE characters SET stats_json = ?, traits_json = ?, skills_json = ?,
         inventory_json = ?, known_recipes_json = ?, updated_at = ? WHERE id = ?`,
		c.StatsJSON, c.TraitsJSON, c.SkillsJSON,
		c.InventoryJSON, c.KnownRecipesJSON, time.Now(), c.ID,
	)
	if err != nil {
		return fmt.Errorf("updating character full: %w", err)
	}
	return nil
}

// UpdateWorldState updates the world state fields.
func (db *DB) UpdateWorldState(ws *WorldState) error {
	_, err := db.conn.Exec(
		`UPDATE world_state SET current_location = ?, known_locations_json = ?,
         global_events_json = ?, faction_standings_json = ?,
         current_chapter = ?, current_turn = ?, updated_at = ?
         WHERE id = ?`,
		ws.CurrentLocation, ws.KnownLocationsJSON,
		ws.GlobalEventsJSON, ws.FactionStandingsJSON,
		ws.CurrentChapter, ws.CurrentTurn, time.Now(),
		ws.ID,
	)
	if err != nil {
		return fmt.Errorf("updating world state: %w", err)
	}
	return nil
}

// UpdateStoryTimestamp updates the story's updated_at to now.
func (db *DB) UpdateStoryTimestamp(storyID string) error {
	_, err := db.conn.Exec(
		`UPDATE stories SET updated_at = ? WHERE id = ?`,
		time.Now(), storyID,
	)
	if err != nil {
		return fmt.Errorf("updating story timestamp: %w", err)
	}
	return nil
}
