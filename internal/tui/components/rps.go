package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/engine"
	appi18n "github.com/crimsab/oneday/internal/i18n"
	"github.com/crimsab/oneday/internal/tui/theme"
)

// RPSResultMsg is emitted when RPS resolves.
type RPSResultMsg struct {
	Result *engine.ChallengeResult
}

// rpsRevealTickMsg triggers reveal animation.
type rpsRevealTickMsg struct{}

// RPSModel handles the rock-paper-scissors mini-game UI.
type RPSModel struct {
	phase        string // "choose", "reveal", "result"
	cursor       int    // 0=rock, 1=paper, 2=scissors
	playerChoice engine.RPSChoice
	aiChoice     engine.RPSChoice
	result       *engine.RPSResult
	revealFrame  int
	width        int
	height       int
	loc          appi18n.Localizer
}

var rpsChoices = []engine.RPSChoice{engine.RPSRock, engine.RPSPaper, engine.RPSScissors}
var rpsEmoji = map[engine.RPSChoice]string{
	engine.RPSRock:     "🪨",
	engine.RPSPaper:    "📄",
	engine.RPSScissors: "✂️ ",
}
var rpsLabelKeys = map[engine.RPSChoice]string{
	engine.RPSRock:     "minigame.rock",
	engine.RPSPaper:    "minigame.paper",
	engine.RPSScissors: "minigame.scissors",
}

// NewRPSModel creates an RPS mini-game overlay.
func NewRPSModel(width, height int, localizers ...appi18n.Localizer) RPSModel {
	return RPSModel{
		phase:  "choose",
		cursor: 0,
		width:  width,
		height: height,
		loc:    componentLocalizer(localizers),
	}
}

// Update handles input.
func (r RPSModel) Update(msg tea.Msg) (RPSModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch r.phase {
		case "choose":
			switch msg.String() {
			case "up", "k":
				if r.cursor > 0 {
					r.cursor--
				}
			case "down", "j":
				if r.cursor < len(rpsChoices)-1 {
					r.cursor++
				}
			case "enter", " ":
				r.playerChoice = rpsChoices[r.cursor]
				r.phase = "reveal"
				r.revealFrame = 0
				return r, tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg {
					return rpsRevealTickMsg{}
				})
			}
		case "result":
			result := *r.result
			return r, func() tea.Msg {
				return RPSResultMsg{Result: engine.RPSToChallengeResult(result)}
			}
		}

	case rpsRevealTickMsg:
		if r.phase == "reveal" {
			r.revealFrame++
			if r.revealFrame >= 3 {
				// Resolve RPS.
				result := engine.ResolveRPS(r.playerChoice)
				r.result = &result
				r.aiChoice = result.AIChoice
				r.phase = "result"
				return r, nil
			}
			return r, tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg {
				return rpsRevealTickMsg{}
			})
		}
	}
	return r, nil
}

// View renders the RPS game.
func (r RPSModel) View() string {
	innerW := 36
	var lines []string

	titleStyle := lipgloss.NewStyle().Foreground(theme.RPSPurple).Bold(true)
	lines = append(lines, titleStyle.Render("✊ ✋ ✌️  "+r.loc.T("minigame.rps_title")))
	lines = append(lines, "")

	switch r.phase {
	case "choose":
		lines = append(lines, "  "+r.loc.T("minigame.choose"))
		lines = append(lines, "")
		for i, choice := range rpsChoices {
			label := fmt.Sprintf("  %s %s", rpsEmoji[choice], r.loc.T(rpsLabelKeys[choice]))
			if i == r.cursor {
				lines = append(lines, lipgloss.NewStyle().Foreground(theme.Highlight).Bold(true).Render("▸ "+label[2:]))
			} else {
				lines = append(lines, label)
			}
		}
		lines = append(lines, "")
		lines = append(lines, theme.MutedText.Render("  "+r.loc.T("minigame.select_move")))

	case "reveal":
		dots := strings.Repeat(".", r.revealFrame+1)
		lines = append(lines, fmt.Sprintf("  %s: %s %s", r.loc.T("minigame.you"), rpsEmoji[r.playerChoice], r.loc.T(rpsLabelKeys[r.playerChoice])))
		lines = append(lines, "  "+r.loc.T("minigame.ai_thinking", "🤔", dots))

	case "result":
		lines = append(lines, fmt.Sprintf("  %s: %s %s", r.loc.T("minigame.you"), rpsEmoji[r.playerChoice], r.loc.T(rpsLabelKeys[r.playerChoice])))
		lines = append(lines, fmt.Sprintf("  %s:  %s %s", r.loc.T("minigame.ai"), rpsEmoji[r.aiChoice], r.loc.T(rpsLabelKeys[r.aiChoice])))
		lines = append(lines, "")
		switch r.result.Outcome {
		case "win":
			lines = append(lines, lipgloss.NewStyle().Foreground(theme.Success).Bold(true).Render("  ✓ "+r.loc.T("minigame.you_win")))
		case "lose":
			lines = append(lines, lipgloss.NewStyle().Foreground(theme.Danger).Bold(true).Render("  ✗ "+r.loc.T("minigame.you_lose")))
		case "draw":
			lines = append(lines, lipgloss.NewStyle().Foreground(theme.Accent).Bold(true).Render("  ~ "+r.loc.T("minigame.draw")))
		}
		lines = append(lines, "")
		lines = append(lines, theme.MutedText.Render("  "+r.loc.T("challenge.continue")))
	}

	inner := strings.Join(lines, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.RPSPurple).
		Padding(1, 2).
		Width(innerW + 4)

	box := boxStyle.Render(inner)
	return lipgloss.Place(r.width, r.height, lipgloss.Center, lipgloss.Center, box)
}
