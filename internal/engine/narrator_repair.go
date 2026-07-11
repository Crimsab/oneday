package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/ai/prompts"
)

func (n *Narrator) repairNarrativeResponse(ctx context.Context, currentInput, invalidOutput string, parseErr error) (*NarrativeResponse, ai.Response, error) {
	baseReq := ai.Request{
		Messages: []ai.Message{
			{Role: ai.RoleSystem, Content: prompts.NarrativeRepairSystemPrompt(
				n.story.Name,
				n.story.Genre,
				n.story.Tone,
				n.story.Language,
				n.world.CurrentLocation,
				n.story.SettingJSON,
			)},
			{Role: ai.RoleUser, Content: prompts.NarrativeRepairUserPrompt(currentInput, invalidOutput, parseErr.Error())},
		},
		Temperature:    0.1,
		MaxTokens:      n.genCfg.MaxTokens,
		ResponseFormat: ai.NarrativeResponseFormat(),
	}
	candidates := n.genCfg.RepairModelCandidates()
	if len(candidates) == 0 {
		candidates = []string{""}
	}

	var errs []string
	for _, model := range candidates {
		req := baseReq
		req.Model = model

		start := time.Now()
		resp, err := n.router.Complete(ctx, req)
		latency := time.Since(start).Milliseconds()
		if err != nil {
			label := strings.TrimSpace(model)
			if label == "" {
				label = "provider-default"
			}
			errs = append(errs, fmt.Sprintf("%s: %v", label, err))
			continue
		}

		n.lastLatency += latency
		n.lastUsage = mergeUsage(n.lastUsage, resp.Usage)
		if resp.Model != "" && strings.TrimSpace(n.lastModel) == "" {
			n.lastModel = resp.Model
		}

		repaired, err := parseNarrativeFromAI(resp.Content)
		if err == nil {
			normalizeNarrativeResponse(repaired)
			return repaired, resp, nil
		}

		label := resp.Model
		if strings.TrimSpace(label) == "" {
			label = strings.TrimSpace(model)
		}
		if label == "" {
			label = "provider-default"
		}
		errs = append(errs, fmt.Sprintf("%s: %v", label, err))
	}

	return nil, ai.Response{}, fmt.Errorf("repair models failed: %s", strings.Join(errs, " | "))
}

func mergeUsage(base, extra ai.Usage) ai.Usage {
	return ai.Usage{
		PromptTokens:       base.PromptTokens + extra.PromptTokens,
		CompletionTokens:   base.CompletionTokens + extra.CompletionTokens,
		ReasoningTokens:    base.ReasoningTokens + extra.ReasoningTokens,
		TotalTokens:        base.TotalTokens + extra.TotalTokens,
		CachedPromptTokens: base.CachedPromptTokens + extra.CachedPromptTokens,
		CostUSD:            base.CostUSD + extra.CostUSD,
	}
}
