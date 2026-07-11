package engine

import (
	"strings"
	"testing"
	"time"
)

// --- RPS Tests ---

func TestRPSAllCombinations(t *testing.T) {
	tests := []struct {
		player   RPSChoice
		ai       RPSChoice
		expected string
	}{
		{RPSRock, RPSRock, "draw"},
		{RPSRock, RPSPaper, "lose"},
		{RPSRock, RPSScissors, "win"},
		{RPSPaper, RPSRock, "win"},
		{RPSPaper, RPSPaper, "draw"},
		{RPSPaper, RPSScissors, "lose"},
		{RPSScissors, RPSRock, "lose"},
		{RPSScissors, RPSPaper, "win"},
		{RPSScissors, RPSScissors, "draw"},
	}

	for _, tt := range tests {
		t.Run(string(tt.player)+"_vs_"+string(tt.ai), func(t *testing.T) {
			outcome := rpsOutcome(tt.player, tt.ai)
			if outcome != tt.expected {
				t.Errorf("rpsOutcome(%s, %s) = %q, want %q", tt.player, tt.ai, outcome, tt.expected)
			}
		})
	}
}

func TestRPSToChallengeResult(t *testing.T) {
	win := RPSResult{PlayerChoice: RPSRock, AIChoice: RPSScissors, Outcome: "win"}
	result := RPSToChallengeResult(win)
	if !result.Passed {
		t.Error("win should produce Passed=true")
	}

	lose := RPSResult{PlayerChoice: RPSRock, AIChoice: RPSPaper, Outcome: "lose"}
	result = RPSToChallengeResult(lose)
	if result.Passed {
		t.Error("lose should produce Passed=false")
	}

	draw := RPSResult{PlayerChoice: RPSRock, AIChoice: RPSRock, Outcome: "draw"}
	result = RPSToChallengeResult(draw)
	if result.Passed {
		t.Error("draw should produce Passed=false")
	}
}

// --- Memory Tests ---

func TestMemoryExactMatch(t *testing.T) {
	seq := []string{"up", "down", "left", "right"}
	mc := NewMemoryChallenge(seq, 0)

	result := mc.CheckMemory([]string{"up", "down", "left", "right"})
	if !result.Passed {
		t.Errorf("exact match should pass, got: %s", result.Detail)
	}
}

func TestMemoryWrongElement(t *testing.T) {
	seq := []string{"up", "down", "left", "right"}
	mc := NewMemoryChallenge(seq, 0)

	result := mc.CheckMemory([]string{"up", "down", "right", "right"})
	if result.Passed {
		t.Errorf("wrong element should fail, got: %s", result.Detail)
	}
}

func TestMemoryWrongLength(t *testing.T) {
	seq := []string{"up", "down", "left"}
	mc := NewMemoryChallenge(seq, 0)

	result := mc.CheckMemory([]string{"up", "down"})
	if result.Passed {
		t.Errorf("wrong length should fail")
	}
}

func TestMemoryCaseInsensitive(t *testing.T) {
	seq := []string{"up", "down"}
	mc := NewMemoryChallenge(seq, 0)

	result := mc.CheckMemory([]string{"UP", "DOWN"})
	if !result.Passed {
		t.Errorf("case-insensitive match should pass, got: %s", result.Detail)
	}
}

func TestMemoryGeneratesRandomSequence(t *testing.T) {
	mc := NewMemoryChallenge(nil, 5)
	if len(mc.Sequence) != 5 {
		t.Errorf("expected 5-length sequence, got %d", len(mc.Sequence))
	}
	for _, s := range mc.Sequence {
		if s != "up" && s != "down" && s != "left" && s != "right" {
			t.Errorf("unexpected symbol in generated sequence: %q", s)
		}
	}
}

func TestMinigameMemoryUsesInjectedRNG(t *testing.T) {
	first := NewMemoryChallengeWithRNG(nil, 6, NewRNGService(42))
	second := NewMemoryChallengeWithRNG(nil, 6, NewRNGService(42))
	if strings.Join(first.Sequence, ",") != strings.Join(second.Sequence, ",") {
		t.Fatalf("seeded memory sequences differ: %v vs %v", first.Sequence, second.Sequence)
	}
}

// --- QuickTime Tests ---

func TestQuickTimeWithinLimit(t *testing.T) {
	qtc := NewQuickTimeChallenge(5.0)
	// Respond immediately (well within 5 seconds).
	response := qtc.StartTime.Add(100 * time.Millisecond)
	result := qtc.CheckQuickTime(response)
	if !result.Passed {
		t.Errorf("response within limit should pass, got: %s", result.Detail)
	}
}

func TestQuickTimeOverLimit(t *testing.T) {
	qtc := NewQuickTimeChallenge(1.0)
	// Respond after the limit.
	response := qtc.StartTime.Add(2 * time.Second)
	result := qtc.CheckQuickTime(response)
	if result.Passed {
		t.Errorf("response over limit should fail, got: %s", result.Detail)
	}
}

func TestQuickTimeExactlyAtLimit(t *testing.T) {
	qtc := NewQuickTimeChallenge(1.0)
	// Respond exactly at the limit.
	response := qtc.StartTime.Add(1 * time.Second)
	result := qtc.CheckQuickTime(response)
	if !result.Passed {
		t.Errorf("response exactly at limit should pass, got: %s", result.Detail)
	}
}

// --- Riddle Tests ---

func TestRiddleExactMatch(t *testing.T) {
	rc := NewRiddleChallenge("What has roots as nobody sees?", "a mountain")
	result := rc.CheckRiddle("a mountain")
	if !result.Passed {
		t.Errorf("exact match should pass, got: %s", result.Detail)
	}
}

func TestRiddleCaseInsensitive(t *testing.T) {
	rc := NewRiddleChallenge("What has roots as nobody sees?", "a mountain")
	result := rc.CheckRiddle("A Mountain")
	if !result.Passed {
		t.Errorf("case-insensitive match should pass, got: %s", result.Detail)
	}
}

func TestRiddleWrongAnswer(t *testing.T) {
	rc := NewRiddleChallenge("What has roots as nobody sees?", "a mountain")
	result := rc.CheckRiddle("a river")
	if result.Passed {
		t.Errorf("wrong answer should fail, got: %s", result.Detail)
	}
}

func TestRiddleWithExtraWhitespace(t *testing.T) {
	rc := NewRiddleChallenge("What has roots as nobody sees?", "a mountain")
	result := rc.CheckRiddle("  a mountain  ")
	if !result.Passed {
		t.Errorf("trimmed match should pass, got: %s", result.Detail)
	}
}

func TestRiddleEmptyAnswerFails(t *testing.T) {
	rc := NewRiddleChallenge("What has roots as nobody sees?", "a mountain")
	result := rc.CheckRiddle("")
	if result.Passed {
		t.Errorf("empty answer should fail, got: %s", result.Detail)
	}
}

func TestRiddleFailureDoesNotRevealExpectedAnswer(t *testing.T) {
	rc := NewRiddleChallenge("What has roots as nobody sees?", "a mountain")
	result := rc.CheckRiddle("a river")
	if strings.Contains(strings.ToLower(result.Detail), "mountain") {
		t.Errorf("failure detail leaked expected answer: %s", result.Detail)
	}
}

func TestRiddleOneLetterPartialDoesNotPass(t *testing.T) {
	rc := NewRiddleChallenge("What has roots as nobody sees?", "a mountain")
	result := rc.CheckRiddle("a")
	if result.Passed {
		t.Errorf("one-letter partial answer should fail, got: %s", result.Detail)
	}
}
