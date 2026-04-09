package views

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crimsab/oneday/internal/engine"
)

type parsedTalkCommand struct {
	Target  string
	Intent  string
	Message string
}

var talkIntentSet = map[string]bool{
	"ask":      true,
	"probe":    true,
	"bond":     true,
	"bargain":  true,
	"threaten": true,
	"promise":  true,
	"lie":      true,
	"confess":  true,
}

func (m NarrativeModel) talkModeActive() bool {
	return strings.TrimSpace(m.talkTarget) != ""
}

func (m NarrativeModel) activeTalkIntent() string {
	intent := strings.TrimSpace(strings.ToLower(m.talkIntent))
	if intent == "" {
		return "ask"
	}
	return intent
}

func (m NarrativeModel) wrapPlayerAction(action string) string {
	action = strings.TrimSpace(action)
	if action == "" || !m.talkModeActive() {
		return action
	}
	return formatTalkAction(m.talkTarget, m.activeTalkIntent(), action)
}

func formatTalkAction(target, intent, message string) string {
	target = strings.TrimSpace(target)
	message = strings.TrimSpace(message)
	intent = strings.TrimSpace(strings.ToLower(intent))
	if target == "" || message == "" {
		return message
	}
	if intent == "" {
		intent = "ask"
	}
	return fmt.Sprintf("[Talk to %s | intent:%s] %s", target, intent, message)
}

func (m *NarrativeModel) closeTalkMode(status string) {
	if m == nil {
		return
	}
	m.talkTarget = ""
	m.talkIntent = ""
	if strings.TrimSpace(status) != "" {
		m.SetStatusMsg(status)
	}
}

func (m NarrativeModel) showHooks() (NarrativeModel, tea.Cmd) {
	trackerText := engine.FormatStoryTrackerView(m.narrator.World())
	m.showOverlay("Tracker", trackerText)
	return m, nil
}

func (m NarrativeModel) handleTalkCommand(args []string) (NarrativeModel, tea.Cmd) {
	if len(args) == 0 {
		m.showOverlay("Nearby NPCs", m.nearbyNPCOverlayText())
		return m, nil
	}

	if len(args) == 1 {
		switch strings.ToLower(strings.TrimSpace(args[0])) {
		case "off", "exit", "stop":
			m.closeTalkMode("Talk mode closed")
			return m, nil
		}
	}

	parsed := m.parseTalkCommand(args)
	if parsed.Target == "" {
		m.errMsg = "Usage: /talk <npc> [ask|probe|bond|bargain|threaten|promise|lie|confess]"
		return m, nil
	}

	npc, err := m.narrator.DB().GetNPCByName(m.narrator.Story().ID, parsed.Target)
	if err != nil || npc == nil {
		m.errMsg = fmt.Sprintf("No known NPC named %q. Use /talk to see nearby candidates.", parsed.Target)
		return m, nil
	}

	if parsed.Message != "" {
		m.SetStatusMsg(fmt.Sprintf("Talking to %s [%s]", npc.Name, parsed.Intent))
		return m, m.sendRawAction(formatTalkAction(npc.Name, parsed.Intent, parsed.Message))
	}

	m.talkTarget = npc.Name
	m.talkIntent = parsed.Intent
	m.statusMsg = fmt.Sprintf("Talk mode: %s [%s]", npc.Name, m.activeTalkIntent())
	m.statusExpiry = time.Now().Add(3 * time.Second)
	return m, nil
}

func (m NarrativeModel) handleDowntimeCommand(args []string) (NarrativeModel, tea.Cmd) {
	if len(args) == 0 {
		help := `Downtime lets you ask for a lower-intensity scene without railroading the story.

Examples:
  /downtime rest by the fire
  /downtime write a letter home
  /downtime browse the market
  /downtime train with a companion
  /downtime confess something to Lyanna`
		m.showOverlay("Downtime", help)
		return m, nil
	}

	m.statusMsg = "Requesting downtime scene..."
	m.statusExpiry = time.Now().Add(3 * time.Second)
	return m, m.sendAction("[Downtime Scene] " + strings.Join(args, " "))
}

