package engine

import (
	"strings"
	"unicode"

	"github.com/crimsab/oneday/internal/storage"
)

type continuityIssue struct {
	Reasons []string
}

func (i *continuityIssue) Error() string {
	if i == nil || len(i.Reasons) == 0 {
		return ""
	}
	return strings.Join(i.Reasons, "; ")
}

func detectNarrativeContinuityIssue(story *storage.Story, narrative *NarrativeResponse) *continuityIssue {
	if story == nil || narrative == nil {
		return nil
	}

	text := strings.TrimSpace(narrative.Narrative)
	if text == "" {
		return nil
	}

	var reasons []string
	if storyLanguageLooksItalian(story.Language) && looksMostlyEnglish(text) {
		reasons = append(reasons, "response drifted into English instead of the configured story language")
	}
	if !storyAllowsCyberpunkLexicon(story) && containsCyberpunkLexicon(text) {
		reasons = append(reasons, "response introduced unrelated cyberpunk or futuristic setting language")
	}
	if len(reasons) == 0 {
		return nil
	}
	return &continuityIssue{Reasons: reasons}
}

func storyLanguageLooksItalian(language string) bool {
	language = strings.ToLower(strings.TrimSpace(language))
	return language == "it" || strings.Contains(language, "ital")
}

func storyAllowsCyberpunkLexicon(story *storage.Story) bool {
	if story == nil {
		return false
	}
	identity := strings.ToLower(strings.Join([]string{
		story.Genre,
		story.Tone,
		story.Name,
		story.SettingJSON,
	}, " "))

	allowed := []string{
		"cyberpunk",
		"sci-fi",
		"science fiction",
		"science-fiction",
		"futur",
		"space opera",
		"space",
		"mecha",
		"android",
		"post-apocalyptic",
		"post apocalyptic",
	}
	for _, keyword := range allowed {
		if strings.Contains(identity, keyword) {
			return true
		}
	}
	return false
}

func containsCyberpunkLexicon(text string) bool {
	lower := strings.ToLower(text)
	keywords := []string{
		"data-haven",
		"neon",
		"chrome-plated",
		"chrome plated",
		"black-market hub",
		"black market hub",
		"cyber",
		"implant",
		"server",
		"terminal",
		"holo",
		"megacorp",
		"drone",
		"rain-slicked",
		"rain slicked",
	}

	matches := 0
	for _, keyword := range keywords {
		if strings.Contains(lower, keyword) {
			matches++
		}
	}
	return matches >= 2
}

func looksMostlyEnglish(text string) bool {
	tokens := tokenizeContinuityText(text)
	if len(tokens) == 0 {
		return false
	}

	englishCount := 0
	italianCount := 0
	for _, token := range tokens {
		if englishContinuityStopwords[token] {
			englishCount++
		}
		if italianContinuityStopwords[token] {
			italianCount++
		}
	}

	return englishCount >= 4 && englishCount >= italianCount*2
}

func tokenizeContinuityText(text string) []string {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	if len(fields) == 0 {
		return nil
	}

	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if len(field) < 2 {
			continue
		}
		out = append(out, field)
	}
	return out
}

var englishContinuityStopwords = map[string]bool{
	"the": true, "and": true, "you": true, "your": true, "with": true, "from": true,
	"into": true, "against": true, "before": true, "after": true, "through": true,
	"stand": true, "lights": true, "lower": true, "district": true, "guard": true,
}

var italianContinuityStopwords = map[string]bool{
	"il": true, "lo": true, "la": true, "gli": true, "le": true, "del": true,
	"della": true, "delle": true, "che": true, "con": true, "per": true, "mentre": true,
	"sono": true, "nella": true, "sulla": true, "verso": true, "luce": true,
}
