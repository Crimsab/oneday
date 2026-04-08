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

// SaveLoadDeleteMsg requests deletion of the selected save.
type SaveLoadDeleteMsg struct {
	SaveID string
}

// SaveLoadCancelMsg signals that the user cancelled the load picker.
type SaveLoadCancelMsg struct{}

// SaveLoadModel shows a list of saves to pick from during gameplay.
type SaveLoadModel struct {
	saves         []storage.SaveSnapshot
	selected      int
	width         int
	height        int
	confirmDelete bool
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
			if m.confirmDelete {
				return m, nil
			}
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.confirmDelete {
				return m, nil
			}
			if m.selected < len(m.saves)-1 {
				m.selected++
			}
		case "enter", " ":
			if len(m.saves) > 0 {
				if m.confirmDelete {
					saveID := m.saves[m.selected].ID
					m.confirmDelete = false
					return m, func() tea.Msg {
						return SaveLoadDeleteMsg{SaveID: saveID}
					}
				}
				saveID := m.saves[m.selected].ID
				return m, func() tea.Msg {
					return SaveLoadSelectedMsg{SaveID: saveID}
				}
			}
		case "x", "delete", "backspace":
			if len(m.saves) > 0 {
				m.confirmDelete = true
			}
		case "esc":
			if m.confirmDelete {
				m.confirmDelete = false
				return m, nil
			}
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
	if m.confirmDelete && len(m.saves) > 0 {
		sb.WriteString(theme.DangerText.Render("  Delete this save? Enter/Space confirm · Esc cancel"))
	} else {
		sb.WriteString(theme.MutedText.Render("  ↑↓ navigate · enter/space load · x delete · esc cancel"))
	}

	content := sb.String()
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// SetSize updates the view dimensions.
func (m *SaveLoadModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetSaves refreshes the save list while preserving valid cursor state.
func (m *SaveLoadModel) SetSaves(saves []storage.SaveSnapshot) {
	m.saves = saves
	if len(m.saves) == 0 {
		m.selected = 0
		m.confirmDelete = false
		return
	}
	if m.selected >= len(m.saves) {
		m.selected = len(m.saves) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
	m.confirmDelete = false
}
