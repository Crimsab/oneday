package views

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/storage"
	"github.com/crimsab/oneday/internal/tui/components"
	"github.com/crimsab/oneday/internal/tui/theme"
)

// narrativeResponseMsg carries a parsed AI narrative response.
type narrativeResponseMsg struct {
	response *engine.NarrativeResponse
	err      error
	instant  bool
}

type narrativeStreamStartedMsg struct {
	stream <-chan engine.NarrativeStreamChunk
	err    error
}

type narrativeStreamChunkMsg struct {
	chunk engine.NarrativeStreamChunk
}

type narrativeASCIIArtMsg struct {
	sceneID int
	art     string
	model   string
	err     error
}

// narratorMetaResponseMsg carries a /narrator command response.
type narratorMetaResponseMsg struct {
	title   string
	message string
	err     error
	overlay bool
}

// clearStatusMsg is sent to clear the temporary status message.
type clearStatusMsg struct{}

// SaveCompleteMsg is sent when a manual save finishes.
type SaveCompleteMsg struct {
	Name string
	Err  error
}

// QuitToMenuMsg signals the app to return to the main menu.
type QuitToMenuMsg struct{}

type sessionMenuAction int

const (
	sessionMenuResume sessionMenuAction = iota
	sessionMenuQuickSave
	sessionMenuLoadSave
	sessionMenuQuitToMenu
)

type sessionMenuItem struct {
	Label  string
	Hint   string
	Action sessionMenuAction
}

// NarrativeModel is the core gameplay view.
type NarrativeModel struct {
	narrator                *engine.Narrator
	viewport                viewport.Model
	typewriter              components.TypewriterModel
	statusBar               components.StatusBarModel
	choices                 components.ChoiceListModel
	slashSuggestions        components.SuggestionListModel
	overlay                 components.OverlayModel
	historyBrowser          *historyBrowserModel
	achievementBrowser      *AchievementBrowserModel
	frontTracker            *FrontTrackerModel
	investigationBrowser    *InvestigationBrowserModel
	projectBrowser          *ProjectBrowserModel
	codexBrowser            *CodexBrowserModel
	achievementPopup        components.AchievementPopupModel
	input                   textarea.Model
	history                 *strings.Builder // full narrative text accumulated so far
	pendingNarrative        string
	queuedNarrative         []queuedNarrativeSegment
	deferredChoiceItems     []components.ChoiceItem
	deferredChoiceHelp      map[int]string
	deferredInputFocus      bool
	deferredChallenges      []*engine.ChallengeSpec
	deferredSocialDuel      *engine.SocialDuelCue
	streamRaw               *strings.Builder // raw streamed JSON chunks for the current turn
	streaming               bool
	streamCh                <-chan engine.NarrativeStreamChunk
	waiting                 bool // waiting for AI response
	errMsg                  string
	statusMsg               string    // temporary status message (e.g. "Autosaved")
	statusExpiry            time.Time // when to clear the status message
	width                   int
	height                  int
	inputFocus              bool // true = free input active, false = choice list active
	historyReturnInputFocus bool
	sessionMenuVisible      bool
	sessionMenuCursor       int
	currentMood             string
	sceneCounter            int
	choiceHelp              map[int]string
	talkTarget              string
	talkIntent              string
	inputHistory            []string
	inputHistoryCursor      int
	inputHistoryDraft       string
	visiblePrivateThoughts  bool

	// Combat sub-view
	combatView *CombatModel
	inCombat   bool

	// Crafting sub-view
	craftingView *CraftingModel
	inCrafting   bool

	// Challenge sub-view
	challengeView     *ChallengeView
	inChallenge       bool
	pendingChallenges []*engine.ChallengeSpec // queue of challenges to resolve
	pendingChallenge  *engine.ChallengeSpec

	// Social duel sub-view
	socialDuelView      *SocialDuelView
	inSocialDuel        bool
	pendingSocialDuel   *engine.SocialDuelCue
	activeSocialDuel    *engine.SocialDuelState
	activeSocialDuelCue *engine.SocialDuelCue
	socialDuelNPC       *storage.NPC

	// Achievement queue — holds achievements earned while the popup is already visible.
	// Dequeued one-by-one as each popup is dismissed.
	pendingAchievements []storage.Achievement
}

// NewNarrativeModel creates the narrative view.
func NewNarrativeModel(narrator *engine.Narrator, typewriterSpeed int, visiblePrivateThoughts bool) NarrativeModel {
	ta := newGameTextarea("Type a free action or press 1-4 to choose...", actionInputHeight)

	vp := viewport.New(80, 20)

	return NarrativeModel{
		narrator:               narrator,
		viewport:               vp,
		typewriter:             components.NewTypewriter(typewriterSpeed),
		statusBar:              components.NewStatusBar(),
		choices:                components.NewChoiceList(),
		slashSuggestions:       components.NewSuggestionList(),
		overlay:                components.NewOverlay(),
		achievementPopup:       components.NewAchievementPopup(),
		input:                  ta,
		history:                &strings.Builder{},
		streamRaw:              &strings.Builder{},
		inputFocus:             false, // start on choice list
		currentMood:            "neutral",
		choiceHelp:             map[int]string{},
		inputHistoryCursor:     -1,
		visiblePrivateThoughts: visiblePrivateThoughts,
	}
}

func (m NarrativeModel) Init() tea.Cmd {
	return textarea.Blink
}

// StartNarration kicks off the first AI turn. Only call for brand-new stories.
func (m *NarrativeModel) StartNarration() tea.Cmd {
	m.waiting = true
	return func() tea.Msg {
		stream, err := m.narrator.StartNarrationStream(context.Background())
		return narrativeStreamStartedMsg{stream: stream, err: err}
	}
}

// ResumeNarration restores the narrator state for an existing story without
// sending a first-turn AI prompt. Use this when loading or resuming a story.
func (m *NarrativeModel) ResumeNarration() tea.Cmd {
	m.waiting = true
	return func() tea.Msg {
		resp, err := m.narrator.ResumeNarration(context.Background())
		return narrativeResponseMsg{response: resp, err: err, instant: true}
	}
}

