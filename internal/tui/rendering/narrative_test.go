package rendering

import (
	"strings"
	"testing"

	"github.com/crimsab/oneday/internal/engine"
)

func TestRenderNarrativeMarkdownUsesCalloutsHighlightsAndDialogue(t *testing.T) {
	rendered := RenderNarrativeMarkdown(NarrativeInput{
		Narrative: "Lyanna studies the Silver Vale from the ruined tower.",
		EventCallouts: []engine.EventCallout{
			{Kind: "location", Title: "Silver Vale", Detail: "Location updated"},
		},
		DialogueBlocks: []engine.DialogueBlock{
			{Speaker: "Lyanna", Role: "npc", Text: "We cannot stay in Silver Vale."},
		},
		KnownEntities: []KnownEntity{
			{Name: "Lyanna", Kind: "npc"},
			{Name: "Silver Vale", Kind: "location"},
		},
	})

	if !strings.Contains(rendered, "**[LOCATION] Silver Vale**") {
		t.Fatalf("expected callout in rendered markdown, got %q", rendered)
	}
	if !strings.Contains(rendered, "**Lyanna** studies the **Silver Vale**") {
		t.Fatalf("expected highlighted narrative text, got %q", rendered)
	}
	if !strings.Contains(rendered, "**Lyanna:** _\"We cannot stay in **Silver Vale**.\"_") {
		t.Fatalf("expected speaker-styled dialogue block, got %q", rendered)
	}
}

func TestRenderNarrativeMarkdownFallsBackToPlainNarrative(t *testing.T) {
	rendered := RenderNarrativeMarkdown(NarrativeInput{
		Narrative: "The corridor is silent.",
	})

	if rendered != "The corridor is silent." {
		t.Fatalf("expected plain narrative fallback, got %q", rendered)
	}
}
