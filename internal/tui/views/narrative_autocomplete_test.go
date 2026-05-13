package views

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/storage"
)

func TestBuildCommandSuggestionsFiltersTalk(t *testing.T) {
	items := buildCommandSuggestions("ta")
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Value != "/talk " {
		t.Fatalf("items[0].Value = %q, want /talk ", items[0].Value)
	}
}

func TestBuildCommandSuggestionsIncludesAdvanceAndTimeskip(t *testing.T) {
	items := buildCommandSuggestions("adv")
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Value != "/advance " {
		t.Fatalf("items[0].Value = %q, want /advance ", items[0].Value)
	}

	items = buildCommandSuggestions("time")
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Value != "/timeskip " {
		t.Fatalf("items[0].Value = %q, want /timeskip ", items[0].Value)
	}
}

func TestAvailableSlashCommandSpecsHideThoughtsWhenDisabled(t *testing.T) {
	model := NewNarrativeModel(nil, 0, false)
	specs := model.availableSlashCommandSpecs()
	for _, spec := range specs {
		if spec.Name == "thoughts" {
			t.Fatalf("thoughts command should be hidden when disabled: %+v", specs)
		}
	}
}

func TestAvailableSlashCommandSpecsShowThoughtsWhenEnabled(t *testing.T) {
	model := NewNarrativeModel(nil, 0, true)
	specs := model.availableSlashCommandSpecs()
	found := false
	for _, spec := range specs {
		if spec.Name == "thoughts" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("thoughts command missing from available specs: %+v", specs)
	}
}

func TestBuildCommandSuggestionsForSpecsIncludesThoughtsWhenEnabled(t *testing.T) {
	model := NewNarrativeModel(nil, 0, true)
	items := buildCommandSuggestionsForSpecs("tho", model.availableSlashCommandSpecs())
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Value != "/thoughts" {
		t.Fatalf("items[0].Value = %q, want /thoughts", items[0].Value)
	}
}

func TestNearbyTalkNPCSuggestionItemsPreferRecentRoster(t *testing.T) {
	db := openAutocompleteTestDB(t)
	now := time.Now()
	story := &storage.Story{
		ID:        "story-talk",
		Name:      "Talk Story",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	mustCreateNPC(t, db, story.ID, "Lyanna", "scout", 9, now)
	mustCreateNPC(t, db, story.ID, "Brother Alden", "healer", 8, now)
	mustCreateNPC(t, db, story.ID, "Distant Duke", "noble", 2, now)

	items := nearbyTalkNPCSuggestionItems(db, story.ID, 10, 6, "")
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2 recent NPCs", len(items))
	}
	if items[0].Label != "Lyanna" {
		t.Fatalf("items[0].Label = %q, want Lyanna", items[0].Label)
	}
	if items[1].Label != "Brother Alden" {
		t.Fatalf("items[1].Label = %q, want Brother Alden", items[1].Label)
	}
}

func TestBuildTalkIntentSuggestionItemsFiltersIntentPrefix(t *testing.T) {
	items := buildTalkIntentSuggestionItems("Lyanna", "pr")
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].Value != "/talk Lyanna probe" {
		t.Fatalf("items[0].Value = %q, want /talk Lyanna probe", items[0].Value)
	}
	if items[1].Value != "/talk Lyanna promise" {
		t.Fatalf("items[1].Value = %q, want /talk Lyanna promise", items[1].Value)
	}
}

func TestBuildCraftingChoiceItemsKeepsExitLast(t *testing.T) {
	items, exitChoiceID := buildCraftingChoiceItems([]engine.Choice{
		{ID: 1, Text: "Forge a knife"},
		{ID: 2, Text: "Go back"},
	})

	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].Text != "Forge a knife" {
		t.Fatalf("items[0].Text = %q, want Forge a knife", items[0].Text)
	}
	if items[1].Text != "Go back" {
		t.Fatalf("items[1].Text = %q, want Go back", items[1].Text)
	}
	if items[1].ID != exitChoiceID {
		t.Fatalf("items[1].ID = %d, want exitChoiceID %d", items[1].ID, exitChoiceID)
	}
}

func TestParseTalkCommandSupportsOneShotMessage(t *testing.T) {
	model := newTalkTestNarrativeModel(t)

	parsed := model.parseTalkCommand([]string{"Lyanna", "ask", "What", "did", "you", "see?"})
	if parsed.Target != "Lyanna" {
		t.Fatalf("parsed.Target = %q, want Lyanna", parsed.Target)
	}
	if parsed.Intent != "ask" {
		t.Fatalf("parsed.Intent = %q, want ask", parsed.Intent)
	}
	if parsed.Message != "What did you see?" {
		t.Fatalf("parsed.Message = %q, want one-shot prompt", parsed.Message)
	}
}

