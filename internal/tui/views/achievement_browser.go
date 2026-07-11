package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/storage"
	"github.com/crimsab/oneday/internal/tui/components"
	"github.com/crimsab/oneday/internal/tui/theme"
)

type achievementRowKind int

const (
	achievementRowStory achievementRowKind = iota
	achievementRowEntry
)

type achievementBrowserRow struct {
	kind             achievementRowKind
	storyIndex       int
	achievementIndex int
}

// AchievementBrowserBackMsg closes the home-surface achievement browser.
type AchievementBrowserBackMsg struct{}

// AchievementBrowserModel renders story-scoped achievement browsing with story accordions.
type AchievementBrowserModel struct {
	title       string
	stories     []engine.StoryArchiveSummary
	expanded    map[string]bool
	rows        []achievementBrowserRow
	selected    int
	width       int
	height      int
	visible     bool
	singleStory bool
	detail      components.OverlayModel
}

func NewAchievementBrowserModel(title string, stories []engine.StoryArchiveSummary, width, height int) AchievementBrowserModel {
	model := AchievementBrowserModel{
		title:    title,
		stories:  stories,
		expanded: map[string]bool{},
		visible:  true,
	}
	model.SetSize(width, height)
	model.rebuildRows()
	return model
}

func NewSingleStoryAchievementBrowser(story engine.StoryArchiveSummary, width, height int) AchievementBrowserModel {
	model := NewAchievementBrowserModel("Achievements", []engine.StoryArchiveSummary{story}, width, height)
	model.singleStory = true
	model.expanded[story.Story.ID] = true
	model.rebuildRows()
	return model
}

func (m *AchievementBrowserModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.detail.SetSize(width, height)
}

func (m AchievementBrowserModel) Visible() bool {
	return m.visible
}

func (m *AchievementBrowserModel) Close() {
	m.visible = false
}

func (m *AchievementBrowserModel) rebuildRows() {
	m.rows = m.rows[:0]
	for storyIndex, story := range m.stories {
		m.rows = append(m.rows, achievementBrowserRow{kind: achievementRowStory, storyIndex: storyIndex})
		expanded := m.singleStory || m.expanded[story.Story.ID]
		if !expanded {
			continue
		}
		for achievementIndex := range story.Achievements {
			m.rows = append(m.rows, achievementBrowserRow{
				kind:             achievementRowEntry,
				storyIndex:       storyIndex,
				achievementIndex: achievementIndex,
			})
		}
	}
	if len(m.rows) == 0 {
		m.selected = 0
		return
	}
	if m.selected >= len(m.rows) {
		m.selected = len(m.rows) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
}

func (m AchievementBrowserModel) selectedRow() (achievementBrowserRow, bool) {
	if len(m.rows) == 0 || m.selected < 0 || m.selected >= len(m.rows) {
		return achievementBrowserRow{}, false
	}
	return m.rows[m.selected], true
}

func (m AchievementBrowserModel) Update(msg tea.Msg) (AchievementBrowserModel, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyMsg:
		if m.detail.Visible() {
			var cmd tea.Cmd
			m.detail, cmd = m.detail.Update(msg)
			return m, cmd
		}
		switch msg.String() {
		case "esc":
			m.Close()
			return m, func() tea.Msg { return AchievementBrowserBackMsg{} }
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.rows)-1 {
				m.selected++
			}
		case "left", "h":
			if row, ok := m.selectedRow(); ok && row.kind == achievementRowStory && !m.singleStory {
				delete(m.expanded, m.stories[row.storyIndex].Story.ID)
				m.rebuildRows()
			}
		case "right", "l":
			if row, ok := m.selectedRow(); ok && row.kind == achievementRowStory {
				m.expanded[m.stories[row.storyIndex].Story.ID] = true
				m.rebuildRows()
			}
		case "enter", " ":
			row, ok := m.selectedRow()
			if !ok {
				return m, nil
			}
			if row.kind == achievementRowStory {
				if m.singleStory {
					return m, nil
				}
				storyID := m.stories[row.storyIndex].Story.ID
				if m.expanded[storyID] {
					delete(m.expanded, storyID)
				} else {
					m.expanded[storyID] = true
				}
				m.rebuildRows()
				return m, nil
			}

			story := m.stories[row.storyIndex]
			achievement := story.Achievements[row.achievementIndex]
			m.detail.Show(achievement.Name, formatAchievementDetail(story, achievement))
			return m, nil
		}
	}

	return m, nil
}

