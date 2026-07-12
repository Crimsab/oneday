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
	Kind             string `json:"kind"`
	RegionID         string `json:"region_id,omitempty"`
	ParentLocationID string `json:"parent_location_id,omitempty"`
	Description      string `json:"description"`
	DiscoveryState   string `json:"discovery_state"`
	DiscoveredTurn   int    `json:"discovered_turn"`
	Visibility       string `json:"visibility"`
	BranchID         string `json:"branch_id"`
	SourceCommitID   string `json:"source_commit_id"`
}
type CanonicalRegion struct {
	ID             string `json:"id"`
	StoryID        string `json:"story_id"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	ParentRegionID string `json:"parent_region_id,omitempty"`
	Visibility     string `json:"visibility"`
	BranchID       string `json:"branch_id"`
	SourceCommitID string `json:"source_commit_id"`
}
type LocationEdge struct {
	ID             string `json:"id"`
	StoryID        string `json:"story_id"`
	FromLocationID string `json:"from_location_id"`
	ToLocationID   string `json:"to_location_id"`
	Direction      string `json:"direction"`
	TravelMinutes  int    `json:"travel_minutes"`
	TravelMode     string `json:"travel_mode"`
	Bidirectional  bool   `json:"bidirectional"`
	ConditionsJSON string `json:"conditions_json"`
	Visibility     string `json:"visibility"`
	BranchID       string `json:"branch_id"`
	SourceCommitID string `json:"source_commit_id"`
}
type SpatialLocationRef struct {
	Name           string   `json:"name"`
	Kind           string   `json:"kind,omitempty"`
	RegionPath     []string `json:"region_path,omitempty"`
	ParentLocation string   `json:"parent_location,omitempty"`
	Description    string   `json:"description,omitempty"`
}
type SpatialRouteRef struct {
	From          string         `json:"from"`
	To            string         `json:"to"`
	Direction     string         `json:"direction,omitempty"`
	TravelMinutes int            `json:"travel_minutes,omitempty"`
	TravelMode    string         `json:"travel_mode,omitempty"`
	Bidirectional bool           `json:"bidirectional,omitempty"`
	Conditions    map[string]any `json:"conditions,omitempty"`
}
type LocationTransition struct {
	From       *SpatialLocationRef  `json:"from,omitempty"`
	To         SpatialLocationRef   `json:"to"`
	Discovered []SpatialLocationRef `json:"discovered,omitempty"`
	Routes     []SpatialRouteRef    `json:"routes,omitempty"`
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
	Regions           []CanonicalRegion   `json:"regions"`
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
			ref := SpatialLocationRef{}
			discoveredTurn := ws.CurrentTurn
			switch item := v.(type) {
			case string:
				ref.Name = item
			case map[string]any:
				ref.Name, _ = item["name"].(string)
				ref.Description, _ = item["description"].(string)
				if region, _ := item["region"].(string); strings.TrimSpace(region) != "" {
					ref.RegionPath = []string{region}
				}
				if turn, ok := item["discovered_turn"].(float64); ok {
					discoveredTurn = int(turn)
				}
			}
			if strings.TrimSpace(ref.Name) != "" {
				_, _ = ensureStructuredLocationTx(tx, ws.StoryID, ref, discoveredTurn)
			}
		}
	}
	rows, err := tx.Query(`SELECT l.canonical_name,l.description,l.discovered_turn,COALESCE(r.name,'') FROM locations l LEFT JOIN regions r ON r.id=l.region_id WHERE l.story_id=? AND l.branch_id=COALESCE(NULLIF((SELECT active_branch_id FROM stories WHERE id=l.story_id),''),l.branch_id) AND l.visibility IN ('public','player') AND l.discovery_state!='unknown' ORDER BY lower(l.canonical_name),l.id`, ws.StoryID)
	if err != nil {
		return err
	}
	projected := make([]map[string]any, 0)
	for rows.Next() {
		var name, description, region string
		var discoveredTurn int
		if err := rows.Scan(&name, &description, &discoveredTurn, &region); err != nil {
			rows.Close()
			return err
		}
		item := map[string]any{"name": name, "discovered_turn": discoveredTurn}
		if description != "" {
			item["description"] = description
		}
		if region != "" {
			item["region"] = region
		}
		projected = append(projected, item)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if payload, marshalErr := json.Marshal(projected); marshalErr == nil {
		ws.KnownLocationsJSON = string(payload)
		if _, err := tx.Exec(`UPDATE world_state SET known_locations_json=?,current_location_id=? WHERE id=?`, ws.KnownLocationsJSON, ws.CurrentLocationID, ws.ID); err != nil {
			return err
		}
	}
	return nil
}

func ensureLocationTx(tx *sql.Tx, storyID, name string, turn int) (*CanonicalLocation, error) {
	return ensureStructuredLocationTx(tx, storyID, SpatialLocationRef{Name: name}, turn)
}

func ensureStructuredLocationTx(tx *sql.Tx, storyID string, ref SpatialLocationRef, turn int) (*CanonicalLocation, error) {
	name := strings.TrimSpace(ref.Name)
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("location name required")
	}
	region, err := ensureRegionPathTx(tx, storyID, ref.RegionPath)
	if err != nil {
		return nil, err
	}
	var parent *CanonicalLocation
	if parentName := strings.TrimSpace(ref.ParentLocation); parentName != "" && !strings.EqualFold(parentName, name) {
		parent, err = ensureStructuredLocationTx(tx, storyID, SpatialLocationRef{
			Name:       parentName,
			Kind:       "place",
			RegionPath: ref.RegionPath,
		}, turn)
		if err != nil {
			return nil, err
		}
	}
	kind := normalizeLocationKind(ref.Kind)
	regionID := ""
	if region != nil {
		regionID = region.ID
	}
	parentID := ""
	if parent != nil {
		parentID = parent.ID
	}
	var l CanonicalLocation
	err = tx.QueryRow(`SELECT id,story_id,canonical_name,location_kind,COALESCE(region_id,''),COALESCE(parent_location_id,''),description,discovery_state,discovered_turn,visibility,branch_id,source_commit_id FROM locations WHERE story_id=? AND branch_id=COALESCE(NULLIF((SELECT active_branch_id FROM stories WHERE id=?),''),branch_id) AND lower(canonical_name)=lower(?) LIMIT 1`, storyID, storyID, name).Scan(&l.ID, &l.StoryID, &l.Name, &l.Kind, &l.RegionID, &l.ParentLocationID, &l.Description, &l.DiscoveryState, &l.DiscoveredTurn, &l.Visibility, &l.BranchID, &l.SourceCommitID)
	if err == nil {
		description := strings.TrimSpace(ref.Description)
		if _, err = tx.Exec(`UPDATE locations SET location_kind=CASE WHEN ?!='place' OR location_kind='' THEN ? ELSE location_kind END,region_id=CASE WHEN ?!='' THEN ? ELSE region_id END,parent_location_id=CASE WHEN ?!='' THEN ? ELSE parent_location_id END,description=CASE WHEN ?!='' THEN ? ELSE description END,discovered_turn=CASE WHEN ?<discovered_turn THEN ? ELSE discovered_turn END,discovery_state=CASE WHEN discovery_state='unknown' THEN 'discovered' ELSE discovery_state END,visibility=CASE WHEN visibility='private' THEN 'player' ELSE visibility END WHERE id=?`, kind, kind, regionID, regionID, parentID, parentID, description, description, turn, turn, l.ID); err != nil {
			return nil, err
		}
		l.Kind = kind
		if regionID != "" {
			l.RegionID = regionID
		}
		if parentID != "" {
			l.ParentLocationID = parentID
		}
		if description != "" {
			l.Description = description
		}
		if turn < l.DiscoveredTurn {
			l.DiscoveredTurn = turn
		}
		if l.DiscoveryState == "unknown" {
			l.DiscoveryState = "discovered"
		}
		return &l, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	b, c, err := activeLineageTx(tx, storyID)
	if err != nil {
		return nil, err
	}
	l = CanonicalLocation{ID: uuid.NewString(), StoryID: storyID, Name: name, Kind: kind, RegionID: regionID, ParentLocationID: parentID, Description: strings.TrimSpace(ref.Description), DiscoveryState: "discovered", DiscoveredTurn: turn, Visibility: "player", BranchID: b, SourceCommitID: c}
	_, err = tx.Exec(`INSERT INTO locations (id,story_id,canonical_name,location_kind,region_id,parent_location_id,description,discovery_state,discovered_turn,visibility,branch_id,source_commit_id) VALUES (?,?,?,?,NULLIF(?,''),NULLIF(?,''),?,?,?,?,?,?)`, l.ID, l.StoryID, l.Name, l.Kind, l.RegionID, l.ParentLocationID, l.Description, l.DiscoveryState, l.DiscoveredTurn, l.Visibility, b, c)
	if err == nil {
		_, _ = tx.Exec(`INSERT OR IGNORE INTO location_aliases (id,story_id,location_id,alias,visibility,branch_id,source_commit_id) VALUES (?,?,?,?,'player',?,?)`, uuid.NewString(), storyID, l.ID, name, b, c)
	}
	return &l, err
}

func ensureRegionPathTx(tx *sql.Tx, storyID string, path []string) (*CanonicalRegion, error) {
	var parent *CanonicalRegion
	for index, rawName := range path {
		name := strings.TrimSpace(rawName)
		if name == "" {
			continue
		}
		parentID := ""
		if parent != nil {
			parentID = parent.ID
		}
		var region CanonicalRegion
		err := tx.QueryRow(`SELECT id,story_id,name,region_kind,COALESCE(parent_region_id,''),visibility,branch_id,source_commit_id FROM regions WHERE story_id=? AND branch_id=COALESCE(NULLIF((SELECT active_branch_id FROM stories WHERE id=?),''),branch_id) AND lower(name)=lower(?) LIMIT 1`, storyID, storyID, name).Scan(&region.ID, &region.StoryID, &region.Name, &region.Kind, &region.ParentRegionID, &region.Visibility, &region.BranchID, &region.SourceCommitID)
		if errors.Is(err, sql.ErrNoRows) {
			branchID, commitID, lineageErr := activeLineageTx(tx, storyID)
			if lineageErr != nil {
				return nil, lineageErr
			}
			kind := "region"
			if index == 0 {
				kind = "macroregion"
			}
			region = CanonicalRegion{ID: uuid.NewString(), StoryID: storyID, Name: name, Kind: kind, ParentRegionID: parentID, Visibility: "player", BranchID: branchID, SourceCommitID: commitID}
			_, err = tx.Exec(`INSERT INTO regions (id,story_id,name,region_kind,parent_region_id,visibility,branch_id,source_commit_id) VALUES (?,?,?,?,NULLIF(?,''),'player',?,?)`, region.ID, region.StoryID, region.Name, region.Kind, region.ParentRegionID, branchID, commitID)
		} else if err == nil && parentID != "" && region.ParentRegionID == "" {
			_, err = tx.Exec(`UPDATE regions SET parent_region_id=?,visibility='player' WHERE id=?`, parentID, region.ID)
			region.ParentRegionID = parentID
		}
		if err != nil {
			return nil, err
		}
		parent = &region
	}
	return parent, nil
}

func normalizeLocationKind(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "world", "region", "district", "site", "building", "interior", "room", "subzone", "landmark":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "place"
	}
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
		if strings.TrimSpace(edge.TravelMode) == "" {
			edge.TravelMode = "travel"
		}
		edge.BranchID = b
		edge.SourceCommitID = c
		_, e = tx.Exec(`INSERT INTO location_edges (id,story_id,from_location_id,to_location_id,direction,travel_minutes,travel_mode,bidirectional,conditions_json,visibility,branch_id,source_commit_id) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, edge.ID, edge.StoryID, edge.FromLocationID, edge.ToLocationID, edge.Direction, edge.TravelMinutes, edge.TravelMode, edge.Bidirectional, edge.ConditionsJSON, edge.Visibility, b, c)
		return e
	})
}

