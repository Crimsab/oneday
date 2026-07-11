package engine

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/storage"
)

func TestLoadGameRejectsMissingVersionedSnapshotWithoutMutation(t *testing.T) {
	db, dataDir := newSaveTestDB(t)
	story, char, world, sessionID := createMinimalSaveState(t, db)

	snap, err := SaveGame(db, dataDir, story, char, world, sessionID, "missing-file")
	if err != nil {
		t.Fatalf("SaveGame: %v", err)
	}
	if err := os.Remove(saveFilePath(dataDir, story.ID, snap.ID)); err != nil {
		t.Fatalf("remove snapshot file: %v", err)
	}

	char.InventoryJSON = `{"items":["current-token"]}`
	world.CurrentLocation = "Current Harbor"
	if err := db.UpdateCharacterFull(char); err != nil {
		t.Fatalf("UpdateCharacterFull: %v", err)
	}
	if err := db.UpdateWorldState(world); err != nil {
		t.Fatalf("UpdateWorldState: %v", err)
	}

	_, err = LoadGame(db, dataDir, snap.ID)
	var loadErr *SnapshotLoadError
	if !errors.As(err, &loadErr) {
		t.Fatalf("LoadGame error = %v, want SnapshotLoadError", err)
	}
	if loadErr.State != storage.SnapshotStateIncomplete {
		t.Fatalf("load error state = %q, want %q", loadErr.State, storage.SnapshotStateIncomplete)
	}

	restoredChar, _ := db.GetCharacterByStory(story.ID)
	restoredWorld, _ := db.GetWorldState(story.ID)
	if !strings.Contains(restoredChar.InventoryJSON, "current-token") || restoredWorld.CurrentLocation != "Current Harbor" {
		t.Fatalf("failed load mutated current state: inventory=%q location=%q", restoredChar.InventoryJSON, restoredWorld.CurrentLocation)
	}
}

func TestLoadGameRejectsTamperedSnapshotWithoutMutation(t *testing.T) {
	db, dataDir := newSaveTestDB(t)
	story, char, world, sessionID := createMinimalSaveState(t, db)

	snap, err := SaveGame(db, dataDir, story, char, world, sessionID, "tampered")
	if err != nil {
		t.Fatalf("SaveGame: %v", err)
	}
	snapPath := saveFilePath(dataDir, story.ID, snap.ID)
	bytes, err := os.ReadFile(snapPath)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	var tampered storage.SaveSnapshot
	if err := json.Unmarshal(bytes, &tampered); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	tampered.WorldStateJSON = strings.Replace(tampered.WorldStateJSON, "Old Harbor", "Future Palace", 1)
	bytes, err = json.Marshal(&tampered)
	if err != nil {
		t.Fatalf("marshal tampered snapshot: %v", err)
	}
	if err := os.WriteFile(snapPath, bytes, 0644); err != nil {
		t.Fatalf("write tampered snapshot: %v", err)
	}

	char.InventoryJSON = `{"items":["current-token"]}`
	if err := db.UpdateCharacterFull(char); err != nil {
		t.Fatalf("UpdateCharacterFull: %v", err)
	}

	_, err = LoadGame(db, dataDir, snap.ID)
	var loadErr *SnapshotLoadError
	if !errors.As(err, &loadErr) {
		t.Fatalf("LoadGame error = %v, want SnapshotLoadError", err)
	}
	if loadErr.State != storage.SnapshotStateCorrupt {
		t.Fatalf("load error state = %q, want %q", loadErr.State, storage.SnapshotStateCorrupt)
	}
	restoredChar, _ := db.GetCharacterByStory(story.ID)
	if !strings.Contains(restoredChar.InventoryJSON, "current-token") {
		t.Fatalf("failed load mutated current inventory to %q", restoredChar.InventoryJSON)
	}
}

