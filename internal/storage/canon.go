package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var ErrCanonicalFieldLocked = errors.New("canonical field is locked")

type CanonicalEntity struct {
	ID              string `json:"id"`
	StoryID         string `json:"story_id"`
	Kind            string `json:"kind"`
	CanonicalName   string `json:"canonical_name"`
	LifecycleStatus string `json:"lifecycle_status"`
	ProfileJSON     string `json:"profile_json"`
	BranchID        string `json:"branch_id"`
	SourceCommitID  string `json:"source_commit_id"`
}

type IdentityClaim struct {
	ID                 string  `json:"id"`
	StoryID            string  `json:"story_id"`
	SubjectEntityID    string  `json:"subject_entity_id"`
	ClaimedEntityID    string  `json:"claimed_entity_id,omitempty"`
	ObserverEntityID   string  `json:"observer_entity_id,omitempty"`
	Label              string  `json:"label"`
	Kind               string  `json:"kind"`
	Status             string  `json:"status"`
	Confidence         float64 `json:"confidence"`
	Visibility         string  `json:"visibility"`
	EvidenceJSON       string  `json:"evidence_json"`
	LearnedTurn        int     `json:"learned_turn"`
	ValidFromWorldTime string  `json:"valid_from_world_time,omitempty"`
	ValidToWorldTime   string  `json:"valid_to_world_time,omitempty"`
	SupersedesClaimID  string  `json:"supersedes_claim_id,omitempty"`
	ContradictsClaimID string  `json:"contradicts_claim_id,omitempty"`
	RetractsClaimID    string  `json:"retracts_claim_id,omitempty"`
	BranchID           string  `json:"branch_id"`
	SourceCommitID     string  `json:"source_commit_id"`
}

type CharacterFact struct {
	ID                 string  `json:"id"`
	StoryID            string  `json:"story_id"`
	SubjectEntityID    string  `json:"subject_entity_id"`
	Predicate          string  `json:"predicate"`
	ObjectJSON         string  `json:"object_json"`
	SourceEntityID     string  `json:"source_entity_id,omitempty"`
	SourceEventID      string  `json:"source_event_id,omitempty"`
	ObserverEntityID   string  `json:"observer_entity_id,omitempty"`
	LearnedTurn        int     `json:"learned_turn"`
	ValidFromWorldTime string  `json:"valid_from_world_time,omitempty"`
	ValidToWorldTime   string  `json:"valid_to_world_time,omitempty"`
	Confidence         float64 `json:"confidence"`
	Visibility         string  `json:"visibility"`
	SupersedesFactID   string  `json:"supersedes_fact_id,omitempty"`
	ContradictsFactID  string  `json:"contradicts_fact_id,omitempty"`
	RetractsFactID     string  `json:"retracts_fact_id,omitempty"`
	EvidenceJSON       string  `json:"evidence_json"`
	BranchID           string  `json:"branch_id"`
	SourceCommitID     string  `json:"source_commit_id"`
}

