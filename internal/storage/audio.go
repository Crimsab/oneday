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

type StoryTTSSettings struct {
	StoryID         string          `json:"story_id"`
	Mode            string          `json:"mode"`
	Autoplay        bool            `json:"autoplay"`
	DefaultLanguage string          `json:"default_language_tag"`
	ProviderPolicy  json.RawMessage `json:"provider_policy"`
	UpdatedAt       string          `json:"updated_at,omitempty"`
}

type VoiceProfile struct {
	ID              string          `json:"id"`
	Provider        string          `json:"provider"`
	Model           string          `json:"model"`
	ProviderVoiceID string          `json:"provider_voice_id"`
	DisplayName     string          `json:"display_name"`
	LanguageTags    json.RawMessage `json:"language_tags"`
	Traits          json.RawMessage `json:"traits"`
	Rights          json.RawMessage `json:"rights"`
	Version         string          `json:"version"`
	StyleFamily     string          `json:"style_family"`
	Enabled         bool            `json:"enabled"`
	CreatedAt       string          `json:"created_at,omitempty"`
	UpdatedAt       string          `json:"updated_at,omitempty"`
}

type VoiceAssignment struct {
	ID             string          `json:"id"`
	AssignmentKey  string          `json:"assignment_key"`
	StoryID        string          `json:"story_id"`
	EntityID       string          `json:"entity_id,omitempty"`
	IdentityID     string          `json:"identity_id,omitempty"`
	FormID         string          `json:"form_id,omitempty"`
	Role           string          `json:"role"`
	VoiceProfileID string          `json:"voice_profile_id"`
	EnabledMode    string          `json:"enabled_mode"`
	LanguageTag    string          `json:"language_tag"`
	Style          json.RawMessage `json:"style"`
	Locked         bool            `json:"locked"`
	Importance     string          `json:"importance"`
	AllowDuplicate bool            `json:"allow_duplicate"`
	CreatedAt      string          `json:"created_at,omitempty"`
	UpdatedAt      string          `json:"updated_at,omitempty"`
}

type PronunciationEntry struct {
	ID            string `json:"id"`
	StoryID       string `json:"story_id"`
	LanguageTag   string `json:"language_tag"`
	SourceText    string `json:"source_text"`
	Pronunciation string `json:"pronunciation"`
	Alphabet      string `json:"alphabet"`
	CaseSensitive bool   `json:"case_sensitive"`
	Revision      int    `json:"revision"`
	CreatedAt     string `json:"created_at,omitempty"`
	UpdatedAt     string `json:"updated_at,omitempty"`
}

type TTSCacheEntry struct {
	CacheKey        string          `json:"cache_key"`
	Provider        string          `json:"provider"`
	Model           string          `json:"model"`
	ProviderVoiceID string          `json:"provider_voice_id"`
	VoiceVersion    string          `json:"voice_version"`
	LanguageTag     string          `json:"language_tag"`
	TextHash        string          `json:"text_hash"`
	StyleHash       string          `json:"style_hash"`
	Style           json.RawMessage `json:"style"`
	Speed           float64         `json:"speed"`
	OutputFormat    string          `json:"output_format"`
	Status          string          `json:"status"`
	FilePath        string          `json:"file_path,omitempty"`
	DurationMS      int64           `json:"duration_ms"`
	Timings         json.RawMessage `json:"timings"`
	Error           string          `json:"error,omitempty"`
	CreatedAt       string          `json:"created_at,omitempty"`
	UpdatedAt       string          `json:"updated_at,omitempty"`
}

type AudioAsset struct {
	ID                    string          `json:"id"`
	StoryID               string          `json:"story_id"`
	BranchID              string          `json:"branch_id"`
	SourceCommitID        string          `json:"source_commit_id"`
	SourceMessageID       int64           `json:"source_message_id"`
	SegmentIndex          int             `json:"segment_index"`
	SegmentKind           string          `json:"segment_kind"`
	SpeakerEntityID       string          `json:"speaker_entity_id,omitempty"`
	IdentityID            string          `json:"identity_id,omitempty"`
	FormID                string          `json:"form_id,omitempty"`
	VoiceProfileID        string          `json:"voice_profile_id"`
	Provider              string          `json:"provider"`
	Model                 string          `json:"model"`
	ProviderVoiceID       string          `json:"provider_voice_id"`
	VoiceVersion          string          `json:"voice_version"`
	LanguageTag           string          `json:"language_tag"`
	PronunciationRevision int             `json:"pronunciation_revision"`
	Text                  string          `json:"text"`
	TextHash              string          `json:"text_hash"`
	CacheKey              string          `json:"cache_key"`
	Style                 json.RawMessage `json:"style"`
	Speed                 float64         `json:"speed"`
	OutputFormat          string          `json:"output_format"`
	Status                string          `json:"status"`
	URL                   string          `json:"url,omitempty"`
	FilePath              string          `json:"file_path,omitempty"`
	DurationMS            int64           `json:"duration_ms"`
	Timings               json.RawMessage `json:"timings"`
	GenerationRunID       string          `json:"generation_run_id,omitempty"`
	Error                 string          `json:"error,omitempty"`
	CreatedAt             string          `json:"created_at,omitempty"`
	UpdatedAt             string          `json:"updated_at,omitempty"`
}

type TTSJob struct {
	ID              string `json:"id"`
	AudioAssetID    string `json:"audio_asset_id"`
	StoryID         string `json:"story_id"`
	BranchID        string `json:"branch_id"`
	SourceCommitID  string `json:"source_commit_id"`
	Status          string `json:"status"`
	Provider        string `json:"provider"`
	Attempts        int    `json:"attempts"`
	MaxAttempts     int    `json:"max_attempts"`
	NextAttemptAt   string `json:"next_attempt_at,omitempty"`
	TraceID         string `json:"trace_id,omitempty"`
	ParentRunID     string `json:"parent_run_id,omitempty"`
	GenerationRunID string `json:"generation_run_id,omitempty"`
	ErrorClass      string `json:"error_class,omitempty"`
	Error           string `json:"error,omitempty"`
	CreatedAt       string `json:"created_at,omitempty"`
	UpdatedAt       string `json:"updated_at,omitempty"`
}

