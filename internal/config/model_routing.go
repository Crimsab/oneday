package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	ModelRoutingErrorValidation = "validation_failed"
	ModelRoutingErrorStale      = "stale_config"
	ModelRoutingErrorWrite      = "write_failed"
	ModelRoutingErrorLocked     = "config_locked"
)

var providerOrder = []string{"codex", "litellm", "openrouter", "claude-code"}

type ModelRoutingError struct {
	Code string
	Err  error
}

func (e ModelRoutingError) Error() string {
	if e.Err == nil {
		return e.Code
	}
	return e.Err.Error()
}

func (e ModelRoutingError) Unwrap() error {
	return e.Err
}

type ModelProviderSetting struct {
	ID                string `json:"id"`
	Label             string `json:"label"`
	Enabled           bool   `json:"enabled"`
	Model             string `json:"model,omitempty"`
	Reasoning         string `json:"reasoning,omitempty"`
	SupportsModel     bool   `json:"supports_model"`
	SupportsReasoning bool   `json:"supports_reasoning"`
}

type ModelRoutingActive struct {
	Provider             string   `json:"provider"`
	NarrativeModel       string   `json:"narrative_model"`
	UtilityModel         string   `json:"utility_model"`
	RepairModel          string   `json:"repair_model"`
	RepairFallbackModels []string `json:"repair_fallback_models"`
	ImageModel           string   `json:"image_model"`
	EmbeddingProvider    string   `json:"embedding_provider"`
	EmbeddingModel       string   `json:"embedding_model"`
	CodexReasoning       string   `json:"codex_reasoning"`
}

type ModelRoutingSettings struct {
	ConfigPath         string                 `json:"config_path"`
	ConfigRevision     string                 `json:"config_revision"`
	ProviderPriority   []string               `json:"provider_priority"`
	Providers          []ModelProviderSetting `json:"providers"`
	NarrativeModels    []string               `json:"narrative_models"`
	UtilityModels      []string               `json:"utility_models"`
	RepairModels       []string               `json:"repair_models"`
	ImageModels        []string               `json:"image_models"`
	EmbeddingProviders []string               `json:"embedding_providers"`
	Active             ModelRoutingActive     `json:"active"`
	TTSStatus          string                 `json:"tts_status"`
}

type ModelProviderUpdate struct {
	ID        string  `json:"id"`
	Enabled   *bool   `json:"enabled,omitempty"`
	Model     *string `json:"model,omitempty"`
	Reasoning *string `json:"reasoning,omitempty"`
}

type ModelRoutingUpdate struct {
	BaseRevision         string                `json:"base_revision,omitempty"`
	ProviderPriority     *[]string             `json:"provider_priority,omitempty"`
	Providers            []ModelProviderUpdate `json:"providers,omitempty"`
	UtilityModel         *string               `json:"utility_model,omitempty"`
	RepairModel          *string               `json:"repair_model,omitempty"`
	RepairFallbackModels *[]string             `json:"repair_fallback_models,omitempty"`
	ImageModel           *string               `json:"image_model,omitempty"`
	EmbeddingProvider    *string               `json:"embedding_provider,omitempty"`
	EmbeddingModel       *string               `json:"embedding_model,omitempty"`
}

func ReadModelRoutingSettings(path string) (ModelRoutingSettings, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return ModelRoutingSettings{}, fmt.Errorf("reading config %s: %w", path, err)
		}
		raw = nil
	}
	cfg, err := configFromEditBytes(path, raw)
	if err != nil {
		return ModelRoutingSettings{}, err
	}
	return BuildModelRoutingSettings(path, cfg, ConfigRevision(raw)), nil
}

func UpdateModelRoutingSettings(path string, update ModelRoutingUpdate) (ModelRoutingSettings, error) {
	var settings ModelRoutingSettings
	err := withConfigLock(path, func() error {
		raw, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return ModelRoutingError{Code: ModelRoutingErrorWrite, Err: fmt.Errorf("reading config %s: %w", path, err)}
			}
			raw = nil
		}
		revision := ConfigRevision(raw)
		if strings.TrimSpace(update.BaseRevision) != "" && update.BaseRevision != revision {
			return ModelRoutingError{Code: ModelRoutingErrorStale, Err: fmt.Errorf("config changed on disk; reload before saving")}
		}

		cfg, err := configFromEditBytes(path, raw)
		if err != nil {
			return err
		}
		if err := ApplyModelRoutingUpdate(&cfg, update); err != nil {
			return ModelRoutingError{Code: ModelRoutingErrorValidation, Err: err}
		}

		nextRaw, err := patchModelRoutingYAML(raw, cfg)
		if err != nil {
			return ModelRoutingError{Code: ModelRoutingErrorWrite, Err: err}
		}
		if _, err := configFromEditBytes(path, nextRaw); err != nil {
			return err
		}
		if err := writeConfigAtomic(path, raw, nextRaw); err != nil {
			return ModelRoutingError{Code: ModelRoutingErrorWrite, Err: err}
		}
		nextRevision := ConfigRevision(nextRaw)
		settings = BuildModelRoutingSettings(path, cfg, nextRevision)
		return nil
	})
	return settings, err
}

