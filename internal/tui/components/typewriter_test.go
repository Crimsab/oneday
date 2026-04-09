package components

import (
	"testing"
)

// simulateTicks drives the typewriter forward by sending N tick messages.
func simulateTicks(tw *TypewriterModel, n int) {
	for i := 0; i < n; i++ {
		updated, _ := tw.Update(typewriterTickMsg{})
		*tw = updated
	}
}

func TestNewTypewriterDefaults(t *testing.T) {
	tw := NewTypewriter(0) // 0 → defaults to 80
	if tw.speed <= 0 {
		t.Error("speed should be positive")
	}
	if tw.IsActive() {
		t.Error("should not be active before SetText")
	}
	if tw.IsDone() {
		t.Error("should not be done before SetText")
	}
}

func TestSetTextStartsAnimation(t *testing.T) {
	tw := NewTypewriter(80)
	cmd := tw.SetText("Hello")
	if cmd == nil {
		t.Error("SetText should return a non-nil Cmd")
	}
	if !tw.IsActive() {
		t.Error("should be active after SetText")
	}
	if tw.IsDone() {
		t.Error("should not be done immediately after SetText")
	}
	if tw.View() != "" {
		t.Errorf("View = %q, want empty string before first tick", tw.View())
	}
}

func TestTickAdvancesOneCharacter(t *testing.T) {
	tw := NewTypewriter(80)
	tw.SetText("ABC")

	simulateTicks(&tw, 1)
	if tw.View() != "A" {
		t.Errorf("after 1 tick View = %q, want %q", tw.View(), "A")
	}

	simulateTicks(&tw, 1)
	if tw.View() != "AB" {
		t.Errorf("after 2 ticks View = %q, want %q", tw.View(), "AB")
	}
}

func TestTicksCompleteText(t *testing.T) {
	tw := NewTypewriter(80)
	tw.SetText("Hi!")

	// Tick through all characters.
	simulateTicks(&tw, 3)

	if tw.View() != "Hi!" {
		t.Errorf("View = %q, want %q", tw.View(), "Hi!")
	}
	if !tw.IsDone() {
		t.Error("should be done after all characters revealed")
	}
	if tw.IsActive() {
		t.Error("should not be active when done")
	}
}

func TestDoneMsgSentOnCompletion(t *testing.T) {
	tw := NewTypewriter(80)
	tw.SetText("X")

	updated, cmd := tw.Update(typewriterTickMsg{})
	tw = updated

	if !tw.IsDone() {
		t.Error("should be done after one tick on single-char text")
	}
	if cmd == nil {
		t.Error("Update should return a Cmd (TypewriterDoneMsg producer) on completion")
	}
}

func TestSkipShowsAllText(t *testing.T) {
	tw := NewTypewriter(80)
	tw.SetText("Long narrative text here.")
	tw.Skip()

	if tw.View() != "Long narrative text here." {
		t.Errorf("Skip View = %q", tw.View())
	}
	if !tw.IsDone() {
		t.Error("Skip should mark as done")
	}
	if tw.IsActive() {
		t.Error("Skip should deactivate animation")
	}
}

func TestAppendTextExtends(t *testing.T) {
	tw := NewTypewriter(80)
	tw.SetText("Hello")
	simulateTicks(&tw, 5) // reveal "Hello"

	// Now append more text — model was done, AppendText should restart.
	tw.done = false // simulate still-running state for append test
	tw.active = false
	cmd := tw.AppendText(" World")
	if cmd == nil {
		t.Error("AppendText on idle (not done) model should return a tick Cmd")
	}
	if !tw.IsActive() {
		t.Error("AppendText should reactivate the typewriter")
	}
}

func TestUnicodeRunes(t *testing.T) {
	tw := NewTypewriter(80)
	tw.SetText("こんにちは") // 5 Japanese runes

	simulateTicks(&tw, 3)
	got := tw.View()
	// Should show first 3 runes: こんに
	if got != "こんに" {
		t.Errorf("View after 3 ticks = %q, want %q", got, "こんに")
	}
}

func TestSpeedCustomization(t *testing.T) {
	tw := NewTypewriter(10) // slow: 100ms per char
	want := tw.speed.Milliseconds()
	if want != 100 {
		t.Errorf("speed = %dms, want 100ms", want)
	}

	tw2 := NewTypewriter(1000) // fast: 1ms per char
	if tw2.speed.Milliseconds() != 1 {
		t.Errorf("speed = %dms, want 1ms", tw2.speed.Milliseconds())
	}
}

func TestNonTickMessagesIgnored(t *testing.T) {
	type otherMsg struct{}
	tw := NewTypewriter(80)
	tw.SetText("Test")

	before := tw.displayed
	updated, cmd := tw.Update(otherMsg{})
	if updated.displayed != before {
		t.Error("non-tick message should not advance displayed count")
	}
	if cmd != nil {
		t.Error("non-tick message should return nil Cmd")
	}
}

func TestVisibleRuneCountIgnoresANSISequences(t *testing.T) {
	text := "\x1b[31mHello\x1b[0m world"
	if got := visibleRuneCount(text); got != 11 {
		t.Fatalf("visibleRuneCount() = %d, want 11", got)
	}
}

func TestTypewriterViewDoesNotSplitANSISequences(t *testing.T) {
	tw := NewTypewriter(80)
	tw.SetText("\x1b[31mHi\x1b[0m!")

	simulateTicks(&tw, 1)
	if got := tw.View(); got != "\x1b[31mH" {
		t.Fatalf("after 1 tick View = %q, want first styled rune without broken ANSI", got)
	}

	simulateTicks(&tw, 1)
	if got := tw.View(); got != "\x1b[31mHi\x1b[0m" {
		t.Fatalf("after 2 ticks View = %q, want styled word with reset sequence", got)
	}

	simulateTicks(&tw, 1)
	if got := tw.View(); got != "\x1b[31mHi\x1b[0m!" {
		t.Fatalf("after 3 ticks View = %q, want full string", got)
	}
}

func TestSetTextInstantCountsVisibleRunesOnly(t *testing.T) {
	tw := NewTypewriter(80)
	text := "\x1b[32mReady\x1b[0m"
	tw.SetTextInstant(text)

	if tw.displayed != 5 {
		t.Fatalf("displayed = %d, want visible rune count 5", tw.displayed)
	}
	if tw.View() != text {
		t.Fatalf("View() = %q, want full ANSI text", tw.View())
	}
}
