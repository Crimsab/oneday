package prompts

import (
	"fmt"
	"strings"
)

func authoringDirectionSection(language, writingStyle, promptDirectives string) string {
	language = strings.TrimSpace(language)
	writingStyle = strings.TrimSpace(writingStyle)
	promptDirectives = strings.TrimSpace(promptDirectives)

	if language == "" && writingStyle == "" && promptDirectives == "" {
		return ""
	}

	var lines []string
	if language != "" {
		lines = append(lines, fmt.Sprintf("- Story language: %s", language))
	}
	if writingStyle != "" {
		lines = append(lines, fmt.Sprintf("- Writing style: %s", writingStyle))
	}
	if promptDirectives != "" {
		lines = append(lines, fmt.Sprintf("- Extra directives: %s", promptDirectives))
	}

	return "\n## Story Language And Authoring Direction\n" +
		strings.Join(lines, "\n") +
		"\n\nRules:\n" +
		"- Write all visible AI text in the story language above unless the player explicitly asks for a temporary out-of-band translation.\n" +
		"- Keep the prose aligned with the writing style above.\n" +
		"- Apply the extra directives above everywhere they fit naturally without breaking clarity or story coherence.\n"
}
