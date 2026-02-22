package middleware

import (
	"context"
	"net/http"

	"github.com/gorilla/sessions"
)

type contextKey string

const SessionKey contextKey = "session"
const SessionName = "libreregistration"

func Session(store sessions.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, _ := store.Get(r, SessionName)
			ctx := context.WithValue(r.Context(), SessionKey, session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetSession(r *http.Request) *sessions.Session {
	if s, ok := r.Context().Value(SessionKey).(*sessions.Session); ok {
		return s
	}
	return nil
}
