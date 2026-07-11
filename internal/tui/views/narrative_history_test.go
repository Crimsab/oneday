package views

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

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

func TestHistoryBrowserArrowKeysMoveVisibleSelection(t *testing.T) {
	now := time.Now()
	browser := newHistoryBrowser("Test Story", []storage.ChatMessage{
		{SessionID: "sess-1", Turn: 1, Role: "assistant", Content: "Opening scene", CreatedAt: now},
		{SessionID: "sess-2", Turn: 2, Role: "assistant", Content: "Second scene", CreatedAt: now.Add(time.Hour)},
	}, []storage.Session{
		{ID: "sess-1", StartedAt: now},
		{ID: "sess-2", StartedAt: now.Add(time.Hour)},
	}, "", 120, 40)

	updated, _ := browser.Update(tea.KeyMsg{Type: tea.KeyDown})
	if updated.cursor != 1 {
		t.Fatalf("expected down arrow to move cursor to second session, got %d", updated.cursor)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyUp})
	if updated.cursor != 0 {
		t.Fatalf("expected up arrow to move cursor back to first session, got %d", updated.cursor)
	}
}

func TestHistoryBrowserArrowKeysExitSearchAndMoveSelection(t *testing.T) {
	now := time.Now()
	browser := newHistoryBrowser("Test Story", []storage.ChatMessage{
		{SessionID: "sess-1", Turn: 1, Role: "assistant", Content: "Opening scene", CreatedAt: now},
		{SessionID: "sess-2", Turn: 2, Role: "assistant", Content: "Second scene", CreatedAt: now.Add(time.Hour)},
	}, []storage.Session{
		{ID: "sess-1", StartedAt: now},
		{ID: "sess-2", StartedAt: now.Add(time.Hour)},
	}, "scene", 120, 40)

	if browser.focus != historyFocusSearch {
		t.Fatalf("expected initial query to focus search")
	}

	updated, _ := browser.Update(tea.KeyMsg{Type: tea.KeyDown})
	if updated.focus != historyFocusTimeline {
		t.Fatalf("expected down arrow to leave search focus")
	}
	if updated.cursor != 1 {
		t.Fatalf("expected down arrow to move to second session, got %d", updated.cursor)
	}
}

func TestHistoryBrowserScrollDoesNotChangeSelection(t *testing.T) {
	now := time.Now()
	browser := newHistoryBrowser("Test Story", []storage.ChatMessage{
		{SessionID: "sess-1", Turn: 1, Role: "assistant", Content: "Opening scene", CreatedAt: now},
		{SessionID: "sess-1", Turn: 2, Role: "assistant", Content: "More details for the first session to make the viewport taller.", CreatedAt: now.Add(time.Minute)},
		{SessionID: "sess-2", Turn: 3, Role: "assistant", Content: "Second scene", CreatedAt: now.Add(time.Hour)},
		{SessionID: "sess-2", Turn: 4, Role: "assistant", Content: "Even more details for the second session to keep content scrollable.", CreatedAt: now.Add(time.Hour + time.Minute)},
	}, []storage.Session{
		{ID: "sess-1", StartedAt: now},
		{ID: "sess-2", StartedAt: now.Add(time.Hour)},
	}, "", 100, 18)

	browser.cursor = 1
	browser.refreshViewport(false)

	mouse := tea.MouseMsg(tea.MouseEvent{
		X:      browser.boxX + 3,
		Y:      browser.viewportTop + 1,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelDown,
	})
	updated, _ := browser.Update(mouse)
	if updated.cursor != 1 {
		t.Fatalf("expected wheel scroll to keep cursor on second session, got %d", updated.cursor)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	if updated.cursor != 1 {
		t.Fatalf("expected page down to keep cursor on second session, got %d", updated.cursor)
	}
}

func TestHistoryBrowserTabStaysInBrowser(t *testing.T) {
	now := time.Now()
	browser := newHistoryBrowser("Test Story", []storage.ChatMessage{
		{SessionID: "sess-1", Turn: 1, Role: "assistant", Content: "Opening scene", CreatedAt: now},
	}, []storage.Session{
		{ID: "sess-1", StartedAt: now},
	}, "", 100, 18)

	updated, _ := browser.Update(tea.KeyMsg{Type: tea.KeyTab})
	if updated.focus != historyFocusSearch {
		t.Fatalf("expected tab to focus history search")
	}
}
