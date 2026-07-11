package views

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/storage"
	"github.com/crimsab/oneday/internal/tui/components"
	"github.com/crimsab/oneday/internal/tui/theme"
)

// CombatEndMsg signals combat is over and carries the summary for the narrative.
type CombatEndMsg struct {
	Summary    string
	Victory    bool
	PersistErr error
}

// combatTurnMsg carries the result of a combat turn.
type combatTurnMsg struct {
	result *engine.CombatTurnResult
	err    error
}

// CombatModel is the TUI view for turn-based combat.
type CombatModel struct {
	combat   *engine.CombatEngine
	narrator *engine.Narrator

	// Layout
	width  int
	height int

	// Sub-components
	viewport   viewport.Model
	typewriter components.TypewriterModel
	choices    components.ChoiceListModel
	input      textarea.Model
	playerHP   components.HPBar
	enemyHP    components.HPBar
	statusBar  components.StatusBarModel

	// State
	history    *strings.Builder
	inputFocus bool
	waiting    bool
	errMsg     string
	turnCount  int
}

// NewCombatModel creates a combat view from an active combat engine.
func NewCombatModel(combat *engine.CombatEngine, narrator *engine.Narrator, width, height int) CombatModel {
	state := combat.State()

	// Create textarea for free input.
	input := newGameTextarea("Describe your action...", actionInputHeight)
	input.SetWidth(width - 4)
	input.Focus()

	// Initialize choice list.
	choices := components.NewChoiceList()
	choices.SetWidth(width - 4)

	// Initialize viewport.
	vp := viewport.New(width-4, height-16)
	vp.SetContent("")

	// Initialize typewriter.
	tw := components.NewTypewriter(80)

	// Initialize HP bars.
	barWidth := width/2 - 4
	if barWidth < 20 {
		barWidth = 20
	}
	playerHP := components.NewHPBar("Player", state.PlayerHP, state.PlayerMaxHP, barWidth)
	enemyHP := components.NewHPBar(state.Enemy.Name, state.Enemy.HP, state.Enemy.MaxHP, barWidth)

	m := CombatModel{
		combat:     combat,
		narrator:   narrator,
		width:      width,
		height:     height,
		viewport:   vp,
		typewriter: tw,
		choices:    choices,
		input:      input,
		playerHP:   playerHP,
		enemyHP:    enemyHP,
		history:    &strings.Builder{},
		inputFocus: true,
		turnCount:  state.Turn,
	}

	return m
}

// Init initialises the combat model — no async command needed at start.
func (m CombatModel) Init() tea.Cmd {
	return nil
}

// SetSize updates the terminal dimensions.
func (m *CombatModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.input.SetWidth(w - 4)
	m.choices.SetWidth(w - 4)
	barWidth := w/2 - 4
	if barWidth < 20 {
		barWidth = 20
	}
	m.playerHP.Width = barWidth
	m.enemyHP.Width = barWidth
	vpH := h - 18
	if vpH < 4 {
		vpH = 4
	}
	m.viewport = viewport.New(w-4, vpH)
	m.viewport.SetContent(m.typewriter.View())
	m.statusBar.SetWidth(w)
}

// sendCombatAction sends a player action to the combat engine as a tea.Cmd.
func (m CombatModel) sendCombatAction(action string) tea.Cmd {
	combat := m.combat
	return func() tea.Msg {
		result, err := combat.PlayerAction(context.Background(), action)
		return combatTurnMsg{result: result, err: err}
	}
}

