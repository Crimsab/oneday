package views

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/tui/components"
	"github.com/crimsab/oneday/internal/tui/theme"
)

type investigationBrowserRowKind int

const (
	investigationBrowserRowSection investigationBrowserRowKind = iota
	investigationBrowserRowCase
)

type investigationBrowserRow struct {
	kind      investigationBrowserRowKind
	label     string
	caseIndex int
}

// InvestigationBrowserBackMsg closes the investigation workspace.
type InvestigationBrowserBackMsg struct{}

// InvestigationBrowserModel renders a dedicated mystery workspace grouped by case status.
type InvestigationBrowserModel struct {
	title    string
	board    engine.InvestigationBoard
	rows     []investigationBrowserRow
	selected int
	width    int
	height   int
	visible  bool
	detail   components.OverlayModel
}

func NewInvestigationBrowserModel(title string, board engine.InvestigationBoard, width, height int) InvestigationBrowserModel {
	model := InvestigationBrowserModel{
		title:   title,
		board:   normalizedInvestigationBrowserBoard(board),
		visible: true,
	}
	model.SetSize(width, height)
	model.rebuildRows()
	return model
}

func (m *InvestigationBrowserModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.detail.SetSize(width, height)
}

func (m InvestigationBrowserModel) Visible() bool {
	return m.visible
}

func (m *InvestigationBrowserModel) Close() {
	m.visible = false
}

func (m *InvestigationBrowserModel) rebuildRows() {
	m.rows = m.rows[:0]
	for _, section := range orderedInvestigationSections() {
		indexes := investigationCaseIndexesForStatus(m.board.Cases, section.statuses...)
		if len(indexes) == 0 {
			continue
		}
		m.rows = append(m.rows, investigationBrowserRow{
			kind:  investigationBrowserRowSection,
			label: section.label,
		})
		for _, idx := range indexes {
			m.rows = append(m.rows, investigationBrowserRow{
				kind:      investigationBrowserRowCase,
				caseIndex: idx,
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

func (m InvestigationBrowserModel) selectedRow() (investigationBrowserRow, bool) {
	if len(m.rows) == 0 || m.selected < 0 || m.selected >= len(m.rows) {
		return investigationBrowserRow{}, false
	}
	return m.rows[m.selected], true
}

func (m InvestigationBrowserModel) selectedCase() (engine.InvestigationCase, bool) {
	row, ok := m.selectedRow()
	if !ok || row.kind != investigationBrowserRowCase || row.caseIndex < 0 || row.caseIndex >= len(m.board.Cases) {
		return engine.InvestigationCase{}, false
	}
	return m.board.Cases[row.caseIndex], true
}

func (m InvestigationBrowserModel) Update(msg tea.Msg) (InvestigationBrowserModel, tea.Cmd) {
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
			return m, func() tea.Msg { return InvestigationBrowserBackMsg{} }
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.rows)-1 {
				m.selected++
			}
		case "enter", " ":
			invCase, ok := m.selectedCase()
			if !ok {
				return m, nil
			}
			m.detail.Show(invCase.Title, formatInvestigationCaseDetail(invCase))
			return m, nil
		}
	}

	return m, nil
}

func (m InvestigationBrowserModel) View() string {
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
	lines = append(lines, theme.MutedText.Render(investigationWorkspaceSummary(m.board)))
	lines = append(lines, "")

	if len(m.rows) == 0 {
		lines = append(lines, theme.MutedText.Render("No investigations yet. Clues, suspects, and contradictions will appear here as the story unfolds."))
		lines = append(lines, "")
		lines = append(lines, theme.MutedText.Render("Esc close"))
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, boxStyle(boxWidth).Render(strings.Join(lines, "\n")))
	}

	rowWidth := boxWidth - 6
	if rowWidth < 30 {
		rowWidth = 30
	}

	for i, row := range m.rows {
		switch row.kind {
		case investigationBrowserRowSection:
			lines = append(lines, theme.Title.Render(row.label))
		case investigationBrowserRowCase:
			invCase := m.board.Cases[row.caseIndex]
			cursor := "  "
			style := theme.MutedText
			if i == m.selected {
				cursor = "▸ "
				style = theme.SelectedItem
			}
			lines = append(lines, style.Render(cursor+investigationRowTitle(invCase, rowWidth)))
			lines = append(lines, theme.MutedText.Render("    "+truncatePlain(investigationRowSubtitle(invCase), rowWidth)))
		}
	}

	lines = append(lines, "")
	lines = append(lines, theme.MutedText.Render("↑↓ navigate · Enter open · Esc close"))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, boxStyle(boxWidth).Height(boxHeight).Render(strings.Join(lines, "\n")))
}

