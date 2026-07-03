package rendering

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/crimsab/oneday/internal/engine"
)

type dialogueRecognizer struct {
	re           *regexp.Regexp
	textGroup    int
	speakerGroup int
}

type dialogueMatch struct {
	Start   int
	End     int
	Speaker string
	Text    string
}

var dialogueSpeakerPattern = `(?:[A-Z][A-Za-z0-9_'\-]*(?:\s+[A-Z][A-Za-z0-9_'\-]*){0,4}|[Ll]ei|[Ll]ui|[Vv]oi|[Nn]oi|[Yy]ou|[Hh]e|[Ss]he|[Tt]hey)`

var dialogueVerbPattern = strings.Join([]string{
	`ask`, `asks`, `asked`,
	`balbetta`, `balbetti`, `balbett[aò]`,
	`bisbiglia`, `bisbigli[aò]`,
	`call`, `calls`, `called`,
	`dice`, `dicono`, `disse`,
	`esclama`, `esclam[aò]`,
	`grida`, `grid[aò]`,
	`growl`, `growls`, `growled`,
	`hiss`, `hisses`, `hissed`,
	`mormora`, `mormor[aò]`,
	`murmur`, `murmurs`, `murmured`,
	`reply`, `replies`, `replied`,
	`ride`, `ridendo`, `rise`,
	`risponde`, `rispos[eo]`,
	`said`, `say`, `says`,
	`shout`, `shouts`, `shouted`,
	`snap`, `snaps`, `snapped`,
	`sussurra`, `sussurr[aò]`,
	`tuba`,
	`urla`, `url[aò]`,
	`whisper`, `whispers`, `whispered`,
	`yell`, `yells`, `yelled`,
}, "|")

var dialogueRecognizers = []dialogueRecognizer{
	{
		re:           regexp.MustCompile(`(?is)\b(` + dialogueSpeakerPattern + `)\s*:\s*["“'‘]([^"\n“”'‘’]{2,260})["”'’]`),
		textGroup:    2,
		speakerGroup: 1,
	},
	{
		re:           regexp.MustCompile(`(?is)\b(` + dialogueSpeakerPattern + `)\b[^.!?\n]{0,80}?\b(?:` + dialogueVerbPattern + `)\b[^.!?\n]{0,80}?["“'‘]([^"\n“”'‘’]{2,260})["”'’]`),
		textGroup:    2,
		speakerGroup: 1,
	},
	{
		re:           regexp.MustCompile(`(?is)["“'‘]([^"\n“”'‘’]{2,260})["”'’]\s*(?:,?\s*)?(?:` + dialogueVerbPattern + `)\b[^.!?\n]{0,80}?\b(` + dialogueSpeakerPattern + `)\b`),
		textGroup:    1,
		speakerGroup: 2,
	},
	{
		re:           regexp.MustCompile(`(?is)["“'‘]([^"\n“”'‘’]{2,260})["”'’]\s*(?:,?\s*)?(?:` + dialogueVerbPattern + `)\b[^.!?\n]{0,80}`),
		textGroup:    1,
		speakerGroup: 0,
	},
}

var strippedDialoguePrefixScaffoldRE = regexp.MustCompile(`(?is)(^|[,;]\s+|\n+)\b[^.!?\n]{0,50}?\b(?:` + dialogueVerbPattern + `)\b[^.!?\n]{0,40}?:\s*(?:''|""|“”|‘’)?`)
var strippedDialogueSuffixScaffoldRE = regexp.MustCompile(`(?is)(^|[.!?]\s+|\n+|\s{2,})\b(?:` + dialogueVerbPattern + `)\b[^.!?\n]{0,40}?\b(?:` + dialogueSpeakerPattern + `)\b\s*,?\s*`)

// RenderNarrativeMarkdown converts structured narrative input into markdown that
// can be passed through the existing markdown renderer. It prefers trusted
// structured metadata but always falls back to safe plain narrative text.
func RenderNarrativeMarkdown(input NarrativeInput) string {
	parts := make([]string, 0, 4)
	entities := collectHighlightEntities(input)
	dialogueBlocks := input.DialogueBlocks

	if art := renderASCIIArt(input.ASCIIArt); art != "" {
		parts = append(parts, art)
	}

	if callouts := renderEventCallouts(input.EventCallouts); callouts != "" {
		parts = append(parts, callouts)
	}

	narrative := strings.TrimSpace(input.Narrative)
	if len(dialogueBlocks) > 0 {
		narrative = stripStructuredDialogueFromNarrative(narrative, dialogueBlocks)
	} else {
		var extracted []engine.DialogueBlock
		narrative, extracted = extractDialogueBlocksFromNarrative(narrative)
		if len(extracted) > 0 {
			dialogueBlocks = extracted
		}
	}
	if narrative != "" {
		parts = append(parts, highlightEntities(narrative, entities))
	}

	if dialogue := renderDialogueBlocks(dialogueBlocks, entities); dialogue != "" {
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
		if title == "" {
			title = detail
			detail = ""
		}
		block := fmt.Sprintf("> **[%s] %s**", strings.ToUpper(label), title)
		if detail != "" {
			block += "\n> " + detail
		}
		blocks = append(blocks, block)
	}
	return strings.Join(blocks, "\n\n")
}

