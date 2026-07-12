package engine

import (
	"github.com/crimsab/oneday/internal/game/contracts"
	"github.com/crimsab/oneday/internal/storage"
)

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
	Vitals     []StatDef    `json:"vitals" yaml:"vitals"`
	Attributes []StatDef    `json:"attributes" yaml:"attributes"`
	Secondary  []StatDef    `json:"secondary" yaml:"secondary"`
	Currency   *CurrencyDef `json:"currency,omitempty" yaml:"currency,omitempty"`
	HasCombat  bool         `json:"has_combat" yaml:"has_combat"`
}

// StatDef is a single stat definition.
type StatDef struct {
	Key            string   `json:"key" yaml:"key"`
	Label          string   `json:"label" yaml:"label"`
	Type           string   `json:"type,omitempty" yaml:"type,omitempty"`
	Starting       int      `json:"starting,omitempty" yaml:"starting,omitempty"`
	Min            *int     `json:"min,omitempty" yaml:"min,omitempty"`
	Max            *int     `json:"max,omitempty" yaml:"max,omitempty"`
	Formula        string   `json:"formula,omitempty" yaml:"formula,omitempty"`
	Tags           []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	RecoveryPolicy string   `json:"recovery_policy,omitempty" yaml:"recovery_policy,omitempty"`
	Progression    string   `json:"progression,omitempty" yaml:"progression,omitempty"`
}

