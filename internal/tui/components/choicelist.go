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

// ChoiceInspectRequestedMsg asks the parent view to explain the selected choice.
type ChoiceInspectRequestedMsg struct {
	ID int
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
	metaCursor int
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
	c.metaCursor = -1
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
			c.metaCursor = -1
		case "down", "j":
			if c.cursor < len(c.choices)-1 {
				c.cursor++
			}
			c.metaCursor = -1
		case "right", "l":
			if len(c.choices) > 0 {
				badges := choiceBadges(c.choices[c.cursor], c.mood)
				if len(badges) > 0 {
					if c.metaCursor < 0 {
						c.metaCursor = 0
					} else {
						c.metaCursor = (c.metaCursor + 1) % len(badges)
					}
				}
			}
			case "left":
			if len(c.choices) > 0 {
				badges := choiceBadges(c.choices[c.cursor], c.mood)
				if len(badges) > 0 {
					if c.metaCursor < 0 {
						c.metaCursor = len(badges) - 1
					} else if c.metaCursor == 0 {
						c.metaCursor = -1
					} else {
						c.metaCursor--
					}
				}
			}
		case "enter", " ":
			if len(c.choices) > 0 {
				selected := c.choices[c.cursor]
				if c.metaCursor >= 0 {
					return c, func() tea.Msg {
						return ChoiceInspectRequestedMsg{ID: selected.ID}
					}
				}
				return c, func() tea.Msg {
					return ChoiceSelectedMsg{
						ID:   selected.ID,
						Text: selected.Text,
					}
				}
			}
		case "?":
			if len(c.choices) > 0 {
				selected := c.choices[c.cursor]
				return c, func() tea.Msg {
					return ChoiceInspectRequestedMsg{ID: selected.ID}
				}
			}
		default:
			if len(msg.String()) == 1 {
				ch := msg.String()[0]
				if ch >= '1' && ch <= '9' {
					idx := int(ch - '0' - 1)
					if idx >= 0 && idx < len(c.choices) {
						selected := c.choices[idx]
						return c, func() tea.Msg {
							return ChoiceSelectedMsg{
								ID:   selected.ID,
								Text: selected.Text,
							}
						}
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
	badges := choiceBadges(choice, c.mood)
	if len(badges) == 0 {
		return ""
	}

	rendered := make([]string, 0, len(badges))
	for i, badge := range badges {
		focused := selected && c.metaCursor == i
		rendered = append(rendered, renderChoiceBadge(badge.label, badge.color, selected, focused))
	}

	return strings.Join(rendered, " ")
}

func choiceHasSemanticMetadata(choice ChoiceItem) bool {
	return choice.Intent != "" ||
		choice.Risk != "" ||
		choice.Scope != "" ||
		choice.Certainty != "" ||
		len(choice.RelatedStats) > 0
}

func renderChoiceBadge(label string, color lipgloss.Color, selected bool, focused bool) string {
	style := lipgloss.NewStyle().Foreground(color)
	if selected {
		style = style.Bold(true)
	}
	if focused {
		style = style.Underline(true).Reverse(true)
	}
	return style.Render("[" + label + "]")
}

type choiceBadge struct {
	label string
	color lipgloss.Color
}

func choiceBadges(choice ChoiceItem, mood string) []choiceBadge {
	if !choiceHasSemanticMetadata(choice) {
		return nil
	}

	palette := theme.GetMoodPalette(mood)
	var badges []choiceBadge

	if choice.Intent != "" {
		badges = append(badges, choiceBadge{label: "intent:" + strings.ToLower(choice.Intent), color: palette.NarrativeAccent})
	}
	if choice.Risk != "" {
		badges = append(badges, choiceBadge{label: "risk:" + strings.ToLower(choice.Risk), color: riskColor(choice.Risk)})
	}
	if choice.Certainty != "" {
		badges = append(badges, choiceBadge{label: "certainty:" + strings.ToLower(choice.Certainty), color: palette.Accent})
	}
	if choice.Scope != "" {
		badges = append(badges, choiceBadge{label: "scope:" + strings.ToLower(choice.Scope), color: theme.Secondary})
	}
	for _, stat := range choice.RelatedStats {
		if strings.TrimSpace(stat) == "" {
			continue
		}
		badges = append(badges, choiceBadge{label: stat, color: theme.Highlight})
	}

	return badges
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