func (db *DB) GetStoryTTSSettings(storyID string) (*StoryTTSSettings, error) {
	settings := StoryTTSSettings{StoryID: storyID, Mode: "off", ProviderPolicy: json.RawMessage(`{}`)}
	var autoplay int
	err := db.conn.QueryRow(`SELECT story_id,mode,autoplay,default_language_tag,provider_policy_json,updated_at FROM story_tts_settings WHERE story_id=?`, storyID).
		Scan(&settings.StoryID, &settings.Mode, &autoplay, &settings.DefaultLanguage, &settings.ProviderPolicy, &settings.UpdatedAt)
	if err == sql.ErrNoRows {
		var exists int
		if checkErr := db.conn.QueryRow(`SELECT COUNT(*) FROM stories WHERE id=?`, storyID).Scan(&exists); checkErr != nil {
			return nil, checkErr
		}
		if exists == 0 {
			return nil, sql.ErrNoRows
		}
		return &settings, nil
	}
	if err != nil {
		return nil, err
	}
	settings.Autoplay = autoplay == 1
	return &settings, nil
}

func (db *DB) UpsertStoryTTSSettings(settings StoryTTSSettings) (*StoryTTSSettings, error) {
	if settings.Mode != "off" && settings.Mode != "narrator" && settings.Mode != "dialogue" && settings.Mode != "all" {
		return nil, fmt.Errorf("invalid TTS mode %q", settings.Mode)
	}
	settings.DefaultLanguage = normalizeLanguageTag(settings.DefaultLanguage)
	if settings.ProviderPolicy == nil {
		settings.ProviderPolicy = json.RawMessage(`{}`)
	}
	if !json.Valid(settings.ProviderPolicy) || strings.TrimSpace(settings.StoryID) == "" {
		return nil, errors.New("story id and valid provider policy JSON are required")
	}
	_, err := db.conn.Exec(`INSERT INTO story_tts_settings(story_id,mode,autoplay,default_language_tag,provider_policy_json)
		VALUES(?,?,?,?,?) ON CONFLICT(story_id) DO UPDATE SET mode=excluded.mode,autoplay=excluded.autoplay,default_language_tag=excluded.default_language_tag,provider_policy_json=excluded.provider_policy_json,updated_at=CURRENT_TIMESTAMP`,
		settings.StoryID, settings.Mode, boolInt(settings.Autoplay), settings.DefaultLanguage, settings.ProviderPolicy)
	if err != nil {
		return nil, fmt.Errorf("saving TTS settings: %w", err)
	}
	return db.GetStoryTTSSettings(settings.StoryID)
}

func (db *DB) UpsertVoiceProfile(voice VoiceProfile) (*VoiceProfile, error) {
	if strings.TrimSpace(voice.ID) == "" {
		voice.ID = uuid.NewString()
	}
	if strings.TrimSpace(voice.Provider) == "" || strings.TrimSpace(voice.Model) == "" || strings.TrimSpace(voice.ProviderVoiceID) == "" || strings.TrimSpace(voice.DisplayName) == "" {
		return nil, errors.New("voice provider, model, provider voice id, and display name are required")
	}
	if strings.TrimSpace(voice.StyleFamily) == "" {
		voice.StyleFamily = "neutral"
	}
	voice.LanguageTags = validJSONOr(voice.LanguageTags, `[]`)
	voice.Traits = validJSONOr(voice.Traits, `{}`)
	voice.Rights = validJSONOr(voice.Rights, `{}`)
	if voice.LanguageTags == nil || voice.Traits == nil || voice.Rights == nil {
		return nil, errors.New("voice language, traits, and rights must be valid JSON")
	}
	_, err := db.conn.Exec(`INSERT INTO voice_profiles(id,provider,model,provider_voice_id,display_name,language_tags_json,traits_json,rights_json,version,style_family,enabled)
		VALUES(?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET provider=excluded.provider,model=excluded.model,provider_voice_id=excluded.provider_voice_id,display_name=excluded.display_name,language_tags_json=excluded.language_tags_json,traits_json=excluded.traits_json,rights_json=excluded.rights_json,version=excluded.version,style_family=excluded.style_family,enabled=excluded.enabled,updated_at=CURRENT_TIMESTAMP`,
		voice.ID, voice.Provider, voice.Model, voice.ProviderVoiceID, voice.DisplayName, voice.LanguageTags, voice.Traits, voice.Rights, voice.Version, voice.StyleFamily, boolInt(voice.Enabled))
	if err != nil {
		return nil, fmt.Errorf("saving voice profile: %w", err)
	}
	return db.GetVoiceProfile(voice.ID)
}

func (db *DB) GetVoiceProfile(id string) (*VoiceProfile, error) {
	var voice VoiceProfile
	var enabled int
	err := db.conn.QueryRow(`SELECT id,provider,model,provider_voice_id,display_name,language_tags_json,traits_json,rights_json,version,style_family,enabled,created_at,updated_at FROM voice_profiles WHERE id=?`, id).
		Scan(&voice.ID, &voice.Provider, &voice.Model, &voice.ProviderVoiceID, &voice.DisplayName, &voice.LanguageTags, &voice.Traits, &voice.Rights, &voice.Version, &voice.StyleFamily, &enabled, &voice.CreatedAt, &voice.UpdatedAt)
	voice.Enabled = enabled == 1
	return &voice, err
}

