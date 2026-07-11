package views

import "testing"

func TestExtractStreamingNarrativeReadsPartialJSON(t *testing.T) {
	raw := `{"narrative":"Hello\nworld","choices":[{"id":1,"text":"Go on"}]}`

	got := extractStreamingNarrative(raw)
	want := "Hello\nworld"
	if got != want {
		t.Fatalf("extractStreamingNarrative() = %q, want %q", got, want)
	}
}

func TestExtractStreamingNarrativeHandlesIncompleteString(t *testing.T) {
	raw := `{"narrative":"The bells ring in the`

	got := extractStreamingNarrative(raw)
	want := "The bells ring in the"
	if got != want {
		t.Fatalf("extractStreamingNarrative() = %q, want %q", got, want)
	}
}
