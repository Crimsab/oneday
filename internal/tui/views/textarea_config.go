package views

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	storyInputHeight  = 4
	actionInputHeight = 3
)

// newGameTextarea applies the shared OneDay textarea behavior so typing feels
// consistent across story setup and live gameplay.
func newGameTextarea(placeholder string, height int) textarea.Model {
	input := textarea.New()
	input.Placeholder = placeholder
	input.ShowLineNumbers = false
	input.CharLimit = 0
	input.SetHeight(height)
	input.KeyMap.InsertNewline.SetKeys("alt+enter", "shift+enter", "ctrl+j")
	return input
}

func isTextareaNewlineKey(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "alt+enter", "shift+enter", "ctrl+j":
		return true
	default:
		return false
	}
}
