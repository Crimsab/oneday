package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/tui/theme"
)

// HPBar renders a horizontal HP bar with label, current/max, and colored fill.
// Example output: "[Kael] HP: ██████░░░░ 62/100"
type HPBar struct {
	Label   string
	Current int
	Max     int
	Width   int // total width of the bar in characters
}

// NewHPBar creates an HP bar.
func NewHPBar(label string, current, max, width int) HPBar {
	if max <= 0 {
		max = 1
	}
	if current < 0 {
		current = 0
	}
	if current > max {
		current = max
	}
	return HPBar{
		Label:   label,
		Current: current,
		Max:     max,
		Width:   width,
	}
}

// View renders the HP bar.
// Format: "Label: [████████░░░░░░░░] 15/20"
func (h HPBar) View() string {
	if h.Max <= 0 {
		return h.Label + ": [?] 0/0"
	}

	pct := float64(h.Current) / float64(h.Max)

	// Calculate available bar width.
	// Fixed parts: "Label: [" + "]" + " XX/XX"
	counterStr := fmt.Sprintf(" %d/%d", h.Current, h.Max)
	// prefix: "Label: [", suffix: "]" + counterStr
	prefixLen := len(h.Label) + 3 // "Label: ["
	suffixLen := 1 + len(counterStr) // "]" + " 15/20"
	barWidth := h.Width - prefixLen - suffixLen
	if barWidth < 4 {
		barWidth = 4
	}

	// Calculate fill.
	fillCount := int(pct * float64(barWidth))
	if fillCount < 0 {
		fillCount = 0
	}
	if fillCount > barWidth {
		fillCount = barWidth
	}
	emptyCount := barWidth - fillCount

	// Choose color based on percentage.
	var fillStyle lipgloss.Style
	switch {
	case pct > 0.5:
		fillStyle = lipgloss.NewStyle().Foreground(theme.Success)
	case pct > 0.25:
		fillStyle = lipgloss.NewStyle().Foreground(theme.Accent)
	default:
		fillStyle = lipgloss.NewStyle().Foreground(theme.Danger)
	}
	emptyStyle := lipgloss.NewStyle().Foreground(theme.Muted)

	filled := fillStyle.Render(strings.Repeat("█", fillCount))
	empty := emptyStyle.Render(strings.Repeat("░", emptyCount))

	return fmt.Sprintf("%s: [%s%s]%s", h.Label, filled, empty, counterStr)
}
