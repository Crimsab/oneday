package engine

// StoryDefinition is the AI-generated story structure (story.json equivalent).
type StoryDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Genre       string      `json:"genre"`
	Tone        string      `json:"tone"`
	Setting     Setting     `json:"setting"`
	StatsSchema StatsSchema `json:"stats_schema"`
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

// NarrativeResponse is the standard AI response format during gameplay (AI-02).
type NarrativeResponse struct {
	Narrative     string                 `json:"narrative"`
	Choices       []Choice               `json:"choices"`
	StateChanges  map[string]interface{} `json:"state_changes,omitempty"`
	Mood          string                 `json:"mood,omitempty"`
	Location      string                 `json:"location,omitempty"`
	Challenges    []interface{}          `json:"challenges,omitempty"`
	Achievements  []interface{}          `json:"achievements,omitempty"`
	ChapterEnd    bool                   `json:"chapter_end,omitempty"`
	ChapterTitle  string                 `json:"chapter_title,omitempty"` // title for the ending chapter
}

// Choice represents an AI-suggested action.
type Choice struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
	Mood string `json:"mood,omitempty"`
}
