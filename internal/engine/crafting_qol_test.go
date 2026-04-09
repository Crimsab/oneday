package engine

import (
	"encoding/json"
	"testing"

	"github.com/crimsab/oneday/internal/storage"
)

func TestGetCraftingGuidanceShowsCraftableAndNearMisses(t *testing.T) {
	recipes, err := json.Marshal([]CraftedItem{
		{Name: "Smoke Bomb", Materials: []string{"Ash", "Oil Flask"}},
		{Name: "Torch Bundle", Materials: []string{"Rag", "Oil Flask"}},
		{Name: "Field Bandage", Materials: []string{"Cloth Strip"}},
	})
	if err != nil {
		t.Fatalf("Marshal recipes: %v", err)
	}

	char := &storage.Character{
		InventoryJSON:    `[{"name":"Ash","type":"material"},{"name":"Oil Flask","type":"consumable"},{"name":"Cloth Strip","type":"material"}]`,
		KnownRecipesJSON: string(recipes),
	}

	guidance := GetCraftingGuidance(char)
	if len(guidance.CraftableNow) != 2 {
		t.Fatalf("CraftableNow = %v, want 2 recipes", guidance.CraftableNow)
	}
	if len(guidance.NearMisses) != 1 || guidance.NearMisses[0] != "Torch Bundle" {
		t.Fatalf("NearMisses = %v, want [Torch Bundle]", guidance.NearMisses)
	}
	if len(guidance.MaterialTags) == 0 {
		t.Fatal("expected material tags to be derived from inventory")
	}
}
