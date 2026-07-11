package engine

import (
	"fmt"
	"strings"
	"time"
)

// RPSChoice represents a rock-paper-scissors choice.
type RPSChoice string

const (
	RPSRock     RPSChoice = "rock"
	RPSPaper    RPSChoice = "paper"
	RPSScissors RPSChoice = "scissors"
)

// RPSResult holds the outcome of a rock-paper-scissors game.
type RPSResult struct {
	PlayerChoice RPSChoice
	AIChoice     RPSChoice
	Outcome      string // "win", "lose", "draw"
}

// ResolveRPS resolves a rock-paper-scissors challenge.
// playerChoice is the player's pick. AI choice is random.
func ResolveRPS(playerChoice RPSChoice) RPSResult {
	return ResolveRPSWithRNG(playerChoice, defaultRNGService())
}

func ResolveRPSWithRNG(playerChoice RPSChoice, rng *RNGService) RPSResult {
	choices := []RPSChoice{RPSRock, RPSPaper, RPSScissors}
	roll := rng.Roll("minigame.rps", len(choices))
	aiChoice := choices[roll.Raw-1]

	outcome := rpsOutcome(playerChoice, aiChoice)
	return RPSResult{
		PlayerChoice: playerChoice,
		AIChoice:     aiChoice,
		Outcome:      outcome,
	}
}

// rpsOutcome returns "win", "lose", or "draw" from the player's perspective.
func rpsOutcome(player, ai RPSChoice) string {
	if player == ai {
		return "draw"
	}
	// Win conditions for player:
	// Rock beats Scissors, Scissors beats Paper, Paper beats Rock
	wins := map[RPSChoice]RPSChoice{
		RPSRock:     RPSScissors,
		RPSScissors: RPSPaper,
		RPSPaper:    RPSRock,
	}
	if wins[player] == ai {
		return "win"
	}
	return "lose"
}

// RPSToChallengeResult converts an RPSResult to a ChallengeResult.
func RPSToChallengeResult(r RPSResult) *ChallengeResult {
	passed := r.Outcome == "win"
	detail := fmt.Sprintf("RPS: you chose %s, opponent chose %s → %s", r.PlayerChoice, r.AIChoice, strings.ToUpper(r.Outcome))
	return &ChallengeResult{
		Passed: passed,
		Detail: detail,
	}
}

// MemoryChallenge represents a memory sequence challenge state.
type MemoryChallenge struct {
	Sequence []string // the sequence to memorize (e.g., ["up", "down", "left", "right"])
	Length   int
}

// NewMemoryChallenge creates a memory challenge from an AI-provided sequence.
// If sequence is nil/empty, generates a random one of given length using default symbols.
func NewMemoryChallenge(sequence []string, length int) *MemoryChallenge {
	return NewMemoryChallengeWithRNG(sequence, length, defaultRNGService())
}

func NewMemoryChallengeWithRNG(sequence []string, length int, rng *RNGService) *MemoryChallenge {
	if len(sequence) > 0 {
		return &MemoryChallenge{
			Sequence: sequence,
			Length:   len(sequence),
		}
	}
	// Generate a random sequence from default symbols.
	symbols := []string{"up", "down", "left", "right"}
	if length <= 0 {
		length = 4
	}
	generated := make([]string, length)
	for i := range generated {
		roll := rng.Roll("minigame.memory", len(symbols))
		generated[i] = symbols[roll.Raw-1]
	}
	return &MemoryChallenge{
		Sequence: generated,
		Length:   length,
	}
}

// CheckMemory validates the player's attempt against the sequence.
// All elements must match for a pass.
func (mc *MemoryChallenge) CheckMemory(playerSequence []string) *ChallengeResult {
	correct := 0
	total := len(mc.Sequence)

	if len(playerSequence) != total {
		detail := fmt.Sprintf("Memory: wrong length — expected %d steps, got %d → FAIL", total, len(playerSequence))
		return &ChallengeResult{
			Passed: false,
			Detail: detail,
		}
	}

	for i, expected := range mc.Sequence {
		if i < len(playerSequence) && strings.EqualFold(playerSequence[i], expected) {
			correct++
		}
	}

	passed := correct == total
	outcome := "PASS"
	if !passed {
		outcome = "FAIL"
	}
	detail := fmt.Sprintf("Memory: %d/%d correct → %s", correct, total, outcome)
	return &ChallengeResult{
		Passed: passed,
		Detail: detail,
	}
}

// QuickTimeChallenge represents a timed key-press challenge.
type QuickTimeChallenge struct {
	TimeLimit time.Duration // how long the player has
	StartTime time.Time     // when the challenge started (set by TUI)
}

// NewQuickTimeChallenge creates a quick-time challenge.
// timeLimitSeconds is the number of seconds the player has to respond.
func NewQuickTimeChallenge(timeLimitSeconds float64) *QuickTimeChallenge {
	if timeLimitSeconds <= 0 {
		timeLimitSeconds = 3.0
	}
	return &QuickTimeChallenge{
		TimeLimit: time.Duration(timeLimitSeconds * float64(time.Second)),
		StartTime: time.Now(),
	}
}

// CheckQuickTime validates that the player responded within the time limit.
func (qtc *QuickTimeChallenge) CheckQuickTime(responseTime time.Time) *ChallengeResult {
	elapsed := responseTime.Sub(qtc.StartTime)
	passed := elapsed <= qtc.TimeLimit

	outcome := "PASS"
	if !passed {
		outcome = "FAIL"
	}
	detail := fmt.Sprintf("QuickTime: responded in %.2fs (limit %.2fs) → %s",
		elapsed.Seconds(), qtc.TimeLimit.Seconds(), outcome)
	return &ChallengeResult{
		Passed: passed,
		Detail: detail,
	}
}

// RiddleChallenge represents a riddle challenge.
type RiddleChallenge struct {
	Riddle string // the riddle text (from AI)
	Answer string // the correct answer (from AI, hidden from player)
}

// NewRiddleChallenge creates a riddle challenge from AI spec.
func NewRiddleChallenge(riddle, answer string) *RiddleChallenge {
	return &RiddleChallenge{
		Riddle: riddle,
		Answer: answer,
	}
}

// CheckRiddle validates the player's answer (case-insensitive, trimmed).
// It accepts exact matches and meaningful partial matches, but never empty or
// tiny one-letter guesses.
func (rc *RiddleChallenge) CheckRiddle(playerAnswer string) *ChallengeResult {
	normalized := strings.TrimSpace(strings.ToLower(playerAnswer))
	expected := strings.TrimSpace(strings.ToLower(rc.Answer))

	passed := false
	if normalized != "" && expected != "" {
		passed = normalized == expected ||
			(len(normalized) >= 3 && strings.Contains(expected, normalized)) ||
			(len(expected) >= 3 && strings.Contains(normalized, expected))
	}

	outcome := "PASS"
	if !passed {
		outcome = "FAIL"
	}
	detail := fmt.Sprintf("Riddle: your answer %q → %s", playerAnswer, outcome)
	return &ChallengeResult{
		Passed: passed,
		Detail: detail,
	}
}
