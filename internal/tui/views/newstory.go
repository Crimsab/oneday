package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/tui/components"
	"github.com/crimsab/oneday/internal/tui/theme"
)

// StoryCreatedMsg signals that story creation is complete.
type StoryCreatedMsg struct {
	StoryID     string
	CharacterID string
}

// aiResponseMsg carries an AI response back to the TUI.
type aiResponseMsg struct {
	content string
	err     error
}

// NewStoryModel handles the story creation conversation.
type NewStoryModel struct {
	creator    *engine.StoryCreator
	viewport   viewport.Model
	choices    components.ChoiceListModel
	input      textarea.Model
	spinner    spinner.Model
	history    *strings.Builder // rendered conversation (pointer to avoid copy panic)
	waiting    bool             // waiting for AI
	errMsg     string
	width      int
	height     int
	inputFocus bool
	actions    []engine.CreationAction
}

// NewNewStoryModel creates the story creation view.
func NewNewStoryModel(creator *engine.StoryCreator) NewStoryModel {
	ta := textarea.New()
	ta.Placeholder = "Type your response..."
	ta.CharLimit = 500
	ta.SetHeight(3)
	ta.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(theme.Accent)

	vp := viewport.New(80, 20)

	m := NewStoryModel{
		creator:    creator,
		viewport:   vp,
		choices:    components.NewChoiceList(),
		input:      ta,
		spinner:    sp,
		history:    &strings.Builder{},
		inputFocus: false,
	}
	m.syncInputPlaceholder()
	m.refreshActions()
	return m
}

func (m NewStoryModel) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
	)
}

// StartCreation kicks off the initial AI message.
func (m *NewStoryModel) StartCreation() tea.Cmd {
	m.waiting = true
	return func() tea.Msg {
		resp, err := m.creator.StartConversation(context.Background())
		return aiResponseMsg{content: resp, err: err}
	}
}

func (m NewStoryModel) Update(msg tea.Msg) (NewStoryModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 15
		m.input.SetWidth(msg.Width - 4)
		m.choices.SetWidth(msg.Width - 4)
		return m, nil

	case aiResponseMsg:
		m.waiting = false
		if msg.err != nil {
			m.errMsg = fmt.Sprintf("AI Error: %v (press any key to retry)", msg.err)
			return m, nil
		}
		m.errMsg = ""
		// Append AI response to history
		m.history.WriteString(theme.Subtitle.Render("Story Guide") + "\n")
		m.history.WriteString(components.RenderMarkdown(msg.content) + "\n")
		m.viewport.SetContent(m.history.String())
		m.viewport.GotoBottom()
		m.refreshActions()
		m.syncInputPlaceholder()

		// Check if creation is done
		if m.creator.Phase() == engine.PhaseDone {
			story := m.creator.Story()
			char := m.creator.Character()
			return m, func() tea.Msg {
				return StoryCreatedMsg{
					StoryID:     story.ID,
					CharacterID: char.ID,
				}
			}
		}
		return m, nil

	case spinner.TickMsg:
		if m.waiting {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case components.ChoiceSelectedMsg:
		if m.waiting {
			return m, nil
		}
		idx := msg.ID - 1
		if idx < 0 || idx >= len(m.actions) {
			return m, nil
		}
		action := m.actions[idx]
		if action.Key == "focus_input" || strings.HasPrefix(action.Key, "edit_") {
			m.inputFocus = true
			m.input.Focus()
			m.syncInputPlaceholder()
			return m, nil
		}
		m.waiting = true
		captured := action.Key
		return m, func() tea.Msg {
			resp, err := m.creator.ExecuteAction(context.Background(), captured)
			return aiResponseMsg{content: resp, err: err}
		}

	case tea.KeyMsg:
		if m.waiting {
			return m, nil // ignore input while waiting
		}
		if m.errMsg != "" {
			m.errMsg = ""
			return m, nil
		}

		switch msg.Type {
		case tea.KeyCtrlC:
			// handled by parent app
		case tea.KeyTab:
			if len(m.actions) > 0 {
				m.inputFocus = !m.inputFocus
				if m.inputFocus {
					m.input.Focus()
				} else {
					m.input.Blur()
				}
			}
		case tea.KeyEnter:
			if !m.inputFocus && len(m.actions) > 0 {
				break
			}
			if msg.Alt {
				// Alt+Enter for newline in textarea — fall through to textarea update
				break
			}
			text := strings.TrimSpace(m.input.Value())
			if text == "" {
				return m, nil
			}
			// Append player input to history
			m.history.WriteString(theme.SelectedItem.Render("You") + "\n")
			m.history.WriteString(text + "\n\n")
			m.viewport.SetContent(m.history.String())
			m.viewport.GotoBottom()
			m.input.Reset()
			m.waiting = true

			// Send to AI
			captured := text
			return m, func() tea.Msg {
				resp, err := m.creator.SendMessage(context.Background(), captured)
				return aiResponseMsg{content: resp, err: err}
			}
		}
	}

	// Update child components
	if !m.waiting {
		if !m.inputFocus && len(m.actions) > 0 {
			var cmd tea.Cmd
			m.choices, cmd = m.choices.Update(msg)
			cmds = append(cmds, cmd)
		}
		var cmd tea.Cmd
		if m.inputFocus || len(m.actions) == 0 {
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m NewStoryModel) View() string {
	header := theme.Title.Render("Create Your Story")
	phaseBar := theme.MutedText.Render(m.creator.StageLabel())

	var statusLine string
	if m.creator.LastModel() != "" {
		statusLine = theme.MutedText.Render(
			fmt.Sprintf("%s · %dms", m.creator.LastModel(), m.creator.LastLatency()),
		)
	}

	body := m.viewport.View()

	var inputArea string
	if m.waiting {
		inputArea = m.spinner.View() + " Thinking..."
	} else if m.errMsg != "" {
		inputArea = theme.DangerText.Render(m.errMsg)
	} else if !m.inputFocus && len(m.actions) > 0 {
		inputArea = theme.MutedText.Render("Press TAB to type a custom reply.")
	} else {
		inputArea = m.input.View()
	}

	choicesView := m.choices.View()
	if choicesView != "" {
		choicesView = theme.MutedText.Render("Quick choices") + "\n" + choicesView
	}

	help := theme.MutedText.Render("tab toggle · enter send/select · esc back · ctrl+c quit")
	if len(m.actions) == 0 {
		help = theme.MutedText.Render("enter send · esc back · ctrl+c quit")
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		phaseBar,
		statusLine,
		"",
		body,
		"",
		choicesView,
		inputArea,
		help,
	)

	return content
}

// SetSize updates dimensions.
func (m *NewStoryModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w - 4
	m.viewport.Height = h - 15
	m.input.SetWidth(w - 4)
	m.choices.SetWidth(w - 4)
}

func (m *NewStoryModel) refreshActions() {
	m.actions = m.creator.Actions()
	items := make([]components.ChoiceItem, 0, len(m.actions))
	for i, action := range m.actions {
		items = append(items, components.ChoiceItem{
			ID:   i + 1,
			Text: action.Label,
		})
	}
	m.choices.SetChoices(items)
	if len(m.actions) == 0 {
		m.inputFocus = true
		m.input.Focus()
		return
	}
	if m.inputFocus {
		m.input.Focus()
	} else {
		m.input.Blur()
	}
}

func (m *NewStoryModel) syncInputPlaceholder() {
	m.input.Placeholder = m.creator.InputPlaceholder()
}
