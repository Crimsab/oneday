package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/config"
	appi18n "github.com/crimsab/oneday/internal/i18n"
	"github.com/crimsab/oneday/internal/tui/theme"
)

// SettingsBackMsg returns to the main menu.
type SettingsBackMsg struct{}
type SettingsLocaleChangedMsg struct{ Locale appi18n.Locale }

// SettingsModel is a lightweight read-only settings screen.
// It surfaces the active config instead of leaving the menu option dead.
type SettingsModel struct {
	cfg    config.Config
	width  int
	height int
	loc    appi18n.Localizer
	cursor int
}

// NewSettingsModel creates the settings view.
func NewSettingsModel(cfg config.Config, localizers ...appi18n.Localizer) SettingsModel {
	loc := viewLocalizer(localizers)
	cursor := 0
	if loc.Locale() == appi18n.Italian {
		cursor = 1
	}
	return SettingsModel{cfg: cfg, loc: loc, cursor: cursor}
}

func (m SettingsModel) Init() tea.Cmd { return nil }

func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k", "left", "h":
			m.cursor = 0
		case "down", "j", "right", "l":
			m.cursor = 1
		case "enter", " ":
			locale := appi18n.English
			if m.cursor == 1 {
				locale = appi18n.Italian
			}
			return m, func() tea.Msg { return SettingsLocaleChangedMsg{Locale: locale} }
		case "esc", "q":
			return m, func() tea.Msg { return SettingsBackMsg{} }
		}
	}
	return m, nil
}

func (m SettingsModel) View() string {
	boolLabel := func(value bool) string {
		if value {
			return m.loc.T("common.yes")
		}
		return m.loc.T("common.no")
	}
	englishMarker, italianMarker := "  ", "  "
	if m.cursor == 0 {
		englishMarker = "▸ "
	} else {
		italianMarker = "▸ "
	}
	lines := []string{
		theme.Title.Render(m.loc.T("settings.title")),
		"",
		theme.Subtitle.Render(m.loc.T("settings.interface")),
		m.loc.T("settings.language") + ":",
		englishMarker + m.loc.T("settings.language_en"),
		italianMarker + m.loc.T("settings.language_it"),
		"",
		theme.Subtitle.Render(m.loc.T("settings.ai")),
		fmt.Sprintf("%s: %s", m.loc.T("settings.provider_priority"), strings.Join(m.cfg.EnabledProviders(), " -> ")),
		fmt.Sprintf("Codex — %s: %s  %s=%s  %s=%s  %s=%s", m.loc.T("settings.enabled"), boolLabel(m.cfg.AI.Codex.Enabled), m.loc.T("settings.binary"), m.cfg.AI.Codex.Binary, m.loc.T("settings.model"), m.cfg.AI.Codex.Model, m.loc.T("settings.reasoning"), m.cfg.AI.Codex.Reasoning),
		fmt.Sprintf("LiteLLM — %s: %s  %s=%s  %s=%s", m.loc.T("settings.enabled"), boolLabel(m.cfg.AI.LiteLLM.Enabled), m.loc.T("settings.base_url"), m.cfg.AI.LiteLLM.BaseURL, m.loc.T("settings.model"), m.cfg.AI.LiteLLM.DefaultModel),
		fmt.Sprintf("OpenRouter — %s: %s  %s=%s  %s=%s", m.loc.T("settings.enabled"), boolLabel(m.cfg.AI.OpenRouter.Enabled), m.loc.T("settings.base_url"), m.cfg.AI.OpenRouter.BaseURL, m.loc.T("settings.model"), m.cfg.AI.OpenRouter.DefaultModel),
		fmt.Sprintf("Claude Code — %s: %s  %s=%s", m.loc.T("settings.enabled"), boolLabel(m.cfg.AI.ClaudeCode.Enabled), m.loc.T("settings.binary"), m.cfg.AI.ClaudeCode.Binary),
		"",
		theme.Subtitle.Render(m.loc.T("settings.generation")),
		fmt.Sprintf("%s: %s", m.loc.T("settings.temperature"), m.loc.Decimal(m.cfg.AI.Generation.Temperature)),
		fmt.Sprintf("%s: %d", m.loc.T("settings.max_tokens"), m.cfg.AI.Generation.MaxTokens),
		fmt.Sprintf("%s: %ds", m.loc.T("settings.timeout"), m.cfg.AI.Generation.TimeoutSeconds),
		"",
		theme.Subtitle.Render(m.loc.T("settings.game")),
		fmt.Sprintf("%s: %d %s", m.loc.T("settings.autosave"), m.cfg.Game.AutosaveEvery, m.loc.T("settings.turns")),
		fmt.Sprintf("%s: %s", m.loc.T("settings.typewriter"), boolLabel(m.cfg.Game.TypewriterEffect)),
		fmt.Sprintf("%s: %d cps", m.loc.T("settings.typewriter_speed"), m.cfg.Game.TypewriterSpeed),
		fmt.Sprintf("%s: %s", m.loc.T("settings.visible_private_thoughts"), boolLabel(m.cfg.Game.VisiblePrivateThoughts)),
		fmt.Sprintf("%s: %s", m.loc.T("settings.reward_budget"), m.cfg.Game.RewardBudget),
		"",
		theme.Subtitle.Render(m.loc.T("settings.config")),
		m.loc.T("settings.read_only"),
		"",
		theme.MutedText.Render(m.loc.T("settings.language_help")),
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, content)
}

// SetSize updates dimensions.
func (m *SettingsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}