func (m NarrativeModel) Update(msg tea.Msg) (NarrativeModel, tea.Cmd) {
	var cmds []tea.Cmd

	// --- Delegate to combat sub-view when in combat ---
	if m.inCombat && m.combatView != nil {
		switch endMsg := msg.(type) {
		case CombatEndMsg:
			m.inCombat = false
			m.combatView = nil
			var cmds []tea.Cmd
			if endMsg.PersistErr != nil {
				m.statusMsg = "Combat summary not saved"
				m.statusExpiry = time.Now().Add(3 * time.Second)
				cmds = append(cmds, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
					return clearStatusMsg{}
				}))
			}
			// Append combat summary to narrative history.
			summary := components.RenderMarkdown("\n---\n**[Riepilogo Combattimento]** " + endMsg.Summary + "\n---\n")
			if cmd := m.appendNarrativeSegment(summary, false); cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		default:
			updated, cmd := m.combatView.Update(msg)
			m.combatView = &updated
			return m, cmd
		}
	}

	// --- Delegate to crafting sub-view when crafting ---
	if m.inCrafting && m.craftingView != nil {
		switch endMsg := msg.(type) {
		case CraftingEndMsg:
			m.inCrafting = false
			m.craftingView = nil
			if endMsg.ItemCrafted {
				note := components.RenderMarkdown(fmt.Sprintf("\n*[Craftato: %s]*\n", endMsg.ItemName))
				cmd := m.appendNarrativeSegment(note, false)
				return m, cmd
			}
			return m, nil
		default:
			updated, cmd := m.craftingView.Update(msg)
			m.craftingView = &updated
			return m, cmd
		}
	}

	// --- Delegate to challenge sub-view when in challenge ---
	if m.inChallenge && m.challengeView != nil {
		updated, cmd := m.challengeView.Update(msg)
		m.challengeView = updated
		if crMsg, ok := msg.(ChallengeResolvedMsg); ok {
			m.inChallenge = false
			m.challengeView = nil
			m.choices.SetChoices(nil)
			m.choiceHelp = map[int]string{}
			m.inputFocus = false
			m.input.Blur()

			// Show brief result note in narrative history.
			var resultNote string
			if crMsg.Result.Passed {
				resultNote = fmt.Sprintf("\n*[✓ %s]*\n", crMsg.Result.Detail)
			} else {
				resultNote = fmt.Sprintf("\n*[✗ %s]*\n", crMsg.Result.Detail)
			}
			histCmd := m.appendNarrativeSegment(components.RenderMarkdown(resultNote), false)
			cmds = append(cmds, histCmd)

			// Send result to AI for narrative continuation.
			outcome := "PASSED"
			if !crMsg.Result.Passed {
				outcome = "FAILED"
			}
			resultMsg := fmt.Sprintf("[Challenge Result: %s %s — %s]",
				crMsg.Spec.Type, outcome, crMsg.Result.Detail)
			m.waiting = true
			cmds = append(cmds, func() tea.Msg {
				resp, err := m.narrator.SendAction(context.Background(), resultMsg)
				return narrativeResponseMsg{response: resp, err: err}
			})

			// Start next pending challenge if any.
			if nextCmd := m.startNextChallenge(); nextCmd != nil {
				return m, nextCmd
			}

			return m, tea.Batch(cmds...)
		}
		return m, cmd
	}

	// --- Delegate to social-duel sub-view when in a duel exchange ---
	if m.inSocialDuel && m.socialDuelView != nil {
		updated, cmd := m.socialDuelView.Update(msg)
		m.socialDuelView = updated
		if duelMsg, ok := msg.(socialDuelRoundResolvedMsg); ok {
			m.inSocialDuel = false
			m.socialDuelView = nil
			m.pendingSocialDuel = nil
			aftermath := engine.ApplySocialDuelAftermath(
				m.narrator.DB(),
				m.narrator.World(),
				m.socialDuelNPC,
				duelMsg.State,
				duelMsg.Round,
				duelMsg.Cue,
				m.narrator.Turn(),
			)

			if duelMsg.State != nil && duelMsg.State.Status == engine.SocialDuelActive {
				m.activeSocialDuel = duelMsg.State
			} else if duelMsg.Round != nil && duelMsg.Round.Resolved {
				m.clearSocialDuelRuntime()
			}

			var cmds []tea.Cmd
			histCmd := m.appendNarrativeSegment(renderSocialDuelRoundNote(duelMsg, aftermath), false)
			cmds = append(cmds, histCmd)
			cmds = append(cmds, m.sendRawAction(buildSocialDuelResultInput(duelMsg, aftermath)))
			return m, tea.Batch(cmds...)
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.overlay.SetSize(msg.Width, msg.Height)
		if m.historyBrowser != nil {
			m.historyBrowser.SetSize(msg.Width, msg.Height)
		}
		if m.achievementBrowser != nil {
			m.achievementBrowser.SetSize(msg.Width, msg.Height)
		}
		if m.frontTracker != nil {
			m.frontTracker.SetSize(msg.Width, msg.Height)
		}
		if m.investigationBrowser != nil {
			m.investigationBrowser.SetSize(msg.Width, msg.Height)
		}
		if m.projectBrowser != nil {
			m.projectBrowser.SetSize(msg.Width, msg.Height)
		}
		if m.codexBrowser != nil {
			m.codexBrowser.SetSize(msg.Width, msg.Height)
		}
		m.relayout()
		return m, nil

	case narrativeStreamStartedMsg:
		if msg.err != nil {
			m.waiting = false
			m.streaming = false
			m.errMsg = fmt.Sprintf("AI Error: %v", msg.err)
			return m, nil
		}
		m.streaming = true
		m.streamCh = msg.stream
		m.streamRaw.Reset()
		return m, waitNarrativeStreamChunk(m.streamCh)

	case narrativeStreamChunkMsg:
		if msg.chunk.Err != nil {
			m.waiting = false
			m.streaming = false
			m.streamCh = nil
			m.errMsg = fmt.Sprintf("AI Error: %v", msg.chunk.Err)
			return m, nil
		}
		if msg.chunk.Delta != "" {
			m.streamRaw.WriteString(msg.chunk.Delta)
			m.renderStreamingNarrative()
			return m, waitNarrativeStreamChunk(m.streamCh)
		}
		if msg.chunk.Done {
			m.streaming = false
			m.streamCh = nil
			if msg.chunk.Response == nil {
				m.waiting = false
				return m, nil
			}
			return m, m.applyNarrativeResponse(msg.chunk.Response, true)
		}
		return m, waitNarrativeStreamChunk(m.streamCh)

	case narrativeResponseMsg:
		if msg.err != nil {
			m.waiting = false
			m.errMsg = fmt.Sprintf("AI Error: %v", msg.err)
			return m, nil
		}
		return m, m.applyNarrativeResponse(msg.response, msg.instant)

	case components.AchievementDismissedMsg:
		// Achievement popup dismissed — show next queued achievement if any.
		if len(m.pendingAchievements) > 0 {
			next := m.pendingAchievements[0]
			m.pendingAchievements = m.pendingAchievements[1:]
			m.achievementPopup.Show(next.Name, next.Description, next.Rarity, next.Category)
			return m, components.AchievementAutoDismissCmd(m.achievementPopup.Generation())
		}
		return m, nil

	case narratorMetaResponseMsg:
		m.waiting = false
		if msg.err != nil {
			m.errMsg = fmt.Sprintf("Narrator Error: %v", msg.err)
			return m, nil
		}
		m.errMsg = ""
		if msg.overlay {
			title := strings.TrimSpace(msg.title)
			if title == "" {
				title = "Aside"
			}
			m.showOverlay(title, msg.message)
			return m, nil
		}
		title := strings.TrimSpace(msg.title)
		if title == "" {
			title = "Game Master"
		}
		rendered := components.RenderMarkdown("\n**[" + title + "]** " + msg.message + "\n")
		cmd := m.appendNarrativeSegment(rendered+"\n", false)
		// Restore input focus (narrator command does not change choices).
		m.inputFocus = true
		m.input.Focus()
		return m, cmd

	case engine.AutosaveCompleteMsg:
		if msg.Err == nil {
			m.statusMsg = "Autosaved"
		} else {
			m.statusMsg = "Autosave failed"
		}
		m.statusExpiry = time.Now().Add(2 * time.Second)
		return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return clearStatusMsg{}
		})

	case SaveCompleteMsg:
		if msg.Err == nil {
			m.statusMsg = fmt.Sprintf("Saved: %s", msg.Name)
		} else {
			m.errMsg = fmt.Sprintf("Save failed: %v", msg.Err)
		}
		m.statusExpiry = time.Now().Add(3 * time.Second)
		return m, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
			return clearStatusMsg{}
		})

	case clearStatusMsg:
		if time.Now().After(m.statusExpiry) {
			m.statusMsg = ""
		}
		return m, nil

	case components.TypewriterDoneMsg:
		return m, m.completePendingNarrative()

	case components.ChoiceSelectedMsg:
		if m.waiting {
			return m, nil
		}
		return m, m.sendAction(fmt.Sprintf("[Choice %d] %s", msg.ID, msg.Text))

	case components.ChoiceInspectRequestedMsg:
		if help := strings.TrimSpace(m.choiceHelp[msg.ID]); help != "" {
			m.showOverlay("Choice Insight", help)
			return m, nil
		}
		m.statusMsg = "No extra info for this choice."
		m.statusExpiry = time.Now().Add(2 * time.Second)
		return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return clearStatusMsg{}
		})

	case components.SuggestionAcceptedMsg:
		m.resetInputHistoryNavigation()
		m.input.SetValue(msg.Item.Value)
		m.input.Focus()
		m.inputFocus = true
		m.refreshSlashSuggestions()
		return m, nil

	case components.SuggestionDismissedMsg:
		m.slashSuggestions.SetItems(nil)
		return m, nil

	case narrativeASCIIArtMsg:
		if msg.sceneID != m.sceneCounter {
			return m, nil
		}
		if msg.err != nil {
			m.SetStatusMsg("ASCII art unavailable for this scene")
			return m, nil
		}
		if strings.TrimSpace(msg.art) == "" {
			return m, nil
		}
		rendered := components.RenderMarkdown("```text\n" + msg.art + "\n```")
		return m, m.appendNarrativeSegment(rendered+"\n", false)

	case components.OverlayDismissedMsg:
		// Restore input focus after overlay closes
		if m.choices.HasChoices() {
			m.inputFocus = false
			m.input.Blur()
		} else {
			m.inputFocus = true
			m.input.Focus()
		}
		return m, nil

	case tea.MouseMsg:
		if m.achievementBrowser != nil && m.achievementBrowser.Visible() {
			updated, cmd := m.achievementBrowser.Update(msg)
			if !updated.Visible() {
				m.achievementBrowser = nil
				m.restoreInputFocusAfterHistory()
				return m, nil
			}
			m.achievementBrowser = &updated
			return m, cmd
		}
		if m.frontTracker != nil && m.frontTracker.Visible() {
			updated, cmd := m.frontTracker.Update(msg)
			if !updated.Visible() {
				m.frontTracker = nil
				m.restoreInputFocusAfterHistory()
				return m, nil
			}
			m.frontTracker = &updated
			return m, cmd
		}
		if m.investigationBrowser != nil && m.investigationBrowser.Visible() {
			updated, cmd := m.investigationBrowser.Update(msg)
			if !updated.Visible() {
				m.investigationBrowser = nil
				m.restoreInputFocusAfterHistory()
				return m, nil
			}
			m.investigationBrowser = &updated
			return m, cmd
		}
		if m.projectBrowser != nil && m.projectBrowser.Visible() {
			updated, cmd := m.projectBrowser.Update(msg)
			if !updated.Visible() {
				m.projectBrowser = nil
				m.restoreInputFocusAfterHistory()
				return m, nil
			}
			m.projectBrowser = &updated
			return m, cmd
		}
		if m.codexBrowser != nil && m.codexBrowser.Visible() {
			updated, cmd := m.codexBrowser.Update(msg)
			if !updated.Visible() {
				m.codexBrowser = nil
				m.restoreInputFocusAfterHistory()
				return m, nil
			}
			m.codexBrowser = &updated
			return m, cmd
		}
		if m.historyBrowser != nil && m.historyBrowser.Visible() {
			var cmd tea.Cmd
			updated, cmd := m.historyBrowser.Update(msg)
			if !updated.Visible() {
				m.historyBrowser = nil
				m.restoreInputFocusAfterHistory()
				return m, nil
			}
			m.historyBrowser = &updated
			return m, cmd
		}
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		// If achievement popup is visible, route key events to it first.
		if m.achievementPopup.Visible() {
			var cmd tea.Cmd
			m.achievementPopup, cmd = m.achievementPopup.Update(msg)
			return m, cmd
		}

		if m.achievementBrowser != nil && m.achievementBrowser.Visible() {
			updated, cmd := m.achievementBrowser.Update(msg)
			if !updated.Visible() {
				m.achievementBrowser = nil
				m.restoreInputFocusAfterHistory()
				return m, nil
			}
			m.achievementBrowser = &updated
			return m, cmd
		}

		if m.frontTracker != nil && m.frontTracker.Visible() {
			if next, cmd, handled := m.handleActiveSystemShortcut(msg); handled {
				return next, cmd
			}
			updated, cmd := m.frontTracker.Update(msg)
			if !updated.Visible() {
				m.frontTracker = nil
				m.restoreInputFocusAfterHistory()
				return m, nil
			}
			m.frontTracker = &updated
			return m, cmd
		}

		if m.investigationBrowser != nil && m.investigationBrowser.Visible() {
			if next, cmd, handled := m.handleActiveSystemShortcut(msg); handled {
				return next, cmd
			}
			updated, cmd := m.investigationBrowser.Update(msg)
			if !updated.Visible() {
				m.investigationBrowser = nil
				m.restoreInputFocusAfterHistory()
				return m, nil
			}
			m.investigationBrowser = &updated
			return m, cmd
		}

		if m.projectBrowser != nil && m.projectBrowser.Visible() {
			if next, cmd, handled := m.handleActiveSystemShortcut(msg); handled {
				return next, cmd
			}
			updated, cmd := m.projectBrowser.Update(msg)
			if !updated.Visible() {
				m.projectBrowser = nil
				m.restoreInputFocusAfterHistory()
				return m, nil
			}
			m.projectBrowser = &updated
			return m, cmd
		}

		if m.codexBrowser != nil && m.codexBrowser.Visible() {
			if next, cmd, handled := m.handleActiveSystemShortcut(msg); handled {
				return next, cmd
			}
			updated, cmd := m.codexBrowser.Update(msg)
			if !updated.Visible() {
				m.codexBrowser = nil
				m.restoreInputFocusAfterHistory()
				return m, nil
			}
			m.codexBrowser = &updated
			return m, cmd
		}

		if m.historyBrowser != nil && m.historyBrowser.Visible() {
			var cmd tea.Cmd
			updated, cmd := m.historyBrowser.Update(msg)
			if !updated.Visible() {
				m.historyBrowser = nil
				m.restoreInputFocusAfterHistory()
				return m, nil
			}
			m.historyBrowser = &updated
			return m, cmd
		}

		// If overlay is visible, route all key events to it first.
		if m.overlay.Visible() {
			var cmd tea.Cmd
			m.overlay, cmd = m.overlay.Update(msg)
			return m, cmd
		}

		if m.sessionMenuVisible {
			return m.handleSessionMenu(msg)
		}

		if m.scenePlaybackActive() {
			switch msg.String() {
			case "enter", " ":
				return m, m.skipCurrentPlayback()
			case "esc":
				m.sessionMenuVisible = true
				m.sessionMenuCursor = 0
				return m, nil
			}
			return m, nil
		}

		if m.pendingChallenge != nil {
			switch msg.String() {
			case "enter", " ":
				return m, m.beginPendingChallenge()
			}
			return m, nil
		}

		if m.pendingSocialDuel != nil {
			switch msg.String() {
			case "enter", " ":
				return m, m.beginPendingSocialDuel()
			}
			return m, nil
		}

		if m.waiting {
			switch msg.String() {
			case "ctrl+@", "ctrl+space":
				if m.inputFocus && m.talkModeActive() {
					m.closeTalkMode("Talk mode closed")
					return m, nil
				}
			case "tab":
				m.inputFocus = !m.inputFocus
				if m.inputFocus {
					m.input.Focus()
				} else {
					m.input.Blur()
				}
				return m, nil
			case "esc":
				m.sessionMenuVisible = true
				m.sessionMenuCursor = 0
				return m, nil
			}
			if m.inputFocus && msg.String() != "enter" {
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
			return m, nil
		}

		switch msg.String() {
		case "h":
			if !m.inputFocus {
				return m.showHistory(nil)
			}

		case "ctrl+@", "ctrl+space":
			if m.inputFocus && m.talkModeActive() {
				m.closeTalkMode("Talk mode closed")
				return m, nil
			}

		case "tab":
			if m.inputFocus && m.slashSuggestions.HasItems() {
				var cmd tea.Cmd
				m.slashSuggestions, cmd = m.slashSuggestions.Update(msg)
				return m, cmd
			}
			// Toggle between choice list and free input
			m.inputFocus = !m.inputFocus
			if m.inputFocus {
				m.input.Focus()
				m.refreshSlashSuggestions()
			} else {
				m.input.Blur()
				m.slashSuggestions.SetItems(nil)
			}
			return m, nil

		case "enter":
			if m.inputFocus && m.slashSuggestions.HasItems() && m.slashSuggestions.Focused() {
				var cmd tea.Cmd
				m.slashSuggestions, cmd = m.slashSuggestions.Update(msg)
				return m, cmd
			}
			if m.inputFocus {
				text := strings.TrimSpace(m.input.Value())
				if text == "" {
					return m, nil
				}
				m.recordInputHistory(text)
				m.input.Reset()
				m.refreshSlashSuggestions()
				if engine.IsCommand(text) {
					cmd := m.parseCommand(text)
					return m.handleCommand(cmd)
				}
				return m, m.sendAction(text)
			}
		// If on choices, let choice list handle it below

		case "s", "S", "f5":
			if !m.inputFocus {
				return m, m.doQuickSave()
			}

		case "esc":
			if m.inputFocus && m.slashSuggestions.HasItems() {
				var cmd tea.Cmd
				m.slashSuggestions, cmd = m.slashSuggestions.Update(msg)
				return m, cmd
			}
			m.sessionMenuVisible = true
			m.sessionMenuCursor = 0
			return m, nil
		}

		if m.inputFocus && m.canNavigateInputHistory() {
			switch msg.String() {
			case "up":
				if m.navigateInputHistory(-1) {
					m.refreshSlashSuggestions()
					return m, nil
				}
			case "down":
				if m.navigateInputHistory(1) {
					m.refreshSlashSuggestions()
					return m, nil
				}
			}
		}

		// Route key to active component
		if m.inputFocus {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
			switch msg.String() {
			case "up", "down":
			default:
				m.resetInputHistoryNavigation()
			}
			m.refreshSlashSuggestions()
		} else {
			var cmd tea.Cmd
			m.choices, cmd = m.choices.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Always update typewriter
	var twCmd tea.Cmd
	m.typewriter, twCmd = m.typewriter.Update(msg)
	if twCmd != nil {
		cmds = append(cmds, twCmd)
		// Update viewport content as typewriter progresses
		m.viewport.SetContent(m.visibleNarrativeContent())
		m.viewport.GotoBottom()
	}

	// Update viewport scrolling
	shouldUpdateViewport := true
	if keyMsg, ok := msg.(tea.KeyMsg); ok && !m.inputFocus && m.choices.HasChoices() {
		switch keyMsg.String() {
		case "up", "down", "left", "right", "j", "k", "h", "l", "enter", " ", "?":
			shouldUpdateViewport = false
		default:
			if len(keyMsg.String()) == 1 {
				ch := keyMsg.String()[0]
				if ch >= '1' && ch <= '9' {
					shouldUpdateViewport = false
				}
			}
		}
	}
	if _, ok := msg.(tea.MouseMsg); ok {
		shouldUpdateViewport = false
	}
	if shouldUpdateViewport {
		var vpCmd tea.Cmd
		m.viewport, vpCmd = m.viewport.Update(msg)
		cmds = append(cmds, vpCmd)
	}

	return m, tea.Batch(cmds...)
}

// handleCommand dispatches a parsed command to the appropriate handler.
func (m NarrativeModel) handleCommand(cmd *engine.Command) (NarrativeModel, tea.Cmd) {
	switch cmd.Name {
	case "inventory":
		return m.showInventory()
	case "stats":
		return m.showStats()
	case "save":
		return m.doSave(cmd.Args)
	case "load":
		return m.doLoad()
	case "help":
		return m.showHelp()
	case "quit":
		return m.doQuit()
	case "narrator":
		if len(cmd.Args) == 0 {
			m.errMsg = "Usage: /n <message to the game master>"
			return m, nil
		}
		input := strings.Join(cmd.Args, " ")
		return m.sendNarratorCommand(input)
	case "btw":
		if len(cmd.Args) == 0 {
			m.errMsg = "Usage: /btw <quick question about the current story>"
			return m, nil
		}
		return m.sendAsideQuestion(strings.Join(cmd.Args, " "))
	case "guide":
		if len(cmd.Args) == 0 {
			m.errMsg = "Usage: /guide <future beat or chapter wish>"
			return m, nil
		}
		return m.sendGuideCommand(strings.Join(cmd.Args, " "))
	case "advance":
		return m.handleAdvanceCommand(cmd.Args)
	case "timeskip":
		return m.handleTimeSkipCommand(cmd.Args)
	case "journal":
		return m.showJournal()
	case "thoughts":
		return m.showThoughts()
	case "history":
		return m.showHistory(cmd.Args)
	case "hooks":
		return m.showHooks()
	case "map":
		return m.showMap()
	case "achievements":
		return m.showAchievements()
	case "characters":
		return m.showCharacters()
	case "codex":
		return m.showCodex()
	case "investigations":
		return m.showInvestigations()
	case "projects":
		return m.showProjects()
	case "craft", "crafting":
		return m.startCrafting()
	case "talk":
		return m.handleTalkCommand(cmd.Args)
	case "downtime":
		return m.handleDowntimeCommand(cmd.Args)
	default:
		name := strings.TrimSpace(cmd.Name)
		if name == "" || name == "unknown" {
			if len(cmd.Args) > 0 {
				name = strings.TrimSpace(cmd.Args[0])
			}
		}
		if name != "" && name != "unknown" {
			m.errMsg = fmt.Sprintf("Unknown command: /%s. Type /help for available commands.", name)
		} else {
			m.errMsg = "Unknown command. Type /help for available commands."
		}
		return m, nil
	}
}

func (m NarrativeModel) parseCommand(input string) *engine.Command {
	if !engine.IsCommand(input) {
		return nil
	}

	trimmed := strings.TrimSpace(input)
	trimmed = strings.TrimPrefix(trimmed, "/")
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return &engine.Command{Name: "unknown", Args: []string{"/"}}
	}

	name := strings.ToLower(strings.TrimSpace(parts[0]))
	args := parts[1:]
	if name == "thoughts" && m.visiblePrivateThoughts {
		return &engine.Command{Name: "thoughts", Args: args}
	}

	return engine.ParseCommand(input)
}

