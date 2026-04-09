package views

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/storage"
	"github.com/crimsab/oneday/internal/tui/components"
	"github.com/crimsab/oneday/internal/tui/theme"
)

type socialDuelFocus int

const (
	socialDuelFocusActions socialDuelFocus = iota
	socialDuelFocusNote
)

type socialDuelRoundResolvedMsg struct {
	Cue          *engine.SocialDuelCue
	State        *engine.SocialDuelState
	Round        *engine.SocialRoundResult
	PlayerAction engine.SocialAction
	NPCAction    engine.SocialAction
	PlayerNote   string
}

type socialDuelResultPayload struct {
	NPCName         string                    `json:"npc_name"`
	Objective       string                    `json:"objective"`
	Stakes          string                    `json:"stakes,omitempty"`
	Round           int                       `json:"round"`
	Resolved        bool                      `json:"resolved"`
	Winner          string                    `json:"winner,omitempty"`
	Outcome         string                    `json:"outcome,omitempty"`
	PlayerAction    engine.SocialAction       `json:"player_action"`
	NPCAction       engine.SocialAction       `json:"npc_action"`
	Exchange        string                    `json:"exchange"`
	PlayerRoll      int                       `json:"player_roll,omitempty"`
	NPCRoll         int                       `json:"npc_roll,omitempty"`
	PlayerDamage    int                       `json:"player_damage,omitempty"`
	NPCDamage       int                       `json:"npc_damage,omitempty"`
	Tempo           int                       `json:"tempo"`
	PlayerComposure int                       `json:"player_composure"`
	NPCComposure    int                       `json:"npc_composure"`
	PlayerPatience  int                       `json:"player_patience"`
	NPCPatience     int                       `json:"npc_patience"`
	PlayerNote      string                    `json:"player_note,omitempty"`
	FailForward     *engine.SocialFailForward `json:"fail_forward,omitempty"`
}

// SocialDuelView renders a dedicated round-based high-stakes dialogue mode.
type SocialDuelView struct {
	cue    *engine.SocialDuelCue
	state  *engine.SocialDuelState
	engine *engine.SocialDuelEngine
	char   *storage.Character
	npc    *storage.NPC

	actions []engine.SocialAction
	cursor  int
	focus   socialDuelFocus
	note    textinput.Model

	width  int
	height int
	errMsg string
}

func NewSocialDuelView(cue *engine.SocialDuelCue, state *engine.SocialDuelState, char *storage.Character, npc *storage.NPC, width, height int) *SocialDuelView {
	note := textinput.New()
	note.Placeholder = "Optional line or tactic to flavor this exchange..."
	note.CharLimit = 140
	note.Width = maxInt(24, minInt(width-16, 64))
	note.Blur()

	return &SocialDuelView{
		cue:     cue,
		state:   state,
		engine:  engine.NewSocialDuelEngine(),
		char:    char,
		npc:     npc,
		actions: socialDuelActionSet(cue, state),
		note:    note,
		width:   width,
		height:  height,
	}
}

