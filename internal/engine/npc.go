package engine

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
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
		RelationshipJSON:   "{}",
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
	if rel := loadRelationshipAxes(npc); rel != (RelationshipAxes{}) {
		fmt.Fprintf(&sb, "Relationship axes: trust=%d fear=%d debt=%d respect=%d intimacy=%d\n",
			rel.Trust, rel.Fear, rel.Debt, rel.Respect, rel.Intimacy)
	}
	if profile := loadNemesisProfile(npc); profile != nil {
		fmt.Fprintf(&sb, "Nemesis status: %s (tier %d, posture %s, score %d)\n",
			profile.Status, profile.EscalationTier, firstNonEmpty(profile.ThreatPosture, "watching"), profile.RivalryScore)
		if profile.Vow != "" {
			fmt.Fprintf(&sb, "Nemesis vow: %s\n", profile.Vow)
		}
		if len(profile.VisibleScars) > 0 {
			fmt.Fprintf(&sb, "Visible scars: %s\n", strings.Join(profile.VisibleScars, "; "))
		}
		if len(profile.RememberedPatterns) > 0 {
			fmt.Fprintf(&sb, "Remembered player patterns: %s\n", strings.Join(profile.RememberedPatterns, ", "))
		}
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

// UpdateNPCLastSeen checks narrativeText for mentions of known NPCs and updates
// their last_seen_turn to currentTurn. Matching is case-insensitive on the full
// name and on the first word of the name (first name). This is best-effort —
// errors are swallowed so game flow is never interrupted.
func UpdateNPCLastSeen(db *storage.DB, storyID string, narrativeText string, currentTurn int) error {
	return updateNPCLastSeen(db, nil, storyID, narrativeText, currentTurn)
}

func UpdateNPCLastSeenTx(db *storage.DB, tx *sql.Tx, storyID string, narrativeText string, currentTurn int) error {
	return updateNPCLastSeen(db, tx, storyID, narrativeText, currentTurn)
}

func updateNPCLastSeen(db *storage.DB, tx *sql.Tx, storyID string, narrativeText string, currentTurn int) error {
	if db == nil || narrativeText == "" {
		return nil
	}
	var npcs []storage.NPC
	var err error
	if tx != nil {
		npcs, err = db.ListNPCsTx(tx, storyID)
	} else {
		npcs, err = db.ListNPCs(storyID)
	}
	if err != nil {
		return nil // non-fatal
	}
	lower := strings.ToLower(narrativeText)
	for i := range npcs {
		npc := &npcs[i]
		// Match full name.
		fullLower := strings.ToLower(npc.Name)
		if strings.Contains(lower, fullLower) {
			npc.LastSeenTurn = currentTurn
			if tx != nil {
				_ = db.UpdateNPCTx(tx, npc)
			} else {
				_ = db.UpdateNPC(npc)
			}
			continue
		}
		// Match first name only (first whitespace-delimited word).
		parts := strings.Fields(npc.Name)
		if len(parts) > 1 {
			firstLower := strings.ToLower(parts[0])
			if strings.Contains(lower, firstLower) {
				npc.LastSeenTurn = currentTurn
				if tx != nil {
					_ = db.UpdateNPCTx(tx, npc)
				} else {
					_ = db.UpdateNPC(npc)
				}
			}
		}
	}
	return nil
}

// NearbyNPCs returns the most relevant NPCs for the current scene.
// It prefers NPCs seen in the last few turns, then falls back to the most recently seen roster.
func NearbyNPCs(db *storage.DB, storyID string, currentTurn, limit int) ([]storage.NPC, error) {
	return RelevantNPCs(db, storyID, currentTurn, 3, limit)
}

// RelevantNPCs returns a context-aware NPC roster for scene generation.
// It keeps recently seen NPCs, but can reintroduce eligible active nemeses even
// when they were not in the immediate last few turns.
func RelevantNPCs(db *storage.DB, storyID string, currentTurn, recentWithin, limit int) ([]storage.NPC, error) {
	if db == nil || storyID == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 5
	}
	if recentWithin <= 0 {
		recentWithin = 3
	}

	recent, recentErr := db.ListRecentNPCs(storyID, currentTurn, recentWithin)
	all, err := db.ListNPCs(storyID)
	if err != nil {
		if recentErr != nil {
			return nil, err
		}
		all = append([]storage.NPC(nil), recent...)
	}
	if len(all) == 0 {
		return nil, err
	}

	roster := prioritizeRelevantNPCs(recent, all, currentTurn)
	if len(roster) > limit {
		roster = roster[:limit]
	}
	return roster, nil
}

func prioritizeRelevantNPCs(recent, all []storage.NPC, currentTurn int) []storage.NPC {
	pool := make([]storage.NPC, 0, len(all))
	seen := map[string]bool{}

	for _, npc := range recent {
		if seen[npc.ID] {
			continue
		}
		seen[npc.ID] = true
		pool = append(pool, npc)
	}

	for _, npc := range all {
		if seen[npc.ID] {
			continue
		}
		if profile := loadNemesisProfile(&npc); profile != nil && nemesisEligibleForReentry(profile, npc.LastSeenTurn, currentTurn) {
			seen[npc.ID] = true
			pool = append(pool, npc)
		}
	}

	if len(pool) == 0 {
		pool = append(pool, all...)
	}

	sort.SliceStable(pool, func(i, j int) bool {
		left := npcEncounterPriority(pool[i], currentTurn)
		right := npcEncounterPriority(pool[j], currentTurn)
		if left == right {
			if pool[i].LastSeenTurn == pool[j].LastSeenTurn {
				return pool[i].FirstAppearedTurn > pool[j].FirstAppearedTurn
			}
			return pool[i].LastSeenTurn > pool[j].LastSeenTurn
		}
		return left > right
	})
	return pool
}

func npcEncounterPriority(npc storage.NPC, currentTurn int) int {
	score := npc.LastSeenTurn
	if profile := loadNemesisProfile(&npc); profile != nil {
		switch profile.Status {
		case NemesisStatusActive:
			score += 200 + profile.EscalationTier*20 + profile.RivalryScore
			if nemesisEligibleForReentry(profile, npc.LastSeenTurn, currentTurn) {
				score += 50
			}
		case NemesisStatusRival:
			score += 60 + profile.RivalryScore
		}
	}
	return score
}

func nemesisEligibleForReentry(profile *NemesisProfile, lastSeenTurn, currentTurn int) bool {
	if profile == nil || profile.Status != NemesisStatusActive {
		return false
	}
	cooldown := maxInt(1, 5-profile.EscalationTier)
	if strings.EqualFold(profile.ThreatPosture, "hunting") || strings.EqualFold(profile.ThreatPosture, "vengeful") {
		cooldown = 1
	}
	return currentTurn-lastSeenTurn >= cooldown
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
	if rel := loadRelationshipAxes(npc); rel != (RelationshipAxes{}) {
		fmt.Fprintf(&sb, "Trust/Fear/Debt/Respect/Intimacy: %d / %d / %d / %d / %d\n",
			rel.Trust, rel.Fear, rel.Debt, rel.Respect, rel.Intimacy)
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
