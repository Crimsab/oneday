package views

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/storage"
	"github.com/crimsab/oneday/internal/tui/components"
	"github.com/crimsab/oneday/internal/tui/theme"
)

// narrativeResponseMsg carries a parsed AI narrative response.
type narrativeResponseMsg struct {
	response *engine.NarrativeResponse
	err      error
}

// narratorMetaResponseMsg carries a /narrator command response.
type narratorMetaResponseMsg struct {
	message string
	err     error
}

// clearStatusMsg is sent to clear the temporary status message.
type clearStatusMsg struct{}

// SaveCompleteMsg is sent when a manual save finishes.
type SaveCompleteMsg struct {
	Name string
	Err  error
}

// QuitToMenuMsg signals the app to return to the main menu.
type QuitToMenuMsg struct{}

// NarrativeModel is the core gameplay view.
type NarrativeModel struct {
	narrator     *engine.Narrator
	viewport     viewport.Model
	typewriter   components.TypewriterModel
	statusBar    components.StatusBarModel
	choices      components.ChoiceListModel
	overlay      components.OverlayModel
	input        textarea.Model
	history      strings.Builder // full narrative text accumulated so far
	waiting      bool            // waiting for AI response
	errMsg       string
	statusMsg    string    // temporary status message (e.g. "Autosaved")
	statusExpiry time.Time // when to clear the status message
	width        int
	height       int
	inputFocus   bool // true = free input active, false = choice list active
}

// NewNarrativeModel creates the narrative view.
func NewNarrativeModel(narrator *engine.Narrator, typewriterSpeed int) NarrativeModel {
	ta := textarea.New()
	ta.Placeholder = "Type a free action or press 1-4 to choose..."
	ta.CharLimit = 500
	ta.SetHeight(2)

	vp := viewport.New(80, 20)

	return NarrativeModel{
		narrator:   narrator,
		viewport:   vp,
		typewriter: components.NewTypewriter(typewriterSpeed),
		statusBar:  components.NewStatusBar(),
		choices:    components.NewChoiceList(),
		overlay:    components.NewOverlay(),
		input:      ta,
		inputFocus: false, // start on choice list
	}
}

func (m NarrativeModel) Init() tea.Cmd {
	return textarea.Blink
}

// StartNarration kicks off the first AI turn.
func (m *NarrativeModel) StartNarration() tea.Cmd {
	m.waiting = true
	return func() tea.Msg {
		resp, err := m.narrator.StartNarration(context.Background())
		return narrativeResponseMsg{response: resp, err: err}
	}
}