func (v *SocialDuelView) Update(msg tea.Msg) (*SocialDuelView, tea.Cmd) {
	if v == nil {
		return v, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.note.Width = maxInt(24, minInt(msg.Width-16, 64))
		return v, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			v.toggleFocus()
			return v, nil
		case "[":
			v.shiftPlayerStance(-1)
			return v, nil
		case "]":
			v.shiftPlayerStance(1)
			return v, nil
		case "esc":
			return v, v.resolveCurrentAction(engine.SocialActionWithdraw)
		case "enter":
			return v, v.resolveCurrentAction(v.currentAction())
		}

		if v.focus == socialDuelFocusNote {
			var cmd tea.Cmd
			v.note, cmd = v.note.Update(msg)
			return v, cmd
		}

		switch msg.String() {
		case "up", "k":
			if v.cursor > 0 {
				v.cursor--
			}
			return v, nil
		case "down", "j":
			if v.cursor < len(v.actions)-1 {
				v.cursor++
			}
			return v, nil
		default:
			if len(msg.String()) == 1 {
				ch := msg.String()[0]
				if ch >= '1' && ch <= byte('0'+len(v.actions)) {
					v.cursor = int(ch - '1')
					return v, v.resolveCurrentAction(v.currentAction())
				}
			}
			return v, nil
		}
	}

	if v.focus == socialDuelFocusNote {
		var cmd tea.Cmd
		v.note, cmd = v.note.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v *SocialDuelView) View() string {
	if v == nil || v.state == nil {
		return ""
	}

	innerWidth := minInt(maxInt(v.width-12, 48), 84)
	lines := []string{
		theme.Title.Render("Social Duel"),
		"",
		fmt.Sprintf("%s vs %s", v.charName(), strings.TrimSpace(v.state.NPCName)),
		fmt.Sprintf("Objective: %s", firstNonEmptyString(v.cueObjective(), v.state.Objective)),
	}

	if stakes := firstNonEmptyString(v.cueStakes(), v.state.Stakes); stakes != "" {
		lines = append(lines, "Stakes: "+stakes)
	}
	if pressure := strings.TrimSpace(v.cuePressure()); pressure != "" {
		lines = append(lines, "Pressure: "+pressure)
	}
	if opening := strings.TrimSpace(v.cueOpening()); opening != "" {
		lines = append(lines, "", opening)
	}

	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Tempo: %s", socialTempoLabel(v.state.Tempo)))
	lines = append(lines, fmt.Sprintf("Your composure %d | patience %d", v.state.PlayerComposure, v.state.PlayerPatience))
	lines = append(lines, fmt.Sprintf("Their composure %d | patience %d", v.state.NPCComposure, v.state.NPCPatience))
	lines = append(lines, fmt.Sprintf("Stance: %s", socialStanceLabel(v.state.PlayerStance)))

	if leverage := socialDuelLeverageSummary(v.state.PlayerLeverage); leverage != "" {
		lines = append(lines, "Leverage: "+leverage)
	}

	lines = append(lines, "")
	lines = append(lines, theme.MutedText.Render("Choose your next move"))
	for i, action := range v.actions {
		prefix := "  "
		style := theme.UnselectedItem
		if i == v.cursor && v.focus == socialDuelFocusActions {
			prefix = "▸ "
			style = theme.SelectedItem
		}
		lines = append(lines, fmt.Sprintf("%s%d. %s", prefix, i+1, style.Render(socialActionLabel(action))))
		lines = append(lines, fmt.Sprintf("   %s", theme.MutedText.Render(socialActionHint(action))))
	}

	lines = append(lines, "")
	if v.focus == socialDuelFocusNote {
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.Highlight).Bold(true).Render("Approach note:")+" "+v.note.View())
	} else {
		lines = append(lines, theme.MutedText.Render("Approach note:")+" "+v.note.View())
	}

	if strings.TrimSpace(v.errMsg) != "" {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(theme.Danger).Render(v.errMsg))
	}

	lines = append(lines, "")
	lines = append(lines, theme.MutedText.Render("Up/Down choose move  Tab note  [ ] stance  Enter resolve  Esc withdraw"))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Secondary).
		Padding(1, 2).
		Width(innerWidth).
		Render(strings.Join(lines, "\n"))

	return lipgloss.Place(v.width, v.height, lipgloss.Center, lipgloss.Center, box)
}

func (v *SocialDuelView) toggleFocus() {
	if v.focus == socialDuelFocusActions {
		v.focus = socialDuelFocusNote
		v.note.Focus()
		return
	}
	v.focus = socialDuelFocusActions
	v.note.Blur()
}

