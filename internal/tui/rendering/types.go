package rendering

import "github.com/crimsab/oneday/internal/engine"

// KnownEntity is a trusted entity sourced from persisted state.
type KnownEntity struct {
	Name string
	Kind string
}

// NarrativeInput is the renderer-facing contract for the narrative view.
type NarrativeInput struct {
	Narrative         string
	DialogueBlocks    []engine.DialogueBlock
	EntitiesMentioned []engine.EntityMention
	EventCallouts     []engine.EventCallout
	KnownEntities     []KnownEntity
}
