package middleware

import (
	"net/http"
	"strings"

	"github.com/toulibre/libreregistration/internal/i18n"
)

const langCookie = "lang"

// Locale detects the user's preferred language and stores it in the context.
// Priority: ?lang= query param > "lang" cookie > Accept-Language header > "fr".
func Locale(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := ""

		// 1. Query param (also sets cookie)
		if q := r.URL.Query().Get("lang"); isSupported(q) {
			lang = q
			http.SetCookie(w, &http.Cookie{
				Name:     langCookie,
				Value:    lang,
				Path:     "/",
				MaxAge:   86400 * 365,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
			// Redirect to strip the ?lang= param to avoid caching issues
			u := *r.URL
			q := u.Query()
			q.Del("lang")
			u.RawQuery = q.Encode()
			http.Redirect(w, r, u.String(), http.StatusFound)
			return
		}

		// 2. Cookie
		if lang == "" {
			if c, err := r.Cookie(langCookie); err == nil && isSupported(c.Value) {
				lang = c.Value
			}
		}

		// 3. Accept-Language header
		if lang == "" {
			lang = parseAcceptLanguage(r.Header.Get("Accept-Language"))
		}

		// 4. Default
		if lang == "" {
			lang = "fr"
		}

		ctx := i18n.WithLocale(r.Context(), lang)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

var supported = map[string]bool{"fr": true, "en": true}

func isSupported(lang string) bool {
	return supported[lang]
}

func parseAcceptLanguage(header string) string {
	if header == "" {
		return ""
	}
	// Simple parser: split by comma, check each tag
	for _, part := range strings.Split(header, ",") {
		tag := strings.TrimSpace(part)
		// Strip quality value (e.g. "en-US;q=0.9" -> "en-US")
		if i := strings.Index(tag, ";"); i != -1 {
			tag = tag[:i]
		}
		tag = strings.TrimSpace(tag)
		// Try exact match
		if isSupported(tag) {
			return tag
		}
		// Try prefix (e.g. "en-US" -> "en")
		if i := strings.Index(tag, "-"); i != -1 {
			prefix := tag[:i]
			if isSupported(prefix) {
				return prefix
			}
		}
	}
	return ""
}
