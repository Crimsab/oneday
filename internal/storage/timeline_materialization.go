package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
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

const deltaTurnSnapshotFormatVersion = 2

type timelineTableDelta struct {
	Name       string            `json:"name"`
	Columns    []string          `json:"columns,omitempty"`
	KeyColumns []string          `json:"key_columns,omitempty"`
	Upserts    [][]timelineValue `json:"upserts,omitempty"`
	Deletes    [][]timelineValue `json:"deletes,omitempty"`
	Replace    *timelineTable    `json:"replace,omitempty"`
}

type timelineSnapshotDelta struct {
	FormatVersion int                  `json:"format_version"`
	StoryID       string               `json:"story_id"`
	BranchID      string               `json:"branch_id"`
	BaseCommitID  string               `json:"base_commit_id"`
	Tables        []timelineTableDelta `json:"tables"`
}

type timelineTableState struct {
	columns    []string
	keyColumns []string
	rows       map[string][]timelineValue
	order      []string
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
	{name: "challenge_runs", filter: "story_id = ? AND branch_id = ?"},
}

func (db *DB) CaptureTimelineMaterializationTx(tx *sql.Tx, storyID, branchID string) (string, error) {
	return captureMaterializationTx(tx, storyID, branchID, append(append([]timelineTableSpec{}, timelineTableSpecs...), canonicalTableSpecs...))
}

// EnsureTurnSnapshotTx seals a bootstrap commit only when its immutable
// snapshot is still missing. The existence check deliberately precedes the
// expensive full-story materialization.
func (db *DB) EnsureTurnSnapshotTx(tx *sql.Tx, commitID, storyID, branchID string) error {
	if tx == nil || commitID == "" || storyID == "" || branchID == "" {
		return errors.New("transaction, commit, story, and branch are required")
	}
	var exists bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM turn_snapshots WHERE commit_id=?)`, commitID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	payload, err := db.CaptureTimelineMaterializationTx(tx, storyID, branchID)
	if err != nil {
		return err
	}
	return db.SealTurnSnapshotTx(tx, commitID, storyID, payload)
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

// encodeTurnSnapshotTx stores row-level changes against the immutable parent
// snapshot when that representation is smaller than another full copy. The
// caller still captures one coherent materialization; only durable storage and
// later reconstruction change.
func (db *DB) encodeTurnSnapshotTx(tx *sql.Tx, parentCommitID, payloadJSON string) (string, int, error) {
	var current timelineMaterialization
	if err := json.Unmarshal([]byte(payloadJSON), &current); err != nil || current.FormatVersion != CurrentTurnSnapshotFormatVersion || current.StoryID == "" || len(current.Tables) == 0 {
		return payloadJSON, CurrentTurnSnapshotFormatVersion, nil
	}
	var parentPayload, parentHash string
	if err := tx.QueryRow(`SELECT payload_json,payload_hash FROM turn_snapshots WHERE commit_id=? AND story_id=?`, parentCommitID, current.StoryID).Scan(&parentPayload, &parentHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return payloadJSON, CurrentTurnSnapshotFormatVersion, nil
		}
		return "", 0, err
	}
	parent, err := db.resolveTimelineMaterializationTx(tx, current.StoryID, parentPayload, parentHash)
	if err != nil {
		return "", 0, fmt.Errorf("resolving parent turn snapshot: %w", err)
	}
	delta, err := buildTimelineSnapshotDeltaTx(tx, parentCommitID, parent, current)
	if err != nil {
		return "", 0, err
	}
	encoded, err := json.Marshal(delta)
	if err != nil {
		return "", 0, fmt.Errorf("marshaling timeline delta: %w", err)
	}
	if len(encoded) >= len(payloadJSON) {
		return payloadJSON, CurrentTurnSnapshotFormatVersion, nil
	}
	return string(encoded), deltaTurnSnapshotFormatVersion, nil
}

func buildTimelineSnapshotDeltaTx(tx *sql.Tx, baseCommitID string, parent, current timelineMaterialization) (timelineSnapshotDelta, error) {
	delta := timelineSnapshotDelta{
		FormatVersion: deltaTurnSnapshotFormatVersion,
		StoryID:       current.StoryID,
		BranchID:      current.BranchID,
		BaseCommitID:  baseCommitID,
	}
	parentByName := make(map[string]timelineTable, len(parent.Tables))
	for _, table := range parent.Tables {
		parentByName[table.Name] = table
	}
	for _, table := range current.Tables {
		before, ok := parentByName[table.Name]
		if !ok || !reflect.DeepEqual(before.Columns, table.Columns) {
			replacement := table
			delta.Tables = append(delta.Tables, timelineTableDelta{Name: table.Name, Replace: &replacement})
			continue
		}
		keyColumns, err := timelinePrimaryKeyColumns(tx, table.Name, table.Columns)
		if err != nil {
			return timelineSnapshotDelta{}, err
		}
		change, err := diffTimelineTable(before, table, keyColumns)
		if err != nil {
			return timelineSnapshotDelta{}, err
		}
		if len(change.Upserts) > 0 || len(change.Deletes) > 0 {
			delta.Tables = append(delta.Tables, change)
		}
	}
	return delta, nil
}

func timelinePrimaryKeyColumns(tx *sql.Tx, table string, fallback []string) ([]string, error) {
	quoted := strings.ReplaceAll(table, `"`, `""`)
	rows, err := tx.Query(fmt.Sprintf(`PRAGMA table_info("%s")`, quoted))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type primaryColumn struct {
		name     string
		position int
	}
	var primary []primaryColumn
	for rows.Next() {
		var cid, notNull, position int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &position); err != nil {
			return nil, err
		}
		if position > 0 {
			primary = append(primary, primaryColumn{name: name, position: position})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(primary) == 0 {
		return append([]string(nil), fallback...), nil
	}
	sort.Slice(primary, func(i, j int) bool { return primary[i].position < primary[j].position })
	columns := make([]string, len(primary))
	for i := range primary {
		columns[i] = primary[i].name
	}
	return columns, nil
}

