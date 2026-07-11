package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/storage"
	"github.com/crimsab/oneday/internal/tui/views"
)

func TestAppShowSaveListCancelReturnsToNarrative(t *testing.T) {
	t.Parallel()

	fx := newAppSmokeFixture(t)
	app := New(fx.cfg, fx.db, nil)
	app.width = 120
	app.height = 40
	app.mountNarrativeView(fx.story, fx.char, fx.world, fx.session, nil, false)

	model, cmd := app.Update(views.ShowSaveListMsg{
		Saves: []storage.SaveSnapshot{
			{
				ID:        "save-1",
				StoryID:   fx.story.ID,
				Name:      "Manual Save",
				Turn:      fx.world.CurrentTurn,
				Location:  fx.world.CurrentLocation,
				CreatedAt: fx.now,
			},
		},
	})
	if cmd != nil {
		t.Fatal("ShowSaveListMsg returned unexpected cmd")
	}

	updated, ok := model.(App)
	if !ok {
		t.Fatalf("Update returned %T, want App", model)
	}
	if updated.view != ViewSaveLoad || updated.saveLoad == nil {
		t.Fatalf("app did not enter save picker view: view=%v saveLoad=%v", updated.view, updated.saveLoad != nil)
	}

	model, cmd = updated.Update(views.SaveLoadCancelMsg{})
	if cmd != nil {
		t.Fatal("SaveLoadCancelMsg returned unexpected cmd")
	}
	updated, ok = model.(App)
	if !ok {
		t.Fatalf("Update returned %T, want App", model)
	}
	if updated.view != ViewNarrative {
		t.Fatalf("view = %v, want ViewNarrative", updated.view)
	}
	if updated.saveLoad != nil {
		t.Fatal("saveLoad should be cleared after cancel")
	}
}