func (m NarrativeModel) Update(msg tea.Msg) (NarrativeModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.overlay.SetSize(msg.Width, msg.Height)
		m.relayout()
		return m, nil

	case narrativeResponseMsg:
		m.waiting = false
		if msg.err != nil {
			m.errMsg = fmt.Sprintf("AI Error: %v", msg.err)
			return m, nil
		}
		m.errMsg = ""
		nr := msg.response

		// Update status bar with current vitals and AI metadata
		m.updateStatusBar()

		// Set choices
		choiceItems := make([]components.ChoiceItem, len(nr.Choices))
		for i, c := range nr.Choices {
			choiceItems[i] = components.ChoiceItem{ID: c.ID, Text: c.Text}
		}
		m.choices.SetChoices(choiceItems)

		// Append rendered narrative to history and start typewriter from beginning
		rendered := components.RenderMarkdown(nr.Narrative)
		m.history.WriteString(rendered + "\n")
		cmd := m.typewriter.SetText(m.history.String())
		cmds = append(cmds, cmd)

		// Focus on choices if available, otherwise free input
		if len(nr.Choices) > 0 {
			m.inputFocus = false
			m.input.Blur()
		} else {
			m.inputFocus = true
			m.input.Focus()
		}

		// Fire autosave cmd if it's time
		if autosaveCmd := m.maybeAutosaveCmd(); autosaveCmd != nil {
			cmds = append(cmds, autosaveCmd)
		}

		return m, tea.Batch(cmds...)

	case narratorMetaResponseMsg:
		m.waiting = false
		if msg.err != nil {
			m.errMsg = fmt.Sprintf("Narrator Error: %v", msg.err)
			return m, nil
		}
		m.errMsg = ""
		// Display narrator response styled distinctly (prefixed with [Game Master]).
		rendered := components.RenderMarkdown("\n**[Game Master]** " + msg.message + "\n")
		m.history.WriteString(rendered + "\n")
		cmd := m.typewriter.SetText(m.history.String())
		// Restore input focus (narrator command does not change choices).
		m.inputFocus = true
		m.input.Focus()
		return m, cmd

	case engine.AutosaveCompleteMsg:
		if msg.Err == nil {
			m.statusMsg = "Autosaved"
		} else {
			m.statusMsg = "Autosave failed"
		}
		m.statusExpiry = time.Now().Add(2 * time.Second)
		return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return clearStatusMsg{}
		})

	case SaveCompleteMsg:
		if msg.Err == nil {
			m.statusMsg = fmt.Sprintf("Saved: %s", msg.Name)
		} else {
			m.errMsg = fmt.Sprintf("Save failed: %v", msg.Err)
		}
		m.statusExpiry = time.Now().Add(3 * time.Second)
		return m, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
			return clearStatusMsg{}
		})

	case clearStatusMsg:
		if time.Now().After(m.statusExpiry) {
			m.statusMsg = ""
		}
		return m, nil

	case components.TypewriterDoneMsg:
		// Typewriter finished — update viewport content and scroll to bottom
		m.viewport.SetContent(m.typewriter.View())
		m.viewport.GotoBottom()
		return m, nil

	case components.ChoiceSelectedMsg:
		if m.waiting {
			return m, nil
		}
		return m, m.sendAction(fmt.Sprintf("[Choice %d] %s", msg.ID, msg.Text))

	case components.OverlayDismissedMsg:
		// Restore input focus after overlay closes
		if m.choices.HasChoices() {
			m.inputFocus = false
			m.input.Blur()
		} else {
			m.inputFocus = true
			m.input.Focus()
		}
		return m, nil

	case tea.KeyMsg:
		// If overlay is visible, route all key events to it first.
		if m.overlay.Visible() {
			var cmd tea.Cmd
			m.overlay, cmd = m.overlay.Update(msg)
			return m, cmd
		}

		if m.waiting {
			return m, nil
		}

		switch msg.String() {
		case "tab":
			// Toggle between choice list and free input
			m.inputFocus = !m.inputFocus
			if m.inputFocus {
				m.input.Focus()
			} else {
				m.input.Blur()
			}
			return m, nil

		case "enter":
			if m.inputFocus {
				text := strings.TrimSpace(m.input.Value())
				if text == "" {
					return m, nil
				}
				m.input.Reset()
				if engine.IsCommand(text) {
					cmd := engine.ParseCommand(text)
					return m.handleCommand(cmd)
				}
				return m, m.sendAction(text)
			}
			// If on choices, let choice list handle it below

		case "esc":
			// Handled by parent app for menu return
			return m, nil
		}

		// Route key to active component
		if m.inputFocus {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		} else {
			var cmd tea.Cmd
			m.choices, cmd = m.choices.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Always update typewriter
	var twCmd tea.Cmd
	m.typewriter, twCmd = m.typewriter.Update(msg)
	if twCmd != nil {
		cmds = append(cmds, twCmd)
		// Update viewport content as typewriter progresses
		m.viewport.SetContent(m.typewriter.View())
		m.viewport.GotoBottom()
	}

	// Update viewport scrolling
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

// handleCommand dispatches a parsed command to the appropriate handler.
func (m NarrativeModel) handleCommand(cmd *engine.Command) (NarrativeModel, tea.Cmd) {
	switch cmd.Name {
	case "inventory":
		return m.showInventory()
	case "stats":
		return m.showStats()
	case "save":
		return m.doSave(cmd.Args)
	case "load":
		return m.doLoad()
	case "help":
		return m.showHelp()
	case "quit":
		return m.doQuit()
	case "narrator":
		if len(cmd.Args) == 0 {
			m.errMsg = "Usage: /n <message to the game master>"
			return m, nil
		}
		input := strings.Join(cmd.Args, " ")
		return m.sendNarratorCommand(input)
	case "journal":
		return m.showJournal()
	case "map":
		return m.showMap()
	case "achievements":
		return m.showAchievements()
	default:
		if len(cmd.Args) > 0 {
			m.errMsg = fmt.Sprintf("Unknown command: /%s. Type /help for available commands.", cmd.Args[0])
		} else {
			m.errMsg = "Unknown command. Type /help for available commands."
		}
		return m, nil
	}
}

// showOverlay is a helper to show the overlay with title and content.
func (m *NarrativeModel) showOverlay(title, content string) {
	m.overlay.SetSize(m.width, m.height)
	m.overlay.Show(title, content)
	// Blur inputs so overlay handles all keys
	m.input.Blur()
}

// sendNarratorCommand sends a /narrator meta-command to the AI game master.
// Does not increment the turn counter.
func (m NarrativeModel) sendNarratorCommand(input string) (NarrativeModel, tea.Cmd) {
	m.waiting = true
	m.statusMsg = "Speaking to the narrator..."
	m.statusExpiry = time.Now().Add(30 * time.Second)
	narrator := m.narrator
	return m, func() tea.Msg {
		resp, err := narrator.ExecuteNarratorCommand(context.Background(), input)
		if err != nil {
			return narratorMetaResponseMsg{err: err}
		}
		return narratorMetaResponseMsg{message: resp.Message}
	}
}

// showJournal displays the chapter journal overlay.
func (m NarrativeModel) showJournal() (NarrativeModel, tea.Cmd) {
	world := m.narrator.World()
	storyID := m.narrator.Story().ID
	currentChapter := 1
	currentTurn := m.narrator.Turn()
	if world != nil {
		currentChapter = world.CurrentChapter
		currentTurn = world.CurrentTurn
	}
	journalText := engine.FormatJournalView(m.narrator.DB(), storyID, currentChapter, currentTurn)
	m.showOverlay("Journal", journalText)
	return m, nil
}

// showMap displays the discovered world locations overlay.
func (m NarrativeModel) showMap() (NarrativeModel, tea.Cmd) {
	mapText := engine.FormatMapView(m.narrator.World())
	m.showOverlay("World Map", mapText)
	return m, nil
}

// showAchievements displays earned achievements overlay.
func (m NarrativeModel) showAchievements() (NarrativeModel, tea.Cmd) {
	storyID := m.narrator.Story().ID
	achText := engine.FormatAchievementsView(m.narrator.DB(), storyID)
	m.showOverlay("Achievements", achText)
	return m, nil
}

// showHelp displays the help overlay.
func (m NarrativeModel) showHelp() (NarrativeModel, tea.Cmd) {
	helpText := `Available Commands:

  /inventory    (/i)   Show your inventory
  /stats        (/s)   Show character sheet
  /map          (/m)   Show discovered world map
  /journal      (/j)   Show chapter journal
  /achievements (/a)   Show earned achievements
  /narrator     (/n)   Speak to the game master
  /save [name]         Save your game
  /load                Load a saved game
  /help         (/h)   Show this help
  /quit         (/q)   Save and quit to menu

Narrator examples:
  /n Add a secret underground city
  /n Make Lyanna secretly jealous
  /n What factions exist in this world?
  /n I want the next area to be a haunted forest`

	m.showOverlay("Help", helpText)
	return m, nil
}

// itemTypeIcon returns a unicode icon for an item type.
func itemTypeIcon(itemType string) string {
	switch strings.ToLower(itemType) {
	case "weapon":
		return "⚔"
	case "armor":
		return "◇"
	case "tool":
		return "⚒"
	case "consumable":
		return "◈"
	case "quest", "key_item":
		return "◆"
	default:
		return "•"
	}
}

// inventoryItemInfo extracts name, type icon, slot, and description from an item (string or map).
type inventoryItemInfo struct {
	name        string
	icon        string
	slot        string
	description string
	rarity      string
	weight      float64
	hasWeight   bool
}

func parseInventoryItem(item interface{}) inventoryItemInfo {
	switch v := item.(type) {
	case string:
		return inventoryItemInfo{name: v, icon: "•"}
	case map[string]interface{}:
		info := inventoryItemInfo{}
		if n, ok := v["name"].(string); ok {
			info.name = n
		}
		itemType := ""
		if t, ok := v["type"].(string); ok {
			itemType = t
		}
		info.icon = itemTypeIcon(itemType)
		if s, ok := v["slot"].(string); ok {
			info.slot = s
		}
		if d, ok := v["description"].(string); ok {
			info.description = d
		}
		if r, ok := v["rarity"].(string); ok {
			info.rarity = r
		}
		if w, ok := v["weight"]; ok {
			info.weight = toFloat64(w)
			info.hasWeight = true
		}
		return info
	}
	return inventoryItemInfo{name: fmt.Sprintf("%v", item), icon: "•"}
}

// toFloat64 converts interface{} to float64.
func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	}
	return 0
}

