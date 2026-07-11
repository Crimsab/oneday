package engine

import (
	"fmt"
	"strings"
	"unicode"
)

func normalizeSocialDuelCue(cue *SocialDuelCue) *SocialDuelCue {
	if cue == nil {
		return nil
	}

	normalized := &SocialDuelCue{
		Mode:             normalizeSocialDuelCueMode(cue.Mode),
		NPCName:          strings.TrimSpace(cue.NPCName),
		Objective:        normalizeNarrativeText(cue.Objective),
		NPCGoal:          normalizeNarrativeText(cue.NPCGoal),
		Stakes:           normalizeNarrativeText(cue.Stakes),
		Pressure:         normalizeNarrativeText(cue.Pressure),
		Opening:          normalizeNarrativeText(cue.Opening),
		ExchangeSummary:  normalizeNarrativeText(cue.ExchangeSummary),
		Leverage:         normalizeSocialDuelLeverage(cue.Leverage),
		SuggestedActions: normalizeSuggestedSocialActions(cue.SuggestedActions),
		FailForward:      normalizeNarrativeText(cue.FailForward),
	}

	if normalized.Stakes == "" {
		normalized.Stakes = firstNonEmpty(normalized.FailForward, normalized.Pressure)
	}

	if normalized.Mode == "" {
		switch {
		case normalized.NPCName != "" && normalized.Objective != "":
			normalized.Mode = SocialDuelCueOffer
		case socialDuelCueHasFrame(normalized):
			normalized.Mode = SocialDuelCueContinue
		default:
			return nil
		}
	}

	switch normalized.Mode {
	case SocialDuelCueOffer:
		if normalized.NPCName == "" || normalized.Objective == "" {
			if !socialDuelCueHasFrame(normalized) {
				return nil
			}
			normalized.Mode = SocialDuelCueContinue
		}
	case SocialDuelCueContinue:
		if !socialDuelCueHasFrame(normalized) && normalized.NPCName == "" && normalized.Objective == "" {
			return nil
		}
	default:
		return nil
	}

	if normalized.Mode == SocialDuelCueContinue {
		normalized.ExchangeSummary = firstNonEmpty(normalized.ExchangeSummary, normalized.Opening, normalized.Pressure, normalized.Stakes)
	}

	if len(normalized.Leverage) == 0 {
		normalized.Leverage = nil
	}
	if len(normalized.SuggestedActions) == 0 {
		normalized.SuggestedActions = nil
	}

	return normalized
}

func normalizeSocialDuelCueMode(mode SocialDuelCueMode) SocialDuelCueMode {
	switch SocialDuelCueMode(strings.ToLower(strings.TrimSpace(string(mode)))) {
	case SocialDuelCueOffer:
		return SocialDuelCueOffer
	case SocialDuelCueContinue:
		return SocialDuelCueContinue
	default:
		return ""
	}
}

func normalizeSuggestedSocialActions(actions []SocialAction) []SocialAction {
	if len(actions) == 0 {
		return nil
	}

	out := make([]SocialAction, 0, len(actions))
	seen := make(map[SocialAction]bool, len(actions))
	for _, action := range actions {
		normalized, ok := sanitizeSuggestedSocialAction(action)
		if !ok || seen[normalized] {
			continue
		}
		seen[normalized] = true
		out = append(out, normalized)
	}
	return out
}

func sanitizeSuggestedSocialAction(action SocialAction) (SocialAction, bool) {
	switch SocialAction(strings.ToLower(strings.TrimSpace(string(action)))) {
	case SocialActionAppeal:
		return SocialActionAppeal, true
	case SocialActionPressure:
		return SocialActionPressure, true
	case SocialActionDeceive:
		return SocialActionDeceive, true
	case SocialActionConcede:
		return SocialActionConcede, true
	case SocialActionExpose:
		return SocialActionExpose, true
	case SocialActionWithdraw:
		return SocialActionWithdraw, true
	case SocialActionEscalate:
		return SocialActionEscalate, true
	default:
		return "", false
	}
}

func normalizeSocialDuelLeverage(items []SocialDuelLeverage) []SocialDuelLeverage {
	if len(items) == 0 {
		return nil
	}

	out := make([]SocialDuelLeverage, 0, len(items))
	seen := make(map[string]bool, len(items))
	for _, item := range items {
		label := strings.Join(strings.Fields(normalizeNarrativeText(item.Label)), " ")
		detail := normalizeNarrativeText(item.Detail)
		if label == "" && detail == "" {
			continue
		}
		if label == "" {
			label = detail
		}

		key := socialDuelLeverageKey(item.Key, label, detail, len(out)+1)
		if seen[key] {
			continue
		}
		seen[key] = true

		out = append(out, SocialDuelLeverage{
			Key:    key,
			Label:  label,
			Detail: detail,
			Kind:   strings.ToLower(strings.TrimSpace(item.Kind)),
		})
	}

	return out
}

func socialDuelLeverageKey(rawKey, label, detail string, index int) string {
	candidates := []string{
		strings.TrimSpace(rawKey),
		label,
		detail,
		fmt.Sprintf("leverage-%d", index),
	}
	for _, candidate := range candidates {
		if key := slugifySocialDuel(candidate); key != "" {
			return key
		}
	}
	return fmt.Sprintf("leverage-%d", index)
}

func socialDuelCueHasFrame(cue *SocialDuelCue) bool {
	if cue == nil {
		return false
	}
	return cue.NPCGoal != "" ||
		cue.Stakes != "" ||
		cue.Pressure != "" ||
		cue.Opening != "" ||
		cue.ExchangeSummary != "" ||
		cue.FailForward != "" ||
		len(cue.Leverage) > 0 ||
		len(cue.SuggestedActions) > 0
}

func slugifySocialDuel(value string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		case !lastDash:
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
