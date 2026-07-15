package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	appi18n "github.com/crimsab/oneday/internal/i18n"
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
	loc           appi18n.Localizer
}

// NewSaveLoadModel creates the save picker view.
func NewSaveLoadModel(saves []storage.SaveSnapshot, localizers ...appi18n.Localizer) SaveLoadModel {
	return SaveLoadModel{
		saves:    saves,
		selected: 0,
		loc:      viewLocalizer(localizers),
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

	sb.WriteString(theme.Title.Render(m.loc.T("saves.title")))
	sb.WriteString("\n\n")

	if len(m.saves) == 0 {
		sb.WriteString(theme.MutedText.Render("  " + m.loc.T("saves.none")))
		sb.WriteString("\n\n")
		sb.WriteString(theme.MutedText.Render("  " + m.loc.T("saves.back")))
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
		lastPlayed := formatRelativeTimeLocalized(save.CreatedAt, m.loc)
		location := save.Location
		if location == "" {
			location = m.loc.T("common.unknown_label")
		}
		tag := ""
		if meta := save.Metadata(); meta != nil {
			switch {
			case strings.TrimSpace(meta.BranchLabel) != "":
				tag = "  ↺ " + truncate(meta.BranchLabel, 26)
			case strings.TrimSpace(meta.Kind) != "":
				tag = "  [" + meta.Kind + "]"
			}
		}
		line := fmt.Sprintf("%s%-24s  %s %-4d  %-18s  %s",
			cursor,
			truncate(save.Name, 22),
			m.loc.T("saves.turn"),
			save.Turn,
			truncate(location, 16),
			lastPlayed,
		)
		if tag != "" {
			line += tag
		}
		sb.WriteString(itemStyle.Render(line))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	if m.confirmDelete && len(m.saves) > 0 {
		sb.WriteString(theme.DangerText.Render("  " + m.loc.T("saves.delete_confirm")))
	} else {
		sb.WriteString(theme.MutedText.Render("  " + m.loc.T("saves.help")))
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