func ConfigRevision(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func LoadForEdit(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return Config{}, fmt.Errorf("reading config %s: %w", path, err)
	}
	return configFromEditBytes(path, raw)
}

func SaveForEdit(path string, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validating config: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("serializing config: %w", err)
	}
	raw, _ := os.ReadFile(path)
	return writeConfigAtomic(path, raw, data)
}

func BuildModelRoutingSettings(path string, cfg Config, revision string) ModelRoutingSettings {
	activeProvider := ""
	if enabled := cfg.EnabledProviders(); len(enabled) > 0 {
		activeProvider = enabled[0]
	}
	activeNarrative := providerModel(cfg, activeProvider)
	repairModels := cfg.AI.Generation.RepairModelCandidates()

	return ModelRoutingSettings{
		ConfigPath:       path,
		ConfigRevision:   revision,
		ProviderPriority: append([]string(nil), cfg.AI.ProviderPriority...),
		Providers: []ModelProviderSetting{
			{
				ID:                "codex",
				Label:             "Codex OAuth",
				Enabled:           cfg.AI.Codex.Enabled,
				Model:             cfg.AI.Codex.Model,
				Reasoning:         cfg.AI.Codex.Reasoning,
				SupportsModel:     true,
				SupportsReasoning: true,
			},
			{
				ID:            "litellm",
				Label:         "LiteLLM",
				Enabled:       cfg.AI.LiteLLM.Enabled,
				Model:         cfg.AI.LiteLLM.DefaultModel,
				SupportsModel: true,
			},
			{
				ID:            "openrouter",
				Label:         "OpenRouter",
				Enabled:       cfg.AI.OpenRouter.Enabled,
				Model:         cfg.AI.OpenRouter.DefaultModel,
				SupportsModel: true,
			},
			{
				ID:            "claude-code",
				Label:         "Claude Code",
				Enabled:       cfg.AI.ClaudeCode.Enabled,
				SupportsModel: false,
			},
		},
		NarrativeModels:    uniqueNonEmpty(providerModel(cfg, "codex"), providerModel(cfg, "litellm"), providerModel(cfg, "openrouter"), activeNarrative),
		UtilityModels:      uniqueNonEmpty(cfg.AI.Generation.UtilityModel, activeNarrative, providerModel(cfg, "litellm"), providerModel(cfg, "openrouter")),
		RepairModels:       uniqueNonEmpty(append([]string{cfg.AI.Generation.RepairModel, cfg.AI.Generation.UtilityModel}, cfg.AI.Generation.RepairFallbackModels...)...),
		ImageModels:        uniqueNonEmpty(cfg.AI.ASCIIArt.Model, activeNarrative),
		EmbeddingProviders: []string{"auto", "litellm", "openrouter", "local"},
		Active: ModelRoutingActive{
			Provider:             activeProvider,
			NarrativeModel:       activeNarrative,
			UtilityModel:         cfg.AI.Generation.UtilityModel,
			RepairModel:          firstNonEmpty(cfg.AI.Generation.RepairModel, firstString(repairModels)),
			RepairFallbackModels: append([]string(nil), cfg.AI.Generation.RepairFallbackModels...),
			ImageModel:           cfg.AI.ASCIIArt.Model,
			EmbeddingProvider:    firstNonEmpty(cfg.AI.Embedding.Provider, "auto"),
			EmbeddingModel:       cfg.AI.Embedding.Model,
			CodexReasoning:       firstNonEmpty(cfg.AI.Codex.Reasoning, "off"),
		},
		TTSStatus: "planned",
	}
}

