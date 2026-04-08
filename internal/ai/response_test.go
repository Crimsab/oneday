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
  "achievement_earned": {"name": "Survivor", "description": "You endured the storm", "rarity": "common", "category": "story", "context": "Turn 1"},
  "challenge": "Endure the storm"
}` + "\n```"

	nr, err := ParseNarrativeJSON(input)
	if err != nil {
		t.Fatal(err)
	}
	if nr.ASCIIArt != "~~~~~" {
		t.Errorf("ASCIIArt = %q, want %q", nr.ASCIIArt, "~~~~~")
	}
	if nr.AchievementEarned == nil {
		t.Fatal("AchievementEarned is nil, want a payload")
	}
	if nr.AchievementEarned.Name != "Survivor" {
		t.Errorf("AchievementEarned.Name = %q, want %q", nr.AchievementEarned.Name, "Survivor")
	}
	if nr.AchievementEarned.Rarity != "common" {
		t.Errorf("AchievementEarned.Rarity = %q, want %q", nr.AchievementEarned.Rarity, "common")
	}
	if nr.Challenge != "Endure the storm" {
		t.Errorf("Challenge = %q", nr.Challenge)
	}
	if v, ok := nr.StateChanges["health"]; !ok || v.(float64) != -5 {
		t.Errorf("StateChanges[health] = %v", nr.StateChanges["health"])
	}
}

func TestParseNarrativeJSONStructuredRenderingMetadata(t *testing.T) {
	input := "```json\n" + `{
  "narrative": "Lyanna leads you into Silver Vale.",
  "location": "Silver Vale",
  "scene_type": "travel",
  "dialogue_blocks": [{"speaker": "Lyanna", "role": "npc", "text": "Stay close."}],
  "entities_mentioned": [{"name": "Lyanna", "type": "npc"}, {"name": "Silver Vale", "type": "location"}],
  "event_callouts": [{"kind": "location", "title": "Silver Vale", "detail": "Location updated"}],
  "choices": [{
    "id": 1,
    "text": "Follow Lyanna",
    "intent": "follow",
    "risk": "low",
    "scope": "immediate",
    "certainty": "clear",
    "related_stats": ["perception", "agility"]
  }]
}` + "\n```"

	nr, err := ParseNarrativeJSON(input)
	if err != nil {
		t.Fatal(err)
	}
	if nr.Location != "Silver Vale" {
		t.Fatalf("Location = %q, want %q", nr.Location, "Silver Vale")
	}
	if nr.SceneType != "travel" {
		t.Fatalf("SceneType = %q, want %q", nr.SceneType, "travel")
	}
	if len(nr.DialogueBlocks) != 1 || nr.DialogueBlocks[0].Speaker != "Lyanna" {
		t.Fatalf("DialogueBlocks = %+v", nr.DialogueBlocks)
	}
	if len(nr.EntitiesMentioned) != 2 || nr.EntitiesMentioned[1].Name != "Silver Vale" {
		t.Fatalf("EntitiesMentioned = %+v", nr.EntitiesMentioned)
	}
	if len(nr.EventCallouts) != 1 || nr.EventCallouts[0].Kind != "location" {
		t.Fatalf("EventCallouts = %+v", nr.EventCallouts)
	}
	if len(nr.Choices) != 1 {
		t.Fatalf("Choices len = %d, want 1", len(nr.Choices))
	}
	if nr.Choices[0].Intent != "follow" || nr.Choices[0].Risk != "low" {
		t.Fatalf("Choice semantic fields not parsed: %+v", nr.Choices[0])
	}
	if len(nr.Choices[0].RelatedStats) != 2 || nr.Choices[0].RelatedStats[0] != "perception" {
		t.Fatalf("Choice related stats = %+v", nr.Choices[0].RelatedStats)
	}
}
