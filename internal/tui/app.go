package tui

import (
	"context"
	"fmt"
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/ai/providers"
	"github.com/crimsab/oneday/internal/aifactory"
	audioservice "github.com/crimsab/oneday/internal/audio"
	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/engine"
	appi18n "github.com/crimsab/oneday/internal/i18n"
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
	ViewAchievementArchive
	ViewSaveLoad
	ViewSettings
)

// App is the top-level Bubbletea model.
type App struct {
	cfg        config.Config
	db         *storage.DB
	router     *ai.Router
	view       View
	width      int
	height     int
	loc        appi18n.Localizer
	configPath string

	// Child view models
	menu               views.MenuModel
	newStory           *views.NewStoryModel
	narrative          *views.NarrativeModel
	loadStory          *views.LoadStoryModel
	achievementArchive *views.AchievementBrowserModel
	saveLoad           *views.SaveLoadModel
	settings           *views.SettingsModel
}

// New creates the app with all dependencies.
func New(cfg config.Config, db *storage.DB, router *ai.Router, args ...any) App {
	loc := appi18n.New(appi18n.Resolve(cfg.Interface.Locale, nil))
	configPath := "config.yaml"
	for _, arg := range args {
		switch value := arg.(type) {
		case appi18n.Localizer:
			loc = value
		case string:
			configPath = value
		}
	}
	return App{
		cfg:        cfg,
		db:         db,
		router:     router,
		view:       ViewMenu,
		loc:        loc,
		configPath: configPath,
		menu:       views.NewMenuModel(loc),
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
		if a.achievementArchive != nil {
			a.achievementArchive.SetSize(msg.Width, msg.Height)
		}
		if a.saveLoad != nil {
			a.saveLoad.SetSize(msg.Width, msg.Height)
		}
		if a.settings != nil {
			a.settings.SetSize(msg.Width, msg.Height)
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
		// User selected an existing story from the load list — resume, don't restart.
		cmd, err := a.enterNarrativeViewResume(msg.StoryID)
		if err != nil {
			a.view = ViewMenu
			return a, nil
		}
		return a, cmd

	case views.LoadStoryBackMsg:
		// User pressed Esc in load story view — back to menu
		a.view = ViewMenu
		return a, nil

	case views.AchievementBrowserBackMsg:
		if a.view == ViewAchievementArchive {
			a.view = ViewMenu
		}
		return a, nil

	case views.StoryArchiveToggleMsg:
		if err := engine.SetStoryArchived(a.db, msg.StoryID, msg.Archived); err != nil {
			return a, nil
		}
		if a.loadStory != nil {
			stories, err := a.db.ListStories()
			if err == nil {
				a.loadStory.SetStories(stories)
			}
		}
		return a, nil

	case views.StoryDeleteMsg:
		if err := engine.DeleteStory(a.db, a.cfg.DataDir, msg.StoryID); err != nil {
			return a, nil
		}
		if a.loadStory != nil {
			stories, err := a.db.ListStories()
			if err == nil {
				a.loadStory.SetStories(stories)
			}
		}
		return a, nil

	case views.SettingsBackMsg:
		a.view = ViewMenu
		return a, nil

	case views.SettingsLocaleChangedMsg:
		if err := config.UpdateInterfaceLocale(a.configPath, string(msg.Locale)); err != nil {
			return a, nil
		}
		a.cfg.Interface.Locale = string(msg.Locale)
		a.loc = appi18n.New(msg.Locale)
		a.menu = views.NewMenuModel(a.loc)
		a.menu.SetSize(a.width, a.height)
		m := views.NewSettingsModel(a.cfg, a.loc)
		m.SetSize(a.width, a.height)
		a.settings = &m
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
				a.narrative.SetStatusMsg(a.loc.T("tui.no_saves"))
			}
			return a, nil
		}
		m := views.NewSaveLoadModel(msg.Saves, a.loc)
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

	case views.SaveLoadDeleteMsg:
		if a.narrative == nil {
			return a, nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := engine.WithStoryMutationLease(ctx, a.db, a.narrative.StoryID(), "save-delete", "terminal", func(lease *engine.StoryMutationLease) error {
			if err := lease.Renew(); err != nil {
				return err
			}
			return engine.DeleteSave(a.db, a.cfg.DataDir, msg.SaveID)
		}); err != nil {
			a.narrative.SetStatusMsg(a.loc.T("tui.save_delete_failed"))
			return a, nil
		}
		saves, err := engine.ListSaves(a.db, a.narrative.StoryID())
		if err != nil || len(saves) == 0 {
			a.saveLoad = nil
			a.view = ViewNarrative
		} else if a.saveLoad != nil {
			a.saveLoad.SetSaves(saves)
			a.view = ViewSaveLoad
		}
		a.narrative.SetStatusMsg(a.loc.T("tui.save_deleted"))
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
			creator := engine.NewStoryCreator(a.router, a.db, a.cfg.AI.Generation, a.loc)
			m := views.NewNewStoryModel(creator, a.loc)
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
			m := views.NewLoadStoryModel(stories, a.loc)
			m.SetSize(a.width, a.height)
			a.loadStory = &m
			a.view = ViewLoadStory
			return a, nil
		case views.ActionAchievementArchive:
			archives, err := engine.BuildStoryArchiveSummaries(a.db)
			if err != nil {
				return a, nil
			}
			m := views.NewAchievementBrowserModel(a.loc.CommandPresentation("achievements", "title", "Achievement Archive"), archives, a.width, a.height, a.loc)
			a.achievementArchive = &m
			a.view = ViewAchievementArchive
			return a, nil
		case views.ActionSettings:
			m := views.NewSettingsModel(a.cfg, a.loc)
			m.SetSize(a.width, a.height)
			a.settings = &m
			a.view = ViewSettings
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

	case ViewAchievementArchive:
		if a.achievementArchive != nil {
			var cmd tea.Cmd
			updated, cmd := a.achievementArchive.Update(msg)
			a.achievementArchive = &updated
			return a, cmd
		}

	case ViewSaveLoad:
		if a.saveLoad != nil {
			var cmd tea.Cmd
			updated, cmd := a.saveLoad.Update(msg)
			a.saveLoad = &updated
			return a, cmd
		}

	case ViewSettings:
		if a.settings != nil {
			var cmd tea.Cmd
			updated, cmd := a.settings.Update(msg)
			a.settings = &updated
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
	case ViewAchievementArchive:
		if a.achievementArchive != nil {
			return a.achievementArchive.View()
		}
		return a.menu.View()
	case ViewSaveLoad:
		if a.saveLoad != nil {
			return a.saveLoad.View()
		}
		return a.menu.View()
	case ViewSettings:
		if a.settings != nil {
			return a.settings.View()
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

	spec, reason := aifactory.SelectEmbeddingProvider(a.cfg)
	if reason != "" {
		log.Printf("oneday: RAG: disabled, reason: %s, story: %s", reason, storyID)
		return nil
	}

	timeout := time.Duration(a.cfg.AI.Generation.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	embProvider := buildEmbeddingProvider(spec, timeout)
	log.Printf("oneday: RAG: enabled, embedding provider: %s, model: %s, story: %s", spec.Name, spec.Model, storyID)

	embedder := rag.NewEmbedder(embProvider, spec.Model, spec.Dimensions)
	store := rag.NewVectorStore(a.db.Conn())
	if removed, err := store.PruneDimensionMismatches(context.Background(), storyID, spec.Dimensions); err != nil {
		log.Printf("oneday: RAG: dimension migration failed, story: %s, err: %v", storyID, err)
	} else if removed > 0 {
		log.Printf("oneday: RAG: removed %d stale embedding chunks for dimensions=%d, story: %s", removed, spec.Dimensions, storyID)
	}
	summarizer := rag.NewSummarizer(embedder, store, a.router, storyID, a.cfg.RAG.SummarizeEvery)
	return rag.NewRAG(embedder, store, summarizer, storyID, a.cfg.RAG.TopK)
}

func buildEmbeddingProvider(spec aifactory.EmbeddingProviderSpec, timeout time.Duration) rag.EmbeddingProvider {
	switch spec.Kind {
	case "ollama":
		return providers.NewOllamaEmbedding(providers.OllamaEmbeddingConfig{
			BaseURL: spec.BaseURL,
			Model:   spec.Model,
			Timeout: timeout,
		})
	case "custom":
		return providers.NewLocalHTTPEmbedding(spec.BaseURL, spec.Model, timeout)
	default:
		return providers.NewOpenAICompat(providers.OpenAICompatConfig{
			Name:         spec.Name,
			BaseURL:      spec.BaseURL,
			APIKey:       spec.APIKey,
			DefaultModel: spec.Model,
			Timeout:      timeout,
		})
	}
}

func (a *App) loadNarrativeState(storyID string) (*storage.Story, *storage.Character, *storage.WorldState, error) {
	story, err := a.db.GetStory(storyID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("loading story: %w", err)
	}

	char, err := a.db.GetCharacterByStory(storyID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("loading character: %w", err)
	}

	world, err := a.db.GetWorldState(storyID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("loading world state: %w", err)
	}

	return story, char, world, nil
}

func (a *App) openNarrativeSession(storyID string, closeExisting bool) (*engine.GameSession, error) {
	if closeExisting && a.narrative != nil {
		a.narrative.CloseSession()
	}

	session, err := engine.NewGameSession(a.db, storyID, a.cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("creating game session: %w", err)
	}

	_ = a.db.UpdateStoryTimestamp(storyID)
	return session, nil
}

func (a *App) mountNarrativeView(
	story *storage.Story,
	char *storage.Character,
	world *storage.WorldState,
	session *engine.GameSession,
	save *storage.SaveSnapshot,
	legacy bool,
) {
	contextCfg := engine.DefaultContextConfig()
	contextCfg.RewardBudget = a.cfg.Game.RewardBudget
	narrator := engine.NewNarrator(
		a.router, a.db, story, char, world, session,
		contextCfg,
		a.cfg.AI.Generation,
		a.cfg.AI.ASCIIArt,
		a.cfg.DataDir,
		a.cfg.Game.AutosaveEvery,
	)
	narrator.SetRAG(a.buildRAG(story.ID))
	audio := audioservice.NewService(a.db, a.cfg.AI.TTS)
	_, _ = audio.EnsureConfiguredVoiceProfiles()
	narrator.SetCommittedAudioQueue(func(ctx context.Context, storyID string, messageID int64) error {
		_, err := audio.QueueCommittedMessage(ctx, storyID, messageID)
		return err
	})
	if save != nil {
		narrator.SetLoadedSaveContext(save)
	}

	model := views.NewNarrativeModel(narrator, a.cfg.Game.TypewriterSpeed, a.cfg.Game.VisiblePrivateThoughts, a.loc)
	model.SetSize(a.width, a.height)
	a.narrative = &model
	a.view = ViewNarrative
	if legacy {
		a.narrative.SetStatusMsg(a.loc.T("save.legacy_partial"))
	}
}

func (a *App) startNarrativeFlow(
	story *storage.Story,
	char *storage.Character,
	world *storage.WorldState,
	session *engine.GameSession,
	save *storage.SaveSnapshot,
	legacy bool,
	resume bool,
) (tea.Cmd, error) {
	a.mountNarrativeView(story, char, world, session, save, legacy)
	if resume {
		return a.narrative.ResumeNarration(), nil
	}
	return a.narrative.StartNarration(), nil
}

// loadSaveAndResume restores state from a save and resumes the narrative view.
func (a *App) loadSaveAndResume(storyID, saveID string) (tea.Cmd, error) {
	var loadResult *engine.LoadResult
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := engine.WithStoryMutationLease(ctx, a.db, storyID, "save-load", "terminal", func(lease *engine.StoryMutationLease) error {
		if err := lease.Renew(); err != nil {
			return err
		}
		var err error
		loadResult, err = engine.LoadGame(a.db, a.cfg.DataDir, saveID)
		return err
	}); err != nil {
		return nil, fmt.Errorf("loading save: %w", err)
	}

	story, err := a.db.GetStory(storyID)
	if err != nil {
		return nil, fmt.Errorf("loading story: %w", err)
	}

	session, err := a.openNarrativeSession(storyID, true)
	if err != nil {
		return nil, err
	}

	// Save was restored — resume from where we left off without re-triggering
	// the first-turn AI prompt.
	return a.startNarrativeFlow(
		story,
		loadResult.Character,
		loadResult.World,
		session,
		loadResult.Save,
		loadResult.Legacy,
		true,
	)
}

// enterNarrativeView loads story data, creates a narrator, and starts narration.
// Only use this for brand-new stories. For existing stories use enterNarrativeViewResume.
func (a *App) enterNarrativeView(storyID string) (tea.Cmd, error) {
	story, char, world, err := a.loadNarrativeState(storyID)
	if err != nil {
		return nil, err
	}

	session, err := a.openNarrativeSession(storyID, false)
	if err != nil {
		return nil, err
	}

	return a.startNarrativeFlow(story, char, world, session, nil, false, false)
}

// enterNarrativeViewResume loads an existing story and resumes from the last
// saved turn without sending a first-turn AI prompt.
func (a *App) enterNarrativeViewResume(storyID string) (tea.Cmd, error) {
	story, char, world, err := a.loadNarrativeState(storyID)
	if err != nil {
		return nil, err
	}

	session, err := a.openNarrativeSession(storyID, true)
	if err != nil {
		return nil, err
	}

	// Resume from last turn — world.CurrentTurn is set from DB, session turn
	// will be synced inside ResumeNarration via SetTurn.
	return a.startNarrativeFlow(story, char, world, session, nil, false, true)
}
