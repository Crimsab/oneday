package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/ai/prompts"
	"github.com/crimsab/oneday/internal/storage"
)

type sceneProgressionAssessment string

const (
	sceneProgressionAssessmentProductive sceneProgressionAssessment = "productive"
	sceneProgressionAssessmentStalled    sceneProgressionAssessment = "stalled"
)

type sceneProgressionStrategy string

const (
	sceneProgressionStrategyDeepen        sceneProgressionStrategy = "deepen"
	sceneProgressionStrategyInterrupt     sceneProgressionStrategy = "interrupt"
	sceneProgressionStrategyReveal        sceneProgressionStrategy = "reveal"
	sceneProgressionStrategyConsequence   sceneProgressionStrategy = "consequence"
	sceneProgressionStrategyTimeSkip      sceneProgressionStrategy = "time_skip"
	sceneProgressionStrategyLocationShift sceneProgressionStrategy = "location_shift"
)

type SceneProgressionGuidance struct {
	Assessment     sceneProgressionAssessment `json:"assessment"`
	Strategy       sceneProgressionStrategy   `json:"strategy"`
	Reason         string                     `json:"reason"`
	Instruction    string                     `json:"instruction"`
	TimeSkipLabel  string                     `json:"time_skip_label,omitempty"`
	TimeSkipDetail string                     `json:"time_skip_detail,omitempty"`
}

func (n *Narrator) evaluateSceneProgression(
	ctx context.Context,
	recentMessages []storage.ChatMessage,
	signal *narrativeMomentumSignal,
) (*SceneProgressionGuidance, ai.Usage, int64, error) {
	if n == nil || n.router == nil || signal == nil {
		return nil, ai.Usage{}, 0, nil
	}

	timeout := 12 * time.Second
	if n.genCfg.TimeoutSeconds > 0 && n.genCfg.TimeoutSeconds < 12 {
		timeout = time.Duration(n.genCfg.TimeoutSeconds) * time.Second
	}
	judgeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req := ai.Request{
		Messages: []ai.Message{
			{
				Role: ai.RoleSystem,
				Content: prompts.SceneProgressionJudgeSystem(
					n.story.Name,
					n.story.Language,
					n.story.WritingStyle,
					n.story.PromptDirectives,
				),
			},
			{
				Role: ai.RoleUser,
				Content: prompts.SceneProgressionJudgeUser(
					n.world.CurrentChapter,
					n.world.CurrentTurn,
					n.world.CurrentLocation,
					firstNonEmpty(formatCharacterTimelineSummary(loadCharacterTimeline(n.world)), "No canonical timeline yet. Exact age unresolved; do not invent one unless the story grounds it."),
					formatMomentumSignalForJudge(signal),
					formatRecentTurnsForJudge(recentMessages, 8),
				),
			},
		},
		Temperature:    0.2,
		MaxTokens:      384,
		ResponseFormat: ai.SceneProgressionResponseFormat(),
	}

	start := time.Now()
	resp, err := n.router.Complete(judgeCtx, req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return nil, ai.Usage{}, latency, err
	}

	guidance, err := parseSceneProgressionGuidance(resp.Content)
	if err != nil {
		return nil, resp.Usage, latency, err
	}
	return guidance, resp.Usage, latency, nil
}

func parseSceneProgressionGuidance(raw string) (*SceneProgressionGuidance, error) {
	payload, err := ai.ExtractJSONPayload(raw)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(payload) == "" {
		return nil, fmt.Errorf("empty scene progression payload")
	}

	var guidance SceneProgressionGuidance
	if err := json.Unmarshal([]byte(payload), &guidance); err != nil {
		return nil, err
	}

	guidance.Assessment = sceneProgressionAssessment(strings.TrimSpace(strings.ToLower(string(guidance.Assessment))))
	guidance.Strategy = sceneProgressionStrategy(strings.TrimSpace(strings.ToLower(string(guidance.Strategy))))
	guidance.Reason = strings.TrimSpace(guidance.Reason)
	guidance.Instruction = strings.TrimSpace(guidance.Instruction)
	guidance.TimeSkipLabel = strings.TrimSpace(guidance.TimeSkipLabel)
	guidance.TimeSkipDetail = strings.TrimSpace(guidance.TimeSkipDetail)

	if guidance.Reason == "" || guidance.Instruction == "" {
		return nil, fmt.Errorf("scene progression guidance missing reason or instruction")
	}
	switch guidance.Assessment {
	case sceneProgressionAssessmentProductive, sceneProgressionAssessmentStalled:
	default:
		return nil, fmt.Errorf("invalid scene progression assessment %q", guidance.Assessment)
	}
	switch guidance.Strategy {
	case sceneProgressionStrategyDeepen, sceneProgressionStrategyInterrupt, sceneProgressionStrategyReveal,
		sceneProgressionStrategyConsequence, sceneProgressionStrategyTimeSkip, sceneProgressionStrategyLocationShift:
	default:
		return nil, fmt.Errorf("invalid scene progression strategy %q", guidance.Strategy)
	}

	return &guidance, nil
}

