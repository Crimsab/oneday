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
	"github.com/crimsab/oneday/internal/tui/components"
	"github.com/crimsab/oneday/internal/tui/theme"
)

// narrativeResponseMsg carries a parsed AI narrative response.
type narrativeResponseMsg struct {
	response *engine.NarrativeResponse
	err      error
}

// NarrativeModel is the core gameplay view.
type NarrativeModel struct {
	narrator   *engine.Narrator
	viewport   viewport.Model
	typewriter components.TypewriterModel
	statusBar  components.StatusBarModel
	choices    components.ChoiceListModel
	input      textarea.Model
	history    strings.Builder // full narrative text accumulated so far
	waiting    bool            // waiting for AI response
	errMsg     string
	width      int
	height     int
	inputFocus bool // true = free input active, false = choice list active
}

// NewNarrativeModel creates the narrative view.
func NewNarrativeModel(narrator *engine.Narrator, typewriterSpeed int) NarrativeModel {
	ta := textarea.New()
	ta.Placeholder = "Type a free action or press 1-4 to choose..."
	ta.CharLimit = 500
	ta.SetHeight(2)

	vp := viewport.New(80, 20)

	return NarrativeModel{
		narrator:   narrator,
		viewport:   vp,
		typewriter: components.NewTypewriter(typewriterSpeed),
		statusBar:  components.NewStatusBar(),
		choices:    components.NewChoiceList(),
		input:      ta,
		inputFocus: false, // start on choice list
	}
}

func (m NarrativeModel) Init() tea.Cmd {
	return textarea.Blink
}

// StartNarration kicks off the first AI turn.
func (m *NarrativeModel) StartNarration() tea.Cmd {
	m.waiting = true
	return func() tea.Msg {
		resp, err := m.narrator.StartNarration(context.Background())
		return narrativeResponseMsg{response: resp, err: err}
	}
}