func (db *DB) ApplyLocationTransitionTx(tx *sql.Tx, storyID, entityID string, transition LocationTransition, turn int) (*CanonicalLocation, error) {
	if tx == nil {
		return nil, errors.New("transaction is required")
	}
	if strings.TrimSpace(storyID) == "" || strings.TrimSpace(transition.To.Name) == "" {
		return nil, errors.New("story and transition destination are required")
	}
	var previousName string
	_ = tx.QueryRow(`SELECT current_location FROM world_state WHERE story_id=?`, storyID).Scan(&previousName)
	fromRef := transition.From
	if fromRef == nil && strings.TrimSpace(previousName) != "" {
		fromRef = &SpatialLocationRef{Name: previousName}
	}
	var from *CanonicalLocation
	var err error
	if fromRef != nil && strings.TrimSpace(fromRef.Name) != "" {
		from, err = ensureStructuredLocationTx(tx, storyID, *fromRef, turn)
		if err != nil {
			return nil, err
		}
	}
	for _, discovered := range transition.Discovered {
		if _, err := ensureStructuredLocationTx(tx, storyID, discovered, turn); err != nil {
			return nil, err
		}
	}
	destination, err := ensureStructuredLocationTx(tx, storyID, transition.To, turn)
	if err != nil {
		return nil, err
	}
	routes := append([]SpatialRouteRef(nil), transition.Routes...)
	if from != nil && from.ID != destination.ID && !hasTransitionRoute(routes, from.Name, destination.Name) {
		routes = append(routes, SpatialRouteRef{From: from.Name, To: destination.Name, TravelMode: "travel", Bidirectional: true})
	}
	travelMinutes := 0
	travelMode := "travel"
	for _, route := range routes {
		if err := upsertSpatialRouteTx(tx, storyID, route, transition, turn); err != nil {
			return nil, err
		}
		if from != nil && routeConnects(route, from.Name, destination.Name) && route.TravelMinutes > travelMinutes {
			travelMinutes = route.TravelMinutes
			if strings.TrimSpace(route.TravelMode) != "" {
				travelMode = strings.TrimSpace(route.TravelMode)
			}
		}
	}
	if from != nil && from.ID != destination.ID && travelMinutes > 0 {
		if _, err := db.AdvanceWorldTimeTx(tx, storyID, "travel:"+travelMode, travelMinutes, turn); err != nil {
			return nil, err
		}
	}
	branchID, commitID, err := activeLineageTx(tx, storyID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(entityID) == "" {
		entityID = "protagonist"
	}
	if from == nil || from.ID != destination.ID {
		_, err = tx.Exec(`INSERT INTO entity_position_events (id,story_id,entity_id,location_id,event_kind,turn,visibility,branch_id,source_commit_id) VALUES (?,?,?,?, 'arrived',?,'player',?,?)`, uuid.NewString(), storyID, entityID, destination.ID, turn, branchID, commitID)
		if err != nil {
			return nil, err
		}
	}
	return destination, nil
}

func routeConnects(route SpatialRouteRef, from, to string) bool {
	direct := strings.EqualFold(strings.TrimSpace(route.From), strings.TrimSpace(from)) && strings.EqualFold(strings.TrimSpace(route.To), strings.TrimSpace(to))
	reverse := route.Bidirectional && strings.EqualFold(strings.TrimSpace(route.From), strings.TrimSpace(to)) && strings.EqualFold(strings.TrimSpace(route.To), strings.TrimSpace(from))
	return direct || reverse
}

func hasTransitionRoute(routes []SpatialRouteRef, from, to string) bool {
	for _, route := range routes {
		if strings.EqualFold(strings.TrimSpace(route.From), strings.TrimSpace(from)) && strings.EqualFold(strings.TrimSpace(route.To), strings.TrimSpace(to)) {
			return true
		}
		if route.Bidirectional && strings.EqualFold(strings.TrimSpace(route.From), strings.TrimSpace(to)) && strings.EqualFold(strings.TrimSpace(route.To), strings.TrimSpace(from)) {
			return true
		}
	}
	return false
}

func upsertSpatialRouteTx(tx *sql.Tx, storyID string, route SpatialRouteRef, transition LocationTransition, turn int) error {
	fromRef := SpatialLocationRef{Name: route.From}
	toRef := SpatialLocationRef{Name: route.To}
	if strings.EqualFold(strings.TrimSpace(route.From), strings.TrimSpace(transition.To.Name)) {
		fromRef = transition.To
	}
	if strings.EqualFold(strings.TrimSpace(route.To), strings.TrimSpace(transition.To.Name)) {
		toRef = transition.To
	}
	if transition.From != nil && strings.EqualFold(strings.TrimSpace(route.From), strings.TrimSpace(transition.From.Name)) {
		fromRef = *transition.From
	}
	if transition.From != nil && strings.EqualFold(strings.TrimSpace(route.To), strings.TrimSpace(transition.From.Name)) {
		toRef = *transition.From
	}
	from, err := ensureStructuredLocationTx(tx, storyID, fromRef, turn)
	if err != nil {
		return err
	}
	to, err := ensureStructuredLocationTx(tx, storyID, toRef, turn)
	if err != nil {
		return err
	}
	conditions := route.Conditions
	if conditions == nil {
		conditions = map[string]any{}
	}
	conditionsJSON, err := json.Marshal(conditions)
	if err != nil {
		return err
	}
	mode := strings.TrimSpace(route.TravelMode)
	if mode == "" {
		mode = "travel"
	}
	branchID, commitID, err := activeLineageTx(tx, storyID)
	if err != nil {
		return err
	}
	insert := func(fromID, toID, direction string) error {
		_, err := tx.Exec(`INSERT INTO location_edges (id,story_id,from_location_id,to_location_id,direction,travel_minutes,travel_mode,bidirectional,conditions_json,visibility,branch_id,source_commit_id) VALUES (?,?,?,?,?,?,?,?,?,'player',?,?) ON CONFLICT DO NOTHING`, uuid.NewString(), storyID, fromID, toID, direction, max(0, route.TravelMinutes), mode, route.Bidirectional, string(conditionsJSON), branchID, commitID)
		return err
	}
	if err := insert(from.ID, to.ID, strings.TrimSpace(route.Direction)); err != nil {
		return err
	}
	return nil
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
	regionRows, err := db.conn.Query(`SELECT id,story_id,name,region_kind,COALESCE(parent_region_id,''),visibility,branch_id,source_commit_id FROM regions WHERE story_id=? AND branch_id=COALESCE(NULLIF((SELECT active_branch_id FROM stories WHERE id=?),''),branch_id) AND visibility IN ('public','player') ORDER BY lower(name),id`, storyID, storyID)
	if err != nil {
		return nil, err
	}
	for regionRows.Next() {
		var region CanonicalRegion
		if regionRows.Scan(&region.ID, &region.StoryID, &region.Name, &region.Kind, &region.ParentRegionID, &region.Visibility, &region.BranchID, &region.SourceCommitID) == nil {
			p.Regions = append(p.Regions, region)
		}
	}
	regionRows.Close()
	rows, err := db.conn.Query(`SELECT id,story_id,canonical_name,location_kind,COALESCE(region_id,''),COALESCE(parent_location_id,''),description,discovery_state,discovered_turn,visibility,branch_id,source_commit_id FROM locations WHERE story_id=? AND branch_id=COALESCE(NULLIF((SELECT active_branch_id FROM stories WHERE id=?),''),branch_id) AND visibility IN ('public','player') AND discovery_state!='unknown' ORDER BY lower(canonical_name),id`, storyID, storyID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var l CanonicalLocation
		if rows.Scan(&l.ID, &l.StoryID, &l.Name, &l.Kind, &l.RegionID, &l.ParentLocationID, &l.Description, &l.DiscoveryState, &l.DiscoveredTurn, &l.Visibility, &l.BranchID, &l.SourceCommitID) == nil {
			p.Locations = append(p.Locations, l)
		}
	}
	rows.Close()
	edgeRows, err := db.conn.Query(`SELECT id,story_id,from_location_id,to_location_id,direction,travel_minutes,travel_mode,bidirectional,conditions_json,visibility,branch_id,source_commit_id FROM location_edges WHERE story_id=? AND branch_id=COALESCE(NULLIF((SELECT active_branch_id FROM stories WHERE id=?),''),branch_id) AND visibility IN ('public','player') ORDER BY from_location_id,to_location_id,direction,travel_mode,id`, storyID, storyID)
	if err != nil {
		return nil, err
	}
	for edgeRows.Next() {
		var e LocationEdge
		if edgeRows.Scan(&e.ID, &e.StoryID, &e.FromLocationID, &e.ToLocationID, &e.Direction, &e.TravelMinutes, &e.TravelMode, &e.Bidirectional, &e.ConditionsJSON, &e.Visibility, &e.BranchID, &e.SourceCommitID) == nil {
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