// showOverlay is a helper to show the overlay with title and content.
func (m *NarrativeModel) showOverlay(title, content string) {
	m.overlay.SetSize(m.width, m.height)
	m.overlay.Show(title, content)
	// Blur inputs so overlay handles all keys
	m.input.Blur()
}

// sendNarratorCommand sends a /narrator meta-command to the AI game master.
// Does not increment the turn counter.
func (m NarrativeModel) sendNarratorCommand(input string) (NarrativeModel, tea.Cmd) {
	m.waiting = true
	m.statusMsg = "Speaking to the narrator..."
	m.statusExpiry = time.Now().Add(30 * time.Second)
	narrator := m.narrator
	return m, func() tea.Msg {
		resp, err := narrator.ExecuteNarratorCommand(context.Background(), input)
		if err != nil {
			return narratorMetaResponseMsg{err: err}
		}
		return narratorMetaResponseMsg{title: "Game Master", message: resp.Message}
	}
}

func (m NarrativeModel) sendAsideQuestion(input string) (NarrativeModel, tea.Cmd) {
	m.waiting = true
	m.statusMsg = "Asking a quick aside..."
	m.statusExpiry = time.Now().Add(30 * time.Second)
	narrator := m.narrator
	return m, func() tea.Msg {
		resp, err := narrator.ExecuteAsideQuestion(context.Background(), input)
		if err != nil {
			return narratorMetaResponseMsg{err: err}
		}
		return narratorMetaResponseMsg{
			title:   "By The Way",
			message: resp,
			overlay: true,
		}
	}
}

