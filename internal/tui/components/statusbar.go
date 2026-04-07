package components

import (
	"fmt"

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
	Vitals  []Vital
	Model   string
	Latency int64 // milliseconds
}

// StatusBarModel renders the bottom status bar.
type StatusBarModel struct {
	data  StatusBarData
	width int
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
		var style lipgloss.Style
		if v.Max > 0 {
			pct := float64(v.Current) / float64(v.Max)
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
		vitals += style.Render(fmt.Sprintf("%s: %d/%d", v.Label, v.Current, v.Max))
	}

	// Right side: AI model + latency
	var aiInfo string
	if s.data.Model != "" {
		aiInfo = theme.MutedText.Render(fmt.Sprintf("%s · %dms", s.data.Model, s.data.Latency))
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

	return theme.StatusBar.Width(s.width).Render(bar)
}
