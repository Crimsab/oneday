package storage

import (
	"database/sql"
	"fmt"
	"time"
)

// CreateStory inserts a new story.
func (db *DB) CreateStory(s *Story) error {
	_, err := db.conn.Exec(
		`INSERT INTO stories (
			id, name, setting_json, stats_schema_json, description, genre, tone,
			language, writing_style, prompt_directives, is_archived, created_at, updated_at
		)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.Name, s.SettingJSON, s.StatsSchemaJSON, s.Description, s.Genre, s.Tone,
		s.Language, s.WritingStyle, s.PromptDirectives, s.IsArchived, s.CreatedAt, s.UpdatedAt,
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
		`SELECT id, name, setting_json, stats_schema_json, description, genre, tone,
                language, writing_style, prompt_directives, is_archived, created_at, updated_at
         FROM stories WHERE id = ?`, id,
	).Scan(
		&s.ID, &s.Name, &s.SettingJSON, &s.StatsSchemaJSON, &s.Description, &s.Genre, &s.Tone,
		&s.Language, &s.WritingStyle, &s.PromptDirectives, &s.IsArchived, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("getting story %s: %w", id, err)
	}
	return s, nil
}

// ListStories returns all stories ordered by most recent.
func (db *DB) ListStories() ([]Story, error) {
	rows, err := db.conn.Query(
		`SELECT id, name, setting_json, stats_schema_json, description, genre, tone,
                language, writing_style, prompt_directives, is_archived, created_at, updated_at
         FROM stories ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing stories: %w", err)
	}
	defer rows.Close()

	var stories []Story
	for rows.Next() {
		var s Story
		if err := rows.Scan(
			&s.ID, &s.Name, &s.SettingJSON, &s.StatsSchemaJSON,
			&s.Description, &s.Genre, &s.Tone,
			&s.Language, &s.WritingStyle, &s.PromptDirectives, &s.IsArchived,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
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
	if ws.KnownLocationsJSON == "" {
		ws.KnownLocationsJSON = "[]"
	}
	if ws.GlobalEventsJSON == "" {
		ws.GlobalEventsJSON = "[]"
	}
	if ws.FactionStandingsJSON == "" {
		ws.FactionStandingsJSON = "{}"
	}
	if ws.StoryHooksJSON == "" {
		ws.StoryHooksJSON = "[]"
	}
	if ws.WorldReactionsJSON == "" {
		ws.WorldReactionsJSON = "[]"
	}
	if ws.InvestigationBoardJSON == "" {
		ws.InvestigationBoardJSON = "{}"
	}
	if ws.ProjectClocksJSON == "" {
		ws.ProjectClocksJSON = "{}"
	}
	if ws.PlayerGuidanceJSON == "" {
		ws.PlayerGuidanceJSON = "[]"
	}
	if ws.FrontsJSON == "" {
		ws.FrontsJSON = "[]"
	}
	_, err := db.conn.Exec(
		`INSERT INTO world_state (id, story_id, current_location, known_locations_json,
         global_events_json, faction_standings_json, story_hooks_json, world_reactions_json,
         investigation_board_json, project_clocks_json, player_guidance_json, fronts_json, current_chapter, current_turn, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ws.ID, ws.StoryID, ws.CurrentLocation, ws.KnownLocationsJSON,
		ws.GlobalEventsJSON, ws.FactionStandingsJSON, ws.StoryHooksJSON, ws.WorldReactionsJSON,
		ws.InvestigationBoardJSON, ws.ProjectClocksJSON, ws.PlayerGuidanceJSON, ws.FrontsJSON,
		ws.CurrentChapter, ws.CurrentTurn, ws.UpdatedAt,
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
         global_events_json, faction_standings_json, story_hooks_json, world_reactions_json,
         investigation_board_json, project_clocks_json, player_guidance_json, fronts_json, current_chapter, current_turn, updated_at
         FROM world_state WHERE story_id = ?`, storyID,
	).Scan(&ws.ID, &ws.StoryID, &ws.CurrentLocation, &ws.KnownLocationsJSON,
		&ws.GlobalEventsJSON, &ws.FactionStandingsJSON, &ws.StoryHooksJSON, &ws.WorldReactionsJSON,
		&ws.InvestigationBoardJSON, &ws.ProjectClocksJSON, &ws.PlayerGuidanceJSON, &ws.FrontsJSON,
		&ws.CurrentChapter, &ws.CurrentTurn, &ws.UpdatedAt)
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
	return updateCharacterFullExec(db.conn, c)
}

// UpdateCharacterFullTx updates the mutable character fields inside an existing transaction.
func (db *DB) UpdateCharacterFullTx(tx *sql.Tx, c *Character) error {
	return updateCharacterFullExec(tx, c)
}

func updateCharacterFullExec(exec sqlExecer, c *Character) error {
	_, err := exec.Exec(
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
	return updateWorldStateExec(db.conn, ws)
}

// UpdateWorldStateTx updates the world state inside an existing transaction.
func (db *DB) UpdateWorldStateTx(tx *sql.Tx, ws *WorldState) error {
	return updateWorldStateExec(tx, ws)
}

func updateWorldStateExec(exec sqlExecer, ws *WorldState) error {
	if ws.KnownLocationsJSON == "" {
		ws.KnownLocationsJSON = "[]"
	}
	if ws.GlobalEventsJSON == "" {
		ws.GlobalEventsJSON = "[]"
	}
	if ws.FactionStandingsJSON == "" {
		ws.FactionStandingsJSON = "{}"
	}
	if ws.StoryHooksJSON == "" {
		ws.StoryHooksJSON = "[]"
	}
	if ws.WorldReactionsJSON == "" {
		ws.WorldReactionsJSON = "[]"
	}
	if ws.InvestigationBoardJSON == "" {
		ws.InvestigationBoardJSON = "{}"
	}
	if ws.ProjectClocksJSON == "" {
		ws.ProjectClocksJSON = "{}"
	}
	if ws.PlayerGuidanceJSON == "" {
		ws.PlayerGuidanceJSON = "[]"
	}
	if ws.FrontsJSON == "" {
		ws.FrontsJSON = "[]"
	}
	_, err := exec.Exec(
		`UPDATE world_state SET current_location = ?, known_locations_json = ?,
         global_events_json = ?, faction_standings_json = ?, story_hooks_json = ?,
         world_reactions_json = ?, investigation_board_json = ?, project_clocks_json = ?, player_guidance_json = ?, fronts_json = ?, current_chapter = ?, current_turn = ?, updated_at = ?
         WHERE id = ?`,
		ws.CurrentLocation, ws.KnownLocationsJSON,
		ws.GlobalEventsJSON, ws.FactionStandingsJSON, ws.StoryHooksJSON, ws.WorldReactionsJSON,
		ws.InvestigationBoardJSON, ws.ProjectClocksJSON, ws.PlayerGuidanceJSON, ws.FrontsJSON,
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

// UpdateStorySetting updates the setting_json for a story and bumps updated_at.
func (db *DB) UpdateStorySetting(storyID, settingJSON string) error {
	_, err := db.conn.Exec(
		`UPDATE stories SET setting_json = ?, updated_at = ? WHERE id = ?`,
		settingJSON, time.Now(), storyID,
	)
	if err != nil {
		return fmt.Errorf("updating story setting for %s: %w", storyID, err)
	}
	return nil
}

// SetStoryArchived updates the archived flag for a story and bumps updated_at.
func (db *DB) SetStoryArchived(storyID string, archived bool) error {
	_, err := db.conn.Exec(
		`UPDATE stories SET is_archived = ?, updated_at = ? WHERE id = ?`,
		archived, time.Now(), storyID,
	)
	if err != nil {
		return fmt.Errorf("updating story archived flag for %s: %w", storyID, err)
	}
	return nil
}

// DeleteStory removes a story row; cascading foreign keys clean up related rows.
func (db *DB) DeleteStory(storyID string) error {
	_, err := db.conn.Exec(`DELETE FROM stories WHERE id = ?`, storyID)
	if err != nil {
		return fmt.Errorf("deleting story %s: %w", storyID, err)
	}
	return nil
}

// InsertCombatLog records a combat encounter outcome.
func (db *DB) InsertCombatLog(log *CombatLog) error {
	_, err := db.conn.Exec(
		`INSERT INTO combat_log
			(story_id, session_id, enemy_name, enemy_hp, turns, victory, defeat_outcome, player_hp_start, player_hp_end, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.StoryID, log.SessionID, log.EnemyName, log.EnemyHP,
		log.Turns, log.Victory, log.DefeatOutcome,
		log.PlayerHPStart, log.PlayerHPEnd, log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting combat log: %w", err)
	}
	return nil
}

// GetCombatStats returns the win/loss totals for a story.
func (db *DB) GetCombatStats(storyID string) (wins int, losses int, err error) {
	rows, err := db.conn.Query(
		`SELECT victory FROM combat_log WHERE story_id = ?`, storyID,
	)
	if err != nil {
		return 0, 0, fmt.Errorf("querying combat stats: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var victory bool
		if scanErr := rows.Scan(&victory); scanErr != nil {
			return wins, losses, fmt.Errorf("scanning combat stat row: %w", scanErr)
		}
		if victory {
			wins++
		} else {
			losses++
		}
	}
	return wins, losses, rows.Err()
}