func diffTimelineTable(before, after timelineTable, keyColumns []string) (timelineTableDelta, error) {
	change := timelineTableDelta{Name: after.Name, Columns: after.Columns, KeyColumns: keyColumns}
	beforeRows := make(map[string][]timelineValue, len(before.Rows))
	for _, row := range before.Rows {
		key, _, err := timelineRowKey(before.Columns, keyColumns, row)
		if err != nil {
			return timelineTableDelta{}, fmt.Errorf("indexing parent %s: %w", before.Name, err)
		}
		beforeRows[key] = row
	}
	afterKeys := make(map[string]struct{}, len(after.Rows))
	for _, row := range after.Rows {
		key, _, err := timelineRowKey(after.Columns, keyColumns, row)
		if err != nil {
			return timelineTableDelta{}, fmt.Errorf("indexing current %s: %w", after.Name, err)
		}
		afterKeys[key] = struct{}{}
		if previous, ok := beforeRows[key]; !ok || !reflect.DeepEqual(previous, row) {
			change.Upserts = append(change.Upserts, row)
		}
	}
	for _, row := range before.Rows {
		key, values, err := timelineRowKey(before.Columns, keyColumns, row)
		if err != nil {
			return timelineTableDelta{}, err
		}
		if _, ok := afterKeys[key]; !ok {
			change.Deletes = append(change.Deletes, values)
		}
	}
	return change, nil
}

func timelineRowKey(columns, keyColumns []string, row []timelineValue) (string, []timelineValue, error) {
	if len(columns) != len(row) || len(keyColumns) == 0 {
		return "", nil, errors.New("timeline row has no usable key")
	}
	index := make(map[string]int, len(columns))
	for i, column := range columns {
		index[column] = i
	}
	values := make([]timelineValue, len(keyColumns))
	for i, column := range keyColumns {
		position, ok := index[column]
		if !ok {
			return "", nil, fmt.Errorf("timeline key column %s is missing", column)
		}
		values[i] = row[position]
	}
	encoded, err := json.Marshal(values)
	return string(encoded), values, err
}