// showInventory displays the character's inventory as an overlay.
func (m NarrativeModel) showInventory() (NarrativeModel, tea.Cmd) {
	char := m.narrator.Character()
	var sb strings.Builder

	sb.WriteString("\n")

	// Parse inventory JSON — could be a list or a map with sections.
	var inventoryRaw interface{}
	if char.InventoryJSON != "" && char.InventoryJSON != "null" {
		_ = json.Unmarshal([]byte(char.InventoryJSON), &inventoryRaw)
	}

	// Parse stats for currency, equipped items, and inventory within stats.
	var stats map[string]interface{}
	if char.StatsJSON != "" {
		_ = json.Unmarshal([]byte(char.StatsJSON), &stats)
	}

	// Check if inventory is in stats (AI may store there).
	if inventoryRaw == nil && stats != nil {
		if inv, ok := stats["inventory"]; ok {
			inventoryRaw = inv
		}
	}

	// Try to extract inventory sections.
	var backpackItems []inventoryItemInfo
	var equippedItems []inventoryItemInfo
	var questItems []inventoryItemInfo

	switch inv := inventoryRaw.(type) {
	case []interface{}:
		for _, item := range inv {
			info := parseInventoryItem(item)
			if m2, ok := item.(map[string]interface{}); ok {
				if eq, _ := m2["equipped"].(bool); eq {
					equippedItems = append(equippedItems, info)
					continue
				}
				if qt, _ := m2["quest"].(bool); qt {
					questItems = append(questItems, info)
					continue
				}
				if info.icon == "◆" {
					questItems = append(questItems, info)
					continue
				}
			}
			backpackItems = append(backpackItems, info)
		}
	case map[string]interface{}:
		if bp, ok := inv["backpack"].([]interface{}); ok {
			for _, item := range bp {
				backpackItems = append(backpackItems, parseInventoryItem(item))
			}
		}
		if eq, ok := inv["equipped"].([]interface{}); ok {
			for _, item := range eq {
				equippedItems = append(equippedItems, parseInventoryItem(item))
			}
		}
		if qt, ok := inv["quest"].([]interface{}); ok {
			for _, item := range qt {
				questItems = append(questItems, parseInventoryItem(item))
			}
		}
	}

	// Currency setup.
	currency := 0
	currencyName := "Gold"
	if stats != nil {
		if c, ok := stats["currency"]; ok {
			currency = toInt(c)
		}
	}
	// Try to get currency name from story stats schema.
	story := m.narrator.Story()
	if story != nil && story.StatsSchemaJSON != "" {
		var schema map[string]interface{}
		if err := json.Unmarshal([]byte(story.StatsSchemaJSON), &schema); err == nil {
			if cur, ok := schema["currency"].(map[string]interface{}); ok {
				if name, ok := cur["name"].(string); ok && name != "" {
					currencyName = name
				}
			}
		}
	}

	// Completely empty inventory: show friendly message.
	totalItems := len(backpackItems) + len(equippedItems) + len(questItems)
	if totalItems == 0 && currency == 0 {
		sb.WriteString(theme.MutedText.Render("  You carry nothing. The journey begins light."))
		sb.WriteString("\n")
		m.showOverlay("Inventory", sb.String())
		return m, nil
	}

	// Equipped section.
	if len(equippedItems) > 0 {
		sb.WriteString(theme.Title.Render("Equipped") + "\n")
		for _, info := range equippedItems {
			if info.slot != "" {
				slotLabel := strings.ReplaceAll(strings.Title(strings.ReplaceAll(info.slot, "_", " ")), " ", " ")
				sb.WriteString(fmt.Sprintf("  %-12s %s %s\n", slotLabel+":", info.icon, info.name))
			} else {
				sb.WriteString(fmt.Sprintf("  %s %s\n", info.icon, info.name))
			}
			if info.description != "" {
				sb.WriteString(theme.MutedText.Render(fmt.Sprintf("      %s", info.description)) + "\n")
			}
		}
		sb.WriteString("\n")
	}

	// Backpack section.
	if len(backpackItems) > 0 {
		sb.WriteString(theme.Title.Render(fmt.Sprintf("Backpack (%d items)", len(backpackItems))) + "\n")
		// Calculate total weight if any items have it.
		totalWeight := 0.0
		hasWeight := false
		for _, info := range backpackItems {
			if info.hasWeight {
				totalWeight += info.weight
				hasWeight = true
			}
		}
		for _, info := range backpackItems {
			sb.WriteString(fmt.Sprintf("  %s %s", info.icon, info.name))
			if info.rarity != "" {
				sb.WriteString(" " + theme.MutedText.Render(fmt.Sprintf("(%s)", info.rarity)))
			}
			sb.WriteString("\n")
			if info.description != "" {
				sb.WriteString(theme.MutedText.Render(fmt.Sprintf("      %s", info.description)) + "\n")
			}
		}
		if hasWeight {
			sb.WriteString(theme.MutedText.Render(fmt.Sprintf("  Weight: %.0f", totalWeight)) + "\n")
		}
		sb.WriteString("\n")
	}

	// Quest items section.
	if len(questItems) > 0 {
		sb.WriteString(theme.Title.Render("Quest Items") + "\n")
		for _, info := range questItems {
			sb.WriteString(fmt.Sprintf("  ◆ %s\n", info.name))
			if info.description != "" {
				sb.WriteString(theme.MutedText.Render(fmt.Sprintf("      %s", info.description)) + "\n")
			}
		}
		sb.WriteString("\n")
	}

	// Currency.
	sb.WriteString(theme.Title.Render("Currency") + "\n")
	sb.WriteString(fmt.Sprintf("  %d %s\n", currency, currencyName))

	m.showOverlay("Inventory", sb.String())
	return m, nil
}

