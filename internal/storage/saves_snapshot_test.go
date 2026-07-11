package storage

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSaveSnapshotSealAndValidateFull(t *testing.T) {
	snap := completeSnapshotFixture(t)

	if err := snap.SealFullRollback(); err != nil {
		t.Fatalf("SealFullRollback: %v", err)
	}

	validation := snap.ValidateRollbackState()
	if validation.State != SnapshotStateFull {
		t.Fatalf("validation = %+v, want state %q", validation, SnapshotStateFull)
	}
	if snap.FormatVersion != CurrentSnapshotFormatVersion {
		t.Fatalf("format version = %d, want %d", snap.FormatVersion, CurrentSnapshotFormatVersion)
	}
	if snap.PayloadChecksum == "" {
		t.Fatal("sealed snapshot has empty checksum")
	}
	if got := snap.Manifest.Collections[SnapshotCollectionNPCs]; got != 1 {
		t.Fatalf("manifest npc count = %d, want 1", got)
	}
	if !snap.HasFullRollbackState() {
		t.Fatal("sealed snapshot should report full rollback state")
	}
}

func TestSaveSnapshotValidationClassifiesUnsafePayloads(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*SaveSnapshot)
		want   SnapshotState
	}{
		{
			name: "tampered payload",
			mutate: func(s *SaveSnapshot) {
				s.WorldStateJSON = `{"id":"world-1","story_id":"story-1","current_turn":99}`
			},
			want: SnapshotStateCorrupt,
		},
		{
			name: "incomplete manifest",
			mutate: func(s *SaveSnapshot) {
				delete(s.Manifest.Collections, SnapshotCollectionChatMessages)
				s.PayloadChecksum, _ = s.payloadChecksum()
			},
			want: SnapshotStateIncomplete,
		},
		{
			name: "future format",
			mutate: func(s *SaveSnapshot) {
				s.FormatVersion = CurrentSnapshotFormatVersion + 1
			},
			want: SnapshotStateIncompatible,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snap := completeSnapshotFixture(t)
			if err := snap.SealFullRollback(); err != nil {
				t.Fatalf("SealFullRollback: %v", err)
			}
			tt.mutate(snap)

			if got := snap.ValidateRollbackState().State; got != tt.want {
				t.Fatalf("validation state = %q, want %q", got, tt.want)
			}
			if snap.HasFullRollbackState() {
				t.Fatalf("unsafe snapshot state %q reported full rollback support", tt.want)
			}
		})
	}
}

func TestSaveSnapshotValidationRecognizesLegacyPayload(t *testing.T) {
	snap := completeSnapshotFixture(t)
	snap.FormatVersion = 0
	snap.Manifest = SnapshotManifest{}
	snap.PayloadChecksum = ""

	validation := snap.ValidateRollbackState()
	if validation.State != SnapshotStateLegacy {
		t.Fatalf("validation = %+v, want state %q", validation, SnapshotStateLegacy)
	}
	if snap.HasFullRollbackState() {
		t.Fatal("legacy snapshot should not be classified as verified full rollback")
	}
}

func completeSnapshotFixture(t *testing.T) *SaveSnapshot {
	t.Helper()
	now := time.Date(2026, time.July, 11, 12, 0, 0, 0, time.UTC)
	charBytes, err := json.Marshal(Character{ID: "char-1", StoryID: "story-1", Name: "Mara"})
	if err != nil {
		t.Fatalf("marshal character: %v", err)
	}
	worldBytes, err := json.Marshal(WorldState{ID: "world-1", StoryID: "story-1", CurrentTurn: 7, CurrentChapter: 2})
	if err != nil {
		t.Fatalf("marshal world: %v", err)
	}

	return &SaveSnapshot{
		ID:             "save-1",
		StoryID:        "story-1",
		Name:           "Crossroads",
		Turn:           7,
		Chapter:        2,
		Location:       "North Gate",
		CharacterJSON:  string(charBytes),
		WorldStateJSON: string(worldBytes),
		SessionID:      "session-1",
		MetadataJSON:   `{}`,
		Story: &Story{
			ID:              "story-1",
			Name:            "Rollback Tale",
			SettingJSON:     `{}`,
			StatsSchemaJSON: `{}`,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		NPCs: []NPC{{
			ID:            "npc-1",
			StoryID:       "story-1",
			Name:          "Gate Watcher",
			DiscoveryJSON: `{"stage":"identified"}`,
			CreatedAt:     now,
			UpdatedAt:     now,
		}},
		Achievements: []Achievement{},
		Chapters:     []Chapter{},
		Sessions: []Session{{
			ID:        "session-1",
			StoryID:   "story-1",
			StartedAt: now,
		}},
		ChatMessages: []ChatMessage{},
		RAGChunks:    []RAGChunkSnapshot{},
		CombatLogs:   []CombatLog{},
		SessionFiles: map[string]string{"sessions/session-1/main.jsonl": "turn-1\n"},
		CreatedAt:    now,
	}
}
