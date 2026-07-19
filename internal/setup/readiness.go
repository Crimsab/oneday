package setup

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/crimsab/oneday/internal/ai"
	"github.com/crimsab/oneday/internal/ai/providers"
	"github.com/crimsab/oneday/internal/aifactory"
	"github.com/crimsab/oneday/internal/config"
)

type Status string

const (
	StatusReady   Status = "ready"
	StatusWarning Status = "warning"
	StatusFailed  Status = "failed"
	StatusSkipped Status = "skipped"
)

// Probe is a stable, presentation-neutral readiness result. Summary and code
// are deliberately sanitized so doctor output can be shared safely.
type Probe struct {
	Name     string `json:"name"`
	Code     string `json:"code"`
	Status   Status `json:"status"`
	Required bool   `json:"required"`
	Summary  string `json:"summary"`
}

type Report struct {
	ConfigSource string  `json:"config_source"`
	Probes       []Probe `json:"probes"`
}

func (r Report) RequiredFailure() bool {
	for _, probe := range r.Probes {
		if probe.Required && probe.Status == StatusFailed {
			return true
		}
	}
	return false
}

type Dependencies struct {
	Narrative  func(context.Context, config.Config) error
	Embedding  func(context.Context, aifactory.EmbeddingProviderSpec) (int, error)
	HTTPGet    func(context.Context, string) error
	Stat       func(string) (os.FileInfo, error)
	GatewayURL string
}

func DefaultDependencies() Dependencies {
	return Dependencies{
		Narrative:  probeNarrative,
		Embedding:  probeEmbedding,
		HTTPGet:    probeHTTPGet,
		Stat:       os.Stat,
		GatewayURL: strings.TrimSpace(os.Getenv("ONEDAY_GATEWAY_URL")),
	}
}

func Run(ctx context.Context, cfg config.Config, configSource string, deps Dependencies) Report {
	deps = withDefaults(deps)
	report := Report{ConfigSource: configSource, Probes: make([]Probe, 0, 7)}
	report.Probes = append(report.Probes, narrativeProbe(ctx, cfg, deps))
	report.Probes = append(report.Probes, embeddingProbe(ctx, cfg, deps))
	report.Probes = append(report.Probes, imageProbe(ctx, cfg, deps))
	report.Probes = append(report.Probes, ttsProbe(ctx, cfg, deps))
	report.Probes = append(report.Probes, gatewayProbe(ctx, deps))
	report.Probes = append(report.Probes, storageProbe(cfg, deps))
	report.Probes = append(report.Probes, backupProbe(cfg, deps))
	return report
}

func withDefaults(deps Dependencies) Dependencies {
	defaults := DefaultDependencies()
	if deps.Narrative == nil {
		deps.Narrative = defaults.Narrative
	}
	if deps.Embedding == nil {
		deps.Embedding = defaults.Embedding
	}
	if deps.HTTPGet == nil {
		deps.HTTPGet = defaults.HTTPGet
	}
	if deps.Stat == nil {
		deps.Stat = defaults.Stat
	}
	return deps
}

func narrativeProbe(ctx context.Context, cfg config.Config, deps Dependencies) Probe {
	if len(cfg.EnabledProviders()) == 0 {
		return Probe{"narrative", "NARRATIVE_NOT_CONFIGURED", StatusFailed, true, "no narrative provider is enabled"}
	}
	if err := deps.Narrative(ctx, cfg); err != nil {
		return Probe{"narrative", "NARRATIVE_UNAVAILABLE", StatusFailed, true, "enabled narrative provider did not pass its readiness check"}
	}
	return Probe{"narrative", "NARRATIVE_READY", StatusReady, true, "narrative provider is ready"}
}

func embeddingProbe(ctx context.Context, cfg config.Config, deps Dependencies) Probe {
	if !cfg.RAG.Enabled {
		return Probe{"embeddings", "EMBEDDINGS_DISABLED", StatusSkipped, false, "RAG embeddings are disabled"}
	}
	spec, reason := aifactory.SelectEmbeddingProvider(cfg)
	if reason != "" {
		return Probe{"embeddings", "EMBEDDINGS_NOT_CONFIGURED", StatusWarning, false, "no usable embedding provider is configured"}
	}
	dimensions, err := deps.Embedding(ctx, spec)
	if err != nil {
		return Probe{"embeddings", "EMBEDDINGS_UNAVAILABLE", StatusWarning, false, "embedding provider did not pass its readiness check"}
	}
	if spec.Dimensions > 0 && dimensions != spec.Dimensions {
		return Probe{"embeddings", "EMBEDDINGS_DIMENSION_MISMATCH", StatusWarning, false, "embedding dimensions differ from configuration"}
	}
	return Probe{"embeddings", "EMBEDDINGS_READY", StatusReady, false, "embedding provider is ready"}
}

func imageProbe(ctx context.Context, cfg config.Config, deps Dependencies) Probe {
	image := cfg.AI.ImageGeneration
	if !image.AutoGenerate {
		return Probe{"image", "IMAGE_DISABLED", StatusSkipped, false, "image generation is disabled"}
	}
	endpoint := image.ImagegenBridgeURL
	if strings.TrimSpace(endpoint) == "" {
		endpoint = image.BaseURL
	}
	if strings.TrimSpace(endpoint) == "" {
		return Probe{"image", "IMAGE_NOT_CONFIGURED", StatusWarning, false, "image generation has no endpoint configured"}
	}
	if strings.EqualFold(image.Provider, "codex-oauth") {
		if err := deps.HTTPGet(ctx, joinURL(endpoint, "/health/ready")); err != nil {
			return Probe{"image", "IMAGE_UNAVAILABLE", StatusWarning, false, "image bridge did not pass its readiness check"}
		}
	}
	return Probe{"image", "IMAGE_READY", StatusReady, false, "image generation is configured"}
}