// renderBar renders a progress bar of the given width using block characters.
// filled = █, empty = ░. current and max define the fill ratio.
func renderBar(current, max, width int) string {
	if max <= 0 || width <= 0 {
		return strings.Repeat("░", width)
	}
	filled := current * width / max
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

// dispositionLabel returns a word label for an NPC disposition value.
func dispositionLabel(d int) string {
	switch {
	case d >= 50:
		return "Allied"
	case d >= 15:
		return "Friendly"
	case d >= -14:
		return "Neutral"
	case d >= -49:
		return "Unfriendly"
	default:
		return "Hostile"
	}
}

// showStats displays the full character sheet as an overlay.
func (m NarrativeModel) showStats() (NarrativeModel, tea.Cmd) {
	char := m.narrator.Character()
	story := m.narrator.Story()
	var sb strings.Builder

	sb.WriteString("\n")

	// Parse stats JSON.
	var stats map[string]interface{}
	if char.StatsJSON != "" {
		_ = json.Unmarshal([]byte(char.StatsJSON), &stats)
	}

	// Parse stats schema for proper labels.
	type statDefSimple struct {
		Key   string
		Label string
	}
	var vitalDefs []statDefSimple
	var attrDefs []statDefSimple
	var secDefs []statDefSimple

	if story != nil && story.StatsSchemaJSON != "" {
		var schema map[string]interface{}
		if err := json.Unmarshal([]byte(story.StatsSchemaJSON), &schema); err == nil {
			parseDefs := func(key string) []statDefSimple {
				raw, ok := schema[key].([]interface{})
				if !ok {
					return nil
				}
				var defs []statDefSimple
				for _, item := range raw {
					if m2, ok := item.(map[string]interface{}); ok {
						k, _ := m2["key"].(string)
						l, _ := m2["label"].(string)
						if k == "" {
							continue
						}
						if l == "" {
							l = strings.ToUpper(k)
						}
						defs = append(defs, statDefSimple{Key: k, Label: l})
					}
				}
				return defs
			}
			vitalDefs = parseDefs("vitals")
			attrDefs = parseDefs("attributes")
			secDefs = parseDefs("secondary")
		}
	}

	// Character name and background.
	sb.WriteString(theme.Title.Render(char.Name) + "\n")
	if char.Background != "" {
		sb.WriteString(theme.MutedText.Render(char.Background) + "\n")
	}

	if stats != nil {
		sb.WriteString("\n")

		// Vitals with progress bars.
		if vitalsMap, ok := stats["vitals"].(map[string]interface{}); ok && len(vitalsMap) > 0 {
			sb.WriteString(theme.Title.Render("Vitals") + "\n")
			// Use schema order if available, else sort keys.
			vitalKeys := make([]string, 0, len(vitalDefs))
			if len(vitalDefs) > 0 {
				for _, def := range vitalDefs {
					if _, exists := vitalsMap[def.Key]; exists {
						vitalKeys = append(vitalKeys, def.Key)
					}
				}
			}
			if len(vitalKeys) == 0 {
				for k := range vitalsMap {
					vitalKeys = append(vitalKeys, k)
				}
				sort.Strings(vitalKeys)
			}
			// Build label map for lookup.
			vitalLabelMap := map[string]string{}
			for _, def := range vitalDefs {
				vitalLabelMap[def.Key] = def.Label
			}
			for _, key := range vitalKeys {
				vMap, ok := vitalsMap[key].(map[string]interface{})
				if !ok {
					continue
				}
				current := toInt(vMap["current"])
				max := toInt(vMap["max"])
				label := vitalLabelMap[key]
				if label == "" {
					label = strings.ToUpper(key)
				}
				bar := renderBar(current, max, 10)
				sb.WriteString(fmt.Sprintf("  %-10s %s  %d/%d\n", label+":", bar, current, max))
			}
			sb.WriteString("\n")
		}

		// Attributes with schema labels, 3 per row.
		if attrsMap, ok := stats["attributes"].(map[string]interface{}); ok && len(attrsMap) > 0 {
			sb.WriteString(theme.Title.Render("Attributes") + "\n")
			// Use schema order if available.
			type attrEntry struct{ label string; val int }
			var attrEntries []attrEntry
			if len(attrDefs) > 0 {
				for _, def := range attrDefs {
					if v, ok := attrsMap[def.Key]; ok {
						attrEntries = append(attrEntries, attrEntry{def.Label, toInt(v)})
					}
				}
			} else {
				keys := make([]string, 0, len(attrsMap))
				for k := range attrsMap {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					attrEntries = append(attrEntries, attrEntry{strings.ToUpper(k), toInt(attrsMap[k])})
				}
			}
			// Display in rows of 3.
			for i, entry := range attrEntries {
				sb.WriteString(fmt.Sprintf("  %-14s %-3d", entry.label+":", entry.val))
				if (i+1)%3 == 0 || i == len(attrEntries)-1 {
					sb.WriteString("\n")
				}
			}
			sb.WriteString("\n")
		}

		// Secondary stats with schema labels.
		if secMap, ok := stats["secondary"].(map[string]interface{}); ok && len(secMap) > 0 {
			sb.WriteString(theme.Title.Render("Secondary") + "\n")
			secLabelMap := map[string]string{}
			for _, def := range secDefs {
				secLabelMap[def.Key] = def.Label
			}
			// Use schema order if available.
			secKeys := make([]string, 0)
			if len(secDefs) > 0 {
				for _, def := range secDefs {
					if _, ok := secMap[def.Key]; ok {
						secKeys = append(secKeys, def.Key)
					}
				}
			}
			if len(secKeys) == 0 {
				for k := range secMap {
					secKeys = append(secKeys, k)
				}
				sort.Strings(secKeys)
			}
			for _, key := range secKeys {
				val := toInt(secMap[key])
				label := secLabelMap[key]
				if label == "" {
					label = strings.Title(strings.ReplaceAll(key, "_", " "))
				}
				sb.WriteString(fmt.Sprintf("  %-16s %d\n", label+":", val))
			}
			sb.WriteString("\n")
		}
	}

	// Skills with level and XP bar.
	sb.WriteString(theme.Title.Render("Skills") + "\n")
	skillsRendered := false
	// Merge skills from char.SkillsJSON and stats["skills"].
	skillsMap := map[string]interface{}{}
	if char.SkillsJSON != "" && char.SkillsJSON != "null" && char.SkillsJSON != "{}" {
		_ = json.Unmarshal([]byte(char.SkillsJSON), &skillsMap)
	}
	if stats != nil {
		if sm, ok := stats["skills"].(map[string]interface{}); ok {
			for k, v := range sm {
				skillsMap[k] = v
			}
		}
	}
	if len(skillsMap) > 0 {
		skillKeys := make([]string, 0, len(skillsMap))
		for k := range skillsMap {
			skillKeys = append(skillKeys, k)
		}
		sort.Strings(skillKeys)
		for _, name := range skillKeys {
			val := skillsMap[name]
			level := 1
			xp := 0
			if sm, ok := val.(map[string]interface{}); ok {
				level = toInt(sm["level"])
				if level < 1 {
					level = 1
				}
				xp = toInt(sm["xp"])
			}
			threshold := level * 100
			bar := renderBar(xp, threshold, 10)
			sb.WriteString(fmt.Sprintf("  %-16s Lv.%-2d  %s  %d/%d XP\n",
				name, level, bar, xp, threshold))
		}
		skillsRendered = true
	}
	if !skillsRendered {
		sb.WriteString(theme.MutedText.Render("  (none yet — try new things to learn!)") + "\n")
	}
	sb.WriteString("\n")

	// Traits — merged from char.TraitsJSON and stats["traits"], deduplicated.
	sb.WriteString(theme.Title.Render("Traits") + "\n")
	traitSet := map[string]bool{}
	var traits []string
	if char.TraitsJSON != "" && char.TraitsJSON != "null" {
		var t []string
		if err := json.Unmarshal([]byte(char.TraitsJSON), &t); err == nil {
			for _, tr := range t {
				if !traitSet[strings.ToLower(tr)] {
					traitSet[strings.ToLower(tr)] = true
					traits = append(traits, tr)
				}
			}
		}
	}
	if stats != nil {
		if t, ok := stats["traits"].([]interface{}); ok {
			for _, tr := range t {
				if s, ok := tr.(string); ok && !traitSet[strings.ToLower(s)] {
					traitSet[strings.ToLower(s)] = true
					traits = append(traits, s)
				}
			}
		}
	}
	if len(traits) == 0 {
		sb.WriteString(theme.MutedText.Render("  (none yet — your character is still forming)") + "\n")
	} else {
		sb.WriteString("  " + strings.Join(traits, ", ") + "\n")
	}
	sb.WriteString("\n")

	// Titles.
	sb.WriteString(theme.Title.Render("Titles") + "\n")
	var titles []string
	if stats != nil {
		if t, ok := stats["titles"].([]interface{}); ok {
			for _, ti := range t {
				if s, ok := ti.(string); ok {
					titles = append(titles, s)
				}
			}
		}
	}
	if len(titles) == 0 {
		sb.WriteString(theme.MutedText.Render("  (none yet — great deeds earn great titles)") + "\n")
	} else {
		sb.WriteString("  " + strings.Join(titles, ", ") + "\n")
	}
	sb.WriteString("\n")

	// Deaths counter (if > 0).
	if stats != nil {
		if deaths := toInt(stats["deaths"]); deaths > 0 {
			sb.WriteString(theme.DangerText.Render(fmt.Sprintf("  Deaths: %d", deaths)) + "\n")
			sb.WriteString("\n")
		}
	}

	// NPC Relationships — load from DB.
	sb.WriteString(theme.Title.Render("Relationships") + "\n")
	if m.narrator.DB() != nil && story != nil {
		npcs, err := m.narrator.DB().ListNPCs(story.ID)
		if err == nil && len(npcs) > 0 {
			for _, npc := range npcs {
				d := npc.Disposition
				// Map disposition [-100, 100] to bar [0, 10].
				barVal := (d + 100) * 10 / 200
				if barVal < 0 {
					barVal = 0
				}
				if barVal > 10 {
					barVal = 10
				}
				bar := renderBar(barVal, 10, 10)
				label := dispositionLabel(d)
				sign := ""
				if d > 0 {
					sign = "+"
				}
				roleStr := ""
				if npc.Role != "" {
					roleStr = fmt.Sprintf(" (%s)", npc.Role)
				}
				sb.WriteString(fmt.Sprintf("  %-24s %s  %s (%s%d)\n",
					npc.Name+roleStr, bar, label, sign, d))
			}
		} else {
			sb.WriteString(theme.MutedText.Render("  (no one met yet)") + "\n")
		}
	} else {
		sb.WriteString(theme.MutedText.Render("  (no one met yet)") + "\n")
	}

	m.showOverlay(fmt.Sprintf("Character: %s", char.Name), sb.String())
	return m, nil
}

// doSave triggers a named manual save.
func (m NarrativeModel) doSave(args []string) (NarrativeModel, tea.Cmd) {
	saveName := "Manual Save"
	if len(args) > 0 {
		saveName = strings.Join(args, " ")
	}
	narrator := m.narrator
	return m, func() tea.Msg {
		_, err := engine.SaveGame(
			narrator.DB(), narrator.DataDir(),
			narrator.Story(), narrator.Character(), narrator.World(),
			narrator.SessionID(), saveName,
		)
		return SaveCompleteMsg{Name: saveName, Err: err}
	}
}

// doLoad shows the save list overlay (triggers a view switch in the app via SaveLoadMsg).
func (m NarrativeModel) doLoad() (NarrativeModel, tea.Cmd) {
	narrator := m.narrator
	return m, func() tea.Msg {
		saves, err := engine.ListSaves(narrator.DB(), narrator.Story().ID)
		if err != nil || len(saves) == 0 {
			return ShowSaveListMsg{Saves: nil}
		}
		return ShowSaveListMsg{Saves: saves}
	}
}

// ShowSaveListMsg carries the list of saves to display in a picker.
type ShowSaveListMsg struct {
	Saves []storage.SaveSnapshot
}

// doQuit autosaves and signals return to menu.
func (m NarrativeModel) doQuit() (NarrativeModel, tea.Cmd) {
	narrator := m.narrator
	return m, func() tea.Msg {
		_ = engine.Autosave(
			narrator.DB(), narrator.DataDir(),
			narrator.Story(), narrator.Character(), narrator.World(),
			narrator.SessionID(),
		)
		narrator.CloseSession()
		return QuitToMenuMsg{}
	}
}

func (m *NarrativeModel) sendAction(action string) tea.Cmd {
	m.waiting = true
	m.choices.SetChoices(nil) // clear choices while waiting for AI
	narrator := m.narrator
	return func() tea.Msg {
		resp, err := narrator.SendAction(context.Background(), action)
		return narrativeResponseMsg{response: resp, err: err}
	}
}

// maybeAutosaveCmd returns an autosave tea.Cmd if the narrator says it's time,
// or nil if not. Called after a successful narrativeResponseMsg.
func (m *NarrativeModel) maybeAutosaveCmd() tea.Cmd {
	if m.narrator.ShouldAutosave() {
		return m.narrator.AutosaveCmd()
	}
	return nil
}

// updateStatusBar reads the character's stats JSON and populates the status bar.
func (m *NarrativeModel) updateStatusBar() {
	var stats map[string]interface{}
	_ = json.Unmarshal([]byte(m.narrator.Character().StatsJSON), &stats)

	var vitals []components.Vital
	if vitalsMap, ok := stats["vitals"].(map[string]interface{}); ok {
		for key, val := range vitalsMap {
			if vMap, ok := val.(map[string]interface{}); ok {
				current := toInt(vMap["current"])
				max := toInt(vMap["max"])
				vitals = append(vitals, components.Vital{
					Label:   strings.ToUpper(key),
					Current: current,
					Max:     max,
				})
			}
		}
	}

	m.statusBar.SetData(components.StatusBarData{
		Vitals:  vitals,
		Model:   m.narrator.LastModel(),
		Latency: m.narrator.LastLatency(),
	})
}

// toInt safely converts interface{} to int (handles float64 from JSON).
func toInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return 0
}

