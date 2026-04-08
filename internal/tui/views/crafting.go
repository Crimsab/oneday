package views

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/tui/components"
	"github.com/crimsab/oneday/internal/tui/theme"
)

// CraftingEndMsg signals crafting session is over.
type CraftingEndMsg struct {
	ItemCrafted bool
	ItemName    string
}

// craftingResponseMsg carries AI crafting evaluation result.
type craftingResponseMsg struct {
	response *engine.CraftingResponse
	err      error
}

// CraftingModel is the TUI view for crafting conversations.
type CraftingModel struct {
	crafting *engine.CraftingEngine
	narrator *engine.Narrator

	// Layout
	width  int
	height int

	// Sub-components
	chatView   viewport.Model
	typewriter components.TypewriterModel
	choices    components.ChoiceListModel
	input      textarea.Model

	// State
	history     strings.Builder
	inventory   string // rendered inventory sidebar
	inputFocus  bool
	waiting     bool
	errMsg      string
	lastCrafted string // name of last crafted item (if any)
}

// NewCraftingModel creates a crafting view.
func NewCraftingModel(crafting *engine.CraftingEngine, narrator *engine.Narrator, width, height int) CraftingModel {
	// Create textarea for free input.
	input := textarea.New()
	input.Placeholder = "Descrivi cosa vuoi creare..."
	input.ShowLineNumbers = false
	input.CharLimit = 512
	chatWidth := craftingChatWidth(width)
	input.SetWidth(chatWidth - 4)
	input.SetHeight(3)
	input.Focus()

	// Initialize choice list.
	choices := components.NewChoiceList()
	choices.SetWidth(chatWidth - 4)

	// Initialize chat viewport.
	vpH := height - 16
	if vpH < 4 {
		vpH = 4
	}
	chatVP := viewport.New(chatWidth-4, vpH)
	chatVP.SetContent("")

	// Initialize typewriter.
	tw := components.NewTypewriter(80)

	m := CraftingModel{
		crafting:   crafting,
		narrator:   narrator,
		width:      width,
		height:     height,
		chatView:   chatVP,
		typewriter: tw,
		choices:    choices,
		input:      input,
		inputFocus: true,
	}

	// Build initial inventory sidebar.
	m.inventory = m.buildInventorySidebar()

	// Add welcome message to history.
	welcome := "Benvenuto alla stazione di crafting. Descrivi cosa vuoi creare e valuterò se è possibile con i materiali che hai.\n\n"
	m.history.WriteString(welcome)
	_ = tw.SetText(m.history.String())

	return m
}

// craftingChatWidth returns the width of the left (chat) panel.
func craftingChatWidth(totalWidth int) int {
	w := totalWidth * 70 / 100
	if w < 40 {
		w = 40
	}
	return w
}

// craftingSidebarWidth returns the width of the right (inventory) panel.
func craftingSidebarWidth(totalWidth int) int {
	return totalWidth - craftingChatWidth(totalWidth)
}

// Init initialises the crafting model.
func (m CraftingModel) Init() tea.Cmd {
	return nil
}

// SetSize updates the terminal dimensions.
func (m *CraftingModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	chatW := craftingChatWidth(w)
	m.input.SetWidth(chatW - 4)
	m.choices.SetWidth(chatW - 4)
	vpH := h - 16
	if vpH < 4 {
		vpH = 4
	}
	m.chatView = viewport.New(chatW-4, vpH)
	m.chatView.SetContent(m.typewriter.View())
}

// sendCraftingMessage sends a message to the crafting engine as a tea.Cmd.
func (m CraftingModel) sendCraftingMessage(msg string) tea.Cmd {
	crafting := m.crafting
	return func() tea.Msg {
		resp, err := crafting.SendMessage(context.Background(), msg)
		return craftingResponseMsg{response: resp, err: err}
	}
}

