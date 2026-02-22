package middleware

import (
	"net/http"

	"github.com/gorilla/csrf"
)

func CSRF(key []byte, secure bool) func(http.Handler) http.Handler {
	protect := csrf.Protect(
		key,
		csrf.Secure(secure),
		csrf.Path("/"),
		csrf.FieldName("csrf_token"),
	)

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
