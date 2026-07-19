package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/game/contracts"
	gameservice "github.com/crimsab/oneday/internal/game/service"
	"github.com/crimsab/oneday/internal/setup"
	"github.com/crimsab/oneday/internal/storage"
)

type matrixFirstRunProvider struct{}

func (matrixFirstRunProvider) Name() string { return "matrix-first-run-fake" }

func (matrixFirstRunProvider) Complete(context.Context, ai.Request) (ai.Response, error) {
	return ai.Response{Model: "matrix-first-run-fake", Content: `{
		"narrative": "The market opens beneath a clear, safe sky.",
		"choices": [{"id": 1, "text": "Ask the vendor for directions.", "intent": "social", "risk": "low", "scope": "npc", "certainty": "safe"}],
		"mood": "curious",
		"location": "Market"
	}`}, nil
}

func TestFirstRunMatrixCLISetupDoctorFixtureStoryAndAction(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	configPath := filepath.Join(root, "config.yaml")
	t.Setenv("ONEDAY_CONFIG", configPath)

	inputReader, inputWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := inputWriter.WriteString("\n1\nmatrix-test-model\n\n\n\n0\n"); err != nil {
		t.Fatal(err)
	}
	if err := inputWriter.Close(); err != nil {
		t.Fatal(err)
	}
	oldStdin := os.Stdin
	os.Stdin = inputReader
	t.Cleanup(func() {
		os.Stdin = oldStdin
		_ = inputReader.Close()
	})
	if err := runSetup([]string{"setup", "--reconfigure"}); err != nil {
		t.Fatalf("setup from empty profile: %v", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("load setup config: %v", err)
	}
	if !cfg.AI.Codex.Enabled || cfg.AI.Codex.Model != "matrix-test-model" {
		t.Fatalf("setup did not persist the selected fake-safe narrative configuration: %#v", cfg.AI.Codex)
	}

	var doctorOut bytes.Buffer
	if err := runDoctorTo([]string{"doctor", "--json"}, &doctorOut, setup.Dependencies{
		Narrative: func(context.Context, config.Config) error { return nil },
	}); err != nil {
		t.Fatalf("doctor from fresh setup: %v", err)
	}
	if strings.Contains(doctorOut.String(), root) || !strings.Contains(doctorOut.String(), `"code": "NARRATIVE_READY"`) {
		t.Fatalf("doctor did not report redacted readiness: %s", doctorOut.String())
	}

	dbPath := filepath.Join(root, "first-run.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("open first-run database: %v", err)
	}
	now := time.Now().UTC()
	storyID := "matrix-first-story"
	createMatrixFirstRunStory(t, db, storyID, now)

	router, err := ai.NewRouter([]ai.Provider{matrixFirstRunProvider{}})
	if err != nil {
		t.Fatalf("build fake router: %v", err)
	}
	turns := gameservice.NewInProcessTurnService(cfg, db, router)
	snapshot, err := turns.Snapshot(context.Background(), storyID)
	if err != nil {
		t.Fatalf("load first story snapshot: %v", err)
	}
	request, err := json.Marshal(contracts.SubmitActionRequest{
		StoryID:        storyID,
		SessionID:      snapshot.SessionID,
		ClientTurn:     snapshot.Turn,
		ClientRevision: snapshot.Revision,
		IdempotencyKey: "matrix-first-action",
		Action:         contracts.PlayerAction{Kind: contracts.ActionKindFreeText, Text: "Ask the vendor for directions."},
	})
	if err != nil {
		t.Fatal(err)
	}
	var actionOut bytes.Buffer
	if err := runGatewayTurn(context.Background(), cfg, db, router, bytes.NewReader(request), &actionOut); err != nil {
		t.Fatalf("submit first playable action: %v", err)
	}
	var action gatewayTurnResponse
	if err := json.Unmarshal(actionOut.Bytes(), &action); err != nil {
		t.Fatalf("decode first action response: %v; output=%s", err, actionOut.String())
	}
	if len(action.Events) == 0 || action.Events[len(action.Events)-1].Type != contracts.EventTurnCommitted {
		t.Fatalf("first action did not commit canonical events: %#v", action.Events)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close first-run database: %v", err)
	}
	reopened, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("reopen persisted first-run database: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	world, err := reopened.GetWorldState(storyID)
	if err != nil {
		t.Fatalf("load persisted world state: %v", err)
	}
	if world.CurrentTurn != 1 || world.CurrentLocation != "Market" {
		t.Fatalf("first action was not persisted: turn=%d location=%q", world.CurrentTurn, world.CurrentLocation)
	}
}

func createMatrixFirstRunStory(t *testing.T, db *storage.DB, storyID string, now time.Time) {
	t.Helper()
	if err := db.CreateStory(&storage.Story{
		ID: storyID, Name: "Matrix First Story", Description: "Fixture story for first-run proof",
		Genre: "fantasy", Tone: "curious", Language: "en", SettingJSON: `{}`, StatsSchemaJSON: `{}`,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create first fixture story: %v", err)
	}
	if err := db.CreateCharacter(&storage.Character{
		ID: "matrix-character", StoryID: storyID, Name: "Mira", Background: "Fixture protagonist",
		StatsJSON: `{"vitals":{"hp":{"current":10,"max":10}},"skills":{}}`, TraitsJSON: `[]`, SkillsJSON: `{}`,
		InventoryJSON: `[]`, KnownRecipesJSON: `[]`, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create first fixture character: %v", err)
	}
	if err := db.CreateWorldState(&storage.WorldState{
		ID: "matrix-world", StoryID: storyID, CurrentLocation: "Harbor", KnownLocationsJSON: `["Harbor"]`,
		GlobalEventsJSON: `[]`, FactionStandingsJSON: `{}`, StoryHooksJSON: `[]`, WorldReactionsJSON: `[]`,
		PlayerGuidanceJSON: `[]`, CurrentChapter: 1, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create first fixture world: %v", err)
	}
}
