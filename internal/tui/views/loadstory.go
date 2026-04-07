package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/storage"
	"github.com/crimsab/oneday/internal/tui/theme"
)

// StorySelectedMsg signals that a story was picked for loading.
type StorySelectedMsg struct {
	StoryID string
}

// LoadStoryBackMsg signals that the user pressed Esc to return to the menu.
type LoadStoryBackMsg struct{}

// LoadStoryModel shows a list of existing stories to continue.
type LoadStoryModel struct {
	stories  []storage.Story
	selected int
	width    int
	height   int
	errMsg   string
}

// NewLoadStoryModel creates the load story view with the given list of stories.
func NewLoadStoryModel(stories []storage.Story) LoadStoryModel {
	return LoadStoryModel{
		stories:  stories,
		selected: 0,
	}
}

func (m LoadStoryModel) Init() tea.Cmd {
	return nil
}

func (m LoadStoryModel) Update(msg tea.Msg) (LoadStoryModel, tea.Cmd) {
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
			if m.selected < len(m.stories)-1 {
				m.selected++
			}
		case "enter":
			if len(m.stories) > 0 {
				return m, func() tea.Msg {
					return StorySelectedMsg{StoryID: m.stories[m.selected].ID}
				}
			}
		case "esc":
			return m, func() tea.Msg {
				return LoadStoryBackMsg{}
			}
		}
	}
	return m, nil
}

func (m LoadStoryModel) View() string {
	var sb strings.Builder

	title := theme.Title.Render("Load Story")
	sb.WriteString(title)
	sb.WriteString("\n\n")

	if len(m.stories) == 0 {
		sb.WriteString(theme.MutedText.Render("  No stories found. Create a new one!"))
		sb.WriteString("\n\n")
		sb.WriteString(theme.MutedText.Render("  esc  back to menu"))
		content := sb.String()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}

	for i, story := range m.stories {
		cursor := "  "
		itemStyle := theme.MutedText
		if i == m.selected {
			cursor = "> "
			itemStyle = lipgloss.NewStyle().Bold(true).Foreground(theme.Primary)
		}

		// Format last played date
		lastPlayed := formatRelativeTime(story.UpdatedAt)

		// Extract genre from setting JSON (best-effort)
		genre := extractGenre(story.SettingJSON)

		line := fmt.Sprintf("%s%-30s  %s  %s", cursor, truncate(story.Name, 28), genre, lastPlayed)
		sb.WriteString(itemStyle.Render(line))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	if m.errMsg != "" {
		sb.WriteString(theme.DangerText.Render("  Error: " + m.errMsg))
		sb.WriteString("\n\n")
	}
	sb.WriteString(theme.MutedText.Render("  ↑↓ navigate · enter select · esc back"))

	content := sb.String()
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// SetSize updates the view dimensions.
func (m *LoadStoryModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// formatRelativeTime returns a human-friendly relative time string.
func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return "never played"
	}
	since := time.Since(t)
	switch {
	case since < time.Minute:
		return "just now"
	case since < time.Hour:
		mins := int(since.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case since < 24*time.Hour:
		hrs := int(since.Hours())
		if hrs == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hrs)
	case since < 7*24*time.Hour:
		days := int(since.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2, 2006")
	}
}

// extractGenre tries to pull the genre from a story's setting JSON.
func extractGenre(settingJSON string) string {
	// Simple string scan — avoids importing encoding/json for a single field.
	prefix := `"genre":"`
	idx := strings.Index(settingJSON, prefix)
	if idx < 0 {
		return ""
	}
	start := idx + len(prefix)
	end := strings.Index(settingJSON[start:], `"`)
	if end < 0 {
		return ""
	}
	genre := settingJSON[start : start+end]
	if len(genre) > 12 {
		genre = genre[:12]
	}
	return "[" + genre + "]"
}

// truncate shortens a string to maxLen runes, adding "…" if needed.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}
