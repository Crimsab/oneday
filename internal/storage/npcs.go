package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// CreateNPC inserts a new NPC into the database.
func (db *DB) CreateNPC(npc *NPC) error {
	_, err := db.conn.Exec(
		`INSERT INTO npcs (id, story_id, name, role, appearance, personality_json, private_thoughts,
         notes_on_protagonist, desires, disposition, is_alive, first_appeared_turn, last_seen_turn,
         can_help, created_at, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		npc.ID, npc.StoryID, npc.Name, npc.Role, npc.Appearance, npc.PersonalityJSON,
		npc.PrivateThoughts, npc.NotesOnProtagonist, npc.Desires, npc.Disposition,
		npc.IsAlive, npc.FirstAppearedTurn, npc.LastSeenTurn, npc.CanHelp,
		npc.CreatedAt, npc.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting npc: %w", err)
	}
	return nil
}

// GetNPC retrieves an NPC by ID.
func (db *DB) GetNPC(id string) (*NPC, error) {
	npc := &NPC{}
	err := db.conn.QueryRow(
		`SELECT id, story_id, name, role, appearance, personality_json, private_thoughts,
         notes_on_protagonist, desires, disposition, is_alive, first_appeared_turn, last_seen_turn,
         can_help, created_at, updated_at
         FROM npcs WHERE id = ?`, id,
	).Scan(
		&npc.ID, &npc.StoryID, &npc.Name, &npc.Role, &npc.Appearance, &npc.PersonalityJSON,
		&npc.PrivateThoughts, &npc.NotesOnProtagonist, &npc.Desires, &npc.Disposition,
		&npc.IsAlive, &npc.FirstAppearedTurn, &npc.LastSeenTurn, &npc.CanHelp,
		&npc.CreatedAt, &npc.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("getting npc %s: %w", id, err)
	}
	return npc, nil
}

// GetNPCByName retrieves an NPC by name within a story (case-insensitive).
// Returns nil, nil if not found.
func (db *DB) GetNPCByName(storyID, name string) (*NPC, error) {
	npc := &NPC{}
	err := db.conn.QueryRow(
		`SELECT id, story_id, name, role, appearance, personality_json, private_thoughts,
         notes_on_protagonist, desires, disposition, is_alive, first_appeared_turn, last_seen_turn,
         can_help, created_at, updated_at
         FROM npcs WHERE story_id = ? AND LOWER(name) = LOWER(?) AND is_alive = 1`, storyID, name,
	).Scan(
		&npc.ID, &npc.StoryID, &npc.Name, &npc.Role, &npc.Appearance, &npc.PersonalityJSON,
		&npc.PrivateThoughts, &npc.NotesOnProtagonist, &npc.Desires, &npc.Disposition,
		&npc.IsAlive, &npc.FirstAppearedTurn, &npc.LastSeenTurn, &npc.CanHelp,
		&npc.CreatedAt, &npc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting npc by name %q in story %s: %w", name, storyID, err)
	}
	return npc, nil
}

// ListNPCs returns all NPCs for a story, ordered by first_appeared_turn.
func (db *DB) ListNPCs(storyID string) ([]NPC, error) {
	rows, err := db.conn.Query(
		`SELECT id, story_id, name, role, appearance, personality_json, private_thoughts,
         notes_on_protagonist, desires, disposition, is_alive, first_appeared_turn, last_seen_turn,
         can_help, created_at, updated_at
         FROM npcs WHERE story_id = ? ORDER BY first_appeared_turn ASC`, storyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing npcs for story %s: %w", storyID, err)
	}
	defer rows.Close()

	var npcs []NPC
	for rows.Next() {
		var npc NPC
		if err := rows.Scan(
			&npc.ID, &npc.StoryID, &npc.Name, &npc.Role, &npc.Appearance, &npc.PersonalityJSON,
			&npc.PrivateThoughts, &npc.NotesOnProtagonist, &npc.Desires, &npc.Disposition,
			&npc.IsAlive, &npc.FirstAppearedTurn, &npc.LastSeenTurn, &npc.CanHelp,
			&npc.CreatedAt, &npc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning npc: %w", err)
		}
		npcs = append(npcs, npc)
	}
	return npcs, rows.Err()
}

// ListRecentNPCs returns NPCs seen within the last N turns (last_seen_turn >= currentTurn - withinTurns).
func (db *DB) ListRecentNPCs(storyID string, currentTurn, withinTurns int) ([]NPC, error) {
	threshold := currentTurn - withinTurns
	rows, err := db.conn.Query(
		`SELECT id, story_id, name, role, appearance, personality_json, private_thoughts,
         notes_on_protagonist, desires, disposition, is_alive, first_appeared_turn, last_seen_turn,
         can_help, created_at, updated_at
         FROM npcs WHERE story_id = ? AND last_seen_turn >= ? AND is_alive = 1
         ORDER BY last_seen_turn DESC`, storyID, threshold,
	)
	if err != nil {
		return nil, fmt.Errorf("listing recent npcs for story %s: %w", storyID, err)
	}
	defer rows.Close()

	var npcs []NPC
	for rows.Next() {
		var npc NPC
		if err := rows.Scan(
			&npc.ID, &npc.StoryID, &npc.Name, &npc.Role, &npc.Appearance, &npc.PersonalityJSON,
			&npc.PrivateThoughts, &npc.NotesOnProtagonist, &npc.Desires, &npc.Disposition,
			&npc.IsAlive, &npc.FirstAppearedTurn, &npc.LastSeenTurn, &npc.CanHelp,
			&npc.CreatedAt, &npc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning npc: %w", err)
		}
		npcs = append(npcs, npc)
	}
	return npcs, rows.Err()
}

// UpdateNPC updates all mutable NPC fields.
func (db *DB) UpdateNPC(npc *NPC) error {
	_, err := db.conn.Exec(
		`UPDATE npcs SET personality_json = ?, private_thoughts = ?, notes_on_protagonist = ?,
         desires = ?, disposition = ?, is_alive = ?, can_help = ?, last_seen_turn = ?,
         updated_at = ? WHERE id = ?`,
		npc.PersonalityJSON, npc.PrivateThoughts, npc.NotesOnProtagonist,
		npc.Desires, npc.Disposition, npc.IsAlive, npc.CanHelp, npc.LastSeenTurn,
		time.Now(), npc.ID,
	)
	if err != nil {
		return fmt.Errorf("updating npc %s: %w", npc.ID, err)
	}
	return nil
}

// UpdateNPCDisposition updates only the disposition and updated_at.
func (db *DB) UpdateNPCDisposition(npcID string, disposition int) error {
	_, err := db.conn.Exec(
		`UPDATE npcs SET disposition = ?, updated_at = ? WHERE id = ?`,
		disposition, time.Now(), npcID,
	)
	if err != nil {
		return fmt.Errorf("updating disposition for npc %s: %w", npcID, err)
	}
	return nil
}
