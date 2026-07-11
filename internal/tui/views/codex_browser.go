package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/tui/theme"
)

type codexBrowserFocus int

const (
	codexFocusRoot codexBrowserFocus = iota
	codexFocusDetail
	codexFocusLinks
)

// CodexBrowserBackMsg closes the codex browser.
type CodexBrowserBackMsg struct{}

// CodexBrowserModel renders a stacked codex / dossier inspector.
type CodexBrowserModel struct {
	title          string
	index          *engine.CodexIndex
	width          int
	height         int
	visible        bool
	categoryCursor int
	entryCursor    map[string]int
	stack          []string
	focus          codexBrowserFocus
	linkCursor     int
	viewport       viewport.Model
}

func NewCodexBrowserModel(title string, index *engine.CodexIndex, width, height int, initialCategory, initialEntryID string) *CodexBrowserModel {
	browser := &CodexBrowserModel{
		title:       title,
		index:       index,
		visible:     true,
		entryCursor: map[string]int{},
		focus:       codexFocusRoot,
		viewport:    viewport.New(40, 12),
	}
	browser.SetSize(width, height)
	if index != nil {
		for i, category := range index.Categories {
			if initialCategory != "" && strings.EqualFold(initialCategory, category.Key) {
				browser.categoryCursor = i
				break
			}
		}
		if initialEntryID != "" {
			browser.push(initialEntryID)
		}
	}
	return browser
}

func (m *CodexBrowserModel) SetSize(width, height int) {
	m.width = width
	m.height = height

	boxWidth := m.boxWidth()
	boxHeight := m.boxHeight()

	m.viewport.Width = boxWidth - 4
	if m.viewport.Width < 20 {
		m.viewport.Width = 20
	}
	m.viewport.Height = boxHeight - 11
	if m.viewport.Height < 5 {
		m.viewport.Height = 5
	}
	m.refreshViewport()
}

func (m CodexBrowserModel) Visible() bool {
	return m.visible
}

func (m *CodexBrowserModel) Close() {
	m.visible = false
}

func (m CodexBrowserModel) currentCategory() string {
	if m.index == nil || len(m.index.Categories) == 0 {
		return ""
	}
	if m.categoryCursor < 0 || m.categoryCursor >= len(m.index.Categories) {
		return m.index.Categories[0].Key
	}
	return m.index.Categories[m.categoryCursor].Key
}

func (m CodexBrowserModel) currentEntryIDs() []string {
	if m.index == nil {
		return nil
	}
	return m.index.CategoryEntries[m.currentCategory()]
}

func (m CodexBrowserModel) currentEntryID() string {
	ids := m.currentEntryIDs()
	if len(ids) == 0 {
		return ""
	}
	cursor := m.entryCursor[m.currentCategory()]
	if cursor < 0 || cursor >= len(ids) {
		cursor = 0
	}
	return ids[cursor]
}

func (m CodexBrowserModel) currentEntry() (engine.CodexEntry, bool) {
	if len(m.stack) == 0 {
		return engine.CodexEntry{}, false
	}
	return m.index.Entry(m.stack[len(m.stack)-1])
}

func (m *CodexBrowserModel) push(entryID string) {
	if m.index == nil {
		return
	}
	if _, ok := m.index.Entry(entryID); !ok {
		return
	}
	m.stack = append(m.stack, entryID)
	m.focus = codexFocusDetail
	m.linkCursor = 0
	m.refreshViewport()
}

func (m *CodexBrowserModel) pop() {
	if len(m.stack) == 0 {
		return
	}
	m.stack = m.stack[:len(m.stack)-1]
	m.linkCursor = 0
	if len(m.stack) == 0 {
		m.focus = codexFocusRoot
	} else {
		m.focus = codexFocusDetail
	}
	m.refreshViewport()
}

func (m *CodexBrowserModel) moveRoot(delta int) {
	ids := m.currentEntryIDs()
	if len(ids) == 0 {
		return
	}
	cursor := m.entryCursor[m.currentCategory()] + delta
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(ids) {
		cursor = len(ids) - 1
	}
	m.entryCursor[m.currentCategory()] = cursor
}