func renderDialogueBlocks(blocks []engine.DialogueBlock, entities []KnownEntity) string {
	if len(blocks) == 0 {
		return ""
	}

	var rendered []string
	for _, block := range blocks {
		text := highlightEntities(strings.TrimSpace(block.Text), entities)
		if text == "" {
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
			rendered = append(rendered, fmt.Sprintf("> **%s:** _%s_", speaker, quoted))
		case "player":
			label := "You"
			if speaker != "" {
				label = speaker
			}
			rendered = append(rendered, fmt.Sprintf("> **%s:** _%s_", label, quoted))
		case "meta", "system":
			label := "[Narrator Control]"
			if speaker != "" {
				label = "[" + speaker + "]"
			}
			rendered = append(rendered, fmt.Sprintf("> **%s** _%s_", label, quoted))
		case "narrator":
			rendered = append(rendered, text)
		default:
			if speaker != "" {
				rendered = append(rendered, fmt.Sprintf("> **%s:** _%s_", speaker, quoted))
			} else {
				rendered = append(rendered, fmt.Sprintf("> _%s_", quoted))
			}
		}
	}

	return strings.Join(rendered, "\n\n")
}

func stripStructuredDialogueFromNarrative(narrative string, blocks []engine.DialogueBlock) string {
	cleaned := narrative
	for _, block := range blocks {
		text := dialogueBodyText(block.Text)
		if text == "" {
			continue
		}
		for _, variant := range dialogueVariants(text) {
			cleaned = strings.ReplaceAll(cleaned, variant, "")
		}
	}
	cleaned = cleanupStrippedDialogueScaffolds(cleaned)
	return cleanupNarrativeSpacing(cleaned)
}

func extractDialogueBlocksFromNarrative(narrative string) (string, []engine.DialogueBlock) {
	matches := collectDialogueMatches(narrative)
	seen := map[string]bool{}
	blocks := make([]engine.DialogueBlock, 0, len(matches))
	for _, match := range matches {
		text := strings.TrimSpace(match.Text)
		if text == "" {
			continue
		}
		speaker := normalizeDialogueSpeaker(match.Speaker)
		key := strings.ToLower(speaker + "|" + normalizeDialogueKey(text))
		if seen[key] {
			continue
		}
		seen[key] = true
		role := "npc"
		if strings.EqualFold(speaker, "you") {
			role = "player"
		}
		blocks = append(blocks, engine.DialogueBlock{
			Speaker: speaker,
			Role:    role,
			Text:    text,
		})
	}
	return cleanupNarrativeSpacing(removeDialogueMatches(narrative, matches)), blocks
}

func dialogueVariants(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	variants := []string{
		text,
		`"` + text + `"`,
		"'" + text + "'",
		"“" + text + "”",
		"‘" + text + "’",
	}
	if strings.HasSuffix(text, ".") || strings.HasSuffix(text, "!") || strings.HasSuffix(text, "?") {
		trimmed := strings.TrimRight(text, ".!?")
		if trimmed != text {
			variants = append(variants,
				`"`+trimmed+`"`,
				"'"+trimmed+"'",
				"“"+trimmed+"”",
				"‘"+trimmed+"’",
			)
		}
	}
	return variants
}

func dialogueBodyText(text string) string {
	text = strings.TrimSpace(text)
	text = strings.Trim(text, `"'“”‘’`)
	return strings.TrimSpace(text)
}

func normalizeDialogueKey(text string) string {
	text = dialogueBodyText(text)
	text = strings.TrimSpace(text)
	text = strings.ToLower(text)
	text = strings.Join(strings.Fields(text), " ")
	return text
}

func cleanupNarrativeSpacing(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")
	cleanedLines := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.Join(strings.Fields(line), " ")
		line = strings.TrimLeft(line, ",;:")
		line = strings.ReplaceAll(line, " ,", ",")
		line = strings.ReplaceAll(line, " .", ".")
		line = strings.ReplaceAll(line, " !", "!")
		line = strings.ReplaceAll(line, " ?", "?")
		line = strings.ReplaceAll(line, " ;", ";")
		line = strings.ReplaceAll(line, " :", ":")
		line = strings.TrimSpace(line)
		if line != "" {
			cleanedLines = append(cleanedLines, line)
		}
	}
	return strings.TrimSpace(strings.Join(cleanedLines, "\n\n"))
}

