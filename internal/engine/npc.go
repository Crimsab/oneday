package engine

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/crimsab/oneday/internal/storage"
	"github.com/google/uuid"
)

// NPCPersonality holds the structured personality of an NPC.
type NPCPersonality struct {
	Traits      []string `json:"traits"`
	SpeechStyle string   `json:"speech_style"`
	Quirks      []string `json:"quirks"`
	Values      []string `json:"values"`
	Fears       []string `json:"fears"`
}

// NPCDesire represents a hidden NPC motivation.
type NPCDesire struct {
	Desire        string `json:"desire"`
	Priority      string `json:"priority"`
	KnownToPlayer bool   `json:"known_to_player"`
}

// NPCData is the AI-generated NPC structure from state_changes.
type NPCData struct {
	Name            string         `json:"name"`
	Role            string         `json:"role"`
	Appearance      string         `json:"appearance"`
	Personality     NPCPersonality `json:"personality"`
	PrivateThoughts []string       `json:"private_thoughts"`
	Desires         []NPCDesire    `json:"desires"`
	Disposition     int            `json:"disposition"`
	CanHelp         bool           `json:"can_help"`
}

// ParseNPCData converts an untyped map (from AI JSON) into a structured NPCData.
// Missing fields receive sensible defaults; never panics on malformed input.
func ParseNPCData(raw map[string]interface{}) (*NPCData, error) {
	if raw == nil {
		return nil, fmt.Errorf("npc data map is nil")
	}

	data := &NPCData{}

	if v, ok := raw["name"].(string); ok {
		data.Name = v
	}
	if v, ok := raw["role"].(string); ok {
		data.Role = v
	}
	if v, ok := raw["appearance"].(string); ok {
		data.Appearance = v
	}
	if v, ok := raw["disposition"]; ok {
		data.Disposition = int(toFloat(v))
	}
	if v, ok := raw["can_help"].(bool); ok {
		data.CanHelp = v
	}

	// Parse personality sub-object
	if pRaw, ok := raw["personality"].(map[string]interface{}); ok {
		if v, ok := pRaw["speech_style"].(string); ok {
			data.Personality.SpeechStyle = v
		}
		data.Personality.Traits = toStringSlice(pRaw["traits"])
		data.Personality.Quirks = toStringSlice(pRaw["quirks"])
		data.Personality.Values = toStringSlice(pRaw["values"])
		data.Personality.Fears = toStringSlice(pRaw["fears"])
	}

	// Parse private_thoughts
	data.PrivateThoughts = toStringSlice(raw["private_thoughts"])

	// Parse desires array
	if dRaw, ok := raw["desires"].([]interface{}); ok {
		for _, item := range dRaw {
			dMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			d := NPCDesire{}
			if v, ok := dMap["desire"].(string); ok {
				d.Desire = v
			}
			if v, ok := dMap["priority"].(string); ok {
				d.Priority = v
			}
			if v, ok := dMap["known_to_player"].(bool); ok {
				d.KnownToPlayer = v
			}
			data.Desires = append(data.Desires, d)
		}
	}

	if data.Name == "" {
		return nil, fmt.Errorf("npc data missing required field: name")
	}

	return data, nil
}

