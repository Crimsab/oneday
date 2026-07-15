package views

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/engine"
	appi18n "github.com/crimsab/oneday/internal/i18n"
	"github.com/crimsab/oneday/internal/tui/components"
	"github.com/crimsab/oneday/internal/tui/theme"
)

type frontTrackerRowKind int

const (
	frontTrackerRowSection frontTrackerRowKind = iota
	frontTrackerRowHook
	frontTrackerRowFront
	frontTrackerRowHotspot
	frontTrackerRowReaction
)

type frontTrackerRow struct {
	kind          frontTrackerRowKind
	label         string
	hookIndex     int
	frontIndex    int
	hotspotIndex  int
	reactionIndex int
}

// FrontTrackerBackMsg closes the tracker workspace.
type FrontTrackerBackMsg struct{}

// FrontTrackerModel renders hooks, visible fronts, pressure hotspots, and fallout in one workspace.
type FrontTrackerModel struct {
	loc      appi18n.Localizer
	title    string
	board    engine.FrontTrackerBoard
	rows     []frontTrackerRow
	selected int
	width    int
	height   int
	visible  bool
	detail   components.OverlayModel
}

func NewFrontTrackerModel(title string, board engine.FrontTrackerBoard, width, height int, localizers ...appi18n.Localizer) FrontTrackerModel {
	loc := viewLocalizer(localizers)
	model := FrontTrackerModel{
		title:   title,
		loc:     loc,
		board:   normalizedFrontTrackerBoard(board),
		visible: true,
		detail:  components.NewOverlay(loc),
	}
	model.SetSize(width, height)
	model.rebuildRows()
	return model
}

func (m *FrontTrackerModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.detail.SetSize(width, height)
}

func (m FrontTrackerModel) Visible() bool {
	return m.visible
}

func (m *FrontTrackerModel) Close() {
	m.visible = false
}

func (m *FrontTrackerModel) rebuildRows() {
	m.rows = m.rows[:0]

	if len(m.board.Hooks) > 0 {
		m.rows = append(m.rows, frontTrackerRow{kind: frontTrackerRowSection, label: m.loc.T("front.open_hooks")})
		for idx := range m.board.Hooks {
			m.rows = append(m.rows, frontTrackerRow{kind: frontTrackerRowHook, hookIndex: idx})
		}
	}

	activeFronts := activeFrontTrackerIndexes(m.board.Fronts)
	if len(activeFronts) > 0 {
		m.rows = append(m.rows, frontTrackerRow{kind: frontTrackerRowSection, label: m.loc.T("front.active_fronts")})
		for _, idx := range activeFronts {
			m.rows = append(m.rows, frontTrackerRow{kind: frontTrackerRowFront, frontIndex: idx})
		}
	}

	resolvedFronts := resolvedFrontTrackerIndexes(m.board.Fronts)
	if len(resolvedFronts) > 0 {
		m.rows = append(m.rows, frontTrackerRow{kind: frontTrackerRowSection, label: m.loc.T("front.resolved_fronts")})
		for _, idx := range resolvedFronts {
			m.rows = append(m.rows, frontTrackerRow{kind: frontTrackerRowFront, frontIndex: idx})
		}
	}

	if len(m.board.Hotspots) > 0 {
		m.rows = append(m.rows, frontTrackerRow{kind: frontTrackerRowSection, label: m.loc.T("front.pressure_hotspots")})
		for idx := range m.board.Hotspots {
			m.rows = append(m.rows, frontTrackerRow{kind: frontTrackerRowHotspot, hotspotIndex: idx})
		}
	}

	if len(m.board.Reactions) > 0 {
		m.rows = append(m.rows, frontTrackerRow{kind: frontTrackerRowSection, label: m.loc.T("front.recent_fallout")})
		for idx := range m.board.Reactions {
			m.rows = append(m.rows, frontTrackerRow{kind: frontTrackerRowReaction, reactionIndex: idx})
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

func (m FrontTrackerModel) selectedRow() (frontTrackerRow, bool) {
	if len(m.rows) == 0 || m.selected < 0 || m.selected >= len(m.rows) {
		return frontTrackerRow{}, false
	}
	return m.rows[m.selected], true
}

func (m FrontTrackerModel) Update(msg tea.Msg) (FrontTrackerModel, tea.Cmd) {
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
			return m, func() tea.Msg { return FrontTrackerBackMsg{} }
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.rows)-1 {
				m.selected++
			}
		case "enter", " ":
			row, ok := m.selectedRow()
			if !ok {
				return m, nil
			}
			switch row.kind {
			case frontTrackerRowHook:
				hook := m.board.Hooks[row.hookIndex]
				m.detail.Show(hook.Title, formatTrackerHookDetail(hook, m.loc))
			case frontTrackerRowFront:
				front := m.board.Fronts[row.frontIndex]
				m.detail.Show(front.Title, formatTrackerFrontDetail(front, m.loc))
			case frontTrackerRowHotspot:
				hotspot := m.board.Hotspots[row.hotspotIndex]
				title := hotspot.Region
				if kind := strings.TrimSpace(hotspot.Kind); kind != "" {
					title += " · " + kind
				}
				m.detail.Show(title, formatTrackerHotspotDetail(hotspot, m.loc))
			case frontTrackerRowReaction:
				reaction := m.board.Reactions[row.reactionIndex]
				m.detail.Show(reaction.Title, formatTrackerReactionDetail(reaction, m.loc))
			}
			return m, nil
		}
	}

	return m, nil
}

