package i18n

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

//go:embed locales/*.json
var localeFS embed.FS

type contextKey string

const localeKey contextKey = "locale"

var translations map[string]map[string]string

func init() {
	translations = make(map[string]map[string]string)
	for _, lang := range []string{"fr", "en"} {
		data, err := localeFS.ReadFile("locales/" + lang + ".json")
		if err != nil {
			panic("i18n: missing " + lang + ".json: " + err.Error())
		}
		m := make(map[string]string)
		if err := json.Unmarshal(data, &m); err != nil {
			panic("i18n: invalid " + lang + ".json: " + err.Error())
		}
		translations[lang] = m
	}
}

// WithLocale stores the locale in the context.
func WithLocale(ctx context.Context, locale string) context.Context {
	return context.WithValue(ctx, localeKey, locale)
}

// Locale returns the locale from the context, defaulting to "fr".
func Locale(ctx context.Context) string {
	if l, ok := ctx.Value(localeKey).(string); ok && l != "" {
		return l
	}
	return "fr"
}

// T returns the translation for the given key in the context's locale.
func T(ctx context.Context, key string) string {
	lang := Locale(ctx)
	if m, ok := translations[lang]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	// Fallback to French
	if m, ok := translations["fr"]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	return key
}

// Tf returns a formatted translation (fmt.Sprintf style).
func Tf(ctx context.Context, key string, args ...any) string {
	return fmt.Sprintf(T(ctx, key), args...)
}

// Tn returns a singular or plural translation based on count.
// It looks up key+".one" and key+".other".
func Tn(ctx context.Context, key string, count int) string {
	if count <= 1 {
		return Tf(ctx, key+".one", count)
	}
	return Tf(ctx, key+".other", count)
}

// FormatDate formats a time as a localized date string.
func FormatDate(ctx context.Context, t time.Time) string {
	lang := Locale(ctx)
	switch lang {
	case "en":
		return t.Format("01/02/2006")
	default:
		return t.Format("02/01/2006")
	}
}

// FormatDateTime formats a time as a localized date+time string.
func FormatDateTime(ctx context.Context, t time.Time) string {
	lang := Locale(ctx)
	switch lang {
	case "en":
		return t.Format("01/02/2006 3:04 PM")
	default:
		return t.Format("02/01/2006 à 15h04")
	}
}

// FormatDateTimeCSV formats a time for CSV export.
func FormatDateTimeCSV(ctx context.Context, t time.Time) string {
	lang := Locale(ctx)
	switch lang {
	case "en":
		return t.Format("01/02/2006 15:04")
	default:
		return t.Format("02/01/2006 15:04")
	}
}

// HTMLLang returns the BCP47 language tag for the HTML lang attribute.
func HTMLLang(ctx context.Context) string {
	return Locale(ctx)
}

// LangSwitchURL returns a URL with ?lang= query param for switching language.
func LangSwitchURL(currentPath, targetLang string) string {
	if i := strings.Index(currentPath, "?"); i != -1 {
		currentPath = currentPath[:i]
	}
	return currentPath + "?lang=" + targetLang
}
