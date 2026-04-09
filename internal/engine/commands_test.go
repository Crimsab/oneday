package engine

import "testing"

func TestParseCommandRecognizesCraftAliases(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "/craft", want: "craft"},
		{input: "/crafting", want: "craft"},
		{input: "/hooks", want: "hooks"},
		{input: "/fronts", want: "hooks"},
		{input: "/front", want: "hooks"},
		{input: "/guide boss fight potente", want: "guide"},
		{input: "/talk Lyanna", want: "talk"},
		{input: "/downtime rest by the fire", want: "downtime"},
		{input: "/codex", want: "codex"},
		{input: "/investigations", want: "investigations"},
		{input: "/projects", want: "projects"},
		{input: "/project", want: "projects"},
		{input: "/characters", want: "characters"},
	}

	for _, tc := range tests {
		cmd := ParseCommand(tc.input)
		if cmd == nil {
			t.Fatalf("ParseCommand(%q) returned nil", tc.input)
		}
		if cmd.Name != tc.want {
			t.Fatalf("ParseCommand(%q).Name = %q, want %q", tc.input, cmd.Name, tc.want)
		}
	}
}
