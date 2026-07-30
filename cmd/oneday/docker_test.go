package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDockerCommandInitializesAndRetrievesToken(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "config.example.yaml"), []byte("config_version: 3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := "ONEDAY_PORT=8788\nONEDAY_GATEWAY_BOOTSTRAP_TOKEN=\nONEDAY_GATEWAY_ALLOWED_HOSTS=\n"
	if err := os.WriteFile(filepath.Join(root, ".env.example"), []byte(env), 0o600); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if err := runDockerCommand([]string{"init", "--root", root}, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "docker compose up -d") {
		t.Fatalf("missing next step:\n%s", output.String())
	}
	output.Reset()
	if err := runDockerCommand([]string{"token", "--root", root}, &output); err != nil {
		t.Fatal(err)
	}
	if len(strings.TrimSpace(output.String())) != 64 {
		t.Fatalf("unexpected token output length: %d", len(strings.TrimSpace(output.String())))
	}
}

func TestRunDockerCommandRejectsUnknownAction(t *testing.T) {
	err := runDockerCommand([]string{"launch"}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "init or token") {
		t.Fatalf("error = %v", err)
	}
}
