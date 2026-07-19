package setup

import (
	"context"
	"errors"
	"net"
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
	// Action is a stable, redacted recovery instruction for CLI, JSON, and web
	// consumers. It never contains a provider response, credential, or path.
	Action string `json:"action"`
}

const (
	ActionConfigure          = "configure"
	ActionCheckCredentials   = "check_credentials"
	ActionCheckConnection    = "check_connection"
	ActionRetryLater         = "retry_later"
	ActionCheckCapability    = "check_capability"
	ActionReviewBilling      = "review_billing"
	ActionCreateBackup       = "create_backup"
	ActionRestoreEmptyTarget = "restore_empty_target"
	ActionPreserveOriginal   = "preserve_original"
)

// ProviderDiagnosticKind lets adapters communicate a safe, actionable
// category without making the raw provider error part of the public contract.
type ProviderDiagnosticKind string

const (
	ProviderMissingCredential ProviderDiagnosticKind = "missing_credential"
	ProviderUnreachable       ProviderDiagnosticKind = "unreachable"
	ProviderTimeout           ProviderDiagnosticKind = "timeout"
	ProviderIncompatible      ProviderDiagnosticKind = "incompatible"
	ProviderAmbiguousPaid     ProviderDiagnosticKind = "ambiguous_paid_outcome"
)

type ProviderDiagnosticError struct{ Kind ProviderDiagnosticKind }

func (e ProviderDiagnosticError) Error() string { return string(e.Kind) }

// ProviderFailure is for provider adapters that can classify a failure without
// exposing the provider response in doctor output.
func ProviderFailure(kind ProviderDiagnosticKind) error { return ProviderDiagnosticError{Kind: kind} }

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
	Narrative    func(context.Context, config.Config) error
	Embedding    func(context.Context, aifactory.EmbeddingProviderSpec) (int, error)
	HTTPGet      func(context.Context, string) error
	Stat         func(string) (os.FileInfo, error)
	GatewayURL   string
	DatabasePath string
}

