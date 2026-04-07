package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/ai/providers"
	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/rag"
	"github.com/crimsab/oneday/internal/storage"
	"github.com/crimsab/oneday/internal/tui/views"
)

// View represents which screen is active.
type View int

const (
	ViewMenu View = iota
	ViewNewStory
	ViewNarrative
	ViewLoadStory
	ViewSaveLoad
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
	menu      views.MenuModel
	newStory  *views.NewStoryModel
	narrative *views.NarrativeModel
	loadStory *views.LoadStoryModel
	saveLoad  *views.SaveLoadModel
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
		if a.narrative != nil {
			a.narrative.SetSize(msg.Width, msg.Height)
		}
		if a.loadStory != nil {
			a.loadStory.SetSize(msg.Width, msg.Height)
		}
		if a.saveLoad != nil {
			a.saveLoad.SetSize(msg.Width, msg.Height)
		}
		return a, nil

	case tea.KeyMsg:
		// Global quit from any view
		if msg.String() == "ctrl+c" {
			a.cleanup()
			return a, tea.Quit
		}

	case views.StoryCreatedMsg:
		// Story and character persisted — load them and transition to narrative view
		cmd, err := a.enterNarrativeView(msg.StoryID)
		if err != nil {
			a.view = ViewMenu
			return a, nil
		}
		return a, cmd

	case views.StorySelectedMsg:
		// User selected a story from the load list
		cmd, err := a.enterNarrativeView(msg.StoryID)
		if err != nil {
			a.view = ViewMenu
			return a, nil
		}
		return a, cmd

	case views.LoadStoryBackMsg:
		// User pressed Esc in load story view — back to menu
		a.view = ViewMenu
		return a, nil

	case engine.AutosaveCompleteMsg:
		// Route autosave notification to the narrative view
		if a.narrative != nil {
			updated, cmd := a.narrative.Update(msg)
			a.narrative = &updated
			return a, cmd
		}
		return a, nil

	case views.QuitToMenuMsg:
		// /quit command: session already closed by the command handler
		a.narrative = nil
		a.view = ViewMenu
		return a, nil

	case views.ShowSaveListMsg:
		// /load command: show save picker if saves exist
		if len(msg.Saves) == 0 {
			// No saves — show a status message in the narrative view
			if a.narrative != nil {
				a.narrative.SetStatusMsg("No saves found.")
			}
			return a, nil
		}
		m := views.NewSaveLoadModel(msg.Saves)
		m.SetSize(a.width, a.height)
		a.saveLoad = &m
		a.view = ViewSaveLoad
		return a, nil

	case views.SaveLoadSelectedMsg:
		// User picked a save — load it and resume narrative
		if a.narrative != nil {
			storyID := a.narrative.StoryID()
			cmd, err := a.loadSaveAndResume(storyID, msg.SaveID)
			if err != nil {
				// Failed to load — return to narrative
				a.view = ViewNarrative
				return a, nil
			}
			a.saveLoad = nil
			return a, cmd
		}
		a.view = ViewNarrative
		return a, nil

	case views.SaveLoadCancelMsg:
		// User pressed Esc in save picker — back to game
		a.saveLoad = nil
		a.view = ViewNarrative
		return a, nil

	case views.SaveCompleteMsg:
		// Route save-complete message to narrative view
		if a.narrative != nil {
			updated, cmd := a.narrative.Update(msg)
			a.narrative = &updated
			return a, cmd
		}
		return a, nil

	case views.MenuSelectedMsg:
		switch msg.Action {
		case views.ActionQuit:
			a.cleanup()
			return a, tea.Quit
		case views.ActionNewStory:
			creator := engine.NewStoryCreator(a.router, a.db)
			m := views.NewNewStoryModel(creator)
			m.SetSize(a.width, a.height)
			a.newStory = &m
			a.view = ViewNewStory
			return a, a.newStory.StartCreation()
		case views.ActionLoadStory:
			stories, err := a.db.ListStories()
			if err != nil {
				// Fall back to menu on error
				return a, nil
			}
			m := views.NewLoadStoryModel(stories)
			m.SetSize(a.width, a.height)
			a.loadStory = &m
			a.view = ViewLoadStory
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
			if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEsc {
				a.view = ViewMenu
				return a, nil
			}
			var cmd tea.Cmd
			updated, cmd := a.newStory.Update(msg)
			a.newStory = &updated
			return a, cmd
		}

	case ViewNarrative:
		if a.narrative != nil {
			if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEsc {
				a.narrative.CloseSession()
				a.view = ViewMenu
				return a, nil
			}
			var cmd tea.Cmd
			updated, cmd := a.narrative.Update(msg)
			a.narrative = &updated
			return a, cmd
		}

	case ViewLoadStory:
		if a.loadStory != nil {
			var cmd tea.Cmd
			updated, cmd := a.loadStory.Update(msg)
			a.loadStory = &updated
			return a, cmd
		}

	case ViewSaveLoad:
		if a.saveLoad != nil {
			var cmd tea.Cmd
			updated, cmd := a.saveLoad.Update(msg)
			a.saveLoad = &updated
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
	case ViewNarrative:
		if a.narrative != nil {
			return a.narrative.View()
		}
		return a.menu.View()
	case ViewLoadStory:
		if a.loadStory != nil {
			return a.loadStory.View()
		}
		return a.menu.View()
	case ViewSaveLoad:
		if a.saveLoad != nil {
			return a.saveLoad.View()
		}
		return a.menu.View()
	default:
		return a.menu.View()
	}
}

