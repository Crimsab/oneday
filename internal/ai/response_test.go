package ai

import (
	"testing"
)

func TestParseNarrativeJSONExtractsBlock(t *testing.T) {
	input := `Here is the story so far.

` + "```json" + `
{
  "narrative": "You enter the dark forest.",
  "choices": [{"id": 1, "text": "Go north"}, {"id": 2, "text": "Turn back"}],
  "mood": "tense"
}
` + "```" + `

Some trailing prose.`

	nr, err := ParseNarrativeJSON(input)
	if err != nil {
		t.Fatalf("ParseNarrativeJSON: %v", err)
	}
	if nr == nil {
		t.Fatal("expected non-nil NarrativeResponse")
	}
	if nr.Narrative != "You enter the dark forest." {
		t.Errorf("Narrative = %q", nr.Narrative)
	}
	if len(nr.Choices) != 2 {
		t.Errorf("Choices len = %d, want 2", len(nr.Choices))
	}
	if nr.Mood != "tense" {
		t.Errorf("Mood = %q, want %q", nr.Mood, "tense")
	}
}

func TestParseNarrativeJSONReturnsNilForPureProse(t *testing.T) {
	input := "Just a plain narrative without any JSON block."
	nr, err := ParseNarrativeJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nr != nil {
		t.Errorf("expected nil for pure prose, got %+v", nr)
	}
}

func TestParseNarrativeJSONErrorsOnInvalidJSON(t *testing.T) {
	input := "```json\n{not valid json}\n```"
	_, err := ParseNarrativeJSON(input)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestExtractNarrativeStripsJSONBlock(t *testing.T) {
	// The JSON block is replaced with empty string; TrimSpace collapses surrounding whitespace.
	input := "Intro text.\n\n```json\n{\"narrative\":\"test\"}\n```\n\nOutro text."
	got := ExtractNarrative(input)
	// After stripping the block the two surrounding blank lines collapse into
	// extra newlines; TrimSpace in ExtractNarrative removes outer whitespace only.
	// Verify that both prose fragments are present and no JSON remains.
	if got == input {
		t.Error("ExtractNarrative should have removed the JSON block")
	}
	if !containsStr(got, "Intro text.") {
		t.Errorf("ExtractNarrative missing intro: %q", got)
	}
	if !containsStr(got, "Outro text.") {
		t.Errorf("ExtractNarrative missing outro: %q", got)
	}
	if containsStr(got, "```json") {
		t.Errorf("ExtractNarrative still contains JSON fence: %q", got)
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStrHelper(s, sub))
}

func containsStrHelper(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestExtractNarrativeNoop(t *testing.T) {
	input := "Just prose, no JSON."
	got := ExtractNarrative(input)
	if got != input {
		t.Errorf("ExtractNarrative = %q, want %q", got, input)
	}
}

func TestNarrativeResponseAllFields(t *testing.T) {
	input := "```json\n" + `{
  "narrative": "Storm approaches.",
  "choices": [{"id": 1, "text": "Seek shelter", "mood": "cautious"}],
  "state_changes": {"health": -5},
  "mood": "stormy",
  "ascii_art": "~~~~~",
  "achievement_earned": "survivor",
  "challenge": "Endure the storm"
}` + "\n```"

	nr, err := ParseNarrativeJSON(input)
	if err != nil {
		t.Fatal(err)
	}
	if nr.ASCIIArt != "~~~~~" {
		t.Errorf("ASCIIArt = %q, want %q", nr.ASCIIArt, "~~~~~")
	}
	if nr.AchievementEarned != "survivor" {
		t.Errorf("AchievementEarned = %q", nr.AchievementEarned)
	}
	if nr.Challenge != "Endure the storm" {
		t.Errorf("Challenge = %q", nr.Challenge)
	}
	if v, ok := nr.StateChanges["health"]; !ok || v.(float64) != -5 {
		t.Errorf("StateChanges[health] = %v", nr.StateChanges["health"])
	}
}
