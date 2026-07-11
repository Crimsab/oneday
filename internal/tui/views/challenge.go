package views

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crimsab/oneday/internal/engine"
	"github.com/crimsab/oneday/internal/game/contracts"
	"github.com/crimsab/oneday/internal/storage"
	"github.com/crimsab/oneday/internal/tui/components"
	"github.com/crimsab/oneday/internal/tui/theme"
)

func challengeMiniGameSeed(storyID string, spec *engine.ChallengeSpec) int64 {
	digest := sha256.Sum256([]byte(storyID + "\x00" + spec.MiniGame + "\x00" + spec.Description))
	return int64(binary.BigEndian.Uint64(digest[:8]) & uint64(contracts.MaxPortableChallengeSeed))
}

func fallbackChallengePrompt(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

// ChallengeResolvedMsg carries the result of a challenge back to the narrative.
type ChallengeResolvedMsg struct {
	Spec   *engine.ChallengeSpec
	Result *engine.ChallengeResult
}

// passiveChallengeMsg is sent after a brief delay for passive challenges.
type passiveChallengeMsg struct {
	spec   *engine.ChallengeSpec
	result *engine.ChallengeResult
}

// ChallengeView manages the appropriate challenge component based on type.
type ChallengeView struct {
	spec   *engine.ChallengeSpec
	result *engine.ChallengeResult

	// Only one of these is active at a time.
	dice      *components.DiceModel
	rps       *components.RPSModel
	memory    *components.MemoryModel
	quicktime *components.QuickTimeModel
	riddle    *components.RiddleModel
	hostGame  *components.HostMiniGameModel

	// For passive challenges (stat/item/skill/relationship checks).
	passive        bool
	passiveMessage string

	width  int
	height int
}

// NewChallengeView creates the appropriate challenge UI for a spec.
// For passive challenges, resolution happens immediately.
// For active challenges, it sets up the interactive component.
func NewChallengeView(
	spec *engine.ChallengeSpec,
	ce *engine.ChallengeEngine,
	char *storage.Character,
	db *storage.DB,
	storyID string,
	width, height int,
) (*ChallengeView, tea.Cmd) {
	cv := &ChallengeView{
		spec:   spec,
		width:  width,
		height: height,
	}

	switch spec.Type {
	case engine.ChallengeStatCheck, engine.ChallengeItemCheck,
		engine.ChallengeSkillCheck, engine.ChallengeRelCheck:
		// Passive challenge: resolve immediately, show brief notification.
		result, err := ce.Resolve(spec, char, db, storyID)
		if err != nil {
			result = &engine.ChallengeResult{
				Passed: false,
				Detail: fmt.Sprintf("Error resolving challenge: %v", err),
			}
		}
		cv.passive = true
		cv.result = result

		msg := buildPassiveMessage(spec, result)
		cv.passiveMessage = msg

		capturedSpec := spec
		capturedResult := result
		cmd := tea.Tick(1800*time.Millisecond, func(time.Time) tea.Msg {
			return passiveChallengeMsg{spec: capturedSpec, result: capturedResult}
		})
		return cv, cmd

	case engine.ChallengeDiceRoll:
		// Pre-compute result, then show animated dice.
		result, err := ce.Resolve(spec, char, db, storyID)
		if err != nil {
			result = &engine.ChallengeResult{
				Passed: false,
				Detail: "Dice roll failed",
			}
		}
		cv.result = result

		mods := make([]components.ModDisplay, len(result.Modifiers))
		for i, m := range result.Modifiers {
			mods[i] = components.ModDisplay{Source: m.Source, Value: m.Value}
		}
		dice := components.NewDiceModel(challengeDisplayContext(spec), result.Roll, result.Total, result.Difficulty, mods, result.Passed, width, height)
		cv.dice = &dice
		return cv, cv.dice.Start()

	case engine.ChallengeMiniGame:
		switch spec.MiniGame {
		case string(engine.MiniGameRPS):
			rps := components.NewRPSModel(width, height)
			cv.rps = &rps
			return cv, nil

		case string(engine.MiniGameMemory):
			mc := engine.NewMemoryChallenge(spec.Sequence, len(spec.Sequence))
			mem := components.NewMemoryModel(mc, width, height)
			cv.memory = &mem
			return cv, cv.memory.Start()

		case string(engine.MiniGameQuickTime):
			qtc := engine.NewQuickTimeChallenge(spec.TimeLimit)
			qt := components.NewQuickTimeModel(qtc, width, height)
			cv.quicktime = &qt
			return cv, cv.quicktime.Start()

		case string(engine.MiniGameRiddle):
			rc := engine.NewRiddleChallenge(spec.Riddle, spec.Answer)
			rid := components.NewRiddleModel(rc, width, height)
			cv.riddle = &rid
			return cv, nil

		case string(engine.MiniGameDeduction), string(engine.MiniGameNegotiation), string(engine.MiniGamePattern), string(engine.MiniGameBidding):
			definition := engine.DefaultMiniGameDefinition(engine.MiniGameType(spec.MiniGame))
			definition.Prompt = fallbackChallengePrompt(spec.Description, definition.Prompt)
			hostGame, err := components.NewHostMiniGameModel(definition, challengeMiniGameSeed(storyID, spec), width, height)
			if err != nil {
				break
			}
			cv.hostGame = &hostGame
			return cv, nil
		}
	}

	// Fallback: treat as passive failure.
	cv.passive = true
	cv.result = &engine.ChallengeResult{Passed: false, Detail: "Unknown challenge type"}
	capturedSpec := spec
	capturedResult := cv.result
	return cv, tea.Tick(1000*time.Millisecond, func(time.Time) tea.Msg {
		return passiveChallengeMsg{spec: capturedSpec, result: capturedResult}
	})
}

// Update handles messages for the active challenge component.
func (cv *ChallengeView) Update(msg tea.Msg) (*ChallengeView, tea.Cmd) {
	// Passive auto-complete.
	if pm, ok := msg.(passiveChallengeMsg); ok {
		return cv, func() tea.Msg {
			return ChallengeResolvedMsg{Spec: pm.spec, Result: pm.result}
		}
	}

	// Active components.
	if cv.dice != nil {
		updated, cmd := cv.dice.Update(msg)
		cv.dice = &updated
		if dr, ok := msg.(components.DiceResultMsg); ok {
			_ = dr
			return cv, func() tea.Msg {
				return ChallengeResolvedMsg{Spec: cv.spec, Result: engine.EnsureLegacyChallengeOutcome(cv.result)}
			}
		}
		return cv, cmd
	}

	if cv.rps != nil {
		updated, cmd := cv.rps.Update(msg)
		cv.rps = &updated
		if rm, ok := msg.(components.RPSResultMsg); ok {
			return cv, func() tea.Msg {
				return ChallengeResolvedMsg{Spec: cv.spec, Result: engine.EnsureLegacyChallengeOutcome(rm.Result)}
			}
		}
		return cv, cmd
	}

	if cv.memory != nil {
		updated, cmd := cv.memory.Update(msg)
		cv.memory = &updated
		if mm, ok := msg.(components.MemoryResultMsg); ok {
			return cv, func() tea.Msg {
				return ChallengeResolvedMsg{Spec: cv.spec, Result: engine.EnsureLegacyChallengeOutcome(mm.Result)}
			}
		}
		return cv, cmd
	}

	if cv.quicktime != nil {
		updated, cmd := cv.quicktime.Update(msg)
		cv.quicktime = &updated
		if qm, ok := msg.(components.QuickTimeResultMsg); ok {
			return cv, func() tea.Msg {
				return ChallengeResolvedMsg{Spec: cv.spec, Result: engine.EnsureLegacyChallengeOutcome(qm.Result)}
			}
		}
		return cv, cmd
	}

	if cv.riddle != nil {
		updated, cmd := cv.riddle.Update(msg)
		cv.riddle = &updated
		if rm, ok := msg.(components.RiddleResultMsg); ok {
			return cv, func() tea.Msg {
				return ChallengeResolvedMsg{Spec: cv.spec, Result: engine.EnsureLegacyChallengeOutcome(rm.Result)}
			}
		}
		return cv, cmd
	}

	if cv.hostGame != nil {
		updated, cmd := cv.hostGame.Update(msg)
		cv.hostGame = &updated
		if resolved, ok := msg.(components.HostMiniGameResultMsg); ok {
			return cv, func() tea.Msg {
				return ChallengeResolvedMsg{Spec: cv.spec, Result: engine.EnsureLegacyChallengeOutcome(resolved.Result)}
			}
		}
		return cv, cmd
	}

	return cv, nil
}

// View renders the active challenge component.
func (cv *ChallengeView) View() string {
	if cv.passive {
		return cv.renderPassiveOverlay()
	}
	if cv.dice != nil {
		return cv.dice.View()
	}
	if cv.rps != nil {
		return cv.rps.View()
	}
	if cv.memory != nil {
		return cv.memory.View()
	}
	if cv.quicktime != nil {
		return cv.quicktime.View()
	}
	if cv.riddle != nil {
		return cv.riddle.View()
	}
	if cv.hostGame != nil {
		return cv.hostGame.View()
	}
	return ""
}

// IsPassive returns true if this is a passive challenge (already resolved).
func (cv *ChallengeView) IsPassive() bool {
	return cv.passive
}

// buildPassiveMessage creates a brief notification string for passive challenges.
func buildPassiveMessage(spec *engine.ChallengeSpec, result *engine.ChallengeResult) string {
	var label string
	switch spec.Type {
	case engine.ChallengeStatCheck:
		label = fmt.Sprintf("Stat Check [%s]", strings.ToUpper(spec.Stat))
	case engine.ChallengeItemCheck:
		label = fmt.Sprintf("Item Check [%s]", spec.Item)
	case engine.ChallengeSkillCheck:
		label = fmt.Sprintf("Skill Check [%s]", spec.Skill)
	case engine.ChallengeRelCheck:
		label = fmt.Sprintf("Relationship Check [%s]", spec.NPCName)
	default:
		label = "Check"
	}
	if context := strings.TrimSpace(spec.Description); context != "" {
		label += " — " + context
	}

	if result.Passed {
		return fmt.Sprintf("✓ %s — Passed", label)
	}
	return fmt.Sprintf("✗ %s — Failed", label)
}

func challengeDisplayContext(spec *engine.ChallengeSpec) string {
	if spec == nil {
		return ""
	}
	if text := strings.TrimSpace(spec.Description); text != "" {
		return text
	}

	switch spec.Type {
	case engine.ChallengeDiceRoll:
		return fmt.Sprintf("Roll against difficulty %d.", spec.Difficulty)
	case engine.ChallengeStatCheck:
		if spec.Stat != "" {
			return fmt.Sprintf("Check %s.", strings.ToUpper(spec.Stat))
		}
	case engine.ChallengeItemCheck:
		if spec.Item != "" {
			return fmt.Sprintf("Requires %s.", spec.Item)
		}
	case engine.ChallengeSkillCheck:
		if spec.Skill != "" {
			return fmt.Sprintf("Check skill %s.", spec.Skill)
		}
	case engine.ChallengeRelCheck:
		if spec.NPCName != "" {
			return fmt.Sprintf("Social check against %s.", spec.NPCName)
		}
	}

	return "Resolve the current challenge."
}

// renderPassiveOverlay renders a brief notification for passive challenges.
func (cv *ChallengeView) renderPassiveOverlay() string {
	innerW := 38
	var lines []string

	var color lipgloss.Color
	if cv.result != nil && cv.result.Passed {
		color = theme.Success
	} else {
		color = theme.Danger
	}

	titleStyle := lipgloss.NewStyle().Foreground(color).Bold(true)
	lines = append(lines, titleStyle.Render(cv.passiveMessage))
	if cv.result != nil {
		lines = append(lines, "")
		lines = append(lines, theme.MutedText.Render("  "+cv.result.Detail))
	}

	inner := strings.Join(lines, "\n")
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color).
		Padding(1, 3).
		Width(innerW + 4)

	box := boxStyle.Render(inner)
	return lipgloss.Place(cv.width, cv.height, lipgloss.Center, lipgloss.Center, box)
}
