package engine

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/game/contracts"
	"github.com/crimsab/oneday/internal/storage"
)

type combatTestProvider struct{}

func (combatTestProvider) Name() string { return "combat-test" }
func (combatTestProvider) Complete(context.Context, ai.Request) (ai.Response, error) {
	return ai.Response{Content: `{"narrative":"The exchange continues.","choices":[{"id":1,"text":"Press on"}]}`, Model: "combat-test"}, nil
}

func TestCombatRNGStreamsAreIsolated(t *testing.T) {
	first := NewDefaultRNGService()
	second := NewDefaultRNGService()
	if first == second {
		t.Fatal("combat RNG streams must not share a singleton")
	}
	second.Roll("other-combat", 20)
	snapshot := first.snapshot()
	first.Roll("failed-combat", 20)
	first.restore(snapshot)
	if len(first.RollLog()) != 0 || len(second.RollLog()) != 1 {
		t.Fatalf("restoring one stream affected another: first=%+v second=%+v", first.RollLog(), second.RollLog())
	}
}

func TestFailedCombatTurnRestoresStateCharacterAndRNG(t *testing.T) {
	root := t.TempDir()
	db, err := storage.Open(filepath.Join(root, "combat-retry.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	createSessionTestStory(t, db, "story-combat-retry", 0)
	session, err := NewGameSession(db, "story-combat-retry", filepath.Join(root, "data"))
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close(db)
	story, _ := db.GetStory("story-combat-retry")
	character, _ := db.GetCharacterByStory(story.ID)
	character.StatsJSON = `{"vitals":{"hp":{"current":100,"max":100}},"attributes":{"str":{"value":0}}}`
	world, _ := db.GetWorldState(story.ID)
	timeline, _ := db.GetActiveTimeline(story.ID)
	router, _ := ai.NewRouter([]ai.Provider{combatTestProvider{}})
	narrator := &Narrator{router: router, db: db, story: story, character: character, world: world, session: session}
	combat := &CombatEngine{
		state:            &CombatState{Enemy: EnemyStats{Name: "Warden", HP: 999, MaxHP: 999, Attack: 0, Defense: 30, Behavior: BehaviorAggressive}, PlayerHP: 100, PlayerMaxHP: 100, Turn: 1, Phase: "player_turn"},
		narrator:         narrator,
		session:          session,
		rng:              NewRNGService(4242),
		expectedBranchID: timeline.Branch.ID,
		expectedHeadID:   timeline.Commit.ID,
		expectedRevision: story.Revision,
	}
	originalState := *combat.state
	originalStats := character.StatsJSON
	if _, err := db.Conn().Exec(`CREATE TRIGGER reject_combat_turn_revision BEFORE UPDATE OF revision ON stories BEGIN SELECT RAISE(ABORT, 'revision failure'); END`); err != nil {
		t.Fatal(err)
	}

	if _, err := combat.PlayerAction(context.Background(), "attack"); err == nil {
		t.Fatal("expected failed combat turn")
	}
	if !reflect.DeepEqual(*combat.state, originalState) {
		t.Fatalf("combat state changed after rollback: before=%+v after=%+v", originalState, *combat.state)
	}
	if character.StatsJSON != originalStats {
		t.Fatalf("character changed after rollback: %s", character.StatsJSON)
	}
	if len(combat.rng.RollLog()) != 0 {
		t.Fatalf("rng log was not restored: %+v", combat.rng.RollLog())
	}
	if _, err := db.Conn().Exec(`DROP TRIGGER reject_combat_turn_revision`); err != nil {
		t.Fatal(err)
	}
	expectedRNG := NewRNGService(4242)
	expectedPlayer := expectedRNG.Roll("combat.player_attack", 20)
	expectedEnemy := expectedRNG.Roll("combat.enemy_attack", 20)
	result, err := combat.PlayerAction(context.Background(), "attack")
	if err != nil {
		t.Fatalf("retry combat turn: %v", err)
	}
	if len(result.RollLog) != 2 || result.RollLog[0].Raw != expectedPlayer.Raw || result.RollLog[1].Raw != expectedEnemy.Raw {
		t.Fatalf("retry rolls=%+v, want raw %d,%d", result.RollLog, expectedPlayer.Raw, expectedEnemy.Raw)
	}
}

func TestCombatFinalizationRollsBackAllCanonicalWrites(t *testing.T) {
	root := t.TempDir()
	db, err := storage.Open(filepath.Join(root, "combat-finalize.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	createSessionTestStory(t, db, "story-combat-finalize", 0)
	session, err := NewGameSession(db, "story-combat-finalize", filepath.Join(root, "data"))
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close(db)
	story, _ := db.GetStory("story-combat-finalize")
	character, _ := db.GetCharacterByStory(story.ID)
	world, _ := db.GetWorldState(story.ID)
	timeline, _ := db.GetActiveTimeline(story.ID)
	originalStats := character.StatsJSON
	character.StatsJSON = `{"vitals":{"hp":{"current":1,"max":20}}}`
	narrator := &Narrator{db: db, story: story, character: character, world: world, session: session}
	combat := &CombatEngine{
		state:            &CombatState{Enemy: EnemyStats{Name: "Warden", MaxHP: 12}, PlayerHP: 1, PlayerMaxHP: 20, Turn: 2, Resolved: true, Victory: true, Summary: "The warden falls."},
		narrator:         narrator,
		session:          session,
		expectedBranchID: timeline.Branch.ID,
		expectedHeadID:   timeline.Commit.ID,
		expectedRevision: story.Revision,
	}
	instance := NewOrdinaryActionChallenge(story.ID, story.ActiveBranchID, 0, "combat-finalize", DefaultOutcomePolicy("combat", "balanced"))
	resolution := contracts.ChallengeResolution{ProtocolVersion: contracts.ChallengeProtocolVersion, InstanceID: instance.ID, Outcome: contracts.OutcomeEnvelope{Version: 1, Degree: contracts.OutcomeFullSuccess, Seed: instance.Seed}}
	result := &CombatTurnResult{CombatOver: true, Victory: true, ChallengeInstance: &instance, ChallengeResolution: &resolution}
	if _, err := db.Conn().Exec(`CREATE TRIGGER reject_combat_log BEFORE INSERT ON combat_log BEGIN SELECT RAISE(ABORT, 'combat log failure'); END`); err != nil {
		t.Fatal(err)
	}

	if _, _, err := combat.commitCombatOutcome(result); err == nil {
		t.Fatal("expected combat finalization failure")
	}
	if _, err := db.GetChallengeRun(instance.ID); err == nil {
		t.Fatal("challenge survived rolled-back finalization")
	}
	persistedCharacter, _ := db.GetCharacterByStory(story.ID)
	if persistedCharacter.StatsJSON != originalStats {
		t.Fatalf("character stats changed despite rollback: %s", persistedCharacter.StatsJSON)
	}
	for table, want := range map[string]int{"combat_log": 0, "chat_messages": 0} {
		var count int
		if err := db.Conn().QueryRow("SELECT COUNT(*) FROM "+table+" WHERE story_id=?", story.ID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Fatalf("%s rows=%d, want %d", table, count, want)
		}
	}
	revision, _ := db.GetStoryRevision(story.ID)
	if revision != 0 {
		t.Fatalf("revision changed despite rollback: %d", revision)
	}
	if _, err := db.Conn().Exec(`DROP TRIGGER reject_combat_log`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`CREATE TRIGGER reject_combat_revision BEFORE UPDATE OF revision ON stories BEGIN SELECT RAISE(ABORT, 'revision failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, _, err := combat.commitCombatOutcome(result); err == nil {
		t.Fatal("expected final revision failure")
	}
	for _, table := range []string{"combat_log", "chat_messages", "challenge_runs"} {
		var count int
		if err := db.Conn().QueryRow("SELECT COUNT(*) FROM "+table+" WHERE story_id=?", story.ID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s survived failed revision bump", table)
		}
	}
	if _, err := db.Conn().Exec(`DROP TRIGGER reject_combat_revision`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`INSERT INTO story_branches (id,story_id,name,fork_commit_id,head_commit_id) VALUES ('combat-other','story-combat-finalize','other',?,?)`, timeline.Commit.ID, timeline.Commit.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn().Exec(`UPDATE stories SET active_branch_id='combat-other' WHERE id=?`, story.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := combat.commitCombatOutcome(result); err == nil {
		t.Fatal("expected stale combat branch failure")
	}
	if _, err := db.Conn().Exec(`UPDATE stories SET active_branch_id=? WHERE id=?`, timeline.Branch.ID, story.ID); err != nil {
		t.Fatal(err)
	}
	committedRevision, entry, err := combat.commitCombatOutcome(result)
	if err != nil {
		t.Fatalf("retry combat finalization: %v", err)
	}
	if committedRevision != 1 || entry == nil {
		t.Fatalf("retry revision=%d entry=%v", committedRevision, entry)
	}
	for table, want := range map[string]int{"combat_log": 1, "chat_messages": 1, "challenge_runs": 1} {
		var count int
		if err := db.Conn().QueryRow("SELECT COUNT(*) FROM "+table+" WHERE story_id=?", story.ID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Fatalf("%s rows after retry=%d, want %d", table, count, want)
		}
	}
	var branchID, sourceCommitID string
	if err := db.Conn().QueryRow(`SELECT branch_id,source_commit_id FROM chat_messages WHERE story_id=? AND message_type='combat_summary'`, story.ID).Scan(&branchID, &sourceCommitID); err != nil {
		t.Fatal(err)
	}
	if branchID != timeline.Branch.ID || sourceCommitID != timeline.Commit.ID {
		t.Fatalf("summary lineage branch=%q commit=%q", branchID, sourceCommitID)
	}
}

func TestCommitTurnAdvancesImmutableTimelineAtomically(t *testing.T) {
	root := t.TempDir()
	db, err := storage.Open(filepath.Join(root, "oneday-test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	createSessionTestStory(t, db, "story-commit-dag", 0)
	session, err := NewGameSession(db, "story-commit-dag", filepath.Join(root, "data"))
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close(db)
	before, err := db.GetActiveTimeline("story-commit-dag")
	if err != nil {
		t.Fatal(err)
	}
	char, _ := db.GetCharacterByStory("story-commit-dag")
	world, _ := db.GetWorldState("story-commit-dag")
	world.CurrentTurn = 1
	world.CurrentLocation = "Crossroads"
	run := &storage.GenerationRun{Stage: "narrator", PromptHash: "sha256:test", StoryID: "story-commit-dag"}
	if err := db.StartGenerationRun(run); err != nil {
		t.Fatal(err)
	}
	if err := db.FinishGenerationRun(run.ID, storage.GenerationCompletion{Status: storage.GenerationStatusSucceeded}); err != nil {
		t.Fatal(err)
	}
	err = session.CommitTurn(db, char, world, ChatEntry{Input: &ChatInput{Type: "choice", Text: "Go north"}, Output: &ChatOutput{Narrative: "You reach the crossroads."}, MessageType: "narrative", GenerationRunID: run.ID, GenerationTraceID: run.TraceID})
	if err != nil {
		t.Fatalf("CommitTurn: %v", err)
	}
	after, err := db.GetActiveTimeline("story-commit-dag")
	if err != nil {
		t.Fatal(err)
	}
	if after.Commit.ParentCommitID != before.Commit.ID || after.Commit.CanonicalTurn != 1 || after.Commit.PayloadHash == "" {
		t.Fatalf("invalid committed head: before=%#v after=%#v", before, after)
	}
	var messages, events int
	if err := db.Conn().QueryRow(`SELECT COUNT(*) FROM chat_messages WHERE story_id=? AND branch_id=? AND source_commit_id=?`, "story-commit-dag", after.Branch.ID, after.Commit.ID).Scan(&messages); err != nil {
		t.Fatal(err)
	}
	if err := db.Conn().QueryRow(`SELECT COUNT(*) FROM canonical_events WHERE commit_id=?`, after.Commit.ID).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if messages != 2 || events != 1 {
		t.Fatalf("lineage messages=%d events=%d", messages, events)
	}
	var authoredMessageID int64
	var generationMetadata string
	if err := db.Conn().QueryRow(`SELECT message_id FROM generation_runs WHERE id=?`, run.ID).Scan(&authoredMessageID); err != nil {
		t.Fatal(err)
	}
	if err := db.Conn().QueryRow(`SELECT metadata_json FROM chat_messages WHERE id=?`, authoredMessageID).Scan(&generationMetadata); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(generationMetadata, run.ID) || !strings.Contains(generationMetadata, run.TraceID) {
		t.Fatalf("assistant generation metadata=%s", generationMetadata)
	}

	failedWorld := *world
	failedWorld.CurrentTurn = 2
	err = session.CommitTurnWithSideEffects(db, char, &failedWorld, ChatEntry{Output: &ChatOutput{Narrative: "must rollback"}}, func(*sql.Tx) error { return errors.New("forced side-effect failure") })
	if err == nil {
		t.Fatal("expected failed commit")
	}
	stable, _ := db.GetActiveTimeline("story-commit-dag")
	persistedWorld, _ := db.GetWorldState("story-commit-dag")
	if stable.Commit.ID != after.Commit.ID || persistedWorld.CurrentTurn != 1 {
		t.Fatalf("failed commit mutated head/state: head=%s turn=%d", stable.Commit.ID, persistedWorld.CurrentTurn)
	}
}

func TestCommitTurnPersistsAutomaticMiniGameOnNewHead(t *testing.T) {
	root := t.TempDir()
	db, err := storage.Open(filepath.Join(root, "oneday-minigame.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	createSessionTestStory(t, db, "story-auto-mini", 0)
	session, err := NewGameSession(db, "story-auto-mini", filepath.Join(root, "data"))
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close(db)
	head, err := db.GetActiveTimeline("story-auto-mini")
	if err != nil {
		t.Fatal(err)
	}
	instance := NewMiniGameInstance("mini-auto-test", "story-auto-mini", head.Branch.ID, 0, 42, DefaultMiniGameDefinition(MiniGameDeduction))
	if err := NewMiniGameHost().Start(&instance); err != nil {
		t.Fatal(err)
	}
	char, _ := db.GetCharacterByStory("story-auto-mini")
	world, _ := db.GetWorldState("story-auto-mini")
	world.CurrentTurn = 1
	if err := session.CommitTurn(db, char, world, ChatEntry{Output: &ChatOutput{Narrative: "The witness contradicts herself.", AutomaticMiniGame: &instance}}); err != nil {
		t.Fatal(err)
	}
	after, err := db.GetActiveTimeline("story-auto-mini")
	if err != nil {
		t.Fatal(err)
	}
	record, err := db.GetActiveMiniGameInstance("story-auto-mini")
	if err != nil {
		t.Fatal(err)
	}
	if record.ID != instance.ID || record.BranchID != after.Branch.ID || record.SourceCommitID != after.Commit.ID || record.Phase != string(MiniGameActive) {
		t.Fatalf("automatic minigame lineage=%+v head=%+v", record, after)
	}
}

func createSessionTestStory(t *testing.T, db *storage.DB, storyID string, currentTurn int) {
	t.Helper()

	now := time.Now()
	story := &storage.Story{
		ID:              storyID,
		Name:            "Session Tale",
		SettingJSON:     `{}`,
		StatsSchemaJSON: `{}`,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	char := &storage.Character{
		ID:               "char-" + storyID,
		StoryID:          storyID,
		Name:             "Hero",
		Background:       "Test",
		StatsJSON:        `{"vitals":{"hp":{"current":10,"max":10}}}`,
		TraitsJSON:       `[]`,
		SkillsJSON:       `{}`,
		InventoryJSON:    `[]`,
		KnownRecipesJSON: `[]`,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := db.CreateCharacter(char); err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}

	world := &storage.WorldState{
		ID:                   "world-" + storyID,
		StoryID:              storyID,
		CurrentLocation:      "Village",
		KnownLocationsJSON:   `["Village"]`,
		GlobalEventsJSON:     `[]`,
		FactionStandingsJSON: `{}`,
		StoryHooksJSON:       `[]`,
		WorldReactionsJSON:   `[]`,
		PlayerGuidanceJSON:   `[]`,
		CurrentChapter:       1,
		CurrentTurn:          currentTurn,
		UpdatedAt:            now,
	}
	if err := db.CreateWorldState(world); err != nil {
		t.Fatalf("CreateWorldState: %v", err)
	}
}

func TestAppendHistoryEntryDoesNotAdvanceTurn(t *testing.T) {
	root := t.TempDir()
	db, err := storage.Open(filepath.Join(root, "oneday-test.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	defer db.Close()

	story := &storage.Story{
		ID:              "story-session",
		Name:            "Session Tale",
		SettingJSON:     `{}`,
		StatsSchemaJSON: `{}`,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	session, err := NewGameSession(db, story.ID, filepath.Join(root, "data"))
	if err != nil {
		t.Fatalf("NewGameSession: %v", err)
	}
	defer session.Close(db)

	session.SetTurn(3)
	if err := session.AppendHistoryEntry(db, ChatEntry{
		Turn:        3,
		Timestamp:   time.Now(),
		MessageType: "combat_summary",
		Output: &ChatOutput{
			Narrative: "Victory summary",
			Mood:      "neutral",
		},
	}); err != nil {
		t.Fatalf("AppendHistoryEntry: %v", err)
	}

	if got := session.Turn(); got != 3 {
		t.Fatalf("session turn = %d, want 3", got)
	}

	msgs, err := db.GetSessionMessages(session.SessionID())
	if err != nil {
		t.Fatalf("GetSessionMessages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("message count = %d, want 1", len(msgs))
	}
	if msgs[0].MessageType != "combat_summary" {
		t.Fatalf("message_type = %q, want combat_summary", msgs[0].MessageType)
	}
	if msgs[0].Turn != 3 {
		t.Fatalf("message turn = %d, want 3", msgs[0].Turn)
	}
}

func TestAppendSubTurnPersistsRollLogToJSONL(t *testing.T) {
	root := t.TempDir()
	db, err := storage.Open(filepath.Join(root, "oneday-test.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	defer db.Close()

	createSessionTestStory(t, db, "story-roll-log", 0)
	session, err := NewGameSession(db, "story-roll-log", filepath.Join(root, "data"))
	if err != nil {
		t.Fatalf("NewGameSession: %v", err)
	}
	defer session.Close(db)

	subID, err := session.OpenSubSession("combat")
	if err != nil {
		t.Fatalf("OpenSubSession: %v", err)
	}
	if err := session.AppendSubTurn(subID, ChatEntry{
		Timestamp:   time.Now(),
		MessageType: "combat",
		Output: &ChatOutput{
			Narrative: "The blade hits.",
			RollLog: []RollRecord{
				{Source: "combat.player_attack", Sides: 20, Raw: 17, Total: 21, Target: 8, Outcome: "damage:13", Seed: 42},
			},
		},
	}); err != nil {
		t.Fatalf("AppendSubTurn: %v", err)
	}
	if err := session.CloseSubSession(subID); err != nil {
		t.Fatalf("CloseSubSession: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "data", "stories", "story-roll-log", "sessions", session.SessionID(), subID+".jsonl"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, `"roll_log"`) || !strings.Contains(text, `"combat.player_attack"`) {
		t.Fatalf("sub-session JSONL = %s, want roll_log with combat label", text)
	}
}

func TestNewGameSessionUsesCanonicalTurnInsteadOfJSONLMirror(t *testing.T) {
	root := t.TempDir()
	db, err := storage.Open(filepath.Join(root, "oneday-test.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	defer db.Close()

	createSessionTestStory(t, db, "story-canonical-cursor", 2)
	sessionRow := &storage.Session{
		ID:        "stale-jsonl",
		StoryID:   "story-canonical-cursor",
		StartedAt: time.Now(),
	}
	if err := db.CreateSession(sessionRow); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	sessionDir := filepath.Join(root, "data", "stories", "story-canonical-cursor", "sessions", "stale-jsonl")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "main.jsonl"), []byte("{\"turn\":99}\n{\"turn\":100}\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	session, err := NewGameSession(db, "story-canonical-cursor", filepath.Join(root, "data"))
	if err != nil {
		t.Fatalf("NewGameSession: %v", err)
	}
	defer session.Close(db)

	if got := session.Turn(); got != 2 {
		t.Fatalf("session turn = %d, want 2 from canonical DB state", got)
	}
}

func TestCloseMirrorsLeavesDBSessionActive(t *testing.T) {
	root := t.TempDir()
	db, err := storage.Open(filepath.Join(root, "oneday-test.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	defer db.Close()

	createSessionTestStory(t, db, "story-close-mirrors", 0)
	session, err := NewGameSession(db, "story-close-mirrors", filepath.Join(root, "data"))
	if err != nil {
		t.Fatalf("NewGameSession: %v", err)
	}
	sessionID := session.SessionID()
	if err := session.CloseMirrors(); err != nil {
		t.Fatalf("CloseMirrors: %v", err)
	}

	active, err := db.GetActiveSession("story-close-mirrors")
	if err != nil {
		t.Fatalf("GetActiveSession: %v", err)
	}
	if active == nil || active.ID != sessionID {
		t.Fatalf("active session = %#v, want %s", active, sessionID)
	}
}

func TestNewGameSessionIgnoresMetaOnlyHistoryWhenRecoveringTurnCursor(t *testing.T) {
	root := t.TempDir()
	db, err := storage.Open(filepath.Join(root, "oneday-test.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	defer db.Close()

	createSessionTestStory(t, db, "story-meta-cursor", 2)
	now := time.Now()
	sessionRow := &storage.Session{
		ID:        "session-meta",
		StoryID:   "story-meta-cursor",
		StartedAt: now,
	}
	if err := db.CreateSession(sessionRow); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if err := db.AppendChatMessage(&storage.ChatMessage{
		SessionID:    sessionRow.ID,
		StoryID:      "story-meta-cursor",
		Turn:         8,
		Role:         "assistant",
		Content:      "Meta answer",
		MessageType:  "narrator",
		MetadataJSON: "{}",
		CreatedAt:    now,
	}); err != nil {
		t.Fatalf("AppendChatMessage narrator: %v", err)
	}
	if err := db.AppendChatMessage(&storage.ChatMessage{
		SessionID:    sessionRow.ID,
		StoryID:      "story-meta-cursor",
		Turn:         9,
		Role:         "assistant",
		Content:      "Combat summary",
		MessageType:  "combat_summary",
		MetadataJSON: "{}",
		CreatedAt:    now,
	}); err != nil {
		t.Fatalf("AppendChatMessage combat_summary: %v", err)
	}

	session, err := NewGameSession(db, "story-meta-cursor", filepath.Join(root, "data"))
	if err != nil {
		t.Fatalf("NewGameSession: %v", err)
	}
	defer session.Close(db)

	if got := session.Turn(); got != 2 {
		t.Fatalf("session turn = %d, want 2 with meta-only history ignored", got)
	}
}

func TestAppendTurnCanonicalDBFailureDoesNotAdvanceTurn(t *testing.T) {
	root := t.TempDir()
	db, err := storage.Open(filepath.Join(root, "oneday-test.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}

	createSessionTestStory(t, db, "story-db-failure", 0)

	session, err := NewGameSession(db, "story-db-failure", filepath.Join(root, "data"))
	if err != nil {
		t.Fatalf("NewGameSession: %v", err)
	}
	defer session.Close(nil)

	if err := db.Close(); err != nil {
		t.Fatalf("db.Close: %v", err)
	}

	err = session.AppendTurn(db, ChatEntry{
		Timestamp: time.Now(),
		Input: &ChatInput{
			Type: "free_action",
			Text: "Inspect the room",
		},
		Output: &ChatOutput{
			Narrative: "You inspect the room.",
		},
	})
	if err == nil {
		t.Fatal("AppendTurn error = nil, want canonical DB failure")
	}
	if got := session.Turn(); got != 0 {
		t.Fatalf("session turn = %d, want 0 after canonical DB failure", got)
	}
}

func TestAppendTurnMirrorFailureStillAdvancesCanonicalTurn(t *testing.T) {
	root := t.TempDir()
	db, err := storage.Open(filepath.Join(root, "oneday-test.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	defer db.Close()

	createSessionTestStory(t, db, "story-mirror-failure", 0)

	session, err := NewGameSession(db, "story-mirror-failure", filepath.Join(root, "data"))
	if err != nil {
		t.Fatalf("NewGameSession: %v", err)
	}
	defer session.Close(db)

	if err := session.jsonlFile.Close(); err != nil {
		t.Fatalf("closing jsonl mirror: %v", err)
	}

	err = session.AppendTurn(db, ChatEntry{
		Timestamp: time.Now(),
		Input: &ChatInput{
			Type: "free_action",
			Text: "Open the gate",
		},
		Output: &ChatOutput{
			Narrative: "The gate groans open.",
			Location:  "Village",
		},
	})
	if err == nil {
		t.Fatal("AppendTurn error = nil, want mirror sync error")
	}
	if !IsMirrorSyncError(err) {
		t.Fatalf("AppendTurn error = %v, want mirror sync error", err)
	}
	if got := session.Turn(); got != 1 {
		t.Fatalf("session turn = %d, want 1 after canonical DB commit", got)
	}

	msgs, err := db.GetSessionMessages(session.SessionID())
	if err != nil {
		t.Fatalf("GetSessionMessages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("message count = %d, want 2 canonical DB messages", len(msgs))
	}
}
