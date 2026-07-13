package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"

	"github.com/crimsab/oneday/internal/storage"
	"github.com/crimsab/oneday/internal/tui/theme"
)

type historyBrowserFocus int

const (
	historyFocusTimeline historyBrowserFocus = iota
	historyFocusSearch
)

type historyBrowserModel struct {
	title          string
	storyName      string
	width          int
	height         int
	visible        bool
	viewport       viewport.Model
	search         textinput.Model
	focus          historyBrowserFocus
	groups         []historySessionGroup
	filteredGroups []historySessionGroup
	expanded       map[string]bool
	headerLines    []int
	cursor         int
	matchCount     int
	boxX           int
	boxY           int
	boxWidth       int
	boxHeight      int
	viewportTop    int
}

type historySessionGroup struct {
	Session      storage.Session
	SessionID    string
	Entries      []storage.ChatMessage
	FirstMessage time.Time
	LastMessage  time.Time
}

func newHistoryBrowser(storyName string, messages []storage.ChatMessage, sessions []storage.Session, initialQuery string, width, height int) *historyBrowserModel {
	search := textinput.New()
	search.Placeholder = "Search by text, type, turn, npc, clue..."
	search.CharLimit = 120
	search.Prompt = "Search > "
	search.SetValue(strings.TrimSpace(initialQuery))
	search.Blur()

	vp := viewport.New(60, 12)
	vp.MouseWheelEnabled = true
	vp.MouseWheelDelta = 4

	browser := &historyBrowserModel{
		title:     "Story History",
		storyName: strings.TrimSpace(storyName),
		visible:   true,
		viewport:  vp,
		search:    search,
		focus:     historyFocusTimeline,
		groups:    buildHistoryGroups(messages, sessions),
		expanded:  make(map[string]bool),
	}

	for _, group := range browser.groups {
		if group.Session.EndedAt == nil {
			browser.expanded[group.SessionID] = true
		}
	}
	if len(browser.groups) > 0 {
		last := browser.groups[len(browser.groups)-1]
		browser.expanded[last.SessionID] = true
	}

	browser.SetSize(width, height)
	browser.applyFilter()
	if strings.TrimSpace(initialQuery) != "" {
		browser.focusSearch()
	}
	return browser
}

func (h *historyBrowserModel) Visible() bool {
	return h != nil && h.visible
}

func (h *historyBrowserModel) Close() {
	if h == nil {
		return
	}
	h.visible = false
}

func (h *historyBrowserModel) SetSize(width, height int) {
	if h == nil {
		return
	}
	h.width = width
	h.height = height

	h.boxWidth = historyMaxInt(78, int(float64(width)*0.84))
	if width > 0 && h.boxWidth > width-4 {
		h.boxWidth = width - 4
	}
	h.boxWidth = historyMaxInt(48, h.boxWidth)

	h.boxHeight = historyMaxInt(18, int(float64(height)*0.82))
	if height > 0 && h.boxHeight > height-2 {
		h.boxHeight = height - 2
	}
	h.boxHeight = historyMaxInt(12, h.boxHeight)

	h.boxX = historyMaxInt(0, (width-h.boxWidth)/2)
	h.boxY = historyMaxInt(0, (height-h.boxHeight)/2)

	innerWidth := h.boxWidth - 4
	if innerWidth < 20 {
		innerWidth = 20
	}
	h.search.Width = innerWidth - lipgloss.Width(h.search.Prompt)
	if h.search.Width < 16 {
		h.search.Width = 16
	}

	viewportHeight := h.boxHeight - 8
	if viewportHeight < 4 {
		viewportHeight = 4
	}
	h.viewport.Width = innerWidth
	h.viewport.Height = viewportHeight
	h.viewportTop = h.boxY + 6
	h.rebuildViewport()
}