func (m NarrativeModel) sendGuideCommand(input string) (NarrativeModel, tea.Cmd) {
	m.waiting = true
	m.statusMsg = "Saving story guidance..."
	m.statusExpiry = time.Now().Add(30 * time.Second)
	narrator := m.narrator
	return m, func() tea.Msg {
		resp, err := narrator.ExecuteGuideCommand(context.Background(), input)
		if err != nil {
			return narratorMetaResponseMsg{err: err}
		}
		return narratorMetaResponseMsg{title: "Guide", message: resp.Message}
	}
}

func (m NarrativeModel) handleAdvanceCommand(args []string) (NarrativeModel, tea.Cmd) {
	m.statusMsg = "Sto mandando avanti la scena..."
	m.statusExpiry = time.Now().Add(3 * time.Second)
	return m, m.sendRawAction(buildAdvanceSceneAction(strings.Join(args, " ")))
}

func (m NarrativeModel) handleTimeSkipCommand(args []string) (NarrativeModel, tea.Cmd) {
	m.statusMsg = "Sto saltando al prossimo momento importante..."
	m.statusExpiry = time.Now().Add(3 * time.Second)
	return m, m.sendRawAction(buildTimeSkipAction(strings.Join(args, " ")))
}

func buildAdvanceSceneAction(hint string) string {
	base := "[Advance Scene] Move to the next meaningful beat now. If this micro-scene is exhausted, do not replay it with near-identical prose or choices. Introduce a concrete change: reveal, consequence, interruption, pressure, location shift, or a natural time skip."
	if hint = strings.TrimSpace(hint); hint != "" {
		base += " Treat any extra text after this tag as the player's desired timing, destination, or arrival point for the next beat. Requested timing or destination: " + hint
	}
	return base
}

func buildTimeSkipAction(hint string) string {
	base := "[Time Skip] Jump forward to a later meaningful moment instead of playing filler turn by turn. Keep continuity clear: show what changed, what stayed true, and why this later beat matters now. If exact age is unclear, use a life stage or milestone rather than inventing a precise number."
	if hint = strings.TrimSpace(hint); hint != "" {
		base += " Treat any extra text after this tag as the player's preferred arrival point, approximate age, or target moment. Requested destination: " + hint
	}
	return base
}

// showJournal displays the chapter journal overlay.
func (m NarrativeModel) showJournal() (NarrativeModel, tea.Cmd) {
	world := m.narrator.World()
	storyID := m.narrator.Story().ID
	currentChapter := 1
	currentTurn := m.narrator.Turn()
	if world != nil {
		currentChapter = world.CurrentChapter
		currentTurn = world.CurrentTurn
	}
	journalText := engine.FormatJournalView(m.narrator.DB(), storyID, currentChapter, currentTurn)
	m.showOverlay("Journal", journalText)
	return m, nil
}

func (m NarrativeModel) showThoughts() (NarrativeModel, tea.Cmd) {
	if !m.visiblePrivateThoughts {
		m.errMsg = "Unknown command: /thoughts. Type /help for available commands."
		return m, nil
	}
	if m.narrator == nil || m.narrator.DB() == nil || m.narrator.Story() == nil {
		m.errMsg = "Private thoughts unavailable right now."
		return m, nil
	}
	thoughtsText := engine.FormatPrivateThoughtsView(m.narrator.DB(), m.narrator.Story().ID)
	m.showOverlay("NPC Private Thoughts", thoughtsText)
	return m, nil
}

// showMap displays the discovered world locations overlay.
func (m NarrativeModel) showMap() (NarrativeModel, tea.Cmd) {
	mapText := engine.FormatMapView(m.narrator.World())
	m.showOverlay("World Map", mapText)
	return m, nil
}

// showAchievements displays earned achievements overlay.
func (m NarrativeModel) showAchievements() (NarrativeModel, tea.Cmd) {
	storyID := m.narrator.Story().ID
	archive, err := engine.BuildStoryArchiveSummary(m.narrator.DB(), storyID)
	if err != nil {
		m.errMsg = fmt.Sprintf("Achievements unavailable: %v", err)
		return m, nil
	}
	browser := NewSingleStoryAchievementBrowser(*archive, m.width, m.height)
	m.achievementBrowser = &browser
	m.historyReturnInputFocus = m.inputFocus
	m.inputFocus = false
	m.input.Blur()
	return m, nil
}

// showHelp displays the help overlay.
// startCrafting opens a crafting conversation session.
func (m NarrativeModel) startCrafting() (NarrativeModel, tea.Cmd) {
	craftingEngine, err := engine.NewCraftingEngine(m.narrator)
	if err != nil {
		m.errMsg = fmt.Sprintf("Cannot start crafting: %v", err)
		return m, nil
	}
	craftingView := NewCraftingModel(craftingEngine, m.narrator, m.width, m.height)
	m.craftingView = &craftingView
	m.inCrafting = true
	return m, nil
}

// startNextChallenge launches the first challenge in pendingChallenges queue.
// Returns nil if the queue is empty.
func (m *NarrativeModel) startNextChallenge() tea.Cmd {
	if len(m.pendingChallenges) == 0 {
		return nil
	}
	spec := m.pendingChallenges[0]
	m.pendingChallenges = m.pendingChallenges[1:]
	if activeChallengeRequiresConfirm(spec) {
		m.pendingChallenge = spec
		return nil
	}
	return m.launchChallenge(spec)
}

