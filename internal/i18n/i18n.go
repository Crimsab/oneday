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

// DoctorRecoveryAction is deliberately keyed by the stable recovery action so
// new human wording cannot change the machine-readable readiness contract.
func (l Localizer) DoctorRecoveryAction(action string) string {
	return doctorRecoveryActions[l.locale][action]
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
		"NARRATIVE_NOT_CONFIGURED": "no narrative provider is enabled", "NARRATIVE_MISSING_CREDENTIAL": "narrative provider needs its configured credential", "NARRATIVE_UNREACHABLE": "narrative provider cannot be reached", "NARRATIVE_TIMEOUT": "narrative provider timed out", "NARRATIVE_INCOMPATIBLE": "narrative provider capability or schema is incompatible", "NARRATIVE_AMBIGUOUS_PAID_OUTCOME": "paid narrative request outcome is unknown", "NARRATIVE_READY": "narrative provider is ready",
		"EMBEDDINGS_DISABLED": "RAG embeddings are disabled", "EMBEDDINGS_NOT_CONFIGURED": "no usable embedding provider is configured", "EMBEDDINGS_MISSING_CREDENTIAL": "embedding provider needs its configured credential", "EMBEDDINGS_UNREACHABLE": "embedding provider cannot be reached", "EMBEDDINGS_TIMEOUT": "embedding provider timed out", "EMBEDDINGS_INCOMPATIBLE": "embedding provider capability or schema is incompatible", "EMBEDDINGS_AMBIGUOUS_PAID_OUTCOME": "paid embedding request outcome is unknown", "EMBEDDINGS_DIMENSION_MISMATCH": "embedding dimensions differ from configuration", "EMBEDDINGS_READY": "embedding provider is ready",
		"IMAGE_DISABLED": "image generation is disabled", "IMAGE_NOT_CONFIGURED": "image generation has no endpoint configured", "IMAGE_MISSING_CREDENTIAL": "image provider needs its configured credential", "IMAGE_UNREACHABLE": "image service cannot be reached", "IMAGE_TIMEOUT": "image service timed out", "IMAGE_INCOMPATIBLE": "image provider capability or schema is incompatible", "IMAGE_AMBIGUOUS_PAID_OUTCOME": "paid image request outcome is unknown", "IMAGE_READY": "image generation is configured",
		"TTS_DISABLED": "text-to-speech is disabled", "TTS_NOT_CONFIGURED": "text-to-speech requires its configured endpoint and credential", "TTS_MISSING_CREDENTIAL": "speech provider needs its configured credential", "TTS_UNREACHABLE": "local speech service cannot be reached", "TTS_TIMEOUT": "local speech service timed out", "TTS_INCOMPATIBLE": "speech provider capability or schema is incompatible", "TTS_AMBIGUOUS_PAID_OUTCOME": "paid speech request outcome is unknown", "TTS_READY": "text-to-speech is configured",
		"GATEWAY_NOT_CONFIGURED": "gateway readiness is not configured", "GATEWAY_UNREACHABLE": "gateway cannot be reached", "GATEWAY_TIMEOUT": "gateway timed out", "GATEWAY_READY": "gateway is ready",
		"STORAGE_NOT_DIRECTORY": "data directory path is not a directory", "STORAGE_READY": "data directory is available", "STORAGE_UNAVAILABLE": "data directory cannot be inspected", "STORAGE_PARENT_UNAVAILABLE": "data directory parent is unavailable", "STORAGE_INITIALIZABLE": "data directory will be created on first start",
		"BACKUP_NO_DATABASE": "no database exists yet to back up", "BACKUP_UNAVAILABLE": "database cannot be inspected for backup readiness", "BACKUP_NOT_FILE": "database path is not a regular file", "BACKUP_READY": "database is available for a SQLite-safe backup",
	},
	Italian: {
		"NARRATIVE_NOT_CONFIGURED": "nessun provider narrativo è attivo", "NARRATIVE_MISSING_CREDENTIAL": "il provider narrativo richiede la credenziale configurata", "NARRATIVE_UNREACHABLE": "il provider narrativo non è raggiungibile", "NARRATIVE_TIMEOUT": "il provider narrativo ha superato il tempo massimo", "NARRATIVE_INCOMPATIBLE": "capacità o schema del provider narrativo non sono compatibili", "NARRATIVE_AMBIGUOUS_PAID_OUTCOME": "l’esito della richiesta narrativa a pagamento non è noto", "NARRATIVE_READY": "il provider narrativo è pronto",
		"EMBEDDINGS_DISABLED": "gli embedding RAG sono disabilitati", "EMBEDDINGS_NOT_CONFIGURED": "nessun provider di embedding utilizzabile è configurato", "EMBEDDINGS_MISSING_CREDENTIAL": "il provider di embedding richiede la credenziale configurata", "EMBEDDINGS_UNREACHABLE": "il provider di embedding non è raggiungibile", "EMBEDDINGS_TIMEOUT": "il provider di embedding ha superato il tempo massimo", "EMBEDDINGS_INCOMPATIBLE": "capacità o schema del provider di embedding non sono compatibili", "EMBEDDINGS_AMBIGUOUS_PAID_OUTCOME": "l’esito della richiesta di embedding a pagamento non è noto", "EMBEDDINGS_DIMENSION_MISMATCH": "le dimensioni degli embedding differiscono dalla configurazione", "EMBEDDINGS_READY": "il provider di embedding è pronto",
		"IMAGE_DISABLED": "la generazione delle immagini è disabilitata", "IMAGE_NOT_CONFIGURED": "la generazione delle immagini non ha un endpoint configurato", "IMAGE_MISSING_CREDENTIAL": "il provider immagini richiede la credenziale configurata", "IMAGE_UNREACHABLE": "il servizio immagini non è raggiungibile", "IMAGE_TIMEOUT": "il servizio immagini ha superato il tempo massimo", "IMAGE_INCOMPATIBLE": "capacità o schema del provider immagini non sono compatibili", "IMAGE_AMBIGUOUS_PAID_OUTCOME": "l’esito della richiesta immagini a pagamento non è noto", "IMAGE_READY": "la generazione delle immagini è configurata",
		"TTS_DISABLED": "la sintesi vocale è disabilitata", "TTS_NOT_CONFIGURED": "la sintesi vocale richiede endpoint e credenziale configurati", "TTS_MISSING_CREDENTIAL": "il provider vocale richiede la credenziale configurata", "TTS_UNREACHABLE": "il servizio vocale locale non è raggiungibile", "TTS_TIMEOUT": "il servizio vocale locale ha superato il tempo massimo", "TTS_INCOMPATIBLE": "capacità o schema del provider vocale non sono compatibili", "TTS_AMBIGUOUS_PAID_OUTCOME": "l’esito della richiesta vocale a pagamento non è noto", "TTS_READY": "la sintesi vocale è configurata",
		"GATEWAY_NOT_CONFIGURED": "la verifica del gateway non è configurata", "GATEWAY_UNREACHABLE": "il gateway non è raggiungibile", "GATEWAY_TIMEOUT": "il gateway ha superato il tempo massimo", "GATEWAY_READY": "il gateway è pronto",
		"STORAGE_NOT_DIRECTORY": "il percorso della directory dati non è una directory", "STORAGE_READY": "la directory dati è disponibile", "STORAGE_UNAVAILABLE": "la directory dati non è ispezionabile", "STORAGE_PARENT_UNAVAILABLE": "la directory padre dei dati non è disponibile", "STORAGE_INITIALIZABLE": "la directory dati verrà creata al primo avvio",
		"BACKUP_NO_DATABASE": "non esiste ancora un database da salvare", "BACKUP_UNAVAILABLE": "il database non è ispezionabile per verificare il backup", "BACKUP_NOT_FILE": "il percorso del database non è un file regolare", "BACKUP_READY": "il database è disponibile per un backup SQLite sicuro",
	},
}

