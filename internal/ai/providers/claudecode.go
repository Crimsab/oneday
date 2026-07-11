package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/config"
)

// ClaudeCode implements ai.Provider by shelling out to the Claude Code CLI.
type ClaudeCode struct {
	binary string
}

// claudeCodeResponse is the JSON output from `claude --output-format json`.
type claudeCodeResponse struct {
	Result string `json:"result"`
	Model  string `json:"model"`
}

// NewClaudeCode creates a Claude Code CLI provider.
func NewClaudeCode(cfg config.ClaudeCodeConfig) *ClaudeCode {
	binary := cfg.Binary
	if binary == "" {
		binary = "claude"
	}
	return &ClaudeCode{binary: binary}
}

func (c *ClaudeCode) Name() string {
	return "claude-code"
}

// Complete sends the last user message as a prompt to claude CLI.
// System messages and history are concatenated into the prompt since the CLI
// does not support multi-turn conversation directly.
func (c *ClaudeCode) Complete(ctx context.Context, req ai.Request) (ai.Response, error) {
	start := time.Now()

	// Build prompt from messages
	prompt := buildPrompt(req.Messages)
	if prompt == "" {
		return ai.Response{}, fmt.Errorf("no messages to send")
	}

	cmd := exec.CommandContext(ctx, c.binary, "-p", prompt, "--output-format", "json")
	output, err := cmd.Output()
	if err != nil {
		return ai.Response{}, fmt.Errorf("claude CLI exec: %w", err)
	}

	var resp claudeCodeResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return ai.Response{}, fmt.Errorf("parsing claude CLI output: %w", err)
	}

	return ai.Response{
		Content:   resp.Result,
		Model:     resp.Model,
		LatencyMs: time.Since(start).Milliseconds(),
		Provider:  c.Name(),
	}, nil
}

// buildPrompt concatenates system and user messages into a single prompt string.
func buildPrompt(messages []ai.Message) string {
	var parts []string
	for _, m := range messages {
		switch m.Role {
		case ai.RoleSystem:
			parts = append(parts, "[System]\n"+m.Content)
		case ai.RoleUser:
			parts = append(parts, "[User]\n"+m.Content)
		case ai.RoleAssistant:
			parts = append(parts, "[Assistant]\n"+m.Content)
		}
	}
	return strings.Join(parts, "\n\n")
}
