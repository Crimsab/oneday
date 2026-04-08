package engine

import "github.com/crimsab/oneday/internal/storage"

// StoryDefinition is the AI-generated story structure (story.json equivalent).
type StoryDefinition struct {
	Name             string      `json:"name"`
	Description      string      `json:"description"`
	Genre            string      `json:"genre"`
	Tone             string      `json:"tone"`
	Language         string      `json:"language"`
	WritingStyle     string      `json:"writing_style"`
	PromptDirectives string      `json:"prompt_directives"`
	Setting          Setting     `json:"setting"`
	StatsSchema      StatsSchema `json:"stats_schema"`
}

// Setting describes the story world.
type Setting struct {
	WorldName       string   `json:"world_name"`
	Era             string   `json:"era"`
	Geography       string   `json:"geography"`
	MagicSystem     string   `json:"magic_system"`
	TechnologyLevel string   `json:"technology_level"`
	Society         string   `json:"society"`
	Rules           []string `json:"rules"`
	Factions        []string `json:"factions"`
	Cultures        []string `json:"cultures"`
	Dangers         []string `json:"dangers"`
	ToneGuidelines  string   `json:"tone_guidelines,omitempty"`
}

// StatsSchema defines what stats exist in this story.
type StatsSchema struct {
	Vitals     []StatDef    `json:"vitals"`
	Attributes []StatDef    `json:"attributes"`
	Secondary  []StatDef    `json:"secondary"`
	Currency   *CurrencyDef `json:"currency,omitempty"`
	HasCombat  bool         `json:"has_combat"`
}

// StatDef is a single stat definition.
type StatDef struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Starting int    `json:"starting,omitempty"`
}

// CurrencyDef defines the in-game currency.
type CurrencyDef struct {
	Name     string `json:"name"`
	Starting int    `json:"starting"`
}

// InitialStats generates starting stats JSON from the schema.
func (s StatsSchema) InitialStats() map[string]interface{} {
	stats := map[string]interface{}{}

	vitals := map[string]map[string]int{}
	for _, v := range s.Vitals {
		vitals[v.Key] = map[string]int{
			"current": v.Starting, "max": v.Starting,
		}
	}
	stats["vitals"] = vitals

	attrs := map[string]int{}
	for _, a := range s.Attributes {
		attrs[a.Key] = a.Starting
	}
	stats["attributes"] = attrs

	secondary := map[string]interface{}{}
	for _, sec := range s.Secondary {
		secondary[sec.Key] = sec.Starting
	}
	stats["secondary"] = secondary

	if s.Currency != nil {
		stats["currency"] = s.Currency.Starting
	}

	stats["traits"] = []string{}
	stats["skills"] = map[string]interface{}{}
	stats["titles"] = []string{}
	stats["deaths"] = 0

	return stats
}

// AchievementData is the structured achievement payload from the AI response.
type AchievementData struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Rarity      string `json:"rarity"`
	Category    string `json:"category"`
	Context     string `json:"context"`
}

// DialogueBlock is optional renderer-facing metadata for structured dialogue.
type DialogueBlock struct {
	Speaker string `json:"speaker"`
	Role    string `json:"role,omitempty"` // narrator, npc, player, meta, system
	Text    string `json:"text"`
}

// EntityMention is optional renderer metadata for important named entities.
type EntityMention struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"` // npc, location, faction, item, skill, title, chapter
}

