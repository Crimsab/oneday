package engine

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/crimsab/oneday/internal/storage"
)

// RelationshipAxes captures richer social state than a single disposition scalar.
type RelationshipAxes struct {
	Trust    int `json:"trust"`
	Fear     int `json:"fear"`
	Debt     int `json:"debt"`
	Respect  int `json:"respect"`
	Intimacy int `json:"intimacy"`
}

// StoryHook tracks unresolved narrative threads without turning them into rigid quests.
type StoryHook struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Detail      string `json:"detail,omitempty"`
	Status      string `json:"status,omitempty"` // active, cooling, resolved
	NPCName     string `json:"npc_name,omitempty"`
	TimerTurns  int    `json:"timer_turns,omitempty"`
	SourceTurn  int    `json:"source_turn,omitempty"`
	UpdatedTurn int    `json:"updated_turn,omitempty"`
}

// WorldReaction captures visible rumors, heat, notoriety, and delayed fallout.
type WorldReaction struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Detail      string `json:"detail,omitempty"`
	Status      string `json:"status,omitempty"` // active, cooling, resolved
	SourceTurn  int    `json:"source_turn,omitempty"`
	CreatedTurn int    `json:"created_turn,omitempty"`
}

// TurnDeltaItem is a single player-facing consequence line.
type TurnDeltaItem struct {
	Kind   string `json:"kind,omitempty"`
	Label  string `json:"label"`
	Detail string `json:"detail,omitempty"`
}

// TurnDelta is the structured "What changed this turn?" payload.
type TurnDelta struct {
	Items []TurnDeltaItem `json:"items,omitempty"`
}

func clampRelationshipValue(v int) int {
	if v > 100 {
		return 100
	}
	if v < -100 {
		return -100
	}
	return v
}

func loadRelationshipAxes(npc *storage.NPC) RelationshipAxes {
	if npc == nil || strings.TrimSpace(npc.RelationshipJSON) == "" || strings.TrimSpace(npc.RelationshipJSON) == "{}" {
		return RelationshipAxes{}
	}
	var axes RelationshipAxes
	if err := json.Unmarshal([]byte(npc.RelationshipJSON), &axes); err != nil {
		return RelationshipAxes{}
	}
	axes.Trust = clampRelationshipValue(axes.Trust)
	axes.Fear = clampRelationshipValue(axes.Fear)
	axes.Debt = clampRelationshipValue(axes.Debt)
	axes.Respect = clampRelationshipValue(axes.Respect)
	axes.Intimacy = clampRelationshipValue(axes.Intimacy)
	return axes
}

func storeRelationshipAxes(npc *storage.NPC, axes RelationshipAxes) {
	if npc == nil {
		return
	}
	axes.Trust = clampRelationshipValue(axes.Trust)
	axes.Fear = clampRelationshipValue(axes.Fear)
	axes.Debt = clampRelationshipValue(axes.Debt)
	axes.Respect = clampRelationshipValue(axes.Respect)
	axes.Intimacy = clampRelationshipValue(axes.Intimacy)
	if payload, err := json.Marshal(axes); err == nil {
		npc.RelationshipJSON = string(payload)
	}
}

func loadStoryHooks(world *storage.WorldState) []StoryHook {
	if world == nil || strings.TrimSpace(world.StoryHooksJSON) == "" || strings.TrimSpace(world.StoryHooksJSON) == "[]" {
		return nil
	}
	var hooks []StoryHook
	if err := json.Unmarshal([]byte(world.StoryHooksJSON), &hooks); err != nil {
		return nil
	}
	return normalizeStoryHooks(hooks)
}

func storeStoryHooks(world *storage.WorldState, hooks []StoryHook) {
	if world == nil {
		return
	}
	if payload, err := json.Marshal(normalizeStoryHooks(hooks)); err == nil {
		world.StoryHooksJSON = string(payload)
	}
}

func loadWorldReactions(world *storage.WorldState) []WorldReaction {
	if world == nil || strings.TrimSpace(world.WorldReactionsJSON) == "" || strings.TrimSpace(world.WorldReactionsJSON) == "[]" {
		return nil
	}
	var reactions []WorldReaction
	if err := json.Unmarshal([]byte(world.WorldReactionsJSON), &reactions); err != nil {
		return nil
	}
	return normalizeWorldReactions(reactions)
}

