package engine

import (
	"context"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/ai/prompts"
)

func (n *Narrator) repairNarrativeResponse(ctx context.Context, invalidOutput string, parseErr error) (*NarrativeResponse, error) {
	req := ai.Request{
		Messages: []ai.Message{
			{Role: ai.RoleSystem, Content: prompts.NarrativeRepairSystemPrompt()},
			{Role: ai.RoleUser, Content: prompts.NarrativeRepairUserPrompt(invalidOutput, parseErr.Error())},
		},
		Temperature:    0.1,
		MaxTokens:      n.genCfg.MaxTokens,
		ResponseFormat: ai.NarrativeResponseFormat(),
	}

	start := time.Now()
	resp, err := n.router.Complete(ctx, req)
	if err != nil {
		return nil, err
	}

	n.lastLatency += time.Since(start).Milliseconds()
	n.lastUsage = mergeUsage(n.lastUsage, resp.Usage)
	if resp.Model != "" {
		n.lastModel = resp.Model
	}

	repaired, err := parseNarrativeFromAI(resp.Content)
	if err != nil {
		return nil, err
	}
	normalizeNarrativeResponse(repaired)
	return repaired, nil
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