// EventCallout is optional renderer metadata for compact state/event summaries.
type EventCallout struct {
	Kind   string `json:"kind,omitempty"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
}

// NarrativeResponse is the standard AI response format during gameplay (AI-02).
type NarrativeResponse struct {
	Narrative            string                 `json:"narrative"`
	Choices              []Choice               `json:"choices"`
	StateChanges         map[string]interface{} `json:"state_changes,omitempty"`
	Mood                 string                 `json:"mood,omitempty"`
	Location             string                 `json:"location,omitempty"`
	SceneType            string                 `json:"scene_type,omitempty"`
	DialogueBlocks       []DialogueBlock        `json:"dialogue_blocks,omitempty"`
	EntitiesMentioned    []EntityMention        `json:"entities_mentioned,omitempty"`
	EventCallouts        []EventCallout         `json:"event_callouts,omitempty"`
	Challenges           []*ChallengeSpec       `json:"challenges,omitempty"`
	AchievementEarned    *AchievementData       `json:"achievement_earned,omitempty"`
	ChapterEnd           bool                   `json:"chapter_end,omitempty"`
	ChapterTitle         string                 `json:"chapter_title,omitempty"` // title for the ending chapter
	CombatStart          *EnemyStats            `json:"combat_start,omitempty"`  // set by AI to initiate combat
	PersistedAchievement *storage.Achievement   `json:"-"`                       // TUI-only: set after DB persist
	AppliedStateChanges  []StateChange          `json:"-"`                       // TUI-only: engine-applied changes for renderer/callouts
}

// Choice represents an AI-suggested action.
type Choice struct {
	ID           int      `json:"id"`
	Text         string   `json:"text"`
	Mood         string   `json:"mood,omitempty"`
	Intent       string   `json:"intent,omitempty"`
	Risk         string   `json:"risk,omitempty"`
	Scope        string   `json:"scope,omitempty"`
	Certainty    string   `json:"certainty,omitempty"`
	RelatedStats []string `json:"related_stats,omitempty"`
}

// --- Challenge Types ---

// ChallengeType identifies the kind of challenge.
type ChallengeType string

const (
	ChallengeStatCheck  ChallengeType = "stat_check"
	ChallengeDiceRoll   ChallengeType = "dice_roll"
	ChallengeItemCheck  ChallengeType = "item_check"
	ChallengeSkillCheck ChallengeType = "skill_check"
	ChallengeRelCheck   ChallengeType = "relationship_check"
	ChallengeMiniGame   ChallengeType = "mini_game"
)

// ChallengeSpec is what the AI includes in its response to request a challenge.
type ChallengeSpec struct {
	Type        ChallengeType `json:"type"`
	Difficulty  int           `json:"difficulty,omitempty"`  // threshold for dice/stat checks
	Stat        string        `json:"stat,omitempty"`        // which stat to check (e.g., "per", "str")
	Item        string        `json:"item,omitempty"`        // required item name
	Skill       string        `json:"skill,omitempty"`       // required skill name
	SkillLevel  int           `json:"skill_level,omitempty"` // minimum skill level
	NPCName     string        `json:"npc_name,omitempty"`    // NPC for relationship check
	Disposition int           `json:"disposition,omitempty"` // min disposition threshold
	MiniGame    string        `json:"mini_game,omitempty"`   // "rps", "memory", "quicktime", "riddle"
	Modifiers   []Modifier    `json:"modifiers,omitempty"`   // bonuses/penalties
	// For riddles: AI provides these
	Riddle string `json:"riddle,omitempty"`
	Answer string `json:"answer,omitempty"`
	// For memory sequence
	Sequence []string `json:"sequence,omitempty"`
	// For quick-time
	TimeLimit float64 `json:"time_limit,omitempty"` // seconds
}

// Modifier is a named bonus or penalty to a roll.
type Modifier struct {
	Source string `json:"source"` // e.g., "Sword of Light", "Stealth skill"
	Value  int    `json:"value"`  // positive = bonus, negative = penalty
}

// ChallengeResult is the engine's resolution of a challenge.
type ChallengeResult struct {
	Passed     bool       `json:"passed"`
	Roll       int        `json:"roll,omitempty"`  // the raw d100 roll (for dice_roll)
	Total      int        `json:"total,omitempty"` // roll + modifiers
	Difficulty int        `json:"difficulty,omitempty"`
	Modifiers  []Modifier `json:"modifiers,omitempty"`
	Detail     string     `json:"detail"` // human-readable summary
}

// --- Combat Types ---

// EnemyBehavior describes an enemy's combat AI pattern.
type EnemyBehavior string

const (
	BehaviorAggressive  EnemyBehavior = "aggressive"
	BehaviorDefensive   EnemyBehavior = "defensive"
	BehaviorTactical    EnemyBehavior = "tactical"
	BehaviorBeast       EnemyBehavior = "beast"
	BehaviorIntelligent EnemyBehavior = "intelligent"
)

// EnemyStats represents a combat enemy, generated by AI and validated by engine.
type EnemyStats struct {
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	HP          int           `json:"hp"`
	MaxHP       int           `json:"max_hp"`
	Attack      int           `json:"attack"`  // base attack power
	Defense     int           `json:"defense"` // damage reduction
	Behavior    EnemyBehavior `json:"behavior"`
	Abilities   []string      `json:"abilities,omitempty"` // special moves (narrative only)
	WeakTo      string        `json:"weak_to,omitempty"`   // narrative weakness hint
}

// CombatState tracks the current state of an active combat.
type CombatState struct {
	Enemy         EnemyStats `json:"enemy"`
	PlayerHP      int        `json:"player_hp"`
	PlayerMaxHP   int        `json:"player_max_hp"`
	Turn          int        `json:"turn"`           // combat turn counter (starts at 1)
	SubSessionID  string     `json:"sub_session_id"` // JSONL sub-session for combat chat
	Phase         string     `json:"phase"`          // "player_turn", "enemy_turn", "resolved"
	Resolved      bool       `json:"resolved"`
	Victory       bool       `json:"victory"`                  // true = player won
	DefeatOutcome string     `json:"defeat_outcome,omitempty"` // "death", "capture", "rescue", "retreat"
	Summary       string     `json:"summary,omitempty"`        // written to main narrative after combat
}

// CombatTurnResult contains everything the TUI needs after a combat turn.
type CombatTurnResult struct {
	Narrative     string
	Choices       []Choice
	PlayerDamage  int // damage dealt TO player this turn
	EnemyDamage   int // damage dealt TO enemy this turn
	PlayerHP      int
	EnemyHP       int
	CombatOver    bool
	Victory       bool
	DefeatOutcome string // only set if CombatOver && !Victory
	Summary       string // only set if CombatOver
	Mood          string
}

// MiniGameType identifies a specific mini-game.
type MiniGameType string

const (
	MiniGameRPS       MiniGameType = "rps"
	MiniGameMemory    MiniGameType = "memory"
	MiniGameQuickTime MiniGameType = "quicktime"
	MiniGameRiddle    MiniGameType = "riddle"
)