func TestLoadGameSessionStagingFailurePreservesCurrentState(t *testing.T) {
	db, dataDir := newSaveTestDB(t)
	story, char, world, sessionID := createMinimalSaveState(t, db)

	snap, err := SaveGame(db, dataDir, story, char, world, sessionID, "unsafe-session-path")
	if err != nil {
		t.Fatalf("SaveGame: %v", err)
	}
	snap.SessionFiles["../escape.jsonl"] = "must-not-escape\n"
	if err := snap.SealFullRollback(); err != nil {
		t.Fatalf("SealFullRollback unsafe fixture: %v", err)
	}
	snapBytes, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("marshal unsafe snapshot: %v", err)
	}
	if err := os.WriteFile(saveFilePath(dataDir, story.ID, snap.ID), snapBytes, 0644); err != nil {
		t.Fatalf("write unsafe snapshot: %v", err)
	}

	char.InventoryJSON = `{"items":["current-token"]}`
	world.CurrentLocation = "Current Harbor"
	if err := db.UpdateCharacterFull(char); err != nil {
		t.Fatalf("UpdateCharacterFull: %v", err)
	}
	if err := db.UpdateWorldState(world); err != nil {
		t.Fatalf("UpdateWorldState: %v", err)
	}

	if _, err := LoadGame(db, dataDir, snap.ID); err == nil || !strings.Contains(err.Error(), "invalid session file path") {
		t.Fatalf("LoadGame error = %v, want invalid session file path", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "stories", story.ID, "escape.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("unsafe session path escaped restore root: %v", err)
	}
	restoredChar, _ := db.GetCharacterByStory(story.ID)
	restoredWorld, _ := db.GetWorldState(story.ID)
	if !strings.Contains(restoredChar.InventoryJSON, "current-token") || restoredWorld.CurrentLocation != "Current Harbor" {
		t.Fatalf("staging failure mutated current state: inventory=%q location=%q", restoredChar.InventoryJSON, restoredWorld.CurrentLocation)
	}
}

