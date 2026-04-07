package storage

import "time"

// Story represents a game story with its setting and rules.
type Story struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	SettingJSON     string    `json:"setting_json"`
	StatsSchemaJSON string    `json:"stats_schema_json"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
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
	ID                string    `json:"id"`
	StoryID           string    `json:"story_id"`
	Name              string    `json:"name"`
	Role              string    `json:"role"`
	PersonalityJSON   string    `json:"personality_json"`
	PrivateThoughts   string    `json:"private_thoughts"`
	Desires           string    `json:"desires"`
	Disposition       int       `json:"disposition"`
	IsAlive           bool      `json:"is_alive"`
	FirstAppearedTurn int       `json:"first_appeared_turn"`
	CanHelp           bool      `json:"can_help"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// WorldState tracks the global state of a story.
type WorldState struct {
	ID                   string    `json:"id"`
	StoryID              string    `json:"story_id"`
	CurrentLocation      string    `json:"current_location"`
	KnownLocationsJSON   string    `json:"known_locations_json"`
	GlobalEventsJSON     string    `json:"global_events_json"`
	FactionStandingsJSON string    `json:"faction_standings_json"`
	CurrentChapter       int       `json:"current_chapter"`
	CurrentTurn          int       `json:"current_turn"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// Session represents a play session within a story.
type Session struct {
	ID        string     `json:"id"`
	StoryID   string     `json:"story_id"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	Summary   string     `json:"summary"`
}

// ChatMessage represents a single message in a session.
type ChatMessage struct {
	ID           int64     `json:"id"`
	SessionID    string    `json:"session_id"`
	StoryID      string    `json:"story_id"`
	Turn         int       `json:"turn"`
	Role         string    `json:"role"`
	Content      string    `json:"content"`
	MessageType  string    `json:"message_type"`
	MetadataJSON string    `json:"metadata_json"`
	CreatedAt    time.Time `json:"created_at"`
}

// Chapter represents an AI-generated chapter summary.
type Chapter struct {
	ID            int64     `json:"id"`
	StoryID       string    `json:"story_id"`
	ChapterNumber int       `json:"chapter_number"`
	Title         string    `json:"title"`
	Summary       string    `json:"summary"`
	StartTurn     int       `json:"start_turn"`
	EndTurn       *int      `json:"end_turn,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
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
