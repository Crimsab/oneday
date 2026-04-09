package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/storage"
)

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
		ID:                   "world-1",
		StoryID:              story.ID,
		CurrentLocation:      "Village",
		KnownLocationsJSON:   `["Village"]`,
		GlobalEventsJSON:     `[]`,
		FactionStandingsJSON: `{}`,
		StoryHooksJSON:       `[{"id":"hook-1","kind":"mystery","title":"Who sold you out?","status":"active"}]`,
		WorldReactionsJSON:   `[{"id":"react-1","kind":"rumor","title":"The guard remembers you","status":"active"}]`,
		FrontsJSON:           `[{"id":"front-1","faction":"Old Guard","title":"The Old Guard is tightening checkpoints","public_title":"Checkpoints Tighten","public_stakes":"Travel is getting tense.","visibility":"known","segments":4,"progress":2}]`,
		CurrentChapter:       1,
		CurrentTurn:          1,
		UpdatedAt:            baseTime,
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

	if err := db.CreateNPC(&storage.NPC{
		ID:                "npc-1",
		StoryID:           story.ID,
		Name:              "Old Guard",
		Role:              "watcher",
		PersonalityJSON:   `{}`,
		RelationshipJSON:  `{"trust":12,"fear":-3,"respect":8}`,
		NemesisJSON:       `{"status":"active","rivalry_score":6,"escalation_tier":2,"threat_posture":"watching"}`,
		Disposition:       1,
		IsAlive:           true,
		FirstAppearedTurn: 1,
		LastSeenTurn:      1,
		CreatedAt:         baseTime,
		UpdatedAt:         baseTime,
	}); err != nil {
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
