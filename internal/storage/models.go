package storage

import "time"

// Story represents a game story with its setting and rules.
type Story struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	SettingJSON      string    `json:"setting_json"`
	StatsSchemaJSON  string    `json:"stats_schema_json"`
	Description      string    `json:"description"`
	Genre            string    `json:"genre"`
	Tone             string    `json:"tone"`
	Language         string    `json:"language"`
	WritingStyle     string    `json:"writing_style"`
	PromptDirectives string    `json:"prompt_directives"`
	Revision         int64     `json:"revision"`
	ActiveBranchID   string    `json:"active_branch_id"`
	IsArchived       bool      `json:"is_archived"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// StoryBranch is a named mutable pointer into the immutable turn commit DAG.
type StoryBranch struct {
	ID           string    `json:"id"`
	StoryID      string    `json:"story_id"`
	Name         string    `json:"name"`
	ForkCommitID string    `json:"fork_commit_id,omitempty"`
	HeadCommitID string    `json:"head_commit_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TurnCommit identifies one immutable canonical materialization.
type TurnCommit struct {
	ID             string    `json:"id"`
	StoryID        string    `json:"story_id"`
	BranchID       string    `json:"branch_id"`
	ParentCommitID string    `json:"parent_commit_id,omitempty"`
	CanonicalTurn  int       `json:"canonical_turn"`
	StoryRevision  int64     `json:"story_revision"`
	PayloadHash    string    `json:"payload_hash"`
	Kind           string    `json:"kind"`
	Message        string    `json:"message,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// TurnSnapshot is the immutable materialization payload owned by a commit.
type TurnSnapshot struct {
	CommitID      string    `json:"commit_id"`
	StoryID       string    `json:"story_id"`
	FormatVersion int       `json:"format_version"`
	PayloadJSON   string    `json:"payload_json"`
	PayloadHash   string    `json:"payload_hash"`
	CreatedAt     time.Time `json:"created_at"`
}

// Character represents the player's protagonist.
type Character struct {
	ID               string    `json:"id"`
	StoryID          string    `json:"story_id"`
	Name             string    `json:"name"`
	Background       string    `json:"background"`
	StatsJSON        string    `json:"stats_json"`
	TraitsJSON       string    `json:"traits_json"`
	SkillsJSON       string    `json:"skills_json"`
	InventoryJSON    string    `json:"inventory_json"`
	KnownRecipesJSON string    `json:"known_recipes_json"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// NPC represents an AI-generated non-player character.
type NPC struct {
	ID                 string    `json:"id"`
	StoryID            string    `json:"story_id"`
	CanonicalEntityID  string    `json:"canonical_entity_id"`
	Name               string    `json:"name"`
	Role               string    `json:"role"`
	Appearance         string    `json:"appearance"`
	PersonalityJSON    string    `json:"personality_json"`
	RelationshipJSON   string    `json:"relationship_json"`
	NemesisJSON        string    `json:"nemesis_json"`
	DiscoveryJSON      string    `json:"discovery_json"`
	PrivateThoughts    string    `json:"private_thoughts"`
	NotesOnProtagonist string    `json:"notes_on_protagonist"`
	Desires            string    `json:"desires"`
	Disposition        int       `json:"disposition"`
	IsAlive            bool      `json:"is_alive"`
	FirstAppearedTurn  int       `json:"first_appeared_turn"`
	LastSeenTurn       int       `json:"last_seen_turn"`
	CanHelp            bool      `json:"can_help"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// WorldState tracks the global state of a story.
type WorldState struct {
	ID                     string    `json:"id"`
	StoryID                string    `json:"story_id"`
	CurrentLocation        string    `json:"current_location"`
	CurrentLocationID      string    `json:"current_location_id"`
	KnownLocationsJSON     string    `json:"known_locations_json"`
	GlobalEventsJSON       string    `json:"global_events_json"`
	FactionStandingsJSON   string    `json:"faction_standings_json"`
	StoryHooksJSON         string    `json:"story_hooks_json"`
	WorldReactionsJSON     string    `json:"world_reactions_json"`
	InvestigationBoardJSON string    `json:"investigation_board_json"`
	ProjectClocksJSON      string    `json:"project_clocks_json"`
	PlayerGuidanceJSON     string    `json:"player_guidance_json"`
	FrontsJSON             string    `json:"fronts_json"`
	CharacterTimelineJSON  string    `json:"character_timeline_json"`
	SceneContractJSON      string    `json:"scene_contract_json"`
	CurrentChapter         int       `json:"current_chapter"`
	CurrentTurn            int       `json:"current_turn"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// Session represents a play session within a story.
type Session struct {
	ID             string     `json:"id"`
	StoryID        string     `json:"story_id"`
	StartedAt      time.Time  `json:"started_at"`
	EndedAt        *time.Time `json:"ended_at,omitempty"`
	Summary        string     `json:"summary"`
	BranchID       string     `json:"branch_id,omitempty"`
	SourceCommitID string     `json:"source_commit_id,omitempty"`
}

// ChatMessage represents a single message in a session.
type ChatMessage struct {
	ID             int64     `json:"id"`
	SessionID      string    `json:"session_id"`
	StoryID        string    `json:"story_id"`
	Turn           int       `json:"turn"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	MessageType    string    `json:"message_type"`
	MetadataJSON   string    `json:"metadata_json"`
	CreatedAt      time.Time `json:"created_at"`
	BranchID       string    `json:"branch_id,omitempty"`
	SourceCommitID string    `json:"source_commit_id,omitempty"`
}

// Chapter represents an AI-generated chapter summary.
type Chapter struct {
	ID             int64     `json:"id"`
	StoryID        string    `json:"story_id"`
	ChapterNumber  int       `json:"chapter_number"`
	Title          string    `json:"title"`
	Summary        string    `json:"summary"`
	StartTurn      int       `json:"start_turn"`
	EndTurn        *int      `json:"end_turn,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	BranchID       string    `json:"branch_id,omitempty"`
	SourceCommitID string    `json:"source_commit_id,omitempty"`
}

// Achievement represents an earned achievement.
type Achievement struct {
	ID          int64     `json:"id"`
	StoryID     string    `json:"story_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Rarity      string    `json:"rarity"`
	Context     string    `json:"context"`
	EarnedAt    time.Time `json:"earned_at"`
}

// CombatLog records the outcome of a single combat encounter.
type CombatLog struct {
	ID             int64     `json:"id"`
	StoryID        string    `json:"story_id"`
	SessionID      string    `json:"session_id"`
	EnemyName      string    `json:"enemy_name"`
	EnemyHP        int       `json:"enemy_hp"`
	Turns          int       `json:"turns"`
	Victory        bool      `json:"victory"`
	DefeatOutcome  string    `json:"defeat_outcome"`
	PlayerHPStart  int       `json:"player_hp_start"`
	PlayerHPEnd    int       `json:"player_hp_end"`
	CreatedAt      time.Time `json:"created_at"`
	BranchID       string    `json:"branch_id,omitempty"`
	SourceCommitID string    `json:"source_commit_id,omitempty"`
}