func (h historyBrowserModel) Update(msg tea.Msg) (historyBrowserModel, tea.Cmd) {
	if !h.visible {
		return h, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h.SetSize(msg.Width, msg.Height)
		return h, nil

	case tea.MouseMsg:
		if h.isMouseInViewport(msg) {
			var cmd tea.Cmd
			h.viewport, cmd = h.viewport.Update(msg)
			h.refreshViewport(true)
			return h, cmd
		}
		return h, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "h":
			h.Close()
			return h, nil
		case "tab", "/":
			if h.focus == historyFocusSearch {
				h.blurSearch()
			} else {
				h.focusSearch()
			}
			return h, nil
		}

		if h.focus == historyFocusSearch {
			switch msg.String() {
			case "up":
				h.blurSearch()
				h.moveCursor(-1)
				return h, nil
			case "down":
				h.blurSearch()
				h.moveCursor(1)
				return h, nil
			case "enter":
				h.blurSearch()
				return h, nil
			}
			prev := h.search.Value()
			var cmd tea.Cmd
			h.search, cmd = h.search.Update(msg)
			if h.search.Value() != prev {
				h.applyFilter()
			}
			return h, cmd
		}

		switch msg.String() {
		case "j", "down":
			h.moveCursor(1)
			return h, nil
		case "k", "up":
			h.moveCursor(-1)
			return h, nil
		case "left":
			h.setCurrentExpanded(false)
			return h, nil
		case "right":
			h.setCurrentExpanded(true)
			return h, nil
		case "pgdown", "ctrl+f":
			h.viewport.ViewDown()
			h.refreshViewport(true)
			return h, nil
		case "pgup", "ctrl+b":
			h.viewport.ViewUp()
			h.refreshViewport(true)
			return h, nil
		case "g":
			h.cursor = 0
			h.refreshViewport(false)
			return h, nil
		case "G":
			if len(h.filteredGroups) > 0 {
				h.cursor = len(h.filteredGroups) - 1
			}
			h.refreshViewport(false)
			return h, nil
		case "enter", " ":
			h.toggleCurrentGroup()
			return h, nil
		case "[":
			h.setAllExpanded(false)
			return h, nil
		case "]":
			h.setAllExpanded(true)
			return h, nil
		default:
			if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 && !msg.Alt {
				h.focusSearch()
				prev := h.search.Value()
				var cmd tea.Cmd
				h.search, cmd = h.search.Update(msg)
				if h.search.Value() != prev {
					h.applyFilter()
				}
				return h, cmd
			}
		}
	}

	return h, nil
}

func (h historyBrowserModel) View() string {
	if !h.visible {
		return ""
	}

	searchLabel := theme.MutedText.Render("Find")
	if h.focus == historyFocusSearch {
		searchLabel = theme.SelectedItem.Render("Find")
	}

	storyLine := theme.MutedText.Render("Search, expand sessions, and inspect the full run.")
	if h.storyName != "" {
		storyLine = theme.MutedText.Render("Story: " + h.storyName)
	}

	summary := fmt.Sprintf("%d sessions · %d matching entries", len(h.filteredGroups), h.matchCount)
	if strings.TrimSpace(h.search.Value()) == "" {
		totalEntries := 0
		for _, group := range h.groups {
			totalEntries += len(group.Entries)
		}
		summary = fmt.Sprintf("%d sessions · %d entries", len(h.groups), totalEntries)
	}

	separator := theme.MutedText.Render(strings.Repeat("─", historyMaxInt(1, h.viewport.Width)))
	footer := theme.MutedText.Render("Mouse wheel scrolls only here · ↑/↓ or j/k select sessions · ←/→ fold · PgUp/PgDn scroll timeline · Tab search · Esc close")

	searchRow := lipgloss.JoinHorizontal(lipgloss.Left, searchLabel, "  ", h.search.View())
	inner := lipgloss.JoinVertical(lipgloss.Left,
		theme.Title.Render(h.title),
		storyLine,
		searchRow,
		theme.MutedText.Render(summary),
		separator,
		h.viewport.View(),
		separator,
		footer,
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Secondary).
		Padding(0, 1).
		Width(h.boxWidth).
		Render(inner)

	return lipgloss.Place(h.width, h.height, lipgloss.Center, lipgloss.Center, box)
}

func (h *historyBrowserModel) focusSearch() {
	h.focus = historyFocusSearch
	h.search.Focus()
}

func (h *historyBrowserModel) blurSearch() {
	h.focus = historyFocusTimeline
	h.search.Blur()
}

func (h *historyBrowserModel) moveCursor(delta int) {
	if len(h.filteredGroups) == 0 {
		h.cursor = 0
		h.refreshViewport(false)
		return
	}
	h.cursor += delta
	if h.cursor < 0 {
		h.cursor = 0
	}
	if h.cursor >= len(h.filteredGroups) {
		h.cursor = len(h.filteredGroups) - 1
	}
	h.refreshViewport(false)
}

