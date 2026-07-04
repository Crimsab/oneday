package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/crimsab/oneday/internal/config"
)

func TestWantsVersion(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		{args: nil, want: false},
		{args: []string{"--version"}, want: true},
		{args: []string{"version"}, want: true},
		{args: []string{"play"}, want: false},
	}

	for _, tc := range tests {
		if got := wantsVersion(tc.args); got != tc.want {
			t.Fatalf("wantsVersion(%v) = %v, want %v", tc.args, got, tc.want)
		}
	}
}

func TestWantsOperatorCommands(t *testing.T) {
	if !wantsConfigShowSafe([]string{"config", "show", "--safe"}) {
		t.Fatal("expected config show --safe")
	}
	if !wantsRAGBenchmark([]string{"rag", "benchmark"}) {
		t.Fatal("expected rag benchmark")
	}
	if !wantsRAGReindex([]string{"rag", "reindex"}) {
		t.Fatal("expected rag reindex")
	}
	if !wantsStoryPacksList([]string{"story-packs", "list"}) {
		t.Fatal("expected story-packs list")
	}
	if !wantsExport([]string{"export"}) {
		t.Fatal("expected export")
	}
	if !wantsGatewayModelSettings([]string{"gateway-model-settings"}) {
		t.Fatal("expected gateway-model-settings")
	}
	if !wantsGatewayModelSettingsUpdate([]string{"gateway-model-settings-update"}) {
		t.Fatal("expected gateway-model-settings-update")
	}
	if !wantsGatewayStoryCreate([]string{"gateway-story-create"}) {
		t.Fatal("expected gateway-story-create")
	}
	if !wantsGatewayStoryWizard([]string{"gateway-story-wizard"}) {
		t.Fatal("expected gateway-story-wizard")
	}
	if !wantsGatewayStoryEnhance([]string{"gateway-story-enhance"}) {
		t.Fatal("expected gateway-story-enhance")
	}
}

func TestProviderConsistencyWarnings(t *testing.T) {
	cfg := config.Default()
	cfg.AI.LiteLLM.Enabled = true
	cfg.AI.LiteLLM.APIKey = ""
	warnings := providerConsistencyWarnings(cfg)
	if len(warnings) == 0 {
		t.Fatal("expected missing LiteLLM key warning")
	}
}

func TestDiscoverStoryPacks(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "plugins", "examples")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pack.yaml"), []byte("id: pack\n"), 0644); err != nil {
		t.Fatal(err)
	}
	packs, err := discoverStoryPacks([]string{root})
	if err != nil {
		t.Fatalf("discoverStoryPacks: %v", err)
	}
	if len(packs) != 1 || filepath.Base(packs[0]) != "pack.yaml" {
		t.Fatalf("packs = %#v", packs)
	}
}

