package engine

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/storage"
)

type stubASCIIProvider struct {
	models []string
}

func (s *stubASCIIProvider) Name() string { return "stub-ascii" }

func (s *stubASCIIProvider) Complete(_ context.Context, req ai.Request) (ai.Response, error) {
	s.models = append(s.models, req.Model)
	if req.Model == "ascii-ambient" {
		return ai.Response{}, context.DeadlineExceeded
	}
	return ai.Response{
		Content: `{"ascii_art":"+-+\n| |\n+-+"}`,
		Model:   "fallback-main",
	}, nil
}

func TestGenerateAmbientASCIIRetriesWithoutExplicitModel(t *testing.T) {
	provider := &stubASCIIProvider{}
	router, err := ai.NewRouter([]ai.Provider{provider})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}

	db, err := storage.Open(filepath.Join(t.TempDir(), "ascii-test.db"))
	if err != nil {
		t.Fatalf("Open db: %v", err)
	}
	defer db.Close()

	if _, err := db.Conn().Exec(
		`INSERT INTO stories (id, name) VALUES (?, ?)`,
		"story-1", "Story",
	); err != nil {
		t.Fatalf("insert story: %v", err)
	}
	if _, err := db.Conn().Exec(
		`INSERT INTO sessions (id, story_id, started_at) VALUES (?, ?, ?)`,
		"session-1", "story-1", time.Now().UTC(),
	); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if _, err := db.Conn().Exec(
		`INSERT INTO chat_messages (session_id, story_id, turn, role, content, message_type) VALUES (?, ?, ?, ?, ?, ?)`,
		"session-1", "story-1", 1, "assistant", "A harbor scene unfolds.", "narrative",
	); err != nil {
		t.Fatalf("insert assistant message: %v", err)
	}

	narrator := NewNarrator(
		router,
		db,
		&storage.Story{ID: "story-1", Name: "Story"},
		nil,
		&storage.WorldState{StoryID: "story-1", CurrentLocation: "Harbor"},
		nil,
		ContextConfig{},
		config.GenerationConfig{},
		config.ASCIIArtConfig{Enabled: true, Model: "ascii-ambient", Temperature: 0.4, MaxTokens: 300, TimeoutSeconds: 5},
		t.TempDir(),
		5,
	)

	art, model, err := narrator.GenerateAmbientASCII(context.Background(), 1, &NarrativeResponse{
		Narrative: "Lanterns tremble over the harbor.",
		Mood:      "tense",
		SceneType: "arrival",
		Location:  "Harbor",
		ASCIICue: &ASCIIArtCue{
			Kind:    "location",
			Subject: "Harbor gate",
			Detail:  "Iron beams and lanterns",
		},
	})
	if err != nil {
		t.Fatalf("GenerateAmbientASCII: %v", err)
	}
	if art == "" {
		t.Fatal("expected fallback ASCII art, got empty string")
	}
	if model != "fallback-main" {
		t.Fatalf("model = %q, want fallback-main", model)
	}
	if len(provider.models) != 2 {
		t.Fatalf("models called = %v, want 2 attempts", provider.models)
	}
	if provider.models[0] != "ascii-ambient" || provider.models[1] != "" {
		t.Fatalf("models called = %v, want explicit model then provider default", provider.models)
	}

	var metadataJSON string
	if err := db.Conn().QueryRow(
		`SELECT metadata_json FROM chat_messages WHERE story_id = ? AND turn = ? AND role = 'assistant' ORDER BY id DESC LIMIT 1`,
		"story-1", 1,
	).Scan(&metadataJSON); err != nil {
		t.Fatalf("query metadata: %v", err)
	}
	if metadataJSON == "" || metadataJSON == "{}" {
		t.Fatal("expected persisted ASCII metadata, got empty metadata_json")
	}
}
