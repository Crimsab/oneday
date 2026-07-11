package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type CanonicalLocation struct {
	ID               string `json:"id"`
	StoryID          string `json:"story_id"`
	Name             string `json:"name"`
	RegionID         string `json:"region_id,omitempty"`
	ParentLocationID string `json:"parent_location_id,omitempty"`
	Description      string `json:"description"`
	DiscoveryState   string `json:"discovery_state"`
	DiscoveredTurn   int    `json:"discovered_turn"`
	Visibility       string `json:"visibility"`
	BranchID         string `json:"branch_id"`
	SourceCommitID   string `json:"source_commit_id"`
}
type LocationEdge struct {
	ID             string `json:"id"`
	StoryID        string `json:"story_id"`
	FromLocationID string `json:"from_location_id"`
	ToLocationID   string `json:"to_location_id"`
	Direction      string `json:"direction"`
	TravelMinutes  int    `json:"travel_minutes"`
	ConditionsJSON string `json:"conditions_json"`
	Visibility     string `json:"visibility"`
	BranchID       string `json:"branch_id"`
	SourceCommitID string `json:"source_commit_id"`
}
type WorldClock struct {
	StoryID     string `json:"story_id"`
	Day         int    `json:"day"`
	MinuteOfDay int    `json:"minute_of_day"`
	DisplayText string `json:"display_text"`
}
type WeatherProjection struct {
	Tracked     bool   `json:"tracked"`
	Label       string `json:"label"`
	Intensity   string `json:"intensity,omitempty"`
	Description string `json:"description,omitempty"`
}
type PlayerWorldProjection struct {
	CurrentLocationID string              `json:"current_location_id"`
	CurrentLocation   string              `json:"current_location"`
	Locations         []CanonicalLocation `json:"locations"`
	Edges             []LocationEdge      `json:"edges"`
	Clock             WorldClock          `json:"clock"`
	Weather           WeatherProjection   `json:"weather"`
}
type CanonicalWorldEvent struct {
	ID              string
	StoryID         string
	Kind            string
	Title           string
	DetailsJSON     string
	LocationID      string
	FactionID       string
	EntityID        string
	CausedByEventID string
	Turn            int
	Visibility      string
	BranchID        string
	SourceCommitID  string
}
type WorldThreadEvent struct {
	ID             string
	StoryID        string
	ThreadID       string
	Title          string
	Status         string
	Pressure       int
	DetailsJSON    string
	Visibility     string
	Turn           int
	BranchID       string
	SourceCommitID string
}

func (db *DB) syncWorldCompatibilityTx(tx *sql.Tx, ws *WorldState) error {
	b, c, err := activeLineageTx(tx, ws.StoryID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT OR IGNORE INTO world_calendars (story_id,name,config_json) VALUES (?,'Default calendar','{"hours_per_day":24,"minutes_per_hour":60}')`, ws.StoryID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT OR IGNORE INTO world_clocks (story_id,calendar_story_id,day,minute_of_day,display_text,branch_id,source_commit_id) VALUES (?,?,0,0,'Day 0, 00:00',?,?)`, ws.StoryID, ws.StoryID, b, c)
	if err != nil {
		return err
	}
	if strings.TrimSpace(ws.CurrentLocation) != "" {
		l, err := ensureLocationTx(tx, ws.StoryID, ws.CurrentLocation, ws.CurrentTurn)
		if err != nil {
			return err
		}
		ws.CurrentLocationID = l.ID
		if _, err = tx.Exec(`UPDATE world_state SET current_location_id=? WHERE id=?`, l.ID, ws.ID); err != nil {
			return err
		}
	}
	var values []any
	if json.Unmarshal([]byte(ws.KnownLocationsJSON), &values) == nil {
		for _, v := range values {
			name := ""
			switch item := v.(type) {
			case string:
				name = item
			case map[string]any:
				name, _ = item["name"].(string)
			}
			if strings.TrimSpace(name) != "" {
				_, _ = ensureLocationTx(tx, ws.StoryID, name, ws.CurrentTurn)
			}
		}
	}
	return nil
}

