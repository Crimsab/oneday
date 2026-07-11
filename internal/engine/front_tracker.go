package engine

import (
	"sort"
	"strings"

	"github.com/crimsab/oneday/internal/storage"
)

// FrontTrackerPressure is a player-safe pressure hotspot derived from visible fronts.
type FrontTrackerPressure struct {
	FrontID         string `json:"front_id,omitempty"`
	FrontTitle      string `json:"front_title,omitempty"`
	FrontStatus     string `json:"front_status,omitempty"`
	FrontVisibility string `json:"front_visibility,omitempty"`
	Region          string `json:"region"`
	Kind            string `json:"kind"`
	Level           int    `json:"level"`
	Severity        string `json:"severity,omitempty"`
	Summary         string `json:"summary,omitempty"`
	Detail          string `json:"detail,omitempty"`
	UpdatedTurn     int    `json:"updated_turn,omitempty"`
}

// FrontTrackerFront is a sanitized front surface for the dedicated tracker UI.
type FrontTrackerFront struct {
	ID                 string                 `json:"id"`
	Title              string                 `json:"title"`
	Faction            string                 `json:"faction,omitempty"`
	Stakes             string                 `json:"stakes,omitempty"`
	Status             string                 `json:"status,omitempty"`
	Visibility         string                 `json:"visibility,omitempty"`
	Progress           int                    `json:"progress,omitempty"`
	Segments           int                    `json:"segments,omitempty"`
	LastAdvancedTurn   int                    `json:"last_advanced_turn,omitempty"`
	NextEscalationTurn int                    `json:"next_escalation_turn,omitempty"`
	Resolution         string                 `json:"resolution,omitempty"`
	Pressures          []FrontTrackerPressure `json:"pressures,omitempty"`
}

// FrontTrackerBoard is the player-facing world-state tracker used by the TUI.
type FrontTrackerBoard struct {
	Hooks     []StoryHook            `json:"hooks,omitempty"`
	Fronts    []FrontTrackerFront    `json:"fronts,omitempty"`
	Hotspots  []FrontTrackerPressure `json:"hotspots,omitempty"`
	Reactions []WorldReaction        `json:"reactions,omitempty"`
}

