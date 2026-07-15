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

// QuickTimeResultMsg is emitted when quick-time resolves.
type QuickTimeResultMsg struct {
	Result *engine.ChallengeResult
}

// quickTimeTickMsg decrements the countdown.
type quickTimeTickMsg struct{}

// QuickTimeModel handles the timed key-press challenge.
type QuickTimeModel struct {
	challenge *engine.QuickTimeChallenge
	phase     string // "ready", "active", "result"
	timeLeft  time.Duration
	pressed   bool
	passed    bool
	result    *engine.ChallengeResult
	now       func() time.Time
	width     int
	height    int
	loc       appi18n.Localizer
}

// NewQuickTimeModel creates a quick-time challenge.
func NewQuickTimeModel(challenge *engine.QuickTimeChallenge, width, height int, localizers ...appi18n.Localizer) QuickTimeModel {
	return NewQuickTimeModelWithClock(challenge, width, height, time.Now, localizers...)
}

// NewQuickTimeModelWithClock makes timing injectable for deterministic tests
// and alternate hosts while retaining the normal TUI wall-clock adapter.
func NewQuickTimeModelWithClock(challenge *engine.QuickTimeChallenge, width, height int, now func() time.Time, localizers ...appi18n.Localizer) QuickTimeModel {
	return QuickTimeModel{
		challenge: challenge,
		phase:     "ready",
		timeLeft:  challenge.TimeLimit,
		width:     width,
		height:    height,
		now:       now,
		loc:       componentLocalizer(localizers),
	}
}

// Start begins the ready countdown then transitions to active.
func (q *QuickTimeModel) Start() tea.Cmd {
	return tea.Tick(800*time.Millisecond, func(time.Time) tea.Msg {
		return quickTimeTickMsg{}
	})
}

// Update handles ticks and key press.
func (q QuickTimeModel) Update(msg tea.Msg) (QuickTimeModel, tea.Cmd) {
	switch msg.(type) {
	case quickTimeTickMsg:
		switch q.phase {
		case "ready":
			q.phase = "active"
			q.challenge.StartTime = q.now()
			q.timeLeft = q.challenge.TimeLimit
			return q, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
				return quickTimeTickMsg{}
			})
		case "active":
			elapsed := q.now().Sub(q.challenge.StartTime)
			q.timeLeft = q.challenge.TimeLimit - elapsed
			if q.timeLeft <= 0 {
				q.timeLeft = 0
				q.pressed = false
				q.passed = false
				q.result = q.challenge.CheckQuickTimeElapsed(q.challenge.TimeLimit + time.Millisecond)
				q.phase = "result"
				return q, nil
			}
			return q, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
				return quickTimeTickMsg{}
			})
		}

	default:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			_ = keyMsg
			switch q.phase {
			case "active":
				// Any key press = success.
				result := q.challenge.CheckQuickTime(q.now())
				q.pressed = true
				q.passed = result.Passed
				q.result = result
				q.phase = "result"
				return q, nil
			case "result":
				result := q.result
				return q, func() tea.Msg {
					return QuickTimeResultMsg{Result: result}
				}
			}
		}
	}
	return q, nil
}

// View renders the quick-time challenge.
func (q QuickTimeModel) View() string {
	innerW := 36
	var lines []string

	titleStyle := lipgloss.NewStyle().Foreground(theme.QuickTimeOrange).Bold(true)
	lines = append(lines, titleStyle.Render("⚡  "+q.loc.T("minigame.quick_title")))
	lines = append(lines, "")

	switch q.phase {
	case "ready":
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.Accent).Bold(true).Render("  "+q.loc.T("minigame.ready")))
		lines = append(lines, "")
		lines = append(lines, theme.MutedText.Render("  "+q.loc.T("minigame.time_limit", q.challenge.TimeLimit.Seconds())))

	case "active":
		promptStyle := lipgloss.NewStyle().Foreground(theme.QuickTimeOrange).Bold(true).Blink(true)
		lines = append(lines, promptStyle.Render("  >>> "+q.loc.T("minigame.press_key")+" <<<"))
		lines = append(lines, "")

		// Countdown bar.
		total := q.challenge.TimeLimit
		remaining := q.timeLeft
		if remaining < 0 {
			remaining = 0
		}
		barWidth := innerW - 4
		filled := int(float64(barWidth) * float64(remaining) / float64(total))
		if filled < 0 {
			filled = 0
		}
		if filled > barWidth {
			filled = barWidth
		}

		var barColor lipgloss.Color
		ratio := float64(remaining) / float64(total)
		switch {
		case ratio > 0.6:
			barColor = theme.Success
		case ratio > 0.3:
			barColor = theme.Accent
		default:
			barColor = theme.Danger
		}

		bar := lipgloss.NewStyle().Foreground(barColor).Render(strings.Repeat("█", filled)) +
			theme.MutedText.Render(strings.Repeat("░", barWidth-filled))
		lines = append(lines, fmt.Sprintf("  [%s] %.1fs", bar, remaining.Seconds()))

	case "result":
		if q.passed {
			lines = append(lines, lipgloss.NewStyle().Foreground(theme.Success).Bold(true).Render("  ✓ "+q.loc.T("minigame.fast")))
		} else {
			lines = append(lines, lipgloss.NewStyle().Foreground(theme.Danger).Bold(true).Render("  ✗ "+q.loc.T("minigame.slow")))
		}
		lines = append(lines, "")
		lines = append(lines, theme.MutedText.Render("  "+q.loc.T("challenge.continue")))
	}

	inner := strings.Join(lines, "\n")
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.QuickTimeOrange).
		Padding(1, 2).
		Width(innerW + 4)

	box := boxStyle.Render(inner)
	return lipgloss.Place(q.width, q.height, lipgloss.Center, lipgloss.Center, box)
}
