package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/tui/theme"
)

type queuedNarrativeSegment struct {
	text    string
	animate bool
}

func (m *NarrativeModel) scenePlaybackActive() bool {
	return m.pendingNarrative != "" || m.typewriter.IsActive()
}

func (m *NarrativeModel) visibleNarrativeContent() string {
	if m.pendingNarrative == "" {
		return m.history.String()
	}
	return m.history.String() + m.typewriter.View()
}

func (m *NarrativeModel) appendNarrativeSegment(segment string, animate bool) tea.Cmd {
	if segment == "" {
		return nil
	}

	if m.pendingNarrative != "" {
		m.queuedNarrative = append(m.queuedNarrative, queuedNarrativeSegment{
			text:    segment,
			animate: animate,
		})
		return nil
	}

	if !animate {
		m.history.WriteString(segment)
		m.viewport.SetContent(m.history.String())
		m.viewport.GotoBottom()
		return nil
	}

	m.pendingNarrative = segment
	cmd := m.typewriter.SetText(segment)
	m.viewport.SetContent(m.visibleNarrativeContent())
	m.viewport.GotoBottom()
	return cmd
}

func (m *NarrativeModel) flushQueuedNarrative() tea.Cmd {
	for len(m.queuedNarrative) > 0 {
		next := m.queuedNarrative[0]
		m.queuedNarrative = m.queuedNarrative[1:]

		if next.animate {
			m.pendingNarrative = next.text
			cmd := m.typewriter.SetText(next.text)
			m.viewport.SetContent(m.visibleNarrativeContent())
			m.viewport.GotoBottom()
			return cmd
		}

		m.history.WriteString(next.text)
	}

	m.activateDeferredSceneState()
	m.viewport.SetContent(m.visibleNarrativeContent())
	m.viewport.GotoBottom()
	return nil
}

func (m *NarrativeModel) completePendingNarrative() tea.Cmd {
	if m.pendingNarrative != "" {
		m.history.WriteString(m.pendingNarrative)
		m.pendingNarrative = ""
	}
	return m.flushQueuedNarrative()
}

func (m *NarrativeModel) activateDeferredSceneState() {
	if len(m.deferredChoiceItems) > 0 {
		m.choices.SetChoices(m.deferredChoiceItems)
		m.choiceHelp = m.deferredChoiceHelp
	} else {
		m.choices.SetChoices(nil)
		m.choiceHelp = map[int]string{}
	}
	m.deferredChoiceItems = nil
	m.deferredChoiceHelp = nil

	if len(m.deferredChallenges) > 0 {
		m.pendingChallenges = m.deferredChallenges
		m.deferredChallenges = nil
	} else {
		m.deferredChallenges = nil
	}

	m.inputFocus = m.deferredInputFocus
	if m.inputFocus {
		m.input.Focus()
	} else {
		m.input.Blur()
	}
}

func (m *NarrativeModel) skipCurrentPlayback() tea.Cmd {
	if !m.scenePlaybackActive() {
		return nil
	}
	m.typewriter.Skip()
	return m.completePendingNarrative()
}

func activeChallengeRequiresConfirm(spec *engine.ChallengeSpec) bool {
	if spec == nil {
		return false
	}
	switch spec.Type {
	case engine.ChallengeDiceRoll, engine.ChallengeMiniGame:
		return true
	default:
		return false
	}
}

func (m *NarrativeModel) launchChallenge(spec *engine.ChallengeSpec) tea.Cmd {
	if spec == nil {
		return nil
	}

	ce := engine.NewChallengeEngine()
	cv, cmd := NewChallengeView(
		spec,
		ce,
		m.narrator.Character(),
		m.narrator.DB(),
		m.narrator.Story().ID,
		m.width, m.height,
	)
	m.challengeView = cv
	m.inChallenge = true
	return cmd
}

func (m *NarrativeModel) beginPendingChallenge() tea.Cmd {
	spec := m.pendingChallenge
	m.pendingChallenge = nil
	return m.launchChallenge(spec)
}

func challengeLabel(spec *engine.ChallengeSpec) string {
	if spec == nil {
		return "Challenge"
	}
	switch spec.Type {
	case engine.ChallengeDiceRoll:
		return "Stat Roll"
	case engine.ChallengeMiniGame:
		if spec.MiniGame != "" {
			return "Mini-game: " + strings.ReplaceAll(spec.MiniGame, "_", " ")
		}
		return "Mini-game"
	case engine.ChallengeStatCheck:
		return "Stat Check"
	case engine.ChallengeItemCheck:
		return "Item Check"
	case engine.ChallengeSkillCheck:
		return "Skill Check"
	case engine.ChallengeRelCheck:
		return "Relationship Check"
	default:
		return "Challenge"
	}
}

func challengePreludeDescription(spec *engine.ChallengeSpec) string {
	if spec == nil {
		return ""
	}
	switch spec.Type {
	case engine.ChallengeDiceRoll:
		return fmt.Sprintf("Roll against difficulty %d. Modifiers and the engine will resolve the outcome fairly.", spec.Difficulty)
	case engine.ChallengeMiniGame:
		if spec.MiniGame == string(engine.MiniGameRiddle) && strings.TrimSpace(spec.Riddle) != "" {
			return "A riddle is about to begin. Read carefully before answering."
		}
		return "A short skill mini-game is about to start. Stay ready."
	default:
		return "The game engine is about to resolve a challenge."
	}
}

func (m NarrativeModel) challengePreludeView() string {
	spec := m.pendingChallenge
	if spec == nil {
		return ""
	}

	lines := []string{
		theme.Title.Render(challengeLabel(spec)),
		"",
		challengePreludeDescription(spec),
	}
	if spec.Stat != "" {
		lines = append(lines, "", fmt.Sprintf("Stat: %s", spec.Stat))
	}
	if spec.Skill != "" {
		lines = append(lines, "", fmt.Sprintf("Skill: %s", spec.Skill))
	}
	if spec.NPCName != "" {
		lines = append(lines, "", fmt.Sprintf("NPC: %s", spec.NPCName))
	}
	if spec.Difficulty > 0 {
		lines = append(lines, fmt.Sprintf("Difficulty: %d", spec.Difficulty))
	}
	lines = append(lines, "", theme.MutedText.Render("Press Enter or Space to begin"))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Secondary).
		Padding(1, 2).
		Width(minInt(maxInt(m.width-8, 36), 72)).
		Render(strings.Join(lines, "\n"))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
