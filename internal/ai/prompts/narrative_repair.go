package prompts

import "fmt"

// NarrativeRepairSystemPrompt repairs an invalid gameplay response while
// preserving the scene intent and returning only the structured JSON payload.
func NarrativeRepairSystemPrompt(storyName, genre, tone, language, currentLocation, settingJSON string) string {
	return fmt.Sprintf(`You are repairing an invalid OneDay gameplay response JSON.

Canonical story continuity:
- Story: %s
- Genre: %s
- Tone: %s
- Story language: %s
- Current location: %s

Canonical world anchor:
%s

Rules:
- Return the FULL corrected JSON object, not a patch.
- Preserve the player's scene intent whenever possible, but keep it inside the canonical story continuity above.
- Do NOT switch to an unrelated genre, language, era, technology level, city, or setting.
- If the previous output drifted into another world or tone, discard the drift and rebuild the scene inside the canonical story instead.
- Keep the response concise and gameable.
- The JSON must include:
  - narrative: non-empty string
  - choices: array of player choices (can be empty only if the scene truly has none)
- Keep optional metadata only when valid.
- If "social_duel" is present, keep it scene-framing only; never invent a duel winner or engine-owned results there.
- Do not add explanations, apologies, or extra prose outside the JSON.

Return ONLY valid JSON matching the OneDay gameplay schema. Markdown fences are optional.`, storyName, genre, tone, language, currentLocation, settingJSON)
}

// NarrativeRepairUserPrompt provides the invalid output and validation error.
func NarrativeRepairUserPrompt(currentInput, invalidOutput, validationError string) string {
	return fmt.Sprintf(`The previous gameplay response was invalid.

Current player input:
%s

Validation error:
%s

Previous output to repair:
%s

Return the corrected gameplay response as JSON only.`, currentInput, validationError, invalidOutput)
}
