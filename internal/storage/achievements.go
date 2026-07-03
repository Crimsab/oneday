package storage

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

// CreateAchievement inserts a new achievement for a story.
func (db *DB) CreateAchievement(a *Achievement) error {
	return createAchievementExec(db.conn, a)
}

func (db *DB) CreateAchievementTx(tx *sql.Tx, a *Achievement) error {
	return createAchievementExec(tx, a)
}

func createAchievementExec(exec sqlExecer, a *Achievement) error {
	a.EarnedAt = time.Now()
	result, err := exec.Exec(
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

// AchievementExistsByName reports whether a story already contains an
// achievement with the given name, using case-insensitive matching.
func (db *DB) AchievementExistsByName(storyID, name string) (bool, error) {
	return achievementExistsByNameQuery(db.conn, storyID, name)
}

func (db *DB) AchievementExistsByNameTx(tx *sql.Tx, storyID, name string) (bool, error) {
	return achievementExistsByNameQuery(tx, storyID, name)
}

func achievementExistsByNameQuery(queryer interface {
	QueryRow(query string, args ...any) *sql.Row
}, storyID, name string) (bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return false, nil
	}

	var id int64
	err := queryer.QueryRow(
		`SELECT id FROM achievements
		 WHERE story_id = ? AND name = ? COLLATE NOCASE
		 LIMIT 1`,
		storyID, name,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
