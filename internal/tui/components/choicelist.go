package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/tui/theme"
)

// ChoiceSelectedMsg is sent when the player picks a choice.
type ChoiceSelectedMsg struct {
	ID   int
	Text string
}

// ChoiceItem is a choice displayed in the list.
type ChoiceItem struct {
	ID           int
	Text         string
	Intent       string
	Risk         string
	Scope        string
	Certainty    string
	RelatedStats []string
}

// ChoiceListModel displays AI-suggested choices and handles selection.
type ChoiceListModel struct {
	choices []ChoiceItem
	cursor  int
	width   int
	mood    string
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

// SetMood updates the mood-aware accenting used by semantic badges.
func (c *ChoiceListModel) SetMood(mood string) {
	c.mood = mood
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
		selected := i == c.cursor
		if i == c.cursor {
			prefix = "▸ "
			style = theme.SelectedItem
		}
		s += fmt.Sprintf("%s%d. %s\n", prefix, ch.ID, style.Render(ch.Text))
		if meta := c.renderChoiceMeta(ch, selected); meta != "" {
			indent := strings.Repeat(" ", len(prefix)+len(fmt.Sprintf("%d", ch.ID))+2)
			s += indent + meta + "\n"
		}
	}
	return s
}

func (c ChoiceListModel) renderChoiceMeta(choice ChoiceItem, selected bool) string {
	if !choiceHasSemanticMetadata(choice) {
		return ""
	}

	palette := theme.GetMoodPalette(c.mood)
	var badges []string

	if choice.Intent != "" {
		badges = append(badges, renderChoiceBadge("intent:"+strings.ToLower(choice.Intent), palette.NarrativeAccent, selected))
	}
	if choice.Risk != "" {
		badges = append(badges, renderChoiceBadge("risk:"+strings.ToLower(choice.Risk), riskColor(choice.Risk), selected))
	}
	if choice.Certainty != "" {
		badges = append(badges, renderChoiceBadge("certainty:"+strings.ToLower(choice.Certainty), palette.Accent, selected))
	}
	if choice.Scope != "" {
		badges = append(badges, renderChoiceBadge("scope:"+strings.ToLower(choice.Scope), theme.Secondary, selected))
	}
	for _, stat := range choice.RelatedStats {
		if strings.TrimSpace(stat) == "" {
			continue
		}
		badges = append(badges, renderChoiceBadge(stat, theme.Highlight, selected))
	}

	return strings.Join(badges, " ")
}

func choiceHasSemanticMetadata(choice ChoiceItem) bool {
	return choice.Intent != "" ||
		choice.Risk != "" ||
		choice.Scope != "" ||
		choice.Certainty != "" ||
		len(choice.RelatedStats) > 0
}

func renderChoiceBadge(label string, color lipgloss.Color, selected bool) string {
	style := lipgloss.NewStyle().Foreground(color)
	if selected {
		style = style.Bold(true)
	}
	return style.Render("[" + label + "]")
}

func riskColor(risk string) lipgloss.Color {
	switch strings.ToLower(strings.TrimSpace(risk)) {
	case "low":
		return theme.Success
	case "medium":
		return theme.Accent
	case "high":
		return theme.Danger
	default:
		return theme.Muted
	}
}