func (m FrontTrackerModel) View() string {
	if !m.visible {
		return ""
	}
	if m.detail.Visible() {
		return m.detail.View()
	}

	boxWidth := maxInt(76, int(float64(m.width)*0.84))
	if m.width > 0 && boxWidth > m.width-4 {
		boxWidth = m.width - 4
	}
	boxHeight := maxInt(18, int(float64(m.height)*0.82))
	if m.height > 0 && boxHeight > m.height-2 {
		boxHeight = m.height - 2
	}

	lines := []string{theme.Title.Render(m.title)}
	lines = append(lines, theme.MutedText.Render(frontTrackerWorkspaceSummary(m.board, m.loc)))
	lines = append(lines, "")

	if len(m.rows) == 0 {
		lines = append(lines, theme.MutedText.Render(m.loc.T("front.empty")))
		lines = append(lines, "")
		lines = append(lines, theme.MutedText.Render(m.loc.T("browser.close")))
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, boxStyle(boxWidth).Render(strings.Join(lines, "\n")))
	}

	rowWidth := boxWidth - 6
	if rowWidth < 30 {
		rowWidth = 30
	}

	for i, row := range m.rows {
		switch row.kind {
		case frontTrackerRowSection:
			lines = append(lines, theme.Title.Render(row.label))
		case frontTrackerRowHook:
			hook := m.board.Hooks[row.hookIndex]
			cursor := "  "
			style := theme.MutedText
			if i == m.selected {
				cursor = "▸ "
				style = theme.SelectedItem
			}
			lines = append(lines, style.Render(cursor+trackerHookRowTitle(hook, rowWidth, m.loc)))
			lines = append(lines, theme.MutedText.Render("    "+truncatePlain(trackerHookRowSubtitle(hook, m.loc), rowWidth)))
		case frontTrackerRowFront:
			front := m.board.Fronts[row.frontIndex]
			cursor := "  "
			style := theme.MutedText
			if i == m.selected {
				cursor = "▸ "
				style = theme.SelectedItem
			}
			lines = append(lines, style.Render(cursor+trackerFrontRowTitle(front, rowWidth)))
			lines = append(lines, theme.MutedText.Render("    "+truncatePlain(trackerFrontRowSubtitle(front, m.loc), rowWidth)))
		case frontTrackerRowHotspot:
			hotspot := m.board.Hotspots[row.hotspotIndex]
			cursor := "  "
			style := theme.MutedText
			if i == m.selected {
				cursor = "▸ "
				style = theme.SelectedItem
			}
			lines = append(lines, style.Render(cursor+trackerHotspotRowTitle(hotspot, rowWidth)))
			lines = append(lines, theme.MutedText.Render("    "+truncatePlain(trackerHotspotRowSubtitle(hotspot, m.loc), rowWidth)))
		case frontTrackerRowReaction:
			reaction := m.board.Reactions[row.reactionIndex]
			cursor := "  "
			style := theme.MutedText
			if i == m.selected {
				cursor = "▸ "
				style = theme.SelectedItem
			}
			lines = append(lines, style.Render(cursor+trackerReactionRowTitle(reaction, rowWidth)))
			lines = append(lines, theme.MutedText.Render("    "+truncatePlain(trackerReactionRowSubtitle(reaction, m.loc), rowWidth)))
		}
	}

	lines = append(lines, "")
	lines = append(lines, theme.MutedText.Render(m.loc.T("front.navigation")))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, boxStyle(boxWidth).Height(boxHeight).Render(strings.Join(lines, "\n")))
}

