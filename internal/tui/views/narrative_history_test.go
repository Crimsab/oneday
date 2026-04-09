package views

import (
	"strings"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/storage"
)

func TestFormatHistoryOverlayFiltersMessages(t *testing.T) {
	now := time.Now()
	rendered := formatHistoryOverlay([]storage.ChatMessage{
		{Turn: 1, Role: "user", Content: "Look at the tower", CreatedAt: now},
		{Turn: 1, Role: "assistant", Content: "The tower answers with a bell.", CreatedAt: now},
	}, "tower")

	if !strings.Contains(rendered, "Turn 1 · Player") {
		t.Fatalf("expected player line, got %q", rendered)
	}
	if !strings.Contains(rendered, "Turn 1 · Narrator") {
		t.Fatalf("expected narrator line, got %q", rendered)
	}
}

func TestFormatHistoryOverlayShowsNoMatchMessage(t *testing.T) {
	now := time.Now()
	rendered := formatHistoryOverlay([]storage.ChatMessage{
		{Turn: 2, Role: "assistant", Content: "The square is empty.", CreatedAt: now},
	}, "bell")

	if !strings.Contains(rendered, "No history entries matched") {
		t.Fatalf("expected no-match message, got %q", rendered)
	}
}