func (v *SocialDuelView) shiftPlayerStance(delta int) {
	if v == nil || v.state == nil {
		return
	}
	order := []engine.SocialStance{
		engine.SocialStanceMeasured,
		engine.SocialStanceBold,
		engine.SocialStanceGuarded,
	}
	idx := 0
	for i, stance := range order {
		if v.state.PlayerStance == stance {
			idx = i
			break
		}
	}
	idx = (idx + delta + len(order)) % len(order)
	v.state.PlayerStance = order[idx]
}

func (v *SocialDuelView) currentAction() engine.SocialAction {
	if len(v.actions) == 0 {
		return engine.SocialActionAppeal
	}
	if v.cursor < 0 || v.cursor >= len(v.actions) {
		v.cursor = 0
	}
	return v.actions[v.cursor]
}

func (v *SocialDuelView) resolveCurrentAction(action engine.SocialAction) tea.Cmd {
	if v == nil || v.state == nil || v.engine == nil || v.char == nil {
		return nil
	}
	npcAction := v.engine.ChooseNPCAction(v.state)
	result, err := v.engine.ResolveRound(v.state, v.char, v.resolvedNPC(), action, npcAction)
	if err != nil {
		v.errMsg = fmt.Sprintf("Could not resolve exchange: %v", err)
		return nil
	}

	playerNote := strings.TrimSpace(v.note.Value())
	return func() tea.Msg {
		return socialDuelRoundResolvedMsg{
			Cue:          v.cue,
			State:        v.state,
			Round:        result,
			PlayerAction: action,
			NPCAction:    npcAction,
			PlayerNote:   playerNote,
		}
	}
}

func (v *SocialDuelView) resolvedNPC() *storage.NPC {
	if v.npc != nil {
		return v.npc
	}
	name := strings.TrimSpace(v.state.NPCName)
	if name == "" {
		name = "Opponent"
	}
	return &storage.NPC{Name: name, RelationshipJSON: `{}`}
}

func (v *SocialDuelView) charName() string {
	if v.char != nil && strings.TrimSpace(v.char.Name) != "" {
		return v.char.Name
	}
	return "You"
}

func (v *SocialDuelView) cueObjective() string {
	if v.cue == nil {
		return ""
	}
	return strings.TrimSpace(v.cue.Objective)
}

func (v *SocialDuelView) cueStakes() string {
	if v.cue == nil {
		return ""
	}
	return strings.TrimSpace(v.cue.Stakes)
}

func (v *SocialDuelView) cuePressure() string {
	if v.cue == nil {
		return ""
	}
	return strings.TrimSpace(v.cue.Pressure)
}

func (v *SocialDuelView) cueOpening() string {
	if v.cue == nil {
		return ""
	}
	return firstNonEmptyString(v.cue.ExchangeSummary, v.cue.Opening)
}

func (m *NarrativeModel) socialDuelInProgress() bool {
	return m.activeSocialDuel != nil && m.activeSocialDuel.Status == engine.SocialDuelActive
}

func (m *NarrativeModel) clearSocialDuelRuntime() {
	m.socialDuelView = nil
	m.inSocialDuel = false
	m.pendingSocialDuel = nil
	m.deferredSocialDuel = nil
	m.activeSocialDuel = nil
	m.activeSocialDuelCue = nil
	m.socialDuelNPC = nil
}

func (m *NarrativeModel) socialDuelCueForTurn(next *engine.SocialDuelCue) *engine.SocialDuelCue {
	switch {
	case next != nil && m.socialDuelInProgress():
		m.activeSocialDuelCue = mergeSocialDuelCue(m.activeSocialDuelCue, next, m.activeSocialDuel)
		return m.activeSocialDuelCue
	case next != nil:
		m.activeSocialDuelCue = mergeSocialDuelCue(nil, next, m.activeSocialDuel)
		return m.activeSocialDuelCue
	case m.socialDuelInProgress():
		m.activeSocialDuelCue = fallbackSocialDuelCue(m.activeSocialDuelCue, m.activeSocialDuel)
		return m.activeSocialDuelCue
	default:
		if m.activeSocialDuel != nil && m.activeSocialDuel.Status != engine.SocialDuelActive {
			m.clearSocialDuelRuntime()
		}
		return nil
	}
}

