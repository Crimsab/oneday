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

// showHelp displays the help overlay.
func (m NarrativeModel) showHelp() (NarrativeModel, tea.Cmd) {
	helpText := `Available Commands:

  /inventory  (/i)   Show your inventory
  /stats      (/s)   Show character sheet
  /save [name]       Save your game
  /load              Load a saved game
  /help       (/h)   Show this help
  /quit       (/q)   Save and quit to menu`

	m.showOverlay("Help", helpText)
	return m, nil
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

	// Parse stats for currency and equipped items.
	var stats map[string]interface{}
	if char.StatsJSON != "" {
		_ = json.Unmarshal([]byte(char.StatsJSON), &stats)
	}

	// Try to extract inventory sections from stats (if AI stored there).
	var backpack []string
	var equipped []string
	var questItems []string

	switch inv := inventoryRaw.(type) {
	case []interface{}:
		for _, item := range inv {
			if s, ok := item.(string); ok {
				backpack = append(backpack, s)
			} else if m, ok := item.(map[string]interface{}); ok {
				name := fmt.Sprintf("%v", m["name"])
				if eq, _ := m["equipped"].(bool); eq {
					equipped = append(equipped, name)
				} else if qt, _ := m["quest"].(bool); qt {
					questItems = append(questItems, name)
				} else {
					backpack = append(backpack, name)
				}
			}
		}
	case map[string]interface{}:
		if bp, ok := inv["backpack"].([]interface{}); ok {
			for _, item := range bp {
				backpack = append(backpack, fmt.Sprintf("%v", item))
			}
		}
		if eq, ok := inv["equipped"].([]interface{}); ok {
			for _, item := range eq {
				equipped = append(equipped, fmt.Sprintf("%v", item))
			}
		}
		if qt, ok := inv["quest"].([]interface{}); ok {
			for _, item := range qt {
				questItems = append(questItems, fmt.Sprintf("%v", item))
			}
		}
	}

	// Backpack section.
	sb.WriteString("Backpack:\n")
	if len(backpack) == 0 {
		sb.WriteString("  (empty)\n")
	} else {
		for _, item := range backpack {
			sb.WriteString(fmt.Sprintf("  • %s\n", item))
		}
	}

	sb.WriteString("\n")

	// Equipped section.
	sb.WriteString("Equipped:\n")
	if len(equipped) == 0 {
		sb.WriteString("  (nothing equipped)\n")
	} else {
		for _, item := range equipped {
			sb.WriteString(fmt.Sprintf("  ⚔ %s\n", item))
		}
	}

	sb.WriteString("\n")

	// Quest items section.
	sb.WriteString("Quest Items:\n")
	if len(questItems) == 0 {
		sb.WriteString("  (none)\n")
	} else {
		for _, item := range questItems {
			sb.WriteString(fmt.Sprintf("  ◆ %s\n", item))
		}
	}

	sb.WriteString("\n")

	// Currency.
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
	sb.WriteString(fmt.Sprintf("Currency: %d %s", currency, currencyName))

	m.showOverlay("Inventory", sb.String())
	return m, nil
}

// showStats displays the full character sheet as an overlay.
func (m NarrativeModel) showStats() (NarrativeModel, tea.Cmd) {
	char := m.narrator.Character()
	var sb strings.Builder

	sb.WriteString("\n")

	// Parse stats JSON.
	var stats map[string]interface{}
	if char.StatsJSON != "" {
		_ = json.Unmarshal([]byte(char.StatsJSON), &stats)
	}

	// Character name and background.
	sb.WriteString(fmt.Sprintf("Name: %s\n", char.Name))
	if char.Background != "" {
		sb.WriteString(fmt.Sprintf("Background: %s\n", char.Background))
	}

	if stats != nil {
		sb.WriteString("\n")

		// Vitals.
		if vitalsMap, ok := stats["vitals"].(map[string]interface{}); ok && len(vitalsMap) > 0 {
			sb.WriteString("Vitals:\n")
			// Sort keys for consistent display.
			keys := make([]string, 0, len(vitalsMap))
			for k := range vitalsMap {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, key := range keys {
				if vMap, ok := vitalsMap[key].(map[string]interface{}); ok {
					current := toInt(vMap["current"])
					max := toInt(vMap["max"])
					sb.WriteString(fmt.Sprintf("  %-10s %d/%d\n", strings.ToUpper(key)+":", current, max))
				}
			}
			sb.WriteString("\n")
		}

		// Attributes.
		if attrsMap, ok := stats["attributes"].(map[string]interface{}); ok && len(attrsMap) > 0 {
			sb.WriteString("Attributes:\n")
			keys := make([]string, 0, len(attrsMap))
			for k := range attrsMap {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			// Display in rows of 3.
			for i, key := range keys {
				val := toInt(attrsMap[key])
				sb.WriteString(fmt.Sprintf("  %-4s %-3d", strings.ToUpper(key)+":", val))
				if (i+1)%3 == 0 || i == len(keys)-1 {
					sb.WriteString("\n")
				}
			}
			sb.WriteString("\n")
		}

		// Secondary stats.
		if secMap, ok := stats["secondary"].(map[string]interface{}); ok && len(secMap) > 0 {
			sb.WriteString("Secondary:\n")
			keys := make([]string, 0, len(secMap))
			for k := range secMap {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, key := range keys {
				val := toInt(secMap[key])
				label := strings.Title(strings.ReplaceAll(key, "_", " "))
				sb.WriteString(fmt.Sprintf("  %-14s %d\n", label+":", val))
			}
			sb.WriteString("\n")
		}
	}

	// Traits.
	sb.WriteString("Traits: ")
	var traits []string
	if char.TraitsJSON != "" && char.TraitsJSON != "null" {
		_ = json.Unmarshal([]byte(char.TraitsJSON), &traits)
	}
	if stats != nil {
		if t, ok := stats["traits"].([]interface{}); ok {
			for _, tr := range t {
				if s, ok := tr.(string); ok {
					traits = append(traits, s)
				}
			}
		}
	}
	if len(traits) == 0 {
		sb.WriteString("(none)\n")
	} else {
		sb.WriteString(strings.Join(traits, ", ") + "\n")
	}

	// Skills.
	sb.WriteString("Skills:  ")
	var skillNames []string
	if char.SkillsJSON != "" && char.SkillsJSON != "null" {
		var skillsMap map[string]interface{}
		if err := json.Unmarshal([]byte(char.SkillsJSON), &skillsMap); err == nil {
			for k := range skillsMap {
				skillNames = append(skillNames, k)
			}
		}
		var skillsList []string
		if err := json.Unmarshal([]byte(char.SkillsJSON), &skillsList); err == nil {
			skillNames = skillsList
		}
	}
	if stats != nil {
		if skillsMap, ok := stats["skills"].(map[string]interface{}); ok {
			for k := range skillsMap {
				skillNames = append(skillNames, k)
			}
		}
	}
	if len(skillNames) == 0 {
		sb.WriteString("(none)\n")
	} else {
		sort.Strings(skillNames)
		sb.WriteString(strings.Join(skillNames, ", ") + "\n")
	}

	// Titles.
	sb.WriteString("Titles:  ")
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
		sb.WriteString("(none)\n")
	} else {
		sb.WriteString(strings.Join(titles, ", ") + "\n")
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