func (m AchievementBrowserModel) View() string {
	if !m.visible {
		return ""
	}
	if m.detail.Visible() {
		return m.detail.View()
	}

	boxWidth := maxInt(70, int(float64(m.width)*0.82))
	if m.width > 0 && boxWidth > m.width-4 {
		boxWidth = m.width - 4
	}
	boxHeight := maxInt(18, int(float64(m.height)*0.82))
	if m.height > 0 && boxHeight > m.height-2 {
		boxHeight = m.height - 2
	}

	var lines []string
	lines = append(lines, theme.Title.Render(m.title))
	lines = append(lines, "")

	if len(m.rows) == 0 {
		lines = append(lines, theme.MutedText.Render("No achievements unlocked yet."))
		lines = append(lines, "")
		lines = append(lines, theme.MutedText.Render("Esc close"))
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, boxStyle(boxWidth).Render(strings.Join(lines, "\n")))
	}

	for i, row := range m.rows {
		cursor := "  "
		style := theme.MutedText
		if i == m.selected {
			cursor = "▸ "
			style = theme.SelectedItem
		}

		story := m.stories[row.storyIndex]
		switch row.kind {
		case achievementRowStory:
			marker := "▸"
			if m.singleStory || m.expanded[story.Story.ID] {
				marker = "▾"
			}
			meta := []string{}
			if story.ProtagonistName != "" {
				meta = append(meta, story.ProtagonistName)
			}
			if story.CurrentLocation != "" {
				meta = append(meta, story.CurrentLocation)
			}
			if story.CurrentTurn > 0 {
				meta = append(meta, fmt.Sprintf("turn %d", story.CurrentTurn))
			}
			if story.Story.IsArchived {
				meta = append(meta, "archived")
			}
			label := fmt.Sprintf("%s %s", marker, story.Story.Name)
			if len(meta) > 0 {
				label += theme.MutedText.Render("  ·  " + strings.Join(meta, " · "))
			}
			label += theme.MutedText.Render(fmt.Sprintf("  (%d)", story.AchievementCount))
			lines = append(lines, style.Render(cursor+label))
		case achievementRowEntry:
			achievement := story.Achievements[row.achievementIndex]
			label := fmt.Sprintf("%s %s [%s]", achievementStars(achievement.Rarity), achievement.Name, strings.ToLower(achievement.Rarity))
			lines = append(lines, style.Render(cursor+"   "+label))
		}
	}

	lines = append(lines, "")
	if m.singleStory {
		lines = append(lines, theme.MutedText.Render("↑↓ navigate · Enter detail · Esc close"))
	} else {
		lines = append(lines, theme.MutedText.Render("↑↓ navigate · Enter expand/detail · ←/→ collapse/expand · Esc close"))
	}

	content := strings.Join(lines, "\n")
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, boxStyle(boxWidth).Height(boxHeight).Render(content))
}

func formatAchievementDetail(story engine.StoryArchiveSummary, achievement storage.Achievement) string {
	lines := []string{
		fmt.Sprintf("Story: %s", story.Story.Name),
		fmt.Sprintf("Category: %s", strings.Title(strings.ToLower(achievement.Category))),
		fmt.Sprintf("Rarity: %s", strings.Title(strings.ToLower(achievement.Rarity))),
	}
	if story.ProtagonistName != "" {
		lines = append(lines, fmt.Sprintf("Protagonist: %s", story.ProtagonistName))
	}
	if !achievement.EarnedAt.IsZero() {
		lines = append(lines, fmt.Sprintf("Earned: %s", achievement.EarnedAt.Format(time.RFC822)))
	}
	if achievement.Context != "" {
		lines = append(lines, "", "Context:", achievement.Context)
	}
	if achievement.Description != "" {
		lines = append(lines, "", "Description:", achievement.Description)
	}
	return strings.Join(lines, "\n")
}

func achievementStars(rarity string) string {
	switch strings.ToLower(strings.TrimSpace(rarity)) {
	case "uncommon":
		return "★★"
	case "rare":
		return "★★★"
	case "epic":
		return "★★★★"
	case "legendary":
		return "★★★★★"
	default:
		return "★"
	}
}

func boxStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Secondary).
		Padding(0, 1).
		Width(width)
}
