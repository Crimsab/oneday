package providers

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/config"
)

func TestCodexCompleteUsesOutputLastMessage(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake is unix-only")
	}
	dir := t.TempDir()
	fake := filepath.Join(dir, "codex")
	script := `#!/bin/sh
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output-last-message" ]; then
    shift
    out="$1"
  fi
  shift
done
cat >/dev/null
printf 'codex response\n' > "$out"
`
	if err := os.WriteFile(fake, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	provider := NewCodex(config.CodexConfig{
		Binary:    fake,
		Model:     "test-codex-model",
		Reasoning: "low",
	})
	resp, err := provider.Complete(context.Background(), ai.Request{
		Messages: []ai.Message{{Role: ai.RoleUser, Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if resp.Provider != "codex" {
		t.Errorf("Provider = %q, want codex", resp.Provider)
	}
	if resp.Model != "test-codex-model" {
		t.Errorf("Model = %q, want test-codex-model", resp.Model)
	}
	if strings.TrimSpace(resp.Content) != "codex response" {
		t.Errorf("Content = %q, want codex response", resp.Content)
	}
}

func TestCodexCompleteMissingBinaryIsActionable(t *testing.T) {
	provider := NewCodex(config.CodexConfig{
		Binary: "oneday-test-codex-missing-binary",
	})

	_, err := provider.Complete(context.Background(), ai.Request{
		Messages: []ai.Message{{Role: ai.RoleUser, Content: "hello"}},
	})
	if err == nil {
		t.Fatal("expected missing binary error")
	}
	msg := err.Error()
	for _, want := range []string{"Codex CLI not found", "oneday doctor"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q missing %q", msg, want)
		}
	}
}

func TestCodexCompleteAuthFailureIsActionable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake is unix-only")
	}
	dir := t.TempDir()
	fake := filepath.Join(dir, "codex")
	script := `#!/bin/sh
echo "401 unauthorized: run codex login" >&2
exit 1
`
	if err := os.WriteFile(fake, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	provider := NewCodex(config.CodexConfig{Binary: fake})
	_, err := provider.Complete(context.Background(), ai.Request{
		Messages: []ai.Message{{Role: ai.RoleUser, Content: "hello"}},
	})
	if err == nil {
		t.Fatal("expected auth error")
	}
	msg := err.Error()
	for _, want := range []string{"Codex authentication failed", "codex login", "oneday doctor"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q missing %q", msg, want)
		}
	}
}