func normalizedFrontTrackerBoard(board engine.FrontTrackerBoard) engine.FrontTrackerBoard {
	normalized := engine.FrontTrackerBoard{
		Hooks:     append([]engine.StoryHook(nil), board.Hooks...),
		Fronts:    append([]engine.FrontTrackerFront(nil), board.Fronts...),
		Hotspots:  append([]engine.FrontTrackerPressure(nil), board.Hotspots...),
		Reactions: append([]engine.WorldReaction(nil), board.Reactions...),
	}
	sort.SliceStable(normalized.Hooks, func(i, j int) bool {
		if normalized.Hooks[i].UpdatedTurn != normalized.Hooks[j].UpdatedTurn {
			return normalized.Hooks[i].UpdatedTurn > normalized.Hooks[j].UpdatedTurn
		}
		return normalized.Hooks[i].Title < normalized.Hooks[j].Title
	})
	sort.SliceStable(normalized.Fronts, func(i, j int) bool {
		left := trackerFrontRowOrder(normalized.Fronts[i].Status)
		right := trackerFrontRowOrder(normalized.Fronts[j].Status)
		if left != right {
			return left < right
		}
		if normalized.Fronts[i].NextEscalationTurn != normalized.Fronts[j].NextEscalationTurn {
			switch {
			case normalized.Fronts[i].NextEscalationTurn == 0:
				return false
			case normalized.Fronts[j].NextEscalationTurn == 0:
				return true
			default:
				return normalized.Fronts[i].NextEscalationTurn < normalized.Fronts[j].NextEscalationTurn
			}
		}
		if normalized.Fronts[i].LastAdvancedTurn != normalized.Fronts[j].LastAdvancedTurn {
			return normalized.Fronts[i].LastAdvancedTurn > normalized.Fronts[j].LastAdvancedTurn
		}
		return normalized.Fronts[i].Title < normalized.Fronts[j].Title
	})
	sort.SliceStable(normalized.Hotspots, func(i, j int) bool {
		if normalized.Hotspots[i].Level != normalized.Hotspots[j].Level {
			return normalized.Hotspots[i].Level > normalized.Hotspots[j].Level
		}
		if normalized.Hotspots[i].UpdatedTurn != normalized.Hotspots[j].UpdatedTurn {
			return normalized.Hotspots[i].UpdatedTurn > normalized.Hotspots[j].UpdatedTurn
		}
		return normalized.Hotspots[i].Region < normalized.Hotspots[j].Region
	})
	sort.SliceStable(normalized.Reactions, func(i, j int) bool {
		if normalized.Reactions[i].CreatedTurn != normalized.Reactions[j].CreatedTurn {
			return normalized.Reactions[i].CreatedTurn > normalized.Reactions[j].CreatedTurn
		}
		return normalized.Reactions[i].Title < normalized.Reactions[j].Title
	})
	return normalized
}

