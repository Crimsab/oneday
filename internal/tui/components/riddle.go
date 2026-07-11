package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/tui/theme"
)

// RiddleResultMsg is emitted when riddle resolves.
type RiddleResultMsg struct {
	Passed bool
}

// RiddleModel handles the riddle challenge.
type RiddleModel struct {
	challenge *engine.RiddleChallenge
	input     textinput.Model
	phase     string // "riddle", "result"
	result    *engine.ChallengeResult
	width     int
	height    int
}

// NewRiddleModel creates a riddle challenge.
func NewRiddleModel(challenge *engine.RiddleChallenge, width, height int) RiddleModel {
	ti := textinput.New()
	ti.Placeholder = "Type your answer..."
	ti.Width = 30
	ti.Focus()

	return RiddleModel{
		challenge: challenge,
		input:     ti,
		phase:     "riddle",
		width:     width,
		height:    height,
	}
}

// Update handles input.
func (r RiddleModel) Update(msg tea.Msg) (RiddleModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch r.phase {
		case "riddle":
			switch msg.String() {
			case "enter":
				answer := strings.TrimSpace(r.input.Value())
				if answer == "" {
					return r, nil
				}
				result := r.challenge.CheckRiddle(answer)
				r.result = result
				r.phase = "result"
				r.input.Blur()
				return r, nil
			case "esc":
				// Skip = fail.
				r.result = &engine.ChallengeResult{
					Passed: false,
					Detail: "Riddle: skipped → FAIL",
				}
				r.phase = "result"
				r.input.Blur()
				return r, nil
			default:
				var cmd tea.Cmd
				r.input, cmd = r.input.Update(msg)
				return r, cmd
			}
		case "result":
			passed := r.result != nil && r.result.Passed
			return r, func() tea.Msg {
				return RiddleResultMsg{Passed: passed}
			}
		}

	default:
		if r.phase == "riddle" {
			var cmd tea.Cmd
			r.input, cmd = r.input.Update(msg)
			return r, cmd
		}
	}
	return r, nil
}

// View renders the riddle.
func (r RiddleModel) View() string {
	innerW := 42
	var lines []string

	titleStyle := lipgloss.NewStyle().Foreground(theme.RiddleCyan).Bold(true)
	lines = append(lines, titleStyle.Render("🔮  RIDDLE"))
	lines = append(lines, "")

	// Wrap riddle text.
	riddleText := r.challenge.Riddle
	wrapped := wrapText(riddleText, innerW-4)
	for _, line := range wrapped {
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.RiddleCyan).Italic(true).Render("  "+line))
	}
	lines = append(lines, "")

	switch r.phase {
	case "riddle":
		lines = append(lines, "  Your answer:")
		lines = append(lines, "  > "+r.input.View())
		lines = append(lines, "")
		lines = append(lines, theme.MutedText.Render("  Enter to submit, Esc to skip"))

	case "result":
		if r.result != nil && r.result.Passed {
			lines = append(lines, lipgloss.NewStyle().Foreground(theme.Success).Bold(true).Render("  ✓ Correct!"))
		} else {
			lines = append(lines, lipgloss.NewStyle().Foreground(theme.Danger).Bold(true).Render("  ✗ Wrong!"))
			lines = append(lines, fmt.Sprintf("  The answer was: %s",
				lipgloss.NewStyle().Foreground(theme.Accent).Render(r.challenge.Answer)))
		}
		lines = append(lines, "")
		lines = append(lines, theme.MutedText.Render("  Press any key to continue"))
	}

	inner := strings.Join(lines, "\n")
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.RiddleCyan).
		Padding(1, 2).
		Width(innerW + 4)

	box := boxStyle.Render(inner)
	return lipgloss.Place(r.width, r.height, lipgloss.Center, lipgloss.Center, box)
}

// wrapText wraps text at word boundaries for a given max width.
func wrapText(text string, maxWidth int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}

	var lines []string
	current := words[0]
	for _, word := range words[1:] {
		if len(current)+1+len(word) <= maxWidth {
			current += " " + word
		} else {
			lines = append(lines, current)
			current = word
		}
	}
	lines = append(lines, current)
	return lines
}
