package engine

import (
	"strings"
	"testing"

	"github.com/crimsab/oneday/internal/storage"
)

func TestDetectNarrativeContinuityIssueFlagsEnglishCyberpunkDrift(t *testing.T) {
	story := &storage.Story{
		Name:        "Mondo di Lucernel",
		Genre:       "fantasy",
		Language:    "italiano",
		SettingJSON: `{"technology_level":"mythic fantasy","world_name":"Lucernel"}`,
	}
	narrative := &NarrativeResponse{
		Narrative: "The neon lights of the lower district flicker over the rain-slicked pavement near the Data-Haven while a chrome-plated guard watches you.",
	}

	issue := detectNarrativeContinuityIssue(story, narrative)
	if issue == nil {
		t.Fatal("expected continuity issue for English cyberpunk drift")
	}
	if !strings.Contains(issue.Error(), "configured story language") {
		t.Fatalf("expected language drift reason, got %q", issue.Error())
	}
	if !strings.Contains(issue.Error(), "cyberpunk") {
		t.Fatalf("expected setting drift reason, got %q", issue.Error())
	}
}

func TestDetectNarrativeContinuityIssueAllowsCyberpunkStories(t *testing.T) {
	story := &storage.Story{
		Name:        "Neon District",
		Genre:       "cyberpunk",
		Language:    "english",
		SettingJSON: `{"technology_level":"high tech sprawl"}`,
	}
	narrative := &NarrativeResponse{
		Narrative: "The neon lights flicker above the district while a chrome-plated courier steps out of the Data-Haven tunnel.",
	}

	if issue := detectNarrativeContinuityIssue(story, narrative); issue != nil {
		t.Fatalf("did not expect continuity issue for configured cyberpunk story: %v", issue)
	}
}
