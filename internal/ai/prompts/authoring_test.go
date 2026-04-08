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
