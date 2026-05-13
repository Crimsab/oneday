package engine

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/crimsab/oneday/internal/storage"
)

// FormatPrivateThoughtsView builds a text overlay listing saved NPC private thoughts.
func FormatPrivateThoughtsView(db *storage.DB, storyID string) string {
	if db == nil || strings.TrimSpace(storyID) == "" {
		return "Private thoughts unavailable right now."
	}

	npcs, err := db.ListNPCs(storyID)
	if err != nil || len(npcs) == 0 {
		return "No private NPC thoughts recorded yet."
	}

	var sb strings.Builder
	sb.WriteString("=== NPC Private Thoughts ===\n")

	shown := 0
	for _, npc := range npcs {
		var thoughts []string
		if err := json.Unmarshal([]byte(npc.PrivateThoughts), &thoughts); err != nil || len(thoughts) == 0 {
			continue
		}

		shown++
		sb.WriteString("\n")
		if role := strings.TrimSpace(npc.Role); role != "" {
			sb.WriteString(fmt.Sprintf("--- %s (%s) ---\n", npc.Name, role))
		} else {
			sb.WriteString(fmt.Sprintf("--- %s ---\n", npc.Name))
		}
		for _, thought := range thoughts {
			thought = strings.TrimSpace(thought)
			if thought == "" {
				continue
			}
			sb.WriteString("- " + thought + "\n")
		}
	}

	if shown == 0 {
		return "No private NPC thoughts recorded yet."
	}

	return strings.TrimRight(sb.String(), "\n")
}
