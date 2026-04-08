package engine

import "testing"

func TestExtractStoryJSONRequiresAuthoringFields(t *testing.T) {
	raw := `{
	  "name": "Le Campane di Vespera",
	  "description": "Una citta in rovina affacciata sul sale.",
	  "genre": "fantasy oscuro",
	  "tone": "malinconico",
	  "language": "italiano",
	  "writing_style": "prosa elegante e inquieta",
	  "prompt_directives": "Dialoghi asciutti.",
	  "setting": {
	    "world_name": "Vespera",
	    "era": "Eta delle Maree Spezzate",
	    "geography": "Laguna nera",
	    "magic_system": "Campane sommerse",
	    "technology_level": "Rinascimento decadente",
	    "society": "Casate e culti",
	    "rules": ["La magia ha un prezzo"],
	    "factions": ["Casata Valcerra"],
	    "cultures": ["Scavatori"],
	    "dangers": ["Nebbie senzienti"]
	  },
	  "stats_schema": {
	    "vitals": [{"key":"hp","label":"Salute","starting":10}],
	    "attributes": [{"key":"agi","label":"Agilita","starting":3}],
	    "secondary": [{"key":"rep","label":"Reputazione","starting":0}],
	    "currency": {"name":"Corone","starting":5},
	    "has_combat": true
	  }
	}`

	def := extractStoryJSON(raw)
	if def == nil {
		t.Fatal("extractStoryJSON returned nil")
	}
	if def.Language != "italiano" {
		t.Fatalf("Language = %q, want italiano", def.Language)
	}
	if def.WritingStyle == "" {
		t.Fatal("WritingStyle unexpectedly empty")
	}
}

func TestExtractStoryJSONRejectsMissingLanguage(t *testing.T) {
	raw := `{
	  "name": "Le Campane di Vespera",
	  "description": "Una citta in rovina affacciata sul sale.",
	  "genre": "fantasy oscuro",
	  "tone": "malinconico",
	  "writing_style": "prosa elegante e inquieta",
	  "prompt_directives": "",
	  "setting": {
	    "world_name": "Vespera",
	    "era": "Eta delle Maree Spezzate",
	    "geography": "Laguna nera",
	    "magic_system": "Campane sommerse",
	    "technology_level": "Rinascimento decadente",
	    "society": "Casate e culti",
	    "rules": ["La magia ha un prezzo"],
	    "factions": ["Casata Valcerra"],
	    "cultures": ["Scavatori"],
	    "dangers": ["Nebbie senzienti"]
	  },
	  "stats_schema": {
	    "vitals": [{"key":"hp","label":"Salute","starting":10}],
	    "attributes": [{"key":"agi","label":"Agilita","starting":3}],
	    "secondary": [{"key":"rep","label":"Reputazione","starting":0}],
	    "currency": {"name":"Corone","starting":5},
	    "has_combat": true
	  }
	}`

	if def := extractStoryJSON(raw); def != nil {
		t.Fatal("extractStoryJSON accepted a story without language")
	}
}