func (m NarrativeModel) showHelp() (NarrativeModel, tea.Cmd) {
	lines := []string{
		"Available Commands:",
		"",
		"  /inventory    (/i)   Show your inventory",
		"  /stats        (/s)   Open the protagonist dossier",
		"  /characters          Browse protagonist and NPC dossiers",
		"  /codex               Open the full story codex",
		"  /fronts              Open the fronts and fallout tracker",
		"  /hooks               Alias for /fronts",
		"  /investigations      Open the dedicated investigation workspace",
		"  /projects            Open the dedicated project workspace",
		"  /map          (/m)   Show discovered world map",
		"  /journal      (/j)   Show chapter journal",
		"  h                   Open the story history browser",
		"  /btw <question>     Ask the AI a quick contextual question without advancing the turn",
		"  /guide <request>    Store a soft future-facing chapter wish without advancing the turn",
		"  /advance [hint]     Move to the next meaningful beat now; free text after it is a soft target",
		"  /timeskip [hint]    Jump ahead to a later meaningful moment; free text sets the target moment",
	}
	if m.visiblePrivateThoughts {
		lines = append(lines, "  /thoughts            Inspect saved NPC private thoughts")
	}
	lines = append(lines,
		"  /achievements (/a)   Show earned achievements",
		"  /narrator     (/n)   Speak to the game master",
		"  /craft               Open crafting station (AI-driven)",
		"  /talk [npc] [intent] Enter nearby-NPC talk mode or send a one-shot line",
		"  /downtime [focus]    Request a quieter downtime beat",
		"  /save [name]         Save your game",
		"  /load                Load a saved game",
		"  /help                Show this help",
		"  /quit         (/q)   Save and quit to menu",
		"",
		"Keyboard Shortcuts:",
		"  s / F5              Quick save snapshot",
		"  h                   Open searchable history browser",
		"  P / I / F / C       Jump between projects, investigations, fronts, and codex",
		"  Up / Down           Browse free-input history (single-line input)",
		"  Mouse wheel         Scroll the current scene",
		"  Ctrl+Space          Close talk mode instantly",
		"  Esc                 Open session menu (resume, quick save, load, main menu)",
		"  Space               Confirms the highlighted option in menus and pickers",
		"  Left / Right        Focus metadata badges on the selected choice",
		"  ?                   Explain the selected choice's related stats/metadata",
		"",
		"Narrator examples:",
		"  /n Add a secret underground city",
		"  /n Make Lyanna secretly jealous",
		"  /n What factions exist in this world?",
		"  /n I want the next area to be a haunted forest",
		"",
		"Guide examples:",
		"  /guide In questo capitolo voglio una boss fight memorabile",
		"  /guide Fammi trovare materiali rari e almeno un reward forte",
		"  /guide Semina una scena tesa con Lyanna, ma falla arrivare quando ha senso",
		"",
		"Pacing examples:",
		"  /advance",
		"  /advance Vai oltre questa scena domestica e portami al primo vero cambiamento",
		"  /advance una settimana dopo",
		"  /advance la mattina seguente, quando torno al mercato",
		"  /timeskip",
		"  /timeskip arrivo a 6 anni",
		"  /timeskip Tre anni dopo, quando la magia e gia parte della routine",
		"  /timeskip al prossimo inverno",
		"  /timeskip Alla prossima tappa davvero importante del viaggio",
	)
	if m.visiblePrivateThoughts {
		lines = append(lines,
			"",
			"Private thoughts:",
			"  /thoughts",
		)
	}
	lines = append(lines,
		"",
		"Talk mode:",
		"  /talk Lyanna",
		"  /talk Lyanna promise",
		"  /talk Lyanna ask What did you see at the docks?",
		"  /talk off",
		"",
		"Downtime examples:",
		"  /downtime rest by the fire",
		"  /downtime write a letter home",
		"  /downtime train with Lyanna",
		"",
		"Challenges:",
		"  Active challenges pause first on a confirmation screen.",
		"  Types: dice roll (d100), rock-paper-scissors, memory sequence,",
		"         quick-time (press key in time), riddle, stat/skill/item checks.",
		"  The game engine resolves the outcome fairly — the AI then narrates the result.",
		"",
		"  Footer legend:",
		"  10.4s         total response time",
		"  ft 5.5s       time to first token",
		"  6016t         total tokens",
		"  4980p/1036c   prompt/completion tokens",
		"  r193          reasoning tokens",
		"  cache 900p    cached prompt tokens",
		"  99.7t/s       completion throughput",
		"",
		"Quick aside:",
		"  /btw Who exactly is Dee Podale Suprema?",
		"  /btw Remind me what this faction wants",
	)

	helpText := strings.Join(lines, "\n")

	m.showOverlay("Help", helpText)
	return m, nil
}

// itemTypeIcon returns a unicode icon for an item type.
func itemTypeIcon(itemType string) string {
	switch strings.ToLower(itemType) {
	case "weapon":
		return "⚔"
	case "armor":
		return "◇"
	case "tool":
		return "⚒"
	case "consumable":
		return "◈"
	case "quest", "key_item":
		return "◆"
	default:
		return "•"
	}
}

// inventoryItemInfo extracts name, type icon, slot, and description from an item (string or map).
type inventoryItemInfo struct {
	name        string
	icon        string
	slot        string
	description string
	rarity      string
	weight      float64
	hasWeight   bool
}

func parseInventoryItem(item interface{}) inventoryItemInfo {
	switch v := item.(type) {
	case string:
		return inventoryItemInfo{name: v, icon: "•"}
	case map[string]interface{}:
		info := inventoryItemInfo{}
		if n, ok := v["name"].(string); ok {
			info.name = n
		}
		itemType := ""
		if t, ok := v["type"].(string); ok {
			itemType = t
		}
		info.icon = itemTypeIcon(itemType)
		if s, ok := v["slot"].(string); ok {
			info.slot = s
		}
		if d, ok := v["description"].(string); ok {
			info.description = d
		}
		if r, ok := v["rarity"].(string); ok {
			info.rarity = r
		}
		if w, ok := v["weight"]; ok {
			info.weight = toFloat64(w)
			info.hasWeight = true
		}
		return info
	}
	return inventoryItemInfo{name: fmt.Sprintf("%v", item), icon: "•"}
}

// toFloat64 converts interface{} to float64.
func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	}
	return 0
}

// showInventory displays the character's inventory as an overlay.
func (m NarrativeModel) showInventory() (NarrativeModel, tea.Cmd) {
	char := m.narrator.Character()
	var sb strings.Builder

	sb.WriteString("\n")

	// Parse inventory JSON from the dedicated InventoryJSON column.
	// This is the canonical source; stats_json no longer stores inventory.
	var inventoryRaw interface{}
	if char.InventoryJSON != "" && char.InventoryJSON != "null" {
		_ = json.Unmarshal([]byte(char.InventoryJSON), &inventoryRaw)
	}

	// Parse stats for currency only.
	var stats map[string]interface{}
	if char.StatsJSON != "" {
		_ = json.Unmarshal([]byte(char.StatsJSON), &stats)
	}

	// Try to extract inventory sections.
	var backpackItems []inventoryItemInfo
	var equippedItems []inventoryItemInfo
	var questItems []inventoryItemInfo

	switch inv := inventoryRaw.(type) {
	case []interface{}:
		for _, item := range inv {
			info := parseInventoryItem(item)
			if m2, ok := item.(map[string]interface{}); ok {
				if eq, _ := m2["equipped"].(bool); eq {
					equippedItems = append(equippedItems, info)
					continue
				}
				if qt, _ := m2["quest"].(bool); qt {
					questItems = append(questItems, info)
					continue
				}
				if info.icon == "◆" {
					questItems = append(questItems, info)
					continue
				}
			}
			backpackItems = append(backpackItems, info)
		}
	case map[string]interface{}:
		if bp, ok := inv["backpack"].([]interface{}); ok {
			for _, item := range bp {
				backpackItems = append(backpackItems, parseInventoryItem(item))
			}
		}
		if eq, ok := inv["equipped"].([]interface{}); ok {
			for _, item := range eq {
				equippedItems = append(equippedItems, parseInventoryItem(item))
			}
		}
		if qt, ok := inv["quest"].([]interface{}); ok {
			for _, item := range qt {
				questItems = append(questItems, parseInventoryItem(item))
			}
		}
	}

	// Currency setup.
	currency := 0
	currencyName := "Gold"
	if stats != nil {
		if c, ok := stats["currency"]; ok {
			currency = toInt(c)
		}
	}
	// Try to get currency name from story stats schema.
	story := m.narrator.Story()
	if story != nil && story.StatsSchemaJSON != "" {
		var schema map[string]interface{}
		if err := json.Unmarshal([]byte(story.StatsSchemaJSON), &schema); err == nil {
			if cur, ok := schema["currency"].(map[string]interface{}); ok {
				if name, ok := cur["name"].(string); ok && name != "" {
					currencyName = name
				}
			}
		}
	}

	// Completely empty inventory: show friendly message.
	totalItems := len(backpackItems) + len(equippedItems) + len(questItems)
	if totalItems == 0 && currency == 0 {
		sb.WriteString(theme.MutedText.Render("  You carry nothing. The journey begins light."))
		sb.WriteString("\n")
		m.showOverlay("Inventory", sb.String())
		return m, nil
	}

	// Equipped section.
	if len(equippedItems) > 0 {
		sb.WriteString(theme.Title.Render("Equipped") + "\n")
		for _, info := range equippedItems {
			if info.slot != "" {
				slotLabel := strings.ReplaceAll(strings.Title(strings.ReplaceAll(info.slot, "_", " ")), " ", " ")
				sb.WriteString(fmt.Sprintf("  %-12s %s %s\n", slotLabel+":", info.icon, info.name))
			} else {
				sb.WriteString(fmt.Sprintf("  %s %s\n", info.icon, info.name))
			}
			if info.description != "" {
				sb.WriteString(theme.MutedText.Render(fmt.Sprintf("      %s", info.description)) + "\n")
			}
		}
		sb.WriteString("\n")
	}

	// Backpack section.
	if len(backpackItems) > 0 {
		sb.WriteString(theme.Title.Render(fmt.Sprintf("Backpack (%d items)", len(backpackItems))) + "\n")
		// Calculate total weight if any items have it.
		totalWeight := 0.0
		hasWeight := false
		for _, info := range backpackItems {
			if info.hasWeight {
				totalWeight += info.weight
				hasWeight = true
			}
		}
		for _, info := range backpackItems {
			sb.WriteString(fmt.Sprintf("  %s %s", info.icon, info.name))
			if info.rarity != "" {
				sb.WriteString(" " + theme.MutedText.Render(fmt.Sprintf("(%s)", info.rarity)))
			}
			sb.WriteString("\n")
			if info.description != "" {
				sb.WriteString(theme.MutedText.Render(fmt.Sprintf("      %s", info.description)) + "\n")
			}
		}
		if hasWeight {
			sb.WriteString(theme.MutedText.Render(fmt.Sprintf("  Weight: %.0f", totalWeight)) + "\n")
		}
		sb.WriteString("\n")
	}

	// Quest items section.
	if len(questItems) > 0 {
		sb.WriteString(theme.Title.Render("Quest Items") + "\n")
		for _, info := range questItems {
			sb.WriteString(fmt.Sprintf("  ◆ %s\n", info.name))
			if info.description != "" {
				sb.WriteString(theme.MutedText.Render(fmt.Sprintf("      %s", info.description)) + "\n")
			}
		}
		sb.WriteString("\n")
	}

	// Currency.
	sb.WriteString(theme.Title.Render("Currency") + "\n")
	sb.WriteString(fmt.Sprintf("  %d %s\n", currency, currencyName))

	m.showOverlay("Inventory", sb.String())
	return m, nil
}

