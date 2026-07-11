package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSuggestionListTabAcceptsCurrentItem(t *testing.T) {
	model := NewSuggestionList()
	model.SetItems([]SuggestionItem{
		{Value: "/talk ", Label: "/talk"},
		{Value: "/save ", Label: "/save"},
	})

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	if cmd == nil {
		t.Fatal("expected tab to return an accept command")
	}

	msg := cmd()
	accepted, ok := msg.(SuggestionAcceptedMsg)
	if !ok {
		t.Fatalf("expected SuggestionAcceptedMsg, got %T", msg)
	}
	if accepted.Item.Value != "/talk " {
		t.Fatalf("accepted value = %q, want /talk ", accepted.Item.Value)
	}
	if updated.cursor != 0 {
		t.Fatalf("cursor = %d, want 0", updated.cursor)
	}
}

func TestSuggestionListEnterRequiresFocusedSelection(t *testing.T) {
	model := NewSuggestionList()
	model.SetItems([]SuggestionItem{
		{Value: "/talk ", Label: "/talk"},
		{Value: "/save ", Label: "/save"},
	})

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("expected enter not to accept while list is not focused")
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	if !updated.Focused() {
		t.Fatal("expected down to focus suggestions")
	}

	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected enter to accept once focused")
	}
	msg := cmd()
	accepted, ok := msg.(SuggestionAcceptedMsg)
	if !ok {
		t.Fatalf("expected SuggestionAcceptedMsg, got %T", msg)
	}
	if accepted.Item.Value != "/save " {
		t.Fatalf("accepted value = %q, want /save ", accepted.Item.Value)
	}
}
