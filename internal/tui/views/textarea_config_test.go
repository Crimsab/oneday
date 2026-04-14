package views

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewGameTextareaConfig(t *testing.T) {
	input := newGameTextarea("Test", 4)

	if input.ShowLineNumbers {
		t.Fatalf("expected line numbers disabled")
	}

	if input.CharLimit != 0 {
		t.Fatalf("expected char limit disabled, got %d", input.CharLimit)
	}

	if input.Height() != 4 {
		t.Fatalf("expected height 4, got %d", input.Height())
	}

	wantKeys := []string{"alt+enter", "shift+enter", "ctrl+j"}
	if got := input.KeyMap.InsertNewline.Keys(); !reflect.DeepEqual(got, wantKeys) {
		t.Fatalf("unexpected newline keys: got %v want %v", got, wantKeys)
	}
}

func TestNewGameTextareaAltEnterInsertsNewline(t *testing.T) {
	input := newGameTextarea("Test", 4)
	input.Focus()
	input.SetWidth(40)
	input.SetValue("hello")

	updated, _ := input.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true})

	if got := updated.Value(); got != "hello\n" {
		t.Fatalf("expected alt+enter to insert newline, got %q", got)
	}
}

func TestIsTextareaNewlineKey(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.KeyMsg
		want bool
	}{
		{
			name: "plain enter is submit",
			msg:  tea.KeyMsg{Type: tea.KeyEnter},
			want: false,
		},
		{
			name: "alt enter is multiline",
			msg:  tea.KeyMsg{Type: tea.KeyEnter, Alt: true},
			want: true,
		},
		{
			name: "ctrl+j is multiline",
			msg:  tea.KeyMsg{Type: tea.KeyCtrlJ},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isTextareaNewlineKey(tc.msg); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