func ensureLocationTx(tx *sql.Tx, storyID, name string, turn int) (*CanonicalLocation, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("location name required")
	}
	var l CanonicalLocation
	err := tx.QueryRow(`SELECT id,story_id,canonical_name,COALESCE(region_id,''),COALESCE(parent_location_id,''),description,discovery_state,discovered_turn,visibility,branch_id,source_commit_id FROM locations WHERE story_id=? AND lower(canonical_name)=lower(?) LIMIT 1`, storyID, name).Scan(&l.ID, &l.StoryID, &l.Name, &l.RegionID, &l.ParentLocationID, &l.Description, &l.DiscoveryState, &l.DiscoveredTurn, &l.Visibility, &l.BranchID, &l.SourceCommitID)
	if err == nil {
		return &l, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	b, c, err := activeLineageTx(tx, storyID)
	if err != nil {
		return nil, err
	}
	l = CanonicalLocation{ID: uuid.NewString(), StoryID: storyID, Name: name, DiscoveryState: "discovered", DiscoveredTurn: turn, Visibility: "player", BranchID: b, SourceCommitID: c}
	_, err = tx.Exec(`INSERT INTO locations (id,story_id,canonical_name,discovery_state,discovered_turn,visibility,branch_id,source_commit_id) VALUES (?,?,?,?,?,?,?,?)`, l.ID, l.StoryID, l.Name, l.DiscoveryState, l.DiscoveredTurn, l.Visibility, b, c)
	if err == nil {
		_, _ = tx.Exec(`INSERT OR IGNORE INTO location_aliases (id,story_id,location_id,alias,visibility,branch_id,source_commit_id) VALUES (?,?,?,?,'player',?,?)`, uuid.NewString(), storyID, l.ID, name, b, c)
	}
	return &l, err
}
func (db *DB) EnsureLocation(storyID, name string, turn int) (*CanonicalLocation, error) {
	var l *CanonicalLocation
	err := db.WithTx(func(tx *sql.Tx) error { var e error; l, e = ensureLocationTx(tx, storyID, name, turn); return e })
	return l, err
}
func (db *DB) SetCurrentLocation(storyID, entityID, name string, turn int) error {
	return db.WithTx(func(tx *sql.Tx) error {
		l, err := ensureLocationTx(tx, storyID, name, turn)
		if err != nil {
			return err
		}
		b, c, err := activeLineageTx(tx, storyID)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(`UPDATE world_state SET current_location=?,current_location_id=? WHERE story_id=?`, l.Name, l.ID, storyID); err != nil {
			return err
		}
		_, err = tx.Exec(`INSERT INTO entity_position_events (id,story_id,entity_id,location_id,event_kind,turn,visibility,branch_id,source_commit_id) VALUES (?,?,?,?, 'arrived',?,'player',?,?)`, uuid.NewString(), storyID, entityID, l.ID, turn, b, c)
		return err
	})
}
func (db *DB) ConnectLocations(edge *LocationEdge) error {
	if edge == nil {
		return errors.New("edge required")
	}
	return db.WithTx(func(tx *sql.Tx) error {
		b, c, e := activeLineageTx(tx, edge.StoryID)
		if e != nil {
			return e
		}
		if edge.ID == "" {
			edge.ID = uuid.NewString()
		}
		if edge.ConditionsJSON == "" {
			edge.ConditionsJSON = "{}"
		}
		if !json.Valid([]byte(edge.ConditionsJSON)) {
			return errors.New("edge conditions must be JSON")
		}
		if edge.Visibility == "" {
			edge.Visibility = "player"
		}
		edge.BranchID = b
		edge.SourceCommitID = c
		_, e = tx.Exec(`INSERT INTO location_edges (id,story_id,from_location_id,to_location_id,direction,travel_minutes,conditions_json,visibility,branch_id,source_commit_id) VALUES (?,?,?,?,?,?,?,?,?,?)`, edge.ID, edge.StoryID, edge.FromLocationID, edge.ToLocationID, edge.Direction, edge.TravelMinutes, edge.ConditionsJSON, edge.Visibility, b, c)
		return e
	})
}
func (db *DB) AdvanceWorldTime(storyID, reason string, deltaMinutes, turn int) (WorldClock, error) {
	var clock WorldClock
	err := db.WithTx(func(tx *sql.Tx) error {
		var err error
		clock, err = db.AdvanceWorldTimeTx(tx, storyID, reason, deltaMinutes, turn)
		return err
	})
	return clock, err
}
func (db *DB) AdvanceWorldTimeTx(tx *sql.Tx, storyID, reason string, deltaMinutes, turn int) (WorldClock, error) {
	var clock WorldClock
	var day, minute int
	if err := tx.QueryRow(`SELECT day,minute_of_day FROM world_clocks WHERE story_id=?`, storyID).Scan(&day, &minute); err != nil {
		return clock, err
	}
	total := day*1440 + minute + deltaMinutes
	if total < 0 {
		total = 0
	}
	toDay, toMinute := total/1440, total%1440
	display := fmt.Sprintf("Day %d, %02d:%02d", toDay, toMinute/60, toMinute%60)
	b, c, err := activeLineageTx(tx, storyID)
	if err != nil {
		return clock, err
	}
	if _, err = tx.Exec(`UPDATE world_clocks SET day=?,minute_of_day=?,display_text=?,branch_id=?,source_commit_id=?,updated_at=? WHERE story_id=?`, toDay, toMinute, display, b, c, time.Now().UTC(), storyID); err != nil {
		return clock, err
	}
	_, err = tx.Exec(`INSERT INTO world_time_events (id,story_id,delta_minutes,reason,turn,from_day,from_minute,to_day,to_minute,branch_id,source_commit_id) VALUES (?,?,?,?,?,?,?,?,?,?,?)`, uuid.NewString(), storyID, deltaMinutes, reason, turn, day, minute, toDay, toMinute, b, c)
	clock = WorldClock{StoryID: storyID, Day: toDay, MinuteOfDay: toMinute, DisplayText: display}
	return clock, err
}

