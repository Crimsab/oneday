package providers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/config"
)

// Codex implements ai.Provider by shelling out to the OpenAI Codex CLI.
type Codex struct {
	binary    string
	model     string
	reasoning string
}

// NewCodex creates a Codex CLI provider.
func NewCodex(cfg config.CodexConfig) *Codex {
	binary := cfg.Binary
	if binary == "" {
		binary = "codex"
	}
	model := cfg.Model
	reasoning := cfg.Reasoning
	if reasoning == "" {
		reasoning = "off"
	}
	return &Codex{binary: binary, model: model, reasoning: reasoning}
}

func (c *Codex) Name() string {
	return "codex"
}

// Complete sends the request to `codex exec` using the user's local Codex auth.
func (c *Codex) Complete(ctx context.Context, req ai.Request) (ai.Response, error) {
	start := time.Now()

	prompt := buildPrompt(req.Messages)
	if prompt == "" {
		return ai.Response{}, fmt.Errorf("no messages to send")
	}

	model := req.Model
	if model == "" {
		model = c.model
	}
	if strings.TrimSpace(model) == "" {
		return ai.Response{}, fmt.Errorf("Codex model missing: set ai.codex.model in config.yaml or choose a Codex model in the browser Options panel")
	}

	dir, err := os.MkdirTemp("", "oneday-codex-*")
	if err != nil {
		return ai.Response{}, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(dir)
	outputPath := filepath.Join(dir, "last-message.txt")

	args := []string{
		"exec",
		"--model", model,
		"--sandbox", "read-only",
		"--skip-git-repo-check",
		"--ephemeral",
		"--output-last-message", outputPath,
	}
	if reasoning := codexReasoningEffort(c.reasoning); reasoning != "" {
		args = append(args, "-c", fmt.Sprintf("model_reasoning_effort=%q", reasoning))
	}
	args = append(args, "-")

	cmd := exec.CommandContext(ctx, c.binary, args...)
	cmd.Stdin = strings.NewReader(prompt)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ai.Response{}, fmt.Errorf("codex CLI exec: %w", codexCLIError(err, string(output)))
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		return ai.Response{}, fmt.Errorf("reading codex output: %w", err)
	}

	return ai.Response{
		Content:   strings.TrimSpace(string(content)),
		Model:     model,
		LatencyMs: time.Since(start).Milliseconds(),
		Provider:  c.Name(),
	}, nil
}

func codexCLIError(err error, output string) error {
	out := strings.TrimSpace(output)
	var pathErr *exec.Error
	if errors.As(err, &pathErr) && pathErr.Err == exec.ErrNotFound {
		return fmt.Errorf("Codex CLI not found: install Codex, ensure `codex` is on PATH, then run `oneday doctor`")
	}
	lower := strings.ToLower(out)
	if strings.Contains(lower, "login") || strings.Contains(lower, "auth") || strings.Contains(lower, "unauthorized") || strings.Contains(lower, "401") {
		return fmt.Errorf("Codex authentication failed: run `codex login`, then `oneday doctor` (%s)", firstNonEmpty(out, err.Error()))
	}
	return fmt.Errorf("%w: %s", err, out)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "no output"
}

func codexReasoningEffort(reasoning string) string {
	switch reasoning {
	case "", "off":
		return "none"
	default:
		return reasoning
	}
}

// Stream wraps Complete in a single synthetic stream chunk so Codex can remain
// first in the router priority chain even though the CLI integration is batch.
func (c *Codex) Stream(ctx context.Context, req ai.Request) (<-chan ai.StreamChunk, error) {
	resp, err := c.Complete(ctx, req)
	if err != nil {
		return nil, err
	}
	ch := make(chan ai.StreamChunk, 2)
	go func() {
		defer close(ch)
		if resp.Content != "" {
			ch <- ai.StreamChunk{Content: resp.Content, Model: resp.Model, Usage: resp.Usage}
		}
		ch <- ai.StreamChunk{Model: resp.Model, Usage: resp.Usage, Done: true}
	}()
	return ch, nil
}
