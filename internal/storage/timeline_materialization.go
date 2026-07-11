package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type timelineValue struct {
	Kind  string  `json:"kind"`
	Text  string  `json:"text,omitempty"`
	Int   int64   `json:"int,omitempty"`
	Float float64 `json:"float,omitempty"`
	Blob  []byte  `json:"blob,omitempty"`
}

type timelineTable struct {
	Name    string            `json:"name"`
	Columns []string          `json:"columns"`
	Rows    [][]timelineValue `json:"rows"`
}

type timelineMaterialization struct {
	FormatVersion int             `json:"format_version"`
	StoryID       string          `json:"story_id"`
	BranchID      string          `json:"branch_id"`
	Tables        []timelineTable `json:"tables"`
}

type timelineTableSpec struct {
	name   string
	filter string
}

var timelineTableSpecs = []timelineTableSpec{
	{name: "stories", filter: "id = ?"},
	{name: "characters", filter: "story_id = ?"},
	{name: "world_state", filter: "story_id = ?"},
	{name: "npcs", filter: "story_id = ?"},
	{name: "achievements", filter: "story_id = ?"},
	{name: "sessions", filter: "story_id = ? AND (branch_id = ? OR branch_id = '')"},
	{name: "chapters", filter: "story_id = ? AND (branch_id = ? OR branch_id = '')"},
	{name: "chat_messages", filter: "story_id = ? AND (branch_id = ? OR branch_id = '')"},
	{name: "rag_chunks", filter: "story_id = ? AND (branch_id = ? OR branch_id = '')"},
	{name: "combat_log", filter: "story_id = ? AND (branch_id = ? OR branch_id = '')"},
}

var canonicalTableSpecs = []timelineTableSpec{
	{name: "canonical_entities", filter: "story_id = ? AND branch_id = ?"},
	{name: "entity_field_locks", filter: "entity_id IN (SELECT id FROM canonical_entities WHERE story_id = ?)"},
	{name: "entity_aliases", filter: "story_id = ? AND branch_id = ?"},
	{name: "identity_claims", filter: "story_id = ? AND branch_id = ?"},
	{name: "entity_forms", filter: "story_id = ? AND branch_id = ?"},
	{name: "entity_controller_events", filter: "story_id = ? AND branch_id = ?"},
	{name: "entity_lifecycle_events", filter: "story_id = ? AND branch_id = ?"},
	{name: "character_facts", filter: "story_id = ? AND branch_id = ?"},
	{name: "factions", filter: "story_id = ? AND branch_id = ?"},
	{name: "faction_membership_events", filter: "story_id = ? AND branch_id = ?"},
	{name: "faction_relationship_events", filter: "story_id = ? AND branch_id = ?"},
	{name: "reputation_events", filter: "story_id = ? AND branch_id = ?"},
	{name: "regions", filter: "story_id = ? AND branch_id = ?"},
	{name: "locations", filter: "story_id = ? AND branch_id = ?"},
	{name: "location_aliases", filter: "story_id = ? AND branch_id = ?"},
	{name: "location_edges", filter: "story_id = ? AND branch_id = ?"},
	{name: "entity_position_events", filter: "story_id = ? AND branch_id = ?"},
	{name: "world_calendars", filter: "story_id = ?"},
	{name: "world_clocks", filter: "story_id = ? AND branch_id = ?"},
	{name: "world_time_events", filter: "story_id = ? AND branch_id = ?"},
	{name: "weather_states", filter: "story_id = ? AND branch_id = ?"},
	{name: "canonical_world_events", filter: "story_id = ? AND branch_id = ?"},
	{name: "world_thread_events", filter: "story_id = ? AND branch_id = ?"},
}

func (db *DB) CaptureTimelineMaterializationTx(tx *sql.Tx, storyID, branchID string) (string, error) {
	return captureMaterializationTx(tx, storyID, branchID, append(append([]timelineTableSpec{}, timelineTableSpecs...), canonicalTableSpecs...))
}

func (db *DB) CaptureCanonicalState(storyID, branchID string) (string, error) {
	var payload string
	err := db.WithTx(func(tx *sql.Tx) error {
		var err error
		payload, err = captureMaterializationTx(tx, storyID, branchID, canonicalTableSpecs)
		return err
	})
	return payload, err
}

