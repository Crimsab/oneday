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
}

// defaultPalette is used when no mood matches.
var defaultPalette = MoodPalette{
	Accent:          Accent,
	Border:          Secondary,
	StatusBarBG:     Background,
	NarrativeAccent: Primary,
}

// GetMoodPalette returns the palette for a mood, falling back to the default theme.
func GetMoodPalette(mood string) MoodPalette {
	if p, ok := MoodPalettes[mood]; ok {
		return p
	}
	return defaultPalette
}
