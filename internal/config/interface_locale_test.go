package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInterfaceLocaleBackwardCompatibleAndValidated(t *testing.T) {
	cfg := Default()
	if cfg.Interface.Locale != "" {
		t.Fatalf("default locale %q", cfg.Interface.Locale)
	}
	cfg.Interface.Locale = "IT"
	if err := cfg.Validate(); err != nil || cfg.Interface.Locale != "it" {
		t.Fatalf("normalize locale=%q err=%v", cfg.Interface.Locale, err)
	}
	cfg.Interface.Locale = "fr"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid locale")
	}
}

func TestUpdateInterfaceLocalePreservesUnrelatedYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	raw := "# keep me\nconfig_version: 2\ndata_dir: ./data\ncustom_field: portable\n"
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	if err := UpdateInterfaceLocale(path, "it"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	for _, want := range []string{"# keep me", "custom_field: portable", "locale: it"} {
		if !strings.Contains(text, want) {
			t.Errorf("updated YAML missing %q:\n%s", want, text)
		}
	}
	cfg, err := Load(path)
	if err != nil || cfg.Interface.Locale != "it" {
		t.Fatalf("loaded locale=%q err=%v", cfg.Interface.Locale, err)
	}
}