var doctorRecoveryActions = map[Locale]map[string]string{
	English: {"configure": "Next: run `oneday setup --reconfigure` and save the required setting.", "check_credentials": "Next: verify the configured credential without printing it, then run `oneday doctor` again.", "check_connection": "Next: verify the configured service is reachable, then retry the readiness check.", "retry_later": "Next: wait for the provider request to settle, then retry. Do not assume a paid request failed or succeeded.", "check_capability": "Next: confirm the selected model and provider capability are compatible, then retry.", "review_billing": "Next: review the provider request history before retrying to avoid a duplicate paid request.", "create_backup": "Next: create and verify a SQLite-safe backup before an upgrade.", "restore_empty_target": "Next: restore only into an empty, stopped target; keep the original unchanged."},
	Italian: {"configure": "Passo successivo: usa `oneday setup --reconfigure` e salva l’impostazione richiesta.", "check_credentials": "Passo successivo: verifica la credenziale configurata senza stamparla, quindi usa di nuovo `oneday doctor`.", "check_connection": "Passo successivo: verifica che il servizio configurato sia raggiungibile, quindi riprova il controllo.", "retry_later": "Passo successivo: attendi che la richiesta al provider si stabilizzi, poi riprova. Non presumere che una richiesta a pagamento sia riuscita o fallita.", "check_capability": "Passo successivo: verifica che modello e capacità del provider selezionato siano compatibili, quindi riprova.", "review_billing": "Passo successivo: controlla la cronologia delle richieste del provider prima di riprovare per evitare un doppio addebito.", "create_backup": "Passo successivo: crea e verifica un backup SQLite sicuro prima di un aggiornamento.", "restore_empty_target": "Passo successivo: ripristina solo in una destinazione vuota e arrestata; conserva l’originale invariato."},
}