func ApplyModelRoutingUpdate(cfg *Config, update ModelRoutingUpdate) error {
	if update.ProviderPriority != nil {
		priority, err := cleanProviderPriority(*update.ProviderPriority)
		if err != nil {
			return err
		}
		cfg.AI.ProviderPriority = priority
	}
	for _, provider := range update.Providers {
		id := strings.TrimSpace(provider.ID)
		switch id {
		case "codex":
			if provider.Enabled != nil {
				cfg.AI.Codex.Enabled = *provider.Enabled
			}
			if provider.Model != nil {
				cfg.AI.Codex.Model = cleanString(*provider.Model)
			}
			if provider.Reasoning != nil {
				cfg.AI.Codex.Reasoning = cleanString(*provider.Reasoning)
			}
		case "litellm":
			if provider.Enabled != nil {
				cfg.AI.LiteLLM.Enabled = *provider.Enabled
			}
			if provider.Model != nil {
				cfg.AI.LiteLLM.DefaultModel = cleanString(*provider.Model)
			}
		case "openrouter":
			if provider.Enabled != nil {
				cfg.AI.OpenRouter.Enabled = *provider.Enabled
			}
			if provider.Model != nil {
				cfg.AI.OpenRouter.DefaultModel = cleanString(*provider.Model)
			}
		case "claude-code":
			if provider.Enabled != nil {
				cfg.AI.ClaudeCode.Enabled = *provider.Enabled
			}
		default:
			return fmt.Errorf("unknown provider %q", provider.ID)
		}
	}
	if update.UtilityModel != nil {
		cfg.AI.Generation.UtilityModel = cleanString(*update.UtilityModel)
	}
	if update.RepairModel != nil {
		cfg.AI.Generation.RepairModel = cleanString(*update.RepairModel)
	}
	if update.RepairFallbackModels != nil {
		cfg.AI.Generation.RepairFallbackModels = cleanStringSlice(*update.RepairFallbackModels)
	}
	if update.ImageModel != nil {
		cfg.AI.ASCIIArt.Model = cleanString(*update.ImageModel)
	}
	if update.EmbeddingProvider != nil {
		cfg.AI.Embedding.Provider = cleanString(*update.EmbeddingProvider)
	}
	if update.EmbeddingModel != nil {
		cfg.AI.Embedding.Model = cleanString(*update.EmbeddingModel)
	}
	if cfg.AI.Embedding.Provider == "local" && !cfg.AI.Embedding.Local.Enabled {
		return fmt.Errorf("ai.embedding.local.enabled must be true when ai.embedding.provider is local")
	}
	if len(cfg.EnabledProviders()) == 0 {
		return fmt.Errorf("at least one provider must be enabled")
	}
	return cfg.Validate()
}

func configFromEditBytes(path string, raw []byte) (Config, error) {
	cfg := Default()
	if len(raw) > 0 {
		if err := yaml.Unmarshal(raw, &cfg); err != nil {
			return cfg, fmt.Errorf("parsing config %s: %w", path, err)
		}
	}
	cfg.Migrate()
	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("validating config: %w", err)
	}
	return cfg, nil
}

func patchModelRoutingYAML(raw []byte, cfg Config) ([]byte, error) {
	var doc yaml.Node
	if len(raw) == 0 {
		raw = []byte("config_version: 2\n")
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parsing YAML document: %w", err)
	}
	root := documentRoot(&doc)
	setStringSlice(root, cfg.AI.ProviderPriority, "ai", "provider_priority")
	setBool(root, cfg.AI.Codex.Enabled, "ai", "codex", "enabled")
	setString(root, cfg.AI.Codex.Model, "ai", "codex", "model")
	setString(root, cfg.AI.Codex.Reasoning, "ai", "codex", "reasoning")
	setBool(root, cfg.AI.LiteLLM.Enabled, "ai", "litellm", "enabled")
	setString(root, cfg.AI.LiteLLM.DefaultModel, "ai", "litellm", "default_model")
	setBool(root, cfg.AI.OpenRouter.Enabled, "ai", "openrouter", "enabled")
	setString(root, cfg.AI.OpenRouter.DefaultModel, "ai", "openrouter", "default_model")
	setBool(root, cfg.AI.ClaudeCode.Enabled, "ai", "claude_code", "enabled")
	setString(root, cfg.AI.Generation.UtilityModel, "ai", "generation", "utility_model")
	setString(root, cfg.AI.Generation.RepairModel, "ai", "generation", "repair_model")
	setStringSlice(root, cfg.AI.Generation.RepairFallbackModels, "ai", "generation", "repair_fallback_models")
	setString(root, cfg.AI.ASCIIArt.Model, "ai", "ascii_art", "model")
	setString(root, cfg.AI.Embedding.Provider, "ai", "embedding", "provider")
	setString(root, cfg.AI.Embedding.Model, "ai", "embedding", "model")
	var out strings.Builder
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if err := encoder.Encode(&doc); err != nil {
		_ = encoder.Close()
		return nil, fmt.Errorf("encoding YAML document: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("closing YAML encoder: %w", err)
	}
	return []byte(out.String()), nil
}

func documentRoot(doc *yaml.Node) *yaml.Node {
	if doc.Kind != yaml.DocumentNode {
		doc.Kind = yaml.DocumentNode
	}
	if len(doc.Content) == 0 || doc.Content[0] == nil {
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode}}
	}
	if doc.Content[0].Kind != yaml.MappingNode {
		doc.Content[0].Kind = yaml.MappingNode
		doc.Content[0].Content = nil
	}
	return doc.Content[0]
}

