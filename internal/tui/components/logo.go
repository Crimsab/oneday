package components

import "github.com/crimsab/oneday/internal/tui/theme"

// Logo returns the ASCII art title for the main menu.
func Logo() string {
	art := `
  ___              ____
 / _ \ _ __   ___ |  _ \  __ _ _   _
| | | | '_ \ / _ \| | | |/ _` + "`" + ` | | | |
| |_| | | | |  __/| |_| | (_| | |_| |
 \___/|_| |_|\___||____/ \__,_|\__, |
                                |___/`
	return theme.Logo.Render(art)
}
