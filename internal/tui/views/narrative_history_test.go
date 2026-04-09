package views

import (
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/storage"
)

func TestBuildHistoryGroupsPreservesSessionOrder(t *testing.T) {
	now := time.Now()
	groups := buildHistoryGroups([]storage.ChatMessage{
		{SessionID: "sess-2", Turn: 3, Role: "assistant", Content: "Later scene", CreatedAt: now.Add(2 * time.Hour)},
		{SessionID: "sess-1", Turn: 1, Role: "user", Content: "Earlier scene", CreatedAt: now},
		{SessionID: "sess-1", Turn: 1, Role: "assistant", Content: "Bell tower", CreatedAt: now.Add(2 * time.Minute)},
	}, []storage.Session{
		{ID: "sess-1", StartedAt: now},
		{ID: "sess-2", StartedAt: now.Add(2 * time.Hour)},
	})

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].SessionID != "sess-1" {
		t.Fatalf("expected first session sess-1, got %s", groups[0].SessionID)
	}
	if groups[1].SessionID != "sess-2" {
		t.Fatalf("expected second session sess-2, got %s", groups[1].SessionID)
	}
}

func TestHistoryMessageMatchesQueryChecksMetadata(t *testing.T) {
	now := time.Now()
	msg := storage.ChatMessage{
		SessionID:   "sess-1",
		Turn:        7,
		Role:        "assistant",
		MessageType: "combat",
		Content:     "The tower guardian lunges forward.",
		CreatedAt:   now,
	}

	for _, query := range []string{"tower", "combat", "turn 7", "narrator"} {
		if !historyMessageMatchesQuery(msg, query) {
			t.Fatalf("expected query %q to match", query)
		}
	}
	if historyMessageMatchesQuery(msg, "crafting") {
		t.Fatalf("expected unrelated query not to match")
	}
}