func TestSessionRestoreStageRollbackRestoresPreviousFiles(t *testing.T) {
	dataDir := t.TempDir()
	root := filepath.Join(dataDir, "stories", "story-1", "sessions")
	currentPath := filepath.Join(root, "current", "main.jsonl")
	if err := os.MkdirAll(filepath.Dir(currentPath), 0755); err != nil {
		t.Fatalf("MkdirAll current session: %v", err)
	}
	if err := os.WriteFile(currentPath, []byte("current-turn\n"), 0644); err != nil {
		t.Fatalf("WriteFile current session: %v", err)
	}

	stage, err := prepareSessionRestore(dataDir, "story-1", map[string]string{
		"saved/main.jsonl": "saved-turn\n",
	})
	if err != nil {
		t.Fatalf("prepareSessionRestore: %v", err)
	}
	defer stage.cleanup()
	if err := stage.activate(); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if bytes, err := os.ReadFile(filepath.Join(root, "saved", "main.jsonl")); err != nil || string(bytes) != "saved-turn\n" {
		t.Fatalf("active staged file = %q, err=%v", string(bytes), err)
	}
	if err := stage.rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if bytes, err := os.ReadFile(currentPath); err != nil || string(bytes) != "current-turn\n" {
		t.Fatalf("compensated current file = %q, err=%v", string(bytes), err)
	}
	if _, err := os.Stat(filepath.Join(root, "saved", "main.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("staged file survived compensation: %v", err)
	}
}

func TestSaveGameAndLoadGameRestoreCanonicalStoryState(t *testing.T) {
	db, dataDir := newSaveTestDB(t)
	baseTime := time.Date(2026, time.April, 9, 10, 0, 0, 0, time.UTC)

	story := &storage.Story{
		ID:              "story-1",
		Name:            "Rollback Tale",
		SettingJSON:     `{"factions":["Old Guard"]}`,
		StatsSchemaJSON: `{}`,
		CreatedAt:       baseTime,
		UpdatedAt:       baseTime,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	char := &storage.Character{
		ID:               "char-1",
		StoryID:          story.ID,
		Name:             "Mara",
		Background:       "Scout",
		StatsJSON:        `{"vitals":{"hp":{"current":12,"max":12}}}`,
		TraitsJSON:       `["careful"]`,
		SkillsJSON:       `["tracking"]`,
		InventoryJSON:    `{"items":["rope"]}`,
		KnownRecipesJSON: `[]`,
		CreatedAt:        baseTime,
		UpdatedAt:        baseTime,
	}
	if err := db.CreateCharacter(char); err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}

	world := &storage.WorldState{
		ID:                     "world-1",
		StoryID:                story.ID,
		CurrentLocation:        "Village",
		KnownLocationsJSON:     `["Village"]`,
		GlobalEventsJSON:       `[]`,
		FactionStandingsJSON:   `{}`,
		StoryHooksJSON:         `[{"id":"hook-1","kind":"mystery","title":"Who sold you out?","status":"active"}]`,
		WorldReactionsJSON:     `[{"id":"react-1","kind":"rumor","title":"The guard remembers you","status":"active"}]`,
		InvestigationBoardJSON: `{"cases":[{"id":"case-1","title":"Who sold you out?","status":"open","clues":[{"id":"clue-1","label":"A missing seal","status":"known"}],"hidden_truths":[{"id":"truth-1","label":"The guard captain was paid","status":"hidden"}]}]}`,
		ProjectClocksJSON:      `{"projects":[{"id":"project-1","title":"Train with Lyanna","kind":"training","status":"active","progress":1,"segments":4,"owner":"Mara","rewards":[{"kind":"skill","label":"Footwork +1"}]}]}`,
		FrontsJSON:             `[{"id":"front-1","faction":"Old Guard","title":"The Old Guard is tightening checkpoints","public_title":"Checkpoints Tighten","public_stakes":"Travel is getting tense.","visibility":"known","segments":4,"progress":2}]`,
		CurrentChapter:         1,
		CurrentTurn:            1,
		UpdatedAt:              baseTime,
	}
	if err := db.CreateWorldState(world); err != nil {
		t.Fatalf("CreateWorldState: %v", err)
	}

	session := &storage.Session{
		ID:        "sess-1",
		StoryID:   story.ID,
		StartedAt: baseTime,
	}
	if err := db.CreateSession(session); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := db.AppendChatMessage(&storage.ChatMessage{
		SessionID:    session.ID,
		StoryID:      story.ID,
		Turn:         1,
		Role:         "user",
		Content:      "Look around",
		MessageType:  "narrative",
		MetadataJSON: "{}",
		CreatedAt:    baseTime,
	}); err != nil {
		t.Fatalf("AppendChatMessage user: %v", err)
	}
	if err := db.AppendChatMessage(&storage.ChatMessage{
		SessionID:    session.ID,
		StoryID:      story.ID,
		Turn:         1,
		Role:         "assistant",
		Content:      "The square is calm.",
		MessageType:  "narrative",
		MetadataJSON: "{}",
		CreatedAt:    baseTime.Add(time.Second),
	}); err != nil {
		t.Fatalf("AppendChatMessage assistant: %v", err)
	}

	savedNPC := &storage.NPC{
		ID:                 "npc-1",
		StoryID:            story.ID,
		Name:               "Old Guard",
		Role:               "watcher",
		Appearance:         "A weathered blue cloak and a brass gate badge",
		PersonalityJSON:    `{}`,
		PrivateThoughts:    `["Mara notices too much"]`,
		RelationshipJSON:   `{"trust":12,"fear":-3,"respect":8}`,
		NemesisJSON:        `{"status":"active","rivalry_score":6,"escalation_tier":2,"threat_posture":"watching"}`,
		DiscoveryJSON:      `{"stage":"identified","confidence":0.85,"aliases":["Gate Watcher"],"field_facts":{"role":{"value":"watcher","source":"observed","turn":1}}}`,
		NotesOnProtagonist: `["Carries a rope"]`,
		Desires:            `["Protect the gate"]`,
		Disposition:        1,
		IsAlive:            true,
		FirstAppearedTurn:  1,
		LastSeenTurn:       1,
		CanHelp:            true,
		CreatedAt:          baseTime,
		UpdatedAt:          baseTime,
	}
	if err := db.CreateNPC(savedNPC); err != nil {
		t.Fatalf("CreateNPC: %v", err)
	}

	achievement := &storage.Achievement{
		StoryID:     story.ID,
		Name:        "First Step",
		Description: "Reached the village",
		Category:    "story",
		Rarity:      "common",
		Context:     "intro",
	}
	if err := db.CreateAchievement(achievement); err != nil {
		t.Fatalf("CreateAchievement: %v", err)
	}

	chapter := &storage.Chapter{
		StoryID:       story.ID,
		ChapterNumber: 1,
		Title:         "Arrival",
		Summary:       "You arrive at the village.",
		StartTurn:     0,
		CreatedAt:     baseTime,
	}
	if err := db.CreateChapter(chapter); err != nil {
		t.Fatalf("CreateChapter: %v", err)
	}

	if _, err := db.Conn().Exec(
		`INSERT INTO rag_chunks (story_id, text, chunk_type, turn_start, turn_end, embedding, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		story.ID, "Old memory", "summary", 0, 1, []byte{1, 2, 3}, baseTime,
	); err != nil {
		t.Fatalf("insert rag chunk: %v", err)
	}

	sessionDir := filepath.Join(dataDir, "stories", story.ID, "sessions", session.ID)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("MkdirAll session dir: %v", err)
	}
	sessionFile := filepath.Join(sessionDir, "main.jsonl")
	if err := os.WriteFile(sessionFile, []byte("old-turn\n"), 0644); err != nil {
		t.Fatalf("WriteFile session snapshot: %v", err)
	}

	snap, err := SaveGame(db, dataDir, story, char, world, session.ID, "slot-1")
	if err != nil {
		t.Fatalf("SaveGame: %v", err)
	}
	if !snap.HasFullRollbackState() {
		t.Fatal("SaveGame produced a legacy snapshot, want full rollback state")
	}
	if validation := snap.ValidateRollbackState(); validation.State != storage.SnapshotStateFull {
		t.Fatalf("SaveGame snapshot validation = %+v, want verified full state", validation)
	}
	if snap.FormatVersion != storage.CurrentSnapshotFormatVersion || snap.PayloadChecksum == "" {
		t.Fatalf("SaveGame snapshot envelope = version %d checksum %q, want current sealed format", snap.FormatVersion, snap.PayloadChecksum)
	}

	story.SettingJSON = `{"factions":["Future Court"]}`
	if err := db.UpdateStorySetting(story.ID, story.SettingJSON); err != nil {
		t.Fatalf("UpdateStorySetting: %v", err)
	}

	char.StatsJSON = `{"vitals":{"hp":{"current":1,"max":12}}}`
	char.InventoryJSON = `{"items":["future-key"]}`
	if err := db.UpdateCharacterFull(char); err != nil {
		t.Fatalf("UpdateCharacterFull: %v", err)
	}

	world.CurrentLocation = "Castle"
	world.CurrentTurn = 99
	world.FrontsJSON = `[{"id":"front-future","faction":"Future Court","title":"The court owns the walls","visibility":"known","segments":4,"progress":4}]`
	if err := db.UpdateWorldState(world); err != nil {
		t.Fatalf("UpdateWorldState: %v", err)
	}

	if err := db.CreateNPC(&storage.NPC{
		ID:                "npc-2",
		StoryID:           story.ID,
		Name:              "Future Echo",
		Role:              "oracle",
		PersonalityJSON:   `{}`,
		Disposition:       -5,
		IsAlive:           true,
		FirstAppearedTurn: 99,
		LastSeenTurn:      99,
		CreatedAt:         baseTime.Add(time.Hour),
		UpdatedAt:         baseTime.Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateNPC future: %v", err)
	}

	if err := db.CreateAchievement(&storage.Achievement{
		StoryID:     story.ID,
		Name:        "Future Badge",
		Description: "Saw the castle",
		Category:    "story",
		Rarity:      "rare",
		Context:     "future",
	}); err != nil {
		t.Fatalf("CreateAchievement future: %v", err)
	}

	if err := db.CreateChapter(&storage.Chapter{
		StoryID:       story.ID,
		ChapterNumber: 2,
		Title:         "Future",
		Summary:       "You jumped ahead.",
		StartTurn:     50,
		CreatedAt:     baseTime.Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateChapter future: %v", err)
	}

	if err := db.AppendChatMessage(&storage.ChatMessage{
		SessionID:    session.ID,
		StoryID:      story.ID,
		Turn:         99,
		Role:         "assistant",
		Content:      "A future memory leaked in.",
		MessageType:  "narrative",
		MetadataJSON: "{}",
		CreatedAt:    baseTime.Add(2 * time.Hour),
	}); err != nil {
		t.Fatalf("AppendChatMessage future: %v", err)
	}

	if _, err := db.Conn().Exec(
		`INSERT INTO rag_chunks (story_id, text, chunk_type, turn_start, turn_end, embedding, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		story.ID, "Future memory", "summary", 50, 99, []byte{9, 9, 9}, baseTime.Add(2*time.Hour),
	); err != nil {
		t.Fatalf("insert future rag chunk: %v", err)
	}

	if err := os.WriteFile(sessionFile, []byte("future-turn\n"), 0644); err != nil {
		t.Fatalf("WriteFile future session file: %v", err)
	}

	result, err := LoadGame(db, dataDir, snap.ID)
	if err != nil {
		t.Fatalf("LoadGame: %v", err)
	}
	if result.Legacy {
		t.Fatal("LoadGame returned legacy=true for a fresh snapshot")
	}

	restoredStory, err := db.GetStory(story.ID)
	if err != nil {
		t.Fatalf("GetStory after load: %v", err)
	}
	if restoredStory.SettingJSON != `{"factions":["Old Guard"]}` {
		t.Fatalf("restored story setting = %s, want original snapshot", restoredStory.SettingJSON)
	}

	restoredChar, err := db.GetCharacterByStory(story.ID)
	if err != nil {
		t.Fatalf("GetCharacterByStory after load: %v", err)
	}
	if restoredChar.InventoryJSON != `{"items":["rope"]}` {
		t.Fatalf("restored inventory = %s, want original inventory", restoredChar.InventoryJSON)
	}

	restoredWorld, err := db.GetWorldState(story.ID)
	if err != nil {
		t.Fatalf("GetWorldState after load: %v", err)
	}
	if restoredWorld.CurrentLocation != "Village" || restoredWorld.CurrentTurn != 1 {
		t.Fatalf("restored world = %s turn %d, want Village turn 1", restoredWorld.CurrentLocation, restoredWorld.CurrentTurn)
	}
	if !strings.Contains(restoredWorld.StoryHooksJSON, "Who sold you out?") {
		t.Fatalf("restored hooks = %s, want original hook payload", restoredWorld.StoryHooksJSON)
	}
	if !strings.Contains(restoredWorld.WorldReactionsJSON, "The guard remembers you") {
		t.Fatalf("restored world reactions = %s, want original reaction payload", restoredWorld.WorldReactionsJSON)
	}
	if !strings.Contains(restoredWorld.InvestigationBoardJSON, "A missing seal") {
		t.Fatalf("restored investigation board = %s, want original board payload", restoredWorld.InvestigationBoardJSON)
	}
	if !strings.Contains(restoredWorld.InvestigationBoardJSON, "The guard captain was paid") {
		t.Fatalf("restored investigation board lost hidden truth payload: %s", restoredWorld.InvestigationBoardJSON)
	}
	if !strings.Contains(restoredWorld.ProjectClocksJSON, "Train with Lyanna") {
		t.Fatalf("restored project clocks = %s, want original project payload", restoredWorld.ProjectClocksJSON)
	}
	if !strings.Contains(restoredWorld.FrontsJSON, "Checkpoints Tighten") {
		t.Fatalf("restored fronts = %s, want original front payload", restoredWorld.FrontsJSON)
	}
	if strings.Contains(restoredWorld.FrontsJSON, "Future Court") {
		t.Fatalf("restored fronts leaked future front payload: %s", restoredWorld.FrontsJSON)
	}

	npcs, err := db.ListNPCs(story.ID)
	if err != nil {
		t.Fatalf("ListNPCs after load: %v", err)
	}
	if len(npcs) != 1 || npcs[0].Name != "Old Guard" {
		t.Fatalf("restored NPCs = %+v, want only Old Guard", npcs)
	}
	if !strings.Contains(npcs[0].RelationshipJSON, `"trust":12`) {
		t.Fatalf("restored npc relationship json = %s, want trust payload", npcs[0].RelationshipJSON)
	}
	if !strings.Contains(npcs[0].NemesisJSON, `"status":"active"`) {
		t.Fatalf("restored npc nemesis json = %s, want active nemesis payload", npcs[0].NemesisJSON)
	}
	if !strings.Contains(npcs[0].DiscoveryJSON, `"stage":"identified"`) ||
		!strings.Contains(npcs[0].DiscoveryJSON, `"Gate Watcher"`) {
		t.Fatalf("restored npc discovery json = %s, want identified stage and alias", npcs[0].DiscoveryJSON)
	}
	if len(snap.NPCs) != 1 || !reflect.DeepEqual(npcs[0], snap.NPCs[0]) {
		t.Fatalf("restored npc fields = %+v, want exact snapshot %+v", npcs[0], snap.NPCs)
	}

	achievements, err := db.ListAchievements(story.ID)
	if err != nil {
		t.Fatalf("ListAchievements after load: %v", err)
	}
	if len(achievements) != 1 || achievements[0].Name != "First Step" {
		t.Fatalf("restored achievements = %+v, want only First Step", achievements)
	}

	chapters, err := db.ListChapters(story.ID)
	if err != nil {
		t.Fatalf("ListChapters after load: %v", err)
	}
	if len(chapters) != 1 || chapters[0].Title != "Arrival" {
		t.Fatalf("restored chapters = %+v, want only Arrival", chapters)
	}

	msgs, err := db.GetStoryMessages(story.ID)
	if err != nil {
		t.Fatalf("GetStoryMessages after load: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("restored message count = %d, want 2", len(msgs))
	}
	for _, msg := range msgs {
		if strings.Contains(msg.Content, "future") {
			t.Fatalf("restored messages still contain future content: %q", msg.Content)
		}
	}

	var ragCount int
	if err := db.Conn().QueryRow(`SELECT COUNT(*) FROM rag_chunks WHERE story_id = ?`, story.ID).Scan(&ragCount); err != nil {
		t.Fatalf("count rag_chunks after load: %v", err)
	}
	if ragCount != 1 {
		t.Fatalf("restored rag chunk count = %d, want 1", ragCount)
	}

	restoredSessionFile, err := os.ReadFile(sessionFile)
	if err != nil {
		t.Fatalf("ReadFile restored session file: %v", err)
	}
	if string(restoredSessionFile) != "old-turn\n" {
		t.Fatalf("restored session file = %q, want original snapshot", string(restoredSessionFile))
	}
}

func TestAutosaveRemovesReplacedSnapshotFile(t *testing.T) {
	db, dataDir := newSaveTestDB(t)
	baseTime := time.Date(2026, time.April, 9, 12, 0, 0, 0, time.UTC)

	story := &storage.Story{
		ID:              "story-autosave",
		Name:            "Autosave Tale",
		SettingJSON:     `{}`,
		StatsSchemaJSON: `{}`,
		CreatedAt:       baseTime,
		UpdatedAt:       baseTime,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	char := &storage.Character{
		ID:               "char-autosave",
		StoryID:          story.ID,
		Name:             "Mara",
		StatsJSON:        `{"vitals":{"hp":{"current":10,"max":10}}}`,
		TraitsJSON:       `[]`,
		SkillsJSON:       `[]`,
		InventoryJSON:    `{}`,
		KnownRecipesJSON: `[]`,
		CreatedAt:        baseTime,
		UpdatedAt:        baseTime,
	}
	if err := db.CreateCharacter(char); err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}

	world := &storage.WorldState{
		ID:                   "world-autosave",
		StoryID:              story.ID,
		CurrentLocation:      "Village",
		KnownLocationsJSON:   `[]`,
		GlobalEventsJSON:     `[]`,
		FactionStandingsJSON: `{}`,
		CurrentChapter:       1,
		CurrentTurn:          1,
		UpdatedAt:            baseTime,
	}
	if err := db.CreateWorldState(world); err != nil {
		t.Fatalf("CreateWorldState: %v", err)
	}

	if err := Autosave(db, dataDir, story, char, world, "sess-1"); err != nil {
		t.Fatalf("Autosave first: %v", err)
	}

	first, err := db.GetAutosave(story.ID)
	if err != nil {
		t.Fatalf("GetAutosave first: %v", err)
	}
	firstPath := filepath.Join(dataDir, "stories", story.ID, "saves", first.ID+".json")
	if _, err := os.Stat(firstPath); err != nil {
		t.Fatalf("first autosave file missing: %v", err)
	}

	world.CurrentTurn = 2
	if err := Autosave(db, dataDir, story, char, world, "sess-1"); err != nil {
		t.Fatalf("Autosave second: %v", err)
	}

	second, err := db.GetAutosave(story.ID)
	if err != nil {
		t.Fatalf("GetAutosave second: %v", err)
	}
	if second.ID == first.ID {
		t.Fatal("second autosave reused the same save id, want a fresh snapshot")
	}
	if _, err := os.Stat(firstPath); !os.IsNotExist(err) {
		t.Fatalf("old autosave file still exists or unexpected error: %v", err)
	}
}

func TestSaveGameWithMetadataPersistsRewindBranchContext(t *testing.T) {
	db, dataDir := newSaveTestDB(t)
	baseTime := time.Date(2026, time.April, 9, 13, 0, 0, 0, time.UTC)

	story := &storage.Story{
		ID:              "story-branch",
		Name:            "Branch Tale",
		SettingJSON:     `{}`,
		StatsSchemaJSON: `{}`,
		CreatedAt:       baseTime,
		UpdatedAt:       baseTime,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}

	char := &storage.Character{
		ID:               "char-branch",
		StoryID:          story.ID,
		Name:             "Mara",
		StatsJSON:        `{"vitals":{"hp":{"current":10,"max":10}}}`,
		TraitsJSON:       `[]`,
		SkillsJSON:       `[]`,
		InventoryJSON:    `[]`,
		KnownRecipesJSON: `[]`,
		CreatedAt:        baseTime,
		UpdatedAt:        baseTime,
	}
	if err := db.CreateCharacter(char); err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}

	world := &storage.WorldState{
		ID:                   "world-branch",
		StoryID:              story.ID,
		CurrentLocation:      "Village",
		KnownLocationsJSON:   `[]`,
		GlobalEventsJSON:     `[]`,
		FactionStandingsJSON: `{}`,
		StoryHooksJSON:       `[]`,
		WorldReactionsJSON:   `[]`,
		CurrentChapter:       1,
		CurrentTurn:          4,
		UpdatedAt:            baseTime,
	}
	if err := db.CreateWorldState(world); err != nil {
		t.Fatalf("CreateWorldState: %v", err)
	}

	meta := &storage.SaveMetadata{
		Kind:               "manual",
		LoadedFromSaveID:   "seed-save",
		LoadedFromSaveName: "Crossroads",
		BranchLabel:        "Rewind branch from Crossroads",
		Notes:              []string{"alternate path"},
	}

	snap, err := SaveGameWithMetadata(db, dataDir, story, char, world, "sess-branch", "Branch Save", meta)
	if err != nil {
		t.Fatalf("SaveGameWithMetadata: %v", err)
	}
	if snap.MetadataJSON == "" || snap.MetadataJSON == "{}" {
		t.Fatal("expected metadata json on snapshot")
	}

	loaded, err := db.GetSave(snap.ID)
	if err != nil {
		t.Fatalf("GetSave: %v", err)
	}
	parsed := loaded.Metadata()
	if parsed == nil {
		t.Fatal("expected parsed metadata from saved snapshot")
	}
	if parsed.LoadedFromSaveID != "seed-save" || parsed.BranchLabel != "Rewind branch from Crossroads" {
		t.Fatalf("metadata = %+v, want rewind branch context", parsed)
	}
	if parsed.BranchID == "" || parsed.CommitID == "" || loaded.BranchID != parsed.BranchID || loaded.SourceCommitID != parsed.CommitID {
		t.Fatalf("save lineage metadata=%+v row=(%q,%q)", parsed, loaded.BranchID, loaded.SourceCommitID)
	}
	var bookmarkCommit string
	if err := db.Conn().QueryRow(`SELECT commit_id FROM save_bookmarks WHERE save_id=?`, snap.ID).Scan(&bookmarkCommit); err != nil {
		t.Fatalf("save bookmark: %v", err)
	}
	if bookmarkCommit != parsed.CommitID {
		t.Fatalf("bookmark commit=%q want %q", bookmarkCommit, parsed.CommitID)
	}
}

func TestSaveLoadRoundTripRestoresCanonicalFactsAndReputation(t *testing.T) {
	db, dataDir := newSaveTestDB(t)
	story, char, world, sessionID := createMinimalSaveState(t, db)
	now := time.Now()
	npc := &storage.NPC{ID: "canon-npc", StoryID: story.ID, Name: "Mira", PersonalityJSON: "{}", RelationshipJSON: "{}", NemesisJSON: "{}", DiscoveryJSON: "{}", PrivateThoughts: "[]", NotesOnProtagonist: "[]", Desires: "[]", IsAlive: true, CreatedAt: now, UpdatedAt: now}
	if err := db.CreateNPC(npc); err != nil {
		t.Fatal(err)
	}
	fact := &storage.CharacterFact{StoryID: story.ID, SubjectEntityID: npc.ID, Predicate: "role", ObjectJSON: `"courier"`, LearnedTurn: 1, Confidence: 1, Visibility: "player"}
	if err := db.AddCharacterFact(fact); err != nil {
		t.Fatal(err)
	}
	faction := &storage.Faction{StoryID: story.ID, Name: "Couriers", Visibility: "player"}
	if err := db.CreateFaction(faction); err != nil {
		t.Fatal(err)
	}
	if err := db.AddReputationEvent(&storage.ReputationEvent{StoryID: story.ID, FactionID: faction.ID, EntityID: npc.ID, Delta: 25, Reason: "delivery", Turn: 1}); err != nil {
		t.Fatal(err)
	}
	snap, err := SaveGame(db, dataDir, story, char, world, sessionID, "canon-save")
	if err != nil {
		t.Fatal(err)
	}
	if snap.CanonicalStateJSON == "" {
		t.Fatal("save omitted canonical state")
	}
	if err := db.AddCharacterFact(&storage.CharacterFact{StoryID: story.ID, SubjectEntityID: npc.ID, Predicate: "future", ObjectJSON: `true`, LearnedTurn: 2, Confidence: 1, Visibility: "player"}); err != nil {
		t.Fatal(err)
	}
	if err := db.AddReputationEvent(&storage.ReputationEvent{StoryID: story.ID, FactionID: faction.ID, EntityID: npc.ID, Delta: 50, Reason: "future", Turn: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadGame(db, dataDir, snap.ID); err != nil {
		t.Fatal(err)
	}
	view, err := db.GetPlayerSafeEntity(story.ID, npc.ID, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Facts) != 1 || view.Facts[0].Predicate != "role" {
		t.Fatalf("restored facts=%#v", view.Facts)
	}
	score, err := db.ReputationScore(story.ID, faction.ID, npc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if score != 25 {
		t.Fatalf("restored reputation=%d", score)
	}
}

func newSaveTestDB(t *testing.T) (*storage.DB, string) {
	t.Helper()

	root := t.TempDir()
	db, err := storage.Open(filepath.Join(root, "oneday-test.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db, filepath.Join(root, "data")
}

func createMinimalSaveState(t *testing.T, db *storage.DB) (*storage.Story, *storage.Character, *storage.WorldState, string) {
	t.Helper()
	now := time.Date(2026, time.July, 11, 13, 0, 0, 0, time.UTC)
	story := &storage.Story{
		ID:              "story-snapshot-policy",
		Name:            "Snapshot Policy",
		SettingJSON:     `{}`,
		StatsSchemaJSON: `{}`,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.CreateStory(story); err != nil {
		t.Fatalf("CreateStory: %v", err)
	}
	char := &storage.Character{
		ID:               "char-snapshot-policy",
		StoryID:          story.ID,
		Name:             "Saved Timeline",
		StatsJSON:        `{}`,
		TraitsJSON:       `[]`,
		SkillsJSON:       `[]`,
		InventoryJSON:    `[]`,
		KnownRecipesJSON: `[]`,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := db.CreateCharacter(char); err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}
	world := &storage.WorldState{
		ID:                   "world-snapshot-policy",
		StoryID:              story.ID,
		CurrentLocation:      "Old Harbor",
		KnownLocationsJSON:   `[]`,
		GlobalEventsJSON:     `[]`,
		FactionStandingsJSON: `{}`,
		CurrentChapter:       1,
		CurrentTurn:          3,
		UpdatedAt:            now,
	}
	if err := db.CreateWorldState(world); err != nil {
		t.Fatalf("CreateWorldState: %v", err)
	}
	session := &storage.Session{ID: "session-snapshot-policy", StoryID: story.ID, StartedAt: now}
	if err := db.CreateSession(session); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	return story, char, world, session.ID
}
