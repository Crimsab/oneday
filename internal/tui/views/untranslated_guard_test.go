package views

import (
	"os"
	"strings"
	"testing"
)

func TestInspectorDetailLabelsStayCatalogBacked(t *testing.T) {
	files := []string{"front_tracker.go", "project_browser.go", "investigation_browser.go"}
	forbidden := []string{`fmt.Sprintf("Status:`, `fmt.Sprintf("Progress:`, `fmt.Sprintf("Kind:`, `fmt.Sprintf("Owner:`, `fmt.Sprintf("Location:`, `fmt.Sprintf("Updated Turn:`, `fmt.Sprintf("Source Turn:`, `, "Summary",`, `, "Detail",`}
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, literal := range forbidden {
			if strings.Contains(string(raw), literal) {
				t.Errorf("%s contains untranslated presentation literal %s", file, literal)
			}
		}
	}
}

func TestResidualPresentationStaysCatalogBacked(t *testing.T) {
	checks := map[string][]string{
		"achievement_browser.go": {`"archived"`, `"↑↓ navigate · Enter`},
		"codex_browser.go":       {`"Related"`, `"↑↓ entries`, `"↑↓ scroll`},
		"crafting.go":            {`"  (zaino vuoto)`, `" [Tab: input libero]`},
		"narrative_choices.go":   {`"This choice signals:"`, `"- no structured metadata`},
	}
	for file, forbidden := range checks {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, literal := range forbidden {
			if strings.Contains(string(raw), literal) {
				t.Errorf("%s contains untranslated presentation literal %s", file, literal)
			}
		}
	}
}
