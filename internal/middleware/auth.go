package middleware

import (
	"context"
	"net/http"

	"github.com/toulibre/libreregistration/internal/i18n"
)

const UserIDKey contextKey = "user_id"
const UsernameKey contextKey = "username"
const DisplayNameKey contextKey = "display_name"
const UserRoleKey contextKey = "user_role"

// RequireAuth redirects to login if not authenticated.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := GetSession(r)
		if session == nil {
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}

		userID, ok := session.Values["user_id"].(string)
		if !ok || userID == "" {
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, UserIDKey, userID)
		if username, ok := session.Values["username"].(string); ok {
			ctx = context.WithValue(ctx, UsernameKey, username)
		}
		if displayName, ok := session.Values["display_name"].(string); ok {
			ctx = context.WithValue(ctx, DisplayNameKey, displayName)
		}
		if role, ok := session.Values["role"].(string); ok {
			ctx = context.WithValue(ctx, UserRoleKey, role)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin returns 403 if user is not an admin.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value(UserRoleKey).(string)
		if role != "admin" {
			http.Error(w, i18n.T(r.Context(), "error.forbidden"), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func GetUserID(r *http.Request) string {
	id, _ := r.Context().Value(UserIDKey).(string)
	return id
}

func GetUsername(r *http.Request) string {
	name, _ := r.Context().Value(UsernameKey).(string)
	return name
}

func GetDisplayName(r *http.Request) string {
	name, _ := r.Context().Value(DisplayNameKey).(string)
	if name != "" {
		return name
	}
	return GetUsername(r)
}

func GetUserRole(r *http.Request) string {
	role, _ := r.Context().Value(UserRoleKey).(string)
	return role
}
