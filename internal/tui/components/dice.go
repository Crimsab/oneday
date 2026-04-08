package components

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/tui/theme"
)

// DiceResultMsg is emitted when the dice animation finishes and player dismisses.
type DiceResultMsg struct {
	Passed bool
}

// diceTickMsg advances the dice animation frame.
type diceTickMsg struct{}

// ModDisplay is a modifier shown in the dice overlay.
type ModDisplay struct {
	Source string
	Value  int
}

// DiceModel renders an animated d100 dice roll with modifiers and result.
type DiceModel struct {
	// Configuration (set once)
	FinalRoll  int          // the actual d100 result (pre-computed by engine)
	Total      int          // roll + sum(modifiers)
	Difficulty int          // threshold to pass
	Modifiers  []ModDisplay // source + value for each modifier
	Passed     bool         // final outcome

	// Animation state
	displayedNumber int
	frame           int
	maxFrames       int
	done            bool
	active          bool
	tickInterval    time.Duration
	dismissed       bool
	width           int
	height          int
}

// NewDiceModel creates a dice roll animation.
// All values are pre-computed by the engine — this is purely visual.
func NewDiceModel(roll, total, difficulty int, modifiers []ModDisplay, passed bool, width, height int) DiceModel {
	return DiceModel{
		FinalRoll:       roll,
		Total:           total,
		Difficulty:      difficulty,
		Modifiers:       modifiers,
		Passed:          passed,
		displayedNumber: rand.Intn(100) + 1,
		maxFrames:       30,
		tickInterval:    50 * time.Millisecond,
		width:           width,
		height:          height,
	}
}

// Start begins the dice animation. Returns the first tick cmd.
func (d *DiceModel) Start() tea.Cmd {
	d.active = true
	d.frame = 0
	d.done = false
	return d.tick()
}

func (d *DiceModel) tick() tea.Cmd {
	interval := d.tickInterval
	// Slow down in last 5 frames for dramatic effect.
	if d.frame >= d.maxFrames-5 {
		interval = 100 * time.Millisecond
	}
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return diceTickMsg{}
	})
}

// Update handles animation ticks and dismissal.
func (d DiceModel) Update(msg tea.Msg) (DiceModel, tea.Cmd) {
	switch msg := msg.(type) {
	case diceTickMsg:
		if !d.active {
			return d, nil
		}
		d.frame++
		if d.frame >= d.maxFrames {
			// Final frame: show actual roll.
			d.displayedNumber = d.FinalRoll
			d.active = false
			d.done = true
			return d, nil
		}
		d.displayedNumber = rand.Intn(100) + 1
		return d, d.tick()

	case tea.KeyMsg:
		if d.done && !d.dismissed {
			_ = msg
			d.dismissed = true
			return d, func() tea.Msg {
				return DiceResultMsg{Passed: d.Passed}
			}
		}
	}
	return d, nil
}

// IsDone returns true when the animation has finished.
func (d DiceModel) IsDone() bool {
	return d.done
}

// View renders the dice animation as a centered overlay.
func (d DiceModel) View() string {
	innerW := 36

	// Title
	title := theme.ChallengeOverlay.Copy().
		Foreground(theme.DiceGold).Bold(true).
		Render("🎲  DICE ROLL")

	// Large animated number
	numStr := fmt.Sprintf("[ %d ]", d.displayedNumber)
	numStyle := lipgloss.NewStyle().Foreground(theme.DiceGold).Bold(true)
	numberLine := lipgloss.NewStyle().Width(innerW).Align(lipgloss.Center).Render(
		numStyle.Render(numStr),
	)

	var lines []string
	lines = append(lines, title)
	lines = append(lines, "")
	lines = append(lines, numberLine)
	lines = append(lines, "")

	if d.done {
		// Show breakdown
		lines = append(lines, fmt.Sprintf("  Roll:       %3d", d.FinalRoll))
		for _, mod := range d.Modifiers {
			sign := "+"
			if mod.Value < 0 {
				sign = ""
			}
			lines = append(lines, fmt.Sprintf("  %-12s %s%d", mod.Source+":", sign, mod.Value))
		}
		lines = append(lines, "  "+strings.Repeat("─", innerW-4))
		lines = append(lines, fmt.Sprintf("  Total:      %3d", d.Total))
		lines = append(lines, fmt.Sprintf("  Difficulty: %3d", d.Difficulty))
		lines = append(lines, "")

		var resultLine string
		if d.Passed {
			resultLine = theme.DicePassed.Render("  ✓  PASSED!")
		} else {
			resultLine = theme.DiceFailed.Render("  ✗  FAILED!")
		}
		lines = append(lines, resultLine)
		lines = append(lines, "")
		lines = append(lines, theme.MutedText.Render("  Press any key to continue"))
	} else {
		lines = append(lines, theme.MutedText.Render("  Rolling..."))
	}

	inner := strings.Join(lines, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.DiceGold).
		Padding(1, 2).
		Width(innerW + 4)

	box := boxStyle.Render(inner)
	return lipgloss.Place(d.width, d.height, lipgloss.Center, lipgloss.Center, box)
}
