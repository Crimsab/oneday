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
	report := Run(context.Background(), cfg, "/tmp/config.yaml", deps)
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
	report := Run(context.Background(), cfg, "custom.yaml", deps)
	if report.RequiredFailure() {
		t.Fatalf("optional readiness warning must not fail doctor: %#v", report)
	}
	if report.ConfigPath != "custom.yaml" {
		t.Fatalf("config path = %q", report.ConfigPath)
	}
	for _, probe := range report.Probes {
		if probe.Name == "embeddings" && probe.Code != "EMBEDDINGS_DIMENSION_MISMATCH" {
			t.Fatalf("embedding probe = %#v", probe)
		}
	}
}
