package theme

import "github.com/charmbracelet/lipgloss"

// Colors — a warm, RPG-appropriate palette.
var (
	Primary    = lipgloss.Color("#D4A574") // warm gold
	Secondary  = lipgloss.Color("#8B7355") // muted brown
	Accent     = lipgloss.Color("#C9A96E") // bright gold
	Danger     = lipgloss.Color("#CD5C5C") // muted red
	Success    = lipgloss.Color("#6B8E6B") // forest green
	Muted      = lipgloss.Color("#666666") // gray
	Background = lipgloss.Color("#1A1A1A") // near-black
	Text       = lipgloss.Color("#E8E0D4") // warm white
	Highlight  = lipgloss.Color("#FFD700") // gold highlight
)

// Combat and crafting specific colors.
var (
	CombatRed    = lipgloss.Color("#FF4444") // bright red for combat header
	CombatGold   = lipgloss.Color("#FFD700") // gold for combat accents
	CraftingBlue = lipgloss.Color("#4A9BD9") // blue for crafting header
)

// Combat styles.
var (
	CombatHeader = lipgloss.NewStyle().
			Foreground(CombatRed).
			Bold(true)

	CombatTurn = lipgloss.NewStyle().
			Foreground(CombatGold).
			Bold(true)

	HPBarFull = lipgloss.NewStyle().
			Foreground(Success)

	HPBarMid = lipgloss.NewStyle().
			Foreground(Accent)

	HPBarLow = lipgloss.NewStyle().
			Foreground(Danger)

	HPBarEmpty = lipgloss.NewStyle().
			Foreground(Muted)
)

// Crafting styles.
var (
	CraftingHeader = lipgloss.NewStyle().
			Foreground(CraftingBlue).
			Bold(true)

	InventorySidebar = lipgloss.NewStyle().
				Foreground(Text).
				BorderForeground(Secondary).
				PaddingLeft(1)

	RecipeItem = lipgloss.NewStyle().
			Foreground(Accent)
)

// Styles — reusable across all views.
var (
	// Title is for view titles and headers.
	Title = lipgloss.NewStyle().
		Foreground(Primary).
		Bold(true)

	// Subtitle for secondary headers (chapter, location).
	Subtitle = lipgloss.NewStyle().
		Foreground(Secondary).
		Italic(true)

	// NormalText for narrative body.
	NormalText = lipgloss.NewStyle().
		Foreground(Text)

	// MutedText for hints, placeholders.
	MutedText = lipgloss.NewStyle().
		Foreground(Muted)

	// SelectedItem for highlighted menu/choice items.
	SelectedItem = lipgloss.NewStyle().
		Foreground(Highlight).
		Bold(true)

	// UnselectedItem for non-highlighted items.
	UnselectedItem = lipgloss.NewStyle().
		Foreground(Text)

	// StatusBar for the bottom status bar.
	StatusBar = lipgloss.NewStyle().
		Foreground(Text).
		Background(lipgloss.Color("#2A2A2A")).
		Padding(0, 1)

	// Border style for the main frame.
	Border = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Secondary)

	// Logo for the ASCII title on the menu.
	Logo = lipgloss.NewStyle().
		Foreground(Primary).
		Bold(true)

	// DangerText for HP warnings, errors.
	DangerText = lipgloss.NewStyle().
		Foreground(Danger)

	// SuccessText for positive feedback.
	SuccessText = lipgloss.NewStyle().
		Foreground(Success)
)