type EntityForm struct {
	ID             string `json:"id"`
	StoryID        string `json:"story_id"`
	EntityID       string `json:"entity_id"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	BodyEntityID   string `json:"body_entity_id,omitempty"`
	AppearanceJSON string `json:"appearance_json"`
	ValidFromTurn  int    `json:"valid_from_turn"`
	ValidToTurn    *int   `json:"valid_to_turn,omitempty"`
	BranchID       string `json:"branch_id"`
	SourceCommitID string `json:"source_commit_id"`
}
type EntityControllerEvent struct {
	ID                 string `json:"id"`
	StoryID            string `json:"story_id"`
	FormID             string `json:"form_id"`
	ControllerEntityID string `json:"controller_entity_id"`
	ControlKind        string `json:"control_kind"`
	Status             string `json:"status"`
	Turn               int    `json:"turn"`
	WorldTime          string `json:"world_time"`
	BranchID           string `json:"branch_id"`
	SourceCommitID     string `json:"source_commit_id"`
}

type PlayerSafeEntityProjection struct {
	ID             string          `json:"id"`
	Kind           string          `json:"kind"`
	DisplayName    string          `json:"display_name"`
	Aliases        []string        `json:"aliases"`
	Facts          []CharacterFact `json:"facts"`
	IdentityClaims []IdentityClaim `json:"identity_claims"`
}

type Faction struct {
	ID             string `json:"id"`
	StoryID        string `json:"story_id"`
	Name           string `json:"name"`
	ProfileJSON    string `json:"profile_json"`
	Visibility     string `json:"visibility"`
	BranchID       string `json:"branch_id"`
	SourceCommitID string `json:"source_commit_id"`
}
type ReputationEvent struct {
	ID             string `json:"id"`
	StoryID        string `json:"story_id"`
	FactionID      string `json:"faction_id"`
	EntityID       string `json:"entity_id"`
	Delta          int    `json:"delta"`
	Reason         string `json:"reason"`
	SourceEventID  string `json:"source_event_id"`
	Visibility     string `json:"visibility"`
	Turn           int    `json:"turn"`
	BranchID       string `json:"branch_id"`
	SourceCommitID string `json:"source_commit_id"`
}
type FactionMembershipEvent struct {
	ID             string `json:"id"`
	StoryID        string `json:"story_id"`
	FactionID      string `json:"faction_id"`
	EntityID       string `json:"entity_id"`
	Status         string `json:"status"`
	Role           string `json:"role"`
	Visibility     string `json:"visibility"`
	Turn           int    `json:"turn"`
	BranchID       string `json:"branch_id"`
	SourceCommitID string `json:"source_commit_id"`
}
type FactionRelationshipEvent struct {
	ID              string `json:"id"`
	StoryID         string `json:"story_id"`
	SourceFactionID string `json:"source_faction_id"`
	TargetFactionID string `json:"target_faction_id"`
	Delta           int    `json:"delta"`
	Reason          string `json:"reason"`
	Turn            int    `json:"turn"`
	BranchID        string `json:"branch_id"`
	SourceCommitID  string `json:"source_commit_id"`
}
type PlayerSafeFactionProjection struct {
	ID           string                   `json:"id"`
	Name         string                   `json:"name"`
	ProfileJSON  string                   `json:"profile_json"`
	KnownMembers []FactionMembershipEvent `json:"known_members"`
	Reputation   map[string]int           `json:"reputation"`
}

func activeLineageTx(tx *sql.Tx, storyID string) (string, string, error) {
	var branch, commit string
	err := tx.QueryRow(`SELECT s.active_branch_id,b.head_commit_id FROM stories s JOIN story_branches b ON b.id=s.active_branch_id WHERE s.id=?`, storyID).Scan(&branch, &commit)
	return branch, commit, err
}

func (db *DB) CreateCanonicalEntity(entity *CanonicalEntity) error {
	if entity == nil || entity.ID == "" || entity.StoryID == "" {
		return errors.New("canonical entity identity is required")
	}
	return db.WithTx(func(tx *sql.Tx) error { return db.createCanonicalEntityTx(tx, entity) })
}

func (db *DB) createCanonicalEntityTx(tx *sql.Tx, entity *CanonicalEntity) error {
	branch, commit, err := activeLineageTx(tx, entity.StoryID)
	if err != nil {
		return err
	}
	if entity.Kind == "" {
		entity.Kind = "character"
	}
	if entity.LifecycleStatus == "" {
		entity.LifecycleStatus = "active"
	}
	if entity.ProfileJSON == "" {
		entity.ProfileJSON = "{}"
	}
	if !json.Valid([]byte(entity.ProfileJSON)) {
		return errors.New("entity profile must be valid JSON")
	}
	entity.BranchID = branch
	entity.SourceCommitID = commit
	now := time.Now().UTC()
	_, err = tx.Exec(`INSERT INTO canonical_entities (id,story_id,entity_kind,canonical_name,lifecycle_status,profile_json,branch_id,source_commit_id,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)`, entity.ID, entity.StoryID, entity.Kind, entity.CanonicalName, entity.LifecycleStatus, entity.ProfileJSON, branch, commit, now, now)
	return err
}

func validateLineageReference(tx *sql.Tx, table, id, storyID, subjectID string) error {
	if id == "" {
		return nil
	}
	var gotStory, gotSubject string
	query := fmt.Sprintf(`SELECT story_id,subject_entity_id FROM %s WHERE id=?`, table)
	if err := tx.QueryRow(query, id).Scan(&gotStory, &gotSubject); err != nil {
		return fmt.Errorf("lineage reference %s: %w", id, err)
	}
	if gotStory != storyID || gotSubject != subjectID {
		return errors.New("lineage reference belongs to another subject or story")
	}
	return nil
}

func (db *DB) AddIdentityClaim(claim *IdentityClaim) error {
	if claim == nil {
		return errors.New("identity claim is required")
	}
	return db.WithTx(func(tx *sql.Tx) error {
		refs := []string{claim.SupersedesClaimID, claim.ContradictsClaimID, claim.RetractsClaimID}
		n := 0
		for _, r := range refs {
			if r != "" {
				n++
				if err := validateLineageReference(tx, "identity_claims", r, claim.StoryID, claim.SubjectEntityID); err != nil {
					return err
				}
			}
		}
		if n > 1 {
			return errors.New("identity claim may have only one lineage action")
		}
		branch, commit, err := activeLineageTx(tx, claim.StoryID)
		if err != nil {
			return err
		}
		claim.BranchID = branch
		claim.SourceCommitID = commit
		if claim.ID == "" {
			claim.ID = uuid.NewString()
		}
		if claim.Status == "" {
			claim.Status = "observed"
		}
		if claim.Visibility == "" {
			claim.Visibility = "private"
		}
		if claim.EvidenceJSON == "" {
			claim.EvidenceJSON = "[]"
		}
		if !json.Valid([]byte(claim.EvidenceJSON)) {
			return errors.New("identity evidence must be valid JSON")
		}
		_, err = tx.Exec(`INSERT INTO identity_claims (id,story_id,subject_entity_id,claimed_entity_id,observer_entity_id,label,claim_kind,status,confidence,visibility,evidence_json,learned_turn,valid_from_world_time,valid_to_world_time,supersedes_claim_id,contradicts_claim_id,retracts_claim_id,branch_id,source_commit_id,created_at) VALUES (?,?,?,NULLIF(?,''),NULLIF(?,''),?,?,?,?,?,?,?,?,?,NULLIF(?,''),NULLIF(?,''),NULLIF(?,''),?,?,?)`, claim.ID, claim.StoryID, claim.SubjectEntityID, claim.ClaimedEntityID, claim.ObserverEntityID, claim.Label, claim.Kind, claim.Status, claim.Confidence, claim.Visibility, claim.EvidenceJSON, claim.LearnedTurn, claim.ValidFromWorldTime, claim.ValidToWorldTime, claim.SupersedesClaimID, claim.ContradictsClaimID, claim.RetractsClaimID, branch, commit, time.Now().UTC())
		return err
	})
}

func (db *DB) AddCharacterFact(fact *CharacterFact) error {
	if fact == nil {
		return errors.New("character fact is required")
	}
	if !json.Valid([]byte(fact.ObjectJSON)) {
		return errors.New("fact object must be valid JSON")
	}
	return db.WithTx(func(tx *sql.Tx) error {
		refs := []string{fact.SupersedesFactID, fact.ContradictsFactID, fact.RetractsFactID}
		n := 0
		for _, r := range refs {
			if r != "" {
				n++
				if err := validateLineageReference(tx, "character_facts", r, fact.StoryID, fact.SubjectEntityID); err != nil {
					return err
				}
			}
		}
		if n > 1 {
			return errors.New("fact may have only one lineage action")
		}
		branch, commit, err := activeLineageTx(tx, fact.StoryID)
		if err != nil {
			return err
		}
		fact.BranchID = branch
		fact.SourceCommitID = commit
		if fact.ID == "" {
			fact.ID = uuid.NewString()
		}
		if fact.Visibility == "" {
			fact.Visibility = "private"
		}
		if fact.EvidenceJSON == "" {
			fact.EvidenceJSON = "[]"
		}
		_, err = tx.Exec(`INSERT INTO character_facts (id,story_id,subject_entity_id,predicate,object_json,source_entity_id,source_event_id,observer_entity_id,learned_turn,valid_from_world_time,valid_to_world_time,confidence,visibility,supersedes_fact_id,contradicts_fact_id,retracts_fact_id,evidence_json,branch_id,source_commit_id,created_at) VALUES (?,?,?,?,?,NULLIF(?,''),?,NULLIF(?,''),?,?,?,?,?,NULLIF(?,''),NULLIF(?,''),NULLIF(?,''),?,?,?,?)`, fact.ID, fact.StoryID, fact.SubjectEntityID, fact.Predicate, fact.ObjectJSON, fact.SourceEntityID, fact.SourceEventID, fact.ObserverEntityID, fact.LearnedTurn, fact.ValidFromWorldTime, fact.ValidToWorldTime, fact.Confidence, fact.Visibility, fact.SupersedesFactID, fact.ContradictsFactID, fact.RetractsFactID, fact.EvidenceJSON, branch, commit, time.Now().UTC())
		return err
	})
}

func (db *DB) AddEntityForm(form *EntityForm) error {
	if form == nil {
		return errors.New("entity form is required")
	}
	return db.WithTx(func(tx *sql.Tx) error {
		b, c, e := activeLineageTx(tx, form.StoryID)
		if e != nil {
			return e
		}
		if form.ID == "" {
			form.ID = uuid.NewString()
		}
		if form.Kind == "" {
			form.Kind = "base"
		}
		if form.AppearanceJSON == "" {
			form.AppearanceJSON = "{}"
		}
		if !json.Valid([]byte(form.AppearanceJSON)) {
			return errors.New("form appearance must be valid JSON")
		}
		form.BranchID = b
		form.SourceCommitID = c
		_, e = tx.Exec(`INSERT INTO entity_forms (id,story_id,entity_id,name,form_kind,body_entity_id,appearance_json,valid_from_turn,valid_to_turn,branch_id,source_commit_id,created_at) VALUES (?,?,?,?,?,NULLIF(?,''),?,?,?,?,?,?)`, form.ID, form.StoryID, form.EntityID, form.Name, form.Kind, form.BodyEntityID, form.AppearanceJSON, form.ValidFromTurn, form.ValidToTurn, b, c, time.Now().UTC())
		return e
	})
}
func (db *DB) AddEntityControllerEvent(event *EntityControllerEvent) error {
	if event == nil {
		return errors.New("controller event is required")
	}
	return db.WithTx(func(tx *sql.Tx) error {
		b, c, e := activeLineageTx(tx, event.StoryID)
		if e != nil {
			return e
		}
		if event.ID == "" {
			event.ID = uuid.NewString()
		}
		event.BranchID = b
		event.SourceCommitID = c
		_, e = tx.Exec(`INSERT INTO entity_controller_events (id,story_id,form_id,controller_entity_id,control_kind,status,turn,world_time,branch_id,source_commit_id,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`, event.ID, event.StoryID, event.FormID, event.ControllerEntityID, event.ControlKind, event.Status, event.Turn, event.WorldTime, b, c, time.Now().UTC())
		return e
	})
}

func (db *DB) LockCanonicalField(entityID, fieldPath, lockKind string, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = db.conn.Exec(`INSERT INTO entity_field_locks (entity_id,field_path,lock_kind,locked_value_json) VALUES (?,?,?,?) ON CONFLICT(entity_id,field_path,lock_kind) DO UPDATE SET locked_value_json=excluded.locked_value_json`, entityID, fieldPath, lockKind, string(payload))
	return err
}

func (db *DB) MergeCanonicalProfile(entityID string, patch map[string]any) error {
	return db.WithTx(func(tx *sql.Tx) error {
		var raw string
		if err := tx.QueryRow(`SELECT profile_json FROM canonical_entities WHERE id=?`, entityID).Scan(&raw); err != nil {
			return err
		}
		var profile map[string]any
		if json.Unmarshal([]byte(raw), &profile) != nil {
			profile = map[string]any{}
		}
		rows, err := tx.Query(`SELECT field_path,locked_value_json FROM entity_field_locks WHERE entity_id=? AND lock_kind='profile'`, entityID)
		if err != nil {
			return err
		}
		locks := map[string]string{}
		for rows.Next() {
			var path, value string
			if err := rows.Scan(&path, &value); err != nil {
				rows.Close()
				return err
			}
			locks[path] = value
		}
		rows.Close()
		for key, value := range patch {
			if locked, ok := locks[key]; ok {
				candidate, _ := json.Marshal(value)
				if string(candidate) != locked {
					return fmt.Errorf("%w: %s", ErrCanonicalFieldLocked, key)
				}
			}
			profile[key] = value
		}
		payload, err := json.Marshal(profile)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`UPDATE canonical_entities SET profile_json=?,updated_at=? WHERE id=?`, string(payload), time.Now().UTC(), entityID)
		return err
	})
}

func (db *DB) MergeCanonicalVisual(entityID string, patch map[string]any) error {
	return db.WithTx(func(tx *sql.Tx) error {
		var raw string
		if err := tx.QueryRow(`SELECT profile_json FROM canonical_entities WHERE id=?`, entityID).Scan(&raw); err != nil {
			return err
		}
		profile := map[string]any{}
		_ = json.Unmarshal([]byte(raw), &profile)
		visual, _ := profile["visual"].(map[string]any)
		if visual == nil {
			visual = map[string]any{}
		}
		rows, err := tx.Query(`SELECT field_path,locked_value_json FROM entity_field_locks WHERE entity_id=? AND lock_kind='visual'`, entityID)
		if err != nil {
			return err
		}
		locks := map[string]string{}
		for rows.Next() {
			var k, v string
			if err := rows.Scan(&k, &v); err != nil {
				rows.Close()
				return err
			}
			locks[k] = v
		}
		rows.Close()
		for k, v := range patch {
			if locked, ok := locks[k]; ok {
				candidate, _ := json.Marshal(v)
				if string(candidate) != locked {
					return fmt.Errorf("%w: visual.%s", ErrCanonicalFieldLocked, k)
				}
			}
			visual[k] = v
		}
		profile["visual"] = visual
		payload, err := json.Marshal(profile)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`UPDATE canonical_entities SET profile_json=?,updated_at=? WHERE id=?`, string(payload), time.Now().UTC(), entityID)
		return err
	})
}

func (db *DB) GetPlayerSafeEntity(storyID, entityID, observerID string, atTurn int) (*PlayerSafeEntityProjection, error) {
	var out PlayerSafeEntityProjection
	out.ID = entityID
	if err := db.conn.QueryRow(`SELECT entity_kind FROM canonical_entities WHERE id=? AND story_id=?`, entityID, storyID).Scan(&out.Kind); err != nil {
		return nil, err
	}
	rows, err := db.conn.Query(`SELECT i.id,i.story_id,i.subject_entity_id,COALESCE(i.claimed_entity_id,''),COALESCE(i.observer_entity_id,''),i.label,i.claim_kind,i.status,i.confidence,i.visibility,i.evidence_json,i.learned_turn,i.valid_from_world_time,i.valid_to_world_time,COALESCE(i.supersedes_claim_id,''),COALESCE(i.contradicts_claim_id,''),COALESCE(i.retracts_claim_id,''),i.branch_id,i.source_commit_id FROM identity_claims i WHERE i.story_id=? AND i.subject_entity_id=? AND i.learned_turn<=? AND (i.visibility IN ('public','player') OR i.observer_entity_id=?) AND i.status NOT IN ('retracted','contradicted') AND NOT EXISTS (SELECT 1 FROM identity_claims newer WHERE newer.retracts_claim_id=i.id OR newer.supersedes_claim_id=i.id) ORDER BY i.confidence DESC,i.learned_turn DESC,i.created_at DESC`, storyID, entityID, atTurn, observerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var c IdentityClaim
		if err := rows.Scan(&c.ID, &c.StoryID, &c.SubjectEntityID, &c.ClaimedEntityID, &c.ObserverEntityID, &c.Label, &c.Kind, &c.Status, &c.Confidence, &c.Visibility, &c.EvidenceJSON, &c.LearnedTurn, &c.ValidFromWorldTime, &c.ValidToWorldTime, &c.SupersedesClaimID, &c.ContradictsClaimID, &c.RetractsClaimID, &c.BranchID, &c.SourceCommitID); err != nil {
			return nil, err
		}
		out.IdentityClaims = append(out.IdentityClaims, c)
		if out.DisplayName == "" {
			out.DisplayName = c.Label
		}
	}
	if out.DisplayName == "" {
		out.DisplayName = "Unknown figure"
	}
	aliasRows, err := db.conn.Query(`SELECT alias FROM entity_aliases WHERE story_id=? AND entity_id=? AND valid_from_turn<=? AND (valid_to_turn IS NULL OR valid_to_turn>=?) AND visibility IN ('public','player') ORDER BY created_at`, storyID, entityID, atTurn, atTurn)
	if err != nil {
		return nil, err
	}
	for aliasRows.Next() {
		var a string
		if aliasRows.Scan(&a) == nil {
			out.Aliases = append(out.Aliases, a)
		}
	}
	aliasRows.Close()
	factRows, err := db.conn.Query(`SELECT f.id,f.story_id,f.subject_entity_id,f.predicate,f.object_json,COALESCE(f.source_entity_id,''),f.source_event_id,COALESCE(f.observer_entity_id,''),f.learned_turn,f.valid_from_world_time,f.valid_to_world_time,f.confidence,f.visibility,COALESCE(f.supersedes_fact_id,''),COALESCE(f.contradicts_fact_id,''),COALESCE(f.retracts_fact_id,''),f.evidence_json,f.branch_id,f.source_commit_id FROM character_facts f JOIN stories s ON s.id=f.story_id AND s.active_branch_id=f.branch_id WHERE f.story_id=? AND f.subject_entity_id=? AND f.learned_turn<=? AND (f.visibility IN ('public','player') OR f.observer_entity_id=?) AND f.retracts_fact_id IS NULL AND NOT EXISTS (SELECT 1 FROM character_facts newer WHERE newer.story_id=f.story_id AND newer.branch_id=f.branch_id AND (newer.retracts_fact_id=f.id OR newer.supersedes_fact_id=f.id)) ORDER BY f.learned_turn,f.created_at`, storyID, entityID, atTurn, observerID)
	if err != nil {
		return nil, err
	}
	defer factRows.Close()
	for factRows.Next() {
		var f CharacterFact
		if err := factRows.Scan(&f.ID, &f.StoryID, &f.SubjectEntityID, &f.Predicate, &f.ObjectJSON, &f.SourceEntityID, &f.SourceEventID, &f.ObserverEntityID, &f.LearnedTurn, &f.ValidFromWorldTime, &f.ValidToWorldTime, &f.Confidence, &f.Visibility, &f.SupersedesFactID, &f.ContradictsFactID, &f.RetractsFactID, &f.EvidenceJSON, &f.BranchID, &f.SourceCommitID); err != nil {
			return nil, err
		}
		out.Facts = append(out.Facts, f)
	}
	return &out, nil
}

func (db *DB) CreateFaction(f *Faction) error {
	if f == nil {
		return errors.New("faction required")
	}
	return db.WithTx(func(tx *sql.Tx) error {
		b, c, e := activeLineageTx(tx, f.StoryID)
		if e != nil {
			return e
		}
		if f.ID == "" {
			f.ID = uuid.NewString()
		}
		if f.ProfileJSON == "" {
			f.ProfileJSON = "{}"
		}
		if f.Visibility == "" {
			f.Visibility = "private"
		}
		f.BranchID = b
		f.SourceCommitID = c
		_, e = tx.Exec(`INSERT INTO factions (id,story_id,name,profile_json,visibility,branch_id,source_commit_id) VALUES (?,?,?,?,?,?,?)`, f.ID, f.StoryID, f.Name, f.ProfileJSON, f.Visibility, b, c)
		return e
	})
}
func (db *DB) AddReputationEvent(e *ReputationEvent) error {
	if e == nil || e.Delta < -100 || e.Delta > 100 {
		return errors.New("valid reputation event required")
	}
	return db.WithTx(func(tx *sql.Tx) error {
		b, c, err := activeLineageTx(tx, e.StoryID)
		if err != nil {
			return err
		}
		if e.ID == "" {
			e.ID = uuid.NewString()
		}
		if e.Visibility == "" {
			e.Visibility = "player"
		}
		e.BranchID = b
		e.SourceCommitID = c
		_, err = tx.Exec(`INSERT INTO reputation_events (id,story_id,faction_id,entity_id,delta,reason,source_event_id,visibility,turn,branch_id,source_commit_id) VALUES (?,?,?,?,?,?,?,?,?,?,?)`, e.ID, e.StoryID, e.FactionID, e.EntityID, e.Delta, e.Reason, e.SourceEventID, e.Visibility, e.Turn, b, c)
		return err
	})
}
func (db *DB) AddFactionMembershipEvent(e *FactionMembershipEvent) error {
	if e == nil {
		return errors.New("membership event required")
	}
	return db.WithTx(func(tx *sql.Tx) error {
		b, c, err := activeLineageTx(tx, e.StoryID)
		if err != nil {
			return err
		}
		if e.ID == "" {
			e.ID = uuid.NewString()
		}
		if e.Visibility == "" {
			e.Visibility = "private"
		}
		e.BranchID = b
		e.SourceCommitID = c
		_, err = tx.Exec(`INSERT INTO faction_membership_events (id,story_id,faction_id,entity_id,status,role,visibility,turn,branch_id,source_commit_id) VALUES (?,?,?,?,?,?,?,?,?,?)`, e.ID, e.StoryID, e.FactionID, e.EntityID, e.Status, e.Role, e.Visibility, e.Turn, b, c)
		return err
	})
}
func (db *DB) ReputationScore(storyID, factionID, entityID string) (int, error) {
	var score int
	err := db.conn.QueryRow(`SELECT MAX(-100,MIN(100,COALESCE(SUM(delta),0))) FROM reputation_events WHERE story_id=? AND faction_id=? AND entity_id=?`, storyID, factionID, entityID).Scan(&score)
	return score, err
}

func (db *DB) ListPlayerSafeFactions(storyID string) ([]PlayerSafeFactionProjection, error) {
	rows, err := db.conn.Query(`SELECT id,name,profile_json FROM factions WHERE story_id=? AND visibility IN ('public','player') ORDER BY name`, storyID)
	if err != nil {
		return nil, err
	}
	var out []PlayerSafeFactionProjection
	for rows.Next() {
		var p PlayerSafeFactionProjection
		if err := rows.Scan(&p.ID, &p.Name, &p.ProfileJSON); err != nil {
			rows.Close()
			return nil, err
		}
		p.Reputation = map[string]int{}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for i := range out {
		p := &out[i]
		members, err := db.conn.Query(`SELECT id,story_id,faction_id,entity_id,status,role,visibility,turn,branch_id,source_commit_id FROM faction_membership_events WHERE faction_id=? AND visibility IN ('public','player') ORDER BY turn,created_at`, p.ID)
		if err != nil {
			return nil, err
		}
		for members.Next() {
			var m FactionMembershipEvent
			if err := members.Scan(&m.ID, &m.StoryID, &m.FactionID, &m.EntityID, &m.Status, &m.Role, &m.Visibility, &m.Turn, &m.BranchID, &m.SourceCommitID); err != nil {
				members.Close()
				return nil, err
			}
			p.KnownMembers = append(p.KnownMembers, m)
		}
		members.Close()
		repRows, err := db.conn.Query(`SELECT entity_id,MAX(-100,MIN(100,SUM(delta))) FROM reputation_events WHERE faction_id=? AND visibility IN ('public','player') GROUP BY entity_id`, p.ID)
		if err != nil {
			return nil, err
		}
		for repRows.Next() {
			var entity string
			var score int
			if repRows.Scan(&entity, &score) == nil {
				p.Reputation[entity] = score
			}
		}
		repRows.Close()
	}
	return out, nil
}

func (db *DB) syncNPCCompatibilityTx(tx *sql.Tx, npc *NPC) error {
	if npc.CanonicalEntityID == "" {
		npc.CanonicalEntityID = npc.ID
	}
	branch, commit, err := activeLineageTx(tx, npc.StoryID)
	if err != nil {
		return err
	}
	status := "active"
	if !npc.IsAlive {
		status = "dead"
	}
	now := time.Now().UTC()
	if _, err = tx.Exec(`INSERT OR IGNORE INTO canonical_entities (id,story_id,entity_kind,canonical_name,lifecycle_status,profile_json,branch_id,source_commit_id,created_at,updated_at) VALUES (?,?,'character',?,?,?,?,?,?,?)`, npc.CanonicalEntityID, npc.StoryID, npc.Name, status, `{"compatibility_projection":"npcs"}`, branch, commit, npc.CreatedAt, now); err != nil {
		return err
	}
	var oldStatus string
	if err = tx.QueryRow(`SELECT lifecycle_status FROM canonical_entities WHERE id=?`, npc.CanonicalEntityID).Scan(&oldStatus); err != nil {
		return err
	}
	if oldStatus != status {
		if _, err = tx.Exec(`UPDATE canonical_entities SET lifecycle_status=?,updated_at=? WHERE id=?`, status, now, npc.CanonicalEntityID); err != nil {
			return err
		}
		if _, err = tx.Exec(`INSERT INTO entity_lifecycle_events (id,story_id,entity_id,status,turn,reason,branch_id,source_commit_id,created_at) VALUES (?,?,?,?,?,'npc compatibility update',?,?,?)`, uuid.NewString(), npc.StoryID, npc.CanonicalEntityID, status, npc.LastSeenTurn, branch, commit, now); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(`INSERT OR IGNORE INTO entity_aliases (id,story_id,entity_id,alias,alias_kind,visibility,valid_from_turn,branch_id,source_commit_id,created_at) VALUES (?,?,?,?,'display','player',?,?,?,?)`, uuid.NewString(), npc.StoryID, npc.CanonicalEntityID, npc.Name, npc.FirstAppearedTurn, branch, commit, now); err != nil {
		return err
	}
	if _, err = tx.Exec(`INSERT INTO identity_claims (id,story_id,subject_entity_id,claimed_entity_id,label,claim_kind,status,confidence,visibility,learned_turn,branch_id,source_commit_id,created_at) SELECT ?,?,?,?,?,'display','confirmed',1.0,'player',?,?,?,? WHERE NOT EXISTS (SELECT 1 FROM identity_claims WHERE subject_entity_id=? AND label=? AND status NOT IN ('retracted','contradicted'))`, uuid.NewString(), npc.StoryID, npc.CanonicalEntityID, npc.CanonicalEntityID, npc.Name, npc.FirstAppearedTurn, branch, commit, now, npc.CanonicalEntityID, npc.Name); err != nil {
		return err
	}
	appearance, _ := json.Marshal(map[string]string{"description": npc.Appearance})
	if _, err = tx.Exec(`INSERT OR IGNORE INTO entity_forms (id,story_id,entity_id,name,form_kind,appearance_json,valid_from_turn,branch_id,source_commit_id,created_at) VALUES (?,?,?,?,'base',?,?,?,?,?)`, "form-"+npc.CanonicalEntityID, npc.StoryID, npc.CanonicalEntityID, npc.Name+" — base form", string(appearance), npc.FirstAppearedTurn, branch, commit, now); err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE npcs SET canonical_entity_id=? WHERE id=?`, npc.CanonicalEntityID, npc.ID)
	return err
}

func (db *DB) RebuildCompatibilityCanonTx(tx *sql.Tx, storyID string, npcs []NPC, char *Character) error {
	for _, table := range []string{"reputation_events", "faction_relationship_events", "faction_membership_events", "entity_controller_events", "entity_lifecycle_events", "character_facts", "identity_claims", "entity_aliases", "entity_field_locks", "entity_forms", "factions", "canonical_entities"} {
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
	branch, commit, err := activeLineageTx(tx, storyID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if char != nil {
		if _, err := tx.Exec(`INSERT INTO canonical_entities (id,story_id,entity_kind,canonical_name,lifecycle_status,profile_json,branch_id,source_commit_id,created_at,updated_at) VALUES (?,?,'protagonist',?,'active',?, ?,?,?,?)`, char.ID, storyID, char.Name, `{"compatibility_projection":"characters"}`, branch, commit, char.CreatedAt, now); err != nil {
			return err
		}
	}
	for i := range npcs {
		if err := db.syncNPCCompatibilityTx(tx, &npcs[i]); err != nil {
			return err
		}
	}
	return nil
}
