package i18n

import (
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type Locale string

const (
	English Locale = "en"
	Italian Locale = "it"
)

var matcher = language.NewMatcher([]language.Tag{language.English, language.Italian})

type Message struct {
	Text  string
	One   string
	Other string
}

type Localizer struct {
	locale  Locale
	printer *message.Printer
}

func Normalize(raw string) Locale {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "_", "-"))
	if raw == "" || strings.EqualFold(raw, "auto") || raw == "C" || raw == "POSIX" {
		return ""
	}
	tag, err := language.Parse(raw)
	if err != nil {
		return ""
	}
	matched, index, confidence := matcher.Match(tag)
	if confidence == language.No {
		return ""
	}
	_ = matched
	if index == 1 {
		return Italian
	}
	return English
}

// Resolve uses the saved preference first, then the conventional POSIX locale
// variables, and finally English. Passing an empty env map reads the process.
func Resolve(saved string, env map[string]string) Locale {
	if locale := Normalize(saved); locale != "" {
		return locale
	}
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		value := env[name]
		if env == nil {
			value = os.Getenv(name)
		}
		if locale := Normalize(strings.Split(value, ".")[0]); locale != "" {
			return locale
		}
	}
	return English
}

func New(locale Locale) Localizer {
	locale = Normalize(string(locale))
	if locale == "" {
		locale = English
	}
	tag := language.English
	if locale == Italian {
		tag = language.Italian
	}
	return Localizer{locale: locale, printer: message.NewPrinter(tag)}
}

func (l Localizer) Locale() Locale { return l.locale }

func (l Localizer) T(key string, args ...any) string {
	msg, ok := catalog(l.locale)[key]
	if !ok || msg.Text == "" {
		msg, ok = englishCatalog[key]
	}
	if !ok || msg.Text == "" {
		return englishCatalog["common.unknown"].Text
	}
	return fmt.Sprintf(msg.Text, args...)
}

func (l Localizer) Plural(key string, count int, args ...any) string {
	msg, ok := catalog(l.locale)[key]
	if !ok {
		msg, ok = englishCatalog[key]
	}
	if !ok {
		return englishCatalog["common.unknown"].Text
	}
	text := msg.Other
	if count == 1 {
		text = msg.One
	}
	if text == "" {
		text = msg.Text
	}
	values := append([]any{count}, args...)
	return fmt.Sprintf(text, values...)
}

func (l Localizer) Number(value int) string { return l.printer.Sprintf("%d", value) }

func (l Localizer) Decimal(value float64) string { return l.printer.Sprintf("%.2f", value) }

func (l Localizer) Date(t time.Time) string {
	if l.locale == Italian {
		return fmt.Sprintf("%02d/%02d/%04d", t.Day(), int(t.Month()), t.Year())
	}
	return t.Format("Jan 2, 2006")
}