func captureMaterializationTx(tx *sql.Tx, storyID, branchID string, specs []timelineTableSpec) (string, error) {
	if tx == nil || storyID == "" || branchID == "" {
		return "", errors.New("transaction, story, and branch are required")
	}
	m := timelineMaterialization{FormatVersion: CurrentTurnSnapshotFormatVersion, StoryID: storyID, BranchID: branchID}
	for _, spec := range specs {
		args := []any{storyID}
		if strings.Contains(spec.filter, "branch_id = ?") {
			args = append(args, branchID)
		}
		table, err := captureTimelineTable(tx, spec.name, spec.filter, args...)
		if err != nil {
			return "", err
		}
		m.Tables = append(m.Tables, table)
	}
	payload, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("marshaling timeline materialization: %w", err)
	}
	return string(payload), nil
}

func (db *DB) RestoreCanonicalStateTx(tx *sql.Tx, storyID, branchID, payloadJSON string) error {
	var m timelineMaterialization
	if err := json.Unmarshal([]byte(payloadJSON), &m); err != nil {
		return err
	}
	if m.FormatVersion != CurrentTurnSnapshotFormatVersion || m.StoryID != storyID {
		return errors.New("canonical snapshot identity or format is incompatible")
	}
	byName := map[string]timelineTable{}
	for _, table := range m.Tables {
		byName[table.Name] = table
	}
	deleteOrder := []string{"entity_position_events", "location_edges", "location_aliases", "weather_states", "canonical_world_events", "world_thread_events", "world_time_events", "world_clocks", "world_calendars", "locations", "regions", "reputation_events", "faction_relationship_events", "faction_membership_events", "entity_controller_events", "entity_lifecycle_events", "character_facts", "identity_claims", "entity_aliases", "entity_field_locks", "entity_forms", "factions", "canonical_entities"}
	for _, table := range deleteOrder {
		if table == "entity_field_locks" {
			if _, err := tx.Exec(`DELETE FROM entity_field_locks WHERE entity_id IN (SELECT id FROM canonical_entities WHERE story_id=?)`, storyID); err != nil {
				return err
			}
			continue
		}
		if _, err := tx.Exec(fmt.Sprintf(`DELETE FROM %s WHERE story_id=?`, table), storyID); err != nil {
			return err
		}
	}
	restoreOrder := []string{"canonical_entities", "entity_field_locks", "entity_aliases", "identity_claims", "entity_forms", "entity_controller_events", "entity_lifecycle_events", "character_facts", "factions", "faction_membership_events", "faction_relationship_events", "reputation_events", "regions", "locations", "location_aliases", "location_edges", "world_calendars", "world_clocks", "world_time_events", "weather_states", "canonical_world_events", "world_thread_events", "entity_position_events"}
	for _, name := range restoreOrder {
		table, ok := byName[name]
		if !ok {
			return fmt.Errorf("canonical snapshot is missing table %s", name)
		}
		if err := restoreTimelineTable(tx, table, branchID); err != nil {
			return err
		}
	}
	return nil
}

func captureTimelineTable(tx *sql.Tx, table, filter string, args ...any) (timelineTable, error) {
	rows, err := tx.Query(fmt.Sprintf(`SELECT * FROM %s WHERE %s`, table, filter), args...)
	if err != nil {
		return timelineTable{}, fmt.Errorf("capturing %s: %w", table, err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return timelineTable{}, err
	}
	out := timelineTable{Name: table, Columns: columns}
	for rows.Next() {
		values := make([]any, len(columns))
		dest := make([]any, len(columns))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return timelineTable{}, err
		}
		encoded := make([]timelineValue, len(values))
		for i, value := range values {
			encoded[i], err = encodeTimelineValue(value)
			if err != nil {
				return timelineTable{}, fmt.Errorf("encoding %s.%s: %w", table, columns[i], err)
			}
		}
		out.Rows = append(out.Rows, encoded)
	}
	return out, rows.Err()
}