func (h *historyBrowserModel) toggleCurrentGroup() {
	if len(h.filteredGroups) == 0 {
		return
	}
	group := h.filteredGroups[h.cursor]
	if strings.TrimSpace(h.search.Value()) == "" {
		h.expanded[group.SessionID] = !h.expanded[group.SessionID]
	}
	h.refreshViewport(false)
}

func (h *historyBrowserModel) setCurrentExpanded(expanded bool) {
	if len(h.filteredGroups) == 0 || strings.TrimSpace(h.search.Value()) != "" {
		return
	}
	group := h.filteredGroups[h.cursor]
	if h.expanded[group.SessionID] == expanded {
		return
	}
	h.expanded[group.SessionID] = expanded
	h.refreshViewport(false)
}

func (h *historyBrowserModel) setAllExpanded(expanded bool) {
	if len(h.groups) == 0 {
		return
	}
	for _, group := range h.groups {
		h.expanded[group.SessionID] = expanded
	}
	h.refreshViewport(false)
}

func (h *historyBrowserModel) isExpanded(group historySessionGroup) bool {
	if strings.TrimSpace(h.search.Value()) != "" {
		return true
	}
	return h.expanded[group.SessionID]
}

func (h *historyBrowserModel) applyFilter() {
	query := normalizeHistoryQuery(h.search.Value())
	h.filteredGroups = make([]historySessionGroup, 0, len(h.groups))
	h.matchCount = 0

	for _, group := range h.groups {
		filteredEntries := make([]storage.ChatMessage, 0, len(group.Entries))
		for _, entry := range group.Entries {
			if query == "" || historyMessageMatchesQuery(entry, query) {
				filteredEntries = append(filteredEntries, entry)
			}
		}
		if len(filteredEntries) == 0 {
			continue
		}
		group.Entries = filteredEntries
		h.filteredGroups = append(h.filteredGroups, group)
		h.matchCount += len(filteredEntries)
	}

	if len(h.filteredGroups) == 0 {
		h.cursor = 0
	} else if h.cursor >= len(h.filteredGroups) {
		h.cursor = len(h.filteredGroups) - 1
	}

	h.refreshViewport(false)
}

func (h *historyBrowserModel) rebuildViewport() {
	h.refreshViewport(false)
}

func (h *historyBrowserModel) refreshViewport(preserveScroll bool) {
	if h == nil {
		return
	}
	content, headerLines := h.renderViewportContent()
	currentOffset := h.viewport.YOffset
	h.headerLines = headerLines
	h.viewport.SetContent(content)
	if preserveScroll {
		h.viewport.SetYOffset(currentOffset)
		return
	}
	h.syncViewportToCursor()
}

func (h *historyBrowserModel) syncViewportToCursor() {
	if h == nil || len(h.headerLines) == 0 || h.cursor >= len(h.headerLines) {
		return
	}
	target := h.headerLines[h.cursor]
	if target < h.viewport.YOffset {
		h.viewport.SetYOffset(target)
		return
	}
	bottom := h.viewport.YOffset + h.viewport.Height - 1
	if target > bottom {
		h.viewport.SetYOffset(target - h.viewport.Height + 1)
	}
}

func (h *historyBrowserModel) renderViewportContent() (string, []int) {
	if len(h.filteredGroups) == 0 {
		return theme.MutedText.Render("No history entries matched the current search."), []int{0}
	}

	lines := make([]string, 0, 128)
	headerLines := make([]int, 0, len(h.filteredGroups))
	lineCount := 0
	contentWidth := historyMaxInt(24, h.viewport.Width-4)
	bodyWidth := historyMaxInt(20, contentWidth-8)

	for idx, group := range h.filteredGroups {
		headerLines = append(headerLines, lineCount)
		header := renderHistorySessionHeader(group, idx == h.cursor, h.isExpanded(group))
		lines = append(lines, header)
		lineCount++

		if h.isExpanded(group) {
			currentTurn := -1
			for _, entry := range group.Entries {
				if entry.Turn != currentTurn {
					turnLine := renderHistoryTurnDivider(entry.Turn, entry.CreatedAt)
					lines = append(lines, turnLine)
					lineCount++
					currentTurn = entry.Turn
				}

				rendered := renderHistoryMessage(entry, bodyWidth)
				parts := strings.Split(rendered, "\n")
				lines = append(lines, parts...)
				lineCount += len(parts)
			}
		}

		lines = append(lines, "")
		lineCount++
	}

	return strings.Join(lines, "\n"), headerLines
}