func (db *DB) resolveTimelineMaterializationTx(tx *sql.Tx, storyID, payloadJSON, expectedHash string) (timelineMaterialization, error) {
	currentPayload, currentHash := payloadJSON, expectedHash
	deltas := []timelineSnapshotDelta{}
	seen := map[string]bool{}
	for {
		if currentHash != "" {
			sum := sha256.Sum256([]byte(currentPayload))
			if got := fmt.Sprintf("sha256:%x", sum[:]); got != currentHash {
				return timelineMaterialization{}, errors.New("turn snapshot checksum does not match its payload")
			}
		}
		var header struct {
			FormatVersion int `json:"format_version"`
		}
		if err := json.Unmarshal([]byte(currentPayload), &header); err != nil {
			return timelineMaterialization{}, fmt.Errorf("decoding turn snapshot header: %w", err)
		}
		switch header.FormatVersion {
		case CurrentTurnSnapshotFormatVersion:
			var base timelineMaterialization
			if err := json.Unmarshal([]byte(currentPayload), &base); err != nil {
				return timelineMaterialization{}, fmt.Errorf("decoding full turn snapshot: %w", err)
			}
			if base.StoryID != storyID {
				return timelineMaterialization{}, errors.New("turn snapshot story does not match")
			}
			return applyTimelineSnapshotDeltas(base, deltas)
		case deltaTurnSnapshotFormatVersion:
			var delta timelineSnapshotDelta
			if err := json.Unmarshal([]byte(currentPayload), &delta); err != nil {
				return timelineMaterialization{}, fmt.Errorf("decoding turn snapshot delta: %w", err)
			}
			if delta.StoryID != storyID || delta.BaseCommitID == "" {
				return timelineMaterialization{}, errors.New("turn snapshot delta identity is incompatible")
			}
			if seen[delta.BaseCommitID] {
				return timelineMaterialization{}, errors.New("turn snapshot delta chain contains a cycle")
			}
			seen[delta.BaseCommitID] = true
			deltas = append(deltas, delta)
			if len(deltas) > 10000 {
				return timelineMaterialization{}, errors.New("turn snapshot delta chain is too deep")
			}
			if err := tx.QueryRow(`SELECT payload_json,payload_hash FROM turn_snapshots WHERE commit_id=? AND story_id=?`, delta.BaseCommitID, storyID).Scan(&currentPayload, &currentHash); err != nil {
				return timelineMaterialization{}, fmt.Errorf("loading base turn snapshot %s: %w", delta.BaseCommitID, err)
			}
		default:
			return timelineMaterialization{}, fmt.Errorf("unsupported turn snapshot format %d", header.FormatVersion)
		}
	}
}

func applyTimelineSnapshotDeltas(base timelineMaterialization, newestFirst []timelineSnapshotDelta) (timelineMaterialization, error) {
	states := make(map[string]*timelineTableState, len(base.Tables))
	order := make([]string, 0, len(base.Tables))
	for _, table := range base.Tables {
		copyTable := table
		states[table.Name] = &timelineTableState{columns: copyTable.Columns, rows: map[string][]timelineValue{}, order: nil}
		states[table.Name].reset(copyTable)
		order = append(order, table.Name)
	}
	for index := len(newestFirst) - 1; index >= 0; index-- {
		delta := newestFirst[index]
		for _, change := range delta.Tables {
			if change.Replace != nil {
				state, exists := states[change.Name]
				if !exists {
					state = &timelineTableState{}
					states[change.Name] = state
					order = append(order, change.Name)
				}
				state.reset(*change.Replace)
				continue
			}
			state := states[change.Name]
			if state == nil {
				return timelineMaterialization{}, fmt.Errorf("turn snapshot delta references missing table %s", change.Name)
			}
			if err := state.apply(change); err != nil {
				return timelineMaterialization{}, fmt.Errorf("applying %s delta: %w", change.Name, err)
			}
		}
		base.StoryID = delta.StoryID
		base.BranchID = delta.BranchID
	}
	base.Tables = make([]timelineTable, 0, len(order))
	for _, name := range order {
		base.Tables = append(base.Tables, states[name].table(name))
	}
	return base, nil
}