func (m NarrativeModel) View() string {
	// If overlay is visible, render it full-screen.
	if m.overlay.Visible() {
		return m.overlay.View()
	}

	// Header: chapter + location
	world := m.narrator.World()
	header := theme.Title.Render(fmt.Sprintf("Chapter %d", world.CurrentChapter))
	if world.CurrentLocation != "" {
		header += "  " + theme.Subtitle.Render(world.CurrentLocation)
	}

	// Narrative viewport
	body := m.viewport.View()

	// Error display
	if m.errMsg != "" {
		body += "\n" + theme.DangerText.Render(m.errMsg)
	}

	// Waiting indicator
	var waitLine string
	if m.waiting {
		waitLine = theme.MutedText.Render("  The narrator is writing...")
	}

	// Choices
	choicesView := m.choices.View()

	// Input area
	var inputView string
	if m.inputFocus {
		inputView = m.input.View()
	} else {
		inputView = theme.MutedText.Render("  [TAB] Free input")
	}

	// Temporary status message (e.g. "Autosaved")
	var statusLine string
	if m.statusMsg != "" {
		statusLine = theme.MutedText.Render("  ✓ " + m.statusMsg)
	}

	// Help line
	help := theme.MutedText.Render("tab toggle · 1-4 choose · enter send · /help commands · esc menu")

	// Status bar
	m.statusBar.SetWidth(m.width)
	statusView := m.statusBar.View()

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		body,
		waitLine,
		"",
		choicesView,
		inputView,
		statusLine,
		help,
		statusView,
	)

	return content
}

