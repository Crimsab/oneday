package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/storage"
	"github.com/crimsab/oneday/internal/tui/views"
)

// View represents which screen is active.
type View int

const (
	ViewMenu View = iota
	ViewNewStory
	ViewNarrative
)

// App is the top-level Bubbletea model.
type App struct {
	cfg    config.Config
	db     *storage.DB
	router *ai.Router
	view   View
	width  int
	height int

	// Child view models
	menu views.MenuModel
	// newStory and narrative will be added in Plans 2.2 and 2.4
}

// New creates the app with all dependencies.
func New(cfg config.Config, db *storage.DB, router *ai.Router) App {
	return App{
		cfg:    cfg,
		db:     db,
		router: router,
		view:   ViewMenu,
		menu:   views.NewMenuModel(),
	}
}

func (a App) Init() tea.Cmd {
	return nil
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.menu.SetSize(msg.Width, msg.Height)
		return a, nil

	case tea.KeyMsg:
		// Global quit from any view
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

	case views.MenuSelectedMsg:
		switch msg.Action {
		case views.ActionQuit:
			return a, tea.Quit
		case views.ActionNewStory:
			// Will be wired in Plan 2.2
			a.view = ViewMenu // stays on menu for now
			return a, nil
		case views.ActionLoadStory:
			// Will be implemented in Phase 3
			return a, nil
		case views.ActionSettings:
			// Will be implemented later
			return a, nil
		}
	}

	// Route to active view
	switch a.view {
	case ViewMenu:
		var cmd tea.Cmd
		a.menu, cmd = a.menu.Update(msg)
		return a, cmd
	}

	return a, nil
}

func (a App) View() string {
	switch a.view {
	case ViewMenu:
		return a.menu.View()
	default:
		return a.menu.View()
	}
}

// SetView changes the active view (used by child views to trigger transitions).
func (a *App) SetView(v View) {
	a.view = v
}