// renderBar renders a progress bar of the given width using block characters.
// filled = █, empty = ░. current and max define the fill ratio.
func renderBar(current, max, width int) string {
	if max <= 0 || width <= 0 {
		return strings.Repeat("░", width)
	}
	filled := current * width / max
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

// dispositionLabel returns a word label for an NPC disposition value.
func dispositionLabel(d int) string {
	switch {
	case d >= 50:
		return "Allied"
	case d >= 15:
		return "Friendly"
	case d >= -14:
		return "Neutral"
	case d >= -49:
		return "Unfriendly"
	default:
		return "Hostile"
	}
}

// showStats opens the protagonist dossier inside the codex browser.
func (m NarrativeModel) showStats() (NarrativeModel, tea.Cmd) {
	return m.openCodexBrowser("Character Sheet", "people", engine.ProtagonistCodexEntryID())
}

func (m NarrativeModel) showCharacters() (NarrativeModel, tea.Cmd) {
	return m.openCodexBrowser("Characters", "people", "")
}

func (m NarrativeModel) showCodex() (NarrativeModel, tea.Cmd) {
	return m.openCodexBrowser("Codex", "", "")
}

func (m NarrativeModel) showInvestigations() (NarrativeModel, tea.Cmd) {
	if m.narrator == nil || m.narrator.Story() == nil {
		m.errMsg = "Investigations unavailable: no active story."
		return m, nil
	}

	browser := NewInvestigationBrowserModel("Investigations", engine.LoadInvestigationBoard(m.narrator.World()), m.width, m.height)
	m.investigationBrowser = &browser
	m.historyReturnInputFocus = m.inputFocus
	m.inputFocus = false
	m.input.Blur()
	return m, nil
}

func (m NarrativeModel) showProjects() (NarrativeModel, tea.Cmd) {
	if m.narrator == nil || m.narrator.Story() == nil {
		m.errMsg = "Projects unavailable: no active story."
		return m, nil
	}

	browser := NewProjectBrowserModel("Projects", engine.LoadProjectBoard(m.narrator.World()), m.width, m.height)
	m.projectBrowser = &browser
	m.historyReturnInputFocus = m.inputFocus
	m.inputFocus = false
	m.input.Blur()
	return m, nil
}

func (m NarrativeModel) openCodexBrowser(title, initialCategory, initialEntryID string) (NarrativeModel, tea.Cmd) {
	if m.narrator == nil || m.narrator.DB() == nil || m.narrator.Story() == nil {
		m.errMsg = "Codex unavailable: no active story."
		return m, nil
	}

	index, err := engine.BuildStoryCodexByID(m.narrator.DB(), m.narrator.Story().ID)
	if err != nil {
		m.errMsg = fmt.Sprintf("Codex unavailable: %v", err)
		return m, nil
	}

	m.codexBrowser = NewCodexBrowserModel(title, index, m.width, m.height, initialCategory, initialEntryID)
	m.historyReturnInputFocus = m.inputFocus
	m.inputFocus = false
	m.input.Blur()
	return m, nil
}

func (m NarrativeModel) handleActiveSystemShortcut(msg tea.KeyMsg) (NarrativeModel, tea.Cmd, bool) {
	key := strings.ToLower(strings.TrimSpace(msg.String()))
	switch key {
	case "p":
		if m.projectBrowser != nil && m.projectBrowser.Visible() {
			return m, nil, true
		}
		return m.openActiveSystemWorkspace("projects", ""), nil, true
	case "i":
		if m.investigationBrowser != nil && m.investigationBrowser.Visible() {
			return m, nil, true
		}
		return m.openActiveSystemWorkspace("investigations", ""), nil, true
	case "f":
		if m.frontTracker != nil && m.frontTracker.Visible() {
			return m, nil, true
		}
		return m.openActiveSystemWorkspace("fronts", ""), nil, true
	case "c":
		if m.codexBrowser != nil && m.codexBrowser.Visible() {
			return m, nil, true
		}
		return m.openActiveSystemWorkspace("codex", m.activeSystemCodexCategory()), nil, true
	default:
		return m, nil, false
	}
}

func (m NarrativeModel) activeSystemCodexCategory() string {
	switch {
	case m.projectBrowser != nil && m.projectBrowser.Visible():
		return "projects"
	case m.investigationBrowser != nil && m.investigationBrowser.Visible():
		return "investigations"
	case m.frontTracker != nil && m.frontTracker.Visible():
		return "fronts"
	default:
		return ""
	}
}

func (m NarrativeModel) openActiveSystemWorkspace(target, codexCategory string) NarrativeModel {
	m.frontTracker = nil
	m.investigationBrowser = nil
	m.projectBrowser = nil
	m.codexBrowser = nil

	switch target {
	case "projects":
		updated, _ := m.showProjects()
		return updated
	case "investigations":
		updated, _ := m.showInvestigations()
		return updated
	case "fronts":
		updated, _ := m.showHooks()
		return updated
	case "codex":
		updated, _ := m.openCodexBrowser("Codex", codexCategory, "")
		return updated
	default:
		return m
	}
}

// doSave triggers a named manual save.
func (m NarrativeModel) doSave(args []string) (NarrativeModel, tea.Cmd) {
	saveName := "Manual Save"
	if len(args) > 0 {
		saveName = strings.Join(args, " ")
	}
	narrator := m.narrator
	return m, func() tea.Msg {
		_, err := engine.SaveGameWithMetadata(
			narrator.DB(), narrator.DataDir(),
			narrator.Story(), narrator.Character(), narrator.World(),
			narrator.SessionID(), saveName, narrator.BuildSaveMetadata("manual"),
		)
		return SaveCompleteMsg{Name: saveName, Err: err}
	}
}

func (m NarrativeModel) doQuickSave() tea.Cmd {
	saveName := fmt.Sprintf("Quicksave T%d", m.narrator.Turn())
	narrator := m.narrator
	return func() tea.Msg {
		_, err := engine.SaveGameWithMetadata(
			narrator.DB(), narrator.DataDir(),
			narrator.Story(), narrator.Character(), narrator.World(),
			narrator.SessionID(), saveName, narrator.BuildSaveMetadata("quicksave"),
		)
		return SaveCompleteMsg{Name: saveName, Err: err}
	}
}

// doLoad shows the save list overlay (triggers a view switch in the app via SaveLoadMsg).
func (m NarrativeModel) doLoad() (NarrativeModel, tea.Cmd) {
	narrator := m.narrator
	return m, func() tea.Msg {
		saves, err := engine.ListSaves(narrator.DB(), narrator.Story().ID)
		if err != nil || len(saves) == 0 {
			return ShowSaveListMsg{Saves: nil}
		}
		return ShowSaveListMsg{Saves: saves}
	}
}

// ShowSaveListMsg carries the list of saves to display in a picker.
type ShowSaveListMsg struct {
	Saves []storage.SaveSnapshot
}

// doQuit autosaves and signals return to menu.
func (m NarrativeModel) doQuit() (NarrativeModel, tea.Cmd) {
	narrator := m.narrator
	return m, func() tea.Msg {
		_ = engine.AutosaveWithMetadata(
			narrator.DB(), narrator.DataDir(),
			narrator.Story(), narrator.Character(), narrator.World(),
			narrator.SessionID(), narrator.BuildSaveMetadata("autosave"),
		)
		narrator.CloseSession()
		return QuitToMenuMsg{}
	}
}

func (m *NarrativeModel) sendAction(action string) tea.Cmd {
	return m.sendRawAction(m.wrapPlayerAction(action))
}

func (m *NarrativeModel) sendRawAction(action string) tea.Cmd {
	m.waiting = true
	m.choices.SetChoices(nil) // clear choices while waiting for AI
	narrator := m.narrator
	action = strings.TrimSpace(action)
	return func() tea.Msg {
		stream, err := narrator.StreamAction(context.Background(), action)
		return narrativeStreamStartedMsg{stream: stream, err: err}
	}
}

// maybeAutosaveCmd returns an autosave tea.Cmd if the narrator says it's time,
// or nil if not. Called after a successful narrativeResponseMsg.
func (m *NarrativeModel) maybeAutosaveCmd() tea.Cmd {
	if m.narrator.ShouldAutosave() {
		return m.narrator.AutosaveCmd()
	}
	return nil
}

func waitNarrativeStreamChunk(stream <-chan engine.NarrativeStreamChunk) tea.Cmd {
	return func() tea.Msg {
		if stream == nil {
			return narrativeStreamChunkMsg{chunk: engine.NarrativeStreamChunk{Done: true}}
		}
		chunk, ok := <-stream
		if !ok {
			return narrativeStreamChunkMsg{chunk: engine.NarrativeStreamChunk{Done: true}}
		}
		return narrativeStreamChunkMsg{chunk: chunk}
	}
}

func (m *NarrativeModel) applyNarrativeResponse(nr *engine.NarrativeResponse, streamed bool) tea.Cmd {
	m.waiting = false
	m.errMsg = ""
	m.sceneCounter++

	m.updateStatusBar()

	if nr.Mood != "" {
		m.currentMood = nr.Mood
		m.statusBar.SetMoodColor(theme.GetMoodPalette(nr.Mood).StatusBarBG)
	}

	rendered := m.renderNarrativeResponse(nr)
	if strings.TrimSpace(rendered) == "" {
		rendered = components.RenderMarkdown(nr.Narrative)
	}
	if callout := turnDeltaStatusCallout(nr.TurnDelta); callout != "" {
		m.SetStatusMsg(callout)
	}
	m.streamRaw.Reset()
	choiceItems, choiceHelp := m.buildChoicePresentation(nr.Choices)
	duelCue := m.socialDuelCueForTurn(nr.SocialDuel)
	duelPending := duelCue != nil
	challengePending := len(nr.Challenges) > 0
	animateScene := !streamed && strings.TrimSpace(rendered) != ""
	m.choices.SetMood(m.currentMood)
	if animateScene {
		if duelPending {
			m.deferredChoiceItems = nil
			m.deferredChoiceHelp = nil
			m.deferredInputFocus = false
			m.deferredChallenges = nil
			m.deferredSocialDuel = duelCue
		} else if challengePending {
			m.deferredChoiceItems = nil
			m.deferredChoiceHelp = nil
			m.deferredInputFocus = false
			m.deferredChallenges = nr.Challenges
			m.deferredSocialDuel = nil
		} else {
			m.deferredChoiceItems = choiceItems
			m.deferredChoiceHelp = choiceHelp
			m.deferredInputFocus = len(nr.Choices) == 0
			m.deferredChallenges = nr.Challenges
			m.deferredSocialDuel = nil
		}
		m.choices.SetChoices(nil)
		m.choiceHelp = map[int]string{}
		m.pendingChallenges = nil
		m.pendingChallenge = nil
		m.pendingSocialDuel = nil
		m.inputFocus = false
		m.input.Blur()
	} else {
		m.deferredChoiceItems = nil
		m.deferredChoiceHelp = nil
		m.deferredChallenges = nil
		m.deferredSocialDuel = nil
		if duelPending {
			m.choiceHelp = map[int]string{}
			m.choices.SetChoices(nil)
			m.pendingChallenges = nil
			m.pendingChallenge = nil
			m.pendingSocialDuel = duelCue
			m.inputFocus = false
			m.input.Blur()
		} else if challengePending {
			m.choiceHelp = map[int]string{}
			m.choices.SetChoices(nil)
			m.pendingSocialDuel = nil
			m.pendingChallenges = nr.Challenges
			m.pendingChallenge = nil
			m.inputFocus = false
			m.input.Blur()
		} else {
			m.choiceHelp = choiceHelp
			m.choices.SetChoices(choiceItems)
			m.pendingSocialDuel = nil
			if len(nr.Choices) > 0 {
				m.inputFocus = false
				m.input.Blur()
			} else {
				m.inputFocus = true
				m.input.Focus()
			}
		}
	}

	var cmds []tea.Cmd
	if cmd := m.appendNarrativeSegment(rendered+"\n", animateScene); cmd != nil {
		cmds = append(cmds, cmd)
	}

	if nr.CombatStart != nil {
		combatEngine, err := engine.NewCombatEngine(m.narrator, nr.CombatStart)
		if err == nil {
			combatView := NewCombatModel(combatEngine, m.narrator, m.width, m.height)
			m.combatView = &combatView
			m.inCombat = true
			return nil
		}
		m.errMsg = fmt.Sprintf("Could not start combat: %v", err)
	}

	if !animateScene && !duelPending && len(nr.Challenges) > 0 {
		m.pendingChallenges = nr.Challenges
		if nextCmd := m.startNextChallenge(); nextCmd != nil {
			cmds = append(cmds, nextCmd)
			return tea.Batch(cmds...)
		}
	}

	if nr.PersistedAchievement != nil {
		if m.achievementPopup.Visible() {
			m.pendingAchievements = append(m.pendingAchievements, *nr.PersistedAchievement)
		} else {
			m.achievementPopup.Show(
				nr.PersistedAchievement.Name,
				nr.PersistedAchievement.Description,
				nr.PersistedAchievement.Rarity,
				nr.PersistedAchievement.Category,
			)
			cmds = append(cmds, components.AchievementAutoDismissCmd(m.achievementPopup.Generation()))
		}
	}

	if autosaveCmd := m.maybeAutosaveCmd(); autosaveCmd != nil {
		cmds = append(cmds, autosaveCmd)
	}
	if asciiCmd := m.requestAmbientASCII(m.sceneCounter, m.narrator.Turn()-1, nr); asciiCmd != nil {
		cmds = append(cmds, asciiCmd)
	}

	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m NarrativeModel) requestAmbientASCII(sceneID, turn int, nr *engine.NarrativeResponse) tea.Cmd {
	if nr == nil || nr.ASCIICue == nil || strings.TrimSpace(nr.ASCIIArt) != "" || !m.narrator.ASCIIArtEnabled() {
		return nil
	}

	snapshot := *nr
	narrator := m.narrator
	return func() tea.Msg {
		art, model, err := narrator.GenerateAmbientASCII(context.Background(), turn, &snapshot)
		return narrativeASCIIArtMsg{sceneID: sceneID, art: art, model: model, err: err}
	}
}

func (m *NarrativeModel) renderStreamingNarrative() {
	partial := extractStreamingNarrative(m.streamRaw.String())
	if strings.TrimSpace(partial) == "" {
		return
	}
	content := m.visibleNarrativeContent() + partial
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

func extractStreamingNarrative(raw string) string {
	const key = `"narrative"`
	start := strings.Index(raw, key)
	if start == -1 {
		return ""
	}

	i := start + len(key)
	for i < len(raw) && (raw[i] == ' ' || raw[i] == '\n' || raw[i] == '\t' || raw[i] == '\r') {
		i++
	}
	if i >= len(raw) || raw[i] != ':' {
		return ""
	}
	i++
	for i < len(raw) && (raw[i] == ' ' || raw[i] == '\n' || raw[i] == '\t' || raw[i] == '\r') {
		i++
	}
	if i >= len(raw) || raw[i] != '"' {
		return ""
	}
	i++

	var out strings.Builder
	escaped := false
	for i < len(raw) {
		ch := raw[i]
		if escaped {
			switch ch {
			case 'n':
				out.WriteByte('\n')
			case 'r':
				out.WriteByte('\r')
			case 't':
				out.WriteByte('\t')
			case '"':
				out.WriteByte('"')
			case '\\':
				out.WriteByte('\\')
			default:
				out.WriteByte(ch)
			}
			escaped = false
			i++
			continue
		}
		if ch == '\\' {
			escaped = true
			i++
			continue
		}
		if ch == '"' {
			break
		}
		out.WriteByte(ch)
		i++
	}

	return out.String()
}

// updateStatusBar reads the character's stats JSON and populates the status bar.
func (m *NarrativeModel) updateStatusBar() {
	var stats map[string]interface{}
	_ = json.Unmarshal([]byte(m.narrator.Character().StatsJSON), &stats)

	var vitals []components.Vital
	if vitalsMap, ok := stats["vitals"].(map[string]interface{}); ok {
		keys := make([]string, 0, len(vitalsMap))
		for key := range vitalsMap {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			leftRank := statusVitalRank(keys[i])
			rightRank := statusVitalRank(keys[j])
			if leftRank == rightRank {
				return keys[i] < keys[j]
			}
			return leftRank < rightRank
		})
		for _, key := range keys {
			val := vitalsMap[key]
			if vMap, ok := val.(map[string]interface{}); ok {
				current := toInt(vMap["current"])
				max := toInt(vMap["max"])
				vitals = append(vitals, components.Vital{
					Label:   strings.ToUpper(key),
					Current: current,
					Max:     max,
				})
			}
		}
	}

	m.statusBar.SetData(components.StatusBarData{
		Vitals:             vitals,
		Model:              m.narrator.LastModel(),
		Latency:            m.narrator.LastLatency(),
		TimeToFirstToken:   m.narrator.LastTimeToFirstToken(),
		PromptTokens:       m.narrator.LastUsage().PromptTokens,
		CompletionTokens:   m.narrator.LastUsage().CompletionTokens,
		ReasoningTokens:    m.narrator.LastUsage().ReasoningTokens,
		TotalTokens:        m.narrator.LastUsage().TotalTokens,
		CachedPromptTokens: m.narrator.LastUsage().CachedPromptTokens,
		CostUSD:            m.narrator.LastUsage().CostUSD,
		Streamed:           m.narrator.LastStreamed(),
	})
}

func statusVitalRank(key string) int {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "energy":
		return 0
	case "hp", "health":
		return 1
	case "stamina":
		return 2
	case "mana":
		return 3
	default:
		return 10
	}
}