type investigationSection struct {
	label    string
	statuses []string
}

func orderedInvestigationSections() []investigationSection {
	return []investigationSection{
		{label: "Open Cases", statuses: []string{"open", "active", "likely"}},
		{label: "Resolved", statuses: []string{"solved", "closed", "resolved"}},
		{label: "Other", statuses: []string{"cold", "dormant", "paused"}},
	}
}

func investigationCaseIndexesForStatus(cases []engine.InvestigationCase, statuses ...string) []int {
	if len(cases) == 0 {
		return nil
	}
	allowed := map[string]bool{}
	for _, status := range statuses {
		allowed[strings.ToLower(strings.TrimSpace(status))] = true
	}

	var indexes []int
	for i, invCase := range cases {
		if allowed[strings.ToLower(strings.TrimSpace(invCase.Status))] {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func normalizedInvestigationBrowserBoard(board engine.InvestigationBoard) engine.InvestigationBoard {
	cases := append([]engine.InvestigationCase(nil), board.Cases...)
	sort.SliceStable(cases, func(i, j int) bool {
		left := investigationStatusOrder(cases[i].Status)
		right := investigationStatusOrder(cases[j].Status)
		if left != right {
			return left < right
		}
		if cases[i].UpdatedTurn != cases[j].UpdatedTurn {
			return cases[i].UpdatedTurn > cases[j].UpdatedTurn
		}
		return strings.ToLower(cases[i].Title) < strings.ToLower(cases[j].Title)
	})
	return engine.InvestigationBoard{Cases: cases}
}

func investigationStatusOrder(status string) int {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "open", "active", "likely":
		return 0
	case "solved", "closed", "resolved":
		return 1
	default:
		return 2
	}
}

func investigationWorkspaceSummary(board engine.InvestigationBoard) string {
	openCases := 0
	resolved := 0
	for _, invCase := range board.Cases {
		switch investigationStatusOrder(invCase.Status) {
		case 0:
			openCases++
		case 1:
			resolved++
		}
	}
	other := len(board.Cases) - openCases - resolved
	parts := []string{
		fmt.Sprintf("Open %d", openCases),
		fmt.Sprintf("Resolved %d", resolved),
	}
	if other > 0 {
		parts = append(parts, fmt.Sprintf("Other %d", other))
	}
	return strings.Join(parts, " · ")
}

func investigationRowTitle(invCase engine.InvestigationCase, width int) string {
	parts := []string{invCase.Title}
	if status := strings.TrimSpace(invCase.Status); status != "" {
		parts = append(parts, strings.ToLower(status))
	}
	if invCase.UpdatedTurn > 0 {
		parts = append(parts, fmt.Sprintf("turn %d", invCase.UpdatedTurn))
	}
	return truncatePlain(strings.Join(parts, "  ·  "), width)
}

func investigationRowSubtitle(invCase engine.InvestigationCase) string {
	parts := []string{}
	if summary := strings.TrimSpace(invCase.Summary); summary != "" {
		parts = append(parts, summary)
	}
	parts = append(parts, fmt.Sprintf("Clues %d", len(invCase.Clues)))
	if len(invCase.Contradictions) > 0 {
		parts = append(parts, fmt.Sprintf("Contradictions %d", len(invCase.Contradictions)))
	}
	if len(invCase.Suspects) > 0 {
		parts = append(parts, fmt.Sprintf("Suspects %d", len(invCase.Suspects)))
	}
	if len(invCase.Theories) > 0 {
		parts = append(parts, fmt.Sprintf("Theories %d", len(invCase.Theories)))
	}
	return strings.Join(parts, " · ")
}

func formatInvestigationCaseDetail(invCase engine.InvestigationCase) string {
	lines := []string{}
	if status := strings.TrimSpace(invCase.Status); status != "" {
		lines = append(lines, fmt.Sprintf("Status: %s", strings.Title(status)))
	}
	if invCase.UpdatedTurn > 0 {
		lines = append(lines, fmt.Sprintf("Updated Turn: %d", invCase.UpdatedTurn))
	}
	if summary := strings.TrimSpace(invCase.Summary); summary != "" {
		lines = append(lines, "", "Summary", summary)
	}
	appendInvestigationList := func(title string, entries []string) {
		if len(entries) == 0 {
			return
		}
		lines = append(lines, "", title)
		for _, entry := range entries {
			lines = append(lines, "• "+entry)
		}
	}

	appendInvestigationList("Clues", investigationClueLines(invCase.Clues))
	appendInvestigationList("Suspects", investigationSuspectLines(invCase.Suspects))
	appendInvestigationList("Claims", investigationClaimLines(invCase.Claims))
	appendInvestigationList("Contradictions", investigationContradictionLines(invCase.Contradictions))
	appendInvestigationList("Leads", investigationLeadLines(invCase.Leads))
	appendInvestigationList("Theories", investigationTheoryLines(invCase.Theories))
	appendInvestigationList("Linked", investigationLinkLines(invCase.Links))

	return strings.Join(lines, "\n")
}

func investigationClueLines(clues []engine.InvestigationClue) []string {
	out := make([]string, 0, len(clues))
	for _, clue := range clues {
		line := strings.TrimSpace(clue.Label)
		if detail := strings.TrimSpace(clue.Detail); detail != "" {
			line += " — " + detail
		}
		if source := strings.TrimSpace(clue.Source); source != "" {
			line += fmt.Sprintf(" (source: %s)", source)
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func investigationSuspectLines(suspects []engine.InvestigationSuspect) []string {
	out := make([]string, 0, len(suspects))
	for _, suspect := range suspects {
		line := strings.TrimSpace(suspect.Name)
		if detail := strings.TrimSpace(suspect.Detail); detail != "" {
			line += " — " + detail
		}
		if status := strings.TrimSpace(suspect.Status); status != "" {
			line += fmt.Sprintf(" (%s)", status)
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func investigationClaimLines(claims []engine.InvestigationClaim) []string {
	out := make([]string, 0, len(claims))
	for _, claim := range claims {
		line := strings.TrimSpace(claim.Statement)
		if confidence := strings.TrimSpace(claim.Confidence); confidence != "" {
			line += fmt.Sprintf(" (%s)", confidence)
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func investigationContradictionLines(items []engine.InvestigationContradiction) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		line := strings.TrimSpace(item.Label)
		if detail := strings.TrimSpace(item.Detail); detail != "" {
			line += " — " + detail
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func investigationLeadLines(items []engine.InvestigationLead) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		line := strings.TrimSpace(item.Title)
		if detail := strings.TrimSpace(item.Detail); detail != "" {
			line += " — " + detail
		}
		if status := strings.TrimSpace(item.Status); status != "" {
			line += fmt.Sprintf(" (%s)", status)
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func investigationTheoryLines(items []engine.InvestigationTheory) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		line := strings.TrimSpace(item.Statement)
		if confidence := strings.TrimSpace(item.Confidence); confidence != "" {
			line += fmt.Sprintf(" (%s)", confidence)
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func investigationLinkLines(links []engine.InvestigationLink) []string {
	out := make([]string, 0, len(links))
	for _, link := range links {
		label := strings.TrimSpace(link.Label)
		if label == "" {
			label = strings.TrimSpace(link.RefID)
		}
		if label == "" {
			continue
		}
		if kind := strings.TrimSpace(link.Kind); kind != "" {
			out = append(out, fmt.Sprintf("%s: %s", strings.Title(kind), label))
		} else {
			out = append(out, label)
		}
	}
	return out
}
