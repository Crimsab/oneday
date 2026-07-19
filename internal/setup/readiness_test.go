package setup

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crimsab/oneday/internal/aifactory"
	"github.com/crimsab/oneday/internal/config"
)

func TestWriteFileAtomicIsRestrictiveAndLeavesNoBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.yaml")
	if err := WriteFileAtomic(path, []byte("secret_reference: ${TEST_SECRET}\n")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
	if _, err := os.Stat(path + ".bak"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("backup should not exist: %v", err)
	}
}

func TestReadinessParityAndRequiredExit(t *testing.T) {
	cfg := config.Default()
	cfg.AI.Codex.Enabled = true
	cfg.AI.Codex.Model = "test"
	cfg.AI.Generation.UtilityModel = "test"
	cfg.DataDir = t.TempDir()
	deps := Dependencies{
		Narrative: func(context.Context, config.Config) error { return errors.New("token=secret") },
		Embedding: func(context.Context, aifactory.EmbeddingProviderSpec) (int, error) { return 0, nil },
		HTTPGet:   func(context.Context, string) error { return nil },
		Stat:      os.Stat,
	}
	report := Run(context.Background(), cfg, "ONEDAY_CONFIG", deps)
	if !report.RequiredFailure() {
		t.Fatal("narrative failure must make doctor exit nonzero")
	}
	if len(report.Probes) != 7 {
		t.Fatalf("probes = %d", len(report.Probes))
	}
	for _, probe := range report.Probes {
		if strings.Contains(probe.Summary, "secret") {
			t.Fatalf("probe leaks secret: %#v", probe)
		}
	}
}

func TestReadinessConfiguresOptionalProbesWithoutRequiredFailure(t *testing.T) {
	cfg := config.Default()
	cfg.AI.Codex.Enabled = true
	cfg.AI.Codex.Model = "test"
	cfg.AI.Generation.UtilityModel = "test"
	cfg.DataDir = t.TempDir()
	cfg.RAG.Enabled = true
	cfg.AI.Embedding.Provider = "local"
	cfg.AI.Embedding.Local.Enabled = true
	cfg.AI.Embedding.Local.Model = "embed"
	deps := Dependencies{
		Narrative:  func(context.Context, config.Config) error { return nil },
		Embedding:  func(context.Context, aifactory.EmbeddingProviderSpec) (int, error) { return 7, nil },
		HTTPGet:    func(context.Context, string) error { return errors.New("offline") },
		Stat:       os.Stat,
		GatewayURL: "http://gateway.invalid",
	}
	report := Run(context.Background(), cfg, "ONEDAY_CONFIG", deps)
	if report.RequiredFailure() {
		t.Fatalf("optional readiness warning must not fail doctor: %#v", report)
	}
	if report.ConfigSource != "ONEDAY_CONFIG" {
		t.Fatalf("config source = %q", report.ConfigSource)
	}
	for _, probe := range report.Probes {
		if probe.Name == "embeddings" && probe.Code != "EMBEDDINGS_DIMENSION_MISMATCH" {
			t.Fatalf("embedding probe = %#v", probe)
		}
	}
}