func (m *NarrativeModel) beginPendingSocialDuel() tea.Cmd {
	cue := m.pendingSocialDuel
	if cue == nil {
		return nil
	}

	state, npc, err := m.ensureSocialDuelState(cue)
	if err != nil {
		m.errMsg = fmt.Sprintf("Could not start social duel: %v", err)
		m.pendingSocialDuel = nil
		return nil
	}

	m.socialDuelView = NewSocialDuelView(cue, state, m.narrator.Character(), npc, m.width, m.height)
	m.inSocialDuel = true
	m.pendingSocialDuel = nil
	return nil
}

func (m *NarrativeModel) ensureSocialDuelState(cue *engine.SocialDuelCue) (*engine.SocialDuelState, *storage.NPC, error) {
	if cue == nil {
		return nil, nil, fmt.Errorf("missing social duel cue")
	}

	if m.socialDuelInProgress() {
		if m.activeSocialDuelCue == nil {
			m.activeSocialDuelCue = cue
		} else {
			m.activeSocialDuelCue = mergeSocialDuelCue(m.activeSocialDuelCue, cue, m.activeSocialDuel)
		}
		if m.socialDuelNPC == nil {
			m.socialDuelNPC = m.lookupSocialDuelNPC(cue.NPCName)
		}
		return m.activeSocialDuel, m.socialDuelNPC, nil
	}

	char := m.narrator.Character()
	if char == nil {
		return nil, nil, fmt.Errorf("missing protagonist state")
	}

	npc := m.lookupSocialDuelNPC(cue.NPCName)
	spec := engine.SocialDuelSpec{
		NPCName:        firstNonEmptyString(cue.NPCName, npc.Name),
		Objective:      cue.Objective,
		Stakes:         cue.Stakes,
		PlayerLeverage: socialDuelCueLeverage(cue),
	}
	state, err := engine.NewSocialDuelEngine().Start(spec, char, npc)
	if err != nil {
		return nil, nil, err
	}

	m.activeSocialDuel = state
	m.activeSocialDuelCue = mergeSocialDuelCue(nil, cue, state)
	m.socialDuelNPC = npc
	return state, npc, nil
}

func (m *NarrativeModel) lookupSocialDuelNPC(name string) *storage.NPC {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		trimmed = "Opponent"
	}
	if m.narrator != nil && m.narrator.DB() != nil {
		if npc, err := m.narrator.DB().GetNPCByName(m.narrator.Story().ID, trimmed); err == nil && npc != nil {
			return npc
		}
	}
	return &storage.NPC{Name: trimmed, RelationshipJSON: `{}`}
}

func fallbackSocialDuelCue(base *engine.SocialDuelCue, state *engine.SocialDuelState) *engine.SocialDuelCue {
	if state == nil || state.Status != engine.SocialDuelActive {
		return nil
	}
	return mergeSocialDuelCue(base, &engine.SocialDuelCue{
		Mode:            engine.SocialDuelCueContinue,
		NPCName:         state.NPCName,
		Objective:       state.Objective,
		Stakes:          state.Stakes,
		ExchangeSummary: state.LastExchangeNote,
	}, state)
}