func ttsProbe(ctx context.Context, cfg config.Config, deps Dependencies) Probe {
	tts := cfg.AI.TTS
	if !tts.Cloud.Enabled && !tts.Local.Enabled {
		return Probe{"tts", "TTS_DISABLED", StatusSkipped, false, "text-to-speech is disabled"}
	}
	if tts.Local.Enabled {
		if strings.TrimSpace(tts.Local.BaseURL) == "" {
			return Probe{"tts", "TTS_NOT_CONFIGURED", StatusWarning, false, "local text-to-speech has no endpoint configured"}
		}
		if err := deps.HTTPGet(ctx, joinURL(tts.Local.BaseURL, "/voices")); err != nil {
			return Probe{"tts", "TTS_UNAVAILABLE", StatusWarning, false, "local text-to-speech did not pass its readiness check"}
		}
	}
	if tts.Cloud.Enabled && (strings.TrimSpace(tts.Cloud.BaseURL) == "" || strings.TrimSpace(tts.Cloud.APIKey) == "") {
		return Probe{"tts", "TTS_NOT_CONFIGURED", StatusWarning, false, "cloud text-to-speech requires an endpoint and API key"}
	}
	return Probe{"tts", "TTS_READY", StatusReady, false, "text-to-speech is configured"}
}

func gatewayProbe(ctx context.Context, deps Dependencies) Probe {
	if strings.TrimSpace(deps.GatewayURL) == "" {
		return Probe{"gateway", "GATEWAY_NOT_CONFIGURED", StatusSkipped, false, "set ONEDAY_GATEWAY_URL to check a running browser gateway"}
	}
	if err := deps.HTTPGet(ctx, joinURL(deps.GatewayURL, "/api/health")); err != nil {
		return Probe{"gateway", "GATEWAY_UNAVAILABLE", StatusWarning, false, "gateway did not pass its health check"}
	}
	return Probe{"gateway", "GATEWAY_READY", StatusReady, false, "gateway is ready"}
}

func storageProbe(cfg config.Config, deps Dependencies) Probe {
	dataDir := cfg.DataDir
	info, err := deps.Stat(dataDir)
	if err == nil {
		if !info.IsDir() {
			return Probe{"storage", "STORAGE_NOT_DIRECTORY", StatusFailed, true, "data directory path is not a directory"}
		}
		return Probe{"storage", "STORAGE_READY", StatusReady, true, "data directory is available"}
	}
	if !errors.Is(err, os.ErrNotExist) {
		return Probe{"storage", "STORAGE_UNAVAILABLE", StatusFailed, true, "data directory cannot be inspected"}
	}
	parent, parentErr := deps.Stat(filepath.Dir(dataDir))
	if parentErr != nil || !parent.IsDir() {
		return Probe{"storage", "STORAGE_PARENT_UNAVAILABLE", StatusFailed, true, "data directory parent is unavailable"}
	}
	return Probe{"storage", "STORAGE_INITIALIZABLE", StatusReady, true, "data directory will be created on first start"}
}

func backupProbe(cfg config.Config, deps Dependencies) Probe {
	info, err := deps.Stat(filepath.Join(cfg.DataDir, "oneday.db"))
	if errors.Is(err, os.ErrNotExist) {
		return Probe{"backup", "BACKUP_NO_DATABASE", StatusSkipped, false, "no database exists yet to back up"}
	}
	if err != nil {
		return Probe{"backup", "BACKUP_UNAVAILABLE", StatusWarning, false, "database cannot be inspected for backup readiness"}
	}
	if info.IsDir() {
		return Probe{"backup", "BACKUP_NOT_FILE", StatusWarning, false, "database path is not a regular file"}
	}
	return Probe{"backup", "BACKUP_READY", StatusReady, false, "database is available for a SQLite-safe backup"}
}

func probeNarrative(ctx context.Context, cfg config.Config) error {
	router, err := aifactory.NewRouterFromConfig(cfg)
	if err != nil {
		return err
	}
	_, err = router.Complete(ctx, ai.Request{Messages: []ai.Message{{Role: ai.RoleUser, Content: "reply with OK"}}, MaxTokens: 8})
	return err
}

func probeEmbedding(ctx context.Context, spec aifactory.EmbeddingProviderSpec) (int, error) {
	var embedding interface {
		Embed(context.Context, ai.EmbeddingRequest) (ai.EmbeddingResponse, error)
	}
	switch spec.Kind {
	case "ollama":
		embedding = providers.NewOllamaEmbedding(providers.OllamaEmbeddingConfig{BaseURL: spec.BaseURL, Model: spec.Model, Timeout: 20 * time.Second})
	case "custom":
		embedding = providers.NewLocalHTTPEmbedding(spec.BaseURL, spec.Model, 20*time.Second)
	default:
		embedding = providers.NewOpenAICompat(providers.OpenAICompatConfig{Name: spec.Name, BaseURL: spec.BaseURL, APIKey: spec.APIKey, DefaultModel: spec.Model, Timeout: 20 * time.Second})
	}
	response, err := embedding.Embed(ctx, ai.EmbeddingRequest{Input: "oneday doctor embedding smoke", Model: spec.Model})
	return len(response.Embedding), err
}

func probeHTTPGet(ctx context.Context, rawURL string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", response.StatusCode)
	}
	return nil
}

func joinURL(base, suffix string) string {
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil {
		return base
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + suffix
	return parsed.String()
}
