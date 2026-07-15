package views

import (
	"strings"
	"testing"

	"github.com/crimsab/oneday/internal/engine"
	appi18n "github.com/crimsab/oneday/internal/i18n"
)

func TestRenderNarrativeResponseFallsBackForPlainResumePayload(t *testing.T) {
	model := &NarrativeModel{}

	rendered := model.renderNarrativeResponse(&engine.NarrativeResponse{
		Narrative: "Welcome back to the ruins.",
	})

	if strings.TrimSpace(rendered) == "" {
		t.Fatal("expected non-empty rendered output")
	}
	if !strings.Contains(rendered, "Welcome back to the ruins.") {
		t.Fatalf("expected plain narrative text in output, got %q", rendered)
	}
}

func TestCollectKnownEntitiesUsesResponseLocationWithoutNarratorState(t *testing.T) {
	model := &NarrativeModel{}

	entities := model.collectKnownEntities(&engine.NarrativeResponse{
		Location: "Old Harbor",
	})

	if len(entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(entities))
	}
	if entities[0].Name != "Old Harbor" || entities[0].Kind != "location" {
		t.Fatalf("unexpected entity: %+v", entities[0])
	}
}

func TestRenderTurnDeltaMarkdownAddsSystemNavigationHints(t *testing.T) {
	delta := &engine.TurnDelta{
		Items: []engine.TurnDeltaItem{
			{Kind: "front", Label: "Front advances: Whispers Around the Bell Tower"},
			{Kind: "project", Label: "Project completed: Train with Lyanna"},
			{Kind: "investigation", Label: "Clue added: Missing seal"},
		},
	}

	rendered := renderTurnDeltaMarkdown(delta, appi18n.New(appi18n.English))
	for _, want := range []string{"`/fronts`", "`/projects`", "`/investigations`", "`/codex`"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("turn delta markdown missing %q:\n%s", want, rendered)
		}
	}
	for _, want := range []string{"Fronts, projects, and investigations changed.", "for details"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("turn delta markdown missing %q:\n%s", want, rendered)
		}
	}
	italian := renderTurnDeltaMarkdown(delta, appi18n.New(appi18n.Italian))
	if !strings.Contains(italian, "Fronti, progetti e indagini sono cambiati.") || !strings.Contains(italian, "per vedere i dettagli") {
		t.Fatalf("Italian turn delta navigation was not localized:\n%s", italian)
	}
	if callout := turnDeltaStatusCallout(delta, appi18n.New(appi18n.Italian)); !strings.Contains(callout, "Premi F, P o I") {
		t.Fatalf("status callout = %q, want combined systems hint", callout)
	}
}

func TestBuildAdvanceSceneActionTreatsHintAsTarget(t *testing.T) {
	got := buildAdvanceSceneAction("una settimana dopo")
	for _, want := range []string{"desired timing", "Requested timing or destination: una settimana dopo"} {
		if !strings.Contains(got, want) {
			t.Fatalf("buildAdvanceSceneAction missing %q in %q", want, got)
		}
	}
}

func TestBuildTimeSkipActionTreatsHintAsArrivalPoint(t *testing.T) {
	got := buildTimeSkipAction("arrivo a 6 anni")
	for _, want := range []string{"preferred arrival point", "Requested destination: arrivo a 6 anni"} {
		if !strings.Contains(got, want) {
			t.Fatalf("buildTimeSkipAction missing %q in %q", want, got)
		}
	}
}
