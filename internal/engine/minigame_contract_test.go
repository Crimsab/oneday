package engine

import (
	"encoding/json"
	"os"
	"testing"
)

func TestCanonicalMiniGameFixtureIsPortableAndPlayerSafe(t *testing.T) {
	raw, err := os.ReadFile("../../contracts/minigame-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Instance MiniGameInstance `json:"instance"`
		Input    MiniGameInput    `json:"input"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if err := fixture.Instance.Validate(); err != nil {
		t.Fatal(err)
	}
	if fixture.Input.Action != "submit" || len(fixture.Instance.Definition.Answers) != 0 {
		t.Fatalf("fixture leaked answers or lost input: %+v", fixture)
	}
}
