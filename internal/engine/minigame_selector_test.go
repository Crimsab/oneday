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

func TestAutomaticMiniGameTagsMapNarrativeSituationToSemanticFamilies(t *testing.T) {
	cases := []struct {
		text string
		want MiniGameType
	}{
		{"Connect the contradictory clues and expose the false identity", MiniGameDeduction},
		{"Convince the guard using leverage without starting a fight", MiniGameNegotiation},
		{"Decode the ritual mechanism and complete its sequence", MiniGamePattern},
		{"Outbid the rival at the auction without exceeding the budget", MiniGameBidding},
		{"Cross-examine the witness during the trial", MiniGameCourtroom},
		{"Recover from the embarrassing banter with a callback", MiniGameComedy},
	}
	for _, tc := range cases {
		selection, err := SelectMiniGame(DefaultMiniGameCandidates(), MiniGameSelectionContext{
			NarrativeTags: automaticMiniGameTags(tc.text), CurrentTurn: 20,
			Difficulty: 50, TimingFreeOnly: true,
		})
		if err != nil {
			t.Fatalf("%q: %v", tc.text, err)
		}
		if selection.Definition.Kind != tc.want {
			t.Fatalf("%q selected %s, want %s: %+v", tc.text, selection.Definition.Kind, tc.want, selection)
		}
	}
}
