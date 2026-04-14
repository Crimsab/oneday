package engine

import (
	"strings"
	"testing"
)

func TestApplyStateChangesTimelineUpdatePersistsCanonicalTimeline(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()

	applied, err := ApplyStateChanges(map[string]interface{}{
		"timeline_update": map[string]interface{}{
			"age":        8,
			"life_stage": "childhood",
			"kind":       "time_skip",
			"label":      "First stable magical habit",
			"detail":     "Three years later, home practice has become a ritual.",
		},
	}, char, world, nil, "test-story-id", 6)
	if err != nil {
		t.Fatalf("ApplyStateChanges: %v", err)
	}
	if len(applied) != 1 {
		t.Fatalf("expected 1 timeline state change, got %d", len(applied))
	}
	if !strings.HasPrefix(applied[0].Description, "Timeline advanced: ") {
		t.Fatalf("unexpected timeline description: %+v", applied[0])
	}

	timeline := LoadCharacterTimeline(world)
	if timeline.CurrentAge != 8 {
		t.Fatalf("timeline age = %d, want 8", timeline.CurrentAge)
	}
	if timeline.LifeStage != "childhood" {
		t.Fatalf("timeline stage = %q, want childhood", timeline.LifeStage)
	}
	if len(timeline.Milestones) != 1 {
		t.Fatalf("timeline milestones = %+v, want 1", timeline.Milestones)
	}
	if timeline.Milestones[0].Label != "First stable magical habit" {
		t.Fatalf("milestone label = %q, want First stable magical habit", timeline.Milestones[0].Label)
	}
}

func TestBuildStateSummaryIncludesCharacterTimeline(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()
	storeCharacterTimeline(world, CharacterTimeline{
		CurrentAge: 8,
		LifeStage:  "childhood",
		Milestones: []TimelineMilestone{
			{
				Kind:      "time_skip",
				Age:       8,
				LifeStage: "childhood",
				Label:     "First stable magical habit",
				Detail:    "Practice at home is now routine.",
				Turn:      6,
			},
		},
	})

	summary := buildStateSummary(char, world, nil, "")
	if !strings.Contains(summary, "Timeline: Age 8") {
		t.Fatalf("summary missing timeline headline:\n%s", summary)
	}
	if !strings.Contains(summary, "Recent Milestones: First stable magical habit") {
		t.Fatalf("summary missing timeline milestone:\n%s", summary)
	}
}

func TestApplyStateChangesTimelineUpdateSupportsAmbiguousAge(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()

	applied, err := ApplyStateChanges(map[string]interface{}{
		"timeline_update": map[string]interface{}{
			"life_stage": "childhood",
			"kind":       "time_skip",
			"label":      "Later childhood",
			"detail":     "Enough seasons pass that home routines feel older, but no exact age is stated.",
		},
	}, char, world, nil, "test-story-id", 7)
	if err != nil {
		t.Fatalf("ApplyStateChanges: %v", err)
	}
	if len(applied) != 1 {
		t.Fatalf("expected 1 timeline state change, got %d", len(applied))
	}

	timeline := LoadCharacterTimeline(world)
	if timeline.CurrentAge != 0 {
		t.Fatalf("timeline age = %d, want unresolved 0", timeline.CurrentAge)
	}
	if timeline.LifeStage != "childhood" {
		t.Fatalf("timeline stage = %q, want childhood", timeline.LifeStage)
	}
	if len(timeline.Milestones) != 1 || timeline.Milestones[0].Label != "Later childhood" {
		t.Fatalf("unexpected timeline milestones: %+v", timeline.Milestones)
	}
	if !strings.Contains(applied[0].Description, "Later childhood") {
		t.Fatalf("expected ambiguous-age description to mention milestone, got %+v", applied[0])
	}
}

func TestBuildStateSummaryShowsUnresolvedTimelineMilestones(t *testing.T) {
	char := newTestChar()
	world := newTestWorld()
	storeCharacterTimeline(world, CharacterTimeline{
		LifeStage: "childhood",
		Milestones: []TimelineMilestone{
			{
				Kind:      "time_skip",
				LifeStage: "childhood",
				Label:     "Later childhood",
				Detail:    "Home practice grows steadier, but the exact age is still unspoken.",
				Turn:      7,
			},
		},
	})

	summary := buildStateSummary(char, world, nil, "")
	if !strings.Contains(summary, "Timeline: childhood") {
		t.Fatalf("summary missing life-stage-only timeline:\n%s", summary)
	}
	if !strings.Contains(summary, "Recent Milestones: Later childhood") {
		t.Fatalf("summary missing unresolved timeline milestone:\n%s", summary)
	}
}
