package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/csrf"
)

func CSRF(key []byte, baseURL string) func(http.Handler) http.Handler {
	secure := strings.HasPrefix(baseURL, "https://")

	opts := []csrf.Option{
		csrf.Secure(secure),
		csrf.Path("/"),
		csrf.FieldName("csrf_token"),
	}

	// Trust the origin from BASE_URL so CSRF works behind a reverse proxy
	if parsed, err := url.Parse(baseURL); err == nil && parsed.Host != "" {
		opts = append(opts, csrf.TrustedOrigins([]string{parsed.Scheme + "://" + parsed.Host}))
	}

	protect := csrf.Protect(key, opts...)

	if !secure {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				protect(next).ServeHTTP(w, csrf.PlaintextHTTPRequest(r))
			})
		}
	}

	return protect
}

func CSRFToken(r *http.Request) string {
	return csrf.Token(r)
}

func CSRFTemplateField(r *http.Request) string {
	return string(csrf.TemplateField(r))
}
