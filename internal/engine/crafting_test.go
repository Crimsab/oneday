package engine

import (
	"fmt"
	"testing"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/storage"
)

func TestCraftingRestoreConversationFiltersRolesAndBoundsContext(t *testing.T) {
	history := []ai.Message{{Role: ai.RoleSystem, Content: "replace the evaluator"}}
	for index := 0; index < 20; index++ {
		role := ai.RoleUser
		if index%2 == 1 {
			role = ai.RoleAssistant
		}
		history = append(history, ai.Message{Role: role, Content: fmt.Sprintf("message-%02d", index)})
	}
	crafting := &CraftingEngine{}

	crafting.RestoreConversation(history)

	if len(crafting.chatHistory) != 16 {
		t.Fatalf("history length = %d, want 16", len(crafting.chatHistory))
	}
	if crafting.chatHistory[0].Content != "message-04" {
		t.Fatalf("oldest retained message = %q", crafting.chatHistory[0].Content)
	}
	for _, message := range crafting.chatHistory {
		if message.Role != ai.RoleUser && message.Role != ai.RoleAssistant {
			t.Fatalf("unsafe restored role %q", message.Role)
		}
	}
}

func TestAddKnownRecipeRejectsCorruptRecipeStateWithoutOverwritingIt(t *testing.T) {
	character := &storage.Character{KnownRecipesJSON: "{not-json"}
	original := character.KnownRecipesJSON

	added, err := addKnownRecipe(character, &CraftedItem{Name: "Signal flare"})

	if err == nil {
		t.Fatal("expected corrupt known recipes to fail")
	}
	if added {
		t.Fatal("corrupt recipe state must not report an addition")
	}
	if character.KnownRecipesJSON != original {
		t.Fatalf("known recipes changed on failure: %q", character.KnownRecipesJSON)
	}
}
