package layouts

import (
	"context"

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
