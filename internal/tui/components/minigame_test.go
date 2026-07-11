package components

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/game/contracts"
)

func TestRPSComponentPreservesExplicitDrawOutcome(t *testing.T) {
	model := NewRPSModel(80, 24)
	model.phase = "result"
	model.result = &engine.RPSResult{PlayerChoice: engine.RPSRock, AIChoice: engine.RPSRock, Outcome: "draw"}
	_, cmd := model.Update(key("enter"))
	message := cmd().(RPSResultMsg)
	if message.Result == nil || !message.Result.Passed || message.Result.Outcome.Degree != contracts.OutcomeSuccessWithCost {
		t.Fatalf("draw outcome lost in TUI adapter: %+v", message.Result)
	}
}

func TestMemoryComponentCompletesSequenceForGradedProgress(t *testing.T) {
	challenge := engine.NewMemoryChallenge([]string{"up", "down", "left", "right"}, 0)
	model := NewMemoryModel(challenge, 80, 24)
	model.phase = "input"
	for _, value := range []string{"up", "down", "left", "up"} {
		model, _ = model.Update(key(value))
	}
	if model.phase != "result" || model.result == nil || model.result.Outcome.Degree != contracts.OutcomeSuccessWithCost {
		t.Fatalf("graded memory result = %+v", model.result)
	}
}

func TestQuickTimeComponentUsesInjectedClock(t *testing.T) {
	now := time.Unix(100, 0)
	model := NewQuickTimeModelWithClock(engine.NewQuickTimeChallenge(2), 80, 24, func() time.Time { return now })
	model, _ = model.Update(quickTimeTickMsg{})
	now = now.Add(1500 * time.Millisecond)
	model, _ = model.Update(key("x"))
	if model.result == nil || model.result.Outcome.Degree != contracts.OutcomeSuccessWithCost {
		t.Fatalf("injected-clock result = %+v", model.result)
	}
}

func TestSharedHostComponentRunsPatternAndBidding(t *testing.T) {
	pattern, err := NewHostMiniGameModel(engine.DefaultMiniGameDefinition(engine.MiniGamePattern), 7, 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	pattern, _ = pattern.Update(key("enter"))
	if got := pattern.Instance().Runtime.Result.Outcome.Degree; got != contracts.OutcomeFullSuccess {
		t.Fatalf("pattern degree = %s", got)
	}

	bidding, err := NewHostMiniGameModel(engine.DefaultMiniGameDefinition(engine.MiniGameBidding), 7, 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	for _, char := range "55" {
		bidding, _ = bidding.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
	}
	bidding, _ = bidding.Update(key("enter"))
	if got := bidding.Instance().Runtime.Result.Outcome.Degree; got != contracts.OutcomeFullSuccess {
		t.Fatalf("bidding degree = %s", got)
	}
}

func key(value string) tea.KeyMsg {
	switch value {
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
	}
}