func storeWorldReactions(world *storage.WorldState, reactions []WorldReaction) {
	if world == nil {
		return
	}
	if payload, err := json.Marshal(normalizeWorldReactions(reactions)); err == nil {
		world.WorldReactionsJSON = string(payload)
	}
}

func normalizeStoryHooks(hooks []StoryHook) []StoryHook {
	if len(hooks) == 0 {
		return nil
	}
	out := make([]StoryHook, 0, len(hooks))
	seen := map[string]bool{}
	for _, hook := range hooks {
		hook.Title = strings.TrimSpace(hook.Title)
		hook.Detail = strings.TrimSpace(hook.Detail)
		hook.Kind = strings.TrimSpace(hook.Kind)
		hook.NPCName = strings.TrimSpace(hook.NPCName)
		hook.Status = strings.TrimSpace(hook.Status)
		if hook.ID == "" {
			hook.ID = uuid.NewString()
		}
		if hook.Status == "" {
			hook.Status = "active"
		}
		if hook.Title == "" {
			continue
		}
		key := strings.ToLower(hook.ID + "|" + hook.Title + "|" + hook.Status)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, hook)
	}
	return out
}

func normalizeWorldReactions(reactions []WorldReaction) []WorldReaction {
	if len(reactions) == 0 {
		return nil
	}
	out := make([]WorldReaction, 0, len(reactions))
	seen := map[string]bool{}
	for _, reaction := range reactions {
		reaction.Kind = strings.TrimSpace(reaction.Kind)
		reaction.Title = strings.TrimSpace(reaction.Title)
		reaction.Detail = strings.TrimSpace(reaction.Detail)
		reaction.Status = strings.TrimSpace(reaction.Status)
		if reaction.ID == "" {
			reaction.ID = uuid.NewString()
		}
		if reaction.Status == "" {
			reaction.Status = "active"
		}
		if reaction.Title == "" {
			continue
		}
		key := strings.ToLower(reaction.ID + "|" + reaction.Title + "|" + reaction.Status)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, reaction)
	}
	return out
}

func activeStoryHooks(hooks []StoryHook) []StoryHook {
	if len(hooks) == 0 {
		return nil
	}
	active := make([]StoryHook, 0, len(hooks))
	for _, hook := range normalizeStoryHooks(hooks) {
		if strings.EqualFold(hook.Status, "resolved") {
			continue
		}
		active = append(active, hook)
	}
	if len(active) == 0 {
		return nil
	}
	return active
}

func visibleWorldReactions(reactions []WorldReaction) []WorldReaction {
	if len(reactions) == 0 {
		return nil
	}
	visible := make([]WorldReaction, 0, len(reactions))
	for _, reaction := range normalizeWorldReactions(reactions) {
		if strings.EqualFold(reaction.Status, "resolved") {
			continue
		}
		visible = append(visible, reaction)
	}
	if len(visible) == 0 {
		return nil
	}
	return visible
}

func mergeTurnDelta(engineDelta, aiDelta *TurnDelta) *TurnDelta {
	switch {
	case engineDelta == nil && aiDelta == nil:
		return nil
	case engineDelta == nil:
		return normalizeTurnDelta(aiDelta)
	case aiDelta == nil:
		return normalizeTurnDelta(engineDelta)
	}

	merged := &TurnDelta{
		Items: append([]TurnDeltaItem{}, engineDelta.Items...),
	}
	merged.Items = append(merged.Items, aiDelta.Items...)
	return normalizeTurnDelta(merged)
}

func normalizeTurnDelta(delta *TurnDelta) *TurnDelta {
	if delta == nil || len(delta.Items) == 0 {
		return nil
	}
	out := &TurnDelta{Items: make([]TurnDeltaItem, 0, len(delta.Items))}
	seen := map[string]bool{}
	for _, item := range delta.Items {
		item.Kind = strings.TrimSpace(item.Kind)
		item.Label = strings.TrimSpace(item.Label)
		item.Detail = strings.TrimSpace(item.Detail)
		if item.Label == "" && item.Detail == "" {
			continue
		}
		key := strings.ToLower(item.Kind + "|" + item.Label + "|" + item.Detail)
		if seen[key] {
			continue
		}
		seen[key] = true
		out.Items = append(out.Items, item)
	}
	if len(out.Items) == 0 {
		return nil
	}
	return out
}