func DefaultDependencies() Dependencies {
	return Dependencies{
		Narrative:    probeNarrative,
		Embedding:    probeEmbedding,
		HTTPGet:      probeHTTPGet,
		Stat:         os.Stat,
		GatewayURL:   strings.TrimSpace(os.Getenv("ONEDAY_GATEWAY_URL")),
		DatabasePath: strings.TrimSpace(os.Getenv("ONEDAY_DB_PATH")),
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
	if deps.DatabasePath == "" {
		deps.DatabasePath = defaults.DatabasePath
	}
	return deps
}

func narrativeProbe(ctx context.Context, cfg config.Config, deps Dependencies) Probe {
	if len(cfg.EnabledProviders()) == 0 {
		return newProbe("narrative", "NARRATIVE_NOT_CONFIGURED", StatusFailed, true, "no narrative provider is enabled")
	}
	if narrativeCredentialMissing(cfg) {
		return providerFailureProbe("narrative", "NARRATIVE", StatusFailed, true, ProviderFailure(ProviderMissingCredential))
	}
	if err := deps.Narrative(ctx, cfg); err != nil {
		return providerFailureProbe("narrative", "NARRATIVE", StatusFailed, true, err)
	}
	return newProbe("narrative", "NARRATIVE_READY", StatusReady, true, "narrative provider is ready")
}

func embeddingProbe(ctx context.Context, cfg config.Config, deps Dependencies) Probe {
	if !cfg.RAG.Enabled {
		return newProbe("embeddings", "EMBEDDINGS_DISABLED", StatusSkipped, false, "RAG embeddings are disabled")
	}
	spec, reason := aifactory.SelectEmbeddingProvider(cfg)
	if reason != "" {
		return newProbe("embeddings", "EMBEDDINGS_NOT_CONFIGURED", StatusWarning, false, "no usable embedding provider is configured")
	}
	dimensions, err := deps.Embedding(ctx, spec)
	if err != nil {
		return providerFailureProbe("embeddings", "EMBEDDINGS", StatusWarning, false, err)
	}
	if spec.Dimensions > 0 && dimensions != spec.Dimensions {
		return newProbe("embeddings", "EMBEDDINGS_DIMENSION_MISMATCH", StatusWarning, false, "embedding dimensions differ from configuration")
	}
	return newProbe("embeddings", "EMBEDDINGS_READY", StatusReady, false, "embedding provider is ready")
}

func imageProbe(ctx context.Context, cfg config.Config, deps Dependencies) Probe {
	image := cfg.AI.ImageGeneration
	if !image.AutoGenerate {
		return newProbe("image", "IMAGE_DISABLED", StatusSkipped, false, "image generation is disabled")
	}
	endpoint := image.ImagegenBridgeURL
	if strings.TrimSpace(endpoint) == "" {
		endpoint = image.BaseURL
	}
	if strings.TrimSpace(endpoint) == "" {
		return newProbe("image", "IMAGE_NOT_CONFIGURED", StatusWarning, false, "image generation has no endpoint configured")
	}
	if strings.EqualFold(image.Provider, "codex-oauth") {
		if err := deps.HTTPGet(ctx, joinURL(endpoint, "/health/ready")); err != nil {
			return providerFailureProbe("image", "IMAGE", StatusWarning, false, err)
		}
	}
	return newProbe("image", "IMAGE_READY", StatusReady, false, "image generation is configured")
}

func ttsProbe(ctx context.Context, cfg config.Config, deps Dependencies) Probe {
	tts := cfg.AI.TTS
	if !tts.Cloud.Enabled && !tts.Local.Enabled {
		return newProbe("tts", "TTS_DISABLED", StatusSkipped, false, "text-to-speech is disabled")
	}
	if tts.Local.Enabled {
		if strings.TrimSpace(tts.Local.BaseURL) == "" {
			return newProbe("tts", "TTS_NOT_CONFIGURED", StatusWarning, false, "local text-to-speech has no endpoint configured")
		}
		if err := deps.HTTPGet(ctx, joinURL(tts.Local.BaseURL, "/voices")); err != nil {
			return providerFailureProbe("tts", "TTS", StatusWarning, false, err)
		}
	}
	if tts.Cloud.Enabled && (strings.TrimSpace(tts.Cloud.BaseURL) == "" || strings.TrimSpace(tts.Cloud.APIKey) == "") {
		return providerFailureProbe("tts", "TTS", StatusWarning, false, ProviderFailure(ProviderMissingCredential))
	}
	return newProbe("tts", "TTS_READY", StatusReady, false, "text-to-speech is configured")
}

func gatewayProbe(ctx context.Context, deps Dependencies) Probe {
	if strings.TrimSpace(deps.GatewayURL) == "" {
		return newProbe("gateway", "GATEWAY_NOT_CONFIGURED", StatusSkipped, false, "gateway readiness is not configured")
	}
	if err := deps.HTTPGet(ctx, joinURL(deps.GatewayURL, "/api/health")); err != nil {
		return providerFailureProbe("gateway", "GATEWAY", StatusWarning, false, err)
	}
	return newProbe("gateway", "GATEWAY_READY", StatusReady, false, "gateway is ready")
}

func storageProbe(cfg config.Config, deps Dependencies) Probe {
	dataDir := cfg.DataDir
	info, err := deps.Stat(dataDir)
	if err == nil {
		if !info.IsDir() {
			return newProbe("storage", "STORAGE_NOT_DIRECTORY", StatusFailed, true, "data directory path is not a directory")
		}
		return newProbe("storage", "STORAGE_READY", StatusReady, true, "data directory is available")
	}
	if !errors.Is(err, os.ErrNotExist) {
		return newProbe("storage", "STORAGE_UNAVAILABLE", StatusFailed, true, "data directory cannot be inspected")
	}
	parent, parentErr := deps.Stat(filepath.Dir(dataDir))
	if parentErr != nil || !parent.IsDir() {
		return newProbe("storage", "STORAGE_PARENT_UNAVAILABLE", StatusFailed, true, "data directory parent is unavailable")
	}
	return newProbe("storage", "STORAGE_INITIALIZABLE", StatusReady, true, "data directory will be created on first start")
}

func backupProbe(cfg config.Config, deps Dependencies) Probe {
	info, err := deps.Stat(backupDatabasePath(cfg, deps))
	if errors.Is(err, os.ErrNotExist) {
		return newProbe("backup", "BACKUP_NO_DATABASE", StatusSkipped, false, "no database exists yet to back up")
	}
	if err != nil {
		return newProbe("backup", "BACKUP_UNAVAILABLE", StatusWarning, false, "database cannot be inspected for backup readiness")
	}
	if info.IsDir() {
		return newProbe("backup", "BACKUP_NOT_FILE", StatusWarning, false, "database path is not a regular file")
	}
	return newProbe("backup", "BACKUP_READY", StatusReady, false, "database is available for a SQLite-safe backup")
}

func newProbe(name, code string, status Status, required bool, summary string) Probe {
	return Probe{Name: name, Code: code, Status: status, Required: required, Summary: summary, Action: recoveryAction(code)}
}

func providerFailureProbe(name, prefix string, status Status, required bool, err error) Probe {
	kind := classifyProviderFailure(err)
	code := prefix + "_" + strings.ToUpper(string(kind))
	summary := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(string(kind), "_", " "), "paid outcome", "paid request outcome"))
	return newProbe(name, code, status, required, "provider readiness check reported "+summary)
}