func (h historyBrowserModel) isMouseInViewport(msg tea.MouseMsg) bool {
	if !tea.MouseEvent(msg).IsWheel() {
		return false
	}
	xMin := h.boxX + 2
	xMax := h.boxX + h.boxWidth - 2
	yMin := h.viewportTop
	yMax := h.viewportTop + h.viewport.Height
	return msg.X >= xMin && msg.X < xMax && msg.Y >= yMin && msg.Y < yMax
}

func (m NarrativeModel) showHistory(args []string) (NarrativeModel, tea.Cmd) {
	query := strings.TrimSpace(strings.Join(args, " "))
	if m.narrator == nil || m.narrator.DB() == nil || m.narrator.Story() == nil {
		m.errMsg = "No story history available yet."
		return m, nil
	}

	storyID := m.narrator.Story().ID
	msgs, err := m.narrator.DB().GetStoryMessages(storyID)
	if err != nil {
		m.errMsg = fmt.Sprintf("History error: %v", err)
		return m, nil
	}

	sessions, err := m.narrator.DB().ListSessions(storyID)
	if err != nil {
		m.errMsg = fmt.Sprintf("History error: %v", err)
		return m, nil
	}

	m.historyBrowser = newHistoryBrowser(m.narrator.Story().Name, msgs, sessions, query, m.width, m.height)
	m.historyReturnInputFocus = m.inputFocus
	m.inputFocus = false
	m.input.Blur()
	return m, nil
}

func buildHistoryGroups(messages []storage.ChatMessage, sessions []storage.Session) []historySessionGroup {
	if len(messages) == 0 {
		return nil
	}

	sessionMap := make(map[string]storage.Session, len(sessions))
	for _, sess := range sessions {
		sessionMap[sess.ID] = sess
	}

	groupsByID := make(map[string]*historySessionGroup, len(sessions))
	order := make([]string, 0, len(sessions))

	for _, msg := range messages {
		if strings.TrimSpace(msg.Content) == "" {
			continue
		}
		group := groupsByID[msg.SessionID]
		if group == nil {
			group = &historySessionGroup{
				Session:   sessionMap[msg.SessionID],
				SessionID: msg.SessionID,
			}
			groupsByID[msg.SessionID] = group
			order = append(order, msg.SessionID)
		}
		group.Entries = append(group.Entries, msg)
		if group.FirstMessage.IsZero() || msg.CreatedAt.Before(group.FirstMessage) {
			group.FirstMessage = msg.CreatedAt
		}
		if group.LastMessage.IsZero() || msg.CreatedAt.After(group.LastMessage) {
			group.LastMessage = msg.CreatedAt
		}
	}

	groups := make([]historySessionGroup, 0, len(order))
	for _, sessionID := range order {
		group := groupsByID[sessionID]
		if group == nil || len(group.Entries) == 0 {
			continue
		}
		if group.Session.ID == "" {
			group.Session.ID = sessionID
		}
		groups = append(groups, *group)
	}

	sort.SliceStable(groups, func(i, j int) bool {
		left := groups[i].FirstMessage
		right := groups[j].FirstMessage
		switch {
		case left.IsZero() && right.IsZero():
			return groups[i].SessionID < groups[j].SessionID
		case left.IsZero():
			return false
		case right.IsZero():
			return true
		default:
			return left.Before(right)
		}
	})

	return groups
}

func normalizeHistoryQuery(query string) string {
	return strings.ToLower(strings.TrimSpace(query))
}

func historyMessageMatchesQuery(msg storage.ChatMessage, query string) bool {
	query = normalizeHistoryQuery(query)
	if query == "" {
		return true
	}
	haystack := []string{
		strings.ToLower(msg.Content),
		strings.ToLower(historyRoleLabel(msg.Role)),
		strings.ToLower(historyMessageTypeLabel(msg.MessageType)),
		fmt.Sprintf("turn %d", msg.Turn),
		msg.CreatedAt.Format("2006-01-02 15:04"),
	}
	joined := strings.Join(haystack, " ")
	return strings.Contains(joined, query)
}

