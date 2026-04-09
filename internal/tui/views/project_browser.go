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

type projectBrowserRowKind int

const (
	projectBrowserRowSection projectBrowserRowKind = iota
	projectBrowserRowProject
)

type projectBrowserRow struct {
	kind         projectBrowserRowKind
	label        string
	projectIndex int
}

// ProjectBrowserBackMsg closes the project workspace.
type ProjectBrowserBackMsg struct{}

// ProjectBrowserModel renders a dedicated project workspace grouped by status.
type ProjectBrowserModel struct {
	title    string
	board    engine.ProjectBoard
	rows     []projectBrowserRow
	selected int
	width    int
	height   int
	visible  bool
	detail   components.OverlayModel
}

func NewProjectBrowserModel(title string, board engine.ProjectBoard, width, height int) ProjectBrowserModel {
	model := ProjectBrowserModel{
		title:   title,
		board:   normalizedProjectBrowserBoard(board),
		visible: true,
	}
	model.SetSize(width, height)
	model.rebuildRows()
	return model
}

func (m *ProjectBrowserModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.detail.SetSize(width, height)
}

func (m ProjectBrowserModel) Visible() bool {
	return m.visible
}

func (m *ProjectBrowserModel) Close() {
	m.visible = false
}

func (m *ProjectBrowserModel) rebuildRows() {
	m.rows = m.rows[:0]

	for _, section := range orderedProjectSections() {
		sectionProjects := projectIndexesForStatus(m.board.Projects, section.statuses...)
		if len(sectionProjects) == 0 {
			continue
		}
		m.rows = append(m.rows, projectBrowserRow{
			kind:  projectBrowserRowSection,
			label: section.label,
		})
		for _, idx := range sectionProjects {
			m.rows = append(m.rows, projectBrowserRow{
				kind:         projectBrowserRowProject,
				projectIndex: idx,
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

func (m ProjectBrowserModel) selectedRow() (projectBrowserRow, bool) {
	if len(m.rows) == 0 || m.selected < 0 || m.selected >= len(m.rows) {
		return projectBrowserRow{}, false
	}
	return m.rows[m.selected], true
}

func (m ProjectBrowserModel) selectedProject() (engine.ProjectClock, bool) {
	row, ok := m.selectedRow()
	if !ok || row.kind != projectBrowserRowProject || row.projectIndex < 0 || row.projectIndex >= len(m.board.Projects) {
		return engine.ProjectClock{}, false
	}
	return m.board.Projects[row.projectIndex], true
}

func (m ProjectBrowserModel) Update(msg tea.Msg) (ProjectBrowserModel, tea.Cmd) {
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
			return m, func() tea.Msg { return ProjectBrowserBackMsg{} }
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.rows)-1 {
				m.selected++
			}
		case "enter", " ":
			project, ok := m.selectedProject()
			if !ok {
				return m, nil
			}
			m.detail.Show(project.Title, formatProjectDetail(project))
			return m, nil
		}
	}

	return m, nil
}

func (m ProjectBrowserModel) View() string {
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
	lines = append(lines, theme.MutedText.Render(projectWorkspaceSummary(m.board)))
	lines = append(lines, "")

	if len(m.rows) == 0 {
		lines = append(lines, theme.MutedText.Render("No downtime projects yet. Start one through play and it will appear here."))
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
		case projectBrowserRowSection:
			lines = append(lines, theme.Title.Render(row.label))
		case projectBrowserRowProject:
			project := m.board.Projects[row.projectIndex]
			cursor := "  "
			style := theme.MutedText
			if i == m.selected {
				cursor = "▸ "
				style = theme.SelectedItem
			}
			lines = append(lines, style.Render(cursor+projectRowTitle(project, rowWidth)))
			lines = append(lines, theme.MutedText.Render("    "+truncatePlain(projectRowSubtitle(project), rowWidth)))
		}
	}

	lines = append(lines, "")
	lines = append(lines, theme.MutedText.Render("↑↓ navigate · Enter open · I investigations · F fronts · C codex · Esc close"))

	content := strings.Join(lines, "\n")
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, boxStyle(boxWidth).Height(boxHeight).Render(content))
}

type projectSection struct {
	label    string
	statuses []string
}

func orderedProjectSections() []projectSection {
	return []projectSection{
		{label: "Active", statuses: []string{"active"}},
		{label: "Paused", statuses: []string{"paused"}},
		{label: "Completed", statuses: []string{"completed"}},
		{label: "Other", statuses: []string{"abandoned", "failed", "stalled"}},
	}
}

