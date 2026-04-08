package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/tui/theme"
)

// MemoryResultMsg is emitted when memory challenge resolves.
type MemoryResultMsg struct {
	Passed bool
}

// memoryShowTickMsg advances the show phase.
type memoryShowTickMsg struct{}

// MemoryModel handles the memory sequence mini-game.
type MemoryModel struct {
	challenge   *engine.MemoryChallenge
	phase       string   // "show", "input", "result"
	showIndex   int      // which symbol to show during "show" phase
	playerInput []string // what the player has entered so far
	result      *engine.ChallengeResult
	failed      bool   // true if player made a wrong input
	wrongAt     int    // index of wrong input
	width       int
	height      int
}

var memorySymbols = map[string]string{
	"up":    "↑",
	"down":  "↓",
	"left":  "←",
	"right": "→",
}

// NewMemoryModel creates a memory sequence challenge.
func NewMemoryModel(challenge *engine.MemoryChallenge, width, height int) MemoryModel {
	return MemoryModel{
		challenge: challenge,
		phase:     "show",
		showIndex: -1, // starts at -1, first tick advances to 0
		width:     width,
		height:    height,
	}
}

// Start begins the show phase.
func (m *MemoryModel) Start() tea.Cmd {
	return tea.Tick(800*time.Millisecond, func(time.Time) tea.Msg {
		return memoryShowTickMsg{}
	})
}

// Update handles ticks and input.
func (m MemoryModel) Update(msg tea.Msg) (MemoryModel, tea.Cmd) {
	switch msg := msg.(type) {
	case memoryShowTickMsg:
		if m.phase != "show" {
			return m, nil
		}
		m.showIndex++
		if m.showIndex >= len(m.challenge.Sequence) {
			// Transition to input phase.
			m.phase = "input"
			m.playerInput = []string{}
			return m, nil
		}
		return m, tea.Tick(900*time.Millisecond, func(time.Time) tea.Msg {
			return memoryShowTickMsg{}
		})

	case tea.KeyMsg:
		if m.phase == "input" {
			var direction string
			switch msg.String() {
			case "up":
				direction = "up"
			case "down":
				direction = "down"
			case "left":
				direction = "left"
			case "right":
				direction = "right"
			default:
				return m, nil
			}
			idx := len(m.playerInput)
			expected := m.challenge.Sequence[idx]
			if !strings.EqualFold(direction, expected) {
				// Wrong key — fail immediately.
				m.failed = true
				m.wrongAt = idx
				m.playerInput = append(m.playerInput, direction)
				result := m.challenge.CheckMemory(m.playerInput)
				m.result = result
				m.phase = "result"
				return m, nil
			}
			m.playerInput = append(m.playerInput, direction)
			if len(m.playerInput) >= len(m.challenge.Sequence) {
				result := m.challenge.CheckMemory(m.playerInput)
				m.result = result
				m.phase = "result"
				return m, nil
			}
		} else if m.phase == "result" {
			passed := m.result != nil && m.result.Passed
			return m, func() tea.Msg {
				return MemoryResultMsg{Passed: passed}
			}
		}
	}
	return m, nil
}

// View renders the memory challenge.
func (m MemoryModel) View() string {
	innerW := 38
	var lines []string

	titleStyle := lipgloss.NewStyle().Foreground(theme.MemoryTeal).Bold(true)
	lines = append(lines, titleStyle.Render("🧠  MEMORY SEQUENCE"))
	lines = append(lines, "")

	switch m.phase {
	case "show":
		if m.showIndex < 0 || m.showIndex >= len(m.challenge.Sequence) {
			lines = append(lines, "  Get ready...")
		} else {
			sym := m.challenge.Sequence[m.showIndex]
			display := memorySymbols[sym]
			if display == "" {
				display = sym
			}
			large := lipgloss.NewStyle().Foreground(theme.MemoryTeal).Bold(true).
				Render(fmt.Sprintf("       %s", display))
			lines = append(lines, large)
			lines = append(lines, "")
			lines = append(lines, fmt.Sprintf("  Showing %d / %d...", m.showIndex+1, len(m.challenge.Sequence)))
		}
		lines = append(lines, "")
		lines = append(lines, theme.MutedText.Render("  Watch carefully!"))

	case "input":
		total := len(m.challenge.Sequence)
		current := len(m.playerInput)
		lines = append(lines, fmt.Sprintf("  Your turn! Step %d / %d", current+1, total))
		lines = append(lines, "")
		// Show progress bar of entered symbols.
		var progress strings.Builder
		progress.WriteString("  ")
		for i := 0; i < total; i++ {
			if i < len(m.playerInput) {
				sym := m.playerInput[i]
				display := memorySymbols[sym]
				if display == "" {
					display = sym
				}
				progress.WriteString(lipgloss.NewStyle().Foreground(theme.Success).Render(display))
			} else if i == len(m.playerInput) {
				progress.WriteString(lipgloss.NewStyle().Foreground(theme.Highlight).Render("_"))
			} else {
				progress.WriteString(theme.MutedText.Render("_"))
			}
			progress.WriteString(" ")
		}
		lines = append(lines, progress.String())
		lines = append(lines, "")
		lines = append(lines, theme.MutedText.Render("  ↑ ↓ ← → to input sequence"))

	case "result":
		if m.result != nil && m.result.Passed {
			lines = append(lines, lipgloss.NewStyle().Foreground(theme.Success).Bold(true).Render("  ✓ CORRECT! Well done!"))
		} else {
			// Show correct vs wrong.
			var progress strings.Builder
			progress.WriteString("  ")
			for i, sym := range m.challenge.Sequence {
				display := memorySymbols[sym]
				if display == "" {
					display = sym
				}
				if i < len(m.playerInput) {
					if i == m.wrongAt && m.failed {
						progress.WriteString(lipgloss.NewStyle().Foreground(theme.Danger).Render(display))
					} else {
						progress.WriteString(lipgloss.NewStyle().Foreground(theme.Success).Render(display))
					}
				} else {
					progress.WriteString(theme.MutedText.Render(display))
				}
				progress.WriteString(" ")
			}
			lines = append(lines, lipgloss.NewStyle().Foreground(theme.Danger).Bold(true).Render("  ✗ Wrong sequence!"))
			lines = append(lines, "")
			lines = append(lines, "  Correct: "+progress.String())
		}
		lines = append(lines, "")
		lines = append(lines, theme.MutedText.Render("  Press any key to continue"))
	}

	inner := strings.Join(lines, "\n")
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.MemoryTeal).
		Padding(1, 2).
		Width(innerW + 4)

	box := boxStyle.Render(inner)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}
