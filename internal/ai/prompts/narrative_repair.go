package prompts

import "fmt"

// NarrativeRepairSystemPrompt repairs an invalid gameplay response while
// preserving the scene intent and returning only the structured JSON payload.
func NarrativeRepairSystemPrompt() string {
	return `You are repairing an invalid OneDay gameplay response JSON.

Rules:
- Return the FULL corrected JSON object, not a patch.
- Preserve the scene intent, tone, and choices whenever possible.
- Keep the response concise and gameable.
- The JSON must include:
  - narrative: non-empty string
  - choices: array of player choices (can be empty only if the scene truly has none)
- Keep optional metadata only when valid.
- If "social_duel" is present, keep it scene-framing only; never invent a duel winner or engine-owned results there.
- Do not add explanations, apologies, or extra prose outside the JSON.

Return ONLY valid JSON matching the OneDay gameplay schema. Markdown fences are optional.`
}

// NarrativeRepairUserPrompt provides the invalid output and validation error.
func NarrativeRepairUserPrompt(invalidOutput, validationError string) string {
	return fmt.Sprintf(`The previous gameplay response was invalid.

Validation error:
%s

Previous output to repair:
%s

Return the corrected gameplay response as JSON only.`, validationError, invalidOutput)
}
