package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/engine"
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
	menu     views.MenuModel
	newStory *views.NewStoryModel
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
		if a.newStory != nil {
			a.newStory.SetSize(msg.Width, msg.Height)
		}
		return a, nil

	case tea.KeyMsg:
		// Global quit from any view
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

	case views.StoryCreatedMsg:
		// Story and character persisted — transition to narrative view (Plan 2.4)
		a.view = ViewMenu // placeholder until narrative view is implemented
		return a, nil

	case views.MenuSelectedMsg:
		switch msg.Action {
		case views.ActionQuit:
			return a, tea.Quit
		case views.ActionNewStory:
			creator := engine.NewStoryCreator(a.router, a.db)
			m := views.NewNewStoryModel(creator)
			m.SetSize(a.width, a.height)
			a.newStory = &m
			a.view = ViewNewStory
			return a, a.newStory.StartCreation()
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

	case ViewNewStory:
		if a.newStory != nil {
			// Handle esc to return to menu
			if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEsc {
				a.view = ViewMenu
				return a, nil
			}
			var cmd tea.Cmd
			updated, cmd := a.newStory.Update(msg)
			a.newStory = &updated
			return a, cmd
		}
	}

	return a, nil
}

func (a App) View() string {
	switch a.view {
	case ViewMenu:
		return a.menu.View()
	case ViewNewStory:
		if a.newStory != nil {
			return a.newStory.View()
		}
		return a.menu.View()
	default:
		return a.menu.View()
	}
}

// SetView changes the active view (used by child views to trigger transitions).
func (a *App) SetView(v View) {
	a.view = v
}