// Update handles messages for the crafting view.
func (m CraftingModel) Update(msg tea.Msg) (CraftingModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, nil

	case craftingResponseMsg:
		m.waiting = false
		if msg.err != nil {
			m.errMsg = fmt.Sprintf("Crafting error: %v", msg.err)
			return m, nil
		}
		resp := msg.response

		// Append narrative to chat history.
		rendered := components.RenderMarkdown(resp.Narrative)
		m.history.WriteString(rendered + "\n")

		// If item was crafted, add a celebration note.
		if resp.Feasible && resp.Item != nil {
			m.lastCrafted = resp.Item.Name
			note := fmt.Sprintf("\n**[Creato: %s]**\n_%s_\n\nEffetto: %s\n\n",
				resp.Item.Name, resp.Item.Description, resp.Item.Effect)
			m.history.WriteString(components.RenderMarkdown(note))
			// Refresh inventory sidebar since items changed.
			m.inventory = m.buildInventorySidebar()
		}

		// Animate new text.
		cmd := m.typewriter.SetText(m.history.String())
		cmds = append(cmds, cmd)

		// Update choices.
		items := make([]components.ChoiceItem, len(resp.Choices))
		for i, c := range resp.Choices {
			items[i] = components.ChoiceItem{ID: c.ID, Text: c.Text}
		}
		m.choices.SetChoices(items)
		m.inputFocus = false
		m.input.Blur()

		return m, tea.Batch(cmds...)

	case components.TypewriterDoneMsg:
		m.chatView.SetContent(m.typewriter.View())
		m.chatView.GotoBottom()
		return m, nil

	case components.ChoiceSelectedMsg:
		if m.waiting {
			return m, nil
		}

		// Check if this is an "exit" choice.
		lowerText := strings.ToLower(msg.Text)
		if strings.Contains(lowerText, "esci") || strings.Contains(lowerText, "leave") ||
			strings.Contains(lowerText, "exit") || strings.Contains(lowerText, "lascia") {
			_ = m.crafting.Close()
			return m, func() tea.Msg {
				return CraftingEndMsg{
					ItemCrafted: m.lastCrafted != "",
					ItemName:    m.lastCrafted,
				}
			}
		}

		m.waiting = true
		m.errMsg = ""
		return m, m.sendCraftingMessage(msg.Text)

	case tea.KeyMsg:
		if m.waiting {
			return m, nil
		}

		switch msg.String() {
		case "tab":
			m.inputFocus = !m.inputFocus
			if m.inputFocus {
				m.input.Focus()
			} else {
				m.input.Blur()
			}
			return m, nil

		case "enter":
			if m.inputFocus {
				action := strings.TrimSpace(m.input.Value())
				if action == "" {
					return m, nil
				}
				// Append player message to history.
				m.history.WriteString("> " + action + "\n\n")
				m.input.Reset()
				m.waiting = true
				m.errMsg = ""
				return m, m.sendCraftingMessage(action)
			}

		case "esc":
			// Exit crafting.
			_ = m.crafting.Close()
			return m, func() tea.Msg {
				return CraftingEndMsg{
					ItemCrafted: m.lastCrafted != "",
					ItemName:    m.lastCrafted,
				}
			}
		}

		// Route key to focused component.
		if m.inputFocus {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		} else {
			var cmd tea.Cmd
			m.choices, cmd = m.choices.Update(msg)
			cmds = append(cmds, cmd)
		}

		// Always update typewriter for animation ticks.
		var twCmd tea.Cmd
		m.typewriter, twCmd = m.typewriter.Update(msg)
		cmds = append(cmds, twCmd)

		return m, tea.Batch(cmds...)
	}

	// Update typewriter for tick messages.
	var twCmd tea.Cmd
	m.typewriter, twCmd = m.typewriter.Update(msg)
	if twCmd != nil {
		cmds = append(cmds, twCmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the crafting screen (two-column layout).
func (m CraftingModel) View() string {
	if m.width == 0 {
		return "Loading crafting..."
	}

	chatW := craftingChatWidth(m.width)
	sideW := craftingSidebarWidth(m.width)
	if sideW < 10 {
		// Narrow terminal: single column.
		return m.viewSingleColumn()
	}

	// --- Left panel: chat ---
	leftParts := []string{}

	// Header.
	header := theme.CraftingHeader.Render("CRAFTING")
	if m.waiting {
		header += theme.MutedText.Render("  [valutando...]")
	}
	leftParts = append(leftParts, lipgloss.NewStyle().Padding(0, 1).Render(header))

	// Separator.
	leftParts = append(leftParts, theme.MutedText.Render(strings.Repeat("─", chatW-2)))

	// Chat viewport.
	m.chatView.SetContent(m.typewriter.View())
	leftParts = append(leftParts, lipgloss.NewStyle().Padding(0, 1).Render(m.chatView.View()))

	// Error message.
	if m.errMsg != "" {
		leftParts = append(leftParts, theme.DangerText.Render("  "+m.errMsg))
	}

	// Separator.
	leftParts = append(leftParts, theme.MutedText.Render(strings.Repeat("─", chatW-2)))

	// Choices.
	if m.choices.HasChoices() {
		leftParts = append(leftParts, lipgloss.NewStyle().Padding(0, 1).Render(m.choices.View()))
	}

	// Focus hint.
	focusHint := ""
	if !m.inputFocus && m.choices.HasChoices() {
		focusHint = theme.MutedText.Render(" [Tab: input libero]")
	} else if m.inputFocus && m.choices.HasChoices() {
		focusHint = theme.MutedText.Render(" [Tab: scelte | Esc: esci]")
	} else {
		focusHint = theme.MutedText.Render(" [Esc: esci]")
	}

	// Input.
	leftParts = append(leftParts, lipgloss.JoinVertical(lipgloss.Left,
		theme.MutedText.Render("  Cosa vuoi creare?")+focusHint,
		lipgloss.NewStyle().Padding(0, 1).Render(m.input.View()),
	))

	leftPanel := lipgloss.NewStyle().
		Width(chatW).
		Height(m.height - 2).
		Render(strings.Join(leftParts, "\n"))

	// --- Right panel: inventory sidebar ---
	sideHeader := theme.Title.Render("Inventario")
	sideContent := lipgloss.JoinVertical(lipgloss.Left,
		sideHeader,
		theme.MutedText.Render(strings.Repeat("─", sideW-4)),
		m.inventory,
	)

	rightPanel := theme.InventorySidebar.
		Width(sideW).
		Height(m.height - 2).
		Render(sideContent)

	// Join columns.
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

// viewSingleColumn renders the crafting view in single-column mode (narrow terminals).
func (m CraftingModel) viewSingleColumn() string {
	var parts []string

	parts = append(parts, theme.CraftingHeader.Render("CRAFTING")+" "+theme.MutedText.Render("[Inventario: /i]"))
	parts = append(parts, theme.MutedText.Render(strings.Repeat("─", m.width-2)))

	m.chatView.SetContent(m.typewriter.View())
	parts = append(parts, lipgloss.NewStyle().Padding(0, 1).Render(m.chatView.View()))

	if m.errMsg != "" {
		parts = append(parts, theme.DangerText.Render("  "+m.errMsg))
	}

	parts = append(parts, theme.MutedText.Render(strings.Repeat("─", m.width-2)))

	if m.choices.HasChoices() {
		parts = append(parts, lipgloss.NewStyle().Padding(0, 1).Render(m.choices.View()))
	}

	parts = append(parts, lipgloss.NewStyle().Padding(0, 1).Render(m.input.View()))

	return strings.Join(parts, "\n")
}

// buildInventorySidebar creates a formatted inventory listing from character stats.
func (m CraftingModel) buildInventorySidebar() string {
	char := m.narrator.Character()
	if char == nil {
		return theme.MutedText.Render("  (vuoto)")
	}

	var sb strings.Builder

	// Parse inventory from StatsJSON.
	var stats map[string]interface{}
	if char.StatsJSON != "" {
		_ = json.Unmarshal([]byte(char.StatsJSON), &stats)
	}

	// Backpack items.
	if stats != nil {
		inv, _ := stats["inventory"].(map[string]interface{})
		if inv != nil {
			backpack, _ := inv["backpack"].([]interface{})
			if len(backpack) > 0 {
				sb.WriteString(theme.MutedText.Render("Zaino:\n"))
				for _, item := range backpack {
					name := fmt.Sprintf("%v", item)
					sb.WriteString(theme.NormalText.Render("  • "+name) + "\n")
				}
			} else {
				sb.WriteString(theme.MutedText.Render("  (zaino vuoto)\n"))
			}
		} else {
			sb.WriteString(theme.MutedText.Render("  (zaino vuoto)\n"))
		}
	}

	// Known recipes.
	recipes, _ := engine.GetKnownRecipes(char)
	if len(recipes) > 0 {
		sb.WriteString("\n" + theme.MutedText.Render("Ricette note:\n"))
		for _, r := range recipes {
			sb.WriteString(theme.RecipeItem.Render("  ★ "+r.Name) + "\n")
		}
	}

	if sb.Len() == 0 {
		return theme.MutedText.Render("  (vuoto)")
	}
	return sb.String()
}
