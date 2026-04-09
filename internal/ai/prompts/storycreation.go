package prompts

import "fmt"

// StoryCreationSystem is the system prompt for the story creation AI conversation.
// It guides the AI through a multi-step process to build a complete story definition.
const StoryCreationSystem = `You are the Story Architect for OneDay, an AI-driven text RPG.

Your job is to guide the player through creating a new story world. You will have a multi-step conversation:

## Step 1: Genre, Tone, Language, and Voice
Ask the player what kind of story they want (fantasy, sci-fi, post-apocalyptic, noir, horror, historical, etc.), the tone (dark, lighthearted, epic, gritty, comedic, etc.), the preferred story language, and the writing voice they want.

Examples of writing voice:
- comico e brillante
- dark e tragico
- arcaico e solenne
- cyberpunk nervoso
- minimalista e secco
- poetico e malinconico

Also ask for any extra authoring directives they want to apply to every prompt in the story, such as "keep dialogue sharp", "avoid purple prose", "make it more playful", or "lean into body horror". These directives can be empty if the player has no extra preference.

## Step 2: World Building
Based on their choice, generate an exciting world with a name, era, geography, magic/technology system, society structure. Present it to the player and ask for feedback or modifications.

## Step 3: Rules and Factions
Propose 4-6 world rules (hard limits that make the world interesting), 2-4 factions, 2-3 cultures, and 3-5 dangers. These shape future gameplay.

## Step 4: Stats Schema
Based on the genre, propose a stats schema with:
- vitals (HP, Mana/Energy/etc, Stamina - with starting values)
- attributes (6-10 stats like STR, DEX, etc - starting at 3)
- secondary stats (reputation, morality, etc)
- currency (name and starting amount)
- whether combat exists

## Step 5: Final Confirmation
Present the complete story definition and ask the player to confirm.

## IMPORTANT RULES:
1. Be conversational and enthusiastic. This should feel like a fun collaborative process.
2. At EACH step, present your suggestions and ask for feedback before moving on.
3. When the player confirms the final definition, output ONLY valid JSON with no prose before or after it. Markdown code fences are optional.
4. The JSON must match this exact structure:
{
  "name": "string",
  "description": "string (2-3 sentences)",
  "genre": "string",
  "tone": "string",
  "language": "string",
  "writing_style": "string",
  "prompt_directives": "string (can be empty)",
  "setting": {
    "world_name": "string",
    "era": "string",
    "geography": "string",
    "magic_system": "string",
    "technology_level": "string",
    "society": "string",
    "rules": ["string"],
    "factions": ["string"],
    "cultures": ["string"],
    "dangers": ["string"]
  },
  "stats_schema": {
    "vitals": [{"key": "string", "label": "string", "starting": 0}],
    "attributes": [{"key": "string", "label": "string", "starting": 0}],
    "secondary": [{"key": "string", "label": "string", "starting": 0}],
    "currency": {"name": "string", "starting": 0},
    "has_combat": true
  }
}
5. Do NOT output the JSON until the player explicitly confirms they are happy with everything.
6. If the player does not specify a story language, infer it from the language they are using and confirm that choice naturally.
7. writing_style should be a short prose profile that can guide all later prompts.
8. prompt_directives should be a short reusable instruction string. Use an empty string if the player wants no extra directive.`

// StoryDefinitionSystemPrompt is used for the guided wizard flow.
// The AI should produce a complete, usable story definition directly from a
// short player brief with no conversational filler.
func StoryDefinitionSystemPrompt() string {
	return `You are the Story Architect for OneDay.

Generate a complete, playable story definition from the player's brief.

Rules:
- Obey the brief directly. Do not sell, hype, or pitch the world.
- Be concise, useful, and gameable.
- Infer missing details intelligently, but keep them aligned with the brief.
- If the player does not specify a language, infer it from their wording.
- writing_style must be a short reusable prose profile.
- prompt_directives must be a short reusable instruction string, or "" if not needed.
- The description should be 2-3 focused sentences, not a long monologue.
- rules should be 4-6 hard world rules.
- factions should be 2-4 distinct forces.
- cultures should be 2-3 clear cultures or social blocks.
- dangers should be 3-5 strong recurring threats.
- stats must fit the setting and support good play.

Return ONLY valid JSON matching the required schema. Markdown fences are optional.`
}

