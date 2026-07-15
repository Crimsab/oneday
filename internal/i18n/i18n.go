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