func (db *DB) ListVoiceProfiles(enabledOnly bool) ([]VoiceProfile, error) {
	query := `SELECT id,provider,model,provider_voice_id,display_name,language_tags_json,traits_json,rights_json,version,style_family,enabled,created_at,updated_at FROM voice_profiles`
	if enabledOnly {
		query += ` WHERE enabled=1`
	}
	query += ` ORDER BY display_name,id`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	voices := []VoiceProfile{}
	for rows.Next() {
		var voice VoiceProfile
		var enabled int
		if err := rows.Scan(&voice.ID, &voice.Provider, &voice.Model, &voice.ProviderVoiceID, &voice.DisplayName, &voice.LanguageTags, &voice.Traits, &voice.Rights, &voice.Version, &voice.StyleFamily, &enabled, &voice.CreatedAt, &voice.UpdatedAt); err != nil {
			return nil, err
		}
		voice.Enabled = enabled == 1
		voices = append(voices, voice)
	}
	return voices, rows.Err()
}

func VoiceAssignmentKey(assignment VoiceAssignment) string {
	parts := []string{strings.ToLower(strings.TrimSpace(assignment.Role))}
	for _, value := range []string{assignment.EntityID, assignment.IdentityID, assignment.FormID} {
		parts = append(parts, strings.TrimSpace(value))
	}
	return strings.Join(parts, ":")
}

func (db *DB) UpsertVoiceAssignment(assignment VoiceAssignment) (*VoiceAssignment, error) {
	if assignment.ID == "" {
		assignment.ID = uuid.NewString()
	}
	if assignment.AssignmentKey == "" {
		assignment.AssignmentKey = VoiceAssignmentKey(assignment)
	}
	if assignment.Role != "narrator" && assignment.Role != "protagonist" && assignment.Role != "npc" {
		return nil, fmt.Errorf("invalid voice role %q", assignment.Role)
	}
	if assignment.Role == "npc" && strings.TrimSpace(assignment.EntityID) == "" {
		return nil, errors.New("NPC voice assignment requires an entity id")
	}
	if assignment.EnabledMode == "" {
		assignment.EnabledMode = "inherit"
	}
	if assignment.EnabledMode != "inherit" && assignment.EnabledMode != "on" && assignment.EnabledMode != "off" {
		return nil, fmt.Errorf("invalid assignment mode %q", assignment.EnabledMode)
	}
	if assignment.Importance == "" {
		assignment.Importance = "supporting"
	}
	if assignment.Importance != "major" && assignment.Importance != "supporting" && assignment.Importance != "minor" {
		return nil, fmt.Errorf("invalid voice importance %q", assignment.Importance)
	}
	assignment.LanguageTag = normalizeLanguageTag(assignment.LanguageTag)
	assignment.Style = validJSONOr(assignment.Style, `{}`)
	if assignment.Style == nil || strings.TrimSpace(assignment.StoryID) == "" || strings.TrimSpace(assignment.VoiceProfileID) == "" {
		return nil, errors.New("story, voice profile, and valid style JSON are required")
	}
	var oldVoice string
	var oldLocked int
	err := db.conn.QueryRow(`SELECT voice_profile_id,locked FROM character_voice_assignments WHERE id=?`, assignment.ID).Scan(&oldVoice, &oldLocked)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if oldLocked == 1 && oldVoice != assignment.VoiceProfileID {
		return nil, errors.New("locked voice assignment cannot be reassigned")
	}
	_, err = db.conn.Exec(`INSERT INTO character_voice_assignments(id,assignment_key,story_id,entity_id,identity_id,form_id,role,voice_profile_id,enabled_mode,language_tag,style_json,locked,importance,allow_duplicate)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET assignment_key=excluded.assignment_key,entity_id=excluded.entity_id,identity_id=excluded.identity_id,form_id=excluded.form_id,role=excluded.role,voice_profile_id=excluded.voice_profile_id,enabled_mode=excluded.enabled_mode,language_tag=excluded.language_tag,style_json=excluded.style_json,locked=excluded.locked,importance=excluded.importance,allow_duplicate=excluded.allow_duplicate,updated_at=CURRENT_TIMESTAMP`,
		assignment.ID, assignment.AssignmentKey, assignment.StoryID, nullableString(assignment.EntityID), nullableString(assignment.IdentityID), nullableString(assignment.FormID), assignment.Role, assignment.VoiceProfileID, assignment.EnabledMode, assignment.LanguageTag, assignment.Style, boolInt(assignment.Locked), assignment.Importance, boolInt(assignment.AllowDuplicate))
	if err != nil {
		if strings.Contains(err.Error(), "idx_major_voice_unique") || strings.Contains(err.Error(), "character_voice_assignments.story_id, character_voice_assignments.voice_profile_id") {
			return nil, errors.New("major characters must use distinct voice profiles unless duplicate override is explicit")
		}
		return nil, fmt.Errorf("saving voice assignment: %w", err)
	}
	return db.GetVoiceAssignment(assignment.StoryID, assignment.ID)
}

func (db *DB) GetVoiceAssignment(storyID, id string) (*VoiceAssignment, error) {
	row := db.conn.QueryRow(`SELECT id,assignment_key,story_id,COALESCE(entity_id,''),COALESCE(identity_id,''),COALESCE(form_id,''),role,voice_profile_id,enabled_mode,language_tag,style_json,locked,importance,allow_duplicate,created_at,updated_at FROM character_voice_assignments WHERE story_id=? AND id=?`, storyID, id)
	return scanVoiceAssignment(row)
}