func (state *timelineTableState) reset(table timelineTable) {
	state.columns = append([]string(nil), table.Columns...)
	state.keyColumns = nil
	state.rows = make(map[string][]timelineValue, len(table.Rows))
	state.order = make([]string, len(table.Rows))
	for i, row := range table.Rows {
		key := fmt.Sprintf("row:%d", i)
		state.rows[key] = row
		state.order[i] = key
	}
}

func (state *timelineTableState) apply(change timelineTableDelta) error {
	if !reflect.DeepEqual(state.columns, change.Columns) {
		return errors.New("timeline table columns changed without replacement")
	}
	if len(change.KeyColumns) == 0 {
		return errors.New("timeline delta has no key columns")
	}
	if !reflect.DeepEqual(state.keyColumns, change.KeyColumns) {
		rows := make(map[string][]timelineValue, len(state.rows))
		order := make([]string, 0, len(state.order))
		for _, oldKey := range state.order {
			row, ok := state.rows[oldKey]
			if !ok {
				continue
			}
			key, _, err := timelineRowKey(state.columns, change.KeyColumns, row)
			if err != nil {
				return err
			}
			if _, duplicate := rows[key]; duplicate {
				return errors.New("timeline table contains duplicate row keys")
			}
			rows[key] = row
			order = append(order, key)
		}
		state.rows = rows
		state.order = order
		state.keyColumns = append([]string(nil), change.KeyColumns...)
	}
	for _, values := range change.Deletes {
		encoded, err := json.Marshal(values)
		if err != nil {
			return err
		}
		delete(state.rows, string(encoded))
	}
	for _, row := range change.Upserts {
		key, _, err := timelineRowKey(state.columns, state.keyColumns, row)
		if err != nil {
			return err
		}
		if _, exists := state.rows[key]; !exists {
			state.order = append(state.order, key)
		}
		state.rows[key] = row
	}
	return nil
}

