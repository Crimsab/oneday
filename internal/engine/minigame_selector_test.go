package engine

import "testing"

func TestMiniGameSelectorUsesFitAccessibilityCooldownAndPreference(t *testing.T) {
	selection, err := SelectMiniGame(DefaultMiniGameCandidates(), MiniGameSelectionContext{
		NarrativeTags: []string{"social", "comedy", "zero-combat"}, CurrentTurn: 12, Difficulty: 50,
		TimingFreeOnly: true, PreferredKinds: []MiniGameType{MiniGameComedy},
		Recent: []MiniGameUsage{{Kind: MiniGameNegotiation, Turn: 11}, {Kind: MiniGameComedy, Turn: 5}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if selection.Definition.Kind != MiniGameComedy {
		t.Fatalf("selected %s, want comedy: %+v", selection.Definition.Kind, selection)
	}
	if selection.Definition.Rules["selection_reason"] == "" {
		t.Fatal("selection reason was not exposed")
	}
}

func TestMiniGameSelectorAvoidsImmediateRepetition(t *testing.T) {
	selection, err := SelectMiniGame(DefaultMiniGameCandidates(), MiniGameSelectionContext{
		NarrativeTags: []string{"mystery", "evidence"}, CurrentTurn: 10, Difficulty: 50,
		TimingFreeOnly: true, Recent: []MiniGameUsage{{Kind: MiniGameDeduction, Turn: 9}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if selection.Definition.Kind == MiniGameDeduction {
		t.Fatalf("cooldown repeated deduction: %+v", selection)
	}
}

func TestMiniGameSelectorFailsWhenPolicyExcludesEverything(t *testing.T) {
	_, err := SelectMiniGame([]MiniGameCandidate{{Definition: DefaultMiniGameDefinition(MiniGameQuickTime), Reflex: true}}, MiniGameSelectionContext{TimingFreeOnly: true})
	if err == nil {
		t.Fatal("timing-free policy accepted reflex-only pool")
	}
}