// CurrencyDef defines the in-game currency.
type CurrencyDef struct {
	Name     string `json:"name" yaml:"name"`
	Starting int    `json:"starting" yaml:"starting"`
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
	SpeakerID string `json:"speaker_id,omitempty"`
	Speaker   string `json:"speaker"`
	Role      string `json:"role,omitempty"` // narrator, npc, player, meta, system
	Text      string `json:"text"`
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

// ASCIIArtCue requests optional same-turn ambient ASCII art for a scene.
type ASCIIArtCue struct {
	Kind      string `json:"kind"`
	Subject   string `json:"subject"`
	Detail    string `json:"detail,omitempty"`
	Placement string `json:"placement,omitempty"` // scene_header, inline
}

// VisualCue requests optional async generated imagery for a scene.
type VisualCue struct {
	Kind          string         `json:"kind"`
	Subject       string         `json:"subject"`
	Mood          string         `json:"mood,omitempty"`
	Composition   string         `json:"composition,omitempty"`
	StylePreset   string         `json:"style_preset,omitempty"`
	Importance    string         `json:"importance,omitempty"`
	SpoilerPolicy string         `json:"spoiler_policy,omitempty"`
	Negative      string         `json:"negative,omitempty"`
	Entities      []VisualEntity `json:"entities,omitempty"`
}

type VisualEntity struct {
	Name      string `json:"name"`
	Type      string `json:"type,omitempty"`
	VisualRef string `json:"visual_ref,omitempty"`
}

// NarrativeResponse is the standard AI response format during gameplay (AI-02).
type NarrativeResponse struct {
	Narrative            string                         `json:"narrative"`
	Choices              []Choice                       `json:"choices"`
	StateChanges         map[string]interface{}         `json:"state_changes,omitempty"`
	TurnDelta            *TurnDelta                     `json:"turn_delta,omitempty"`
	Mood                 string                         `json:"mood,omitempty"`
	Location             string                         `json:"location,omitempty"`
	LocationTransition   *storage.LocationTransition    `json:"location_transition,omitempty"`
	SceneType            string                         `json:"scene_type,omitempty"`
	DialogueBlocks       []DialogueBlock                `json:"dialogue_blocks,omitempty"`
	EntitiesMentioned    []EntityMention                `json:"entities_mentioned,omitempty"`
	EventCallouts        []EventCallout                 `json:"event_callouts,omitempty"`
	ASCIICue             *ASCIIArtCue                   `json:"ascii_cue,omitempty"`
	VisualCue            *VisualCue                     `json:"visual_cue,omitempty"`
	ASCIIArt             string                         `json:"ascii_art,omitempty"`
	OpenHooks            []StoryHook                    `json:"open_hooks,omitempty"`
	WorldReactions       []WorldReaction                `json:"world_reactions,omitempty"`
	Challenges           []*ChallengeSpec               `json:"challenges,omitempty"`
	SocialDuel           *SocialDuelCue                 `json:"social_duel,omitempty"`
	AchievementEarned    *AchievementData               `json:"achievement_earned,omitempty"`
	ChapterEnd           bool                           `json:"chapter_end,omitempty"`
	ChapterTitle         string                         `json:"chapter_title,omitempty"` // title for the ending chapter
	CombatStart          *EnemyStats                    `json:"combat_start,omitempty"`  // set by AI to initiate combat
	PersistedAchievement *storage.Achievement           `json:"-"`                       // TUI-only: set after DB persist
	AppliedStateChanges  []StateChange                  `json:"-"`                       // TUI-only: engine-applied changes for renderer/callouts
	ResolvedOutcome      *contracts.OutcomeEnvelope     `json:"resolved_outcome,omitempty"`
	ChallengeInstance    *contracts.ChallengeInstance   `json:"challenge_instance,omitempty"`
	ChallengeResolution  *contracts.ChallengeResolution `json:"challenge_resolution,omitempty"`
	AutomaticMiniGame    *MiniGameInstance              `json:"automatic_minigame,omitempty"`
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
	Description string        `json:"description,omitempty"` // short player-facing context for what this challenge represents
	Difficulty  int           `json:"difficulty,omitempty"`  // threshold for dice/stat checks
	Stat        string        `json:"stat,omitempty"`        // which stat to check (e.g., "per", "str")
	Item        string        `json:"item,omitempty"`        // required item name
	Skill       string        `json:"skill,omitempty"`       // required skill name
	SkillLevel  int           `json:"skill_level,omitempty"` // minimum skill level
	NPCName     string        `json:"npc_name,omitempty"`    // NPC for relationship check
	Disposition int           `json:"disposition,omitempty"` // min disposition threshold
	MiniGame    string        `json:"mini_game,omitempty"`   // registered MiniGameType
	Modifiers   []Modifier    `json:"modifiers,omitempty"`   // bonuses/penalties
	// For riddles: AI provides these
	Riddle string `json:"riddle,omitempty"`
	Answer string `json:"answer,omitempty"`
	// For memory sequence
	Sequence []string `json:"sequence,omitempty"`
	// For quick-time
	TimeLimit float64 `json:"time_limit,omitempty"` // seconds
}

// SocialDuelCueMode identifies how the AI wants the runtime to treat a social duel beat.
type SocialDuelCueMode string

const (
	SocialDuelCueOffer    SocialDuelCueMode = "offer"
	SocialDuelCueContinue SocialDuelCueMode = "continue"
)

// SocialDuelCue is AI-authored framing for a high-stakes dialogue scene.
// The engine still owns rounds, rolls, composure, and outcomes.
type SocialDuelCue struct {
	Mode             SocialDuelCueMode    `json:"mode,omitempty"`
	NPCName          string               `json:"npc_name,omitempty"`
	Objective        string               `json:"objective,omitempty"`
	NPCGoal          string               `json:"npc_goal,omitempty"`
	Stakes           string               `json:"stakes,omitempty"`
	Pressure         string               `json:"pressure,omitempty"`
	Opening          string               `json:"opening,omitempty"`
	ExchangeSummary  string               `json:"exchange_summary,omitempty"`
	Leverage         []SocialDuelLeverage `json:"leverage,omitempty"`
	SuggestedActions []SocialAction       `json:"suggested_actions,omitempty"`
	FailForward      string               `json:"fail_forward,omitempty"`
}

// SocialDuelLeverage is a narrative leverage opportunity the AI can surface.
type SocialDuelLeverage struct {
	Key    string `json:"key,omitempty"`
	Label  string `json:"label,omitempty"`
	Detail string `json:"detail,omitempty"`
	Kind   string `json:"kind,omitempty"`
}

// Modifier is a named bonus or penalty to a roll.
type Modifier struct {
	Source string `json:"source"` // e.g., "Sword of Light", "Stealth skill"
	Value  int    `json:"value"`  // positive = bonus, negative = penalty
}

// ChallengeResult is the engine's resolution of a challenge.
type ChallengeResult struct {
	Passed     bool                       `json:"passed"`
	Roll       int                        `json:"roll,omitempty"`  // the raw d100 roll (for dice_roll)
	Total      int                        `json:"total,omitempty"` // roll + modifiers
	Difficulty int                        `json:"difficulty,omitempty"`
	Modifiers  []Modifier                 `json:"modifiers,omitempty"`
	Detail     string                     `json:"detail"` // human-readable summary
	RollLog    []RollRecord               `json:"roll_log,omitempty"`
	Outcome    *contracts.OutcomeEnvelope `json:"outcome,omitempty"`
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
	Narrative           string
	Choices             []Choice
	PlayerDamage        int // damage dealt TO player this turn
	EnemyDamage         int // damage dealt TO enemy this turn
	PlayerHP            int
	EnemyHP             int
	CombatOver          bool
	Victory             bool
	DefeatOutcome       string // only set if CombatOver && !Victory
	Summary             string // only set if CombatOver
	Mood                string
	RollLog             []RollRecord
	Outcome             *contracts.OutcomeEnvelope
	ChallengeInstance   *contracts.ChallengeInstance
	ChallengeResolution *contracts.ChallengeResolution
}

// MiniGameType identifies a specific mini-game.
type MiniGameType string

const (
	MiniGameRPS         MiniGameType = "rps"
	MiniGameMemory      MiniGameType = "memory"
	MiniGameQuickTime   MiniGameType = "quicktime"
	MiniGameRiddle      MiniGameType = "riddle"
	MiniGameDeduction   MiniGameType = "deduction"
	MiniGameNegotiation MiniGameType = "negotiation"
	MiniGamePattern     MiniGameType = "pattern"
	MiniGameBidding     MiniGameType = "bidding"
	MiniGameCourtroom   MiniGameType = "courtroom"
	MiniGameComedy      MiniGameType = "comedy"
)
