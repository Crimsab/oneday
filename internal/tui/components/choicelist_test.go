package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestChoiceListViewPlainChoicesStayCompact(t *testing.T) {
	choices := NewChoiceList()
	choices.SetChoices([]ChoiceItem{
		{ID: 1, Text: "Open the door"},
	})

	view := choices.View()
	if !strings.Contains(view, "1. Open the door") {
		t.Fatalf("expected plain choice text, got %q", view)
	}
	if strings.Contains(view, "intent:") || strings.Contains(view, "risk:") {
		t.Fatalf("expected no semantic metadata for plain choice, got %q", view)
	}
}

func TestChoiceListViewRendersSemanticMetadata(t *testing.T) {
	choices := NewChoiceList()
	choices.SetMood("tense")
	choices.SetChoices([]ChoiceItem{
		{
			ID:           1,
			Text:         "Talk your way through the checkpoint",
			Intent:       "social",
			Risk:         "medium",
			Scope:        "npc",
			Certainty:    "uncertain",
			RelatedStats: []string{"Charisma", "Willpower"},
		},
	})

	view := choices.View()
	for _, fragment := range []string{"intent:social", "risk:medium", "scope:npc", "certainty:uncertain", "Charisma", "Willpower"} {
		if !strings.Contains(view, fragment) {
			t.Fatalf("expected %q in rendered view, got %q", fragment, view)
		}
	}
}

func TestChoiceListKeyboardSelectionUnchanged(t *testing.T) {
	choices := NewChoiceList()
	choices.SetChoices([]ChoiceItem{
		{ID: 1, Text: "Wait"},
		{ID: 2, Text: "Move"},
	})

	updated, cmd := choices.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if cmd == nil {
		t.Fatal("expected selection command for numeric key")
	}
	msg := cmd()
	selected, ok := msg.(ChoiceSelectedMsg)
	if !ok {
		t.Fatalf("expected ChoiceSelectedMsg, got %T", msg)
	}
	if selected.ID != 2 || selected.Text != "Move" {
		t.Fatalf("unexpected selection: %+v", selected)
	}
	if updated.cursor != 0 {
		t.Fatalf("numeric selection should not move cursor, got %d", updated.cursor)
	}
}

func TestChoiceListCanFocusMetadataAndInspect(t *testing.T) {
	choices := NewChoiceList()
	choices.SetChoices([]ChoiceItem{
		{ID: 1, Text: "Observe the room", Intent: "observe", Risk: "low", RelatedStats: []string{"Perception"}},
	})

	updated, _ := choices.Update(tea.KeyMsg{Type: tea.KeyRight})
	if updated.metaCursor != 0 {
		t.Fatalf("metaCursor = %d, want 0 after first right", updated.metaCursor)
	}

	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected inspect command when metadata is focused")
	}
	msg := cmd()
	inspect, ok := msg.(ChoiceInspectRequestedMsg)
	if !ok {
		t.Fatalf("expected ChoiceInspectRequestedMsg, got %T", msg)
	}
	if inspect.ID != 1 {
		t.Fatalf("inspect ID = %d, want 1", inspect.ID)
	}
}