func classifyProviderFailure(err error) ProviderDiagnosticKind {
	var diagnostic ProviderDiagnosticError
	if errors.As(err, &diagnostic) {
		return diagnostic.Kind
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ProviderTimeout
	}
	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		return ProviderTimeout
	}
	return ProviderUnreachable
}

func narrativeCredentialMissing(cfg config.Config) bool {
	enabled := cfg.EnabledProviders()
	if len(enabled) == 0 {
		return false
	}
	for _, provider := range enabled {
		if provider != "openrouter" || strings.TrimSpace(cfg.AI.OpenRouter.APIKey) != "" {
			return false
		}
	}
	return true
}

func recoveryAction(code string) string {
	switch {
	case code == "BACKUP_NO_DATABASE":
		return ""
	case code == "BACKUP_UNAVAILABLE":
		return ActionPreserveOriginal
	case code == "BACKUP_NOT_FILE":
		return ActionConfigure
	case strings.Contains(code, "NOT_CONFIGURED"):
		return ActionConfigure
	case strings.Contains(code, "MISSING_CREDENTIAL"):
		return ActionCheckCredentials
	case strings.Contains(code, "TIMEOUT"):
		return ActionRetryLater
	case strings.Contains(code, "UNREACHABLE"):
		return ActionCheckConnection
	case strings.Contains(code, "INCOMPATIBLE") || strings.Contains(code, "DIMENSION_MISMATCH"):
		return ActionCheckCapability
	case strings.Contains(code, "AMBIGUOUS_PAID_OUTCOME"):
		return ActionReviewBilling
	case code == "BACKUP_READY":
		return ActionCreateBackup
	case strings.HasPrefix(code, "BACKUP_"):
		return ActionRestoreEmptyTarget
	default:
		return ""
	}
}

func backupDatabasePath(cfg config.Config, deps Dependencies) string {
	if path := strings.TrimSpace(deps.DatabasePath); path != "" {
		return path
	}
	return filepath.Join(cfg.DataDir, "oneday.db")
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
		switch response.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return ProviderFailure(ProviderMissingCredential)
		case http.StatusRequestTimeout, http.StatusGatewayTimeout:
			return ProviderFailure(ProviderTimeout)
		case http.StatusPaymentRequired:
			return ProviderFailure(ProviderAmbiguousPaid)
		case http.StatusNotFound, http.StatusMethodNotAllowed, http.StatusUnsupportedMediaType, http.StatusUnprocessableEntity:
			return ProviderFailure(ProviderIncompatible)
		default:
			return ProviderFailure(ProviderUnreachable)
		}
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
