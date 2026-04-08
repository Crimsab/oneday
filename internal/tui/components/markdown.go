package components

import (
	"strings"

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
	out, err := renderer.Render(normalizeMarkdown(md))
	if err != nil {
		return md
	}
	return out
}

func normalizeMarkdown(md string) string {
	lines := strings.Split(md, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		level := 0
		for level < len(trimmed) && trimmed[level] == '#' {
			level++
		}
		if level == 0 || level >= len(trimmed) || trimmed[level] != ' ' {
			continue
		}
		title := strings.TrimSpace(trimmed[level:])
		if title == "" {
			continue
		}
		lines[i] = "**" + title + "**"
	}
	return strings.Join(lines, "\n")
}
