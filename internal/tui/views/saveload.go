package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/storage"
	"github.com/crimsab/oneday/internal/tui/theme"
)

// SaveLoadSelectedMsg signals that a save was picked for loading.
type SaveLoadSelectedMsg struct {
	SaveID string
}

// SaveLoadCancelMsg signals that the user cancelled the load picker.
type SaveLoadCancelMsg struct{}

// SaveLoadModel shows a list of saves to pick from during gameplay.
type SaveLoadModel struct {
	saves    []storage.SaveSnapshot
	selected int
	width    int
	height   int
}

// NewSaveLoadModel creates the save picker view.
func NewSaveLoadModel(saves []storage.SaveSnapshot) SaveLoadModel {
	return SaveLoadModel{
		saves:    saves,
		selected: 0,
	}
}

func (m SaveLoadModel) Init() tea.Cmd {
	return nil
}

func (m SaveLoadModel) Update(msg tea.Msg) (SaveLoadModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.saves)-1 {
				m.selected++
			}
		case "enter":
			if len(m.saves) > 0 {
				saveID := m.saves[m.selected].ID
				return m, func() tea.Msg {
					return SaveLoadSelectedMsg{SaveID: saveID}
				}
			}
		case "esc":
			return m, func() tea.Msg {
				return SaveLoadCancelMsg{}
			}
		}
	}
	return m, nil
}

func (m SaveLoadModel) View() string {
	var sb strings.Builder

	sb.WriteString(theme.Title.Render("Load Save"))
	sb.WriteString("\n\n")

	if len(m.saves) == 0 {
		sb.WriteString(theme.MutedText.Render("  No saves found."))
		sb.WriteString("\n\n")
		sb.WriteString(theme.MutedText.Render("  esc  back to game"))
		content := sb.String()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	for i, save := range m.saves {
		cursor := "  "
		itemStyle := theme.UnselectedItem
		if i == m.selected {
			cursor = "> "
			itemStyle = theme.SelectedItem
		}

		// Format save info: name, turn, location, time
		lastPlayed := formatRelativeTime(save.CreatedAt)
		location := save.Location
		if location == "" {
			location = "Unknown"
		}
		line := fmt.Sprintf("%s%-24s  Turn %-4d  %-18s  %s",
			cursor,
			truncate(save.Name, 22),
			save.Turn,
			truncate(location, 16),
			lastPlayed,
		)
		sb.WriteString(itemStyle.Render(line))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(theme.MutedText.Render("  ↑↓ navigate · enter load · esc cancel"))

	content := sb.String()
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// SetSize updates the view dimensions.
func (m *SaveLoadModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}
