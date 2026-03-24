package handlers

import (
	"errors"
	"net/http"

	"github.com/toulibre/libreregistration/internal/i18n"
	"github.com/toulibre/libreregistration/internal/middleware"
	"github.com/toulibre/libreregistration/internal/models"
	"github.com/toulibre/libreregistration/internal/services"
	"github.com/toulibre/libreregistration/templates/admin"
	"github.com/toulibre/libreregistration/templates/public"
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
	allowReg := h.settings.AllowSelfRegistration()
	admin.Login(siteName, accentColor, csrfField, "", allowReg).Render(r.Context(), w)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := h.auth.Authenticate(username, password)
	if err != nil || user == nil {
		siteName, accentColor := h.settings.GetSiteSettings()
		csrfField := middleware.CSRFTemplateField(r)
		allowReg := h.settings.AllowSelfRegistration()
		w.WriteHeader(http.StatusUnauthorized)
		admin.Login(siteName, accentColor, csrfField, i18n.T(r.Context(), "error.invalid_credentials"), allowReg).Render(r.Context(), w)
		return
	}

	h.setSession(w, r, user)

	if user.Role == models.RoleUser {
		http.Redirect(w, r, "/account", http.StatusFound)
	} else {
		http.Redirect(w, r, "/admin/", http.StatusFound)
	}
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	session := middleware.GetSession(r)
	session.Values = make(map[interface{}]interface{})
	session.Options.MaxAge = -1
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *AuthHandler) RegisterForm(w http.ResponseWriter, r *http.Request) {
	if !h.settings.AllowSelfRegistration() {
		http.Redirect(w, r, "/admin/login", http.StatusFound)
		return
	}

	siteName, accentColor := h.settings.GetSiteSettings()
	csrfField := middleware.CSRFTemplateField(r)
	public.RegisterForm(siteName, accentColor, csrfField, "", &models.User{}).Render(r.Context(), w)
}

func (h *AuthHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	if !h.settings.AllowSelfRegistration() {
		http.Redirect(w, r, "/admin/login", http.StatusFound)
		return
	}

	username := r.FormValue("username")
	name := r.FormValue("name")
	email := r.FormValue("email")
	password := r.FormValue("password")

	if username == "" || password == "" {
		siteName, accentColor := h.settings.GetSiteSettings()
		csrfField := middleware.CSRFTemplateField(r)
		u := &models.User{Username: username, Name: name, Email: email}
		public.RegisterForm(siteName, accentColor, csrfField, i18n.T(r.Context(), "error.login_password_required"), u).Render(r.Context(), w)
		return
	}

	user, err := h.auth.Register(username, name, email, password)
	if err != nil {
		errorKey := "error.creation_failed"
		if errors.Is(err, services.ErrUsernameTaken) {
			errorKey = "error.username_taken"
		}
		siteName, accentColor := h.settings.GetSiteSettings()
		csrfField := middleware.CSRFTemplateField(r)
		u := &models.User{Username: username, Name: name, Email: email}
		public.RegisterForm(siteName, accentColor, csrfField, i18n.T(r.Context(), errorKey), u).Render(r.Context(), w)
		return
	}

	h.setSession(w, r, user)
	http.Redirect(w, r, "/account", http.StatusFound)
}

func (h *AuthHandler) setSession(w http.ResponseWriter, r *http.Request, user *models.User) {
	session := middleware.GetSession(r)
	session.Values["user_id"] = user.ID
	session.Values["username"] = user.Username
	session.Values["display_name"] = user.DisplayName()
	session.Values["role"] = string(user.Role)
	session.Save(r, w)
}