func mergeSocialDuelCue(base, next *engine.SocialDuelCue, state *engine.SocialDuelState) *engine.SocialDuelCue {
	if base == nil && next == nil && state == nil {
		return nil
	}

	merged := &engine.SocialDuelCue{
		Mode:             engine.SocialDuelCueContinue,
		NPCName:          firstNonEmptyString(cueField(next, func(c *engine.SocialDuelCue) string { return c.NPCName }), cueField(base, func(c *engine.SocialDuelCue) string { return c.NPCName }), stateField(state, func(s *engine.SocialDuelState) string { return s.NPCName })),
		Objective:        firstNonEmptyString(cueField(next, func(c *engine.SocialDuelCue) string { return c.Objective }), cueField(base, func(c *engine.SocialDuelCue) string { return c.Objective }), stateField(state, func(s *engine.SocialDuelState) string { return s.Objective })),
		NPCGoal:          firstNonEmptyString(cueField(next, func(c *engine.SocialDuelCue) string { return c.NPCGoal }), cueField(base, func(c *engine.SocialDuelCue) string { return c.NPCGoal })),
		Stakes:           firstNonEmptyString(cueField(next, func(c *engine.SocialDuelCue) string { return c.Stakes }), cueField(base, func(c *engine.SocialDuelCue) string { return c.Stakes }), stateField(state, func(s *engine.SocialDuelState) string { return s.Stakes })),
		Pressure:         firstNonEmptyString(cueField(next, func(c *engine.SocialDuelCue) string { return c.Pressure }), cueField(base, func(c *engine.SocialDuelCue) string { return c.Pressure })),
		Opening:          firstNonEmptyString(cueField(next, func(c *engine.SocialDuelCue) string { return c.Opening }), cueField(base, func(c *engine.SocialDuelCue) string { return c.Opening })),
		ExchangeSummary:  firstNonEmptyString(cueField(next, func(c *engine.SocialDuelCue) string { return c.ExchangeSummary }), cueField(next, func(c *engine.SocialDuelCue) string { return c.Opening }), cueField(base, func(c *engine.SocialDuelCue) string { return c.ExchangeSummary }), stateField(state, func(s *engine.SocialDuelState) string { return s.LastExchangeNote })),
		FailForward:      firstNonEmptyString(cueField(next, func(c *engine.SocialDuelCue) string { return c.FailForward }), cueField(base, func(c *engine.SocialDuelCue) string { return c.FailForward })),
		Leverage:         chooseCueLeverage(base, next),
		SuggestedActions: chooseCueActions(base, next),
	}
	if next != nil && next.Mode != "" {
		merged.Mode = next.Mode
	} else if base != nil && base.Mode != "" {
		merged.Mode = base.Mode
	}
	if state != nil && state.Status == engine.SocialDuelActive && merged.Mode == "" {
		merged.Mode = engine.SocialDuelCueContinue
	}
	return merged
}

func buildSocialDuelResultInput(msg socialDuelRoundResolvedMsg) string {
	if msg.State == nil || msg.Round == nil {
		return "[Social Duel Result]"
	}

	roundPlayed := msg.State.Round
	if !msg.Round.Resolved && roundPlayed > 1 {
		roundPlayed--
	}

	payload := socialDuelResultPayload{
		NPCName:         msg.State.NPCName,
		Objective:       msg.State.Objective,
		Stakes:          msg.State.Stakes,
		Round:           roundPlayed,
		Resolved:        msg.Round.Resolved,
		Winner:          msg.Round.Winner,
		Outcome:         msg.Round.Outcome,
		PlayerAction:    msg.PlayerAction,
		NPCAction:       msg.NPCAction,
		Exchange:        msg.Round.ExchangeLabel,
		PlayerRoll:      msg.Round.PlayerRoll,
		NPCRoll:         msg.Round.NPCRoll,
		PlayerDamage:    msg.Round.PlayerDamage,
		NPCDamage:       msg.Round.NPCDamage,
		Tempo:           msg.State.Tempo,
		PlayerComposure: msg.State.PlayerComposure,
		NPCComposure:    msg.State.NPCComposure,
		PlayerPatience:  msg.State.PlayerPatience,
		NPCPatience:     msg.State.NPCPatience,
		PlayerNote:      strings.TrimSpace(msg.PlayerNote),
		FailForward:     msg.Round.FailForward,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf("[Social Duel Result] round=%d exchange=%s", payload.Round, payload.Exchange)
	}
	return "[Social Duel Result] " + string(raw)
}

