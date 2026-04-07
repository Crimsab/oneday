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
	creator  *engine.StoryCreator
	viewport viewport.Model
	input    textarea.Model
	spinner  spinner.Model
	history  strings.Builder // rendered conversation
	waiting  bool            // waiting for AI
	errMsg   string
	width    int
	height   int
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

	return NewStoryModel{
		creator:  creator,
		viewport: vp,
		input:    ta,
		spinner:  sp,
	}
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
		m.viewport.Height = msg.Height - 10
		m.input.SetWidth(msg.Width - 4)
		return m, nil

	case aiResponseMsg:
		m.waiting = false
		if msg.err != nil {
			m.errMsg = fmt.Sprintf("AI Error: %v (press any key to retry)", msg.err)
			return m, nil
		}
		m.errMsg = ""
		// Append AI response to history
		m.history.WriteString(theme.Subtitle.Render("Narrator") + "\n")
		m.history.WriteString(msg.content + "\n\n")
		m.viewport.SetContent(m.history.String())
		m.viewport.GotoBottom()

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
		case tea.KeyEnter:
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
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m NewStoryModel) View() string {
	header := theme.Title.Render("Create Your Story")

	phaseLabel := "Building your world..."
	if m.creator.Phase() == engine.PhaseCharacter {
		phaseLabel = "Create your protagonist..."
	}
	phaseBar := theme.MutedText.Render(phaseLabel)

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
	} else {
		inputArea = m.input.View()
	}

	help := theme.MutedText.Render("enter send · esc back · ctrl+c quit")

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		phaseBar,
		statusLine,
		"",
		body,
		"",
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
	m.viewport.Height = h - 10
	m.input.SetWidth(w - 4)
}
