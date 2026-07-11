package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/tui/theme"
)

type HostMiniGameResultMsg struct {
	Result *engine.ChallengeResult
}

type HostMiniGameModel struct {
	host     *engine.MiniGameHost
	instance engine.MiniGameInstance
	cursor   int
	input    textinput.Model
	width    int
	height   int
	error    string
}

func NewHostMiniGameModel(definition engine.MiniGameDefinition, seed int64, width, height int) (HostMiniGameModel, error) {
	host := engine.NewMiniGameHost()
	instance := engine.NewMiniGameInstance("tui-"+definition.ID, "", "", 0, seed, definition)
	if err := host.Start(&instance); err != nil {
		return HostMiniGameModel{}, err
	}
	input := textinput.New()
	input.Placeholder = "Type a response"
	input.Width = 28
	if len(definition.Options) == 0 {
		input.Focus()
	}
	return HostMiniGameModel{host: host, instance: instance, input: input, width: width, height: height}, nil
}

func (model HostMiniGameModel) Update(msg tea.Msg) (HostMiniGameModel, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return model, nil
	}
	if model.instance.Runtime.Phase == engine.MiniGameResolved {
		result := model.instance.Runtime.Result
		return model, func() tea.Msg { return HostMiniGameResultMsg{Result: result} }
	}
	if key.String() == "p" {
		action := "pause"
		if model.instance.Runtime.Phase == engine.MiniGamePaused {
			action = "resume"
		}
		if err := model.host.Apply(&model.instance, engine.MiniGameInput{Action: action}); err != nil {
			model.error = err.Error()
		}
		return model, nil
	}
	if model.instance.Runtime.Phase == engine.MiniGamePaused {
		return model, nil
	}
	options := model.instance.Definition.Options
	switch key.String() {
	case "up", "k":
		if model.cursor > 0 {
			model.cursor--
		}
	case "down", "j":
		if model.cursor < len(options)-1 {
			model.cursor++
		}
	case "enter":
		value := strings.TrimSpace(model.input.Value())
		if len(options) > 0 {
			value = options[model.cursor]
		}
		if value == "" {
			model.error = "A response is required."
			return model, nil
		}
		if err := model.host.Apply(&model.instance, engine.MiniGameInput{Action: "submit", Value: value}); err != nil {
			model.error = err.Error()
		}
	default:
		if len(options) == 0 {
			var cmd tea.Cmd
			model.input, cmd = model.input.Update(msg)
			return model, cmd
		}
	}
	return model, nil
}

func (model HostMiniGameModel) View() string {
	definition := model.instance.Definition
	lines := []string{
		lipgloss.NewStyle().Foreground(theme.Accent).Bold(true).Render("CHALLENGE HOST · " + strings.ToUpper(string(definition.Kind))),
		"", "  " + definition.Prompt, "",
	}
	if model.instance.Runtime.Phase == engine.MiniGamePaused {
		lines = append(lines, theme.MutedText.Render("  Paused · press P to resume"))
	} else if result := model.instance.Runtime.Result; result != nil {
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.Success).Bold(true).Render("  "+strings.ToUpper(strings.ReplaceAll(string(result.Outcome.Degree), "_", " "))), "", "  "+result.Detail, "", theme.MutedText.Render("  Press any key to continue"))
	} else if len(definition.Options) > 0 {
		for index, option := range definition.Options {
			prefix := "  "
			if index == model.cursor {
				prefix = "▸ "
			}
			lines = append(lines, prefix+option)
		}
		lines = append(lines, "", theme.MutedText.Render("  ↑/↓ select · Enter resolve · P pause"))
	} else {
		lines = append(lines, "  > "+model.input.View(), "", theme.MutedText.Render("  Enter resolve · P pause"))
	}
	if model.error != "" {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(theme.Danger).Render("  "+model.error))
	}
	inner := strings.Join(lines, "\n")
	box := lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(theme.Accent).Padding(1, 2).Width(48).Render(inner)
	return lipgloss.Place(model.width, model.height, lipgloss.Center, lipgloss.Center, box)
}

func (model HostMiniGameModel) Instance() engine.MiniGameInstance {
	return model.instance
}

func (model HostMiniGameModel) String() string {
	return fmt.Sprintf("%s:%s", model.instance.Definition.Kind, model.instance.Runtime.Phase)
}