func renderSocialDuelRoundNote(msg socialDuelRoundResolvedMsg) string {
	if msg.State == nil || msg.Round == nil {
		return components.RenderMarkdown("\n*[Social duel exchange resolved.]*\n")
	}

	lines := []string{
		fmt.Sprintf("**[Social Duel]** %s", msg.Round.ExchangeLabel),
		fmt.Sprintf("Tempo: %s", socialTempoLabel(msg.State.Tempo)),
	}
	if msg.Round.NPCDamage > 0 {
		lines = append(lines, fmt.Sprintf("You shake %s's composure by %d.", msg.State.NPCName, msg.Round.NPCDamage))
	}
	if msg.Round.PlayerDamage > 0 {
		lines = append(lines, fmt.Sprintf("You lose %d composure.", msg.Round.PlayerDamage))
	}
	if strings.TrimSpace(msg.PlayerNote) != "" {
		lines = append(lines, fmt.Sprintf("Your angle: \"%s\"", strings.TrimSpace(msg.PlayerNote)))
	}
	if msg.Round.FailForward != nil {
		lines = append(lines, fmt.Sprintf("%s: %s", msg.Round.FailForward.Title, msg.Round.FailForward.Detail))
	}
	return components.RenderMarkdown("\n" + strings.Join(lines, "\n") + "\n")
}

func (m NarrativeModel) socialDuelPreludeView() string {
	cue := m.pendingSocialDuel
	if cue == nil {
		return ""
	}

	lines := []string{
		theme.Title.Render("Social Duel"),
		"",
		fmt.Sprintf("Objective: %s", firstNonEmptyString(strings.TrimSpace(cue.Objective), stateField(m.activeSocialDuel, func(s *engine.SocialDuelState) string { return s.Objective }))),
	}
	if stakes := firstNonEmptyString(strings.TrimSpace(cue.Stakes), stateField(m.activeSocialDuel, func(s *engine.SocialDuelState) string { return s.Stakes })); stakes != "" {
		lines = append(lines, "Stakes: "+stakes)
	}
	if pressure := strings.TrimSpace(cue.Pressure); pressure != "" {
		lines = append(lines, "Pressure: "+pressure)
	}
	if summary := firstNonEmptyString(strings.TrimSpace(cue.ExchangeSummary), strings.TrimSpace(cue.Opening)); summary != "" {
		lines = append(lines, "", summary)
	}
	lines = append(lines, "", theme.MutedText.Render("Press Enter or Space to enter the exchange"))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Secondary).
		Padding(1, 2).
		Width(minInt(maxInt(m.width-8, 40), 78)).
		Render(strings.Join(lines, "\n"))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func socialDuelActionSet(cue *engine.SocialDuelCue, state *engine.SocialDuelState) []engine.SocialAction {
	var actions []engine.SocialAction
	if cue != nil && len(cue.SuggestedActions) > 0 {
		actions = append(actions, cue.SuggestedActions...)
	}

	defaults := []engine.SocialAction{
		engine.SocialActionAppeal,
		engine.SocialActionPressure,
		engine.SocialActionDeceive,
		engine.SocialActionConcede,
		engine.SocialActionEscalate,
	}
	if state != nil && hasActiveLeverage(state.PlayerLeverage) {
		defaults = append(defaults, engine.SocialActionExpose)
	}
	defaults = append(defaults, engine.SocialActionWithdraw)

	seen := map[engine.SocialAction]bool{}
	out := make([]engine.SocialAction, 0, len(actions)+len(defaults))
	for _, action := range append(actions, defaults...) {
		if action == "" || seen[action] {
			continue
		}
		seen[action] = true
		out = append(out, action)
	}
	return out
}

func hasActiveLeverage(items []engine.SocialLeverage) bool {
	for _, item := range items {
		if !item.Spent {
			return true
		}
	}
	return false
}