type stalledDraftIssue struct {
	Reason           string
	RepeatedTerms    []string
	ChoiceSimilarity float64
}

func (n *Narrator) rerollStalledNarrativeDraft(ctx context.Context, prep *preparedTurn, initial ai.Response) (ai.Response, error) {
	if prep == nil || prep.sceneProgression == nil {
		return initial, nil
	}

	initialCandidate, err := parseNarrativeFromAI(initial.Content)
	if err != nil {
		return initial, nil
	}
	normalizeNarrativeResponse(initialCandidate)

	issue := detectStalledNarrativeDraft(prep.recentMessages, initialCandidate, prep.sceneProgression)
	if issue == nil {
		return initial, nil
	}

	rerollReq := prep.req
	rerollReq.Messages = injectSystemMessageBeforeFinalUser(
		prep.req.Messages,
		prompts.NarrativeRerollCorrection(
			string(prep.sceneProgression.Strategy),
			issue.Reason,
			prep.sceneProgression.Instruction,
			issue.RepeatedTerms,
			prep.sceneProgression.TimeSkipLabel,
			prep.sceneProgression.TimeSkipDetail,
		),
	)
	if rerollReq.Temperature > 0.65 || rerollReq.Temperature == 0 {
		rerollReq.Temperature = 0.65
	}
	if rerollReq.Temperature < 0.35 {
		rerollReq.Temperature = 0.35
	}

	start := time.Now()
	rerollResp, rerollErr := n.router.Complete(ctx, rerollReq)
	rerollLatency := time.Since(start).Milliseconds()
	combinedLatency := initial.LatencyMs + rerollLatency
	combinedUsage := mergeUsage(initial.Usage, rerollResp.Usage)

	if rerollErr != nil {
		initial.LatencyMs = combinedLatency
		return initial, nil
	}

	rerollCandidate, err := parseNarrativeFromAI(rerollResp.Content)
	if err != nil {
		initial.LatencyMs = combinedLatency
		initial.Usage = combinedUsage
		return initial, nil
	}
	normalizeNarrativeResponse(rerollCandidate)
	if detectStalledNarrativeDraft(prep.recentMessages, rerollCandidate, prep.sceneProgression) != nil {
		initial.LatencyMs = combinedLatency
		initial.Usage = combinedUsage
		return initial, nil
	}

	rerollResp.LatencyMs = combinedLatency
	rerollResp.Usage = combinedUsage
	return rerollResp, nil
}

func detectStalledNarrativeDraft(
	recentMessages []storage.ChatMessage,
	candidate *NarrativeResponse,
	guidance *SceneProgressionGuidance,
) *stalledDraftIssue {
	if candidate == nil || guidance == nil {
		return nil
	}

	recentBeats := extractRecentAssistantBeats(recentMessages, 3)
	if len(recentBeats) == 0 {
		return nil
	}

	candidateBeat := buildAssistantBeatFromNarrativeResponse(candidate, lastBeatLocation(recentBeats))
	recentRepeatedTerms := repeatedBeatTerms(recentBeats)
	overlap := intersectTerms(candidateBeat.terms, recentRepeatedTerms)
	choiceSimilarity := maxChoiceSimilarity(candidateBeat.choiceTerms, recentBeats)
	sameLocation := strings.EqualFold(strings.TrimSpace(candidateBeat.location), strings.TrimSpace(lastBeatLocation(recentBeats)))
	hasStructuralMovement := narrativeHasStructuralMovement(candidate)
	strategySatisfied := narrativeSatisfiesProgressionStrategy(candidate, recentBeats, guidance, sameLocation)

	shouldReroll := false
	switch guidance.Strategy {
	case sceneProgressionStrategyTimeSkip:
		shouldReroll = !strategySatisfied && (sameLocation || choiceSimilarity >= 0.25 || len(overlap) >= 2)
	case sceneProgressionStrategyLocationShift:
		shouldReroll = !strategySatisfied && (sameLocation || choiceSimilarity >= 0.25 || len(overlap) >= 1)
	default:
		shouldReroll = !strategySatisfied && !hasStructuralMovement && len(overlap) >= 2 && (choiceSimilarity >= 0.35 || sameLocation)
	}
	if !shouldReroll {
		return nil
	}

	reason := fmt.Sprintf("the draft still overlaps with recent motifs (%s) and does not land the requested %s progression cleanly", strings.Join(overlapOrFallback(overlap), ", "), guidance.Strategy)
	return &stalledDraftIssue{
		Reason:           reason,
		RepeatedTerms:    overlapOrFallback(overlap),
		ChoiceSimilarity: choiceSimilarity,
	}
}

