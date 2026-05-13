package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/tui/theme"
)

// SettingsBackMsg returns to the main menu.
type SettingsBackMsg struct{}

// SettingsModel is a lightweight read-only settings screen.
// It surfaces the active config instead of leaving the menu option dead.
type SettingsModel struct {
	cfg    config.Config
	width  int
	height int
}

// NewSettingsModel creates the settings view.
func NewSettingsModel(cfg config.Config) SettingsModel {
	return SettingsModel{cfg: cfg}
}

func (m SettingsModel) Init() tea.Cmd { return nil }

func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return m, func() tea.Msg { return SettingsBackMsg{} }
		}
	}
	return m, nil
}

func (m SettingsModel) View() string {
	lines := []string{
		theme.Title.Render("Settings"),
		"",
		theme.Subtitle.Render("AI"),
		fmt.Sprintf("Provider priority: %s", strings.Join(m.cfg.EnabledProviders(), " -> ")),
		fmt.Sprintf("Codex: %t  binary=%s  model=%s  reasoning=%s", m.cfg.AI.Codex.Enabled, m.cfg.AI.Codex.Binary, m.cfg.AI.Codex.Model, m.cfg.AI.Codex.Reasoning),
		fmt.Sprintf("LiteLLM: %t  %s  model=%s", m.cfg.AI.LiteLLM.Enabled, m.cfg.AI.LiteLLM.BaseURL, m.cfg.AI.LiteLLM.DefaultModel),
		fmt.Sprintf("OpenRouter: %t  %s  model=%s", m.cfg.AI.OpenRouter.Enabled, m.cfg.AI.OpenRouter.BaseURL, m.cfg.AI.OpenRouter.DefaultModel),
		fmt.Sprintf("Claude Code: %t  binary=%s", m.cfg.AI.ClaudeCode.Enabled, m.cfg.AI.ClaudeCode.Binary),
		"",
		theme.Subtitle.Render("Generation"),
		fmt.Sprintf("Temperature: %.2f", m.cfg.AI.Generation.Temperature),
		fmt.Sprintf("Max tokens: %d", m.cfg.AI.Generation.MaxTokens),
		fmt.Sprintf("Timeout: %ds", m.cfg.AI.Generation.TimeoutSeconds),
		"",
		theme.Subtitle.Render("Game"),
		fmt.Sprintf("Autosave every: %d turns", m.cfg.Game.AutosaveEvery),
		fmt.Sprintf("Typewriter effect: %t", m.cfg.Game.TypewriterEffect),
		fmt.Sprintf("Typewriter speed: %d cps", m.cfg.Game.TypewriterSpeed),
		fmt.Sprintf("visible_private_thoughts: %t", m.cfg.Game.VisiblePrivateThoughts),
		"",
		theme.Subtitle.Render("Config"),
		"Read-only for now. Edit config.yaml to change providers, keys, and defaults.",
		"",
		theme.MutedText.Render("esc/q back"),
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, content)
}

// SetSize updates dimensions.
func (m *SettingsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}
