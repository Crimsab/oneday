package components

import (
	"strings"
	"testing"
)

func TestStatusBarFallsBackToTwoLinesOnNarrowWidths(t *testing.T) {
	bar := NewStatusBar()
	bar.SetWidth(40)
	bar.SetData(StatusBarData{
		Vitals: []Vital{
			{Label: "HP", Current: 10, Max: 10},
			{Label: "ENERGY", Current: 8, Max: 10},
		},
		Model:            "test-status-model",
		Latency:          10400,
		TimeToFirstToken: 5500,
		PromptTokens:     4980,
		CompletionTokens: 1036,
		ReasoningTokens:  193,
		TotalTokens:      6016,
		Streamed:         true,
	})

	view := bar.View()
	if !strings.Contains(view, "\n") {
		t.Fatalf("expected multiline status bar on narrow width, got %q", view)
	}
}