func collectDialogueMatches(narrative string) []dialogueMatch {
	if strings.TrimSpace(narrative) == "" {
		return nil
	}

	matches := make([]dialogueMatch, 0, 8)
	for _, recognizer := range dialogueRecognizers {
		indexes := recognizer.re.FindAllStringSubmatchIndex(narrative, -1)
		for _, idx := range indexes {
			if len(idx) < 2 {
				continue
			}
			textStart := subgroupIndex(idx, recognizer.textGroup, 0)
			textEnd := subgroupIndex(idx, recognizer.textGroup, 1)
			if textStart < 0 || textEnd <= textStart {
				continue
			}
			speaker := ""
			if recognizer.speakerGroup > 0 {
				speakerStart := subgroupIndex(idx, recognizer.speakerGroup, 0)
				speakerEnd := subgroupIndex(idx, recognizer.speakerGroup, 1)
				if speakerStart >= 0 && speakerEnd > speakerStart {
					speaker = narrative[speakerStart:speakerEnd]
				}
			}
			matches = append(matches, dialogueMatch{
				Start:   idx[0],
				End:     idx[1],
				Speaker: strings.TrimSpace(speaker),
				Text:    strings.TrimSpace(narrative[textStart:textEnd]),
			})
		}
	}

	if len(matches) == 0 {
		return nil
	}

	sort.SliceStable(matches, func(i, j int) bool {
		return matches[i].Start < matches[j].Start
	})

	filtered := make([]dialogueMatch, 0, len(matches))
	lastEnd := -1
	for _, match := range matches {
		if match.Start < lastEnd {
			continue
		}
		if strings.TrimSpace(match.Text) == "" {
			continue
		}
		filtered = append(filtered, match)
		lastEnd = match.End
	}
	return filtered
}

func subgroupIndex(indexes []int, group, edge int) int {
	slot := group * 2
	if slot+edge >= len(indexes) {
		return -1
	}
	return indexes[slot+edge]
}

func removeDialogueMatches(narrative string, matches []dialogueMatch) string {
	if len(matches) == 0 {
		return narrative
	}
	var builder strings.Builder
	last := 0
	for _, match := range matches {
		if match.Start < last {
			continue
		}
		if match.Start > len(narrative) {
			break
		}
		builder.WriteString(narrative[last:match.Start])
		last = match.End
	}
	if last < len(narrative) {
		builder.WriteString(narrative[last:])
	}
	return builder.String()
}

func normalizeDialogueSpeaker(speaker string) string {
	speaker = strings.TrimSpace(speaker)
	if speaker == "" {
		return ""
	}
	switch strings.ToLower(speaker) {
	case "lei":
		return "Lei"
	case "lui":
		return "Lui"
	case "voi":
		return "Voi"
	case "noi":
		return "Noi"
	case "he":
		return "He"
	case "she":
		return "She"
	case "they":
		return "They"
	case "you":
		return "You"
	default:
		return speaker
	}
}

func cleanupStrippedDialogueScaffolds(text string) string {
	text = strings.ReplaceAll(text, "''", "")
	text = strings.ReplaceAll(text, `""`, "")
	text = strings.ReplaceAll(text, "“”", "")
	text = strings.ReplaceAll(text, "‘’", "")
	text = strippedDialoguePrefixScaffoldRE.ReplaceAllString(text, "$1")
	text = strippedDialogueSuffixScaffoldRE.ReplaceAllString(text, "$1")
	return text
}

func collectHighlightEntities(input NarrativeInput) []KnownEntity {
	entities := make([]KnownEntity, 0, len(input.KnownEntities)+len(input.EntitiesMentioned))
	byName := map[string]KnownEntity{}

	add := func(name, kind string) {
		name = strings.TrimSpace(name)
		if len([]rune(name)) < 3 {
			return
		}
		key := strings.ToLower(name)
		existing, ok := byName[key]
		if ok && existing.Kind != "" {
			return
		}
		byName[key] = KnownEntity{Name: name, Kind: strings.ToLower(strings.TrimSpace(kind))}
	}

	for _, entity := range input.KnownEntities {
		add(entity.Name, entity.Kind)
	}
	for _, entity := range input.EntitiesMentioned {
		add(entity.Name, entity.Type)
	}

	for _, entity := range byName {
		entities = append(entities, entity)
	}
	sort.Slice(entities, func(i, j int) bool {
		return len([]rune(entities[i].Name)) > len([]rune(entities[j].Name))
	})
	return entities
}

func highlightEntities(text string, entities []KnownEntity) string {
	if text == "" || len(entities) == 0 {
		return text
	}

	highlighted := text
	for _, entity := range entities {
		pattern := entityPattern(entity.Name)
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		highlighted = re.ReplaceAllStringFunc(highlighted, func(match string) string {
			if strings.Contains(match, "`") || strings.Contains(match, "**") || strings.Contains(match, "_") {
				return match
			}
			return renderEntityMarkdown(match, entity.Kind)
		})
	}
	return highlighted
}

func renderEntityMarkdown(match, kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "location", "world":
		return fmt.Sprintf("**`%s`**", match)
	case "faction":
		return fmt.Sprintf("_**%s**_", match)
	case "item", "skill", "title":
		return fmt.Sprintf("`%s`", match)
	case "chapter":
		return fmt.Sprintf("_%s_", match)
	default:
		return fmt.Sprintf("**%s**", match)
	}
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
