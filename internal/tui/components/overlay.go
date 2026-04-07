package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/tui/theme"
)

// OverlayDismissedMsg signals the overlay was closed.
type OverlayDismissedMsg struct{}

// OverlayModel renders a dismissable text overlay on top of the narrative view.
type OverlayModel struct {
	Title   string
	Content string
	visible bool
	width   int
	height  int
	scroll  int
	lines   []string // content split into lines
}

// NewOverlay creates a new OverlayModel.
func NewOverlay() OverlayModel {
	return OverlayModel{}
}

// Show makes the overlay visible with given title and content.
func (m *OverlayModel) Show(title, content string) {
	m.Title = title
	m.Content = content
	m.lines = strings.Split(content, "\n")
	m.scroll = 0
	m.visible = true
}

// Hide hides the overlay.
func (m *OverlayModel) Hide() {
	m.visible = false
}

// Visible returns whether the overlay is currently shown.
func (m OverlayModel) Visible() bool {
	return m.visible
}

// SetSize updates the overlay dimensions.
func (m *OverlayModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// Update handles key events for the overlay.
// When visible, it consumes Esc/Enter to dismiss, and up/down to scroll.
func (m OverlayModel) Update(msg tea.Msg) (OverlayModel, tea.Cmd) {
	if !m.visible {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "enter", " ":
			m.Hide()
			return m, func() tea.Msg { return OverlayDismissedMsg{} }
		case "up", "k":
			if m.scroll > 0 {
				m.scroll--
			}
		case "down", "j":
			maxScroll := m.maxScroll()
			if m.scroll < maxScroll {
				m.scroll++
			}
		}
	}
	return m, nil
}

// maxScroll returns the maximum scroll offset.
func (m OverlayModel) maxScroll() int {
	visibleLines := m.innerHeight()
	if len(m.lines) <= visibleLines {
		return 0
	}
	return len(m.lines) - visibleLines
}

// innerHeight returns the number of visible content lines.
func (m OverlayModel) innerHeight() int {
	h := int(float64(m.height) * 0.70)
	if h < 5 {
		h = 5
	}
	// Subtract: title(1) + footer(1) + borders(2) + padding(2)
	inner := h - 6
	if inner < 1 {
		inner = 1
	}
	return inner
}

// View renders the overlay.
func (m OverlayModel) View() string {
	if !m.visible {
		return ""
	}

	w := int(float64(m.width) * 0.60)
	if w < 40 {
		w = 40
	}
	if w > m.width-4 {
		w = m.width - 4
	}

	h := int(float64(m.height) * 0.70)
	if h < 8 {
		h = 8
	}
	if h > m.height-2 {
		h = m.height - 2
	}

	innerW := w - 4 // account for border+padding

	// Title line.
	titleLine := theme.Title.Render(m.Title)

	// Content lines with scroll.
	visibleLines := h - 6 // title(1) + sep(1) + footer(1) + borders(2) + gap(1)
	if visibleLines < 1 {
		visibleLines = 1
	}

	start := m.scroll
	end := start + visibleLines
	if end > len(m.lines) {
		end = len(m.lines)
	}

	var contentLines []string
	for _, line := range m.lines[start:end] {
		// Truncate lines that are too wide
		if len(line) > innerW {
			line = line[:innerW]
		}
		contentLines = append(contentLines, line)
	}

	// Pad to fill height
	for len(contentLines) < visibleLines {
		contentLines = append(contentLines, "")
	}

	contentBlock := strings.Join(contentLines, "\n")

	// Footer hint.
	hint := theme.MutedText.Render("Press Esc, Enter or Space to close")
	if m.maxScroll() > 0 {
		hint = theme.MutedText.Render("↑/↓ scroll · Esc/Enter close")
	}

	// Build the box content.
	separator := strings.Repeat("─", innerW)
	inner := lipgloss.JoinVertical(lipgloss.Left,
		titleLine,
		theme.MutedText.Render(separator),
		contentBlock,
		theme.MutedText.Render(separator),
		hint,
	)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Secondary).
		Padding(0, 1).
		Width(w)

	box := boxStyle.Render(inner)

	// Center the box in the terminal.
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}
