package engine

import (
	"fmt"
	"strings"
	"time"

	"github.com/crimsab/oneday/internal/game/contracts"
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
	passed := r.Outcome != "lose"
	detail := fmt.Sprintf("RPS: you chose %s, opponent chose %s → %s", r.PlayerChoice, r.AIChoice, strings.ToUpper(r.Outcome))
	outcome := OutcomeFromLegacy(passed, 0)
	if r.Outcome == "draw" {
		outcome.Degree = contracts.OutcomeSuccessWithCost
		outcome.Costs = []contracts.OutcomeEffect{{Kind: "tempo", Amount: 1, Detail: "The contest remains unresolved."}}
		outcome.Consequences = []string{"The player avoids defeat, but gains no decisive advantage."}
	}
	return &ChallengeResult{
		Passed:  passed,
		Detail:  detail,
		Outcome: &outcome,
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

	for i, expected := range mc.Sequence {
		if i < len(playerSequence) && strings.EqualFold(playerSequence[i], expected) {
			correct++
		}
	}

	degree := contracts.OutcomeHardFailure
	if total > 0 {
		ratio := float64(correct) / float64(total)
		switch {
		case correct == total && len(playerSequence) == total:
			degree = contracts.OutcomeFullSuccess
		case ratio >= 0.75:
			degree = contracts.OutcomeSuccessWithCost
		case ratio >= 0.5:
			degree = contracts.OutcomeFailureWithProgress
		}
	}
	passed := degree == contracts.OutcomeFullSuccess || degree == contracts.OutcomeSuccessWithCost
	detail := fmt.Sprintf("Memory: %d/%d correct → %s", correct, total, strings.ToUpper(strings.ReplaceAll(string(degree), "_", " ")))
	graded := contracts.OutcomeEnvelope{Version: contracts.ChallengeProtocolVersion, Degree: degree, Difficulty: total, Total: correct, Margin: correct - total}
	applyDefaultOutcomeBudget(&graded, DefaultOutcomePolicy("", "balanced"))
	return &ChallengeResult{
		Passed:  passed,
		Detail:  detail,
		Outcome: &graded,
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
	}
}

// CheckQuickTime validates that the player responded within the time limit.
func (qtc *QuickTimeChallenge) CheckQuickTime(responseTime time.Time) *ChallengeResult {
	elapsed := responseTime.Sub(qtc.StartTime)
	return qtc.CheckQuickTimeElapsed(elapsed)
}

// CheckQuickTimeElapsed resolves from a supplied duration so browser, TUI,
// autoplay, replay, and tests do not depend on the local wall clock.
func (qtc *QuickTimeChallenge) CheckQuickTimeElapsed(elapsed time.Duration) *ChallengeResult {
	if elapsed < 0 {
		elapsed = 0
	}
	degree := contracts.OutcomeHardFailure
	switch {
	case elapsed*2 <= qtc.TimeLimit:
		degree = contracts.OutcomeFullSuccess
	case elapsed <= qtc.TimeLimit:
		degree = contracts.OutcomeSuccessWithCost
	}
	passed := degree == contracts.OutcomeFullSuccess || degree == contracts.OutcomeSuccessWithCost

	detail := fmt.Sprintf("QuickTime: responded in %.2fs (limit %.2fs) → %s",
		elapsed.Seconds(), qtc.TimeLimit.Seconds(), strings.ToUpper(strings.ReplaceAll(string(degree), "_", " ")))
	graded := contracts.OutcomeEnvelope{Version: contracts.ChallengeProtocolVersion, Degree: degree, Difficulty: int(qtc.TimeLimit.Milliseconds()), Total: int(elapsed.Milliseconds()), Margin: int(qtc.TimeLimit.Milliseconds() - elapsed.Milliseconds())}
	applyDefaultOutcomeBudget(&graded, DefaultOutcomePolicy("", "balanced"))
	return &ChallengeResult{
		Passed:  passed,
		Detail:  detail,
		Outcome: &graded,
	}
}

// RiddleChallenge represents a riddle challenge.
type RiddleChallenge struct {
	Riddle  string   // the riddle text (from AI)
	Answer  string   // the correct answer (from AI, hidden from player)
	Answers []string // explicit accepted aliases; never inferred by substring
}

// NewRiddleChallenge creates a riddle challenge from AI spec.
func NewRiddleChallenge(riddle, answer string, aliases ...string) *RiddleChallenge {
	return &RiddleChallenge{
		Riddle:  riddle,
		Answer:  answer,
		Answers: append([]string{answer}, aliases...),
	}
}

// CheckRiddle validates the player's answer (case-insensitive, trimmed).
// It accepts exact matches and meaningful partial matches, but never empty or
// tiny one-letter guesses.
func (rc *RiddleChallenge) CheckRiddle(playerAnswer string) *ChallengeResult {
	normalized := normalizeRiddleAnswer(playerAnswer)
	passed := false
	if normalized != "" {
		answers := rc.Answers
		if len(answers) == 0 {
			answers = []string{rc.Answer}
		}
		for _, answer := range answers {
			if normalized == normalizeRiddleAnswer(answer) {
				passed = true
				break
			}
		}
	}

	outcome := "PASS"
	if !passed {
		outcome = "FAIL"
	}
	detail := fmt.Sprintf("Riddle: your answer %q → %s", playerAnswer, outcome)
	graded := OutcomeFromLegacy(passed, 0)
	return &ChallengeResult{
		Passed:  passed,
		Detail:  detail,
		Outcome: &graded,
	}
}

func normalizeRiddleAnswer(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == ' ' {
			return r
		}
		switch r {
		case 'à', 'á', 'â', 'ä':
			return 'a'
		case 'è', 'é', 'ê', 'ë':
			return 'e'
		case 'ì', 'í', 'î', 'ï':
			return 'i'
		case 'ò', 'ó', 'ô', 'ö':
			return 'o'
		case 'ù', 'ú', 'û', 'ü':
			return 'u'
		}
		return ' '
	}, value)
	articles := map[string]bool{"a": true, "an": true, "the": true, "il": true, "lo": true, "la": true, "i": true, "gli": true, "le": true, "un": true, "uno": true, "una": true}
	words := strings.Fields(value)
	filtered := words[:0]
	for _, word := range words {
		if !articles[word] {
			filtered = append(filtered, word)
		}
	}
	return strings.Join(filtered, " ")
}
