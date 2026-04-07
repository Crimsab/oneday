package components

import (
	"github.com/charmbracelet/glamour"
)

var renderer *glamour.TermRenderer

func init() {
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		// Fallback: no rendering
		return
	}
	renderer = r
}

// RenderMarkdown converts markdown text to styled terminal output.
// Falls back to raw text if renderer is unavailable.
func RenderMarkdown(md string) string {
	if renderer == nil || md == "" {
		return md
	}
	out, err := renderer.Render(md)
	if err != nil {
		return md
	}
	return out
}
