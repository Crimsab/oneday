package prompts

import "fmt"

// ASCIIArtSystem defines the dedicated prompt for ambient ASCII-art generation.
func ASCIIArtSystem() string {
	return `You generate compact ASCII art for a terminal-based narrative RPG.

Your job is to create a single compact ASCII visual that supports the current scene.

Rules:
- Return ONLY valid JSON
- Put the final art inside the "ascii_art" string
- No prose, no explanations, no markdown fences
- Keep it readable in a narrow terminal
- Maximum 12 lines
- Maximum 72 characters per line
- Use plain ASCII characters only
- Prefer silhouettes, signage, maps, symbols, diagrams, and iconic environmental cues
- Do not include labels unless the cue explicitly calls for signage or a terminal display
- Leave "ascii_art" empty if the cue clearly should not produce usable art`
}

// ASCIIArtUser builds the focused user prompt for a scene-local ASCII request.
func ASCIIArtUser(
	storyName string,
	location string,
	sceneType string,
	mood string,
	narrative string,
	kind string,
	subject string,
	detail string,
	placement string,
) string {
	return fmt.Sprintf(`Create ambient ASCII art for this OneDay scene.

Story: %s
Location: %s
Scene type: %s
Mood: %s
Cue kind: %s
Cue subject: %s
Cue detail: %s
Placement: %s

Narrative context:
%s

Return JSON only:
{"ascii_art":"..."}
`, storyName, location, sceneType, mood, kind, subject, detail, placement, narrative)
}
