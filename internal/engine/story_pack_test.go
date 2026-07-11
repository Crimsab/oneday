package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadStoryPackValidatesAndSelectsAuthorableChallenges(t *testing.T) {
	pack, err := LoadStoryPack("../../plugins/examples/noir-investigation.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if pack.StatsSchema == nil || pack.StatsSchema.HasCombat || pack.StatsSchema.Currency == nil {
		t.Fatalf("stats schema was not decoded: %+v", pack.StatsSchema)
	}
	if pack.WorldRules == nil || pack.VisualBible == nil || pack.VoiceBible == nil || len(pack.VoiceBible.Profiles) != 1 {
		t.Fatalf("authoring bibles were not decoded: world=%+v visual=%+v voice=%+v", pack.WorldRules, pack.VisualBible, pack.VoiceBible)
	}
	selection, err := pack.Select("investigation", "accessible", MiniGameSelectionContext{NarrativeTags: []string{"clue"}, CurrentTurn: 4})
	if err != nil {
		t.Fatal(err)
	}
	if selection.Definition.Difficulty < 1 || selection.Definition.Kind == MiniGameQuickTime {
		t.Fatalf("invalid accessible selection: %+v", selection)
	}
}

func TestLoadStoryPackRejectsDuplicateStatsAndUnknownMiniGames(t *testing.T) {
	for name, body := range map[string]string{
		"duplicate-stat":   "id: bad\nname: Bad\ndescription: Bad\nstats_schema:\n  vitals: [{key: focus, label: Focus}]\n  attributes: [{key: focus, label: Focus}]\n",
		"unknown-kind":     "id: bad\nname: Bad\ndescription: Bad\nchallenge_pools:\n  x:\n    definitions: [{id: x, kind: imaginary, difficulty: 50}]\n",
		"unknown-field":    "id: bad\nname: Bad\ndescription: Bad\nmagic_knob: true\n",
		"bad-stat-range":   "id: bad\nname: Bad\ndescription: Bad\nstats_schema:\n  attributes: [{key: focus, label: Focus, starting: 9, min: 0, max: 5}]\n",
		"bad-world":        "id: bad\nname: Bad\ndescription: Bad\nworld_rules: {clock_mode: guessed, weather_mode: tracked, travel_mode: graph}\n",
		"unlicensed-voice": "id: bad\nname: Bad\ndescription: Bad\nvoice_bible:\n  default_language: en\n  profiles: [{id: voice, provider: local, provider_voice_id: piper, languages: [en]}]\n",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "pack.yaml")
			if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadStoryPack(path); err == nil {
				t.Fatal("invalid story pack was accepted")
			}
		})
	}
}
