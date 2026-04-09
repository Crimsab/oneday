package engine

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/crimsab/oneday/internal/storage"
)

type NemesisStatus string

const (
	NemesisStatusRival    NemesisStatus = "rival"
	NemesisStatusActive   NemesisStatus = "active"
	NemesisStatusResolved NemesisStatus = "resolved"
)

type NemesisEvent struct {
	Kind    string `json:"kind"`
	Outcome string `json:"outcome,omitempty"`
	Detail  string `json:"detail,omitempty"`
	Turn    int    `json:"turn,omitempty"`
	Impact  int    `json:"impact,omitempty"`
	Pattern string `json:"pattern,omitempty"`
	Scar    string `json:"scar,omitempty"`
}

type NemesisProfile struct {
	Status             NemesisStatus  `json:"status"`
	RivalryScore       int            `json:"rivalry_score"`
	EscalationTier     int            `json:"escalation_tier"`
	ThreatPosture      string         `json:"threat_posture,omitempty"`
	Vow                string         `json:"vow,omitempty"`
	LastOutcome        string         `json:"last_outcome,omitempty"`
	LastSeenTurn       int            `json:"last_seen_turn,omitempty"`
	VisibleScars       []string       `json:"visible_scars,omitempty"`
	RememberedPatterns []string       `json:"remembered_patterns,omitempty"`
	EventHistory       []NemesisEvent `json:"event_history,omitempty"`
}

func loadNemesisProfile(npc *storage.NPC) *NemesisProfile {
	if npc == nil || strings.TrimSpace(npc.NemesisJSON) == "" || strings.TrimSpace(npc.NemesisJSON) == "{}" {
		return nil
	}
	var profile NemesisProfile
	if err := json.Unmarshal([]byte(npc.NemesisJSON), &profile); err != nil {
		return nil
	}
	normalizeNemesisProfile(&profile)
	return &profile
}

func storeNemesisProfile(npc *storage.NPC, profile *NemesisProfile) {
	if npc == nil {
		return
	}
	if profile == nil {
		npc.NemesisJSON = "{}"
		return
	}
	normalizeNemesisProfile(profile)
	payload, err := json.Marshal(profile)
	if err != nil {
		return
	}
	npc.NemesisJSON = string(payload)
}

func RecordNemesisEvent(npc *storage.NPC, event NemesisEvent) *NemesisProfile {
	if npc == nil {
		return nil
	}
	event = normalizeNemesisEvent(event)
	if !qualifiesForNemesisTracking(event) && loadNemesisProfile(npc) == nil {
		return nil
	}

	profile := loadNemesisProfile(npc)
	if profile == nil {
		profile = &NemesisProfile{
			Status:         NemesisStatusRival,
			ThreatPosture:  "watching",
			EscalationTier: 1,
		}
	}
	if profile.Status == NemesisStatusResolved {
		return profile
	}

	profile.RivalryScore += maxInt(1, event.Impact)
	profile.LastOutcome = firstNonEmpty(strings.TrimSpace(event.Outcome), profile.LastOutcome)
	profile.LastSeenTurn = maxInt(profile.LastSeenTurn, event.Turn)
	profile.ThreatPosture = updateNemesisThreatPosture(profile.ThreatPosture, event)
	profile.EventHistory = append(profile.EventHistory, event)
	if len(profile.EventHistory) > 8 {
		profile.EventHistory = profile.EventHistory[len(profile.EventHistory)-8:]
	}
	if pattern := strings.TrimSpace(event.Pattern); pattern != "" {
		profile.RememberedPatterns = appendUniqueLimited(profile.RememberedPatterns, pattern, 6)
	}
	if scar := strings.TrimSpace(event.Scar); scar != "" {
		profile.VisibleScars = appendUniqueLimited(profile.VisibleScars, scar, 4)
	}

	if shouldPromoteNemesis(profile, event) {
		profile.Status = NemesisStatusActive
	}
	profile.EscalationTier = nemesisEscalationTier(profile.RivalryScore, profile.Status)
	if profile.Status == NemesisStatusActive && strings.TrimSpace(profile.Vow) == "" {
		profile.Vow = defaultNemesisVow(npc, event)
	}

	storeNemesisProfile(npc, profile)
	return profile
}