func TestValidateStoryPack(t *testing.T) {
	dir := t.TempDir()
	pack := filepath.Join(dir, "pack.yaml")
	if err := os.WriteFile(pack, []byte("id: pack\nname: Pack\ndescription: Demo\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateStoryPack(pack); err != nil {
		t.Fatalf("validateStoryPack: %v", err)
	}
	bad := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(bad, []byte("id: bad\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateStoryPack(bad); err == nil {
		t.Fatal("expected invalid story pack")
	}
}

func TestSetupConfigForChoice(t *testing.T) {
	tests := []struct {
		name           string
		choice         string
		wantProvider   string
		wantCodex      bool
		wantLiteLLM    bool
		wantOpenRouter bool
		wantRAG        bool
		wantKey        string
	}{
		{
			name:         "codex",
			choice:       "1",
			wantProvider: "codex",
			wantCodex:    true,
			wantRAG:      false,
		},
		{
			name:         "litellm",
			choice:       "2",
			wantProvider: "litellm",
			wantLiteLLM:  true,
			wantRAG:      true,
			wantKey:      "${ONEDAY_LITELLM_API_KEY}",
		},
		{
			name:           "openrouter",
			choice:         "3",
			wantProvider:   "openrouter",
			wantOpenRouter: true,
			wantRAG:        true,
			wantKey:        "${ONEDAY_OPENROUTER_API_KEY}",
		},
		{
			name:         "codex local rag",
			choice:       "4",
			wantProvider: "codex",
			wantCodex:    true,
			wantRAG:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := setupConfigForChoice(config.Default(), tc.choice)
			if err != nil {
				t.Fatalf("setupConfigForChoice: %v", err)
			}
			if err := cfg.Validate(); err != nil {
				t.Fatalf("Validate: %v", err)
			}
			if cfg.AI.ProviderPriority[0] != tc.wantProvider {
				t.Fatalf("first provider = %q, want %q", cfg.AI.ProviderPriority[0], tc.wantProvider)
			}
			if cfg.AI.Codex.Enabled != tc.wantCodex {
				t.Fatalf("Codex enabled = %v, want %v", cfg.AI.Codex.Enabled, tc.wantCodex)
			}
			if cfg.AI.LiteLLM.Enabled != tc.wantLiteLLM {
				t.Fatalf("LiteLLM enabled = %v, want %v", cfg.AI.LiteLLM.Enabled, tc.wantLiteLLM)
			}
			if cfg.AI.OpenRouter.Enabled != tc.wantOpenRouter {
				t.Fatalf("OpenRouter enabled = %v, want %v", cfg.AI.OpenRouter.Enabled, tc.wantOpenRouter)
			}
			if cfg.RAG.Enabled != tc.wantRAG {
				t.Fatalf("RAG enabled = %v, want %v", cfg.RAG.Enabled, tc.wantRAG)
			}
			if tc.wantKey != "" && cfg.AI.LiteLLM.APIKey != tc.wantKey && cfg.AI.OpenRouter.APIKey != tc.wantKey {
				t.Fatalf("expected placeholder key %q in setup config", tc.wantKey)
			}
			if tc.choice == "4" {
				if cfg.AI.Embedding.Provider != "local" || cfg.AI.Embedding.Local.Type != "ollama" {
					t.Fatalf("local RAG config not selected: %#v", cfg.AI.Embedding)
				}
			}
		})
	}
}

func TestWantsSetupForce(t *testing.T) {
	if !wantsSetupForce([]string{"setup", "--reconfigure"}) {
		t.Fatal("expected --reconfigure to force setup")
	}
	if !wantsSetupForce([]string{"setup", "--force"}) {
		t.Fatal("expected --force to force setup")
	}
	if wantsSetupForce([]string{"setup"}) {
		t.Fatal("plain setup should not force")
	}
}

func TestWantsJSON(t *testing.T) {
	if !wantsJSON([]string{"doctor", "--json"}) {
		t.Fatal("expected --json")
	}
	if wantsJSON([]string{"doctor"}) {
		t.Fatal("plain doctor should not request json")
	}
}

func TestGatewayModelSettingsCommands(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := config.Default()
	cfg.AI.ProviderPriority = []string{"codex", "litellm", "openrouter", "claude-code"}
	cfg.AI.Codex.Enabled = true
	cfg.AI.Codex.Model = "gpt-5.5"
	data, err := config.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := runGatewayModelSettings(path, &out); err != nil {
		t.Fatalf("runGatewayModelSettings: %v", err)
	}
	var resp gatewayModelSettingsResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if resp.Settings == nil || resp.Settings.Active.Provider != "codex" {
		t.Fatalf("settings response = %#v", resp)
	}
	if resp.Settings.ConfigRevision == "" {
		t.Fatalf("settings response missing config revision = %#v", resp)
	}

	priority := []string{"litellm", "codex", "openrouter", "claude-code"}
	litellmEnabled := true
	litellmModel := "grok-4.1-fast-updated"
	update := config.ModelRoutingUpdate{
		BaseRevision:     resp.Settings.ConfigRevision,
		ProviderPriority: &priority,
		Providers: []config.ModelProviderUpdate{
			{ID: "litellm", Enabled: &litellmEnabled, Model: &litellmModel},
		},
	}
	input, err := json.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := runGatewayModelSettingsUpdate(path, bytes.NewReader(input), &out); err != nil {
		t.Fatalf("runGatewayModelSettingsUpdate: %v", err)
	}
	resp = gatewayModelSettingsResponse{}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("decode update: %v", err)
	}
	if resp.Settings == nil || resp.Settings.Active.Provider != "litellm" || resp.Settings.Active.NarrativeModel != litellmModel {
		t.Fatalf("updated settings response = %#v", resp)
	}
	saved, err := config.LoadForEdit(path)
	if err != nil {
		t.Fatal(err)
	}
	if saved.AI.LiteLLM.DefaultModel != litellmModel {
		t.Fatalf("saved LiteLLM model = %q", saved.AI.LiteLLM.DefaultModel)
	}

	out.Reset()
	update.BaseRevision = "stale"
	input, err = json.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}
	if err := runGatewayModelSettingsUpdate(path, bytes.NewReader(input), &out); err == nil {
		t.Fatal("expected stale update to return an error")
	}
	resp = gatewayModelSettingsResponse{}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("decode stale update: %v", err)
	}
	if resp.ErrorCode != config.ModelRoutingErrorStale {
		t.Fatalf("stale ErrorCode = %q, want %q", resp.ErrorCode, config.ModelRoutingErrorStale)
	}
}