func (state *timelineTableState) table(name string) timelineTable {
	table := timelineTable{Name: name, Columns: append([]string(nil), state.columns...)}
	for _, key := range state.order {
		if row, ok := state.rows[key]; ok {
			table.Rows = append(table.Rows, row)
		}
	}
	return table
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
	deleteOrder := []string{"challenge_runs", "entity_position_events", "location_edges", "location_aliases", "weather_states", "canonical_world_events", "world_thread_events", "world_time_events", "world_clocks", "world_calendars", "locations", "regions", "reputation_events", "faction_relationship_events", "faction_membership_events", "entity_controller_events", "entity_lifecycle_events", "character_facts", "identity_claims", "entity_aliases", "entity_field_locks", "entity_forms", "factions", "canonical_entities"}
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
	restoreOrder := []string{"canonical_entities", "entity_field_locks", "entity_aliases", "identity_claims", "entity_forms", "entity_controller_events", "entity_lifecycle_events", "character_facts", "factions", "faction_membership_events", "faction_relationship_events", "reputation_events", "regions", "locations", "location_aliases", "location_edges", "world_calendars", "world_clocks", "world_time_events", "weather_states", "canonical_world_events", "world_thread_events", "entity_position_events", "challenge_runs"}
	for _, name := range restoreOrder {
		table, ok := byName[name]
		if !ok && name == "challenge_runs" {
			continue // snapshots produced before protocol v1 remain restorable
		}
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
	m, err := db.resolveTimelineMaterializationTx(tx, storyID, payloadJSON, expectedHash)
	if err != nil {
		return err
	}
	if m.FormatVersion != CurrentTurnSnapshotFormatVersion || m.StoryID != storyID {
		return errors.New("turn snapshot identity or format is incompatible")
	}
	byName := make(map[string]timelineTable, len(m.Tables))
	for _, table := range m.Tables {
		byName[table.Name] = table
	}

	for _, table := range []string{"challenge_runs", "entity_position_events", "location_edges", "location_aliases", "weather_states", "canonical_world_events", "world_thread_events", "world_time_events", "world_clocks", "world_calendars", "locations", "regions", "reputation_events", "faction_relationship_events", "faction_membership_events", "entity_controller_events", "entity_lifecycle_events", "character_facts", "identity_claims", "entity_aliases", "entity_field_locks", "entity_forms", "factions", "canonical_entities", "chat_messages", "combat_log", "rag_chunks", "chapters", "sessions", "achievements", "npcs", "world_state", "characters"} {
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
	for _, table := range []string{"characters", "world_state", "npcs", "achievements", "sessions", "chapters", "chat_messages", "rag_chunks", "combat_log", "canonical_entities", "entity_field_locks", "entity_aliases", "identity_claims", "entity_forms", "entity_controller_events", "entity_lifecycle_events", "character_facts", "factions", "faction_membership_events", "faction_relationship_events", "reputation_events", "regions", "locations", "location_aliases", "location_edges", "world_calendars", "world_clocks", "world_time_events", "weather_states", "canonical_world_events", "world_thread_events", "entity_position_events", "challenge_runs"} {
		data, ok := byName[table]
		if !ok && table == "challenge_runs" {
			continue
		}
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

// ForkAndCheckoutStoryBranch creates an alternate branch and restores its
// source commit in one transaction. A failed restore cannot leave an orphaned
// branch behind.
func (db *DB) ForkAndCheckoutStoryBranch(storyID, fromCommitID, name string, expectedRevision int64) (*StoryBranch, error) {
	name = strings.TrimSpace(name)
	if storyID == "" || fromCommitID == "" || name == "" {
		return nil, errors.New("story, source commit, and branch name are required")
	}
	var branch *StoryBranch
	err := db.WithTx(func(tx *sql.Tx) error {
		var existing StoryBranch
		readErr := tx.QueryRow(
			`SELECT id,story_id,name,COALESCE(fork_commit_id,''),COALESCE(head_commit_id,''),created_at,updated_at
			 FROM story_branches WHERE story_id=? AND name=?`, storyID, name,
		).Scan(&existing.ID, &existing.StoryID, &existing.Name, &existing.ForkCommitID, &existing.HeadCommitID, &existing.CreatedAt, &existing.UpdatedAt)
		if readErr == nil {
			if existing.ForkCommitID != fromCommitID {
				return fmt.Errorf("branch name %q already exists", name)
			}
			branch = &existing
		} else if !errors.Is(readErr, sql.ErrNoRows) {
			return readErr
		}

		var activeBranchID string
		if err := tx.QueryRow(`SELECT active_branch_id FROM stories WHERE id=?`, storyID).Scan(&activeBranchID); err != nil {
			return err
		}
		if branch != nil && activeBranchID == branch.ID {
			return nil
		}
		if err := db.RequireStoryRevisionTx(tx, storyID, expectedRevision); err != nil {
			return err
		}
		var payload, payloadHash string
		if err := tx.QueryRow(
			`SELECT COALESCE(s.payload_json,''),c.payload_hash
			 FROM turn_commits c LEFT JOIN turn_snapshots s ON s.commit_id=c.id
			 WHERE c.id=? AND c.story_id=?`, fromCommitID, storyID,
		).Scan(&payload, &payloadHash); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrCommitNotFound
			}
			return err
		}
		if payload == "" {
			return ErrCommitSnapshotMissing
		}

		if branch == nil {
			now := time.Now().UTC()
			branch = &StoryBranch{
				ID: uuid.NewString(), StoryID: storyID, Name: name,
				ForkCommitID: fromCommitID, HeadCommitID: fromCommitID,
				CreatedAt: now, UpdatedAt: now,
			}
			if _, err := tx.Exec(
				`INSERT INTO story_branches (id,story_id,name,fork_commit_id,head_commit_id,created_at,updated_at)
				 VALUES (?,?,?,?,?,?,?)`,
				branch.ID, branch.StoryID, branch.Name, branch.ForkCommitID,
				branch.HeadCommitID, branch.CreatedAt, branch.UpdatedAt,
			); err != nil {
				return fmt.Errorf("creating and checking out branch: %w", err)
			}
		}

		if err := db.RestoreTimelineMaterializationTx(tx, storyID, branch.ID, payload, payloadHash); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE stories SET active_branch_id=? WHERE id=?`, branch.ID, storyID); err != nil {
			return err
		}
		if err := db.ClearTurnIdempotencyTx(tx, storyID); err != nil {
			return err
		}
		_, err := db.BumpStoryRevisionTx(tx, storyID)
		return err
	})
	return branch, err
}