func (m NarrativeModel) nearbyNPCOverlayText() string {
	npcs, err := engine.NearbyNPCs(m.narrator.DB(), m.narrator.Story().ID, m.narrator.Turn(), 6)
	if err != nil || len(npcs) == 0 {
		return "No nearby NPCs stand out right now.\n\nUsage:\n  /talk <npc>\n  /talk <npc> promise\n  /talk <npc> ask <question>\n  /talk off"
	}

	var lines []string
	lines = append(lines, "Use /talk <npc> [intent] to enter a scoped conversation flow.")
	lines = append(lines, "")
	lines = append(lines, "Nearby candidates:")
	for _, npc := range npcs {
		label := npc.Name
		if npc.Role != "" {
			label += " (" + npc.Role + ")"
		}
		lines = append(lines, "  - "+label)
	}
	lines = append(lines, "")
	lines = append(lines, "Intents: ask, probe, bond, bargain, threaten, promise, lie, confess")
	lines = append(lines, "")
	lines = append(lines, "Examples:")
	lines = append(lines, "  /talk "+npcs[0].Name)
	lines = append(lines, "  /talk "+npcs[0].Name+" promise")
	lines = append(lines, "  /talk "+npcs[0].Name+" ask What did you see?")
	lines = append(lines, "  /talk off")
	return strings.Join(lines, "\n")
}

func (m NarrativeModel) parseTalkCommand(args []string) parsedTalkCommand {
	parsed := parsedTalkCommand{Intent: "ask"}
	intent := "ask"
	if len(args) == 0 {
		return parsed
	}

	npcs, _ := engine.NearbyNPCs(m.narrator.DB(), m.narrator.Story().ID, m.narrator.Turn(), 8)
	joined := strings.Join(args, " ")
	bestMatch := ""
	bestCount := 0
	for _, npc := range npcs {
		name := strings.TrimSpace(npc.Name)
		if name == "" {
			continue
		}
		nameParts := strings.Fields(strings.ToLower(name))
		argPrefix := strings.Fields(strings.ToLower(joined))
		if len(argPrefix) < len(nameParts) {
			continue
		}
		match := true
		for i := range nameParts {
			if argPrefix[i] != nameParts[i] {
				match = false
				break
			}
		}
		if match && len(nameParts) > bestCount {
			bestMatch = npc.Name
			bestCount = len(nameParts)
		}
	}

	if bestMatch != "" {
		remaining := args[bestCount:]
		if len(remaining) > 0 {
			candidate := strings.ToLower(strings.TrimSpace(remaining[0]))
			if talkIntentSet[candidate] {
				intent = candidate
				remaining = remaining[1:]
			}
		}
		return parsedTalkCommand{
			Target:  bestMatch,
			Intent:  intent,
			Message: strings.TrimSpace(strings.Join(remaining, " ")),
		}
	}

	last := strings.ToLower(strings.TrimSpace(args[len(args)-1]))
	if talkIntentSet[last] && len(args) > 1 {
		return parsedTalkCommand{
			Target: strings.Join(args[:len(args)-1], " "),
			Intent: last,
		}
	}
	return parsedTalkCommand{
		Target: strings.Join(args, " "),
		Intent: intent,
	}
}

func relationshipAxesSummary(raw string) string {
	type axes struct {
		Trust    int `json:"trust"`
		Fear     int `json:"fear"`
		Debt     int `json:"debt"`
		Respect  int `json:"respect"`
		Intimacy int `json:"intimacy"`
	}

	var decoded axes
	if strings.TrimSpace(raw) == "" || strings.TrimSpace(raw) == "{}" || json.Unmarshal([]byte(raw), &decoded) != nil {
		return "trust 0 · fear 0 · debt 0 · respect 0 · intimacy 0"
	}
	return fmt.Sprintf("trust %d · fear %d · debt %d · respect %d · intimacy %d",
		decoded.Trust, decoded.Fear, decoded.Debt, decoded.Respect, decoded.Intimacy)
}
