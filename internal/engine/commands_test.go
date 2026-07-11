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
		{input: "/advance", want: "advance"},
		{input: "/timeskip next winter", want: "timeskip"},
		{input: "/talk Lyanna", want: "talk"},
		{input: "/downtime rest by the fire", want: "downtime"},
		{input: "/codex", want: "codex"},
		{input: "/investigations", want: "investigations"},
		{input: "/projects", want: "projects"},
		{input: "/project", want: "projects"},
		{input: "/characters", want: "characters"},
		{input: "/saves", want: "load"},
		{input: "/branches", want: "branches"},
		{input: "/branch", want: "branches"},
		{input: "/fork alternate", want: "fork"},
		{input: "/rename-branch main", want: "branch-rename"},
		{input: "/checkout main", want: "checkout"},
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