func TestReadinessUsesExplicitDatabasePathForBackupWithoutLeakingIt(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	cfg.DataDir = filepath.Join(root, "configured-data")
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	databasePath := filepath.Join(root, "private", "gateway-state.sqlite")
	if err := os.MkdirAll(filepath.Dir(databasePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(databasePath, []byte("sqlite"), 0o600); err != nil {
		t.Fatal(err)
	}

	report := Run(context.Background(), cfg, "ONEDAY_CONFIG", Dependencies{
		Narrative:    func(context.Context, config.Config) error { return nil },
		Embedding:    func(context.Context, aifactory.EmbeddingProviderSpec) (int, error) { return 0, nil },
		HTTPGet:      func(context.Context, string) error { return nil },
		Stat:         os.Stat,
		DatabasePath: databasePath,
	})

	for _, probe := range report.Probes {
		if strings.Contains(probe.Summary, databasePath) || strings.Contains(probe.Summary, cfg.DataDir) {
			t.Fatalf("readiness leaked a private path: %#v", probe)
		}
		if probe.Name == "storage" && probe.Code != "STORAGE_READY" {
			t.Fatalf("storage must continue using configured data_dir: %#v", probe)
		}
		if probe.Name == "backup" && probe.Code != "BACKUP_READY" {
			t.Fatalf("backup must inspect explicit database path: %#v", probe)
		}
		if probe.Name == "backup" && probe.Action != ActionCreateBackup {
			t.Fatalf("backup recovery action = %q, want %q", probe.Action, ActionCreateBackup)
		}
	}
}

func TestDefaultDependenciesReadsExplicitDatabasePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway-state.sqlite")
	t.Setenv("ONEDAY_DB_PATH", path)
	if got := DefaultDependencies().DatabasePath; got != path {
		t.Fatalf("database path = %q, want %q", got, path)
	}
}

func TestProviderDiagnosticStatesAreStableAndRedacted(t *testing.T) {
	cfg := config.Default()
	cfg.AI.Codex.Enabled = true
	cfg.AI.Codex.Model = "test"
	cfg.AI.Generation.UtilityModel = "test"
	cfg.DataDir = t.TempDir()

	tests := []struct {
		name    string
		failure error
		code    string
		action  string
	}{
		{"unreachable", ProviderFailure(ProviderUnreachable), "NARRATIVE_UNREACHABLE", ActionCheckConnection},
		{"timeout", context.DeadlineExceeded, "NARRATIVE_TIMEOUT", ActionRetryLater},
		{"incompatible", ProviderFailure(ProviderIncompatible), "NARRATIVE_INCOMPATIBLE", ActionCheckCapability},
		{"ambiguous paid outcome", ProviderFailure(ProviderAmbiguousPaid), "NARRATIVE_AMBIGUOUS_PAID_OUTCOME", ActionReviewBilling},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := Run(context.Background(), cfg, "ONEDAY_CONFIG", Dependencies{
				Narrative: func(context.Context, config.Config) error { return tt.failure },
				Embedding: func(context.Context, aifactory.EmbeddingProviderSpec) (int, error) { return 0, nil },
				HTTPGet:   func(context.Context, string) error { return nil },
				Stat:      os.Stat,
			})
			probe := report.Probes[0]
			if probe.Code != tt.code || probe.Action != tt.action || probe.Status != StatusFailed {
				t.Fatalf("probe = %#v", probe)
			}
			if strings.Contains(probe.Summary, "secret") || strings.Contains(probe.Summary, "http") {
				t.Fatalf("probe leaked raw diagnostic data: %#v", probe)
			}
		})
	}
}

func TestNarrativeMissingCredentialHasRecoveryAction(t *testing.T) {
	cfg := config.Default()
	cfg.AI.ProviderPriority = []string{"openrouter"}
	cfg.AI.OpenRouter.Enabled = true
	cfg.AI.OpenRouter.DefaultModel = "test"
	cfg.AI.Generation.UtilityModel = "test"
	cfg.DataDir = t.TempDir()
	report := Run(context.Background(), cfg, "ONEDAY_CONFIG", Dependencies{
		Narrative: func(context.Context, config.Config) error {
			t.Fatal("provider must not be called without its credential")
			return nil
		},
		Embedding: func(context.Context, aifactory.EmbeddingProviderSpec) (int, error) { return 0, nil },
		HTTPGet:   func(context.Context, string) error { return nil },
		Stat:      os.Stat,
	})
	probe := report.Probes[0]
	if probe.Code != "NARRATIVE_MISSING_CREDENTIAL" || probe.Action != ActionCheckCredentials {
		t.Fatalf("probe = %#v", probe)
	}
}
