package engine

import "strings"

// StateChangesToEventCallouts converts applied state changes into compact
// renderer-facing callouts. The renderer can merge these with any AI-provided
// event_callouts without depending on prose parsing.
func StateChangesToEventCallouts(changes []StateChange) []EventCallout {
	callouts := make([]EventCallout, 0, len(changes))
	seen := map[string]bool{}

	appendCallout := func(kind, title, detail string) {
		title = strings.TrimSpace(title)
		detail = strings.TrimSpace(detail)
		if title == "" && detail == "" {
			return
		}
		key := strings.ToLower(kind + "|" + title + "|" + detail)
		if seen[key] {
			return
		}
		seen[key] = true
		callouts = append(callouts, EventCallout{
			Kind:   kind,
			Title:  title,
			Detail: detail,
		})
	}

	for _, change := range changes {
		desc := strings.TrimSpace(change.Description)
		switch {
		case strings.HasPrefix(desc, "Gained trait: "):
			appendCallout("trait", strings.TrimPrefix(desc, "Gained trait: "), "Trait gained")
		case strings.HasPrefix(desc, "Earned title: "):
			appendCallout("title", strings.TrimPrefix(desc, "Earned title: "), "Title earned")
		case strings.HasPrefix(desc, "Learned new skill: "):
			appendCallout("skill", strings.TrimPrefix(desc, "Learned new skill: "), "New skill learned")
		case strings.HasPrefix(desc, "Gained ") && strings.Contains(desc, " XP in "):
			appendCallout("skill", "Skill progress", desc)
		case strings.HasPrefix(desc, "New NPC encountered: "):
			appendCallout("npc", desc, "New NPC encountered")
		case strings.HasPrefix(desc, "NPC seen again: "):
			appendCallout("npc", strings.TrimPrefix(desc, "NPC seen again: "), "NPC encountered again")
		case strings.Contains(desc, "'s disposition: "):
			appendCallout("relationship", "Disposition changed", desc)
		case strings.Contains(desc, " trust: ") || strings.Contains(desc, " fear: ") || strings.Contains(desc, " debt: ") || strings.Contains(desc, " respect: ") || strings.Contains(desc, " intimacy: "):
			appendCallout("relationship", "Relationship shifted", desc)
		case strings.HasSuffix(desc, "had a new thought (private)"):
			appendCallout("npc", "NPC perspective shifted", desc)
		case strings.HasSuffix(desc, "noted something about protagonist (private)"):
			appendCallout("npc", "NPC note recorded", desc)
		case strings.HasSuffix(desc, "'s desires updated"):
			appendCallout("npc", "NPC desire updated", desc)
		case strings.HasPrefix(desc, "New hook: "):
			appendCallout("hook", strings.TrimPrefix(desc, "New hook: "), "Open thread added")
		case strings.HasPrefix(desc, "Hook updated: "):
			appendCallout("hook", strings.TrimPrefix(desc, "Hook updated: "), "Open thread updated")
		case strings.HasPrefix(desc, "Hook progressed: "):
			appendCallout("hook", strings.TrimPrefix(desc, "Hook progressed: "), "Open thread progressed")
		case strings.HasPrefix(desc, "Hook resolved: "):
			appendCallout("hook", strings.TrimPrefix(desc, "Hook resolved: "), "Open thread resolved")
		case strings.HasPrefix(desc, "World reacts: "):
			appendCallout("reaction", strings.TrimPrefix(desc, "World reacts: "), "Visible consequence")
		case desc == "Combat initiated!":
			appendCallout("combat", "Combat initiated", "")
		case desc == "Crafting session initiated":
			appendCallout("crafting", "Crafting session started", "")
		default:
			switch {
			case change.Field == "location":
				title, _ := change.New.(string)
				appendCallout("location", title, "Location updated")
			case change.Field == "inventory":
				switch {
				case change.Old == nil && change.New != nil:
					appendCallout("item", inventoryCalloutName(change.New), "Item acquired")
				case change.Old != nil && change.New == nil:
					appendCallout("item", inventoryCalloutName(change.Old), "Item removed")
				}
			}
		}
	}

	return callouts
}

func MergeEventCallouts(primary, secondary []EventCallout) []EventCallout {
	if len(primary) == 0 {
		return secondary
	}
	if len(secondary) == 0 {
		return primary
	}

	merged := make([]EventCallout, 0, len(primary)+len(secondary))
	seen := map[string]bool{}
	for _, source := range [][]EventCallout{primary, secondary} {
		for _, callout := range source {
			key := strings.ToLower(callout.Kind + "|" + callout.Title + "|" + callout.Detail)
			if seen[key] {
				continue
			}
			seen[key] = true
			merged = append(merged, callout)
		}
	}
	return merged
}

func inventoryCalloutName(v interface{}) string {
	switch item := v.(type) {
	case string:
		return item
	case map[string]interface{}:
		if name, ok := item["name"].(string); ok {
			return name
		}
	}
	return "Inventory updated"
}