// NPCToStorage converts NPCData + metadata into a storage.NPC ready for DB insert.
func NPCToStorage(data *NPCData, storyID string, turn int) (*storage.NPC, error) {
	personalityBytes, err := json.Marshal(data.Personality)
	if err != nil {
		return nil, fmt.Errorf("marshaling npc personality: %w", err)
	}

	thoughtsBytes, err := json.Marshal(data.PrivateThoughts)
	if err != nil {
		return nil, fmt.Errorf("marshaling npc private_thoughts: %w", err)
	}

	desiresBytes, err := json.Marshal(data.Desires)
	if err != nil {
		return nil, fmt.Errorf("marshaling npc desires: %w", err)
	}

	now := time.Now()
	npc := &storage.NPC{
		ID:                 uuid.New().String(),
		StoryID:            storyID,
		Name:               data.Name,
		Role:               data.Role,
		Appearance:         data.Appearance,
		PersonalityJSON:    string(personalityBytes),
		PrivateThoughts:    string(thoughtsBytes),
		NotesOnProtagonist: "[]",
		Desires:            string(desiresBytes),
		Disposition:        data.Disposition,
		IsAlive:            true,
		FirstAppearedTurn:  turn,
		LastSeenTurn:       turn,
		CanHelp:            data.CanHelp,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	return npc, nil
}

// FormatNPCForContext returns a full text representation of an NPC for injection
// into the AI context. Includes private thoughts, desires, and notes (the AI sees all).
func FormatNPCForContext(npc *storage.NPC) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "### NPC: %s (%s)\n", npc.Name, npc.Role)
	if npc.Appearance != "" {
		fmt.Fprintf(&sb, "Appearance: %s\n", npc.Appearance)
	}

	// Parse and format personality
	var personality NPCPersonality
	if err := json.Unmarshal([]byte(npc.PersonalityJSON), &personality); err == nil {
		if len(personality.Traits) > 0 {
			fmt.Fprintf(&sb, "Personality: %s", strings.Join(personality.Traits, ", "))
		}
		if personality.SpeechStyle != "" {
			fmt.Fprintf(&sb, " | Speech: %s", personality.SpeechStyle)
		}
		sb.WriteString("\n")
		if len(personality.Quirks) > 0 {
			fmt.Fprintf(&sb, "Quirks: %s\n", strings.Join(personality.Quirks, "; "))
		}
		if len(personality.Values) > 0 || len(personality.Fears) > 0 {
			fmt.Fprintf(&sb, "Values: %s | Fears: %s\n",
				strings.Join(personality.Values, ", "),
				strings.Join(personality.Fears, ", "))
		}
	}

	fmt.Fprintf(&sb, "Disposition toward protagonist: %d", npc.Disposition)
	switch {
	case npc.Disposition > 50:
		sb.WriteString(" (friendly)\n")
	case npc.Disposition < -50:
		sb.WriteString(" (hostile)\n")
	default:
		sb.WriteString(" (neutral)\n")
	}

	// Private thoughts
	var thoughts []string
	if err := json.Unmarshal([]byte(npc.PrivateThoughts), &thoughts); err == nil && len(thoughts) > 0 {
		sb.WriteString("Private thoughts: ")
		quoted := make([]string, len(thoughts))
		for i, t := range thoughts {
			quoted[i] = fmt.Sprintf("%q", t)
		}
		sb.WriteString(strings.Join(quoted, "; "))
		sb.WriteString("\n")
	}

	// Desires
	var desires []NPCDesire
	if err := json.Unmarshal([]byte(npc.Desires), &desires); err == nil && len(desires) > 0 {
		for _, d := range desires {
			visibility := "hidden from player"
			if d.KnownToPlayer {
				visibility = "known to player"
			}
			fmt.Fprintf(&sb, "Desires: [%s] %s (%s)\n", strings.ToUpper(d.Priority), d.Desire, visibility)
		}
	}

	// Notes on protagonist
	var notes []string
	if err := json.Unmarshal([]byte(npc.NotesOnProtagonist), &notes); err == nil {
		if len(notes) > 0 {
			fmt.Fprintf(&sb, "Notes on protagonist: %s\n", strings.Join(notes, "; "))
		} else {
			sb.WriteString("Notes on protagonist: (none)\n")
		}
	}

	return sb.String()
}

// FormatNPCForPlayer returns a player-visible summary of an NPC.
// Excludes private thoughts and hidden desires.
func FormatNPCForPlayer(npc *storage.NPC) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "### %s (%s)\n", npc.Name, npc.Role)
	if npc.Appearance != "" {
		fmt.Fprintf(&sb, "Appearance: %s\n", npc.Appearance)
	}

	// Parse and format personality (public info only)
	var personality NPCPersonality
	if err := json.Unmarshal([]byte(npc.PersonalityJSON), &personality); err == nil {
		if len(personality.Traits) > 0 {
			fmt.Fprintf(&sb, "Traits: %s\n", strings.Join(personality.Traits, ", "))
		}
		if personality.SpeechStyle != "" {
			fmt.Fprintf(&sb, "Speech: %s\n", personality.SpeechStyle)
		}
	}

	fmt.Fprintf(&sb, "Disposition: %d", npc.Disposition)
	switch {
	case npc.Disposition > 50:
		sb.WriteString(" (friendly)\n")
	case npc.Disposition < -50:
		sb.WriteString(" (hostile)\n")
	default:
		sb.WriteString(" (neutral)\n")
	}

	// Only show desires that are known to the player
	var desires []NPCDesire
	if err := json.Unmarshal([]byte(npc.Desires), &desires); err == nil {
		for _, d := range desires {
			if d.KnownToPlayer {
				fmt.Fprintf(&sb, "Known goal: %s\n", d.Desire)
			}
		}
	}

	return sb.String()
}
