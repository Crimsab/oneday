package views

import (
	"strings"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/i18n"
)

func TestItalianImportantSurfacesDoNotLeakEnglishHeadings(t *testing.T) {
	loc := i18n.New(i18n.Italian)
	menu := NewMenuModel(loc)
	menu.SetSize(80, 24)
	load := NewLoadStoryModel(nil, loc)
	load.SetSize(80, 24)
	saves := NewSaveLoadModel(nil, loc)
	saves.SetSize(80, 24)
	settings := NewSettingsModel(config.Default(), loc)
	settings.SetSize(100, 40)
	creator := engine.NewStoryCreator(nil, nil, config.GenerationConfig{}, loc)
	wizard := NewNewStoryModel(creator, loc)
	wizard.SetSize(100, 40)
	fronts := NewFrontTrackerModel("Fronti", engine.FrontTrackerBoard{}, 100, 30, loc)
	projects := NewProjectBrowserModel("Progetti", engine.ProjectBoard{}, 100, 30, loc)
	investigations := NewInvestigationBrowserModel("Indagini", engine.InvestigationBoard{}, 100, 30, loc)

	for label, view := range map[string]string{
		"menu": menu.View(), "library": load.View(), "saves": saves.View(),
		"settings": settings.View(), "wizard": wizard.View(), "fronts": fronts.View(),
		"projects": projects.View(), "investigations": investigations.View(),
	} {
		if strings.Contains(view, "Settings") || strings.Contains(view, "Load Story") || strings.Contains(view, "Create Your Story") || strings.Contains(view, "No investigations") || strings.Contains(view, "No downtime") {
			t.Errorf("%s leaked an English heading:\n%s", label, view)
		}
	}
}

func TestItalianResidualPresentationKeepsCanonicalTokens(t *testing.T) {
	t.Parallel()
	loc := i18n.New(i18n.Italian)

	intents := buildTalkIntentSuggestionItems("Lyanna", "", loc)
	if len(intents) == 0 || intents[0].Label != "ask" || !strings.Contains(intents[0].Hint, "Chiedi") {
		t.Fatalf("talk intent token or localized hint changed unexpectedly: %#v", intents)
	}

	items := sessionMenuItems(loc)
	if items[0].Label != "Riprendi" || items[len(items)-1].Label != "Menu principale" {
		t.Fatalf("session menu was not localized: %#v", items)
	}

	if got := historyRoleLabel("assistant", loc); got != "Narratore" {
		t.Fatalf("history role label=%q", got)
	}
	if got := renderHistoryTurnDivider(4, time.Time{}, loc); !strings.Contains(got, "Turno 4") {
		t.Fatalf("history turn divider=%q", got)
	}
}

func TestItalianChoiceFallbackAndBrowserHints(t *testing.T) {
	t.Parallel()
	loc := i18n.New(i18n.Italian)

	help := buildChoiceHelp(engine.Choice{Text: "Attraversa il ponte"}, nil, nil, loc)
	if !strings.Contains(help, "questa scelta non include metadati strutturati") || strings.Contains(help, "no structured metadata") {
		t.Fatalf("choice fallback was not localized: %q", help)
	}

	codex := NewCodexBrowserModel("Codice", nil, 100, 30, "", "", loc)
	if view := codex.View(); !strings.Contains(view, "progetti") || strings.Contains(view, "projects") {
		t.Fatalf("codex navigation was not localized: %q", view)
	}

	achievements := NewAchievementBrowserModel("Traguardi", nil, 100, 30, loc)
	if view := achievements.View(); strings.Contains(view, "Enter") || strings.Contains(view, "close") {
		t.Fatalf("achievement navigation leaked English: %q", view)
	}
}
