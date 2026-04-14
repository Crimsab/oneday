package prompts

import (
	"strings"
	"testing"
)

func TestNarratorSystemIncludesAuthoringDirection(t *testing.T) {
	prompt := NarratorSystem(
		"Test Story",
		"noir",
		"dark",
		"italiano",
		"prosa asciutta e tesa",
		"Dialoghi taglienti e poco espositivi.",
		`{"world_name":"Vespera"}`,
		`{"attributes":[]}`,
		"Nerea",
		"Corriere dei canali",
		`{"attributes":{"agi":3}}`,
		"",
	)

	for _, want := range []string{
		"Story language: italiano",
		"Writing style: prosa asciutta e tesa",
		"Extra directives: Dialoghi taglienti e poco espositivi.",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q", want)
		}
	}
}

func TestNarratorSystemOmitsAuthoringSectionWhenUnset(t *testing.T) {
	prompt := NarratorSystem(
		"Test Story",
		"fantasy",
		"bright",
		"",
		"",
		"",
		`{"world_name":"Vespera"}`,
		`{"attributes":[]}`,
		"Nerea",
		"Corriere dei canali",
		`{"attributes":{"agi":3}}`,
		"",
	)

	if strings.Contains(prompt, "Story Language And Authoring Direction") {
		t.Fatal("prompt unexpectedly included authoring section")
	}
}

func TestNarratorSystemIncludesCombatAndGuidanceInstructions(t *testing.T) {
	prompt := NarratorSystem(
		"Test Story",
		"fantasy",
		"grim",
		"italiano",
		"",
		"",
		`{"world_name":"Vespera"}`,
		`{"has_combat":true}`,
		"Nerea",
		"Corriere dei canali",
		`{"attributes":{"agi":3}}`,
		"",
	)

	for _, want := range []string{
		`"combat_start": null`,
		"## Combat Encounters",
		"## Player Guidance",
		"## Special Pacing Commands",
		"[Advance Scene] ...",
		"[Time Skip] ...",
		"free text after [Advance Scene]",
		"free text after [Time Skip]",
		`"guide_update": {"title": "Guidance title", "status": "seeded"`,
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q", want)
		}
	}
}

func TestGuideMetaSystemIncludesNonAdvancingRules(t *testing.T) {
	prompt := GuideMetaSystem(
		"Test Story",
		"italiano",
		"",
		"",
		`{"world_name":"Vespera"}`,
		`{"location":"Canali","chapter":2,"turn":14}`,
		"- Lyanna {npc_scene}",
		"- Boss finale [chapter/high/active] — Ancora da introdurre",
	)

	for _, want := range []string{
		"The player is not taking an in-story action.",
		`"guidance": [`,
		`"status": "active"`,
		"Do not advance the story",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q", want)
		}
	}
}
