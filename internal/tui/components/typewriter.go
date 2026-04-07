package components

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TypewriterDoneMsg is sent when the typewriter finishes rendering all text.
type TypewriterDoneMsg struct{}

// typewriterTickMsg is the internal timer message that advances one character.
type typewriterTickMsg struct{}

// TypewriterModel renders text character-by-character at a configurable speed.
// It is a standalone Bubbletea model: embed it in a parent model and forward
// messages to its Update method.
//
// Usage:
//
//	tw := components.NewTypewriter(80)   // 80 chars/sec
//	cmd := tw.SetText("Hello, world!")   // start animating
//	// In parent Update: tw, cmd = tw.Update(msg)
//	// In parent View:   rendered := tw.View()
type TypewriterModel struct {
	fullText  string        // complete target text
	displayed int           // rune count currently visible
	speed     time.Duration // delay between characters
	active    bool          // animation is running
	done      bool          // all characters have been shown
}

// NewTypewriter creates a TypewriterModel.
// speed is in characters per second; values <= 0 default to 80.
func NewTypewriter(speed int) TypewriterModel {
	if speed <= 0 {
		speed = 80
	}
	return TypewriterModel{
		speed: time.Second / time.Duration(speed),
	}
}

// SetText replaces the current text and restarts the typewriter effect from
// the beginning.  Returns the first tick Cmd; pass it to the Bubbletea runtime.
func (t *TypewriterModel) SetText(text string) tea.Cmd {
	t.fullText = text
	t.displayed = 0
	t.active = true
	t.done = false
	return t.tick()
}

// AppendText adds text to the end (useful for streaming chunks).
// Restarts ticking if the model was idle but not yet done.
func (t *TypewriterModel) AppendText(text string) tea.Cmd {
	t.fullText += text
	if !t.active && !t.done {
		t.active = true
		return t.tick()
	}
	// If already ticking the goroutine will naturally consume the new runes.
	return nil
}

// Skip immediately reveals all text and marks the animation done.
func (t *TypewriterModel) Skip() {
	t.displayed = len([]rune(t.fullText))
	t.active = false
	t.done = true
}

// IsActive reports whether the typewriter is currently animating.
func (t TypewriterModel) IsActive() bool { return t.active }

// IsDone reports whether all text has been revealed.
func (t TypewriterModel) IsDone() bool { return t.done }

// View returns the currently visible portion of the text.
func (t TypewriterModel) View() string {
	runes := []rune(t.fullText)
	if t.displayed >= len(runes) {
		return t.fullText
	}
	return string(runes[:t.displayed])
}

// Update processes Bubbletea messages.  Only typewriterTickMsg is handled
// internally; all other messages are ignored and passed through unchanged.
func (t TypewriterModel) Update(msg tea.Msg) (TypewriterModel, tea.Cmd) {
	if _, ok := msg.(typewriterTickMsg); !ok {
		return t, nil
	}
	if !t.active {
		return t, nil
	}

	runes := []rune(t.fullText)
	t.displayed++
	if t.displayed >= len(runes) {
		t.displayed = len(runes)
		t.active = false
		t.done = true
		return t, func() tea.Msg { return TypewriterDoneMsg{} }
	}
	return t, t.tick()
}

// tick returns a Cmd that fires typewriterTickMsg after one character interval.
func (t TypewriterModel) tick() tea.Cmd {
	return tea.Tick(t.speed, func(time.Time) tea.Msg {
		return typewriterTickMsg{}
	})
}