// LoadFrontTrackerBoard returns a sanitized tracker board built from active hooks, visible fronts, hotspots, and fallout.
func LoadFrontTrackerBoard(world *storage.WorldState) FrontTrackerBoard {
	if world == nil {
		return FrontTrackerBoard{}
	}

	hooks := activeStoryHooks(loadStoryHooks(world))
	sort.SliceStable(hooks, func(i, j int) bool {
		left := trackerHookOrder(hooks[i])
		right := trackerHookOrder(hooks[j])
		if left != right {
			return left < right
		}
		if hooks[i].TimerTurns != hooks[j].TimerTurns {
			switch {
			case hooks[i].TimerTurns == 0:
				return false
			case hooks[j].TimerTurns == 0:
				return true
			default:
				return hooks[i].TimerTurns < hooks[j].TimerTurns
			}
		}
		if hooks[i].UpdatedTurn != hooks[j].UpdatedTurn {
			return hooks[i].UpdatedTurn > hooks[j].UpdatedTurn
		}
		return strings.ToLower(hooks[i].Title) < strings.ToLower(hooks[j].Title)
	})

	visibleFronts := knownFronts(loadFronts(world))
	trackerFronts := make([]FrontTrackerFront, 0, len(visibleFronts))
	hotspots := make([]FrontTrackerPressure, 0)
	for _, front := range visibleFronts {
		trackerFront := FrontTrackerFront{
			ID:                 strings.TrimSpace(front.ID),
			Title:              strings.TrimSpace(frontDisplayTitle(front)),
			Status:             strings.TrimSpace(front.Status),
			Visibility:         strings.TrimSpace(front.Visibility),
			Progress:           front.Progress,
			Segments:           front.Segments,
			LastAdvancedTurn:   front.LastAdvancedTurn,
			NextEscalationTurn: front.NextEscalationTurn,
			Stakes:             strings.TrimSpace(frontDisplayStakes(front)),
		}
		if strings.EqualFold(front.Visibility, "known") {
			trackerFront.Faction = strings.TrimSpace(front.Faction)
		}
		if strings.EqualFold(front.Visibility, "known") && strings.EqualFold(front.Status, "resolved") {
			trackerFront.Resolution = strings.TrimSpace(front.Resolution)
		}

		pressures := normalizeFrontPressures(front.Pressures)
		trackerFront.Pressures = make([]FrontTrackerPressure, 0, len(pressures))
		for _, pressure := range pressures {
			item := FrontTrackerPressure{
				FrontID:         trackerFront.ID,
				FrontTitle:      trackerFront.Title,
				FrontStatus:     trackerFront.Status,
				FrontVisibility: trackerFront.Visibility,
				Region:          strings.TrimSpace(pressure.Region),
				Kind:            strings.TrimSpace(pressure.Kind),
				Level:           pressure.Level,
				Severity:        frontPressureSeverity(pressure.Level),
				Summary:         formatFrontPressureDisplay(pressure),
				Detail:          strings.TrimSpace(pressure.Detail),
				UpdatedTurn:     pressure.UpdatedTurn,
			}
			trackerFront.Pressures = append(trackerFront.Pressures, item)
			hotspots = append(hotspots, item)
		}

		sort.SliceStable(trackerFront.Pressures, func(i, j int) bool {
			if trackerFront.Pressures[i].Level != trackerFront.Pressures[j].Level {
				return trackerFront.Pressures[i].Level > trackerFront.Pressures[j].Level
			}
			if trackerFront.Pressures[i].UpdatedTurn != trackerFront.Pressures[j].UpdatedTurn {
				return trackerFront.Pressures[i].UpdatedTurn > trackerFront.Pressures[j].UpdatedTurn
			}
			if trackerFront.Pressures[i].Region != trackerFront.Pressures[j].Region {
				return trackerFront.Pressures[i].Region < trackerFront.Pressures[j].Region
			}
			return trackerFront.Pressures[i].Kind < trackerFront.Pressures[j].Kind
		})

		trackerFronts = append(trackerFronts, trackerFront)
	}

	sort.SliceStable(trackerFronts, func(i, j int) bool {
		left := trackerFrontStatusOrder(trackerFronts[i].Status)
		right := trackerFrontStatusOrder(trackerFronts[j].Status)
		if left != right {
			return left < right
		}
		if trackerFronts[i].NextEscalationTurn != trackerFronts[j].NextEscalationTurn {
			switch {
			case trackerFronts[i].NextEscalationTurn == 0:
				return false
			case trackerFronts[j].NextEscalationTurn == 0:
				return true
			default:
				return trackerFronts[i].NextEscalationTurn < trackerFronts[j].NextEscalationTurn
			}
		}
		leftPeak := trackerFrontPeakPressure(trackerFronts[i])
		rightPeak := trackerFrontPeakPressure(trackerFronts[j])
		if leftPeak != rightPeak {
			return leftPeak > rightPeak
		}
		if trackerFronts[i].LastAdvancedTurn != trackerFronts[j].LastAdvancedTurn {
			return trackerFronts[i].LastAdvancedTurn > trackerFronts[j].LastAdvancedTurn
		}
		return strings.ToLower(trackerFronts[i].Title) < strings.ToLower(trackerFronts[j].Title)
	})

	sort.SliceStable(hotspots, func(i, j int) bool {
		if hotspots[i].Level != hotspots[j].Level {
			return hotspots[i].Level > hotspots[j].Level
		}
		if hotspots[i].UpdatedTurn != hotspots[j].UpdatedTurn {
			return hotspots[i].UpdatedTurn > hotspots[j].UpdatedTurn
		}
		if hotspots[i].Region != hotspots[j].Region {
			return hotspots[i].Region < hotspots[j].Region
		}
		if hotspots[i].FrontTitle != hotspots[j].FrontTitle {
			return hotspots[i].FrontTitle < hotspots[j].FrontTitle
		}
		return hotspots[i].Kind < hotspots[j].Kind
	})

	reactions := visibleWorldReactions(loadWorldReactions(world))
	sort.SliceStable(reactions, func(i, j int) bool {
		if reactions[i].CreatedTurn != reactions[j].CreatedTurn {
			return reactions[i].CreatedTurn > reactions[j].CreatedTurn
		}
		if reactions[i].SourceTurn != reactions[j].SourceTurn {
			return reactions[i].SourceTurn > reactions[j].SourceTurn
		}
		return strings.ToLower(reactions[i].Title) < strings.ToLower(reactions[j].Title)
	})

	return FrontTrackerBoard{
		Hooks:     hooks,
		Fronts:    trackerFronts,
		Hotspots:  hotspots,
		Reactions: reactions,
	}
}

func trackerHookOrder(hook StoryHook) int {
	switch strings.ToLower(strings.TrimSpace(hook.Status)) {
	case "active":
		return 0
	case "cooling":
		return 1
	default:
		return 2
	}
}

func trackerFrontStatusOrder(status string) int {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active":
		return 0
	case "stalled":
		return 1
	case "resolved":
		return 2
	default:
		return 3
	}
}

func trackerFrontPeakPressure(front FrontTrackerFront) int {
	peak := 0
	for _, pressure := range front.Pressures {
		if pressure.Level > peak {
			peak = pressure.Level
		}
	}
	return peak
}
