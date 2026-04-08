package ai

// NewJSONObjectResponseFormat asks a compatible provider to emit a JSON object.
func NewJSONObjectResponseFormat() *ResponseFormat {
	return &ResponseFormat{Type: "json_object"}
}

// NewJSONSchemaResponseFormat asks a compatible provider to emit JSON matching
// the supplied schema.
func NewJSONSchemaResponseFormat(name string, schema map[string]any) *ResponseFormat {
	return &ResponseFormat{
		Type: "json_schema",
		JSONSchema: &JSONSchemaConfig{
			Name:   name,
			Strict: true,
			Schema: schema,
		},
	}
}

// NarrativeResponseFormat is the runtime schema for gameplay/combat narrative turns.
func NarrativeResponseFormat() *ResponseFormat {
	return NewJSONSchemaResponseFormat("oneday_narrative_response", narrativeSchema())
}

// StoryDefinitionResponseFormat is the final story-definition schema used by
// the benchmark and any future dedicated finalization call.
func StoryDefinitionResponseFormat() *ResponseFormat {
	return NewJSONSchemaResponseFormat("oneday_story_definition", storyDefinitionSchema())
}

// CharacterCreationResponseFormat is the final character payload schema.
func CharacterCreationResponseFormat() *ResponseFormat {
	return NewJSONSchemaResponseFormat("oneday_character_creation", characterCreationSchema())
}

// ChapterSummaryResponseFormat is the schema for chapter title+summary generation.
func ChapterSummaryResponseFormat() *ResponseFormat {
	return NewJSONSchemaResponseFormat("oneday_chapter_summary", chapterSummarySchema())
}

// NarratorMetaResponseFormat is the schema for /narrator meta responses.
func NarratorMetaResponseFormat() *ResponseFormat {
	return NewJSONSchemaResponseFormat("oneday_narrator_meta", narratorMetaSchema())
}

// CraftingResponseFormat is the schema for crafting evaluations.
func CraftingResponseFormat() *ResponseFormat {
	return NewJSONSchemaResponseFormat("oneday_crafting_response", craftingSchema())
}

// CombatDefeatResponseFormat is the schema for defeat resolution.
func CombatDefeatResponseFormat() *ResponseFormat {
	return NewJSONSchemaResponseFormat("oneday_combat_defeat", combatDefeatSchema())
}

func storyDefinitionSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"name", "description", "genre", "tone", "setting", "stats_schema"},
		"properties": map[string]any{
			"name":        stringSchema(),
			"description": stringSchema(),
			"genre":       stringSchema(),
			"tone":        stringSchema(),
			"setting": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required": []string{
					"world_name", "era", "geography", "magic_system", "technology_level",
					"society", "rules", "factions", "cultures", "dangers",
				},
				"properties": map[string]any{
					"world_name":       stringSchema(),
					"era":              stringSchema(),
					"geography":        stringSchema(),
					"magic_system":     stringSchema(),
					"technology_level": stringSchema(),
					"society":          stringSchema(),
					"rules":            stringArraySchema(),
					"factions":         stringArraySchema(),
					"cultures":         stringArraySchema(),
					"dangers":          stringArraySchema(),
					"tone_guidelines":  stringSchema(),
				},
			},
			"stats_schema": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"vitals", "attributes", "secondary", "has_combat"},
				"properties": map[string]any{
					"vitals":     statDefArraySchema(),
					"attributes": statDefArraySchema(),
					"secondary":  statDefArraySchema(),
					"currency": map[string]any{
						"type":                 "object",
						"additionalProperties": false,
						"required":             []string{"name", "starting"},
						"properties": map[string]any{
							"name":     stringSchema(),
							"starting": integerSchema(),
						},
					},
					"has_combat": boolSchema(),
				},
			},
		},
	}
}

func characterCreationSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"name", "background"},
		"properties": map[string]any{
			"name":       stringSchema(),
			"background": stringSchema(),
		},
	}
}

func chapterSummarySchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"title", "summary"},
		"properties": map[string]any{
			"title":   stringSchema(),
			"summary": stringSchema(),
		},
	}
}

func narratorMetaSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"message"},
		"properties": map[string]any{
			"message": stringSchema(),
			"state_changes": map[string]any{
				"type":                 "object",
				"additionalProperties": true,
			},
		},
	}
}

func craftingSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"feasible", "narrative", "choices"},
		"properties": map[string]any{
			"feasible":  boolSchema(),
			"narrative": stringSchema(),
			"item": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"name", "description", "effect", "materials"},
				"properties": map[string]any{
					"name":        stringSchema(),
					"description": stringSchema(),
					"effect":      stringSchema(),
					"materials":   stringArraySchema(),
					"crafted_at":  stringSchema(),
				},
			},
			"missing":      stringArraySchema(),
			"alternatives": stringArraySchema(),
			"choices":      choiceArraySchema(),
		},
	}
}

func combatDefeatSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"outcome", "narrative"},
		"properties": map[string]any{
			"outcome": map[string]any{
				"type": "string",
				"enum": []string{"death", "capture", "rescue", "retreat", "unconscious"},
			},
			"narrative":     stringSchema(),
			"state_changes": map[string]any{"type": "object", "additionalProperties": true},
		},
	}
}

func narrativeSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"narrative", "choices"},
		"properties": map[string]any{
			"narrative": stringSchema(),
			"choices":   choiceArraySchema(),
			"mood":      stringSchema(),
			"location":  stringSchema(),
			"scene_type": map[string]any{
				"type": "string",
			},
			"dialogue_blocks": map[string]any{
				"type":  "array",
				"items": dialogueBlockSchema(),
			},
			"entities_mentioned": map[string]any{
				"type":  "array",
				"items": entityMentionSchema(),
			},
			"event_callouts": map[string]any{
				"type":  "array",
				"items": eventCalloutSchema(),
			},
			"state_changes": map[string]any{
				"type":                 "object",
				"additionalProperties": true,
			},
			"challenges": map[string]any{
				"type":  "array",
				"items": challengeSchema(),
			},
			"achievement_earned": nullableObjectSchema(map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"name", "description", "rarity", "category", "context"},
				"properties": map[string]any{
					"name":        stringSchema(),
					"description": stringSchema(),
					"rarity":      stringSchema(),
					"category":    stringSchema(),
					"context":     stringSchema(),
				},
			}),
			"chapter_end":   boolSchema(),
			"chapter_title": stringSchema(),
			"combat_start": nullableObjectSchema(map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"name", "hp", "max_hp", "attack", "defense"},
				"properties": map[string]any{
					"name":        stringSchema(),
					"description": stringSchema(),
					"hp":          integerSchema(),
					"max_hp":      integerSchema(),
					"attack":      integerSchema(),
					"defense":     integerSchema(),
					"behavior":    stringSchema(),
				},
			}),
			"ascii_art": stringSchema(),
		},
	}
}

func challengeSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"type"},
		"properties": map[string]any{
			"type": map[string]any{
				"type": "string",
				"enum": []string{"stat_check", "dice_roll", "item_check", "skill_check", "relationship_check", "mini_game"},
			},
			"difficulty":  integerSchema(),
			"stat":        stringSchema(),
			"item":        stringSchema(),
			"skill":       stringSchema(),
			"skill_level": integerSchema(),
			"npc_name":    stringSchema(),
			"disposition": integerSchema(),
			"mini_game":   stringSchema(),
			"modifiers": map[string]any{
				"type":  "array",
				"items": modifierSchema(),
			},
			"riddle":     stringSchema(),
			"answer":     stringSchema(),
			"sequence":   stringArraySchema(),
			"time_limit": numberSchema(),
		},
	}
}

func choiceArraySchema() map[string]any {
	return map[string]any{
		"type":  "array",
		"items": choiceSchema(),
	}
}

func choiceSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"id", "text"},
		"properties": map[string]any{
			"id":            integerSchema(),
			"text":          stringSchema(),
			"mood":          stringSchema(),
			"intent":        stringSchema(),
			"risk":          stringSchema(),
			"scope":         stringSchema(),
			"certainty":     stringSchema(),
			"related_stats": stringArraySchema(),
		},
	}
}

func dialogueBlockSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"speaker", "text"},
		"properties": map[string]any{
			"speaker": stringSchema(),
			"role":    stringSchema(),
			"text":    stringSchema(),
		},
	}
}

func entityMentionSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"name"},
		"properties": map[string]any{
			"name": stringSchema(),
			"type": stringSchema(),
		},
	}
}

func eventCalloutSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"title"},
		"properties": map[string]any{
			"kind":   stringSchema(),
			"title":  stringSchema(),
			"detail": stringSchema(),
		},
	}
}

func modifierSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"source", "value"},
		"properties": map[string]any{
			"source": stringSchema(),
			"value":  integerSchema(),
		},
	}
}

func statDefArraySchema() map[string]any {
	return map[string]any{
		"type": "array",
		"items": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"required":             []string{"key", "label"},
			"properties": map[string]any{
				"key":      stringSchema(),
				"label":    stringSchema(),
				"starting": integerSchema(),
			},
		},
	}
}

func stringSchema() map[string]any {
	return map[string]any{"type": "string"}
}

func stringArraySchema() map[string]any {
	return map[string]any{
		"type":  "array",
		"items": stringSchema(),
	}
}

func integerSchema() map[string]any {
	return map[string]any{"type": "integer"}
}

func numberSchema() map[string]any {
	return map[string]any{"type": "number"}
}

func boolSchema() map[string]any {
	return map[string]any{"type": "boolean"}
}

func nullableObjectSchema(schema map[string]any) map[string]any {
	return map[string]any{
		"anyOf": []any{
			schema,
			map[string]any{"type": "null"},
		},
	}
}