func (db *DB) RecordWorldEvent(e *CanonicalWorldEvent) error {
	if e == nil {
		return errors.New("world event required")
	}
	return db.WithTx(func(tx *sql.Tx) error {
		b, c, err := activeLineageTx(tx, e.StoryID)
		if err != nil {
			return err
		}
		var day, minute int
		if err = tx.QueryRow(`SELECT day,minute_of_day FROM world_clocks WHERE story_id=?`, e.StoryID).Scan(&day, &minute); err != nil {
			return err
		}
		if e.ID == "" {
			e.ID = uuid.NewString()
		}
		if e.DetailsJSON == "" {
			e.DetailsJSON = "{}"
		}
		if e.Visibility == "" {
			e.Visibility = "private"
		}
		e.BranchID = b
		e.SourceCommitID = c
		_, err = tx.Exec(`INSERT INTO canonical_world_events (id,story_id,event_kind,title,details_json,location_id,faction_id,entity_id,caused_by_event_id,turn,world_day,world_minute,visibility,branch_id,source_commit_id) VALUES (?,?,?,?,?,NULLIF(?,''),NULLIF(?,''),NULLIF(?,''),NULLIF(?,''),?,?,?,?,?,?)`, e.ID, e.StoryID, e.Kind, e.Title, e.DetailsJSON, e.LocationID, e.FactionID, e.EntityID, e.CausedByEventID, e.Turn, day, minute, e.Visibility, b, c)
		return err
	})
}
func (db *DB) RecordWorldThread(e *WorldThreadEvent) error {
	if e == nil {
		return errors.New("world thread event required")
	}
	return db.WithTx(func(tx *sql.Tx) error {
		b, c, err := activeLineageTx(tx, e.StoryID)
		if err != nil {
			return err
		}
		if e.ID == "" {
			e.ID = uuid.NewString()
		}
		if e.DetailsJSON == "" {
			e.DetailsJSON = "{}"
		}
		if e.Visibility == "" {
			e.Visibility = "private"
		}
		e.BranchID = b
		e.SourceCommitID = c
		_, err = tx.Exec(`INSERT INTO world_thread_events (id,story_id,thread_id,title,status,pressure,details_json,visibility,turn,branch_id,source_commit_id) VALUES (?,?,?,?,?,?,?,?,?,?,?)`, e.ID, e.StoryID, e.ThreadID, e.Title, e.Status, e.Pressure, e.DetailsJSON, e.Visibility, e.Turn, b, c)
		return err
	})
}
func (db *DB) RecordWeather(storyID, locationID, kind, intensity, description string, fromDay, fromMinute int) error {
	return db.WithTx(func(tx *sql.Tx) error {
		b, c, e := activeLineageTx(tx, storyID)
		if e != nil {
			return e
		}
		_, e = tx.Exec(`INSERT INTO weather_states (id,story_id,location_id,weather_kind,intensity,description,valid_from_day,valid_from_minute,visibility,branch_id,source_commit_id) VALUES (?,?,?,?,?,?,?,?,'player',?,?)`, uuid.NewString(), storyID, locationID, kind, intensity, description, fromDay, fromMinute, b, c)
		return e
	})
}
func (db *DB) PlayerWorld(storyID string) (*PlayerWorldProjection, error) {
	var p PlayerWorldProjection
	if err := db.conn.QueryRow(`SELECT current_location_id,current_location FROM world_state WHERE story_id=?`, storyID).Scan(&p.CurrentLocationID, &p.CurrentLocation); err != nil {
		return nil, err
	}
	if err := db.conn.QueryRow(`SELECT story_id,day,minute_of_day,display_text FROM world_clocks WHERE story_id=?`, storyID).Scan(&p.Clock.StoryID, &p.Clock.Day, &p.Clock.MinuteOfDay, &p.Clock.DisplayText); err != nil {
		return nil, err
	}
	rows, err := db.conn.Query(`SELECT id,story_id,canonical_name,COALESCE(region_id,''),COALESCE(parent_location_id,''),description,discovery_state,discovered_turn,visibility,branch_id,source_commit_id FROM locations WHERE story_id=? AND visibility IN ('public','player') AND discovery_state!='unknown' ORDER BY canonical_name`, storyID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var l CanonicalLocation
		if rows.Scan(&l.ID, &l.StoryID, &l.Name, &l.RegionID, &l.ParentLocationID, &l.Description, &l.DiscoveryState, &l.DiscoveredTurn, &l.Visibility, &l.BranchID, &l.SourceCommitID) == nil {
			p.Locations = append(p.Locations, l)
		}
	}
	rows.Close()
	edgeRows, err := db.conn.Query(`SELECT id,story_id,from_location_id,to_location_id,direction,travel_minutes,conditions_json,visibility,branch_id,source_commit_id FROM location_edges WHERE story_id=? AND visibility IN ('public','player')`, storyID)
	if err != nil {
		return nil, err
	}
	for edgeRows.Next() {
		var e LocationEdge
		if edgeRows.Scan(&e.ID, &e.StoryID, &e.FromLocationID, &e.ToLocationID, &e.Direction, &e.TravelMinutes, &e.ConditionsJSON, &e.Visibility, &e.BranchID, &e.SourceCommitID) == nil {
			p.Edges = append(p.Edges, e)
		}
	}
	edgeRows.Close()
	p.Weather = WeatherProjection{Label: "Not tracked"}
	var kind, intensity, description string
	err = db.conn.QueryRow(`SELECT weather_kind,intensity,description FROM weather_states WHERE story_id=? AND (location_id=? OR location_id IS NULL) AND visibility IN ('public','player') AND (valid_to_day IS NULL OR valid_to_day>? OR (valid_to_day=? AND valid_to_minute>=?)) ORDER BY valid_from_day DESC,valid_from_minute DESC LIMIT 1`, storyID, p.CurrentLocationID, p.Clock.Day, p.Clock.Day, p.Clock.MinuteOfDay).Scan(&kind, &intensity, &description)
	if err == nil {
		p.Weather = WeatherProjection{Tracked: true, Label: kind, Intensity: intensity, Description: description}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return &p, nil
}