func TestParseTalkCommandDefaultsIntentForOneShot(t *testing.T) {
	model := newTalkTestNarrativeModel(t)

	parsed := model.parseTalkCommand([]string{"Brother", "Alden", "We", "need", "your", "help"})
	if parsed.Target != "Brother Alden" {
		t.Fatalf("parsed.Target = %q, want Brother Alden", parsed.Target)
	}
	if parsed.Intent != "ask" {
		t.Fatalf("parsed.Intent = %q, want ask", parsed.Intent)
	}
	if parsed.Message != "We need your help" {
		t.Fatalf("parsed.Message = %q, want default-intent one-shot prompt", parsed.Message)
	}
}

func TestNavigateInputHistoryRestoresDraft(t *testing.T) {
	model := NewNarrativeModel(nil, 0, true)
	model.inputFocus = true
	model.recordInputHistory("/talk Lyanna ask")
	model.recordInputHistory("Ask about the lantern")
	model.input.SetValue("/ta")

	if !model.navigateInputHistory(-1) {
		t.Fatal("expected first history navigation to succeed")
	}
	if got := model.input.Value(); got != "Ask about the lantern" {
		t.Fatalf("first recalled value = %q, want latest entry", got)
	}

	if !model.navigateInputHistory(-1) {
		t.Fatal("expected second history navigation to succeed")
	}
	if got := model.input.Value(); got != "/talk Lyanna ask" {
		t.Fatalf("second recalled value = %q, want previous entry", got)
	}

	if !model.navigateInputHistory(1) {
		t.Fatal("expected forward history navigation to succeed")
	}
	if got := model.input.Value(); got != "Ask about the lantern" {
		t.Fatalf("forward value = %q, want latest entry", got)
	}

	if !model.navigateInputHistory(1) {
		t.Fatal("expected restoring draft to succeed")
	}
	if got := model.input.Value(); got != "/ta" {
		t.Fatalf("restored draft = %q, want original draft", got)
	}
}

func TestFormatTalkActionNormalizesIntent(t *testing.T) {
	got := formatTalkAction("Lyanna", "PROMISE", "I will come back.")
	want := "[Talk to Lyanna | intent:promise] I will come back."
	if got != want {
		t.Fatalf("formatTalkAction = %q, want %q", got, want)
	}
}

func openAutocompleteTestDB(t *testing.T) *storage.DB {
	t.Helper()

	db, err := storage.Open(filepath.Join(t.TempDir(), "autocomplete.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func mustCreateNPC(t *testing.T, db *storage.DB, storyID, name, role string, lastSeenTurn int, now time.Time) {
	t.Helper()

	if err := db.CreateNPC(&storage.NPC{
		ID:                name + "-id",
		StoryID:           storyID,
		Name:              name,
		Role:              role,
		PersonalityJSON:   `{}`,
		RelationshipJSON:  `{}`,
		Disposition:       0,
		IsAlive:           true,
		FirstAppearedTurn: lastSeenTurn,
		LastSeenTurn:      lastSeenTurn,
		CreatedAt:         now,
		UpdatedAt:         now,
	}); err != nil {
		t.Fatalf("CreateNPC(%s): %v", name, err)
	}
}

func newTalkTestNarrativeModel(t *testing.T) NarrativeModel {
	t.Helper()

	db := openAutocompleteTestDB(t)
	now := time.Now()
	story := &storage.Story{
		ID:        "story-talk-model",
		Name:      "Talk Model Story",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	world := &storage.WorldState{
		ID:              "world-talk-model",
		StoryID:         story.ID,
		CurrentChapter:  1,
		CurrentTurn:     10,
		CurrentLocation: "Harbor",
		UpdatedAt:       now,
	}
	if err := db.CreateWorldState(world); err != nil {
		t.Fatalf("CreateWorldState: %v", err)
	}

	mustCreateNPC(t, db, story.ID, "Lyanna", "scout", 10, now)
	mustCreateNPC(t, db, story.ID, "Brother Alden", "healer", 9, now)

	session, err := engine.NewGameSession(db, story.ID, t.TempDir())
	if err != nil {
		t.Fatalf("NewGameSession: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close(db)
	})

	narrator := engine.NewNarrator(nil, db, story, &storage.Character{StoryID: story.ID}, world, session, engine.ContextConfig{}, config.GenerationConfig{}, config.ASCIIArtConfig{}, t.TempDir(), 5)
	return NewNarrativeModel(narrator, 0, true)
}
