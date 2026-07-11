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

	barBG := lipgloss.Color("#2A2A2A")
	if s.hasMoodBG {
		barBG = s.moodBG
	}
	baseTextStyle := lipgloss.NewStyle().Background(barBG)

	// Left side: vitals
	var vitals string
	for i, v := range s.data.Vitals {
		if i > 0 {
			vitals += baseTextStyle.Render("  ")
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
				style = baseTextStyle.Copy().Foreground(theme.Danger)
			} else if pct <= 0.5 {
				style = baseTextStyle.Copy().Foreground(theme.Accent)
			} else {
				style = baseTextStyle.Copy().Foreground(theme.Text)
			}
		} else {
			style = baseTextStyle.Copy().Foreground(theme.Text)
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
		aiText := joinStatusParts(parts)
		aiInfo = baseTextStyle.Copy().Foreground(theme.Muted).Render(aiText)
	}

	barStyle := lipgloss.NewStyle().
		Foreground(theme.Text).
		Background(barBG).
		Padding(0, 1).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(theme.Secondary).
		Width(s.width)

	contentWidth := s.width - 2 // account for horizontal padding
	if contentWidth < 1 {
		contentWidth = 1
	}

	if aiInfo == "" {
		return barStyle.Render(vitals)
	}

	leftWidth := lipgloss.Width(vitals)
	rightWidth := lipgloss.Width(aiInfo)
	availableRight := contentWidth - leftWidth - 2
	if availableRight <= 0 {
		return barStyle.Render(vitals)
	}
	if rightWidth > availableRight {
		aiInfo = baseTextStyle.Copy().Foreground(theme.Muted).Render(truncateStatusText(joinStatusParts([]string{
			compactModelName(s.data.Model),
			formatDurationSeconds(s.data.Latency),
			func() string {
				if s.data.Streamed && s.data.TimeToFirstToken > 0 {
					return "ft " + formatDurationSeconds(s.data.TimeToFirstToken)
				}
				return ""
			}(),
			func() string {
				if s.data.TotalTokens > 0 {
					tokenPart := fmt.Sprintf("%dt", s.data.TotalTokens)
					if s.data.CompletionTokens > 0 || s.data.PromptTokens > 0 {
						tokenPart = fmt.Sprintf("%dt (%dp/%dc", s.data.TotalTokens, s.data.PromptTokens, s.data.CompletionTokens)
						if s.data.ReasoningTokens > 0 {
							tokenPart += fmt.Sprintf(", r%d", s.data.ReasoningTokens)
						}
						tokenPart += ")"
					}
					return tokenPart
				}
				return ""
			}(),
			func() string {
				if s.data.CachedPromptTokens > 0 {
					return fmt.Sprintf("cache %dp", s.data.CachedPromptTokens)
				}
				return ""
			}(),
			func() string {
				if rate := throughputTokensPerSecond(s.data.CompletionTokens, s.data.Latency); rate > 0 {
					return fmt.Sprintf("%.1ft/s", rate)
				}
				return ""
			}(),
			func() string {
				if s.data.CostUSD > 0 {
					return fmt.Sprintf("$%.5f", s.data.CostUSD)
				}
				return ""
			}(),
		}), availableRight))
		rightWidth = lipgloss.Width(aiInfo)
	}

	gap := contentWidth - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}
	spacer := baseTextStyle.Copy().Width(gap).Render("")
	bar := lipgloss.JoinHorizontal(lipgloss.Top, vitals, spacer, aiInfo)
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

func truncateStatusText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= maxWidth {
		return string(runes)
	}
	if maxWidth <= 1 {
		return string(runes[:maxWidth])
	}
	return string(runes[:maxWidth-1]) + "…"
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