func renderHistorySessionHeader(group historySessionGroup, selected, expanded bool) string {
	cursor := "  "
	if selected {
		cursor = "▸ "
	}

	fold := "▸"
	if expanded {
		fold = "▾"
	}

	label := "Session"
	if !group.Session.StartedAt.IsZero() {
		label = "Session · " + group.Session.StartedAt.Format("02 Jan 2006 15:04")
	}
	if group.Session.EndedAt == nil {
		label += " · active"
	}

	count := fmt.Sprintf("%d msgs", len(group.Entries))
	if len(group.Entries) == 1 {
		count = "1 msg"
	}

	style := lipgloss.NewStyle().Foreground(theme.Text)
	if selected {
		style = lipgloss.NewStyle().Foreground(theme.Highlight).Bold(true)
	}

	metaStyle := lipgloss.NewStyle().Foreground(theme.Muted)
	if selected {
		metaStyle = lipgloss.NewStyle().Foreground(theme.Accent)
	}

	line := fmt.Sprintf("%s%s %s", cursor, fold, style.Render(label))
	return line + "  " + metaStyle.Render(count)
}

func renderHistoryTurnDivider(turn int, createdAt time.Time) string {
	label := fmt.Sprintf("  Turn %d", turn)
	if !createdAt.IsZero() {
		label += " · " + createdAt.Format("15:04")
	}
	return lipgloss.NewStyle().Foreground(theme.Secondary).Render(label)
}

func renderHistoryMessage(msg storage.ChatMessage, wrapWidth int) string {
	roleBadge := historyRoleBadge(msg.Role)
	metaBits := []string{roleBadge}
	if typeBadge := historyTypeBadge(msg.MessageType); typeBadge != "" {
		metaBits = append(metaBits, typeBadge)
	}
	if !msg.CreatedAt.IsZero() {
		metaBits = append(metaBits, lipgloss.NewStyle().Foreground(theme.Muted).Render(msg.CreatedAt.Format("15:04")))
	}

	header := "    " + lipgloss.JoinHorizontal(lipgloss.Left, metaBits...)
	body := wrap.String(strings.TrimSpace(msg.Content), wrapWidth)
	bodyLines := strings.Split(body, "\n")
	for i, line := range bodyLines {
		bodyLines[i] = "      " + line
	}
	return header + "\n" + strings.Join(bodyLines, "\n")
}

func historyRoleBadge(role string) string {
	label := historyRoleLabel(role)
	style := lipgloss.NewStyle().
		Foreground(theme.Text).
		Background(lipgloss.Color("#2C2620")).
		Padding(0, 1).
		Bold(true)

	switch strings.ToLower(strings.TrimSpace(role)) {
	case "assistant":
		style = style.Foreground(theme.Primary).Background(lipgloss.Color("#241C16"))
	case "user":
		style = style.Foreground(theme.Success).Background(lipgloss.Color("#1E2A1E"))
	case "system":
		style = style.Foreground(theme.Secondary).Background(lipgloss.Color("#25211C"))
	}

	return style.Render(label)
}

func historyTypeBadge(messageType string) string {
	label := historyMessageTypeLabel(messageType)
	if label == "" {
		return ""
	}
	style := lipgloss.NewStyle().
		Foreground(theme.Muted).
		Background(lipgloss.Color("#202020")).
		Padding(0, 1)

	switch label {
	case "combat":
		style = style.Foreground(theme.CombatRed).Background(lipgloss.Color("#2A1C1C"))
	case "crafting":
		style = style.Foreground(theme.CraftingBlue).Background(lipgloss.Color("#1B2430"))
	case "meta":
		style = style.Foreground(theme.Accent).Background(lipgloss.Color("#2A251A"))
	case "dialogue":
		style = style.Foreground(theme.Highlight).Background(lipgloss.Color("#242018"))
	}

	return style.Render(label)
}

func historyRoleLabel(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "assistant":
		return "Narrator"
	case "user":
		return "Player"
	case "system":
		return "System"
	default:
		if role == "" {
			return "Unknown"
		}
		return strings.ToUpper(role[:1]) + role[1:]
	}
}

func historyMessageTypeLabel(messageType string) string {
	switch strings.ToLower(strings.TrimSpace(messageType)) {
	case "", "narrative":
		return ""
	case "combat":
		return "combat"
	case "crafting":
		return "crafting"
	case "dialogue":
		return "dialogue"
	case "narrator":
		return "meta"
	case "combat_summary":
		return "combat summary"
	default:
		return strings.TrimSpace(messageType)
	}
}

func historyMaxInt(values ...int) int {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, value := range values[1:] {
		if value > max {
			max = value
		}
	}
	return max
}
