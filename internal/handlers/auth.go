package handlers

import (
	"net/http"

	"github.com/toulibre/libreregistration/internal/i18n"
	"github.com/toulibre/libreregistration/internal/middleware"
	"github.com/toulibre/libreregistration/internal/services"
	"github.com/toulibre/libreregistration/templates/admin"
)

type AuthHandler struct {
	auth     *services.AuthService
	settings *services.SettingsService
}

func NewAuthHandler(auth *services.AuthService, settings *services.SettingsService) *AuthHandler {
	return &AuthHandler{auth: auth, settings: settings}
}

func (h *AuthHandler) LoginForm(w http.ResponseWriter, r *http.Request) {
	siteName, accentColor := h.settings.GetSiteSettings()
	csrfField := middleware.CSRFTemplateField(r)
	admin.Login(siteName, accentColor, csrfField, "").Render(r.Context(), w)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := h.auth.Authenticate(username, password)
	if err != nil || user == nil {
		siteName, accentColor := h.settings.GetSiteSettings()
		csrfField := middleware.CSRFTemplateField(r)
		w.WriteHeader(http.StatusUnauthorized)
		admin.Login(siteName, accentColor, csrfField, i18n.T(r.Context(), "error.invalid_credentials")).Render(r.Context(), w)
		return
	}

	session := middleware.GetSession(r)
	session.Values["user_id"] = user.ID
	session.Values["username"] = user.Username
	session.Values["display_name"] = user.DisplayName()
	session.Values["role"] = string(user.Role)
	session.Save(r, w)

	http.Redirect(w, r, "/admin/", http.StatusFound)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	session := middleware.GetSession(r)
	session.Values = make(map[interface{}]interface{})
	session.Options.MaxAge = -1
	session.Save(r, w)

	http.Redirect(w, r, "/admin/login", http.StatusFound)
}
