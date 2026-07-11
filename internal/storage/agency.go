package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

type AgencyEventView struct {
	ID            int64  `json:"id"`
	StoryID       string `json:"story_id"`
	BranchID      string `json:"branch_id"`
	CommitID      string `json:"commit_id"`
	CanonicalTurn int    `json:"canonical_turn"`
	EntityID      string `json:"entity_id"`
	EntityName    string `json:"entity_name"`
	Action        string `json:"action"`
	Summary       string `json:"summary"`
	CreatedAt     string `json:"created_at"`
}

// PlanOffscreenAgencyEventsTx deterministically emits at most maxEvents public,
// provenance-bearing NPC actions. It never mutates NPC private state directly.
func (db *DB) PlanOffscreenAgencyEventsTx(tx *sql.Tx, storyID, branchID string, turn, maxEvents int) ([]CanonicalEventInput, error) {
	if tx == nil || maxEvents <= 0 || turn <= 0 {
		return nil, nil
	}
	if maxEvents > 3 {
		maxEvents = 3
	}
	rows, err := tx.Query(`SELECT n.canonical_entity_id,n.name,n.role,n.disposition
		FROM npcs n JOIN canonical_entities e ON e.id=n.canonical_entity_id
		WHERE n.story_id=? AND n.is_alive=1 AND n.last_seen_turn<? AND e.branch_id=?
		AND NOT EXISTS (
			SELECT 1 FROM canonical_events ce JOIN turn_commits tc ON tc.id=ce.commit_id
			WHERE ce.story_id=? AND ce.branch_id=? AND ce.event_type='npc.agency'
			AND json_extract(ce.payload_json,'$.entity_id')=n.canonical_entity_id
			AND tc.canonical_turn>?
		)
		ORDER BY n.last_seen_turn ASC,n.canonical_entity_id ASC LIMIT ?`, storyID, turn, branchID, storyID, branchID, turn-3, maxEvents)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []CanonicalEventInput{}
	for rows.Next() {
		var entityID, name, role string
		var disposition int
		if err := rows.Scan(&entityID, &name, &role, &disposition); err != nil {
			return nil, err
		}
		action, summary := "pursues_goal", fmt.Sprintf("%s advances an offscreen goal.", name)
		if disposition >= 35 {
			action, summary = "prepares_help", fmt.Sprintf("%s prepares help away from the current scene.", name)
		} else if disposition <= -35 {
			action, summary = "advances_pressure", fmt.Sprintf("%s advances pressure away from the current scene.", name)
		}
		payload, _ := json.Marshal(map[string]any{"version": 1, "entity_id": entityID, "entity_name": name, "role": role, "action": action, "summary": summary, "turn": turn, "bounded": true})
		events = append(events, CanonicalEventInput{Type: "npc.agency", PayloadJSON: string(payload)})
	}
	return events, rows.Err()
}

func (db *DB) ListRecentAgencyEvents(storyID string, limit int) ([]AgencyEventView, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := db.conn.Query(`SELECT ce.id,ce.story_id,ce.branch_id,ce.commit_id,tc.canonical_turn,
		COALESCE(json_extract(ce.payload_json,'$.entity_id'),''),COALESCE(json_extract(ce.payload_json,'$.entity_name'),''),
		COALESCE(json_extract(ce.payload_json,'$.action'),''),COALESCE(json_extract(ce.payload_json,'$.summary'),''),CAST(ce.created_at AS TEXT)
		FROM canonical_events ce JOIN turn_commits tc ON tc.id=ce.commit_id JOIN stories s ON s.id=ce.story_id
		WHERE ce.story_id=? AND ce.branch_id=s.active_branch_id AND ce.event_type='npc.agency'
		ORDER BY tc.canonical_turn DESC,ce.sequence DESC LIMIT ?`, storyID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []AgencyEventView{}
	for rows.Next() {
		var event AgencyEventView
		if err := rows.Scan(&event.ID, &event.StoryID, &event.BranchID, &event.CommitID, &event.CanonicalTurn, &event.EntityID, &event.EntityName, &event.Action, &event.Summary, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}
