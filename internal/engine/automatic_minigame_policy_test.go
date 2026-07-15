package engine

import (
	"testing"

	"github.com/crimsab/oneday/internal/storage"
)

func TestDefaultAutomaticMiniGamePolicyPreservesExistingBehavior(t *testing.T) {
	policy := DefaultAutomaticMiniGamePolicy()
	if !policy.Enabled || !policy.TimingFreeOnly || !policy.UseCooldowns {
		t.Fatalf("unexpected default automatic mini-game policy: %#v", policy)
	}
}

func TestAutomaticMiniGamePolicyCanDisableAutomaticChallenges(t *testing.T) {
	narrator := &Narrator{story: &storage.Story{ID: "story-1"}, automaticMiniGamePolicy: DefaultAutomaticMiniGamePolicy()}
	narrator.SetAutomaticMiniGamePolicy(AutomaticMiniGamePolicy{Enabled: false})
	narrative := &NarrativeResponse{Challenges: []*ChallengeSpec{{Type: ChallengeMiniGame, Description: "Decode the lock"}}}

	instance, err := narrator.prepareAutomaticMiniGame(narrative, 3)
	if err != nil {
		t.Fatalf("prepareAutomaticMiniGame returned an error: %v", err)
	}
	if instance != nil {
		t.Fatalf("expected automatic challenge to be disabled, got %#v", instance)
	}
}