func TestAppLoadSaveAndResumeRestoresStoryState(t *testing.T) {
	t.Parallel()

	fx := newAppSmokeFixture(t)
	if err := fx.db.AppendChatMessage(&storage.ChatMessage{
		SessionID:    fx.session.SessionID(),
		StoryID:      fx.story.ID,
		Turn:         fx.world.CurrentTurn,
		Role:         "assistant",
		Content:      "The square is calm.",
		MessageType:  "narrative",
		MetadataJSON: "{}",
		CreatedAt:    fx.now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("AppendChatMessage: %v", err)
	}

	snap, err := engine.SaveGame(fx.db, fx.dataDir, fx.story, fx.char, fx.world, fx.session.SessionID(), "Recovery Slot")
	if err != nil {
		t.Fatalf("SaveGame: %v", err)
	}

	mutatedChar := *fx.char
	mutatedChar.Name = "Future Mara"
	mutatedChar.InventoryJSON = `{"items":["future-key"]}`
	if err := fx.db.UpdateCharacterFull(&mutatedChar); err != nil {
		t.Fatalf("UpdateCharacterFull: %v", err)
	}

	mutatedWorld := *fx.world
	mutatedWorld.CurrentLocation = "Future Keep"
	mutatedWorld.CurrentTurn = 99
	if err := fx.db.UpdateWorldState(&mutatedWorld); err != nil {
		t.Fatalf("UpdateWorldState: %v", err)
	}

	app := New(fx.cfg, fx.db, nil)
	app.width = 120
	app.height = 40

	cmd, err := app.loadSaveAndResume(fx.story.ID, snap.ID)
	if err != nil {
		t.Fatalf("loadSaveAndResume: %v", err)
	}
	if cmd == nil {
		t.Fatal("loadSaveAndResume returned nil cmd")
	}
	if app.view != ViewNarrative || app.narrative == nil {
		t.Fatalf("app did not mount narrative view: view=%v narrative=%v", app.view, app.narrative != nil)
	}

	restoredChar, err := fx.db.GetCharacterByStory(fx.story.ID)
	if err != nil {
		t.Fatalf("GetCharacterByStory: %v", err)
	}
	if restoredChar.Name != fx.char.Name {
		t.Fatalf("restored character name = %q, want %q", restoredChar.Name, fx.char.Name)
	}
	if restoredChar.InventoryJSON != fx.char.InventoryJSON {
		t.Fatalf("restored inventory = %q, want %q", restoredChar.InventoryJSON, fx.char.InventoryJSON)
	}

	restoredWorld, err := fx.db.GetWorldState(fx.story.ID)
	if err != nil {
		t.Fatalf("GetWorldState: %v", err)
	}
	if restoredWorld.CurrentLocation != fx.world.CurrentLocation {
		t.Fatalf("restored location = %q, want %q", restoredWorld.CurrentLocation, fx.world.CurrentLocation)
	}
	if restoredWorld.CurrentTurn != fx.world.CurrentTurn {
		t.Fatalf("restored turn = %d, want %d", restoredWorld.CurrentTurn, fx.world.CurrentTurn)
	}

	msg := cmd()
	if msg == nil {
		t.Fatal("resume cmd returned nil msg")
	}

	model, _ := app.Update(msg)
	updated, ok := model.(App)
	if !ok {
		t.Fatalf("Update returned %T, want App", model)
	}
	if updated.narrative == nil {
		t.Fatal("narrative view missing after resume message")
	}
	if !strings.Contains(updated.View(), "The square is calm.") {
		t.Fatalf("resume view missing restored narrative:\n%s", updated.View())
	}
}

func TestAppLoadLegacySaveShowsPartialRollbackWarning(t *testing.T) {
	t.Parallel()

	fx := newAppSmokeFixture(t)
	snap, err := engine.SaveGame(fx.db, fx.dataDir, fx.story, fx.char, fx.world, fx.session.SessionID(), "Legacy Slot")
	if err != nil {
		t.Fatalf("SaveGame: %v", err)
	}

	legacySnapshot := *snap
	legacySnapshot.FormatVersion = 0
	legacySnapshot.Manifest = storage.SnapshotManifest{}
	legacySnapshot.PayloadChecksum = ""
	legacySnapshot.Story = nil
	legacySnapshot.NPCs = nil
	legacySnapshot.Achievements = nil
	legacySnapshot.Chapters = nil
	legacySnapshot.Sessions = nil
	legacySnapshot.ChatMessages = nil
	legacySnapshot.RAGChunks = nil
	legacySnapshot.CombatLogs = nil
	legacySnapshot.SessionFiles = nil

	legacyBytes, err := json.Marshal(&legacySnapshot)
	if err != nil {
		t.Fatalf("json.Marshal legacy snapshot: %v", err)
	}
	legacyPath := filepath.Join(fx.dataDir, "stories", fx.story.ID, "saves", snap.ID+".json")
	if err := os.WriteFile(legacyPath, legacyBytes, 0644); err != nil {
		t.Fatalf("WriteFile legacy snapshot: %v", err)
	}

	mutatedChar := *fx.char
	mutatedChar.Name = "Future Mara"
	if err := fx.db.UpdateCharacterFull(&mutatedChar); err != nil {
		t.Fatalf("UpdateCharacterFull: %v", err)
	}

	mutatedWorld := *fx.world
	mutatedWorld.CurrentLocation = "Future Keep"
	mutatedWorld.CurrentTurn = 77
	if err := fx.db.UpdateWorldState(&mutatedWorld); err != nil {
		t.Fatalf("UpdateWorldState: %v", err)
	}

	app := New(fx.cfg, fx.db, nil)
	app.width = 120
	app.height = 40

	cmd, err := app.loadSaveAndResume(fx.story.ID, snap.ID)
	if err != nil {
		t.Fatalf("loadSaveAndResume legacy: %v", err)
	}
	if cmd == nil {
		t.Fatal("loadSaveAndResume legacy returned nil cmd")
	}

	if !strings.Contains(app.View(), "Legacy save loaded: history rollback is partial") {
		t.Fatalf("legacy warning missing from narrative view:\n%s", app.View())
	}

	restoredChar, err := fx.db.GetCharacterByStory(fx.story.ID)
	if err != nil {
		t.Fatalf("GetCharacterByStory: %v", err)
	}
	if restoredChar.Name != fx.char.Name {
		t.Fatalf("restored legacy character name = %q, want %q", restoredChar.Name, fx.char.Name)
	}

	restoredWorld, err := fx.db.GetWorldState(fx.story.ID)
	if err != nil {
		t.Fatalf("GetWorldState: %v", err)
	}
	if restoredWorld.CurrentLocation != fx.world.CurrentLocation {
		t.Fatalf("restored legacy location = %q, want %q", restoredWorld.CurrentLocation, fx.world.CurrentLocation)
	}
	if restoredWorld.CurrentTurn != fx.world.CurrentTurn {
		t.Fatalf("restored legacy turn = %d, want %d", restoredWorld.CurrentTurn, fx.world.CurrentTurn)
	}
}

type appSmokeFixture struct {
	cfg     config.Config
	db      *storage.DB
	dataDir string
	now     time.Time
	story   *storage.Story
	char    *storage.Character
	world   *storage.WorldState
	session *engine.GameSession
}

func newAppSmokeFixture(t *testing.T) appSmokeFixture {
	t.Helper()

	db, err := storage.Open(filepath.Join(t.TempDir(), "app-smoke.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	dataDir := t.TempDir()
	now := time.Date(2026, time.April, 9, 12, 0, 0, 0, time.UTC)

	story := &storage.Story{
		ID:              "story-app-smoke",
		Name:            "App Smoke Story",
		SettingJSON:     `{"factions":["Old Guard"]}`,
		StatsSchemaJSON: `{}`,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	char := &storage.Character{
		ID:               "char-app-smoke",
		StoryID:          story.ID,
		Name:             "Mara",
		Background:       "Scout",
		StatsJSON:        `{"vitals":{"hp":{"current":12,"max":12}}}`,
		TraitsJSON:       `["careful"]`,
		SkillsJSON:       `["tracking"]`,
		InventoryJSON:    `{"items":["rope"]}`,
		KnownRecipesJSON: `[]`,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := db.CreateCharacter(char); err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}

	world := &storage.WorldState{
		ID:                     "world-app-smoke",
		StoryID:                story.ID,
		CurrentLocation:        "Village Square",
		KnownLocationsJSON:     `[{"name":"Village Square","description":"A quiet square.","discovered_turn":1}]`,
		GlobalEventsJSON:       `[]`,
		FactionStandingsJSON:   `{}`,
		StoryHooksJSON:         `[]`,
		WorldReactionsJSON:     `[]`,
		InvestigationBoardJSON: `{"cases":[{"id":"case-1","title":"Who sold you out?","status":"open"}]}`,
		ProjectClocksJSON:      `{"projects":[{"id":"project-1","title":"Train with Lyanna","kind":"training","status":"active","progress":1,"segments":4}]}`,
		FrontsJSON:             `[{"id":"front-1","faction":"Old Guard","title":"The checkpoints are tightening","public_title":"Checkpoints Tighten","public_stakes":"Travel is getting tense.","visibility":"known","segments":4,"progress":2}]`,
		CurrentChapter:         1,
		CurrentTurn:            3,
		UpdatedAt:              now,
	}
	if err := db.CreateWorldState(world); err != nil {
		t.Fatalf("CreateWorldState: %v", err)
	}

	session, err := engine.NewGameSession(db, story.ID, dataDir)
	if err != nil {
		t.Fatalf("NewGameSession: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close(db)
	})

	cfg := config.Default()
	cfg.DataDir = dataDir
	cfg.RAG.Enabled = false
	cfg.Game.TypewriterSpeed = 0

	return appSmokeFixture{
		cfg:     cfg,
		db:      db,
		dataDir: dataDir,
		now:     now,
		story:   story,
		char:    char,
		world:   world,
		session: session,
	}
}