func encodeTimelineValue(value any) (timelineValue, error) {
	switch v := value.(type) {
	case nil:
		return timelineValue{Kind: "null"}, nil
	case int64:
		return timelineValue{Kind: "int", Int: v}, nil
	case float64:
		return timelineValue{Kind: "float", Float: v}, nil
	case bool:
		if v {
			return timelineValue{Kind: "int", Int: 1}, nil
		}
		return timelineValue{Kind: "int"}, nil
	case string:
		return timelineValue{Kind: "text", Text: v}, nil
	case []byte:
		return timelineValue{Kind: "blob", Blob: v}, nil
	case time.Time:
		return timelineValue{Kind: "time", Text: v.Format(time.RFC3339Nano)}, nil
	default:
		return timelineValue{}, fmt.Errorf("unsupported SQLite value %T", value)
	}
}

func decodeTimelineValue(value timelineValue) (any, error) {
	switch value.Kind {
	case "null":
		return nil, nil
	case "int":
		return value.Int, nil
	case "float":
		return value.Float, nil
	case "text":
		return value.Text, nil
	case "blob":
		return value.Blob, nil
	case "time":
		return value.Text, nil
	default:
		return nil, fmt.Errorf("unknown timeline value kind %q", value.Kind)
	}
}

func (db *DB) RestoreTimelineMaterializationTx(tx *sql.Tx, storyID, branchID, payloadJSON, expectedHash string) error {
	if expectedHash != "" {
		sum := sha256.Sum256([]byte(payloadJSON))
		if got := fmt.Sprintf("sha256:%x", sum[:]); got != expectedHash {
			return errors.New("turn snapshot checksum does not match its payload")
		}
	}
	var m timelineMaterialization
	if err := json.Unmarshal([]byte(payloadJSON), &m); err != nil {
		return fmt.Errorf("decoding turn snapshot: %w", err)
	}
	if m.FormatVersion != CurrentTurnSnapshotFormatVersion || m.StoryID != storyID {
		return errors.New("turn snapshot identity or format is incompatible")
	}
	byName := make(map[string]timelineTable, len(m.Tables))
	for _, table := range m.Tables {
		byName[table.Name] = table
	}

	for _, table := range []string{"entity_position_events", "location_edges", "location_aliases", "weather_states", "canonical_world_events", "world_thread_events", "world_time_events", "world_clocks", "world_calendars", "locations", "regions", "reputation_events", "faction_relationship_events", "faction_membership_events", "entity_controller_events", "entity_lifecycle_events", "character_facts", "identity_claims", "entity_aliases", "entity_field_locks", "entity_forms", "factions", "canonical_entities", "chat_messages", "combat_log", "rag_chunks", "chapters", "sessions", "achievements", "npcs", "world_state", "characters"} {
		if table == "entity_field_locks" {
			if _, err := tx.Exec(`DELETE FROM entity_field_locks WHERE entity_id IN (SELECT id FROM canonical_entities WHERE story_id=?)`, storyID); err != nil {
				return err
			}
			continue
		}
		if _, err := tx.Exec(fmt.Sprintf(`DELETE FROM %s WHERE story_id=?`, table), storyID); err != nil {
			return fmt.Errorf("clearing %s for checkout: %w", table, err)
		}
	}
	for _, table := range []string{"characters", "world_state", "npcs", "achievements", "sessions", "chapters", "chat_messages", "rag_chunks", "combat_log", "canonical_entities", "entity_field_locks", "entity_aliases", "identity_claims", "entity_forms", "entity_controller_events", "entity_lifecycle_events", "character_facts", "factions", "faction_membership_events", "faction_relationship_events", "reputation_events", "regions", "locations", "location_aliases", "location_edges", "world_calendars", "world_clocks", "world_time_events", "weather_states", "canonical_world_events", "world_thread_events", "entity_position_events"} {
		data, ok := byName[table]
		if !ok {
			return fmt.Errorf("turn snapshot is missing table %s", table)
		}
		if err := restoreTimelineTable(tx, data, branchID); err != nil {
			return err
		}
	}
	story, ok := byName["stories"]
	if !ok || len(story.Rows) != 1 {
		return errors.New("turn snapshot is missing story metadata")
	}
	return restoreStoryMetadata(tx, story, storyID)
}