// SetView changes the active view.
func (a *App) SetView(v View) {
	a.view = v
}

// cleanup closes any active session before quitting.
func (a *App) cleanup() {
	if a.narrative != nil {
		a.narrative.CloseSession()
	}
}

// buildRAG constructs the RAG pipeline for a story, or returns nil if RAG is disabled
// or no embedding provider is configured. Failure is non-fatal — gameplay continues without RAG.
func (a *App) buildRAG(storyID string) *rag.RAG {
	if !a.cfg.RAG.Enabled {
		return nil
	}

	// Use LiteLLM as the embedding provider (same base URL, /v1/embeddings endpoint).
	liteCfg := a.cfg.AI.LiteLLM
	if !liteCfg.Enabled || liteCfg.BaseURL == "" {
		return nil
	}

	timeout := time.Duration(a.cfg.AI.Generation.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	embProvider := providers.NewOpenAICompat(providers.OpenAICompatConfig{
		Name:         "litellm-embed",
		BaseURL:      liteCfg.BaseURL,
		APIKey:       liteCfg.APIKey,
		DefaultModel: a.cfg.AI.Embedding.Model,
		Timeout:      timeout,
	})

	embedder := rag.NewEmbedder(embProvider, a.cfg.AI.Embedding.Model, a.cfg.RAG.Dimensions)
	store := rag.NewVectorStore(a.db.Conn())
	summarizer := rag.NewSummarizer(embedder, store, a.router, storyID, a.cfg.RAG.SummarizeEvery)
	return rag.NewRAG(embedder, store, summarizer, storyID, a.cfg.RAG.TopK)
}

// loadSaveAndResume restores state from a save and resumes the narrative view.
func (a *App) loadSaveAndResume(storyID, saveID string) (tea.Cmd, error) {
	char, world, err := engine.LoadGame(a.db, saveID)
	if err != nil {
		return nil, fmt.Errorf("loading save: %w", err)
	}

	story, err := a.db.GetStory(storyID)
	if err != nil {
		return nil, fmt.Errorf("loading story: %w", err)
	}

	// Close existing session before opening new one.
	if a.narrative != nil {
		a.narrative.CloseSession()
	}

	session, err := engine.NewGameSession(a.db, storyID, a.cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("creating game session: %w", err)
	}

	_ = a.db.UpdateStoryTimestamp(storyID)

	narrator := engine.NewNarrator(
		a.router, a.db, story, char, world, session,
		engine.DefaultContextConfig(),
		a.cfg.DataDir,
		a.cfg.Game.AutosaveEvery,
	)
	narrator.SetRAG(a.buildRAG(storyID))
	m := views.NewNarrativeModel(narrator, a.cfg.Game.TypewriterSpeed)
	m.SetSize(a.width, a.height)
	a.narrative = &m
	a.view = ViewNarrative

	return a.narrative.StartNarration(), nil
}

// enterNarrativeView loads story data, creates a narrator, and starts narration.
func (a *App) enterNarrativeView(storyID string) (tea.Cmd, error) {
	story, err := a.db.GetStory(storyID)
	if err != nil {
		return nil, fmt.Errorf("loading story: %w", err)
	}

	char, err := a.db.GetCharacterByStory(storyID)
	if err != nil {
		return nil, fmt.Errorf("loading character: %w", err)
	}

	world, err := a.db.GetWorldState(storyID)
	if err != nil {
		return nil, fmt.Errorf("loading world state: %w", err)
	}

	session, err := engine.NewGameSession(a.db, storyID, a.cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("creating game session: %w", err)
	}

	// Update story timestamp to mark it as recently played.
	_ = a.db.UpdateStoryTimestamp(storyID)

	narrator := engine.NewNarrator(
		a.router, a.db, story, char, world, session,
		engine.DefaultContextConfig(),
		a.cfg.DataDir,
		a.cfg.Game.AutosaveEvery,
	)
	narrator.SetRAG(a.buildRAG(storyID))
	m := views.NewNarrativeModel(narrator, a.cfg.Game.TypewriterSpeed)
	m.SetSize(a.width, a.height)
	a.narrative = &m
	a.view = ViewNarrative

	return a.narrative.StartNarration(), nil
}
