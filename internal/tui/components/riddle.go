package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/engine"
	appi18n "github.com/crimsab/oneday/internal/i18n"
	"github.com/crimsab/oneday/internal/tui/theme"
)

// RiddleResultMsg is emitted when riddle resolves.
type RiddleResultMsg struct {
	Result *engine.ChallengeResult
}

// RiddleModel handles the riddle challenge.
type RiddleModel struct {
	challenge *engine.RiddleChallenge
	input     textinput.Model
	phase     string // "riddle", "result"
	result    *engine.ChallengeResult
	width     int
	height    int
	loc       appi18n.Localizer
}

// NewRiddleModel creates a riddle challenge.
func NewRiddleModel(challenge *engine.RiddleChallenge, width, height int, localizers ...appi18n.Localizer) RiddleModel {
	loc := componentLocalizer(localizers)
	ti := textinput.New()
	ti.Placeholder = loc.T("minigame.riddle_answer")
	ti.Width = 30
	ti.Focus()

	return RiddleModel{
		challenge: challenge,
		input:     ti,
		phase:     "riddle",
		width:     width,
		height:    height,
		loc:       loc,
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
					Detail: r.loc.T("minigame.riddle_skipped"),
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
			result := r.result
			return r, func() tea.Msg {
				return RiddleResultMsg{Result: result}
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
	lines = append(lines, titleStyle.Render("🔮  "+r.loc.T("minigame.riddle_title")))
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
		lines = append(lines, "  "+r.loc.T("minigame.answer"))
		lines = append(lines, "  > "+r.input.View())
		lines = append(lines, "")
		lines = append(lines, theme.MutedText.Render("  "+r.loc.T("minigame.submit")))

	case "result":
		if r.result != nil && r.result.Passed {
			lines = append(lines, lipgloss.NewStyle().Foreground(theme.Success).Bold(true).Render("  ✓ "+r.loc.T("minigame.riddle_correct")))
		} else {
			lines = append(lines, lipgloss.NewStyle().Foreground(theme.Danger).Bold(true).Render("  ✗ "+r.loc.T("minigame.riddle_wrong")))
			lines = append(lines, fmt.Sprintf("  "+r.loc.T("minigame.riddle_solution"),
				lipgloss.NewStyle().Foreground(theme.Accent).Render(r.challenge.Answer)))
		}
		lines = append(lines, "")
		lines = append(lines, theme.MutedText.Render("  "+r.loc.T("challenge.continue")))
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
