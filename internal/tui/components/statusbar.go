package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/tui/theme"
)

// Vital represents a single vital stat for the status bar.
type Vital struct {
	Label   string
	Current int
	Max     int
}

// StatusBarData holds all the data the status bar needs to render.
type StatusBarData struct {
	Vitals             []Vital
	Model              string
	Latency            int64 // milliseconds
	TimeToFirstToken   int64 // milliseconds
	PromptTokens       int
	CompletionTokens   int
	ReasoningTokens    int
	TotalTokens        int
	CachedPromptTokens int
	CostUSD            float64
	Streamed           bool
}

// StatusBarModel renders the bottom status bar.
type StatusBarModel struct {
	data      StatusBarData
	width     int
	moodBG    lipgloss.Color
	hasMoodBG bool
}

// NewStatusBar creates a status bar.
func NewStatusBar() StatusBarModel {
	return StatusBarModel{}
}

// SetData updates the status bar data.
func (s *StatusBarModel) SetData(data StatusBarData) {
	s.data = data
}

// SetWidth updates the bar width.
func (s *StatusBarModel) SetWidth(w int) {
	s.width = w
}

// View renders the status bar.
func (s StatusBarModel) View() string {
	if s.width == 0 {
		return ""
	}

	// Left side: vitals
	var vitals string
	for i, v := range s.data.Vitals {
		if i > 0 {
			vitals += "  "
		}
		current := v.Current
		if v.Max > 0 && current > v.Max {
			current = v.Max
		}
		if current < 0 {
			current = 0
		}
		var style lipgloss.Style
		if v.Max > 0 {
			pct := float64(current) / float64(v.Max)
			if pct <= 0.25 {
				style = theme.DangerText
			} else if pct <= 0.5 {
				style = lipgloss.NewStyle().Foreground(theme.Accent)
			} else {
				style = theme.NormalText
			}
		} else {
			style = theme.NormalText
		}
		vitals += style.Render(fmt.Sprintf("%s: %d/%d", v.Label, current, v.Max))
	}

	// Right side: AI model + latency
	var aiInfo string
	if s.data.Model != "" {
		parts := []string{
			compactModelName(s.data.Model),
			formatDurationSeconds(s.data.Latency),
		}
		if s.data.Streamed && s.data.TimeToFirstToken > 0 {
			parts = append(parts, "ft "+formatDurationSeconds(s.data.TimeToFirstToken))
		}
		if s.data.TotalTokens > 0 {
			tokenPart := fmt.Sprintf("%dt", s.data.TotalTokens)
			if s.data.CompletionTokens > 0 || s.data.PromptTokens > 0 {
				tokenPart = fmt.Sprintf("%dt (%dp/%dc", s.data.TotalTokens, s.data.PromptTokens, s.data.CompletionTokens)
				if s.data.ReasoningTokens > 0 {
					tokenPart += fmt.Sprintf(", r%d", s.data.ReasoningTokens)
				}
				tokenPart += ")"
			}
			parts = append(parts, tokenPart)
			if s.data.CachedPromptTokens > 0 {
				parts = append(parts, fmt.Sprintf("cache %dp", s.data.CachedPromptTokens))
			}
			if rate := throughputTokensPerSecond(s.data.CompletionTokens, s.data.Latency); rate > 0 {
				parts = append(parts, fmt.Sprintf("%.1ft/s", rate))
			}
		}
		if s.data.CostUSD > 0 {
			parts = append(parts, fmt.Sprintf("$%.5f", s.data.CostUSD))
		}
		aiInfo = theme.MutedText.Render(joinStatusParts(parts))
	}

	// Compose the bar with vitals on left, ai info on right
	leftWidth := lipgloss.Width(vitals)
	rightWidth := lipgloss.Width(aiInfo)
	gap := s.width - leftWidth - rightWidth - 2 // 2 for padding
	if gap < 1 {
		gap = 1
	}
	spacer := lipgloss.NewStyle().Width(gap).Render("")

	bar := lipgloss.JoinHorizontal(lipgloss.Top, vitals, spacer, aiInfo)

	barStyle := theme.StatusBar.Width(s.width)
	if s.hasMoodBG {
		barStyle = barStyle.Background(s.moodBG)
	}
	return barStyle.Render(bar)
}

func compactModelName(model string) string {
	model = strings.TrimSpace(model)
	model = strings.TrimPrefix(model, "x-ai/")
	model = strings.TrimPrefix(model, "google/")
	model = strings.TrimPrefix(model, "qwen/")
	return model
}

func formatDurationSeconds(ms int64) string {
	if ms <= 0 {
		return "0.0s"
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}

func throughputTokensPerSecond(tokens int, latencyMs int64) float64 {
	if tokens <= 0 || latencyMs <= 0 {
		return 0
	}
	return float64(tokens) / (float64(latencyMs) / 1000)
}

func joinStatusParts(parts []string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		filtered = append(filtered, part)
	}
	return strings.Join(filtered, " · ")
}

// SetMoodColor sets the background color tint for the status bar based on narrative mood.
// Pass an empty string to reset to the default theme color.
func (s *StatusBarModel) SetMoodColor(bg lipgloss.Color) {
	if bg == "" {
		s.hasMoodBG = false
		s.moodBG = ""
	} else {
		s.moodBG = bg
		s.hasMoodBG = true
	}
}