func buildAssistantBeatFromNarrativeResponse(narrative *NarrativeResponse, fallbackLocation string) recentAssistantBeat {
	beat := recentAssistantBeat{
		location:    firstNonEmpty(narrative.Location, fallbackLocation),
		choiceTerms: make(map[string]struct{}),
	}
	termSet := make(map[string]struct{})
	for _, token := range significantNarrativeTokens(narrative.Narrative) {
		termSet[token] = struct{}{}
	}
	for _, choice := range narrative.Choices {
		for _, token := range significantNarrativeTokens(choice.Text) {
			termSet[token] = struct{}{}
			beat.choiceTerms[token] = struct{}{}
		}
	}
	beat.terms = sortedKeys(termSet)
	return beat
}

func narrativeHasStructuralMovement(narrative *NarrativeResponse) bool {
	if narrative == nil {
		return false
	}
	return len(narrative.EventCallouts) > 0 ||
		len(narrative.OpenHooks) > 0 ||
		len(narrative.WorldReactions) > 0 ||
		len(narrative.StateChanges) > 0 ||
		len(narrative.Challenges) > 0 ||
		narrative.SocialDuel != nil ||
		narrative.CombatStart != nil ||
		narrative.ChapterEnd
}

func narrativeSatisfiesProgressionStrategy(
	candidate *NarrativeResponse,
	recentBeats []recentAssistantBeat,
	guidance *SceneProgressionGuidance,
	sameLocation bool,
) bool {
	if candidate == nil || guidance == nil {
		return true
	}

	switch guidance.Strategy {
	case sceneProgressionStrategyTimeSkip:
		return narrativeHasTemporalAdvance(candidate.Narrative, guidance) || narrativeHasTimelineUpdate(candidate) || (narrativeHasStructuralMovement(candidate) && !sameLocation)
	case sceneProgressionStrategyLocationShift:
		lastLocation := lastBeatLocation(recentBeats)
		return lastLocation != "" && candidate.Location != "" && !strings.EqualFold(strings.TrimSpace(candidate.Location), strings.TrimSpace(lastLocation))
	case sceneProgressionStrategyInterrupt, sceneProgressionStrategyReveal, sceneProgressionStrategyConsequence:
		return narrativeHasStructuralMovement(candidate)
	case sceneProgressionStrategyDeepen:
		return narrativeHasStructuralMovement(candidate) || len(significantNarrativeTokens(candidate.Narrative)) >= 8
	default:
		return true
	}
}

func narrativeHasTemporalAdvance(text string, guidance *SceneProgressionGuidance) bool {
	normalized := strings.ToLower(normalizeNarrativeText(text))
	if normalized == "" {
		return false
	}
	markers := []string{
		"years later", "months later", "weeks later", "days later", "later that year", "at age ",
		"anni dopo", "anno dopo", "anni piu tardi", "anni più tardi", "mesi dopo", "settimane dopo",
		"giorni dopo", "piu tardi", "più tardi", "all'eta", "all'età", "a cinque anni", "a sei anni",
		"a sette anni", "a otto anni", "a nove anni", "a dieci anni", "quando compi", "crescendo",
	}
	for _, marker := range markers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	if guidance != nil {
		for _, token := range significantNarrativeTokens(guidance.TimeSkipLabel + " " + guidance.TimeSkipDetail) {
			if strings.Contains(normalized, token) {
				return true
			}
		}
	}
	return false
}

func narrativeHasTimelineUpdate(narrative *NarrativeResponse) bool {
	if narrative == nil || len(narrative.StateChanges) == 0 {
		return false
	}
	_, ok := narrative.StateChanges["timeline_update"]
	return ok
}