func (m NarrativeModel) Update(msg tea.Msg) (NarrativeModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.relayout()
		return m, nil

	case narrativeResponseMsg:
		m.waiting = false
		if msg.err != nil {
			m.errMsg = fmt.Sprintf("AI Error: %v", msg.err)
			return m, nil
		}
		m.errMsg = ""
		nr := msg.response

		// Update status bar with current vitals and AI metadata
		m.updateStatusBar()

		// Set choices
		choiceItems := make([]components.ChoiceItem, len(nr.Choices))
		for i, c := range nr.Choices {
			choiceItems[i] = components.ChoiceItem{ID: c.ID, Text: c.Text}
		}
		m.choices.SetChoices(choiceItems)

		// Append narrative to history and start typewriter from beginning
		m.history.WriteString(nr.Narrative + "\n\n")
		cmd := m.typewriter.SetText(m.history.String())
		cmds = append(cmds, cmd)

		// Focus on choices if available, otherwise free input
		if len(nr.Choices) > 0 {
			m.inputFocus = false
			m.input.Blur()
		} else {
			m.inputFocus = true
			m.input.Focus()
		}

		return m, tea.Batch(cmds...)

	case components.TypewriterDoneMsg:
		// Typewriter finished — update viewport content and scroll to bottom
		m.viewport.SetContent(m.typewriter.View())
		m.viewport.GotoBottom()
		return m, nil

	case components.ChoiceSelectedMsg:
		if m.waiting {
			return m, nil
		}
		return m, m.sendAction(fmt.Sprintf("[Choice %d] %s", msg.ID, msg.Text))

	case tea.KeyMsg:
		if m.waiting {
			return m, nil
		}

		switch msg.String() {
		case "tab":
			// Toggle between choice list and free input
			m.inputFocus = !m.inputFocus
			if m.inputFocus {
				m.input.Focus()
			} else {
				m.input.Blur()
			}
			return m, nil

		case "enter":
			if m.inputFocus {
				text := strings.TrimSpace(m.input.Value())
				if text == "" {
					return m, nil
				}
				m.input.Reset()
				return m, m.sendAction(text)
			}
			// If on choices, let choice list handle it below

		case "esc":
			// Handled by parent app for menu return
			return m, nil
		}

		// Route key to active component
		if m.inputFocus {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		} else {
			var cmd tea.Cmd
			m.choices, cmd = m.choices.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Always update typewriter
	var twCmd tea.Cmd
	m.typewriter, twCmd = m.typewriter.Update(msg)
	if twCmd != nil {
		cmds = append(cmds, twCmd)
		// Update viewport content as typewriter progresses
		m.viewport.SetContent(m.typewriter.View())
		m.viewport.GotoBottom()
	}

	// Update viewport scrolling
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

func (m *NarrativeModel) sendAction(action string) tea.Cmd {
	m.waiting = true
	m.choices.SetChoices(nil) // clear choices while waiting for AI
	return func() tea.Msg {
		resp, err := m.narrator.SendAction(context.Background(), action)
		return narrativeResponseMsg{response: resp, err: err}
	}
}

// updateStatusBar reads the character's stats JSON and populates the status bar.
func (m *NarrativeModel) updateStatusBar() {
	var stats map[string]interface{}
	_ = json.Unmarshal([]byte(m.narrator.Character().StatsJSON), &stats)

	var vitals []components.Vital
	if vitalsMap, ok := stats["vitals"].(map[string]interface{}); ok {
		for key, val := range vitalsMap {
			if vMap, ok := val.(map[string]interface{}); ok {
				current := toInt(vMap["current"])
				max := toInt(vMap["max"])
				vitals = append(vitals, components.Vital{
					Label:   strings.ToUpper(key),
					Current: current,
					Max:     max,
				})
			}
		}
	}

	m.statusBar.SetData(components.StatusBarData{
		Vitals:  vitals,
		Model:   m.narrator.LastModel(),
		Latency: m.narrator.LastLatency(),
	})
}

// toInt safely converts interface{} to int (handles float64 from JSON).
func toInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return 0
}

func (m NarrativeModel) View() string {
	// Header: chapter + location
	world := m.narrator.World()
	header := theme.Title.Render(fmt.Sprintf("Chapter %d", world.CurrentChapter))
	if world.CurrentLocation != "" {
		header += "  " + theme.Subtitle.Render(world.CurrentLocation)
	}

	// Narrative viewport
	body := m.viewport.View()

	// Error display
	if m.errMsg != "" {
		body += "\n" + theme.DangerText.Render(m.errMsg)
	}

	// Waiting indicator
	var waitLine string
	if m.waiting {
		waitLine = theme.MutedText.Render("  The narrator is writing...")
	}

	// Choices
	choicesView := m.choices.View()

	// Input area
	var inputView string
	if m.inputFocus {
		inputView = m.input.View()
	} else {
		inputView = theme.MutedText.Render("  [TAB] Free input")
	}

	// Help line
	help := theme.MutedText.Render("tab toggle · 1-4 choose · enter send · esc menu")

	// Status bar
	m.statusBar.SetWidth(m.width)
	statusView := m.statusBar.View()

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		body,
		waitLine,
		"",
		choicesView,
		inputView,
		help,
		statusView,
	)

	return content
}

// SetSize updates the view dimensions and re-layouts sub-components.
func (m *NarrativeModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.relayout()
}

func (m *NarrativeModel) relayout() {
	// Reserve: header(1) + gap(1) + choices(~6) + input(2) + help(1) + status(1) + gaps(4) = ~16
	vpHeight := m.height - 16
	if vpHeight < 5 {
		vpHeight = 5
	}
	m.viewport.Width = m.width - 2
	m.viewport.Height = vpHeight
	m.input.SetWidth(m.width - 4)
	m.choices.SetWidth(m.width - 4)
	m.statusBar.SetWidth(m.width)
}
