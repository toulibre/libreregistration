package middleware

import (
	"context"
	"net/http"
)

const AllowSelfRegistrationKey contextKey = "allow_self_registration"

type SelfRegistrationChecker func() bool

func InjectSelfRegistration(check SelfRegistrationChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), AllowSelfRegistrationKey, check())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
