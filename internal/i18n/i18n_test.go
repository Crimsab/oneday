package i18n

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestCatalogParity(t *testing.T) {
	verbs := regexp.MustCompile(`%[-+0-9.#]*[a-zA-Z]`)
	for key := range englishCatalog {
		if _, ok := italianCatalog[key]; !ok {
			t.Errorf("Italian catalog missing %q", key)
			continue
		}
		en, it := englishCatalog[key], italianCatalog[key]
		for _, pair := range [][2]string{{en.Text, it.Text}, {en.One, it.One}, {en.Other, it.Other}} {
			if strings.Join(verbs.FindAllString(pair[0], -1), ",") != strings.Join(verbs.FindAllString(pair[1], -1), ",") {
				t.Errorf("interpolation mismatch for %q: %q vs %q", key, pair[0], pair[1])
			}
		}
	}
	for key := range italianCatalog {
		if _, ok := englishCatalog[key]; !ok {
			t.Errorf("English catalog missing %q", key)
		}
	}
}

func TestNormalize(t *testing.T) {
	for raw, want := range map[string]Locale{"en": English, "en-US": English, "en_GB": English, "it": Italian, "it-IT": Italian, "it_CH": Italian, "fr": "", "auto": ""} {
		if got := Normalize(raw); got != want {
			t.Errorf("Normalize(%q)=%q want %q", raw, got, want)
		}
	}
}

func TestResolvePrecedence(t *testing.T) {
	env := map[string]string{"LC_ALL": "it_IT.UTF-8", "LC_MESSAGES": "en_US", "LANG": "en_GB.UTF-8"}
	if got := Resolve("en-US", env); got != English {
		t.Fatalf("saved preference got %q", got)
	}
	if got := Resolve("", env); got != Italian {
		t.Fatalf("LC_ALL got %q", got)
	}
	delete(env, "LC_ALL")
	if got := Resolve("", env); got != English {
		t.Fatalf("LC_MESSAGES got %q", got)
	}
	if got := Resolve("", map[string]string{"LANG": "fr_FR"}); got != English {
		t.Fatalf("fallback got %q", got)
	}
}

func TestFallbackPluralAndFormatting(t *testing.T) {
	it := New(Italian)
	if got := it.T("missing.raw.key"); strings.Contains(got, "missing.raw.key") || got == "" {
		t.Fatalf("unsafe fallback %q", got)
	}
	if got := it.Plural("time.minutes", 1); got != "1 minuto fa" {
		t.Fatalf("singular %q", got)
	}
	if got := it.Plural("time.minutes", 3); got != "3 minuti fa" {
		t.Fatalf("plural %q", got)
	}
	when := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	if got := it.Date(when); got != "15/07/2026" {
		t.Fatalf("date %q", got)
	}
	if got := it.Decimal(0.75); got != "0,75" {
		t.Fatalf("decimal %q", got)
	}
}

func TestItalianPublicCLIResiduals(t *testing.T) {
	it := New(Italian)
	checks := map[string]string{
		"cli.rag_reindex_usage": "Uso: oneday rag reindex",
		"cli.story_packs_none":  "Nessun pacchetto storia",
		"cli.doctor_title":      "Diagnostica OneDay",
		"cli.model_required":    "configurati dall'utente",
		"cli.help.usage":        "Uso: oneday",
	}
	for key, want := range checks {
		got := it.T(key)
		if key == "cli.model_required" {
			got = it.T(key, "modello")
		}
		if !strings.Contains(got, want) {
			t.Errorf("%s=%q, want fragment %q", key, got, want)
		}
	}
}