func maxChoiceSimilarity(candidate map[string]struct{}, beats []recentAssistantBeat) float64 {
	maxValue := 0.0
	for i := len(beats) - 1; i >= 0; i-- {
		if value := jaccardSimilarity(candidate, beats[i].choiceTerms); value > maxValue {
			maxValue = value
		}
	}
	return maxValue
}

func lastBeatLocation(beats []recentAssistantBeat) string {
	if len(beats) == 0 {
		return ""
	}
	return beats[len(beats)-1].location
}

func intersectTerms(candidate []string, repeated []string) []string {
	if len(candidate) == 0 || len(repeated) == 0 {
		return nil
	}
	repeatedSet := make(map[string]struct{}, len(repeated))
	for _, term := range repeated {
		repeatedSet[term] = struct{}{}
	}
	var out []string
	for _, term := range candidate {
		if _, ok := repeatedSet[term]; ok {
			out = append(out, term)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func overlapOrFallback(overlap []string) []string {
	if len(overlap) > 0 {
		if len(overlap) > 4 {
			return overlap[:4]
		}
		return overlap
	}
	return []string{"same beat", "same choices"}
}

func injectSystemMessageBeforeFinalUser(messages []ai.Message, content string) []ai.Message {
	if strings.TrimSpace(content) == "" {
		return append([]ai.Message(nil), messages...)
	}
	cloned := append([]ai.Message(nil), messages...)
	insertAt := len(cloned)
	if len(cloned) > 0 && cloned[len(cloned)-1].Role == ai.RoleUser {
		insertAt = len(cloned) - 1
	}
	cloned = append(cloned, ai.Message{})
	copy(cloned[insertAt+1:], cloned[insertAt:])
	cloned[insertAt] = ai.Message{Role: ai.RoleSystem, Content: content}
	return cloned
}

func formatMomentumSignalForJudge(signal *narrativeMomentumSignal) string {
	if signal == nil {
		return "No heuristic stall signal."
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("recent_turns=%d", signal.recentTurns))
	if signal.sameLocation {
		parts = append(parts, "same_location=true")
	}
	if signal.lowWorldPressure {
		parts = append(parts, "low_world_pressure=true")
	}
	if signal.similarChoicePairs > 0 {
		parts = append(parts, fmt.Sprintf("similar_choice_pairs=%d", signal.similarChoicePairs))
	}
	if len(signal.repeatedTerms) > 0 {
		parts = append(parts, "repeated_terms="+strings.Join(signal.repeatedTerms, ", "))
	}
	return strings.Join(parts, " | ")
}

func formatRecentTurnsForJudge(recentMessages []storage.ChatMessage, limit int) string {
	if limit <= 0 {
		limit = 8
	}
	start := len(recentMessages) - limit
	if start < 0 {
		start = 0
	}

	var sb strings.Builder
	for _, msg := range recentMessages[start:] {
		role := strings.ToUpper(strings.TrimSpace(msg.Role))
		if role == "" {
			role = "UNKNOWN"
		}
		content := compactJudgeText(msg.Content, 240)
		sb.WriteString(fmt.Sprintf("Turn %d %s: %s\n", msg.Turn, role, content))
		if !strings.EqualFold(msg.Role, "assistant") {
			continue
		}
		if meta, ok := parsePersistedAssistantMeta(msg.MetadataJSON); ok {
			var choices []string
			switch {
			case meta.Output != nil && len(meta.Output.ChoicesData) > 0:
				for _, choice := range meta.Output.ChoicesData {
					if text := compactJudgeText(choice.Text, 80); text != "" {
						choices = append(choices, text)
					}
				}
			case meta.Output != nil && len(meta.Output.Choices) > 0:
				for _, choice := range meta.Output.Choices {
					if text := compactJudgeText(choice, 80); text != "" {
						choices = append(choices, text)
					}
				}
			case len(meta.Choices) > 0:
				for _, choice := range meta.Choices {
					if text := compactJudgeText(choice, 80); text != "" {
						choices = append(choices, text)
					}
				}
			}
			if len(choices) > 0 {
				sb.WriteString("Choices: " + strings.Join(choices, " | ") + "\n")
			}
		}
	}
	return strings.TrimSpace(sb.String())
}

func compactJudgeText(text string, limit int) string {
	text = strings.Join(strings.Fields(normalizeNarrativeText(text)), " ")
	if text == "" || limit <= 0 || len(text) <= limit {
		return text
	}
	return strings.TrimSpace(text[:limit-3]) + "..."
}