func (l Localizer) RelativeTime(t, now time.Time) string {
	if t.IsZero() {
		return l.T("time.never")
	}
	d := now.Sub(t)
	if d < time.Minute {
		return l.T("time.now")
	}
	if d < time.Hour {
		return l.Plural("time.minutes", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return l.Plural("time.hours", int(d.Hours()))
	}
	if d < 7*24*time.Hour {
		return l.Plural("time.days", int(d.Hours()/24))
	}
	return l.Date(t)
}

func (l Localizer) CommandPresentation(id, field, fallback string) string {
	if l.locale == Italian {
		if translated := italianCommandCatalog[id+"."+field]; translated != "" {
			return translated
		}
	}
	return fallback
}

func (l Localizer) StoryPresentation(key, fallback string) string {
	if l.locale == Italian && italianStoryCatalog[key] != "" {
		return italianStoryCatalog[key]
	}
	return fallback
}

func (l Localizer) SetupPresentation(key, fallback string) string {
	if l.locale == Italian && italianSetupCatalog[key] != "" {
		return italianSetupCatalog[key]
	}
	return fallback
}

// DoctorProbeSummary translates only human presentation. Probe codes and the
// JSON report remain stable, English, and safe for scripts.
func (l Localizer) DoctorProbeSummary(code, fallback string) string {
	if summary := doctorProbeSummaries[l.locale][code]; summary != "" {
		return summary
	}
	return fallback
}

// DoctorProbeAction is deliberately keyed by the stable probe code so new
// human wording cannot change the machine-readable readiness contract.
func (l Localizer) DoctorProbeAction(code string) string {
	return doctorProbeActions[l.locale][code]
}

func (l Localizer) SocialPresentation(value string) string {
	if l.locale == Italian && italianSocialCatalog[value] != "" {
		return italianSocialCatalog[value]
	}
	return value
}

func catalog(locale Locale) map[string]Message {
	if locale == Italian {
		return italianCatalog
	}
	return englishCatalog
}

var doctorProbeSummaries = map[Locale]map[string]string{
	English: {
		"NARRATIVE_NOT_CONFIGURED": "no narrative provider is enabled", "NARRATIVE_UNAVAILABLE": "enabled narrative provider did not pass its readiness check", "NARRATIVE_READY": "narrative provider is ready",
		"EMBEDDINGS_DISABLED": "RAG embeddings are disabled", "EMBEDDINGS_NOT_CONFIGURED": "no usable embedding provider is configured", "EMBEDDINGS_UNAVAILABLE": "embedding provider did not pass its readiness check", "EMBEDDINGS_DIMENSION_MISMATCH": "embedding dimensions differ from configuration", "EMBEDDINGS_READY": "embedding provider is ready",
		"IMAGE_DISABLED": "image generation is disabled", "IMAGE_NOT_CONFIGURED": "image generation has no endpoint configured", "IMAGE_UNAVAILABLE": "image bridge did not pass its readiness check", "IMAGE_READY": "image generation is configured",
		"TTS_DISABLED": "text-to-speech is disabled", "TTS_NOT_CONFIGURED": "text-to-speech requires its configured endpoint and credential", "TTS_UNAVAILABLE": "local text-to-speech did not pass its readiness check", "TTS_READY": "text-to-speech is configured",
		"GATEWAY_NOT_CONFIGURED": "set ONEDAY_GATEWAY_URL to check a running browser gateway", "GATEWAY_UNAVAILABLE": "gateway did not pass its health check", "GATEWAY_READY": "gateway is ready",
		"STORAGE_NOT_DIRECTORY": "data directory path is not a directory", "STORAGE_READY": "data directory is available", "STORAGE_UNAVAILABLE": "data directory cannot be inspected", "STORAGE_PARENT_UNAVAILABLE": "data directory parent is unavailable", "STORAGE_INITIALIZABLE": "data directory will be created on first start",
		"BACKUP_NO_DATABASE": "no database exists yet to back up", "BACKUP_UNAVAILABLE": "database cannot be inspected for backup readiness", "BACKUP_NOT_FILE": "database path is not a regular file", "BACKUP_READY": "database is available for a SQLite-safe backup",
	},
	Italian: {
		"NARRATIVE_NOT_CONFIGURED": "nessun provider narrativo è attivo", "NARRATIVE_UNAVAILABLE": "il provider narrativo attivo non ha superato il controllo di disponibilità", "NARRATIVE_READY": "il provider narrativo è pronto",
		"EMBEDDINGS_DISABLED": "gli embedding RAG sono disabilitati", "EMBEDDINGS_NOT_CONFIGURED": "nessun provider di embedding utilizzabile è configurato", "EMBEDDINGS_UNAVAILABLE": "il provider di embedding non ha superato il controllo di disponibilità", "EMBEDDINGS_DIMENSION_MISMATCH": "le dimensioni degli embedding differiscono dalla configurazione", "EMBEDDINGS_READY": "il provider di embedding è pronto",
		"IMAGE_DISABLED": "la generazione delle immagini è disabilitata", "IMAGE_NOT_CONFIGURED": "la generazione delle immagini non ha un endpoint configurato", "IMAGE_UNAVAILABLE": "il bridge immagini non ha superato il controllo di disponibilità", "IMAGE_READY": "la generazione delle immagini è configurata",
		"TTS_DISABLED": "la sintesi vocale è disabilitata", "TTS_NOT_CONFIGURED": "la sintesi vocale richiede endpoint e credenziale configurati", "TTS_UNAVAILABLE": "la sintesi vocale locale non ha superato il controllo di disponibilità", "TTS_READY": "la sintesi vocale è configurata",
		"GATEWAY_NOT_CONFIGURED": "imposta ONEDAY_GATEWAY_URL per controllare un gateway browser in esecuzione", "GATEWAY_UNAVAILABLE": "il gateway non ha superato il controllo di integrità", "GATEWAY_READY": "il gateway è pronto",
		"STORAGE_NOT_DIRECTORY": "il percorso della directory dati non è una directory", "STORAGE_READY": "la directory dati è disponibile", "STORAGE_UNAVAILABLE": "la directory dati non è ispezionabile", "STORAGE_PARENT_UNAVAILABLE": "la directory padre dei dati non è disponibile", "STORAGE_INITIALIZABLE": "la directory dati verrà creata al primo avvio",
		"BACKUP_NO_DATABASE": "non esiste ancora un database da salvare", "BACKUP_UNAVAILABLE": "il database non è ispezionabile per verificare il backup", "BACKUP_NOT_FILE": "il percorso del database non è un file regolare", "BACKUP_READY": "il database è disponibile per un backup SQLite sicuro",
	},
}

var doctorProbeActions = map[Locale]map[string]string{
	English: {"NARRATIVE_NOT_CONFIGURED": "Next: run `oneday setup --reconfigure` and configure a narrative provider.", "NARRATIVE_UNAVAILABLE": "Next: check the configured narrative provider, then run `oneday doctor` again.", "IMAGE_NOT_CONFIGURED": "Next: run `oneday setup --reconfigure` to configure images or choose text-only.", "IMAGE_UNAVAILABLE": "Next: check the image service, then run `oneday doctor` again.", "TTS_NOT_CONFIGURED": "Next: run `oneday setup --reconfigure` to configure speech or disable it.", "TTS_UNAVAILABLE": "Next: check the local speech service, then run `oneday doctor` again."},
	Italian: {"NARRATIVE_NOT_CONFIGURED": "Passo successivo: usa `oneday setup --reconfigure` e configura un provider narrativo.", "NARRATIVE_UNAVAILABLE": "Passo successivo: controlla il provider narrativo configurato, quindi usa di nuovo `oneday doctor`.", "IMAGE_NOT_CONFIGURED": "Passo successivo: usa `oneday setup --reconfigure` per configurare le immagini o scegli solo testo.", "IMAGE_UNAVAILABLE": "Passo successivo: controlla il servizio immagini, quindi usa di nuovo `oneday doctor`.", "TTS_NOT_CONFIGURED": "Passo successivo: usa `oneday setup --reconfigure` per configurare la voce o disabilitarla.", "TTS_UNAVAILABLE": "Passo successivo: controlla il servizio vocale locale, quindi usa di nuovo `oneday doctor`."},
}
