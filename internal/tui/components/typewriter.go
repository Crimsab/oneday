package components

import (
	"strings"
	"time"
	"unicode/utf8"

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
	fullText     string        // complete target text
	displayed    int           // visible rune count currently revealed
	totalVisible int           // total visible rune count (ANSI excluded)
	speed        time.Duration // delay between characters
	active       bool          // animation is running
	done         bool          // all characters have been shown
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
	t.totalVisible = visibleRuneCount(text)
	if t.totalVisible == 0 {
		t.active = false
		t.done = true
		return nil
	}
	t.active = true
	t.done = false
	return t.tick()
}

// SetTextInstant replaces the current text and marks it as fully rendered.
func (t *TypewriterModel) SetTextInstant(text string) {
	t.fullText = text
	t.totalVisible = visibleRuneCount(text)
	t.displayed = t.totalVisible
	t.active = false
	t.done = true
}

// AppendText adds text to the end (useful for streaming chunks).
// Restarts ticking if the model was idle but not yet done.
func (t *TypewriterModel) AppendText(text string) tea.Cmd {
	addedVisible := visibleRuneCount(text)
	t.fullText += text
	t.totalVisible += addedVisible
	if addedVisible == 0 {
		return nil
	}
	if !t.active && (!t.done || t.displayed < t.totalVisible) {
		t.done = false
		t.active = true
		return t.tick()
	}
	// If already ticking the goroutine will naturally consume the new runes.
	return nil
}

// Skip immediately reveals all text and marks the animation done.
func (t *TypewriterModel) Skip() {
	t.displayed = t.totalVisible
	t.active = false
	t.done = true
}

// IsActive reports whether the typewriter is currently animating.
func (t TypewriterModel) IsActive() bool { return t.active }

// IsDone reports whether all text has been revealed.
func (t TypewriterModel) IsDone() bool { return t.done }

// View returns the currently visible portion of the text.
func (t TypewriterModel) View() string {
	if t.displayed >= t.totalVisible {
		return t.fullText
	}
	return visiblePrefixWithANSI(t.fullText, t.displayed)
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

	t.displayed++
	if t.displayed >= t.totalVisible {
		t.displayed = t.totalVisible
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

func visibleRuneCount(text string) int {
	count := 0
	for i := 0; i < len(text); {
		if seqLen := ansiSequenceLength(text, i); seqLen > 0 {
			i += seqLen
			continue
		}
		_, size := utf8.DecodeRuneInString(text[i:])
		if size <= 0 {
			break
		}
		count++
		i += size
	}
	return count
}

func visiblePrefixWithANSI(text string, visibleLimit int) string {
	if visibleLimit <= 0 || text == "" {
		return ""
	}

	var out strings.Builder
	visible := 0
	afterLimit := false

	for i := 0; i < len(text); {
		if seqLen := ansiSequenceLength(text, i); seqLen > 0 {
			if visible < visibleLimit || afterLimit {
				out.WriteString(text[i : i+seqLen])
			}
			i += seqLen
			continue
		}

		if visible >= visibleLimit {
			break
		}

		_, size := utf8.DecodeRuneInString(text[i:])
		if size <= 0 {
			break
		}
		out.WriteString(text[i : i+size])
		i += size
		visible++
		if visible >= visibleLimit {
			afterLimit = true
		}
	}

	return out.String()
}

func ansiSequenceLength(text string, start int) int {
	if start < 0 || start >= len(text) || text[start] != 0x1b || start+1 >= len(text) {
		return 0
	}

	switch text[start+1] {
	case '[':
		for i := start + 2; i < len(text); i++ {
			if text[i] >= '@' && text[i] <= '~' {
				return i - start + 1
			}
		}
	case ']':
		for i := start + 2; i < len(text); i++ {
			if text[i] == 0x07 {
				return i - start + 1
			}
			if text[i] == 0x1b && i+1 < len(text) && text[i+1] == '\\' {
				return i - start + 2
			}
		}
	default:
		return 2
	}

	return len(text) - start
}
