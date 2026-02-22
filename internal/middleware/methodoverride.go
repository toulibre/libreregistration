package middleware

import (
	"net/http"
	"strings"
)

// MethodOverride checks for a _method form field to override POST requests
// with PUT or DELETE, since HTML forms only support GET/POST.
func MethodOverride(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			override := r.FormValue("_method")
			if override != "" {
				method := strings.ToUpper(override)
				if method == http.MethodPut || method == http.MethodDelete {
					r.Method = method
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}