func activeFrontTrackerIndexes(fronts []engine.FrontTrackerFront) []int {
	var indexes []int
	for i, front := range fronts {
		if !strings.EqualFold(front.Status, "resolved") {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func resolvedFrontTrackerIndexes(fronts []engine.FrontTrackerFront) []int {
	var indexes []int
	for i, front := range fronts {
		if strings.EqualFold(front.Status, "resolved") {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func trackerFrontRowOrder(status string) int {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active":
		return 0
	case "stalled":
		return 1
	case "resolved":
		return 2
	default:
		return 3
	}
}

func frontTrackerWorkspaceSummary(board engine.FrontTrackerBoard, localizers ...appi18n.Localizer) string {
	loc := viewLocalizer(localizers)
	return strings.Join([]string{
		loc.T("front.hooks_count", len(board.Hooks)),
		loc.T("front.fronts_count", len(board.Fronts)),
		loc.T("front.hotspots_count", len(board.Hotspots)),
		loc.T("front.fallout_count", len(board.Reactions)),
	}, " · ")
}

func trackerHookRowTitle(hook engine.StoryHook, width int, localizers ...appi18n.Localizer) string {
	loc := viewLocalizer(localizers)
	parts := []string{hook.Title}
	if kind := strings.TrimSpace(hook.Kind); kind != "" {
		parts = append(parts, kind)
	}
	if hook.TimerTurns > 0 {
		parts = append(parts, loc.T("front.timer_short", hook.TimerTurns))
	}
	return truncatePlain(strings.Join(parts, "  ·  "), width)
}

func trackerHookRowSubtitle(hook engine.StoryHook, localizers ...appi18n.Localizer) string {
	loc := viewLocalizer(localizers)
	parts := []string{}
	if npc := strings.TrimSpace(hook.NPCName); npc != "" {
		parts = append(parts, loc.T("front.npc_value", npc))
	}
	if detail := strings.TrimSpace(hook.Detail); detail != "" {
		parts = append(parts, detail)
	}
	if len(parts) == 0 {
		return loc.T("front.no_extra_details")
	}
	return strings.Join(parts, " ")
}

func trackerFrontRowTitle(front engine.FrontTrackerFront, width int) string {
	parts := []string{front.Title}
	if visibility := strings.TrimSpace(front.Visibility); visibility != "" {
		parts = append(parts, visibility)
	}
	if front.Segments > 0 {
		parts = append(parts, fmt.Sprintf("%d/%d", front.Progress, front.Segments))
	}
	if status := strings.TrimSpace(front.Status); status != "" && !strings.EqualFold(status, "active") {
		parts = append(parts, status)
	}
	return truncatePlain(strings.Join(parts, "  ·  "), width)
}

func trackerFrontRowSubtitle(front engine.FrontTrackerFront, localizers ...appi18n.Localizer) string {
	loc := viewLocalizer(localizers)
	parts := []string{}
	if faction := strings.TrimSpace(front.Faction); faction != "" {
		parts = append(parts, faction)
	}
	if stakes := strings.TrimSpace(front.Stakes); stakes != "" {
		parts = append(parts, stakes)
	} else if len(front.Pressures) > 0 {
		parts = append(parts, front.Pressures[0].Summary)
	}
	if strings.EqualFold(front.Status, "resolved") && strings.TrimSpace(front.Resolution) != "" {
		parts = append(parts, loc.T("front.outcome_value", strings.TrimSpace(front.Resolution)))
	}
	if len(parts) == 0 {
		return loc.T("front.no_details")
	}
	return strings.Join(parts, " ")
}

func trackerHotspotRowTitle(hotspot engine.FrontTrackerPressure, width int) string {
	parts := []string{hotspot.Region}
	if kind := strings.TrimSpace(hotspot.Kind); kind != "" {
		parts = append(parts, kind)
	}
	if hotspot.Level > 0 {
		parts = append(parts, fmt.Sprintf("%d", hotspot.Level))
	}
	if severity := strings.TrimSpace(hotspot.Severity); severity != "" {
		parts = append(parts, severity)
	}
	return truncatePlain(strings.Join(parts, "  ·  "), width)
}

func trackerHotspotRowSubtitle(hotspot engine.FrontTrackerPressure, localizers ...appi18n.Localizer) string {
	loc := viewLocalizer(localizers)
	parts := []string{}
	if title := strings.TrimSpace(hotspot.FrontTitle); title != "" {
		parts = append(parts, loc.T("front.from", title))
	}
	if summary := strings.TrimSpace(hotspot.Summary); summary != "" {
		parts = append(parts, summary)
	} else if detail := strings.TrimSpace(hotspot.Detail); detail != "" {
		parts = append(parts, detail)
	}
	if len(parts) == 0 {
		return loc.T("front.no_fallout_details")
	}
	return strings.Join(parts, " · ")
}

func trackerReactionRowTitle(reaction engine.WorldReaction, width int) string {
	parts := []string{reaction.Title}
	if kind := strings.TrimSpace(reaction.Kind); kind != "" {
		parts = append(parts, kind)
	}
	return truncatePlain(strings.Join(parts, "  ·  "), width)
}

func trackerReactionRowSubtitle(reaction engine.WorldReaction, localizers ...appi18n.Localizer) string {
	if detail := strings.TrimSpace(reaction.Detail); detail != "" {
		return detail
	}
	return viewLocalizer(localizers).T("front.world_shifting")
}

func formatTrackerHookDetail(hook engine.StoryHook, localizers ...appi18n.Localizer) string {
	loc := viewLocalizer(localizers)
	lines := []string{}
	if kind := strings.TrimSpace(hook.Kind); kind != "" {
		lines = append(lines, fmt.Sprintf("%s: %s", loc.T("field.kind"), strings.Title(kind)))
	}
	if npc := strings.TrimSpace(hook.NPCName); npc != "" {
		lines = append(lines, loc.T("front.npc_value", npc))
	}
	if status := strings.TrimSpace(hook.Status); status != "" {
		lines = append(lines, fmt.Sprintf("%s: %s", loc.T("field.status"), strings.Title(status)))
	}
	if hook.TimerTurns > 0 {
		lines = append(lines, loc.Plural("front.timer_turns", hook.TimerTurns, hook.TimerTurns))
	}
	if hook.SourceTurn > 0 {
		lines = append(lines, fmt.Sprintf("%s: %d", loc.T("field.source_turn"), hook.SourceTurn))
	}
	if hook.UpdatedTurn > 0 {
		lines = append(lines, fmt.Sprintf("%s: %d", loc.T("field.updated_turn"), hook.UpdatedTurn))
	}
	if detail := strings.TrimSpace(hook.Detail); detail != "" {
		lines = append(lines, "", loc.T("field.detail"), detail)
	}
	return strings.Join(lines, "\n")
}

func formatTrackerFrontDetail(front engine.FrontTrackerFront, localizers ...appi18n.Localizer) string {
	loc := viewLocalizer(localizers)
	lines := []string{}
	if status := strings.TrimSpace(front.Status); status != "" {
		lines = append(lines, fmt.Sprintf("%s: %s", loc.T("field.status"), strings.Title(status)))
	}
	if visibility := strings.TrimSpace(front.Visibility); visibility != "" {
		lines = append(lines, fmt.Sprintf("%s: %s", loc.T("field.visibility"), strings.Title(visibility)))
	}
	if front.Segments > 0 {
		lines = append(lines, fmt.Sprintf("%s: %d/%d", loc.T("field.progress"), front.Progress, front.Segments))
	}
	if faction := strings.TrimSpace(front.Faction); faction != "" {
		lines = append(lines, fmt.Sprintf("%s: %s", loc.T("field.faction"), faction))
	}
	if front.LastAdvancedTurn > 0 {
		lines = append(lines, loc.T("front.last_advanced_turn", front.LastAdvancedTurn))
	}
	if front.NextEscalationTurn > 0 && !strings.EqualFold(front.Status, "resolved") {
		lines = append(lines, loc.T("front.next_escalation_turn", front.NextEscalationTurn))
	}
	if stakes := strings.TrimSpace(front.Stakes); stakes != "" {
		lines = append(lines, "", loc.T("field.stakes"), stakes)
	}
	if len(front.Pressures) > 0 {
		lines = append(lines, "", loc.T("field.pressure"))
		for _, pressure := range front.Pressures {
			lines = append(lines, "• "+pressure.Summary)
		}
	}
	if strings.EqualFold(front.Status, "resolved") && strings.TrimSpace(front.Resolution) != "" {
		lines = append(lines, "", loc.T("field.outcome"), front.Resolution)
	}
	return strings.Join(lines, "\n")
}

func formatTrackerHotspotDetail(hotspot engine.FrontTrackerPressure, localizers ...appi18n.Localizer) string {
	loc := viewLocalizer(localizers)
	lines := []string{
		loc.T("front.region", hotspot.Region),
		fmt.Sprintf("%s: %s", loc.T("field.kind"), hotspot.Kind),
		fmt.Sprintf("%s: %d (%s)", loc.T("field.pressure"), hotspot.Level, hotspot.Severity),
	}
	if hotspot.UpdatedTurn > 0 {
		lines = append(lines, fmt.Sprintf("%s: %d", loc.T("field.updated_turn"), hotspot.UpdatedTurn))
	}
	if title := strings.TrimSpace(hotspot.FrontTitle); title != "" {
		lines = append(lines, loc.T("front.source_front", title))
	}
	if status := strings.TrimSpace(hotspot.FrontStatus); status != "" {
		lines = append(lines, loc.T("front.front_status", strings.Title(status)))
	}
	if summary := strings.TrimSpace(hotspot.Summary); summary != "" {
		lines = append(lines, "", loc.T("field.summary"), summary)
	}
	if detail := strings.TrimSpace(hotspot.Detail); detail != "" && detail != summaryOrEmpty(hotspot) {
		lines = append(lines, "", loc.T("field.detail"), detail)
	}
	return strings.Join(lines, "\n")
}

func summaryOrEmpty(hotspot engine.FrontTrackerPressure) string {
	return strings.TrimSpace(hotspot.Summary)
}

func formatTrackerReactionDetail(reaction engine.WorldReaction, localizers ...appi18n.Localizer) string {
	loc := viewLocalizer(localizers)
	lines := []string{}
	if kind := strings.TrimSpace(reaction.Kind); kind != "" {
		lines = append(lines, fmt.Sprintf("%s: %s", loc.T("field.kind"), strings.Title(kind)))
	}
	if status := strings.TrimSpace(reaction.Status); status != "" {
		lines = append(lines, fmt.Sprintf("%s: %s", loc.T("field.status"), strings.Title(status)))
	}
	if reaction.CreatedTurn > 0 {
		lines = append(lines, fmt.Sprintf("%s: %d", loc.T("field.created_turn"), reaction.CreatedTurn))
	}
	if reaction.SourceTurn > 0 {
		lines = append(lines, fmt.Sprintf("%s: %d", loc.T("field.source_turn"), reaction.SourceTurn))
	}
	if detail := strings.TrimSpace(reaction.Detail); detail != "" {
		lines = append(lines, "", loc.T("field.detail"), detail)
	}
	return strings.Join(lines, "\n")
}