func (db *DB) ListVoiceAssignments(storyID string) ([]VoiceAssignment, error) {
	rows, err := db.conn.Query(`SELECT id,assignment_key,story_id,COALESCE(entity_id,''),COALESCE(identity_id,''),COALESCE(form_id,''),role,voice_profile_id,enabled_mode,language_tag,style_json,locked,importance,allow_duplicate,created_at,updated_at FROM character_voice_assignments WHERE story_id=? ORDER BY role,assignment_key`, storyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	assignments := []VoiceAssignment{}
	for rows.Next() {
		assignment, err := scanVoiceAssignment(rows)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, *assignment)
	}
	return assignments, rows.Err()
}

type rowScanner interface{ Scan(...any) error }

func scanVoiceAssignment(row rowScanner) (*VoiceAssignment, error) {
	var assignment VoiceAssignment
	var locked, duplicate int
	err := row.Scan(&assignment.ID, &assignment.AssignmentKey, &assignment.StoryID, &assignment.EntityID, &assignment.IdentityID, &assignment.FormID, &assignment.Role, &assignment.VoiceProfileID, &assignment.EnabledMode, &assignment.LanguageTag, &assignment.Style, &locked, &assignment.Importance, &duplicate, &assignment.CreatedAt, &assignment.UpdatedAt)
	assignment.Locked, assignment.AllowDuplicate = locked == 1, duplicate == 1
	return &assignment, err
}

func (db *DB) UpsertPronunciation(entry PronunciationEntry) (*PronunciationEntry, error) {
	if entry.ID == "" {
		entry.ID = uuid.NewString()
	}
	entry.LanguageTag = normalizeLanguageTag(entry.LanguageTag)
	if entry.Alphabet == "" {
		entry.Alphabet = "ipa"
	}
	if entry.Alphabet != "ipa" && entry.Alphabet != "x-sampa" && entry.Alphabet != "provider" {
		return nil, fmt.Errorf("invalid pronunciation alphabet %q", entry.Alphabet)
	}
	if entry.Revision <= 0 {
		entry.Revision = 1
	}
	if strings.TrimSpace(entry.StoryID) == "" || entry.LanguageTag == "" || strings.TrimSpace(entry.SourceText) == "" || strings.TrimSpace(entry.Pronunciation) == "" {
		return nil, errors.New("story, language, source text, and pronunciation are required")
	}
	_, err := db.conn.Exec(`INSERT INTO pronunciation_lexicon(id,story_id,language_tag,source_text,pronunciation,alphabet,case_sensitive,revision)
		VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(story_id,language_tag,source_text,case_sensitive) DO UPDATE SET pronunciation=excluded.pronunciation,alphabet=excluded.alphabet,revision=pronunciation_lexicon.revision+1,updated_at=CURRENT_TIMESTAMP`,
		entry.ID, entry.StoryID, entry.LanguageTag, entry.SourceText, entry.Pronunciation, entry.Alphabet, boolInt(entry.CaseSensitive), entry.Revision)
	if err != nil {
		return nil, fmt.Errorf("saving pronunciation: %w", err)
	}
	return db.GetPronunciation(entry.StoryID, entry.LanguageTag, entry.SourceText, entry.CaseSensitive)
}

func (db *DB) GetPronunciation(storyID, languageTag, sourceText string, caseSensitive bool) (*PronunciationEntry, error) {
	var entry PronunciationEntry
	var sensitive int
	err := db.conn.QueryRow(`SELECT id,story_id,language_tag,source_text,pronunciation,alphabet,case_sensitive,revision,created_at,updated_at FROM pronunciation_lexicon WHERE story_id=? AND language_tag=? AND source_text=? AND case_sensitive=?`, storyID, normalizeLanguageTag(languageTag), sourceText, boolInt(caseSensitive)).
		Scan(&entry.ID, &entry.StoryID, &entry.LanguageTag, &entry.SourceText, &entry.Pronunciation, &entry.Alphabet, &sensitive, &entry.Revision, &entry.CreatedAt, &entry.UpdatedAt)
	entry.CaseSensitive = sensitive == 1
	return &entry, err
}