func socialActionLabel(action engine.SocialAction) string {
	switch action {
	case engine.SocialActionAppeal:
		return "Appeal"
	case engine.SocialActionPressure:
		return "Pressure"
	case engine.SocialActionDeceive:
		return "Deceive"
	case engine.SocialActionConcede:
		return "Concede"
	case engine.SocialActionExpose:
		return "Expose"
	case engine.SocialActionWithdraw:
		return "Withdraw"
	case engine.SocialActionEscalate:
		return "Escalate"
	default:
		return strings.Title(string(action))
	}
}

func socialActionHint(action engine.SocialAction) string {
	switch action {
	case engine.SocialActionAppeal:
		return "Lean on trust, empathy, debt, or shared cause."
	case engine.SocialActionPressure:
		return "Push the other side toward a mistake or costly concession."
	case engine.SocialActionDeceive:
		return "Bluff, conceal, or redirect attention."
	case engine.SocialActionConcede:
		return "Give ground now to preserve patience and shape the next beat."
	case engine.SocialActionExpose:
		return "Reveal leverage, proof, or a secret when timing matters."
	case engine.SocialActionWithdraw:
		return "Exit before you break, accepting a playable cost."
	case engine.SocialActionEscalate:
		return "Raise the stakes and dare them to blink first."
	default:
		return "Play the exchange."
	}
}

func socialTempoLabel(tempo int) string {
	switch {
	case tempo >= 3:
		return "strongly in your favor"
	case tempo >= 1:
		return "leaning your way"
	case tempo <= -3:
		return "dangerously against you"
	case tempo <= -1:
		return "slipping away from you"
	default:
		return "balanced"
	}
}

func socialStanceLabel(stance engine.SocialStance) string {
	switch stance {
	case engine.SocialStanceBold:
		return "Bold"
	case engine.SocialStanceGuarded:
		return "Guarded"
	default:
		return "Measured"
	}
}

func socialDuelLeverageSummary(items []engine.SocialLeverage) string {
	var labels []string
	for _, item := range items {
		if item.Spent {
			continue
		}
		label := strings.TrimSpace(item.Label)
		if label == "" {
			continue
		}
		labels = append(labels, label)
	}
	return strings.Join(labels, ", ")
}

func socialDuelCueLeverage(cue *engine.SocialDuelCue) []engine.SocialLeverage {
	if cue == nil || len(cue.Leverage) == 0 {
		return nil
	}
	out := make([]engine.SocialLeverage, 0, len(cue.Leverage))
	for _, item := range cue.Leverage {
		label := strings.TrimSpace(item.Label)
		if label == "" {
			continue
		}
		out = append(out, engine.SocialLeverage{
			ID:         strings.TrimSpace(item.Key),
			Label:      label,
			Kind:       strings.TrimSpace(item.Kind),
			Strength:   socialDuelLeverageStrength(item.Kind),
			Consumable: true,
		})
	}
	return out
}

func socialDuelLeverageStrength(kind string) int {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "evidence", "secret", "witness":
		return 2
	case "debt", "favor", "status":
		return 1
	default:
		return 1
	}
}

func cueField(cue *engine.SocialDuelCue, fn func(*engine.SocialDuelCue) string) string {
	if cue == nil {
		return ""
	}
	return strings.TrimSpace(fn(cue))
}

func stateField(state *engine.SocialDuelState, fn func(*engine.SocialDuelState) string) string {
	if state == nil {
		return ""
	}
	return strings.TrimSpace(fn(state))
}

func chooseCueLeverage(base, next *engine.SocialDuelCue) []engine.SocialDuelLeverage {
	if next != nil && len(next.Leverage) > 0 {
		return next.Leverage
	}
	if base != nil && len(base.Leverage) > 0 {
		return base.Leverage
	}
	return nil
}

func chooseCueActions(base, next *engine.SocialDuelCue) []engine.SocialAction {
	if next != nil && len(next.SuggestedActions) > 0 {
		return next.SuggestedActions
	}
	if base != nil && len(base.SuggestedActions) > 0 {
		return base.SuggestedActions
	}
	return nil
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
