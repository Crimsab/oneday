package rendering

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/crimsab/oneday/internal/engine"
)

// RenderNarrativeMarkdown converts structured narrative input into markdown that
// can be passed through the existing markdown renderer. It prefers trusted
// structured metadata but always falls back to safe plain narrative text.
func RenderNarrativeMarkdown(input NarrativeInput) string {
	parts := make([]string, 0, 4)

	if art := renderASCIIArt(input.ASCIIArt); art != "" {
		parts = append(parts, art)
	}

	if callouts := renderEventCallouts(input.EventCallouts); callouts != "" {
		parts = append(parts, callouts)
	}

	narrative := strings.TrimSpace(input.Narrative)
	if narrative != "" {
		parts = append(parts, highlightEntities(narrative, collectHighlightCandidates(input)))
	}

	if dialogue := renderDialogueBlocks(input.DialogueBlocks, collectHighlightCandidates(input), narrative); dialogue != "" {
		parts = append(parts, dialogue)
	}

	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func renderEventCallouts(callouts []engine.EventCallout) string {
	if len(callouts) == 0 {
		return ""
	}

	var blocks []string
	for _, callout := range callouts {
		title := strings.TrimSpace(callout.Title)
		detail := strings.TrimSpace(callout.Detail)
		if title == "" && detail == "" {
			continue
		}
		label := strings.TrimSpace(callout.Kind)
		if label == "" {
			label = "event"
		}
		block := fmt.Sprintf("**[%s] %s**", strings.ToUpper(label), title)
		if detail != "" {
			block += "\n" + detail
		}
		blocks = append(blocks, block)
	}
	return strings.Join(blocks, "\n\n")
}

func renderDialogueBlocks(blocks []engine.DialogueBlock, candidates []string, narrative string) string {
	if len(blocks) == 0 {
		return ""
	}

	var rendered []string
	narrativeLower := strings.ToLower(narrative)
	for _, block := range blocks {
		text := highlightEntities(strings.TrimSpace(block.Text), candidates)
		if text == "" {
			continue
		}
		if narrativeLower != "" && strings.Contains(narrativeLower, strings.ToLower(strings.TrimSpace(block.Text))) {
			continue
		}
		quoted := quoteDialogueText(text)

		role := strings.ToLower(strings.TrimSpace(block.Role))
		speaker := strings.TrimSpace(block.Speaker)
		switch role {
		case "npc":
			if speaker == "" {
				speaker = "Someone"
			}
			rendered = append(rendered, fmt.Sprintf("**%s:** _%s_", speaker, quoted))
		case "player":
			label := "You"
			if speaker != "" {
				label = speaker
			}
			rendered = append(rendered, fmt.Sprintf("**%s:** _%s_", label, quoted))
		case "meta", "system":
			label := "[Game Master]"
			if speaker != "" {
				label = "[" + speaker + "]"
			}
			rendered = append(rendered, fmt.Sprintf("**%s** _%s_", label, quoted))
		case "narrator":
			rendered = append(rendered, text)
		default:
			if speaker != "" {
				rendered = append(rendered, fmt.Sprintf("**%s:** _%s_", speaker, quoted))
			} else {
				rendered = append(rendered, text)
			}
		}
	}

	return strings.Join(rendered, "\n\n")
}

func collectHighlightCandidates(input NarrativeInput) []string {
	seen := map[string]bool{}
	candidates := make([]string, 0, len(input.KnownEntities)+len(input.EntitiesMentioned))

	add := func(name string) {
		name = strings.TrimSpace(name)
		if len([]rune(name)) < 3 {
			return
		}
		key := strings.ToLower(name)
		if seen[key] {
			return
		}
		seen[key] = true
		candidates = append(candidates, name)
	}

	for _, entity := range input.KnownEntities {
		add(entity.Name)
	}
	for _, entity := range input.EntitiesMentioned {
		add(entity.Name)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return len([]rune(candidates[i])) > len([]rune(candidates[j]))
	})
	return candidates
}

func highlightEntities(text string, candidates []string) string {
	if text == "" || len(candidates) == 0 {
		return text
	}

	highlighted := text
	for _, candidate := range candidates {
		pattern := entityPattern(candidate)
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		highlighted = re.ReplaceAllStringFunc(highlighted, func(match string) string {
			if strings.HasPrefix(match, "**") && strings.HasSuffix(match, "**") {
				return match
			}
			return fmt.Sprintf("**%s**", match)
		})
	}
	return highlighted
}

func entityPattern(name string) string {
	escaped := regexp.QuoteMeta(name)
	first := []rune(name)[0]
	last := []rune(name)[len([]rune(name))-1]

	pattern := escaped
	if isWordRune(first) {
		pattern = `\b` + pattern
	}
	if isWordRune(last) {
		pattern = pattern + `\b`
	}
	return `(?i)` + pattern
}

func isWordRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

func renderASCIIArt(text string) string {
	text = strings.Trim(text, "\n")
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")
	if len(lines) > 12 {
		return ""
	}

	maxWidth := 0
	for _, line := range lines {
		if width := len([]rune(line)); width > maxWidth {
			maxWidth = width
		}
	}
	if maxWidth > 72 {
		return ""
	}

	return "```text\n" + text + "\n```"
}

func quoteDialogueText(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return trimmed
	}
	if strings.HasPrefix(trimmed, "\"") || strings.HasPrefix(trimmed, "“") {
		return trimmed
	}
	return "\"" + trimmed + "\""
}