func (m *CodexBrowserModel) moveCategory(delta int) {
	if m.index == nil || len(m.index.Categories) == 0 {
		return
	}
	m.categoryCursor += delta
	if m.categoryCursor < 0 {
		m.categoryCursor = 0
	}
	if m.categoryCursor >= len(m.index.Categories) {
		m.categoryCursor = len(m.index.Categories) - 1
	}
}

func (m *CodexBrowserModel) refreshViewport() {
	entry, ok := m.currentEntry()
	if !ok {
		m.viewport.SetContent("")
		m.viewport.GotoTop()
		return
	}

	lines := []string{}
	if entry.Summary != "" {
		lines = append(lines, entry.Summary, "")
	}
	for _, section := range entry.Sections {
		lines = append(lines, theme.Title.Render(section.Title))
		for _, line := range section.Lines {
			lines = append(lines, "• "+line)
		}
		lines = append(lines, "")
	}
	m.viewport.SetContent(strings.TrimSpace(strings.Join(lines, "\n")))
	m.viewport.GotoTop()
}

func (m CodexBrowserModel) Update(msg tea.Msg) (CodexBrowserModel, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyMsg:
		switch {
		case len(m.stack) == 0:
			switch msg.String() {
			case "esc":
				m.Close()
				return m, func() tea.Msg { return CodexBrowserBackMsg{} }
			case "up", "k":
				m.moveRoot(-1)
			case "down", "j":
				m.moveRoot(1)
			case "left", "h":
				m.moveCategory(-1)
			case "right", "l":
				m.moveCategory(1)
			case "enter", " ":
				if entryID := m.currentEntryID(); entryID != "" {
					m.push(entryID)
				}
			}
			return m, nil
		default:
			entry, ok := m.currentEntry()
			if !ok {
				m.pop()
				return m, nil
			}
			switch msg.String() {
			case "esc":
				m.Close()
				return m, func() tea.Msg { return CodexBrowserBackMsg{} }
			case "backspace", "left", "h":
				m.pop()
				return m, nil
			case "tab":
				if len(entry.Related) == 0 {
					return m, nil
				}
				if m.focus == codexFocusLinks {
					m.focus = codexFocusDetail
				} else {
					m.focus = codexFocusLinks
				}
				return m, nil
			}

			if m.focus == codexFocusLinks {
				switch msg.String() {
				case "up", "k":
					if m.linkCursor > 0 {
						m.linkCursor--
					}
				case "down", "j":
					if m.linkCursor < len(entry.Related)-1 {
						m.linkCursor++
					}
				case "enter", " ":
					if len(entry.Related) > 0 {
						m.push(entry.Related[m.linkCursor].EntryID)
					}
				}
				return m, nil
			}

			switch msg.String() {
			case "up", "k", "pgup", "ctrl+b":
				var cmd tea.Cmd
				m.viewport, cmd = m.viewport.Update(msg)
				return m, cmd
			case "down", "j", "pgdown", "ctrl+f":
				var cmd tea.Cmd
				m.viewport, cmd = m.viewport.Update(msg)
				return m, cmd
			}
		}
	}

	return m, nil
}

func (m CodexBrowserModel) View() string {
	if !m.visible {
		return ""
	}

	boxWidth := m.boxWidth()
	boxHeight := m.boxHeight()

	if len(m.stack) == 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, codexBoxStyle(boxWidth).Height(boxHeight).Render(m.rootView()))
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, codexBoxStyle(boxWidth).Height(boxHeight).Render(m.detailView()))
}

