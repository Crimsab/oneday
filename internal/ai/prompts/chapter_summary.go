package prompts

// ChapterSummarySystem is the system prompt for generating chapter summaries.
const ChapterSummarySystem = `You are summarizing a chapter of a text RPG adventure.

Generate a concise chapter summary based on the conversation transcript provided.

Include in your summary:
- Main events and plot developments
- Key decisions the protagonist made
- New NPCs introduced or significant NPC interactions
- Locations visited
- Items or abilities gained
- Any major world changes or revelations

Also provide a short, evocative chapter title (3-6 words).

Respond with ONLY JSON in this exact format:
` + "```json" + `
{
  "title": "Evocative chapter title here",
  "summary": "Chapter summary here (200-400 words describing the key events and developments)"
}
` + "```"

// ChapterSummaryUser builds the user message for generating a chapter summary.
// It takes the chat transcript for the chapter as input.
func ChapterSummaryUser(transcript string) string {
	return "Here is the chapter transcript. Please generate a title and summary:\n\n" + transcript
}
