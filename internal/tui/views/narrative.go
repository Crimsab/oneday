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
	overlay                 components.OverlayModel
	historyBrowser          *historyBrowserModel
	achievementPopup        components.AchievementPopupModel
	input                   textarea.Model
	history                 *strings.Builder // full narrative text accumulated so far
	pendingNarrative        string
	queuedNarrative         []queuedNarrativeSegment
	deferredChoiceItems     []components.ChoiceItem
	deferredChoiceHelp      map[int]string
	deferredInputFocus      bool
	deferredChallenges      []*engine.ChallengeSpec
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

	// Achievement queue — holds achievements earned while the popup is already visible.
	// Dequeued one-by-one as each popup is dismissed.
	pendingAchievements []storage.Achievement
}

// NewNarrativeModel creates the narrative view.
func NewNarrativeModel(narrator *engine.Narrator, typewriterSpeed int) NarrativeModel {
	ta := textarea.New()
	ta.Placeholder = "Type a free action or press 1-4 to choose..."
	ta.CharLimit = 500
	ta.SetHeight(2)

	vp := viewport.New(80, 20)

	return NarrativeModel{
		narrator:         narrator,
		viewport:         vp,
		typewriter:       components.NewTypewriter(typewriterSpeed),
		statusBar:        components.NewStatusBar(),
		choices:          components.NewChoiceList(),
		overlay:          components.NewOverlay(),
		achievementPopup: components.NewAchievementPopup(),
		input:            ta,
		history:          &strings.Builder{},
		streamRaw:        &strings.Builder{},
		inputFocus:       false, // start on choice list
		currentMood:      "neutral",
		choiceHelp:       map[int]string{},
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

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.overlay.SetSize(msg.Width, msg.Height)
		if m.historyBrowser != nil {
			m.historyBrowser.SetSize(msg.Width, msg.Height)
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
		// Display narrator response styled distinctly (prefixed with [Game Master]).
		rendered := components.RenderMarkdown("\n**[Game Master]** " + msg.message + "\n")
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

	case narrativeASCIIArtMsg:
		if msg.err != nil || msg.sceneID != m.sceneCounter || strings.TrimSpace(msg.art) == "" {
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
		return m, nil

	case tea.KeyMsg:
		// If achievement popup is visible, route key events to it first.
		if m.achievementPopup.Visible() {
			var cmd tea.Cmd
			m.achievementPopup, cmd = m.achievementPopup.Update(msg)
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

		if m.waiting {
			switch msg.String() {
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

		case "tab":
			// Toggle between choice list and free input
			m.inputFocus = !m.inputFocus
			if m.inputFocus {
				m.input.Focus()
			} else {
				m.input.Blur()
			}
			return m, nil

		case "enter":
			if m.inputFocus {
				text := strings.TrimSpace(m.input.Value())
				if text == "" {
					return m, nil
				}
				m.input.Reset()
				if engine.IsCommand(text) {
					cmd := engine.ParseCommand(text)
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
			m.sessionMenuVisible = true
			m.sessionMenuCursor = 0
			return m, nil
		}

		// Route key to active component
		if m.inputFocus {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
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
	case "journal":
		return m.showJournal()
	case "history":
		return m.showHistory(cmd.Args)
	case "hooks":
		return m.showHooks()
	case "map":
		return m.showMap()
	case "achievements":
		return m.showAchievements()
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

// showMap displays the discovered world locations overlay.
func (m NarrativeModel) showMap() (NarrativeModel, tea.Cmd) {
	mapText := engine.FormatMapView(m.narrator.World())
	m.showOverlay("World Map", mapText)
	return m, nil
}

// showAchievements displays earned achievements overlay.
func (m NarrativeModel) showAchievements() (NarrativeModel, tea.Cmd) {
	storyID := m.narrator.Story().ID
	achText := engine.FormatAchievementsView(m.narrator.DB(), storyID)
	m.showOverlay("Achievements", achText)
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
	helpText := `Available Commands:

  /inventory    (/i)   Show your inventory
  /stats        (/s)   Show character sheet
  /map          (/m)   Show discovered world map
  /journal      (/j)   Show chapter journal
  /hooks               Show open hooks and world reactions
  h                   Open the story history browser
  /btw <question>     Ask the AI a quick contextual question without advancing the turn
  /achievements (/a)   Show earned achievements
  /narrator     (/n)   Speak to the game master
  /craft               Open crafting station (AI-driven)
  /talk [npc] [intent] Enter nearby-NPC talk mode
  /downtime [focus]    Request a quieter downtime beat
  /save [name]         Save your game
  /load                Load a saved game
  /help                Show this help
  /quit         (/q)   Save and quit to menu

Keyboard Shortcuts:
  s / F5              Quick save snapshot
  h                   Open searchable history browser
  Esc                 Open session menu (resume, quick save, load, main menu)
  Space               Confirms the highlighted option in menus and pickers
  Left / Right        Focus metadata badges on the selected choice
  ?                   Explain the selected choice's related stats/metadata

Narrator examples:
  /n Add a secret underground city
  /n Make Lyanna secretly jealous
  /n What factions exist in this world?
  /n I want the next area to be a haunted forest

Talk mode:
  /talk Lyanna
  /talk Lyanna promise
  /talk off

Downtime examples:
  /downtime rest by the fire
  /downtime write a letter home
  /downtime train with Lyanna

Challenges:
  Active challenges pause first on a confirmation screen.
  Types: dice roll (d100), rock-paper-scissors, memory sequence,
         quick-time (press key in time), riddle, stat/skill/item checks.
  The game engine resolves the outcome fairly — the AI then narrates the result.

	Footer legend:
  10.4s         total response time
  ft 5.5s       time to first token
  6016t         total tokens
  4980p/1036c   prompt/completion tokens
  r193          reasoning tokens
  cache 900p    cached prompt tokens
  99.7t/s       completion throughput

Quick aside:
  /btw Who exactly is Dee Podale Suprema?
  /btw Remind me what this faction wants`

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

// showStats displays the full character sheet as an overlay.
func (m NarrativeModel) showStats() (NarrativeModel, tea.Cmd) {
	char := m.narrator.Character()
	story := m.narrator.Story()
	var sb strings.Builder

	sb.WriteString("\n")

	// Parse stats JSON.
	var stats map[string]interface{}
	if char.StatsJSON != "" {
		_ = json.Unmarshal([]byte(char.StatsJSON), &stats)
	}

	// Parse stats schema for proper labels.
	type statDefSimple struct {
		Key   string
		Label string
	}
	var vitalDefs []statDefSimple
	var attrDefs []statDefSimple
	var secDefs []statDefSimple

	if story != nil && story.StatsSchemaJSON != "" {
		var schema map[string]interface{}
		if err := json.Unmarshal([]byte(story.StatsSchemaJSON), &schema); err == nil {
			parseDefs := func(key string) []statDefSimple {
				raw, ok := schema[key].([]interface{})
				if !ok {
					return nil
				}
				var defs []statDefSimple
				for _, item := range raw {
					if m2, ok := item.(map[string]interface{}); ok {
						k, _ := m2["key"].(string)
						l, _ := m2["label"].(string)
						if k == "" {
							continue
						}
						if l == "" {
							l = strings.ToUpper(k)
						}
						defs = append(defs, statDefSimple{Key: k, Label: l})
					}
				}
				return defs
			}
			vitalDefs = parseDefs("vitals")
			attrDefs = parseDefs("attributes")
			secDefs = parseDefs("secondary")
		}
	}

	// Character name and background.
	sb.WriteString(theme.Title.Render(char.Name) + "\n")
	if char.Background != "" {
		sb.WriteString(theme.MutedText.Render(char.Background) + "\n")
	}

	if stats != nil {
		sb.WriteString("\n")

		// Vitals with progress bars.
		if vitalsMap, ok := stats["vitals"].(map[string]interface{}); ok && len(vitalsMap) > 0 {
			sb.WriteString(theme.Title.Render("Vitals") + "\n")
			// Use schema order if available, else sort keys.
			vitalKeys := make([]string, 0, len(vitalDefs))
			if len(vitalDefs) > 0 {
				for _, def := range vitalDefs {
					if _, exists := vitalsMap[def.Key]; exists {
						vitalKeys = append(vitalKeys, def.Key)
					}
				}
			}
			if len(vitalKeys) == 0 {
				for k := range vitalsMap {
					vitalKeys = append(vitalKeys, k)
				}
				sort.Strings(vitalKeys)
			}
			// Build label map for lookup.
			vitalLabelMap := map[string]string{}
			for _, def := range vitalDefs {
				vitalLabelMap[def.Key] = def.Label
			}
			for _, key := range vitalKeys {
				vMap, ok := vitalsMap[key].(map[string]interface{})
				if !ok {
					continue
				}
				current := toInt(vMap["current"])
				max := toInt(vMap["max"])
				label := vitalLabelMap[key]
				if label == "" {
					label = strings.ToUpper(key)
				}
				bar := renderBar(current, max, 10)
				sb.WriteString(fmt.Sprintf("  %-10s %s  %d/%d\n", label+":", bar, current, max))
			}
			sb.WriteString("\n")
		}

		// Attributes with schema labels, 3 per row.
		if attrsMap, ok := stats["attributes"].(map[string]interface{}); ok && len(attrsMap) > 0 {
			sb.WriteString(theme.Title.Render("Attributes") + "\n")
			// Use schema order if available.
			type attrEntry struct {
				label string
				val   int
			}
			var attrEntries []attrEntry
			if len(attrDefs) > 0 {
				for _, def := range attrDefs {
					if v, ok := attrsMap[def.Key]; ok {
						attrEntries = append(attrEntries, attrEntry{def.Label, toInt(v)})
					}
				}
			} else {
				keys := make([]string, 0, len(attrsMap))
				for k := range attrsMap {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					attrEntries = append(attrEntries, attrEntry{strings.ToUpper(k), toInt(attrsMap[k])})
				}
			}
			// Display in rows of 3.
			for i, entry := range attrEntries {
				sb.WriteString(fmt.Sprintf("  %-14s %-3d", entry.label+":", entry.val))
				if (i+1)%3 == 0 || i == len(attrEntries)-1 {
					sb.WriteString("\n")
				}
			}
			sb.WriteString("\n")
		}

		// Secondary stats with schema labels.
		if secMap, ok := stats["secondary"].(map[string]interface{}); ok && len(secMap) > 0 {
			sb.WriteString(theme.Title.Render("Secondary") + "\n")
			secLabelMap := map[string]string{}
			for _, def := range secDefs {
				secLabelMap[def.Key] = def.Label
			}
			// Use schema order if available.
			secKeys := make([]string, 0)
			if len(secDefs) > 0 {
				for _, def := range secDefs {
					if _, ok := secMap[def.Key]; ok {
						secKeys = append(secKeys, def.Key)
					}
				}
			}
			if len(secKeys) == 0 {
				for k := range secMap {
					secKeys = append(secKeys, k)
				}
				sort.Strings(secKeys)
			}
			for _, key := range secKeys {
				val := toInt(secMap[key])
				label := secLabelMap[key]
				if label == "" {
					label = strings.Title(strings.ReplaceAll(key, "_", " "))
				}
				sb.WriteString(fmt.Sprintf("  %-16s %d\n", label+":", val))
			}
			sb.WriteString("\n")
		}
	}

	// Skills with level and XP bar.
	sb.WriteString(theme.Title.Render("Skills") + "\n")
	skillsRendered := false
	// Merge skills from char.SkillsJSON and stats["skills"].
	skillsMap := map[string]interface{}{}
	if char.SkillsJSON != "" && char.SkillsJSON != "null" && char.SkillsJSON != "{}" {
		_ = json.Unmarshal([]byte(char.SkillsJSON), &skillsMap)
	}
	if stats != nil {
		if sm, ok := stats["skills"].(map[string]interface{}); ok {
			for k, v := range sm {
				skillsMap[k] = v
			}
		}
	}
	if len(skillsMap) > 0 {
		skillKeys := make([]string, 0, len(skillsMap))
		for k := range skillsMap {
			skillKeys = append(skillKeys, k)
		}
		sort.Strings(skillKeys)
		for _, name := range skillKeys {
			val := skillsMap[name]
			level := 1
			xp := 0
			if sm, ok := val.(map[string]interface{}); ok {
				level = toInt(sm["level"])
				if level < 1 {
					level = 1
				}
				xp = toInt(sm["xp"])
			}
			threshold := level * 100
			bar := renderBar(xp, threshold, 10)
			sb.WriteString(fmt.Sprintf("  %-16s Lv.%-2d  %s  %d/%d XP\n",
				name, level, bar, xp, threshold))
		}
		skillsRendered = true
	}
	if !skillsRendered {
		sb.WriteString(theme.MutedText.Render("  (none yet — try new things to learn!)") + "\n")
	}
	sb.WriteString("\n")

	// Traits — merged from char.TraitsJSON and stats["traits"], deduplicated.
	sb.WriteString(theme.Title.Render("Traits") + "\n")
	traitSet := map[string]bool{}
	var traits []string
	if char.TraitsJSON != "" && char.TraitsJSON != "null" {
		var t []string
		if err := json.Unmarshal([]byte(char.TraitsJSON), &t); err == nil {
			for _, tr := range t {
				if !traitSet[strings.ToLower(tr)] {
					traitSet[strings.ToLower(tr)] = true
					traits = append(traits, tr)
				}
			}
		}
	}
	if stats != nil {
		if t, ok := stats["traits"].([]interface{}); ok {
			for _, tr := range t {
				if s, ok := tr.(string); ok && !traitSet[strings.ToLower(s)] {
					traitSet[strings.ToLower(s)] = true
					traits = append(traits, s)
				}
			}
		}
	}
	if len(traits) == 0 {
		sb.WriteString(theme.MutedText.Render("  (none yet — your character is still forming)") + "\n")
	} else {
		sb.WriteString("  " + strings.Join(traits, ", ") + "\n")
	}
	sb.WriteString("\n")

	// Titles.
	sb.WriteString(theme.Title.Render("Titles") + "\n")
	var titles []string
	if stats != nil {
		if t, ok := stats["titles"].([]interface{}); ok {
			for _, ti := range t {
				if s, ok := ti.(string); ok {
					titles = append(titles, s)
				}
			}
		}
	}
	if len(titles) == 0 {
		sb.WriteString(theme.MutedText.Render("  (none yet — great deeds earn great titles)") + "\n")
	} else {
		sb.WriteString("  " + strings.Join(titles, ", ") + "\n")
	}
	sb.WriteString("\n")

	// Deaths counter (if > 0).
	if stats != nil {
		if deaths := toInt(stats["deaths"]); deaths > 0 {
			sb.WriteString(theme.DangerText.Render(fmt.Sprintf("  Deaths: %d", deaths)) + "\n")
			sb.WriteString("\n")
		}
	}

	// NPC Relationships — load from DB.
	sb.WriteString(theme.Title.Render("Relationships") + "\n")
	if m.narrator.DB() != nil && story != nil {
		npcs, err := m.narrator.DB().ListNPCs(story.ID)
		if err == nil && len(npcs) > 0 {
			for _, npc := range npcs {
				d := npc.Disposition
				// Map disposition [-100, 100] to bar [0, 10].
				barVal := (d + 100) * 10 / 200
				if barVal < 0 {
					barVal = 0
				}
				if barVal > 10 {
					barVal = 10
				}
				bar := renderBar(barVal, 10, 10)
				label := dispositionLabel(d)
				sign := ""
				if d > 0 {
					sign = "+"
				}
				roleStr := ""
				if npc.Role != "" {
					roleStr = fmt.Sprintf(" (%s)", npc.Role)
				}
				sb.WriteString(fmt.Sprintf("  %-24s %s  %s (%s%d)\n",
					npc.Name+roleStr, bar, label, sign, d))
				sb.WriteString(theme.MutedText.Render("      " + relationshipAxesSummary(npc.RelationshipJSON)) + "\n")
			}
		} else {
			sb.WriteString(theme.MutedText.Render("  (no one met yet)") + "\n")
		}
	} else {
		sb.WriteString(theme.MutedText.Render("  (no one met yet)") + "\n")
	}

	m.showOverlay(fmt.Sprintf("Character: %s", char.Name), sb.String())
	return m, nil
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
	m.waiting = true
	m.choices.SetChoices(nil) // clear choices while waiting for AI
	narrator := m.narrator
	action = m.wrapPlayerAction(action)
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
	m.streamRaw.Reset()
	choiceItems, choiceHelp := m.buildChoicePresentation(nr.Choices)
	animateScene := !streamed && strings.TrimSpace(rendered) != ""
	m.choices.SetMood(m.currentMood)
	if animateScene {
		m.deferredChoiceItems = choiceItems
		m.deferredChoiceHelp = choiceHelp
		m.deferredInputFocus = len(nr.Choices) == 0
		m.deferredChallenges = nr.Challenges
		m.choices.SetChoices(nil)
		m.choiceHelp = map[int]string{}
		m.pendingChallenges = nil
		m.pendingChallenge = nil
		m.inputFocus = false
		m.input.Blur()
	} else {
		m.deferredChoiceItems = nil
		m.deferredChoiceHelp = nil
		m.deferredChallenges = nil
		m.choiceHelp = choiceHelp
		m.choices.SetChoices(choiceItems)
		if len(nr.Choices) > 0 {
			m.inputFocus = false
			m.input.Blur()
		} else {
			m.inputFocus = true
			m.input.Focus()
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

	if !animateScene && len(nr.Challenges) > 0 {
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
		if cmd := engine.ParseCommand(value); cmd != nil && cmd.Name != "unknown" {
			textColor = theme.Success
			promptColor = theme.Success
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

	if m.pendingChallenge != nil && !m.scenePlaybackActive() {
		return m.challengePreludeView()
	}

	if m.historyBrowser != nil && m.historyBrowser.Visible() {
		return m.historyBrowser.View()
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
	if !sceneReady {
		inputView = theme.MutedText.Render("  [Enter/Space] Finish scene")
	} else if m.inputFocus {
		m.applyInputCommandStyle()
		inputView = m.input.View()
	} else {
		inputView = theme.MutedText.Render("  [TAB] Free input")
	}

	// Temporary status message (e.g. "Autosaved")
	var statusLine string
	if m.statusMsg != "" {
		statusLine = theme.SuccessText.Render("  ✓ " + m.statusMsg)
	}

	// Help line
	help := theme.MutedText.Render("h history · tab toggle · 1-9 choose · ←/→ badge info · enter send · s quicksave · esc session")

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
	m.statusBar.SetWidth(m.width)
}
