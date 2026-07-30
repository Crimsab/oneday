package dockerbootstrap

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPrepareCreatesPrivatePortableConfigurationAndIsIdempotent(t *testing.T) {
	root := t.TempDir()
	writeTemplate(t, root, configTemplateName, "config_version: 3\n")
	writeTemplate(t, root, envTemplateName, strings.Join([]string{
		"TZ=UTC",
		"ONEDAY_PORT=9988",
		bootstrapTokenKey + "=",
		allowedHostsKey + "=",
		"",
	}, "\n"))

	first, err := Prepare(root)
	if err != nil {
		t.Fatal(err)
	}
	if !first.CreatedConfig || !first.CreatedEnv || !first.GeneratedToken || !first.UpdatedHostRules {
		t.Fatalf("unexpected first result: %+v", first)
	}
	token, err := BootstrapToken(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(token) != 64 {
		t.Fatalf("token length = %d, want 64", len(token))
	}
	content, err := os.ReadFile(filepath.Join(root, envName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), allowedHostsKey+"=localhost:9988,127.0.0.1:9988,oneday-gateway:8788") {
		t.Fatalf("custom port host rules missing:\n%s", content)
	}
	if runtime.GOOS != "windows" {
		for _, name := range []string{envName, configName} {
			info, statErr := os.Stat(filepath.Join(root, name))
			if statErr != nil {
				t.Fatal(statErr)
			}
			if info.Mode().Perm() != privateFileMode {
				t.Fatalf("%s mode = %o, want %o", name, info.Mode().Perm(), privateFileMode)
			}
		}
	}

	second, err := Prepare(root)
	if err != nil {
		t.Fatal(err)
	}
	if second.CreatedConfig || second.CreatedEnv || second.GeneratedToken || second.UpdatedHostRules {
		t.Fatalf("second prepare changed existing configuration: %+v", second)
	}
	secondToken, err := BootstrapToken(root)
	if err != nil {
		t.Fatal(err)
	}
	if secondToken != token {
		t.Fatal("idempotent prepare rotated the bootstrap token")
	}
}

func TestPreparePreservesCustomHostRulesAndExistingConfig(t *testing.T) {
	root := t.TempDir()
	writeTemplate(t, root, configTemplateName, "template: true\n")
	writeTemplate(t, root, envTemplateName, "unused=true\n")
	writeTemplate(t, root, configName, "custom: true\n")
	writeTemplate(t, root, envName, strings.Join([]string{
		"ONEDAY_PORT=8788",
		bootstrapTokenKey + "=existing-token",
		allowedHostsKey + "=story.example.test",
		"",
	}, "\n"))

	result, err := Prepare(root)
	if err != nil {
		t.Fatal(err)
	}
	if result != (Result{}) {
		t.Fatalf("custom installation changed: %+v", result)
	}
	config, err := os.ReadFile(filepath.Join(root, configName))
	if err != nil {
		t.Fatal(err)
	}
	if string(config) != "custom: true\n" {
		t.Fatalf("config overwritten: %q", config)
	}
	token, err := BootstrapToken(root)
	if err != nil {
		t.Fatal(err)
	}
	if token != "existing-token" {
		t.Fatalf("token = %q, want existing-token", token)
	}
}

func TestPrepareRejectsInvalidPortBeforeWritingEnvironment(t *testing.T) {
	root := t.TempDir()
	writeTemplate(t, root, configTemplateName, "config_version: 3\n")
	writeTemplate(t, root, envTemplateName, strings.Join([]string{
		"ONEDAY_PORT=not-a-port",
		bootstrapTokenKey + "=",
		allowedHostsKey + "=",
		"",
	}, "\n"))

	if _, err := Prepare(root); err == nil || !strings.Contains(err.Error(), portKey) {
		t.Fatalf("Prepare error = %v, want invalid port", err)
	}
	content, err := os.ReadFile(filepath.Join(root, envName))
	if err != nil {
		t.Fatal(err)
	}
	if token := strings.TrimSpace(parseEnv(content)[bootstrapTokenKey]); token != "" {
		t.Fatal("invalid prepare wrote a token")
	}
}

func writeTemplate(t *testing.T, root, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
