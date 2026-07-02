package components

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/tui/theme"
)

var renderer *glamour.TermRenderer
var (
	calloutTagPattern   = regexp.MustCompile(`\[(NPC|RELATIONSHIP|ALLIANCE|ITEM|LOCATION|SKILL|FACTION|WORLD|CHAPTER)\]`)
	dialogueLinePattern = regexp.MustCompile(`^(\s*[│>\s]*)([^:\n]{2,48}:)\s+(.+)$`)
)

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
	return StylizeNarrativeOutput(out)
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

// StylizeNarrativeOutput adds a small amount of deterministic ANSI styling on top
// of glamour output for structured callouts and dialogue labels.
func StylizeNarrativeOutput(text string) string {
	if strings.TrimSpace(text) == "" {
		return text
	}

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		line = stylizeCalloutTags(line)
		line = stylizeDialogueLine(line)
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

func stylizeCalloutTags(line string) string {
	return calloutTagPattern.ReplaceAllStringFunc(line, func(match string) string {
		label := strings.Trim(match, "[]")
		style := lipgloss.NewStyle().Bold(true).Foreground(theme.Highlight)
		switch label {
		case "NPC":
			style = lipgloss.NewStyle().Bold(true).Foreground(theme.Primary)
		case "RELATIONSHIP", "ALLIANCE":
			style = lipgloss.NewStyle().Bold(true).Foreground(theme.Success)
		case "ITEM":
			style = lipgloss.NewStyle().Bold(true).Foreground(theme.CraftingBlue)
		case "LOCATION", "WORLD", "CHAPTER":
			style = lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
		case "FACTION":
			style = lipgloss.NewStyle().Bold(true).Foreground(theme.Secondary)
		case "SKILL":
			style = lipgloss.NewStyle().Bold(true).Foreground(theme.Highlight)
		}
		return style.Render(match)
	})
}

func stylizeDialogueLine(line string) string {
	matches := dialogueLinePattern.FindStringSubmatch(line)
	if len(matches) != 4 {
		return line
	}
	prefix, speaker, spoken := matches[1], matches[2], strings.TrimSpace(matches[3])
	if strings.HasPrefix(spoken, "[") {
		return line
	}
	speakerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Primary)
	textStyle := lipgloss.NewStyle().Foreground(theme.Text).Italic(true)
	return prefix + speakerStyle.Render(speaker) + " " + textStyle.Render(spoken)
}
