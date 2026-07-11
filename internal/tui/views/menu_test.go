package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMenuQuitShortcutLowercase(t *testing.T) {
	menu := NewMenuModel()
	updated, cmd := menu.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if updated.cursor != 0 {
		t.Fatalf("expected cursor to stay on current item, got %d", updated.cursor)
	}
	if cmd == nil {
		t.Fatal("expected quit command for lowercase q")
	}

	msg := cmd()
	selected, ok := msg.(MenuSelectedMsg)
	if !ok {
		t.Fatalf("expected MenuSelectedMsg, got %T", msg)
	}
	if selected.Action != ActionQuit {
		t.Fatalf("expected ActionQuit, got %v", selected.Action)
	}
}

func TestMenuQuitShortcutUppercase(t *testing.T) {
	menu := NewMenuModel()
	_, cmd := menu.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Q'}})
	if cmd == nil {
		t.Fatal("expected quit command for uppercase Q")
	}

	msg := cmd()
	selected, ok := msg.(MenuSelectedMsg)
	if !ok {
		t.Fatalf("expected MenuSelectedMsg, got %T", msg)
	}
	if selected.Action != ActionQuit {
		t.Fatalf("expected ActionQuit, got %v", selected.Action)
	}
}
