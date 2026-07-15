package tui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/crimsab/oneday/internal/config"
	"github.com/crimsab/oneday/internal/i18n"
	"github.com/crimsab/oneday/internal/tui/views"

	"github.com/crimsab/oneday/internal/aifactory"
)

func TestSettingsLocaleChangePersistsAndUpdatesMenuImmediately(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := config.Default()
	app := New(cfg, nil, nil, i18n.New(i18n.English), path)
	updated, _ := app.Update(views.SettingsLocaleChangedMsg{Locale: i18n.Italian})
	got := updated.(App)
	if got.loc.Locale() != i18n.Italian || !strings.Contains(got.menu.View(), "Nuova storia") {
		t.Fatalf("locale did not update immediately: %q", got.menu.View())
	}
	loaded, err := config.Load(path)
	if err != nil || loaded.Interface.Locale != "it" {
		t.Fatalf("persisted locale=%q err=%v", loaded.Interface.Locale, err)
	}
}

func TestSelectEmbeddingProviderUsesFirstEmbeddingCapableProvider(t *testing.T) {
	cfg := config.Default()
	cfg.AI.ClaudeCode.Enabled = true
	cfg.AI.OpenRouter.Enabled = true
	cfg.AI.OpenRouter.APIKey = "openrouter-key"
	cfg.AI.LiteLLM.Enabled = false
	cfg.AI.ProviderPriority = []string{"claude-code", "openrouter", "litellm"}

	spec, reason := aifactory.SelectEmbeddingProvider(cfg)
	if reason != "" {
		t.Fatalf("selectEmbeddingProvider returned unexpected reason: %s", reason)
	}
	if spec.Name != "openrouter" {
		t.Fatalf("selectEmbeddingProvider picked %q, want openrouter", spec.Name)
	}
	if spec.BaseURL != cfg.AI.OpenRouter.BaseURL {
		t.Fatalf("selectEmbeddingProvider base URL = %q, want %q", spec.BaseURL, cfg.AI.OpenRouter.BaseURL)
	}
}

func TestSelectEmbeddingProviderHonorsExplicitProvider(t *testing.T) {
	cfg := config.Default()
	cfg.AI.Embedding.Provider = "openrouter"
	cfg.AI.LiteLLM.Enabled = true
	cfg.AI.OpenRouter.Enabled = true
	cfg.AI.OpenRouter.APIKey = "openrouter-key"

	spec, reason := aifactory.SelectEmbeddingProvider(cfg)
	if reason != "" {
		t.Fatalf("selectEmbeddingProvider returned unexpected reason: %s", reason)
	}
	if spec.Name != "openrouter" {
		t.Fatalf("selectEmbeddingProvider picked %q, want openrouter", spec.Name)
	}
}

func TestSelectEmbeddingProviderRejectsExplicitUnsupportedProvider(t *testing.T) {
	cfg := config.Default()
	cfg.AI.Embedding.Provider = "claude-code"
	cfg.AI.ClaudeCode.Enabled = true

	_, reason := aifactory.SelectEmbeddingProvider(cfg)
	if reason == "" {
		t.Fatal("selectEmbeddingProvider reason = empty, want unsupported-provider explanation")
	}
}

func TestSelectEmbeddingProviderReportsMissingSupport(t *testing.T) {
	cfg := config.Default()
	cfg.AI.ClaudeCode.Enabled = true
	cfg.AI.LiteLLM.Enabled = false
	cfg.AI.OpenRouter.Enabled = false
	cfg.AI.ProviderPriority = []string{"claude-code"}

	_, reason := aifactory.SelectEmbeddingProvider(cfg)
	if reason == "" {
		t.Fatal("selectEmbeddingProvider reason = empty, want a missing-provider explanation")
	}
}
