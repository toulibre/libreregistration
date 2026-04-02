package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/toulibre/libreregistration/internal/captcha"
	"github.com/toulibre/libreregistration/internal/config"
	"github.com/toulibre/libreregistration/internal/i18n"
	"github.com/toulibre/libreregistration/internal/mail"
	"github.com/toulibre/libreregistration/internal/middleware"
	"github.com/toulibre/libreregistration/internal/models"
	"github.com/toulibre/libreregistration/internal/services"
	"github.com/toulibre/libreregistration/templates/admin"
	"github.com/toulibre/libreregistration/templates/public"
)

type AuthHandler struct {
	auth     *services.AuthService
	settings *services.SettingsService
	cfg      *config.Config
}

func NewAuthHandler(auth *services.AuthService, settings *services.SettingsService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{auth: auth, settings: settings, cfg: cfg}
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
	challenge := captcha.Generate(w, r)
	public.RegisterForm(siteName, accentColor, csrfField, "", &models.User{}, challenge.Question).Render(r.Context(), w)
}

func (h *AuthHandler) renderRegisterError(w http.ResponseWriter, r *http.Request, u *models.User, errorKey string) {
	siteName, accentColor := h.settings.GetSiteSettings()
	csrfField := middleware.CSRFTemplateField(r)
	challenge := captcha.Generate(w, r)
	public.RegisterForm(siteName, accentColor, csrfField, i18n.T(r.Context(), errorKey), u, challenge.Question).Render(r.Context(), w)
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
	u := &models.User{Username: username, Name: name, Email: email}

	// Honeypot check
	if captcha.IsHoneypotFilled(r) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Captcha check
	if !captcha.Verify(w, r) {
		h.renderRegisterError(w, r, u, "error.captcha_invalid")
		return
	}

	if username == "" || password == "" {
		h.renderRegisterError(w, r, u, "error.login_password_required")
		return
	}

	user, err := h.auth.Register(username, name, email, password)
	if err != nil {
		errorKey := "error.creation_failed"
		if errors.Is(err, services.ErrUsernameTaken) {
			errorKey = "error.username_taken"
		}
		h.renderRegisterError(w, r, u, errorKey)
		return
	}

	// Send verification email if SMTP configured and user has email
	if user.Email != "" && user.EmailVerifyToken != "" && h.cfg.SMTPHost != "" {
		verifyURL := fmt.Sprintf("%s/verify-email/%s", h.cfg.BaseURL, user.EmailVerifyToken)
		go mail.SendEmailVerification(h.cfg, r.Context(), user.Email, verifyURL)
	}

	h.setSession(w, r, user)
	http.Redirect(w, r, "/account", http.StatusFound)
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	user, err := h.auth.VerifyEmail(token)
	if err != nil || user == nil {
		http.NotFound(w, r)
		return
	}

	middleware.SetFlash(w, r, "success", i18n.T(r.Context(), "flash.email_verified"))
	http.Redirect(w, r, "/account", http.StatusFound)
}

func (h *AuthHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	user, err := h.auth.GetUser(userID)
	if err != nil || user == nil || user.EmailVerified || user.Email == "" {
		http.Redirect(w, r, "/account", http.StatusFound)
		return
	}

	token, err := h.auth.GenerateVerifyToken(userID)
	if err != nil {
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	if h.cfg.SMTPHost != "" {
		verifyURL := fmt.Sprintf("%s/verify-email/%s", h.cfg.BaseURL, token)
		go mail.SendEmailVerification(h.cfg, r.Context(), user.Email, verifyURL)
	}

	middleware.SetFlash(w, r, "success", i18n.T(r.Context(), "flash.verification_sent"))
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
