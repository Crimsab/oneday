package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/tui/theme"
)

// SuggestionItem is a single autocomplete row.
type SuggestionItem struct {
	Value string
	Label string
	Hint  string
}

// SuggestionAcceptedMsg is emitted when a suggestion is accepted.
type SuggestionAcceptedMsg struct {
	Item SuggestionItem
}

// SuggestionDismissedMsg is emitted when the visible suggestion list is closed.
type SuggestionDismissedMsg struct{}

// SuggestionListModel renders a lightweight suggestion dropdown.
type SuggestionListModel struct {
	items   []SuggestionItem
	cursor  int
	width   int
	focused bool
}

// NewSuggestionList creates a new suggestion list model.
func NewSuggestionList() SuggestionListModel {
	return SuggestionListModel{}
}

// SetItems replaces the visible suggestions and resets navigation focus.
func (s *SuggestionListModel) SetItems(items []SuggestionItem) {
	s.items = items
	s.cursor = 0
	s.focused = false
}

// SetWidth sets the target render width.
func (s *SuggestionListModel) SetWidth(width int) {
	s.width = width
}

// HasItems reports whether suggestions are visible.
func (s SuggestionListModel) HasItems() bool {
	return len(s.items) > 0
}

// Focused reports whether keyboard navigation is currently attached to the list.
func (s SuggestionListModel) Focused() bool {
	return s.focused && len(s.items) > 0
}

// Update handles keyboard interaction for the suggestion list.
func (s SuggestionListModel) Update(msg tea.Msg) (SuggestionListModel, tea.Cmd) {
	if len(s.items) == 0 {
		return s, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			if !s.focused {
				s.focused = true
				return s, nil
			}
			if s.cursor > 0 {
				s.cursor--
			}
		case "down":
			if !s.focused {
				s.focused = true
			}
			if s.cursor < len(s.items)-1 {
				s.cursor++
			}
		case "tab":
			item := s.items[s.cursor]
			return s, func() tea.Msg {
				return SuggestionAcceptedMsg{Item: item}
			}
		case "enter":
			if !s.focused {
				return s, nil
			}
			item := s.items[s.cursor]
			return s, func() tea.Msg {
				return SuggestionAcceptedMsg{Item: item}
			}
		case "esc":
			return s, func() tea.Msg {
				return SuggestionDismissedMsg{}
			}
		}
	}

	return s, nil
}

// View renders the suggestion dropdown.
func (s SuggestionListModel) View() string {
	if len(s.items) == 0 {
		return ""
	}

	lines := make([]string, 0, len(s.items)+1)
	lines = append(lines, theme.MutedText.Render("Suggestions"))
	separatorWidth := 24
	if s.width > 0 && s.width-4 < separatorWidth {
		separatorWidth = s.width - 4
	}
	if separatorWidth < 8 {
		separatorWidth = 8
	}
	lines = append(lines, theme.MutedText.Render(strings.Repeat("─", separatorWidth)))
	limit := len(s.items)
	if limit > 6 {
		limit = 6
	}

	for i := 0; i < limit; i++ {
		item := s.items[i]
		prefix := "  "
		labelStyle := theme.UnselectedItem
		if i == s.cursor {
			if s.focused {
				prefix = "▸ "
				labelStyle = theme.SelectedItem
			} else {
				prefix = "• "
				labelStyle = theme.NormalText
			}
		}

		line := prefix + labelStyle.Render(strings.TrimSpace(item.Label))
		hint := strings.TrimSpace(item.Hint)
		if hint != "" {
			line += "  " + theme.MutedText.Render(hint)
		}
		lines = append(lines, line)
	}

	if len(s.items) > limit {
		lines = append(lines, theme.MutedText.Render("  …"))
	}

	lines = append(lines, theme.MutedText.Render("Tab complete · type to filter · Esc close"))
	content := strings.Join(lines, "\n")
	if s.width <= 0 {
		return lipgloss.NewStyle().PaddingLeft(1).Render(content)
	}

	return lipgloss.NewStyle().
		MaxWidth(s.width).
		PaddingLeft(1).
		Render(content)
}