func normalizeNemesisProfile(profile *NemesisProfile) {
	if profile == nil {
		return
	}
	switch profile.Status {
	case NemesisStatusActive, NemesisStatusResolved:
	default:
		profile.Status = NemesisStatusRival
	}
	if profile.RivalryScore < 0 {
		profile.RivalryScore = 0
	}
	if profile.EscalationTier <= 0 {
		profile.EscalationTier = 1
	}
	if profile.EscalationTier > 4 {
		profile.EscalationTier = 4
	}
	profile.ThreatPosture = strings.TrimSpace(profile.ThreatPosture)
	profile.Vow = strings.TrimSpace(profile.Vow)
	profile.LastOutcome = strings.TrimSpace(profile.LastOutcome)
	profile.VisibleScars = normalizeLimitedStrings(profile.VisibleScars, 4)
	profile.RememberedPatterns = normalizeLimitedStrings(profile.RememberedPatterns, 6)
	if len(profile.EventHistory) > 8 {
		profile.EventHistory = profile.EventHistory[len(profile.EventHistory)-8:]
	}
}

func normalizeNemesisEvent(event NemesisEvent) NemesisEvent {
	event.Kind = strings.ToLower(strings.TrimSpace(event.Kind))
	event.Outcome = strings.TrimSpace(event.Outcome)
	event.Detail = strings.TrimSpace(event.Detail)
	event.Pattern = strings.TrimSpace(event.Pattern)
	event.Scar = strings.TrimSpace(event.Scar)
	if event.Impact <= 0 {
		event.Impact = 1
	}
	return event
}

func qualifiesForNemesisTracking(event NemesisEvent) bool {
	switch event.Kind {
	case "escape", "humiliation", "betrayal", "political_fallout", "major_wound":
		return true
	default:
		return event.Impact >= 2
	}
}

func shouldPromoteNemesis(profile *NemesisProfile, event NemesisEvent) bool {
	if profile == nil {
		return false
	}
	if profile.Status == NemesisStatusActive {
		return true
	}
	if event.Kind == "betrayal" || event.Kind == "major_wound" {
		return true
	}
	return profile.RivalryScore >= 5
}

func updateNemesisThreatPosture(current string, event NemesisEvent) string {
	switch event.Kind {
	case "betrayal", "major_wound":
		return "vengeful"
	case "political_fallout":
		return "political"
	case "humiliation":
		return "obsessive"
	case "escape":
		return "hunting"
	default:
		if strings.TrimSpace(current) != "" {
			return current
		}
		return "watching"
	}
}

func nemesisEscalationTier(score int, status NemesisStatus) int {
	tier := 1
	switch {
	case score >= 9:
		tier = 4
	case score >= 7:
		tier = 3
	case score >= 5:
		tier = 2
	}
	if status == NemesisStatusActive && tier < 2 {
		tier = 2
	}
	return tier
}

func defaultNemesisVow(npc *storage.NPC, event NemesisEvent) string {
	name := "They"
	if npc != nil && strings.TrimSpace(npc.Name) != "" {
		name = npc.Name
	}
	if strings.TrimSpace(event.Detail) != "" {
		return fmt.Sprintf("%s swears to answer for %s.", name, event.Detail)
	}
	return fmt.Sprintf("%s swears this rivalry is not finished.", name)
}

func appendUniqueLimited(items []string, value string, limit int) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return normalizeLimitedStrings(items, limit)
	}
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), value) {
			return normalizeLimitedStrings(items, limit)
		}
	}
	items = append(items, value)
	return normalizeLimitedStrings(items, limit)
}

func normalizeLimitedStrings(items []string, limit int) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, trimmed)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}