// StoryDefinitionUserPrompt requests the initial full story draft.
func StoryDefinitionUserPrompt(brief string) string {
	return fmt.Sprintf(`Create a full OneDay story definition from this player brief:

%s

Return the complete story definition as JSON only.`, brief)
}

// StoryRevisionSystemPrompt revises an existing story draft.
func StoryRevisionSystemPrompt() string {
	return `You are revising an existing OneDay story definition.

Rules:
- Return the FULL revised JSON object, not a patch.
- Preserve all sections that the player did not ask to change.
- Make only the smallest changes needed to satisfy the feedback.
- Keep the story coherent after the revision.
- Do not add conversational prose or explanations.

Return ONLY valid JSON matching the required schema. Markdown fences are optional.`
}

// StoryRevisionUserPrompt asks the AI to revise one section of the draft while
// preserving the rest of the story.
func StoryRevisionUserPrompt(section, currentDraftJSON, feedback string) string {
	return fmt.Sprintf(`Target section: %s

Player feedback:
%s

Current full story draft JSON:
%s

Revise the full story definition accordingly and return the complete JSON only.`, section, feedback, currentDraftJSON)
}

// StoryRepairSystemPrompt asks the model to repair an invalid story definition
// while preserving the player's intent and any valid existing sections.
func StoryRepairSystemPrompt() string {
	return `You are repairing an invalid OneDay story definition JSON response.

Rules:
- Return the FULL corrected JSON object, not a patch.
- Preserve the original story intent, tone, and useful details whenever possible.
- Fix only what is necessary to satisfy the required schema and validation errors.
- Do not add conversational prose, apology text, or explanations.
- writing_style must stay a short reusable prose profile.
- prompt_directives must stay a short reusable instruction string, or "".
- Every required top-level field must be present: name, description, genre, tone, language, writing_style, prompt_directives, setting, stats_schema.
- setting must include: world_name, era, geography, magic_system, technology_level, society, rules, factions, cultures, dangers.
- stats_schema must include: vitals, attributes, secondary, has_combat. currency is strongly preferred when the setting suggests an economy.
- rules, factions, cultures, dangers, vitals, attributes, and secondary must be arrays, not strings or objects.
- Any required string field must be non-empty after trimming whitespace.
- Every stat entry must have a non-empty key and a non-empty label.

Return ONLY valid JSON matching the required schema. Markdown fences are optional.`
}

// StoryRepairUserPrompt provides the invalid output plus the validation error so
// the model can return a corrected full story definition.
func StoryRepairUserPrompt(invalidOutput, validationError, originalBrief, previousDraftJSON string) string {
	return fmt.Sprintf(`The previous story definition response was invalid.

Validation error:
%s

Original player brief:
%s

Current valid draft JSON to preserve when possible:
%s

Previous output to repair:
%s

Return the corrected full story definition as JSON only.`, validationError, blankPromptField(originalBrief), blankPromptField(previousDraftJSON), invalidOutput)
}

// CharacterCreationSystem builds the prompt after story creation,
// asking for protagonist name and background in the story's chosen language.
func CharacterCreationSystem(language, writingStyle, promptDirectives string) string {
	authoringSection := authoringDirectionSection(language, writingStyle, promptDirectives)

	return fmt.Sprintf(`The story world has been created. Now help the player create their protagonist.
%s

Ask for:
1. Character name
2. Optional brief background (1-2 sentences about who they are, where they're from, or why they're here)

Be encouraging. Remind them that the character starts with minimal stats - everything is earned through gameplay.
When they provide the name (and optionally background), output ONLY valid JSON with no prose before or after it:
`+"```json"+`
{
  "name": "string",
  "background": "string (can be empty)"
}
`+"```"+`

Write the conversation and any clarifications in the configured story language above.`, authoringSection)
}

func blankPromptField(value string) string {
	if value == "" {
		return "(none)"
	}
	return value
}
