package components

import "testing"

func TestNormalizeMarkdownStripsATXHeadingMarkers(t *testing.T) {
	got := normalizeMarkdown("### Danger Ahead\n\nText")
	want := "**Danger Ahead**\n\nText"
	if got != want {
		t.Fatalf("normalizeMarkdown() = %q, want %q", got, want)
	}
}
