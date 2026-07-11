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

	if !strings.Contains(rendered, "> **[LOCATION] Silver Vale**") {
		t.Fatalf("expected callout in rendered markdown, got %q", rendered)
	}
	if !strings.Contains(rendered, "**Lyanna** studies the **`Silver Vale`**") {
		t.Fatalf("expected highlighted narrative text, got %q", rendered)
	}
	if !strings.Contains(rendered, "> **Lyanna:** _\"We cannot stay in **`Silver Vale`**.\"_") {
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

func TestRenderNarrativeMarkdownHighlightsLocationWithoutLocationCallout(t *testing.T) {
	rendered := RenderNarrativeMarkdown(NarrativeInput{
		Narrative: "You arrive at Silver Vale under a red sky.",
		KnownEntities: []KnownEntity{
			{Name: "Silver Vale", Kind: "location"},
		},
	})

	if !strings.Contains(rendered, "**`Silver Vale`**") {
		t.Fatalf("expected location highlight without explicit callout, got %q", rendered)
	}
	if strings.Contains(rendered, "[LOCATION]") {
		t.Fatalf("did not expect synthetic location callout, got %q", rendered)
	}
}

func TestRenderNarrativeMarkdownStripsQuotedNarrativeWhenDialogueBlocksExist(t *testing.T) {
	rendered := RenderNarrativeMarkdown(NarrativeInput{
		Narrative: "Lyanna narrows her eyes. Lyanna says, 'We cannot stay in Silver Vale.'",
		DialogueBlocks: []engine.DialogueBlock{
			{Speaker: "Lyanna", Role: "npc", Text: "We cannot stay in Silver Vale."},
		},
		KnownEntities: []KnownEntity{
			{Name: "Lyanna", Kind: "npc"},
			{Name: "Silver Vale", Kind: "location"},
		},
	})

	if strings.Contains(rendered, "'We cannot stay in Silver Vale.'") {
		t.Fatalf("expected duplicate single-quoted prose to be removed, got %q", rendered)
	}
	if !strings.Contains(rendered, "> **Lyanna:** _\"We cannot stay in **`Silver Vale`**.\"_") {
		t.Fatalf("expected structured dialogue block to remain, got %q", rendered)
	}
}

func TestRenderNarrativeMarkdownPromotesSingleQuotedDialogueWithoutBlocks(t *testing.T) {
	rendered := RenderNarrativeMarkdown(NarrativeInput{
		Narrative: "Lyanna says, 'Stay close.' The corridor is silent.",
		KnownEntities: []KnownEntity{
			{Name: "Lyanna", Kind: "npc"},
		},
	})

	if !strings.Contains(rendered, "> **Lyanna:** _\"Stay close.\"_") {
		t.Fatalf("expected single-quoted dialogue to become a dialogue block, got %q", rendered)
	}
	if !strings.Contains(rendered, "The corridor is silent.") {
		t.Fatalf("expected surrounding narration to remain, got %q", rendered)
	}
}

func TestRenderNarrativeMarkdownStripsSuffixSpeechScaffoldWithStructuredDialogue(t *testing.T) {
	rendered := RenderNarrativeMarkdown(NarrativeInput{
		Narrative: "ZUUUUM! Sfrecciate nel vicolo neon. 'Ammazza quel suola-nera schifosa, consorte mio!' urla lei, sparando laser-arcobaleno dalle unghie mentre il drone precipita.",
		DialogueBlocks: []engine.DialogueBlock{
			{Speaker: "Dee Podale Suprema", Role: "npc", Text: "Ammazza quel suola-nera schifosa, consorte mio!"},
		},
	})

	if strings.Contains(rendered, "urla lei") {
		t.Fatalf("expected speech scaffold to be stripped, got %q", rendered)
	}
	if strings.Contains(rendered, "'Ammazza quel suola-nera schifosa, consorte mio!'") {
		t.Fatalf("expected quoted prose duplicate to be removed, got %q", rendered)
	}
	if !strings.Contains(rendered, "sparando laser-arcobaleno dalle unghie") {
		t.Fatalf("expected surrounding action prose to remain, got %q", rendered)
	}
	if !strings.Contains(rendered, "> **Dee Podale Suprema:** _\"Ammazza quel suola-nera schifosa, consorte mio!\"_") {
		t.Fatalf("expected structured dialogue block to remain, got %q", rendered)
	}
}

func TestRenderNarrativeMarkdownPromotesQuoteBeforeVerbWithoutBlocks(t *testing.T) {
	rendered := RenderNarrativeMarkdown(NarrativeInput{
		Narrative: "Il blob arretra. 'Ammazza quel suola-nera schifosa, consorte mio!' urla lei, sparando laser-arcobaleno dalle unghie.",
	})

	if strings.Contains(rendered, "'Ammazza quel suola-nera schifosa, consorte mio!'") {
		t.Fatalf("expected quoted prose to be promoted out of plain narrative, got %q", rendered)
	}
	if strings.Contains(rendered, "urla lei") {
		t.Fatalf("expected verb scaffold to be removed with the extracted dialogue, got %q", rendered)
	}
	if !strings.Contains(rendered, "> **Lei:** _\"Ammazza quel suola-nera schifosa, consorte mio!\"_") {
		t.Fatalf("expected fallback dialogue rendering for quote-before-verb pattern, got %q", rendered)
	}
	if !strings.Contains(rendered, "sparando laser-arcobaleno dalle unghie.") {
		t.Fatalf("expected trailing prose to remain, got %q", rendered)
	}
}

func TestRenderNarrativeMarkdownStripsPrefixSpeechScaffoldWithStructuredDialogue(t *testing.T) {
	rendered := RenderNarrativeMarkdown(NarrativeInput{
		Narrative: "La Dee Podale Suprema ti bacia l'arco plantare con unghie arcobaleno, ridendo isterica: 'Mio sposo guerriero, hai scatenato l'Apocalisse Podale!' Le Fosse tremano.",
		DialogueBlocks: []engine.DialogueBlock{
			{Speaker: "Dee Podale Suprema", Role: "npc", Text: "Mio sposo guerriero, hai scatenato l'Apocalisse Podale!"},
		},
	})

	if strings.Contains(rendered, "ridendo isterica:") {
		t.Fatalf("expected prefix speech scaffold to be stripped, got %q", rendered)
	}
	if !strings.Contains(rendered, "Le Fosse tremano.") {
		t.Fatalf("expected remaining narration to stay visible, got %q", rendered)
	}
	if !strings.Contains(rendered, "> **Dee Podale Suprema:** _\"Mio sposo guerriero, hai scatenato l'Apocalisse Podale!\"_") {
		t.Fatalf("expected structured dialogue to render cleanly, got %q", rendered)
	}
}
