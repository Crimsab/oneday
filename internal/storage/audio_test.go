package storage

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func newAudioStory(t *testing.T) (*DB, *Story) {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "audio.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`INSERT INTO stories(id,name,language) VALUES('story-audio','Audio','it-IT')`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if _, err := db.EnsureStoryTimeline("story-audio"); err != nil {
		db.Close()
		t.Fatal(err)
	}
	story, err := db.GetStory("story-audio")
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	return db, story
}

func TestMigrationV35CreatesCanonicalAudioSchema(t *testing.T) {
	db, _ := newAudioStory(t)
	defer db.Close()
	for _, table := range []string{"story_tts_settings", "voice_profiles", "character_voice_assignments", "pronunciation_lexicon", "tts_cache_entries", "audio_assets", "tts_jobs"} {
		var got string
		if err := db.Conn().QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&got); err != nil {
			t.Fatalf("missing %s: %v", table, err)
		}
	}
	var version int
	if err := db.Conn().QueryRow(`SELECT MAX(version) FROM schema_version`).Scan(&version); err != nil || version < 35 {
		t.Fatalf("schema version=%d err=%v", version, err)
	}
}

func TestTTSSettingsDefaultOffAndValidatePolicy(t *testing.T) {
	db, story := newAudioStory(t)
	defer db.Close()
	settings, err := db.GetStoryTTSSettings(story.ID)
	if err != nil || settings.Mode != "off" || settings.Autoplay {
		t.Fatalf("default settings=%+v err=%v", settings, err)
	}
	saved, err := db.UpsertStoryTTSSettings(StoryTTSSettings{StoryID: story.ID, Mode: "dialogue", Autoplay: true, DefaultLanguage: "it_it", ProviderPolicy: json.RawMessage(`{"order":["local"]}`)})
	if err != nil || saved.DefaultLanguage != "it-IT" || !saved.Autoplay {
		t.Fatalf("saved settings=%+v err=%v", saved, err)
	}
	if _, err := db.UpsertStoryTTSSettings(StoryTTSSettings{StoryID: story.ID, Mode: "provisional"}); err == nil {
		t.Fatal("invalid mode was accepted")
	}
}

func TestMajorVoiceUniquenessAndExplicitOverride(t *testing.T) {
	db, story := newAudioStory(t)
	defer db.Close()
	voice, err := db.UpsertVoiceProfile(VoiceProfile{Provider: "local", Model: "piper", ProviderVoiceID: "it-paola", DisplayName: "Paola", LanguageTags: json.RawMessage(`["it-IT"]`), Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	narrator, err := db.UpsertVoiceAssignment(VoiceAssignment{StoryID: story.ID, Role: "narrator", VoiceProfileID: voice.ID, EnabledMode: "on", Importance: "major", Locked: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.UpsertVoiceAssignment(VoiceAssignment{StoryID: story.ID, Role: "protagonist", VoiceProfileID: voice.ID, EnabledMode: "on", Importance: "major"})
	if err == nil || !strings.Contains(err.Error(), "distinct") {
		t.Fatalf("major collision error=%v", err)
	}
	if _, err := db.UpsertVoiceAssignment(VoiceAssignment{StoryID: story.ID, Role: "protagonist", VoiceProfileID: voice.ID, EnabledMode: "on", Importance: "major", AllowDuplicate: true}); err != nil {
		t.Fatalf("explicit duplicate override failed: %v", err)
	}
	other, err := db.UpsertVoiceProfile(VoiceProfile{Provider: "cloud", Model: "speech", ProviderVoiceID: "alloy", DisplayName: "Alloy", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	narrator.VoiceProfileID = other.ID
	if _, err := db.UpsertVoiceAssignment(*narrator); err == nil || !strings.Contains(err.Error(), "locked") {
		t.Fatalf("locked assignment changed: %v", err)
	}
}

func TestPronunciationUpsertRevisesEntry(t *testing.T) {
	db, story := newAudioStory(t)
	defer db.Close()
	first, err := db.UpsertPronunciation(PronunciationEntry{StoryID: story.ID, LanguageTag: "it_IT", SourceText: "Lyanna", Pronunciation: "liˈanna", Alphabet: "ipa"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := db.UpsertPronunciation(PronunciationEntry{StoryID: story.ID, LanguageTag: "it-IT", SourceText: "Lyanna", Pronunciation: "liˈaːnna", Alphabet: "ipa"})
	if err != nil || second.ID != first.ID || second.Revision != 2 {
		t.Fatalf("revised=%+v first=%+v err=%v", second, first, err)
	}
}
