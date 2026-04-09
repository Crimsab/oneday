package ai

import (
	"encoding/json"
	"regexp"
	"strings"
)

// jsonBlockRe matches fenced ```json ... ``` blocks in AI output.
var jsonBlockRe = regexp.MustCompile("(?s)```json\\s*\\n(.*?)\\n```")

// AchievementPayload is the structured achievement data the AI returns when awarding an achievement.
type AchievementPayload struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Rarity      string `json:"rarity"`
	Category    string `json:"category"`
	Context     string `json:"context"`
}

// DialogueBlock is optional renderer-facing metadata for structured dialogue.
type DialogueBlock struct {
	Speaker string `json:"speaker"`
	Role    string `json:"role,omitempty"`
	Text    string `json:"text"`
}

// EntityMention is optional renderer-facing metadata for important named entities.
type EntityMention struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

// EventCallout is optional renderer-facing metadata for compact state/event summaries.
type EventCallout struct {
	Kind   string `json:"kind,omitempty"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
}

// ASCIIArtCue requests optional same-turn ambient ASCII art.
type ASCIIArtCue struct {
	Kind      string `json:"kind"`
	Subject   string `json:"subject"`
	Detail    string `json:"detail,omitempty"`
	Placement string `json:"placement,omitempty"`
}

// NarrativeResponse is the structured JSON payload the AI embeds in its reply.
// Fields mirror the game engine's NarrativeResponse in internal/engine/types.go
// but are redeclared here so the ai package stays self-contained.
type NarrativeResponse struct {
	Narrative         string                 `json:"narrative"`
	Choices           []NarrativeChoice      `json:"choices,omitempty"`
	StateChanges      map[string]interface{} `json:"state_changes,omitempty"`
	Mood              string                 `json:"mood,omitempty"`
	Location          string                 `json:"location,omitempty"`
	SceneType         string                 `json:"scene_type,omitempty"`
	DialogueBlocks    []DialogueBlock        `json:"dialogue_blocks,omitempty"`
	EntitiesMentioned []EntityMention        `json:"entities_mentioned,omitempty"`
	EventCallouts     []EventCallout         `json:"event_callouts,omitempty"`
	ASCIICue          *ASCIIArtCue           `json:"ascii_cue,omitempty"`
	ASCIIArt          string                 `json:"ascii_art,omitempty"`
	AchievementEarned *AchievementPayload    `json:"achievement_earned,omitempty"`
	Challenge         string                 `json:"challenge,omitempty"`
}

// ASCIIArtGenerationResponse is the structured payload returned by the
// dedicated ASCII-art generation prompt.
type ASCIIArtGenerationResponse struct {
	ASCIIArt string `json:"ascii_art"`
}

// NarrativeChoice is a single player-facing option embedded in a NarrativeResponse.
type NarrativeChoice struct {
	ID           int      `json:"id"`
	Text         string   `json:"text"`
	Mood         string   `json:"mood,omitempty"`
	Intent       string   `json:"intent,omitempty"`
	Risk         string   `json:"risk,omitempty"`
	Scope        string   `json:"scope,omitempty"`
	Certainty    string   `json:"certainty,omitempty"`
	RelatedStats []string `json:"related_stats,omitempty"`
}

// ParseNarrativeJSON extracts the JSON block from an AI text response and
// unmarshals it into a NarrativeResponse.  Returns nil, nil when no JSON block
// is present (pure prose response).
func ParseNarrativeJSON(text string) (*NarrativeResponse, error) {
	raw, err := ExtractJSONPayload(text)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return nil, nil
	}
	var nr NarrativeResponse
	if err := json.Unmarshal([]byte(raw), &nr); err != nil {
		return nil, err
	}
	return &nr, nil
}

// ParseASCIIArtJSON extracts a structured ASCII-art payload from an AI response.
func ParseASCIIArtJSON(text string) (*ASCIIArtGenerationResponse, error) {
	raw, err := ExtractJSONPayload(text)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return nil, nil
	}
	var payload ASCIIArtGenerationResponse
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// ExtractNarrative returns the prose portion of an AI response, stripping any
// fenced JSON blocks and trimming surrounding whitespace.
func ExtractNarrative(text string) string {
	stripped := jsonBlockRe.ReplaceAllString(text, "")
	return strings.TrimSpace(stripped)
}

// extractJSONBlock pulls the raw JSON string out of the first ```json block.
// Returns empty string (no error) when no block is found.
func extractJSONBlock(text string) (string, error) {
	matches := jsonBlockRe.FindStringSubmatch(text)
	if len(matches) < 2 {
		return "", nil
	}
	// Validate it is actually JSON before returning.
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(matches[1]), &raw); err != nil {
		return "", err
	}
	return matches[1], nil
}

// ExtractJSONPayload returns a JSON payload from an AI response.
// It first looks for a fenced ```json block, then falls back to the entire
// trimmed response when that response is itself valid JSON.
func ExtractJSONPayload(text string) (string, error) {
	raw, err := extractJSONBlock(text)
	if err != nil || raw != "" {
		return raw, err
	}

	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", nil
	}

	var payload json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return "", nil
	}
	return trimmed, nil
}
