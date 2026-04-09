package views

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/tui/components"
	"github.com/crimsab/oneday/internal/tui/theme"
)

// MenuItem represents a main menu option.
type MenuItem struct {
	Label  string
	Action MenuAction
}

// MenuAction is what happens when a menu item is selected.
type MenuAction int

const (
	ActionNewStory MenuAction = iota
	ActionLoadStory
	ActionAchievementArchive
	ActionSettings
	ActionQuit
)

// MenuSelectedMsg is sent when the player picks a menu item.
type MenuSelectedMsg struct {
	Action MenuAction
}

// MenuModel is the main menu view.
type MenuModel struct {
	items  []MenuItem
	cursor int
	width  int
	height int
}

// NewMenuModel creates the main menu.
func NewMenuModel() MenuModel {
	return MenuModel{
		items: []MenuItem{
			{Label: "New Story", Action: ActionNewStory},
			{Label: "Load Story", Action: ActionLoadStory},
			{Label: "Achievements", Action: ActionAchievementArchive},
			{Label: "Settings", Action: ActionSettings},
			{Label: "Quit", Action: ActionQuit},
		},
	}
}

func (m MenuModel) Init() tea.Cmd { return nil }

func (m MenuModel) Update(msg tea.Msg) (MenuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter", " ":
			return m, func() tea.Msg {
				return MenuSelectedMsg{Action: m.items[m.cursor].Action}
			}
		case "q", "Q":
			return m, func() tea.Msg {
				return MenuSelectedMsg{Action: ActionQuit}
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m MenuModel) View() string {
	logo := components.Logo()

	var items string
	for i, item := range m.items {
		cursor := "  "
		style := theme.UnselectedItem
		if i == m.cursor {
			cursor = "▸ "
			style = theme.SelectedItem
		}
		items += fmt.Sprintf("%s%s\n", cursor, style.Render(item.Label))
	}

	help := theme.MutedText.Render("↑/↓ navigate • enter/space select • q quit")

	content := lipgloss.JoinVertical(lipgloss.Left,
		logo,
		"",
		items,
		"",
		help,
	)

	// Center in the terminal
	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

// SetSize updates the menu dimensions.
func (m *MenuModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}
