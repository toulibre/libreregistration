package layouts

import (
	"context"
	"strings"

	"github.com/toulibre/libreregistration/internal/middleware"
)

func IsLoggedIn(ctx context.Context) bool {
	id, _ := ctx.Value(middleware.UserIDKey).(string)
	return id != ""
}

func IsUserRole(ctx context.Context) bool {
	role, _ := ctx.Value(middleware.UserRoleKey).(string)
	return role == "user"
}

func CtxDisplayName(ctx context.Context) string {
	name, _ := ctx.Value(middleware.DisplayNameKey).(string)
	return name
}

func AllowSelfRegistration(ctx context.Context) bool {
	v, _ := ctx.Value(middleware.AllowSelfRegistrationKey).(bool)
	return v
}

func NavIsActive(ctx context.Context, prefix string) bool {
	path, _ := ctx.Value(middleware.RequestPathKey).(string)
	if prefix == "/admin/" {
		return path == "/admin/" || path == "/admin"
	}
	return strings.HasPrefix(path, prefix)
}
