package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/storage"
)

func TestSocialDuelStartSeedsState(t *testing.T) {
	engine := NewSocialDuelEngineWithRollFn(func() int { return 10 })
	char := newTestChar()
	npc := &storage.NPC{
		Name:             "Lyanna",
		Disposition:      20,
		RelationshipJSON: `{"trust":25,"respect":15,"fear":0,"debt":5}`,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	state, err := engine.Start(SocialDuelSpec{
		NPCName:      npc.Name,
		Objective:    "Get Lyanna to betray the checkpoint rota",
		PlayerStance: SocialStanceMeasured,
		PlayerLeverage: []SocialLeverage{
			{ID: "lev-1", Label: "You know where her brother is hidden", Strength: 2, Consumable: true},
		},
	}, char, npc)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	if state.Status != SocialDuelActive || state.Round != 1 {
		t.Fatalf("state = %+v, want active round 1", state)
	}
	if state.PlayerComposure < 6 || state.NPCComposure < 6 {
		t.Fatalf("state composure too low: %+v", state)
	}
	if len(state.PlayerLeverage) != 1 {
		t.Fatalf("player leverage = %+v, want seeded leverage", state.PlayerLeverage)
	}
}

func TestSocialDuelExposeConsumesLeverageAndWinsRound(t *testing.T) {
	rolls := []int{18, 5}
	engine := NewSocialDuelEngineWithRollFn(func() int {
		value := rolls[0]
		rolls = rolls[1:]
		return value
	})
	char := newTestChar()
	npc := &storage.NPC{
		Name:             "Lyanna",
		Disposition:      10,
		RelationshipJSON: `{"trust":10,"respect":15,"fear":5,"debt":20}`,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	state, err := engine.Start(SocialDuelSpec{
		NPCName:      npc.Name,
		Objective:    "Force a confession",
		PlayerStance: SocialStanceBold,
		PlayerLeverage: []SocialLeverage{
			{ID: "lev-1", Label: "Signed ledger page", Strength: 3, Consumable: true},
		},
	}, char, npc)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	result, err := engine.ResolveRound(state, char, npc, SocialActionExpose, SocialActionConcede)
	if err != nil {
		t.Fatalf("ResolveRound: %v", err)
	}

	if result.NPCDamage == 0 {
		t.Fatalf("result = %+v, want NPC damage from exposed leverage", result)
	}
	if result.Consumed == nil || result.Consumed.Label != "Signed ledger page" {
		t.Fatalf("result consumed leverage = %+v, want spent ledger leverage", result.Consumed)
	}
	if !state.PlayerLeverage[0].Spent {
		t.Fatalf("state leverage = %+v, want consumable leverage marked spent", state.PlayerLeverage)
	}
}

func TestSocialDuelLossProducesFailForward(t *testing.T) {
	rolls := []int{3, 18}
	engine := NewSocialDuelEngineWithRollFn(func() int {
		value := rolls[0]
		rolls = rolls[1:]
		return value
	})
	char := newTestChar()
	npc := &storage.NPC{
		Name:             "Magistrate Vey",
		Disposition:      60,
		RelationshipJSON: `{"trust":-10,"respect":5,"fear":0,"debt":0}`,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	state := &SocialDuelState{
		NPCName:         npc.Name,
		Objective:       "Leave with the seized papers",
		PlayerStance:    SocialStanceBold,
		NPCStance:       SocialStanceGuarded,
		Status:          SocialDuelActive,
		Round:           2,
		PlayerComposure: 1,
		NPCComposure:    7,
		PlayerPatience:  1,
		NPCPatience:     3,
	}

	result, err := engine.ResolveRound(state, char, npc, SocialActionEscalate, SocialActionPressure)
	if err != nil {
		t.Fatalf("ResolveRound: %v", err)
	}

	if !result.Resolved || result.FailForward == nil {
		t.Fatalf("result = %+v, want resolved fail-forward loss", result)
	}
	if state.Status != SocialDuelResolved || state.Winner != npc.Name {
		t.Fatalf("state = %+v, want NPC victory", state)
	}
	if result.FailForward.Kind != "suspicion" {
		t.Fatalf("fail forward = %+v, want suspicion fallout", result.FailForward)
	}
}

func TestSocialDuelWithdrawEndsPlayableConcession(t *testing.T) {
	engine := NewSocialDuelEngineWithRollFn(func() int { return 10 })
	char := newTestChar()
	npc := &storage.NPC{
		Name:             "Lyanna",
		Disposition:      5,
		RelationshipJSON: `{}`,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	state, err := engine.Start(SocialDuelSpec{
		NPCName:   npc.Name,
		Objective: "Get through the gate tonight",
	}, char, npc)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	result, err := engine.ResolveRound(state, char, npc, SocialActionWithdraw, SocialActionAppeal)
	if err != nil {
		t.Fatalf("ResolveRound: %v", err)
	}

	if !result.Resolved || result.FailForward == nil {
		t.Fatalf("result = %+v, want resolved concession", result)
	}
	if state.Outcome != "withdrawal" {
		t.Fatalf("state outcome = %q, want withdrawal", state.Outcome)
	}
	if !strings.Contains(result.FailForward.Detail, "playable") {
		t.Fatalf("fail forward detail = %q, want playable continuation", result.FailForward.Detail)
	}
}

func TestSocialDuelChooseNPCActionUsesLeverageWhenPressed(t *testing.T) {
	engine := NewSocialDuelEngine()
	state := &SocialDuelState{
		Status:       SocialDuelActive,
		Round:        2,
		Tempo:        3,
		NPCComposure: 5,
		NPCPatience:  1,
		NPCLeverage: []SocialLeverage{
			{Label: "Witness statement", Strength: 2},
		},
	}

	action := engine.ChooseNPCAction(state)
	if action != SocialActionExpose {
		t.Fatalf("ChooseNPCAction = %q, want expose", action)
	}
}

func TestSocialDuelChooseNPCActionConcedesWhenComposureBreaks(t *testing.T) {
	engine := NewSocialDuelEngine()
	state := &SocialDuelState{
		Status:       SocialDuelActive,
		Round:        4,
		NPCComposure: 1,
		NPCPatience:  2,
	}

	action := engine.ChooseNPCAction(state)
	if action != SocialActionConcede {
		t.Fatalf("ChooseNPCAction = %q, want concede", action)
	}
}