func setString(root *yaml.Node, value string, path ...string) {
	node := ensurePath(root, path)
	node.Kind = yaml.ScalarNode
	node.Tag = "!!str"
	node.Value = value
}

func setBool(root *yaml.Node, value bool, path ...string) {
	node := ensurePath(root, path)
	node.Kind = yaml.ScalarNode
	node.Tag = "!!bool"
	if value {
		node.Value = "true"
	} else {
		node.Value = "false"
	}
}

func setStringSlice(root *yaml.Node, values []string, path ...string) {
	node := ensurePath(root, path)
	node.Kind = yaml.SequenceNode
	node.Tag = "!!seq"
	node.Content = node.Content[:0]
	for _, value := range values {
		node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value})
	}
}

func ensurePath(root *yaml.Node, path []string) *yaml.Node {
	current := root
	for index, key := range path {
		value := mappingValue(current, key)
		if value == nil {
			value = &yaml.Node{Kind: yaml.MappingNode}
			current.Content = append(current.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, value)
		}
		if index == len(path)-1 {
			return value
		}
		if value.Kind != yaml.MappingNode {
			value.Kind = yaml.MappingNode
			value.Tag = "!!map"
			value.Value = ""
			value.Content = nil
		}
		current = value
	}
	return current
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node.Kind != yaml.MappingNode {
		node.Kind = yaml.MappingNode
		node.Tag = "!!map"
		node.Content = nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func writeConfigAtomic(path string, oldRaw, nextRaw []byte) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	mode := os.FileMode(0600)
	uid, gid := -1, -1
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			uid = int(stat.Uid)
			gid = int(stat.Gid)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat config %s: %w", path, err)
	}
	if len(oldRaw) > 0 {
		_ = writeBackup(path+".bak", oldRaw, mode, uid, gid)
	}
	tmp, err := os.CreateTemp(dir, base+".tmp-*")
	if err != nil {
		return fmt.Errorf("creating temp config: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp config: %w", err)
	}
	if uid >= 0 && os.Geteuid() == 0 {
		if err := tmp.Chown(uid, gid); err != nil {
			_ = tmp.Close()
			return fmt.Errorf("chown temp config: %w", err)
		}
	}
	if _, err := tmp.Write(nextRaw); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing temp config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("syncing temp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp config: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replacing config: %w", err)
	}
	cleanup = false
	return fsyncDir(dir)
}

func writeBackup(path string, data []byte, mode os.FileMode, uid, gid int) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	if uid >= 0 && os.Geteuid() == 0 {
		if err := os.Chown(tmp, uid, gid); err != nil {
			_ = os.Remove(tmp)
			return err
		}
	}
	return os.Rename(tmp, path)
}

func fsyncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func withConfigLock(path string, fn func() error) error {
	lockPath := path + ".lock"
	deadline := time.Now().Add(5 * time.Second)
	for {
		file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			_, _ = fmt.Fprintf(file, "%d\n", os.Getpid())
			_ = file.Close()
			defer os.Remove(lockPath)
			return fn()
		}
		if !os.IsExist(err) {
			return ModelRoutingError{Code: ModelRoutingErrorLocked, Err: fmt.Errorf("creating config lock: %w", err)}
		}
		if time.Now().After(deadline) {
			return ModelRoutingError{Code: ModelRoutingErrorLocked, Err: fmt.Errorf("config lock is held: %s", lockPath)}
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func cleanProviderPriority(values []string) ([]string, error) {
	out := cleanStringSlice(values)
	if len(out) == 0 {
		return nil, fmt.Errorf("ai.provider_priority must have at least one provider")
	}
	for _, provider := range out {
		if !validProviders[provider] {
			return nil, fmt.Errorf("unknown provider in priority chain: %q", provider)
		}
	}
	for _, provider := range providerOrder {
		if !containsString(out, provider) {
			out = append(out, provider)
		}
	}
	return out, nil
}

func providerModel(cfg Config, provider string) string {
	switch provider {
	case "codex":
		return cfg.AI.Codex.Model
	case "litellm":
		return cfg.AI.LiteLLM.DefaultModel
	case "openrouter":
		return cfg.AI.OpenRouter.DefaultModel
	default:
		return ""
	}
}

func cleanString(value string) string {
	return strings.TrimSpace(value)
}

func cleanStringSlice(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		clean := cleanString(value)
		if clean == "" || seen[clean] {
			continue
		}
		seen[clean] = true
		out = append(out, clean)
	}
	return out
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func uniqueNonEmpty(values ...string) []string {
	return cleanStringSlice(values)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if clean := cleanString(value); clean != "" {
			return clean
		}
	}
	return ""
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