// Update handles messages for the combat view.
func (m CombatModel) Update(msg tea.Msg) (CombatModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, nil

	case combatTurnMsg:
		m.waiting = false
		if msg.err != nil {
			m.errMsg = fmt.Sprintf("Combat error: %v", msg.err)
			return m, nil
		}
		result := msg.result

		// Update HP bars.
		state := m.combat.State()
		barWidth := m.width/2 - 4
		if barWidth < 20 {
			barWidth = 20
		}
		m.playerHP = components.NewHPBar("Player", result.PlayerHP, state.PlayerMaxHP, barWidth)
		m.enemyHP = components.NewHPBar(state.Enemy.Name, result.EnemyHP, state.Enemy.MaxHP, barWidth)

		// Append narrative to history.
		rendered := components.RenderMarkdown(result.Narrative)
		m.history.WriteString(rendered + "\n")

		// Animate new narrative text.
		cmd := m.typewriter.SetText(m.history.String())
		cmds = append(cmds, cmd)

		// Update choices.
		items := make([]components.ChoiceItem, len(result.Choices))
		for i, c := range result.Choices {
			items[i] = components.ChoiceItem{ID: c.ID, Text: c.Text}
		}
		m.choices.SetChoices(items)
		m.inputFocus = false
		m.input.Blur()

		// Increment turn counter.
		m.turnCount = state.Turn

		// Check if combat is over.
		if result.CombatOver {
			persistErr := m.combat.WriteSummaryToMain()
			// Emit CombatEndMsg to parent (narrative view).
			summary := result.Summary
			victory := result.Victory
			return m, func() tea.Msg {
				return CombatEndMsg{Summary: summary, Victory: victory, PersistErr: persistErr}
			}
		}

		return m, tea.Batch(cmds...)

	case components.TypewriterDoneMsg:
		m.viewport.SetContent(m.typewriter.View())
		m.viewport.GotoBottom()
		return m, nil

	case components.ChoiceSelectedMsg:
		if m.waiting {
			return m, nil
		}
		m.waiting = true
		m.errMsg = ""
		action := fmt.Sprintf("[Choice %d] %s", msg.ID, msg.Text)
		return m, m.sendCombatAction(action)

	case tea.KeyMsg:
		if m.waiting {
			return m, nil
		}

		switch msg.String() {
		case "tab":
			m.inputFocus = !m.inputFocus
			if m.inputFocus {
				m.input.Focus()
			} else {
				m.input.Blur()
			}
			return m, nil

		case "enter":
			if m.inputFocus {
				action := strings.TrimSpace(m.input.Value())
				if action == "" {
					return m, nil
				}
				m.input.Reset()
				m.waiting = true
				m.errMsg = ""
				return m, m.sendCombatAction(action)
			}

		case "esc":
			// Attempt to flee.
			if !m.waiting {
				m.waiting = true
				m.errMsg = ""
				return m, m.sendCombatAction("flee")
			}
			return m, nil
		}

		// Route key to focused component.
		if m.inputFocus {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		} else {
			var cmd tea.Cmd
			m.choices, cmd = m.choices.Update(msg)
			cmds = append(cmds, cmd)
		}

		// Always update typewriter.
		var twCmd tea.Cmd
		m.typewriter, twCmd = m.typewriter.Update(msg)
		cmds = append(cmds, twCmd)

		return m, tea.Batch(cmds...)
	}

	// Update typewriter for tick messages.
	var twCmd tea.Cmd
	m.typewriter, twCmd = m.typewriter.Update(msg)
	if twCmd != nil {
		cmds = append(cmds, twCmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the combat screen.
func (m CombatModel) View() string {
	if m.width == 0 {
		return "Loading combat..."
	}

	var sections []string

	// --- Header ---
	combatTitle := theme.CombatHeader.Render("COMBAT")
	turnInfo := theme.CombatTurn.Render(fmt.Sprintf("Turn %d", m.turnCount))
	headerContent := lipgloss.JoinHorizontal(lipgloss.Top, combatTitle, " — ", turnInfo)
	if m.waiting {
		headerContent += theme.MutedText.Render("  [AI thinking...]")
	}
	header := lipgloss.NewStyle().
		Width(m.width).
		Padding(0, 1).
		BorderBottom(true).
		BorderForeground(theme.Secondary).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		Render(headerContent)
	sections = append(sections, header)

	// --- HP Bars ---
	playerBar := m.playerHP.View()
	enemyBar := m.enemyHP.View()
	barWidth := m.width/2 - 2
	if barWidth < 20 {
		barWidth = 20
	}
	hpSection := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(barWidth).Render(playerBar),
		"  ",
		lipgloss.NewStyle().Width(barWidth).Render(enemyBar),
	)
	hpBlock := lipgloss.NewStyle().
		Width(m.width).
		Padding(0, 1).
		Render(hpSection)
	sections = append(sections, hpBlock)

	// --- Narrative viewport ---
	m.viewport.SetContent(m.typewriter.View())
	vpStyle := lipgloss.NewStyle().
		Width(m.width-2).
		Padding(0, 1)
	sections = append(sections, vpStyle.Render(m.viewport.View()))

	// --- Separator ---
	sep := theme.MutedText.Render(strings.Repeat("─", m.width-2))
	sections = append(sections, sep)

	// --- Choices ---
	if m.choices.HasChoices() {
		choicesView := lipgloss.NewStyle().Padding(0, 1).Render(m.choices.View())
		sections = append(sections, choicesView)
	}

	// --- Error message ---
	if m.errMsg != "" {
		sections = append(sections, theme.DangerText.Render("  "+m.errMsg))
	}

	// --- Free input ---
	focusHint := ""
	if !m.inputFocus && m.choices.HasChoices() {
		focusHint = theme.MutedText.Render(" [Tab: switch to free input]")
	} else if m.inputFocus && m.choices.HasChoices() {
		focusHint = theme.MutedText.Render(" [Tab: switch to choices | Alt+Enter/Ctrl+J: newline]")
	}
	inputSection := lipgloss.JoinVertical(lipgloss.Left,
		theme.MutedText.Render("  Free action:")+focusHint,
		lipgloss.NewStyle().Padding(0, 1).Render(m.input.View()),
	)
	sections = append(sections, inputSection)

	// --- Status bar ---
	char := m.narrator.Character()
	playerHP, playerMaxHP := getVitalsFromChar(char)
	statusData := components.StatusBarData{
		Vitals: []components.Vital{
			{Label: "HP", Current: playerHP, Max: playerMaxHP},
		},
		Model:            m.narrator.LastModel(),
		Latency:          m.narrator.LastLatency(),
		TimeToFirstToken: m.narrator.LastTimeToFirstToken(),
		PromptTokens:     m.narrator.LastUsage().PromptTokens,
		CompletionTokens: m.narrator.LastUsage().CompletionTokens,
		ReasoningTokens:  m.narrator.LastUsage().ReasoningTokens,
		TotalTokens:      m.narrator.LastUsage().TotalTokens,
		CostUSD:          m.narrator.LastUsage().CostUSD,
		Streamed:         m.narrator.LastStreamed(),
	}
	m.statusBar.SetData(statusData)
	m.statusBar.SetWidth(m.width)
	sections = append(sections, m.statusBar.View())

	return strings.Join(sections, "\n")
}

// getVitalsFromChar reads current/max HP from a character's StatsJSON.
func getVitalsFromChar(char *storage.Character) (current, max int) {
	current = 20
	max = 20
	if char == nil {
		return
	}
	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(char.StatsJSON), &stats); err != nil {
		return
	}
	vitals, ok := stats["vitals"].(map[string]interface{})
	if !ok {
		return
	}
	hp, ok := vitals["hp"].(map[string]interface{})
	if !ok {
		return
	}
	if cur, ok := hp["current"].(float64); ok {
		current = int(cur)
	}
	if mx, ok := hp["max"].(float64); ok {
		max = int(mx)
	}
	return
}
