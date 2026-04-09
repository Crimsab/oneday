package views

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crimsab/oneday/internal/engine"
)

func TestNarrativeSlashCommandsOpenCodexBrowsers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name              string
		input             string
		wantTitle         string
		wantCategory      string
		wantEntries       bool
		wantProjectScreen bool
	}{
		{name: "codex", input: "/codex", wantTitle: "Codex"},
		{name: "investigations", input: "/investigations", wantTitle: "Investigations", wantCategory: "investigations", wantEntries: true},
		{name: "projects", input: "/projects", wantTitle: "Projects", wantProjectScreen: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			model := newSystemSmokeNarrativeModel(t)
			model.SetSize(120, 40)
			model.inputFocus = true
			model.input.Focus()
			model.input.SetValue(tc.input)

			updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
			if cmd != nil {
				t.Fatalf("Update(%s) returned unexpected cmd", tc.input)
			}
			if tc.wantProjectScreen {
				if updated.projectBrowser == nil || !updated.projectBrowser.Visible() {
					t.Fatalf("project browser not visible after %s", tc.input)
				}
				if updated.projectBrowser.title != tc.wantTitle {
					t.Fatalf("project browser title = %q, want %q", updated.projectBrowser.title, tc.wantTitle)
				}
				if updated.codexBrowser != nil && updated.codexBrowser.Visible() {
					t.Fatalf("codex browser should stay closed for %s", tc.input)
				}
				return
			}
			if updated.codexBrowser == nil || !updated.codexBrowser.Visible() {
				t.Fatalf("codex browser not visible after %s", tc.input)
			}
			if updated.codexBrowser.title != tc.wantTitle {
				t.Fatalf("codex title = %q, want %q", updated.codexBrowser.title, tc.wantTitle)
			}
			if tc.wantCategory != "" && updated.codexBrowser.currentCategory() != tc.wantCategory {
				t.Fatalf("current category = %q, want %q", updated.codexBrowser.currentCategory(), tc.wantCategory)
			}
			if tc.wantEntries && len(updated.codexBrowser.currentEntryIDs()) == 0 {
				t.Fatalf("%s category has no visible entries", tc.wantCategory)
			}
		})
	}
}

func TestNarrativeQuickSaveHotkeyCreatesSnapshot(t *testing.T) {
	t.Parallel()

	model := newSystemSmokeNarrativeModel(t)
	model.SetSize(120, 40)
	model.inputFocus = false

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if cmd == nil {
		t.Fatal("quicksave hotkey returned nil cmd")
	}

	msg := cmd()
	saveMsg, ok := msg.(SaveCompleteMsg)
	if !ok {
		t.Fatalf("quicksave cmd returned %T, want SaveCompleteMsg", msg)
	}
	if saveMsg.Err != nil {
		t.Fatalf("quicksave returned unexpected error: %v", saveMsg.Err)
	}
	if saveMsg.Name != "Quicksave T10" {
		t.Fatalf("quicksave name = %q, want Quicksave T10", saveMsg.Name)
	}

	updated, _ = updated.Update(saveMsg)
	if updated.statusMsg != "Saved: Quicksave T10" {
		t.Fatalf("statusMsg = %q, want quicksave confirmation", updated.statusMsg)
	}

	saves, err := engine.ListSaves(updated.narrator.DB(), updated.StoryID())
	if err != nil {
		t.Fatalf("ListSaves: %v", err)
	}
	if len(saves) != 1 {
		t.Fatalf("len(saves) = %d, want 1", len(saves))
	}
	if saves[0].Name != "Quicksave T10" {
		t.Fatalf("save name = %q, want Quicksave T10", saves[0].Name)
	}
	meta := saves[0].Metadata()
	if meta == nil || meta.Kind != "quicksave" {
		t.Fatalf("save metadata = %+v, want quicksave kind", meta)
	}

	savePath := filepath.Join(updated.narrator.DataDir(), "stories", updated.StoryID(), "saves", saves[0].ID+".json")
	if _, err := os.Stat(savePath); err != nil {
		t.Fatalf("save file missing at %s: %v", savePath, err)
	}
}

func newSystemSmokeNarrativeModel(t *testing.T) NarrativeModel {
	t.Helper()

	model := newTalkTestNarrativeModel(t)
	world := model.narrator.World()
	world.KnownLocationsJSON = `[{"name":"Harbor","description":"A wind-scoured harbor.","discovered_turn":1}]`
	world.InvestigationBoardJSON = `{"cases":[{"id":"case-harbor","title":"Who sold you out?","summary":"Find the broker who leaked your route.","status":"open","updated_turn":10,"clues":[{"id":"clue-seal","label":"A missing seal","detail":"The ledger page was torn out.","status":"known"}],"suspects":[{"id":"suspect-courier","name":"Old Guard courier","detail":"He vanished after dusk.","status":"person_of_interest"}],"contradictions":[{"id":"contr-alibi","label":"The courier's alibi is false","detail":"Two dockhands place him elsewhere.","status":"open"}],"leads":[{"id":"lead-warehouse","title":"Bell Quarter warehouse","detail":"Someone is using the old loading bay.","status":"active"}]}]}`
	world.ProjectClocksJSON = `{"projects":[{"id":"project-training","title":"Train with Lyanna","kind":"training","summary":"You keep drilling footwork before dawn.","status":"active","progress":2,"segments":4,"updated_turn":10,"owner":"Lyanna","location":"Harbor","stakes":"Be ready before the Bell Choir moves.","rewards":[{"kind":"skill","label":"Blade Forms"}]}]}`
	if err := model.narrator.DB().UpdateWorldState(world); err != nil {
		t.Fatalf("UpdateWorldState: %v", err)
	}

	return model
}
