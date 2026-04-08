package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/tui/theme"
)

// AchievementDismissedMsg is sent when the achievement popup is dismissed.
type AchievementDismissedMsg struct{}

// achievementTickMsg is an internal tick for the auto-dismiss timer.
type achievementTickMsg struct{}

// AchievementPopupModel renders a brief rarity-colored achievement notification.
type AchievementPopupModel struct {
	name        string
	description string
	rarity      string
	category    string
	visible     bool
	width       int
	height      int
	showAt      time.Time
}

// NewAchievementPopup creates a new AchievementPopupModel.
func NewAchievementPopup() AchievementPopupModel {
	return AchievementPopupModel{}
}

// Show displays the achievement popup with the given data.
func (m *AchievementPopupModel) Show(name, description, rarity, category string) {
	m.name = name
	m.description = description
	m.rarity = rarity
	m.category = category
	m.visible = true
	m.showAt = time.Now()
}

// Visible returns whether the popup is currently shown.
func (m AchievementPopupModel) Visible() bool {
	return m.visible
}

// SetSize updates the terminal dimensions for centering.
func (m *AchievementPopupModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// AchievementAutoDismissCmd returns a Cmd that auto-dismisses the popup after 5 seconds.
func AchievementAutoDismissCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(_ time.Time) tea.Msg {
		return achievementTickMsg{}
	})
}

// Update handles messages for the achievement popup.
func (m AchievementPopupModel) Update(msg tea.Msg) (AchievementPopupModel, tea.Cmd) {
	if !m.visible {
		return m, nil
	}
	switch msg.(type) {
	case achievementTickMsg:
		m.visible = false
		return m, func() tea.Msg { return AchievementDismissedMsg{} }
	case tea.KeyMsg:
		m.visible = false
		return m, func() tea.Msg { return AchievementDismissedMsg{} }
	}
	return m, nil
}

// View renders the achievement popup centered on screen.
func (m AchievementPopupModel) View() string {
	if !m.visible {
		return ""
	}

	rarityColor := theme.RarityColor(m.rarity)
	innerW := 46

	// Build rarity stars.
	stars := rarityStars(m.rarity)
	rarityLabel := fmt.Sprintf("[%s · %s]", m.rarity, m.category)

	// Title line.
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(rarityColor)
	title := titleStyle.Render("★ ACHIEVEMENT UNLOCKED ★")

	// Name line.
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Text)
	nameLine := nameStyle.Render(truncate(fmt.Sprintf("%s  %s", stars, m.name), innerW))

	// Description — word-wrap if needed.
	descLines := achWrapText(m.description, innerW)
	var descRendered []string
	mutedStyle := lipgloss.NewStyle().Foreground(theme.Muted)
	for _, dl := range descLines {
		descRendered = append(descRendered, mutedStyle.Render(dl))
	}

	// Rarity/category badge.
	badgeStyle := lipgloss.NewStyle().Foreground(rarityColor).Italic(true)
	badge := badgeStyle.Render(rarityLabel)

	// Hint.
	hint := mutedStyle.Render("Press any key to dismiss...")

	separator := lipgloss.NewStyle().Foreground(theme.Muted).Render(strings.Repeat("─", innerW))

	innerParts := []string{title, separator, nameLine}
	innerParts = append(innerParts, descRendered...)
	innerParts = append(innerParts, badge, "", hint)

	inner := lipgloss.JoinVertical(lipgloss.Left, innerParts...)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(rarityColor).
		Padding(0, 1).
		Width(innerW + 2)

	box := boxStyle.Render(inner)

	// Position at top-third of screen.
	topOffset := m.height / 3
	if topOffset < 2 {
		topOffset = 2
	}

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Top,
		lipgloss.NewStyle().MarginTop(topOffset).Render(box),
	)
}

// rarityStars returns a star string for a given rarity.
func rarityStars(rarity string) string {
	switch rarity {
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

// truncate clips a string to maxLen runes.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}

// achWrapText wraps text to fit within maxWidth characters per line.
func achWrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 || text == "" {
		return []string{text}
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	current := words[0]
	for _, w := range words[1:] {
		if len(current)+1+len(w) <= maxWidth {
			current += " " + w
		} else {
			lines = append(lines, current)
			current = w
		}
	}
	lines = append(lines, current)
	return lines
}