func (m CodexBrowserModel) rootView() string {
	lines := []string{theme.Title.Render(m.title)}
	if m.index != nil && m.index.StoryName != "" {
		lines = append(lines, theme.MutedText.Render(m.index.StoryName))
	}
	lines = append(lines, "")

	if m.index == nil || len(m.index.Categories) == 0 {
		lines = append(lines, theme.MutedText.Render("No codex entries available yet."))
		lines = append(lines, "", theme.MutedText.Render("P projects · I investigations · F fronts · Esc close"))
		return strings.Join(lines, "\n")
	}

	categoryLabels := make([]string, 0, len(m.index.Categories))
	for i, category := range m.index.Categories {
		label := category.Label
		if i == m.categoryCursor {
			label = theme.SelectedItem.Render("[" + label + "]")
		} else {
			label = theme.MutedText.Render(label)
		}
		categoryLabels = append(categoryLabels, label)
	}
	lines = append(lines, strings.Join(categoryLabels, "  "))
	lines = append(lines, "")

	entryIDs := m.currentEntryIDs()
	if len(entryIDs) == 0 {
		lines = append(lines, theme.MutedText.Render("No entries in this category yet."))
	} else {
		cursor := m.entryCursor[m.currentCategory()]
		if cursor < 0 || cursor >= len(entryIDs) {
			cursor = 0
		}
		for i, entryID := range entryIDs {
			entry, ok := m.index.Entry(entryID)
			if !ok {
				continue
			}
			prefix := "  "
			style := theme.MutedText
			if i == cursor {
				prefix = "▸ "
				style = theme.SelectedItem
			}
			line := prefix + entry.Title
			if entry.Subtitle != "" {
				line += theme.MutedText.Render("  ·  " + entry.Subtitle)
			}
			lines = append(lines, style.Render(line))
			if entry.Summary != "" {
				lines = append(lines, theme.MutedText.Render("    "+truncatePlain(entry.Summary, m.viewport.Width)))
			}
		}
	}

	lines = append(lines, "", theme.MutedText.Render("↑↓ entries · ←→ categories · Enter open · P projects · I investigations · F fronts · Esc close"))
	return strings.Join(lines, "\n")
}

func (m CodexBrowserModel) detailView() string {
	entry, ok := m.currentEntry()
	if !ok {
		return ""
	}

	lines := []string{
		theme.Title.Render(m.title),
		theme.MutedText.Render(m.breadcrumb()),
		"",
		theme.SelectedItem.Render(entry.Title),
	}
	if entry.Subtitle != "" {
		lines = append(lines, theme.MutedText.Render(entry.Subtitle))
	}
	lines = append(lines, "", m.viewport.View())

	lines = append(lines, "")
	if len(entry.Related) > 0 {
		lines = append(lines, theme.Title.Render("Related"))
		for i, link := range entry.Related {
			prefix := "  "
			style := theme.MutedText
			if m.focus == codexFocusLinks && i == m.linkCursor {
				prefix = "▸ "
				style = theme.SelectedItem
			}
			lines = append(lines, style.Render(prefix+link.Label))
		}
		lines = append(lines, "")
	}

	hint := "↑↓ scroll · Tab links · Backspace back · Esc close"
	if len(entry.Related) == 0 {
		hint = "↑↓ scroll · Backspace back · Esc close"
	}
	hint += " · P projects · I investigations · F fronts"
	lines = append(lines, theme.MutedText.Render(hint))
	return strings.Join(lines, "\n")
}

func (m CodexBrowserModel) breadcrumb() string {
	if len(m.stack) == 0 || m.index == nil {
		return ""
	}
	parts := make([]string, 0, len(m.stack))
	for _, entryID := range m.stack {
		if entry, ok := m.index.Entry(entryID); ok {
			parts = append(parts, entry.Title)
		}
	}
	return strings.Join(parts, " → ")
}

func (m CodexBrowserModel) boxWidth() int {
	width := maxInt(78, int(float64(m.width)*0.84))
	if m.width > 0 && width > m.width-4 {
		width = m.width - 4
	}
	return maxInt(48, width)
}

func (m CodexBrowserModel) boxHeight() int {
	height := maxInt(18, int(float64(m.height)*0.82))
	if m.height > 0 && height > m.height-2 {
		height = m.height - 2
	}
	return maxInt(14, height)
}

func codexBoxStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Secondary).
		Padding(0, 1).
		Width(width)
}

func truncatePlain(text string, width int) string {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	if width <= 0 || len([]rune(text)) <= width {
		return text
	}
	if width < 2 {
		return text
	}
	return string([]rune(text)[:width-1]) + "…"
}