// toInt safely converts interface{} to int (handles float64 from JSON).
func toInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return 0
}

func (m *NarrativeModel) applyInputCommandStyle() {
	if m == nil {
		return
	}
	textColor := theme.Text
	promptColor := theme.Secondary
	value := strings.TrimSpace(m.input.Value())
	if strings.HasPrefix(value, "/") {
		if cmd := m.parseCommand(value); cmd != nil && cmd.Name != "unknown" {
			textColor = theme.Success
			promptColor = theme.Success
		} else if m.slashSuggestions.HasItems() {
			textColor = theme.Secondary
			promptColor = theme.Secondary
		} else {
			textColor = theme.Danger
			promptColor = theme.Danger
		}
	}
	baseStyle := lipgloss.NewStyle().Foreground(textColor)
	m.input.FocusedStyle.Text = lipgloss.NewStyle().Foreground(textColor)
	m.input.BlurredStyle.Text = lipgloss.NewStyle().Foreground(textColor)
	m.input.FocusedStyle.Base = baseStyle
	m.input.BlurredStyle.Base = baseStyle
	m.input.FocusedStyle.CursorLine = m.input.FocusedStyle.CursorLine.Foreground(textColor)
	m.input.BlurredStyle.CursorLine = m.input.BlurredStyle.CursorLine.Foreground(textColor)
	m.input.FocusedStyle.CursorLineNumber = m.input.FocusedStyle.CursorLineNumber.Foreground(textColor)
	m.input.BlurredStyle.CursorLineNumber = m.input.BlurredStyle.CursorLineNumber.Foreground(textColor)
	m.input.FocusedStyle.LineNumber = m.input.FocusedStyle.LineNumber.Foreground(textColor)
	m.input.BlurredStyle.LineNumber = m.input.BlurredStyle.LineNumber.Foreground(textColor)
	m.input.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(promptColor).Bold(true)
	m.input.BlurredStyle.Prompt = lipgloss.NewStyle().Foreground(promptColor)
	if m.inputFocus {
		m.input.Focus()
	} else {
		m.input.Blur()
	}
}