func projectIndexesForStatus(projects []engine.ProjectClock, statuses ...string) []int {
	if len(projects) == 0 {
		return nil
	}
	allowed := map[string]bool{}
	for _, status := range statuses {
		allowed[strings.ToLower(strings.TrimSpace(status))] = true
	}

	var indexes []int
	for i, project := range projects {
		if allowed[strings.ToLower(strings.TrimSpace(project.Status))] {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func normalizedProjectBrowserBoard(board engine.ProjectBoard) engine.ProjectBoard {
	projects := append([]engine.ProjectClock(nil), board.Projects...)
	sort.SliceStable(projects, func(i, j int) bool {
		left := projectStatusOrder(projects[i].Status)
		right := projectStatusOrder(projects[j].Status)
		if left != right {
			return left < right
		}
		if projects[i].UpdatedTurn != projects[j].UpdatedTurn {
			return projects[i].UpdatedTurn > projects[j].UpdatedTurn
		}
		if projects[i].CompletedTurn != projects[j].CompletedTurn {
			return projects[i].CompletedTurn > projects[j].CompletedTurn
		}
		return strings.ToLower(projects[i].Title) < strings.ToLower(projects[j].Title)
	})
	return engine.ProjectBoard{Projects: projects}
}

func projectStatusOrder(status string) int {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active":
		return 0
	case "paused":
		return 1
	case "completed":
		return 2
	default:
		return 3
	}
}

func projectWorkspaceSummary(board engine.ProjectBoard) string {
	counts := map[string]int{}
	for _, project := range board.Projects {
		counts[strings.ToLower(strings.TrimSpace(project.Status))]++
	}
	parts := []string{
		fmt.Sprintf("Active %d", counts["active"]),
		fmt.Sprintf("Paused %d", counts["paused"]),
		fmt.Sprintf("Completed %d", counts["completed"]),
	}
	other := len(board.Projects) - counts["active"] - counts["paused"] - counts["completed"]
	if other > 0 {
		parts = append(parts, fmt.Sprintf("Other %d", other))
	}
	return strings.Join(parts, " · ")
}

func projectRowTitle(project engine.ProjectClock, width int) string {
	parts := []string{project.Title}
	if progress := formatProjectProgress(project); progress != "" {
		parts = append(parts, progress)
	}
	if kind := strings.TrimSpace(project.Kind); kind != "" {
		parts = append(parts, kind)
	}
	return truncatePlain(strings.Join(parts, "  ·  "), width)
}

func projectRowSubtitle(project engine.ProjectClock) string {
	parts := []string{}
	if summary := strings.TrimSpace(project.Summary); summary != "" {
		parts = append(parts, summary)
	}
	if outcome := strings.TrimSpace(project.Outcome); outcome != "" {
		parts = append(parts, "Outcome: "+outcome)
	}
	if stakes := strings.TrimSpace(project.Stakes); stakes != "" && !strings.EqualFold(project.Status, "completed") {
		parts = append(parts, "Stakes: "+stakes)
	}
	if len(parts) == 0 {
		return "No additional details yet."
	}
	return strings.Join(parts, " ")
}

func formatProjectProgress(project engine.ProjectClock) string {
	if project.Segments <= 0 {
		return ""
	}
	return fmt.Sprintf("%d/%d", project.Progress, project.Segments)
}

func formatProjectDetail(project engine.ProjectClock) string {
	lines := []string{}
	lines = append(lines, fmt.Sprintf("Status: %s", strings.Title(strings.TrimSpace(project.Status))))
	if progress := formatProjectProgress(project); progress != "" {
		lines = append(lines, fmt.Sprintf("Progress: %s", progress))
	}
	if kind := strings.TrimSpace(project.Kind); kind != "" {
		lines = append(lines, fmt.Sprintf("Kind: %s", kind))
	}
	if owner := strings.TrimSpace(project.Owner); owner != "" {
		lines = append(lines, fmt.Sprintf("Owner: %s", owner))
	}
	if location := strings.TrimSpace(project.Location); location != "" {
		lines = append(lines, fmt.Sprintf("Location: %s", location))
	}
	if project.UpdatedTurn > 0 {
		lines = append(lines, fmt.Sprintf("Updated Turn: %d", project.UpdatedTurn))
	}
	if project.CompletedTurn > 0 {
		lines = append(lines, fmt.Sprintf("Completed Turn: %d", project.CompletedTurn))
	}

	if summary := strings.TrimSpace(project.Summary); summary != "" {
		lines = append(lines, "", "Summary", summary)
	}
	if stakes := strings.TrimSpace(project.Stakes); stakes != "" {
		lines = append(lines, "", "Stakes", stakes)
	}
	if outcome := strings.TrimSpace(project.Outcome); outcome != "" {
		lines = append(lines, "", "Outcome", outcome)
	}
	if len(project.Rewards) > 0 {
		lines = append(lines, "", "Rewards")
		for _, reward := range project.Rewards {
			line := reward.Label
			if kind := strings.TrimSpace(reward.Kind); kind != "" {
				line = fmt.Sprintf("%s: %s", strings.Title(kind), reward.Label)
			}
			if detail := strings.TrimSpace(reward.Detail); detail != "" {
				line += " — " + detail
			}
			lines = append(lines, "• "+line)
		}
	}
	if len(project.Links) > 0 {
		lines = append(lines, "", "Linked")
		for _, link := range project.Links {
			label := strings.TrimSpace(link.Label)
			if label == "" {
				label = strings.TrimSpace(link.RefID)
			}
			if label == "" {
				continue
			}
			if kind := strings.TrimSpace(link.Kind); kind != "" {
				lines = append(lines, fmt.Sprintf("• %s: %s", strings.Title(kind), label))
			} else {
				lines = append(lines, "• "+label)
			}
		}
	}

	return strings.Join(lines, "\n")
}