func restoreTimelineTable(tx *sql.Tx, table timelineTable, branchID string) error {
	if len(table.Rows) == 0 {
		return nil
	}
	quoted := make([]string, len(table.Columns))
	marks := make([]string, len(table.Columns))
	for i, column := range table.Columns {
		quoted[i] = `"` + strings.ReplaceAll(column, `"`, `""`) + `"`
		marks[i] = "?"
	}
	stmt := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`, table.Name, strings.Join(quoted, ","), strings.Join(marks, ","))
	for _, row := range table.Rows {
		if len(row) != len(table.Columns) {
			return fmt.Errorf("invalid %s snapshot row width", table.Name)
		}
		args := make([]any, len(row))
		var err error
		for i, value := range row {
			args[i], err = decodeTimelineValue(value)
			if err != nil {
				return err
			}
			if table.Columns[i] == "branch_id" {
				args[i] = branchID
			}
		}
		if _, err := tx.Exec(stmt, args...); err != nil {
			return fmt.Errorf("restoring %s: %w", table.Name, err)
		}
	}
	return nil
}

func restoreStoryMetadata(tx *sql.Tx, table timelineTable, storyID string) error {
	set := []string{}
	args := []any{}
	for i, column := range table.Columns {
		if column == "id" || column == "revision" || column == "active_branch_id" || column == "updated_at" {
			continue
		}
		value, err := decodeTimelineValue(table.Rows[0][i])
		if err != nil {
			return err
		}
		set = append(set, `"`+column+`"=?`)
		args = append(args, value)
	}
	args = append(args, storyID)
	_, err := tx.Exec(`UPDATE stories SET `+strings.Join(set, ",")+` WHERE id=?`, args...)
	return err
}

func (db *DB) CheckoutStoryBranch(storyID, branchID string, expectedRevision int64) (*TimelineHead, error) {
	var out *TimelineHead
	err := db.WithTx(func(tx *sql.Tx) error {
		var activeBranchID string
		if err := tx.QueryRow(`SELECT active_branch_id FROM stories WHERE id=?`, storyID).Scan(&activeBranchID); err != nil {
			return err
		}
		var b StoryBranch
		var c TurnCommit
		var payload string
		err := tx.QueryRow(`SELECT b.id,b.story_id,b.name,COALESCE(b.fork_commit_id,''),COALESCE(b.head_commit_id,''),b.created_at,b.updated_at,
		 c.id,c.story_id,c.branch_id,COALESCE(c.parent_commit_id,''),c.canonical_turn,c.story_revision,c.payload_hash,c.kind,c.message,c.created_at,
		 COALESCE(s.payload_json,'') FROM story_branches b JOIN turn_commits c ON c.id=b.head_commit_id LEFT JOIN turn_snapshots s ON s.commit_id=c.id WHERE b.id=? AND b.story_id=?`, branchID, storyID).Scan(
			&b.ID, &b.StoryID, &b.Name, &b.ForkCommitID, &b.HeadCommitID, &b.CreatedAt, &b.UpdatedAt,
			&c.ID, &c.StoryID, &c.BranchID, &c.ParentCommitID, &c.CanonicalTurn, &c.StoryRevision, &c.PayloadHash, &c.Kind, &c.Message, &c.CreatedAt, &payload)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrBranchNotFound
		}
		if err != nil {
			return err
		}
		if activeBranchID == branchID {
			out = &TimelineHead{Branch: b, Commit: c}
			return nil
		}
		if err := db.RequireStoryRevisionTx(tx, storyID, expectedRevision); err != nil {
			return err
		}
		if payload == "" {
			return ErrCommitSnapshotMissing
		}
		if err := db.RestoreTimelineMaterializationTx(tx, storyID, branchID, payload, c.PayloadHash); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE stories SET active_branch_id=? WHERE id=?`, branchID, storyID); err != nil {
			return err
		}
		if err := db.ClearTurnIdempotencyTx(tx, storyID); err != nil {
			return err
		}
		revision, err := db.BumpStoryRevisionTx(tx, storyID)
		if err != nil {
			return err
		}
		c.StoryRevision = revision
		out = &TimelineHead{Branch: b, Commit: c}
		return nil
	})
	return out, err
}
