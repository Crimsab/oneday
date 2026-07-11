package prompts

import (
	"fmt"
	"strings"
)

func SceneProgressionJudgeSystem(storyName, language, writingStyle, promptDirectives string) string {
	return fmt.Sprintf(`You are the scene progression judge for "%s".

You are NOT writing story prose for the player.
You are deciding how the NEXT narrative turn should handle pacing.

Story language: %s
Writing style: %s
Extra directives: %s

Your job:
- Read the recent turns and the heuristic stall signal
- Read any existing protagonist timeline summary when available
- Decide whether the current scene is still productive or is stalling
- Choose the BEST next-step strategy for the narrator
- Return ONLY JSON

Assessment meanings:
- "productive" = the scene can stay where it is, but only if the next turn still changes stakes, information, pressure, or relationships in a meaningful way
- "stalled" = the scene is looping, rephrasing the same beat, or offering choices that do not meaningfully move anything

Strategy meanings:
- "deepen" = remain in the scene, but materially deepen it with new information, sharper stakes, or a clear emotional shift
- "interrupt" = bring in an arrival, interruption, external event, or immediate complication
- "reveal" = surface hidden information, a truth, a clue, or a change in understanding
- "consequence" = cash out prior actions with fallout, cost, reward, pressure, or reaction
- "time_skip" = jump forward to the next meaningful beat without playing low-value filler turn by turn
- "location_shift" = move the story to a new place because the current scene has clearly finished its useful work

Time skip guidance:
- A time skip does NOT require changing location
- Prefer "time_skip" for childhood growth, training, recovery, travel drift, incubation, seasonal routine, or any repetitive slice where the next interesting beat is later
- If the protagonist is a child or is clearly in an early-life montage, you may jump to the next meaningful age or milestone
- If the exact age is unclear, do NOT invent one; prefer a life-stage or milestone jump instead
- A good time skip still preserves continuity: mention what changed, what stayed true, and what new tension now matters

Output rules:
- instruction must be a short internal directive to the narrator, not player-facing prose
- reason must briefly explain why this choice is best now
- Keep both concise and specific
- If strategy is not "time_skip", leave time_skip_label and time_skip_detail empty`, storyName, language, writingStyle, promptDirectives)
}

func SceneProgressionJudgeUser(chapter, turn int, location, timelineSummary, heuristicSignal, recentTurns string) string {
	return fmt.Sprintf(`Current state:
- Chapter: %d
- Turn: %d
- Location: %s
- Timeline: %s

Heuristic stall signal:
%s

Recent turns:
%s

Decide the best pacing move for the NEXT narrator response.`, chapter, turn, location, timelineSummary, heuristicSignal, recentTurns)
}

func NarrativeRerollCorrection(strategy, reason, instruction string, repeatedTerms []string, timeSkipLabel, timeSkipDetail string) string {
	var sb strings.Builder
	sb.WriteString("Draft correction:\n")
	sb.WriteString("- The previous internal draft was rejected because it still felt too stagnant.\n")
	if strings.TrimSpace(reason) != "" {
		sb.WriteString("- Why it was rejected: " + strings.TrimSpace(reason) + "\n")
	}
	if strings.TrimSpace(strategy) != "" {
		sb.WriteString("- Required pacing strategy: " + strings.TrimSpace(strategy) + "\n")
	}
	if strings.TrimSpace(instruction) != "" {
		sb.WriteString("- Apply this now: " + strings.TrimSpace(instruction) + "\n")
	}
	if len(repeatedTerms) > 0 {
		sb.WriteString("- Do not recycle these motifs or choice families: " + strings.Join(repeatedTerms, ", ") + "\n")
	}
	if strings.TrimSpace(timeSkipLabel) != "" {
		sb.WriteString("- If using a time skip, jump directly to: " + strings.TrimSpace(timeSkipLabel) + "\n")
	}
	if strings.TrimSpace(timeSkipDetail) != "" {
		sb.WriteString("- Preserve continuity like this: " + strings.TrimSpace(timeSkipDetail) + "\n")
	}
	sb.WriteString("- Rewrite from scratch. Keep the same JSON schema. Do not paraphrase the rejected draft.")
	return sb.String()
}
