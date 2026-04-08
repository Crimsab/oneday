package theme

import "github.com/charmbracelet/lipgloss"

// MoodPalette holds color overrides for a specific narrative mood.
type MoodPalette struct {
	Accent          lipgloss.Color
	Border          lipgloss.Color
	StatusBarBG     lipgloss.Color
	NarrativeAccent lipgloss.Color
}

// MoodPalettes maps mood strings to color overrides.
var MoodPalettes = map[string]MoodPalette{
	"tense": {
		Accent:          lipgloss.Color("#CD5C5C"),
		Border:          lipgloss.Color("#8B0000"),
		StatusBarBG:     lipgloss.Color("#3A1A1A"),
		NarrativeAccent: lipgloss.Color("#FF6B6B"),
	},
	"peaceful": {
		Accent:          lipgloss.Color("#6B8E6B"),
		Border:          lipgloss.Color("#2E4F2E"),
		StatusBarBG:     lipgloss.Color("#1A2E1A"),
		NarrativeAccent: lipgloss.Color("#88C888"),
	},
	"dark": {
		Accent:          lipgloss.Color("#666666"),
		Border:          lipgloss.Color("#333333"),
		StatusBarBG:     lipgloss.Color("#1A1A1A"),
		NarrativeAccent: lipgloss.Color("#999999"),
	},
	"epic": {
		Accent:          lipgloss.Color("#FFD700"),
		Border:          lipgloss.Color("#B8860B"),
		StatusBarBG:     lipgloss.Color("#2A2A1A"),
		NarrativeAccent: lipgloss.Color("#FFED4A"),
	},
	"mysterious": {
		Accent:          lipgloss.Color("#9B59B6"),
		Border:          lipgloss.Color("#6C3483"),
		StatusBarBG:     lipgloss.Color("#2A1A3A"),
		NarrativeAccent: lipgloss.Color("#BB8FCE"),
	},
	"lighthearted": {
		Accent:          lipgloss.Color("#C9A96E"),
		Border:          lipgloss.Color("#8B7355"),
		StatusBarBG:     lipgloss.Color("#2A2A1A"),
		NarrativeAccent: lipgloss.Color("#E8D5A8"),
	},
	"dramatic": {
		Accent:          lipgloss.Color("#E74C3C"),
		Border:          lipgloss.Color("#922B21"),
		StatusBarBG:     lipgloss.Color("#3A1A1A"),
		NarrativeAccent: lipgloss.Color("#F1948A"),
	},
	// "neutral" is the default on story start/resume — no strong tint, uses base theme.
	"neutral": {
		Accent:          Accent,
		Border:          Secondary,
		StatusBarBG:     lipgloss.Color("#2A2A2A"),
		NarrativeAccent: Primary,
	},
}

// moodAliases maps common AI-generated mood words to canonical palette keys.
// Allows the AI to use natural language mood descriptions that map to our palettes.
var moodAliases = map[string]string{
	"calm":         "peaceful",
	"serene":       "peaceful",
	"tranquil":     "peaceful",
	"hopeful":      "lighthearted",
	"cheerful":     "lighthearted",
	"comedic":      "lighthearted",
	"humorous":     "lighthearted",
	"horror":       "dark",
	"grim":         "dark",
	"gloomy":       "dark",
	"bleak":        "dark",
	"ominous":      "dark",
	"triumphant":   "epic",
	"heroic":       "epic",
	"glorious":     "epic",
	"suspenseful":  "mysterious",
	"eerie":        "mysterious",
	"foreboding":   "mysterious",
	"combat":       "tense",
	"sad":          "dramatic",
	"melancholic":  "dramatic",
	"sorrowful":    "dramatic",
	"tense":        "tense",
	"intense":      "tense",
}

// defaultPalette is used when no mood matches.
var defaultPalette = MoodPalette{
	Accent:          Accent,
	Border:          Secondary,
	StatusBarBG:     Background,
	NarrativeAccent: Primary,
}

// GetMoodPalette returns the palette for a mood, resolving aliases first,
// then falling back to the default theme if no match is found.
func GetMoodPalette(mood string) MoodPalette {
	// Direct palette match.
	if p, ok := MoodPalettes[mood]; ok {
		return p
	}
	// Alias resolution.
	if canonical, ok := moodAliases[mood]; ok {
		if p, ok := MoodPalettes[canonical]; ok {
			return p
		}
	}
	return defaultPalette
}