func (db *DB) ListPronunciations(storyID, languageTag string) ([]PronunciationEntry, error) {
	query := `SELECT id,story_id,language_tag,source_text,pronunciation,alphabet,case_sensitive,revision,created_at,updated_at FROM pronunciation_lexicon WHERE story_id=?`
	args := []any{storyID}
	if normalized := normalizeLanguageTag(languageTag); normalized != "" {
		query += ` AND language_tag=?`
		args = append(args, normalized)
	}
	query += ` ORDER BY length(source_text) DESC,source_text`
	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []PronunciationEntry{}
	for rows.Next() {
		var entry PronunciationEntry
		var sensitive int
		if err := rows.Scan(&entry.ID, &entry.StoryID, &entry.LanguageTag, &entry.SourceText, &entry.Pronunciation, &entry.Alphabet, &sensitive, &entry.Revision, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
			return nil, err
		}
		entry.CaseSensitive = sensitive == 1
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (db *DB) DeletePronunciation(storyID, id string) error {
	result, err := db.conn.Exec(`DELETE FROM pronunciation_lexicon WHERE story_id=? AND id=?`, storyID, id)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func (db *DB) GetActiveEntityFormID(storyID, entityID string) string {
	var id string
	_ = db.conn.QueryRow(`SELECT f.id FROM entity_forms f JOIN stories s ON s.id=f.story_id WHERE f.story_id=? AND f.entity_id=? AND f.branch_id=s.active_branch_id AND f.valid_to_turn IS NULL ORDER BY f.valid_from_turn DESC,f.created_at DESC LIMIT 1`, storyID, entityID).Scan(&id)
	return id
}

func (db *DB) GetTTSCacheEntry(cacheKey string) (*TTSCacheEntry, error) {
	entry, err := scanTTSCacheEntry(db.conn.QueryRow(`SELECT cache_key,provider,model,provider_voice_id,voice_version,language_tag,text_hash,style_hash,CAST(style_json AS TEXT),speed,output_format,status,file_path,duration_ms,CAST(timings_json AS TEXT),error,created_at,updated_at FROM tts_cache_entries WHERE cache_key=?`, cacheKey))
	return &entry, err
}

func (db *DB) ListReadyTTSCacheEntries() ([]TTSCacheEntry, error) {
	rows, err := db.conn.Query(`SELECT cache_key,provider,model,provider_voice_id,voice_version,language_tag,text_hash,style_hash,CAST(style_json AS TEXT),speed,output_format,status,file_path,duration_ms,CAST(timings_json AS TEXT),error,created_at,updated_at FROM tts_cache_entries WHERE status='ready' ORDER BY cache_key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []TTSCacheEntry{}
	for rows.Next() {
		entry, err := scanTTSCacheEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// ListPrunableTTSCacheEntries returns a bounded batch of cache rows that no
// audio asset references and that exceed either the age or count retention
// limit. Newest unreferenced rows within the TTL are retained.
func (db *DB) ListPrunableTTSCacheEntries(cutoff time.Time, retain, limit int) ([]TTSCacheEntry, error) {
	if retain < 0 {
		retain = 0
	}
	if limit < 1 {
		limit = 1
	} else if limit > 1000 {
		limit = 1000
	}
	rows, err := db.conn.Query(`SELECT c.cache_key,c.provider,c.model,c.provider_voice_id,c.voice_version,c.language_tag,c.text_hash,c.style_hash,CAST(c.style_json AS TEXT),c.speed,c.output_format,c.status,c.file_path,c.duration_ms,CAST(c.timings_json AS TEXT),c.error,c.created_at,c.updated_at
		FROM tts_cache_entries c
		WHERE NOT EXISTS (SELECT 1 FROM audio_assets a WHERE a.cache_key=c.cache_key)
		  AND (c.updated_at < ? OR c.cache_key NOT IN (
			SELECT c2.cache_key FROM tts_cache_entries c2
			WHERE NOT EXISTS (SELECT 1 FROM audio_assets a2 WHERE a2.cache_key=c2.cache_key)
			ORDER BY c2.updated_at DESC,c2.cache_key DESC LIMIT ?
		  ))
		ORDER BY c.updated_at,c.cache_key LIMIT ?`, cutoff.UTC().Format("2006-01-02 15:04:05"), retain, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []TTSCacheEntry{}
	for rows.Next() {
		entry, err := scanTTSCacheEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

type ttsCacheRowScanner interface {
	Scan(dest ...any) error
}

func scanTTSCacheEntry(row ttsCacheRowScanner) (TTSCacheEntry, error) {
	var entry TTSCacheEntry
	var styleJSON, timingsJSON string
	err := row.Scan(&entry.CacheKey, &entry.Provider, &entry.Model, &entry.ProviderVoiceID, &entry.VoiceVersion, &entry.LanguageTag, &entry.TextHash, &entry.StyleHash, &styleJSON, &entry.Speed, &entry.OutputFormat, &entry.Status, &entry.FilePath, &entry.DurationMS, &timingsJSON, &entry.Error, &entry.CreatedAt, &entry.UpdatedAt)
	entry.Style = json.RawMessage(styleJSON)
	entry.Timings = json.RawMessage(timingsJSON)
	return entry, err
}

// DeleteUnreferencedTTSCacheEntry removes a cache row only if no audio asset
// acquired it since the pruning candidates were selected.
func (db *DB) DeleteUnreferencedTTSCacheEntry(cacheKey string) (bool, error) {
	result, err := db.conn.Exec(`DELETE FROM tts_cache_entries
		WHERE cache_key=? AND NOT EXISTS (SELECT 1 FROM audio_assets a WHERE a.cache_key=tts_cache_entries.cache_key)`, strings.TrimSpace(cacheKey))
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected == 1, err
}

func (db *DB) InvalidateTTSCacheEntry(cacheKey, reason string) error {
	_, err := db.conn.Exec(`UPDATE tts_cache_entries SET status='failed',file_path='',error=?,updated_at=CURRENT_TIMESTAMP WHERE cache_key=?`, reason, cacheKey)
	return err
}

func (db *DB) QueueAudioAsset(asset AudioAsset, job TTSJob) (*AudioAsset, *TTSJob, error) {
	if asset.ID == "" {
		asset.ID = uuid.NewString()
	}
	if job.ID == "" {
		job.ID = uuid.NewString()
	}
	if strings.TrimSpace(asset.StoryID) == "" || asset.SourceMessageID <= 0 || asset.SegmentIndex < 0 || strings.TrimSpace(asset.Text) == "" || strings.TrimSpace(asset.CacheKey) == "" {
		return nil, nil, errors.New("canonical audio asset needs story, source message, segment, text, and cache identity")
	}
	asset.Style = validJSONOr(asset.Style, `{}`)
	asset.Timings = validJSONOr(asset.Timings, `[]`)
	if asset.Style == nil || asset.Timings == nil {
		return nil, nil, errors.New("audio style and timings must be valid JSON")
	}
	if asset.Speed == 0 {
		asset.Speed = 1
	}
	if asset.Status == "" {
		asset.Status = "queued"
	}
	var queuedJob *TTSJob
	err := db.WithTx(func(tx *sql.Tx) error {
		var branchID, commitID string
		if err := tx.QueryRow(`SELECT m.branch_id,m.source_commit_id FROM chat_messages m JOIN stories s ON s.id=m.story_id WHERE m.id=? AND m.story_id=? AND m.role='assistant' AND m.branch_id=s.active_branch_id AND m.source_commit_id!=''`, asset.SourceMessageID, asset.StoryID).Scan(&branchID, &commitID); err != nil {
			if err == sql.ErrNoRows {
				return errors.New("audio source must be a committed assistant message on the active branch")
			}
			return err
		}
		asset.BranchID, asset.SourceCommitID = branchID, commitID
		result, err := tx.Exec(`INSERT OR IGNORE INTO audio_assets(id,story_id,branch_id,source_commit_id,source_message_id,segment_index,segment_kind,speaker_entity_id,identity_id,form_id,voice_profile_id,provider,model,provider_voice_id,voice_version,language_tag,pronunciation_revision,text,text_hash,cache_key,style_json,speed,output_format,status,url,file_path,duration_ms,timings_json,generation_run_id,error)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			asset.ID, asset.StoryID, asset.BranchID, asset.SourceCommitID, asset.SourceMessageID, asset.SegmentIndex, asset.SegmentKind, nullableString(asset.SpeakerEntityID), nullableString(asset.IdentityID), nullableString(asset.FormID), asset.VoiceProfileID, asset.Provider, asset.Model, asset.ProviderVoiceID, asset.VoiceVersion, asset.LanguageTag, asset.PronunciationRevision, asset.Text, asset.TextHash, asset.CacheKey, asset.Style, asset.Speed, asset.OutputFormat, asset.Status, asset.URL, asset.FilePath, asset.DurationMS, asset.Timings, nullableString(asset.GenerationRunID), asset.Error)
		if err != nil {
			return fmt.Errorf("inserting audio asset: %w", err)
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			if err := tx.QueryRow(`SELECT id FROM audio_assets WHERE story_id=? AND branch_id=? AND source_message_id=? AND segment_index=? AND voice_profile_id=? AND cache_key=?`, asset.StoryID, asset.BranchID, asset.SourceMessageID, asset.SegmentIndex, asset.VoiceProfileID, asset.CacheKey).Scan(&asset.ID); err != nil {
				return err
			}
			return nil
		}
		if asset.Status == "ready" {
			return nil
		}
		job.AudioAssetID, job.StoryID, job.BranchID, job.SourceCommitID = asset.ID, asset.StoryID, asset.BranchID, asset.SourceCommitID
		if job.Status == "" {
			job.Status = "queued"
		}
		if job.MaxAttempts == 0 {
			job.MaxAttempts = 3
		}
		_, err = tx.Exec(`INSERT OR IGNORE INTO tts_jobs(id,audio_asset_id,story_id,branch_id,source_commit_id,status,provider,attempts,max_attempts,trace_id,parent_run_id,generation_run_id,error_class,error)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, job.ID, job.AudioAssetID, job.StoryID, job.BranchID, job.SourceCommitID, job.Status, job.Provider, job.Attempts, job.MaxAttempts, job.TraceID, nullableString(job.ParentRunID), nullableString(job.GenerationRunID), job.ErrorClass, job.Error)
		if err != nil {
			return fmt.Errorf("inserting TTS job: %w", err)
		}
		queuedJob = &job
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	stored, err := db.GetAudioAsset(asset.StoryID, asset.ID)
	if err != nil {
		// A deduplicated insert may have retained an earlier id.
		stored, err = db.GetAudioAssetByIdentity(asset.StoryID, asset.SourceMessageID, asset.SegmentIndex, asset.VoiceProfileID, asset.CacheKey)
	}
	if err != nil {
		return nil, nil, err
	}
	if stored.ID != asset.ID && queuedJob != nil {
		queuedJob = nil
	}
	return stored, queuedJob, nil
}

func (db *DB) GetAudioAsset(storyID, id string) (*AudioAsset, error) {
	return scanAudioAsset(db.conn.QueryRow(audioAssetSelect+` WHERE a.story_id=? AND a.id=? AND a.branch_id=(SELECT active_branch_id FROM stories WHERE id=?)`, storyID, id, storyID))
}

func (db *DB) GetAudioAssetByIdentity(storyID string, messageID int64, segmentIndex int, voiceProfileID, cacheKey string) (*AudioAsset, error) {
	return scanAudioAsset(db.conn.QueryRow(audioAssetSelect+` WHERE a.story_id=? AND a.source_message_id=? AND a.segment_index=? AND a.voice_profile_id=? AND a.cache_key=? AND a.branch_id=(SELECT active_branch_id FROM stories WHERE id=?)`, storyID, messageID, segmentIndex, voiceProfileID, cacheKey, storyID))
}

func (db *DB) ListMessageAudio(storyID string, messageID int64) ([]AudioAsset, error) {
	rows, err := db.conn.Query(audioAssetSelect+` WHERE a.story_id=? AND a.source_message_id=? AND a.branch_id=(SELECT active_branch_id FROM stories WHERE id=?) ORDER BY a.segment_index,a.created_at`, storyID, messageID, storyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	assets := []AudioAsset{}
	for rows.Next() {
		asset, err := scanAudioAsset(rows)
		if err != nil {
			return nil, err
		}
		assets = append(assets, *asset)
	}
	return assets, rows.Err()
}

func (db *DB) ListStoryAudio(storyID string) ([]AudioAsset, error) {
	rows, err := db.conn.Query(audioAssetSelect+` WHERE a.story_id=? AND a.branch_id=(SELECT active_branch_id FROM stories WHERE id=?) ORDER BY a.source_message_id,a.segment_index,a.created_at`, storyID, storyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	assets := []AudioAsset{}
	for rows.Next() {
		asset, err := scanAudioAsset(rows)
		if err != nil {
			return nil, err
		}
		assets = append(assets, *asset)
	}
	return assets, rows.Err()
}

// GetActiveAudioAssetByID resolves an asset only when its immutable branch
// lineage still belongs to the story's active branch.
func (db *DB) GetActiveAudioAssetByID(id string) (*AudioAsset, error) {
	return scanAudioAsset(db.conn.QueryRow(audioAssetSelect+` JOIN stories s ON s.id=a.story_id WHERE a.id=? AND a.branch_id=s.active_branch_id`, id))
}

func (db *DB) ListMessageTTSJobs(storyID string, messageID int64) ([]TTSJob, error) {
	rows, err := db.conn.Query(`SELECT j.id,j.audio_asset_id,j.story_id,j.branch_id,j.source_commit_id,j.status,j.provider,j.attempts,j.max_attempts,j.next_attempt_at,j.trace_id,COALESCE(j.parent_run_id,''),COALESCE(j.generation_run_id,''),j.error_class,j.error,j.created_at,j.updated_at
		FROM tts_jobs j JOIN audio_assets a ON a.id=j.audio_asset_id JOIN stories s ON s.id=j.story_id
		WHERE j.story_id=? AND a.source_message_id=? AND j.branch_id=s.active_branch_id ORDER BY a.segment_index,j.created_at`, storyID, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs := []TTSJob{}
	for rows.Next() {
		var job TTSJob
		var next sql.NullString
		if err := rows.Scan(&job.ID, &job.AudioAssetID, &job.StoryID, &job.BranchID, &job.SourceCommitID, &job.Status, &job.Provider, &job.Attempts, &job.MaxAttempts, &next, &job.TraceID, &job.ParentRunID, &job.GenerationRunID, &job.ErrorClass, &job.Error, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, err
		}
		if next.Valid {
			job.NextAttemptAt = next.String
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (db *DB) ListStoryTTSJobs(storyID string) ([]TTSJob, error) {
	rows, err := db.conn.Query(`SELECT j.id,j.audio_asset_id,j.story_id,j.branch_id,j.source_commit_id,j.status,j.provider,j.attempts,j.max_attempts,j.next_attempt_at,j.trace_id,COALESCE(j.parent_run_id,''),COALESCE(j.generation_run_id,''),j.error_class,j.error,j.created_at,j.updated_at
		FROM tts_jobs j JOIN stories s ON s.id=j.story_id WHERE j.story_id=? AND j.branch_id=s.active_branch_id ORDER BY j.created_at,j.id`, storyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs := []TTSJob{}
	for rows.Next() {
		var job TTSJob
		var next sql.NullString
		if err := rows.Scan(&job.ID, &job.AudioAssetID, &job.StoryID, &job.BranchID, &job.SourceCommitID, &job.Status, &job.Provider, &job.Attempts, &job.MaxAttempts, &next, &job.TraceID, &job.ParentRunID, &job.GenerationRunID, &job.ErrorClass, &job.Error, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, err
		}
		if next.Valid {
			job.NextAttemptAt = next.String
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

const audioAssetSelect = `SELECT a.id,a.story_id,a.branch_id,a.source_commit_id,a.source_message_id,a.segment_index,a.segment_kind,COALESCE(a.speaker_entity_id,''),COALESCE(a.identity_id,''),COALESCE(a.form_id,''),a.voice_profile_id,a.provider,a.model,a.provider_voice_id,a.voice_version,a.language_tag,a.pronunciation_revision,a.text,a.text_hash,a.cache_key,a.style_json,a.speed,a.output_format,a.status,a.url,a.file_path,a.duration_ms,a.timings_json,COALESCE(a.generation_run_id,''),a.error,a.created_at,a.updated_at FROM audio_assets a`

func scanAudioAsset(row rowScanner) (*AudioAsset, error) {
	var asset AudioAsset
	err := row.Scan(&asset.ID, &asset.StoryID, &asset.BranchID, &asset.SourceCommitID, &asset.SourceMessageID, &asset.SegmentIndex, &asset.SegmentKind, &asset.SpeakerEntityID, &asset.IdentityID, &asset.FormID, &asset.VoiceProfileID, &asset.Provider, &asset.Model, &asset.ProviderVoiceID, &asset.VoiceVersion, &asset.LanguageTag, &asset.PronunciationRevision, &asset.Text, &asset.TextHash, &asset.CacheKey, &asset.Style, &asset.Speed, &asset.OutputFormat, &asset.Status, &asset.URL, &asset.FilePath, &asset.DurationMS, &asset.Timings, &asset.GenerationRunID, &asset.Error, &asset.CreatedAt, &asset.UpdatedAt)
	return &asset, err
}

func (db *DB) ClaimTTSJob(jobID string) (*TTSJob, error) {
	result, err := db.conn.Exec(`UPDATE tts_jobs SET status='running',attempts=attempts+1,updated_at=CURRENT_TIMESTAMP WHERE id=? AND status IN ('queued','failed') AND attempts<max_attempts`, jobID)
	if err != nil {
		return nil, err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return nil, errors.New("TTS job is not claimable")
	}
	if _, err := db.conn.Exec(`UPDATE audio_assets SET status='running',updated_at=CURRENT_TIMESTAMP WHERE id=(SELECT audio_asset_id FROM tts_jobs WHERE id=?)`, jobID); err != nil {
		return nil, err
	}
	return db.GetTTSJob(jobID)
}

func (db *DB) GetTTSJob(jobID string) (*TTSJob, error) {
	var job TTSJob
	var next sql.NullString
	err := db.conn.QueryRow(`SELECT id,audio_asset_id,story_id,branch_id,source_commit_id,status,provider,attempts,max_attempts,next_attempt_at,trace_id,COALESCE(parent_run_id,''),COALESCE(generation_run_id,''),error_class,error,created_at,updated_at FROM tts_jobs WHERE id=?`, jobID).
		Scan(&job.ID, &job.AudioAssetID, &job.StoryID, &job.BranchID, &job.SourceCommitID, &job.Status, &job.Provider, &job.Attempts, &job.MaxAttempts, &next, &job.TraceID, &job.ParentRunID, &job.GenerationRunID, &job.ErrorClass, &job.Error, &job.CreatedAt, &job.UpdatedAt)
	if next.Valid {
		job.NextAttemptAt = next.String
	}
	return &job, err
}

func (db *DB) CancelTTSJob(storyID, jobID string) (*TTSJob, error) {
	err := db.WithTx(func(tx *sql.Tx) error {
		result, err := tx.Exec(`UPDATE tts_jobs SET status='cancelled',next_attempt_at=NULL,error_class='',error='',updated_at=CURRENT_TIMESTAMP
			WHERE id=? AND story_id=? AND branch_id=(SELECT active_branch_id FROM stories WHERE id=?) AND status IN ('queued','running','failed')`, jobID, storyID, storyID)
		if err != nil {
			return err
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return errors.New("TTS job is not cancelable on the active branch")
		}
		_, err = tx.Exec(`UPDATE audio_assets SET status='cancelled',error='',updated_at=CURRENT_TIMESTAMP WHERE id=(SELECT audio_asset_id FROM tts_jobs WHERE id=?)`, jobID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return db.GetTTSJob(jobID)
}

func (db *DB) RetryTTSJob(storyID, jobID string) (*TTSJob, error) {
	err := db.WithTx(func(tx *sql.Tx) error {
		result, err := tx.Exec(`UPDATE tts_jobs SET status='queued',attempts=0,next_attempt_at=NULL,generation_run_id=NULL,error_class='',error='',updated_at=CURRENT_TIMESTAMP
			WHERE id=? AND story_id=? AND branch_id=(SELECT active_branch_id FROM stories WHERE id=?) AND status IN ('failed','cancelled')`, jobID, storyID, storyID)
		if err != nil {
			return err
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return errors.New("TTS job is not retryable on the active branch")
		}
		_, err = tx.Exec(`UPDATE audio_assets SET status='queued',error='',updated_at=CURRENT_TIMESTAMP WHERE id=(SELECT audio_asset_id FROM tts_jobs WHERE id=?)`, jobID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return db.GetTTSJob(jobID)
}

func (db *DB) CompleteTTSJob(jobID string, cache TTSCacheEntry, filePath string, durationMS int64, timings json.RawMessage, generationRunID string) error {
	cache.Style = validJSONOr(cache.Style, `{}`)
	timings = validJSONOr(timings, `[]`)
	if cache.Style == nil || timings == nil {
		return errors.New("cache style and timings must be valid JSON")
	}
	return db.WithTx(func(tx *sql.Tx) error {
		var status string
		if err := tx.QueryRow(`SELECT status FROM tts_jobs WHERE id=?`, jobID).Scan(&status); err != nil {
			return err
		}
		if status != "running" {
			return fmt.Errorf("TTS job completion rejected from status %q", status)
		}
		_, err := tx.Exec(`INSERT INTO tts_cache_entries(cache_key,provider,model,provider_voice_id,voice_version,language_tag,text_hash,style_hash,style_json,speed,output_format,status,file_path,duration_ms,timings_json,error)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,'ready',?,?,?,'') ON CONFLICT(cache_key) DO UPDATE SET status='ready',file_path=excluded.file_path,duration_ms=excluded.duration_ms,timings_json=excluded.timings_json,error='',updated_at=CURRENT_TIMESTAMP`, cache.CacheKey, cache.Provider, cache.Model, cache.ProviderVoiceID, cache.VoiceVersion, cache.LanguageTag, cache.TextHash, cache.StyleHash, cache.Style, cache.Speed, cache.OutputFormat, filePath, durationMS, timings)
		if err != nil {
			return err
		}
		result, err := tx.Exec(`UPDATE audio_assets SET status='ready',file_path=?,duration_ms=?,timings_json=?,generation_run_id=?,error='',updated_at=CURRENT_TIMESTAMP WHERE id=(SELECT audio_asset_id FROM tts_jobs WHERE id=?)`, filePath, durationMS, timings, nullableString(generationRunID), jobID)
		if err != nil {
			return err
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			return errors.New("TTS job audio asset not found")
		}
		_, err = tx.Exec(`UPDATE tts_jobs SET status='succeeded',generation_run_id=?,error_class='',error='',updated_at=CURRENT_TIMESTAMP WHERE id=?`, nullableString(generationRunID), jobID)
		return err
	})
}

func (db *DB) FailTTSJob(jobID, errorClass, detail, generationRunID string, retry bool) error {
	status := "failed"
	next := any(nil)
	if retry {
		next = time.Now().UTC().Add(30 * time.Second)
	}
	return db.WithTx(func(tx *sql.Tx) error {
		result, err := tx.Exec(`UPDATE tts_jobs SET status=?,next_attempt_at=?,generation_run_id=?,error_class=?,error=?,updated_at=CURRENT_TIMESTAMP WHERE id=? AND status!='cancelled'`, status, next, nullableString(generationRunID), errorClass, detail, jobID)
		if err != nil {
			return err
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			return nil
		}
		_, err = tx.Exec(`UPDATE audio_assets SET status='failed',generation_run_id=?,error=?,updated_at=CURRENT_TIMESTAMP WHERE id=(SELECT audio_asset_id FROM tts_jobs WHERE id=?)`, nullableString(generationRunID), detail, jobID)
		return err
	})
}

func normalizeLanguageTag(tag string) string {
	tag = strings.TrimSpace(strings.ReplaceAll(tag, "_", "-"))
	if tag == "" {
		return ""
	}
	parts := strings.Split(tag, "-")
	parts[0] = strings.ToLower(parts[0])
	for index := 1; index < len(parts); index++ {
		if len(parts[index]) == 2 {
			parts[index] = strings.ToUpper(parts[index])
		} else {
			parts[index] = strings.ToLower(parts[index])
		}
	}
	return strings.Join(parts, "-")
}

func validJSONOr(value json.RawMessage, fallback string) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(fallback)
	}
	if !json.Valid(value) {
		return nil
	}
	return value
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
