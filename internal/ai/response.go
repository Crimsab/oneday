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

// NarrativeResponse is the structured JSON payload the AI embeds in its reply.
// Fields mirror the game engine's NarrativeResponse in internal/engine/types.go
// but are redeclared here so the ai package stays self-contained.
type NarrativeResponse struct {
	Narrative         string                 `json:"narrative"`
	Choices           []NarrativeChoice      `json:"choices,omitempty"`
	StateChanges      map[string]interface{} `json:"state_changes,omitempty"`
	Mood              string                 `json:"mood,omitempty"`
	ASCIIArt          string                 `json:"ascii_art,omitempty"`
	AchievementEarned *AchievementPayload    `json:"achievement_earned,omitempty"`
	Challenge         string                 `json:"challenge,omitempty"`
}

// NarrativeChoice is a single player-facing option embedded in a NarrativeResponse.
type NarrativeChoice struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
	Mood string `json:"mood,omitempty"`
}

// ParseNarrativeJSON extracts the JSON block from an AI text response and
// unmarshals it into a NarrativeResponse.  Returns nil, nil when no JSON block
// is present (pure prose response).
func ParseNarrativeJSON(text string) (*NarrativeResponse, error) {
	raw, err := extractJSONBlock(text)
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
