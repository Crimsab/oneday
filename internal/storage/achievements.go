package storage

import "time"

// CreateAchievement inserts a new achievement for a story.
func (db *DB) CreateAchievement(a *Achievement) error {
	a.EarnedAt = time.Now()
	result, err := db.conn.Exec(
		`INSERT INTO achievements (story_id, name, description, category, rarity, context, earned_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		a.StoryID, a.Name, a.Description, a.Category, a.Rarity, a.Context, a.EarnedAt,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	a.ID = id
	return nil
}

// ListAchievements returns all achievements for a story, ordered by earned_at ascending.
func (db *DB) ListAchievements(storyID string) ([]Achievement, error) {
	rows, err := db.conn.Query(
		`SELECT id, story_id, name, description, category, rarity, context, earned_at
		 FROM achievements WHERE story_id = ? ORDER BY earned_at ASC`,
		storyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var achievements []Achievement
	for rows.Next() {
		var a Achievement
		if err := rows.Scan(
			&a.ID, &a.StoryID, &a.Name, &a.Description,
			&a.Category, &a.Rarity, &a.Context, &a.EarnedAt,
		); err != nil {
			return nil, err
		}
		achievements = append(achievements, a)
	}
	return achievements, rows.Err()
}