func (m NarrativeModel) View() string {
	// Delegate to combat sub-view when in combat.
	if m.inCombat && m.combatView != nil {
		return m.combatView.View()
	}

	// Delegate to crafting sub-view when crafting.
	if m.inCrafting && m.craftingView != nil {
		return m.craftingView.View()
	}

	// Delegate to challenge sub-view when in challenge.
	if m.inChallenge && m.challengeView != nil {
		return m.challengeView.View()
	}

	if m.inSocialDuel && m.socialDuelView != nil {
		return m.socialDuelView.View()
	}

	if m.pendingChallenge != nil && !m.scenePlaybackActive() {
		return m.challengePreludeView()
	}

	if m.pendingSocialDuel != nil && !m.scenePlaybackActive() {
		return m.socialDuelPreludeView()
	}

	if m.historyBrowser != nil && m.historyBrowser.Visible() {
		return m.historyBrowser.View()
	}

	if m.achievementBrowser != nil && m.achievementBrowser.Visible() {
		return m.achievementBrowser.View()
	}

	if m.frontTracker != nil && m.frontTracker.Visible() {
		return m.frontTracker.View()
	}

	if m.investigationBrowser != nil && m.investigationBrowser.Visible() {
		return m.investigationBrowser.View()
	}

	if m.projectBrowser != nil && m.projectBrowser.Visible() {
		return m.projectBrowser.View()
	}

	if m.codexBrowser != nil && m.codexBrowser.Visible() {
		return m.codexBrowser.View()
	}

	// If overlay is visible, render it full-screen.
	if m.overlay.Visible() {
		return m.overlay.View()
	}

	// Determine mood palette for theming accents.
	moodPalette := theme.GetMoodPalette(m.currentMood)
	accentStyle := lipgloss.NewStyle().Foreground(moodPalette.Accent)
	subtitleStyle := lipgloss.NewStyle().Foreground(moodPalette.NarrativeAccent)

	// Header: chapter + location
	world := m.narrator.World()
	header := accentStyle.Bold(true).Render(fmt.Sprintf("Chapter %d", world.CurrentChapter))
	if world.CurrentLocation != "" {
		header += "  " + subtitleStyle.Render(world.CurrentLocation)
	}
	if timelineSummary := engine.CharacterTimelineSummary(engine.LoadCharacterTimeline(world)); timelineSummary != "" {
		header += "  " + theme.MutedText.Render(timelineSummary)
	}
	if m.talkModeActive() {
		header += "  " + theme.MutedText.Render(fmt.Sprintf("talking to %s [%s]", m.talkTarget, m.activeTalkIntent()))
	}

	// Narrative viewport
	body := m.viewport.View()

	// Error display
	if m.errMsg != "" {
		body += "\n" + theme.DangerText.Render(m.errMsg)
	}

	// Waiting indicator
	var waitLine string
	if m.waiting {
		if m.streaming {
			waitLine = theme.MutedText.Render("  The narrator is streaming...")
		} else {
			waitLine = theme.MutedText.Render("  The narrator is writing...")
		}
	}

	sceneReady := !m.scenePlaybackActive()

	// Choices
	choicesView := ""
	if sceneReady {
		choicesView = m.choices.View()
	}

	// Input area
	var inputView string
	var slashSuggestionsView string
	if !sceneReady {
		inputView = theme.MutedText.Render("  [Enter/Space] Finish scene")
	} else if m.inputFocus {
		m.applyInputCommandStyle()
		inputView = m.input.View()
		if m.slashSuggestions.HasItems() {
			slashSuggestionsView = m.slashSuggestions.View()
		}
	} else {
		inputView = theme.MutedText.Render("  [TAB] Free input")
	}

	// Temporary status message (e.g. "Autosaved")
	var statusLine string
	if m.statusMsg != "" {
		statusLine = theme.SuccessText.Render("  ✓ " + m.statusMsg)
	}

	// Help line
	help := theme.MutedText.Render("↑/↓ input history · alt+enter/ctrl+j newline · wheel scroll scene · tab complete slash · ctrl+space close talk · s quicksave · esc session")

	// Status bar
	m.statusBar.SetWidth(m.width)
	statusView := m.statusBar.View()

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		body,
		waitLine,
		"",
		choicesView,
		inputView,
		slashSuggestionsView,
		statusLine,
		help,
		statusView,
	)

	// Render session menu on top if visible.
	if m.sessionMenuVisible {
		return m.sessionMenuView(content)
	}

	// Render achievement popup on top if visible.
	if m.achievementPopup.Visible() {
		return m.achievementPopup.View()
	}

	return content
}

// SetStatusMsg sets a temporary status message visible in the narrative view.
func (m *NarrativeModel) SetStatusMsg(msg string) {
	m.statusMsg = msg
	m.statusExpiry = time.Now().Add(3 * time.Second)
}

func (m *NarrativeModel) restoreInputFocusAfterHistory() {
	if m == nil {
		return
	}
	if m.historyReturnInputFocus || !m.choices.HasChoices() {
		m.inputFocus = true
		m.input.Focus()
	} else {
		m.inputFocus = false
		m.input.Blur()
	}
	m.historyReturnInputFocus = false
}

func (m NarrativeModel) handleSessionMenu(msg tea.KeyMsg) (NarrativeModel, tea.Cmd) {
	items := sessionMenuItems()
	switch msg.String() {
	case "up", "k":
		if m.sessionMenuCursor > 0 {
			m.sessionMenuCursor--
		}
		return m, nil
	case "down", "j":
		if m.sessionMenuCursor < len(items)-1 {
			m.sessionMenuCursor++
		}
		return m, nil
	case "esc":
		m.sessionMenuVisible = false
		return m, nil
	case "enter", " ":
		m.sessionMenuVisible = false
		switch items[m.sessionMenuCursor].Action {
		case sessionMenuResume:
			return m, nil
		case sessionMenuQuickSave:
			return m, m.doQuickSave()
		case sessionMenuLoadSave:
			updated, cmd := m.doLoad()
			return updated, cmd
		case sessionMenuQuitToMenu:
			updated, cmd := m.doQuit()
			return updated, cmd
		}
	}
	return m, nil
}

func sessionMenuItems() []sessionMenuItem {
	return []sessionMenuItem{
		{Label: "Resume", Hint: "Return to the current scene", Action: sessionMenuResume},
		{Label: "Quick Save", Hint: "Create a new snapshot right now", Action: sessionMenuQuickSave},
		{Label: "Load Save", Hint: "Open the save picker", Action: sessionMenuLoadSave},
		{Label: "Main Menu", Hint: "Autosave and leave this session", Action: sessionMenuQuitToMenu},
	}
}

func (m NarrativeModel) sessionMenuView(background string) string {
	_ = background
	items := sessionMenuItems()
	lines := []string{theme.Title.Render("Session"), ""}
	for i, item := range items {
		cursor := "  "
		style := theme.UnselectedItem
		if i == m.sessionMenuCursor {
			cursor = "▸ "
			style = theme.SelectedItem
		}
		lines = append(lines, cursor+style.Render(item.Label))
		lines = append(lines, "   "+theme.MutedText.Render(item.Hint))
	}
	lines = append(lines, "", theme.MutedText.Render("↑/↓ navigate · Enter/Space select · Esc resume"))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Secondary).
		Padding(1, 2).
		Width(44).
		Render(strings.Join(lines, "\n"))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// StoryID returns the ID of the current story.
func (m *NarrativeModel) StoryID() string {
	if m.narrator == nil || m.narrator.Story() == nil {
		return ""
	}
	return m.narrator.Story().ID
}

// CloseSession closes the active game session. Safe to call multiple times.
func (m *NarrativeModel) CloseSession() {
	if m.narrator != nil {
		m.narrator.CloseSession()
	}
}

// SetSize updates the view dimensions and re-layouts sub-components.
func (m *NarrativeModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.overlay.SetSize(w, h)
	m.achievementPopup.SetSize(w, h)
	if m.historyBrowser != nil {
		m.historyBrowser.SetSize(w, h)
	}
	if m.achievementBrowser != nil {
		m.achievementBrowser.SetSize(w, h)
	}
	if m.frontTracker != nil {
		m.frontTracker.SetSize(w, h)
	}
	if m.investigationBrowser != nil {
		m.investigationBrowser.SetSize(w, h)
	}
	if m.projectBrowser != nil {
		m.projectBrowser.SetSize(w, h)
	}
	if m.codexBrowser != nil {
		m.codexBrowser.SetSize(w, h)
	}
	m.relayout()
}

func (m *NarrativeModel) relayout() {
	// Reserve: header(1) + gap(1) + choices(~6) + input(2) + help(1) + status(1) + gaps(4) = ~16
	vpHeight := m.height - 16
	if vpHeight < 5 {
		vpHeight = 5
	}
	m.viewport.Width = m.width - 2
	m.viewport.Height = vpHeight
	m.input.SetWidth(m.width - 4)
	m.choices.SetWidth(m.width - 4)
	m.slashSuggestions.SetWidth(m.width - 4)
	m.statusBar.SetWidth(m.width)
}