// SetStatusMsg sets a temporary status message visible in the narrative view.
func (m *NarrativeModel) SetStatusMsg(msg string) {
	m.statusMsg = msg
	m.statusExpiry = time.Now().Add(3 * time.Second)
}

// StoryID returns the ID of the current story.
func (m *NarrativeModel) StoryID() string {
	if m.narrator == nil || m.narrator.Story() == nil {
		return ""
	}
	return m.narrator.Story().ID
}

// CloseSession closes the active game session. Safe to call multiple times.
func (m *NarrativeModel) CloseSession() {
	if m.narrator != nil {
		m.narrator.CloseSession()
	}
}

// SetSize updates the view dimensions and re-layouts sub-components.
func (m *NarrativeModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.overlay.SetSize(w, h)
	m.relayout()
}

func (m *NarrativeModel) relayout() {
	// Reserve: header(1) + gap(1) + choices(~6) + input(2) + help(1) + status(1) + gaps(4) = ~16
	vpHeight := m.height - 16
	if vpHeight < 5 {
		vpHeight = 5
	}
	m.viewport.Width = m.width - 2
	m.viewport.Height = vpHeight
	m.input.SetWidth(m.width - 4)
	m.choices.SetWidth(m.width - 4)
	m.statusBar.SetWidth(m.width)
}
