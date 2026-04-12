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
	auth          *services.AuthService
	registrations *services.RegistrationService
	settings      *services.SettingsService
	cfg           *config.Config
}

func NewAuthHandler(auth *services.AuthService, registrations *services.RegistrationService, settings *services.SettingsService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{auth: auth, registrations: registrations, settings: settings, cfg: cfg}
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

	// Link anonymous registrations with same email to this account
	if user.Email != "" {
		h.registrations.LinkAnonymousByEmail(user.Email, user.ID)
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

func (h *AuthHandler) ForgotPasswordForm(w http.ResponseWriter, r *http.Request) {
	siteName, accentColor := h.settings.GetSiteSettings()
	csrfField := middleware.CSRFTemplateField(r)
	public.ForgotPasswordForm(siteName, accentColor, csrfField, "", "").Render(r.Context(), w)
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	siteName, accentColor := h.settings.GetSiteSettings()
	csrfField := middleware.CSRFTemplateField(r)

	if email == "" {
		public.ForgotPasswordForm(siteName, accentColor, csrfField, i18n.T(r.Context(), "forgot_password.error.email_required"), "").Render(r.Context(), w)
		return
	}

	token, user, err := h.auth.RequestPasswordReset(email)
	if err != nil {
		public.ForgotPasswordForm(siteName, accentColor, csrfField, i18n.T(r.Context(), "error.internal"), "").Render(r.Context(), w)
		return
	}

	// Send email if user exists and SMTP configured — but always show success to avoid user enumeration
	if user != nil && token != "" && h.cfg.SMTPHost != "" {
		resetURL := fmt.Sprintf("%s/reset-password/%s", h.cfg.BaseURL, token)
		go mail.SendPasswordReset(h.cfg, r.Context(), user.Email, resetURL)
	}

	public.ForgotPasswordForm(siteName, accentColor, csrfField, "", i18n.T(r.Context(), "forgot_password.success")).Render(r.Context(), w)
}

func (h *AuthHandler) ResetPasswordForm(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	siteName, accentColor := h.settings.GetSiteSettings()
	csrfField := middleware.CSRFTemplateField(r)
	public.ResetPasswordForm(siteName, accentColor, csrfField, token, "").Render(r.Context(), w)
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	siteName, accentColor := h.settings.GetSiteSettings()
	csrfField := middleware.CSRFTemplateField(r)

	if password == "" {
		public.ResetPasswordForm(siteName, accentColor, csrfField, token, i18n.T(r.Context(), "password.error.fields_required")).Render(r.Context(), w)
		return
	}

	if password != confirmPassword {
		public.ResetPasswordForm(siteName, accentColor, csrfField, token, i18n.T(r.Context(), "password.error.mismatch")).Render(r.Context(), w)
		return
	}

	user, err := h.auth.ResetPassword(token, password)
	if err != nil {
		errorKey := "reset_password.error.invalid_token"
		if errors.Is(err, services.ErrResetTokenExpired) {
			errorKey = "reset_password.error.expired"
		}
		public.ResetPasswordForm(siteName, accentColor, csrfField, token, i18n.T(r.Context(), errorKey)).Render(r.Context(), w)
		return
	}

	h.setSession(w, r, user)
	middleware.SetFlash(w, r, "success", i18n.T(r.Context(), "flash.password_reset"))
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