func buildTurnDelta(changes []StateChange) *TurnDelta {
	if len(changes) == 0 {
		return nil
	}
	items := make([]TurnDeltaItem, 0, len(changes))
	for _, change := range changes {
		label := strings.TrimSpace(change.Description)
		kind := "world"
		switch {
		case strings.HasPrefix(change.Field, "vitals.") || strings.HasPrefix(change.Field, "attributes.") || strings.HasPrefix(change.Field, "secondary.") || change.Field == "currency":
			kind = "stat"
		case change.Field == "inventory":
			if change.Old == nil && change.New != nil {
				kind = "inventory_gain"
			} else if change.Old != nil && change.New == nil {
				kind = "inventory_loss"
			} else {
				kind = "inventory"
			}
		case strings.HasPrefix(change.Field, "npc.") || strings.HasPrefix(change.Field, "relationship."):
			kind = "relationship"
		case strings.HasPrefix(change.Field, "hook."):
			kind = "hook"
		case strings.HasPrefix(change.Field, "reaction."):
			kind = "reaction"
		case strings.Contains(change.Field, "setting_") || strings.HasPrefix(change.Field, "world_"):
			kind = "lore"
		case change.Field == "location":
			kind = "world"
		}

		if label == "" {
			if change.Field == "location" {
				label = "Location changed"
			} else {
				label = strings.ReplaceAll(change.Field, ".", " ")
			}
		}

		detail := ""
		if kind == "stat" && change.Old != nil && change.New != nil {
			detail = strings.TrimSpace(formatValueTransition(change.Old, change.New))
		}
		items = append(items, TurnDeltaItem{
			Kind:   kind,
			Label:  label,
			Detail: detail,
		})
	}
	return normalizeTurnDelta(&TurnDelta{Items: items})
}

func formatValueTransition(oldValue, newValue interface{}) string {
	oldText := strings.TrimSpace(toDisplayString(oldValue))
	newText := strings.TrimSpace(toDisplayString(newValue))
	switch {
	case oldText == "" && newText == "":
		return ""
	case oldText == "":
		return newText
	case newText == "":
		return oldText
	default:
		return oldText + " -> " + newText
	}
}

func toDisplayString(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case int:
		return jsonNumberString(v)
	case int64:
		return jsonNumberString(v)
	case float64:
		return jsonNumberString(v)
	case time.Time:
		return v.Format(time.RFC3339)
	default:
		payload, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(payload)
	}
}

func jsonNumberString[T ~int | ~int64 | ~float64](value T) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(payload)
}

// FormatStoryTrackerView renders the persistent hooks and world-reaction feed for overlays.
func FormatStoryTrackerView(world *storage.WorldState) string {
	if world == nil {
		return "No tracker data available."
	}

	hooks := activeStoryHooks(loadStoryHooks(world))
	reactions := visibleWorldReactions(loadWorldReactions(world))
	if len(hooks) == 0 && len(reactions) == 0 {
		return "No open hooks or active world reactions yet."
	}

	var sb strings.Builder
	if len(hooks) > 0 {
		sb.WriteString("## Open Hooks\n")
		for _, hook := range hooks {
			sb.WriteString("- " + hook.Title)
			if hook.Kind != "" {
				sb.WriteString(fmt.Sprintf(" [%s]", hook.Kind))
			}
			if hook.NPCName != "" {
				sb.WriteString(fmt.Sprintf(" {%s}", hook.NPCName))
			}
			if hook.TimerTurns > 0 {
				sb.WriteString(fmt.Sprintf(" (timer %d)", hook.TimerTurns))
			}
			sb.WriteString("\n")
			if hook.Detail != "" {
				sb.WriteString("  " + hook.Detail + "\n")
			}
		}
	}

	if len(reactions) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("## World Reaction Feed\n")
		for _, reaction := range reactions {
			sb.WriteString("- " + reaction.Title)
			if reaction.Kind != "" {
				sb.WriteString(fmt.Sprintf(" [%s]", reaction.Kind))
			}
			sb.WriteString("\n")
			if reaction.Detail != "" {
				sb.WriteString("  " + reaction.Detail + "\n")
			}
		}
	}

	return strings.TrimSpace(sb.String())
}
