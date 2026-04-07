package components

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crimsab/oneday/internal/tui/theme"
)

// ChoiceSelectedMsg is sent when the player picks a choice.
type ChoiceSelectedMsg struct {
	ID   int
	Text string
}

// ChoiceItem is a choice displayed in the list.
type ChoiceItem struct {
	ID   int
	Text string
}

// ChoiceListModel displays AI-suggested choices and handles selection.
type ChoiceListModel struct {
	choices []ChoiceItem
	cursor  int
	width   int
}

// NewChoiceList creates a choice list.
func NewChoiceList() ChoiceListModel {
	return ChoiceListModel{}
}

// SetChoices updates the available choices and resets the cursor.
func (c *ChoiceListModel) SetChoices(items []ChoiceItem) {
	c.choices = items
	c.cursor = 0
}

// SetWidth sets the component width.
func (c *ChoiceListModel) SetWidth(w int) {
	c.width = w
}

// HasChoices returns true if there are choices to display.
func (c ChoiceListModel) HasChoices() bool {
	return len(c.choices) > 0
}

// Update handles key navigation within the choice list.
func (c ChoiceListModel) Update(msg tea.Msg) (ChoiceListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if c.cursor > 0 {
				c.cursor--
			}
		case "down", "j":
			if c.cursor < len(c.choices)-1 {
				c.cursor++
			}
		case "1", "2", "3", "4", "5":
			idx := int(msg.String()[0]-'0') - 1
			if idx >= 0 && idx < len(c.choices) {
				selected := c.choices[idx]
				return c, func() tea.Msg {
					return ChoiceSelectedMsg{
						ID:   selected.ID,
						Text: selected.Text,
					}
				}
			}
		case "enter":
			if len(c.choices) > 0 {
				selected := c.choices[c.cursor]
				return c, func() tea.Msg {
					return ChoiceSelectedMsg{
						ID:   selected.ID,
						Text: selected.Text,
					}
				}
			}
		}
	}
	return c, nil
}

// View renders the choice list.
func (c ChoiceListModel) View() string {
	if len(c.choices) == 0 {
		return ""
	}

	var s string
	for i, ch := range c.choices {
		prefix := "  "
		style := theme.UnselectedItem
		if i == c.cursor {
			prefix = "▸ "
			style = theme.SelectedItem
		}
		s += fmt.Sprintf("%s%d. %s\n", prefix, ch.ID, style.Render(ch.Text))
	}
	return s
}
