package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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

func (voice VoiceProfile) Signature() string {
	return strings.Join([]string{voice.Provider, voice.Model, voice.ProviderVoiceID, voice.Version, voice.StyleFamily}, ":")
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
