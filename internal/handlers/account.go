package handlers

import (
	"errors"
	"net/http"

	"github.com/toulibre/libreregistration/internal/i18n"
	"github.com/toulibre/libreregistration/internal/middleware"
	"github.com/toulibre/libreregistration/internal/services"
	"github.com/toulibre/libreregistration/templates/public"
)

type AccountHandler struct {
	auth          *services.AuthService
	registrations *services.RegistrationService
	settings      *services.SettingsService
	uploadDir     string
}

func NewAccountHandler(auth *services.AuthService, registrations *services.RegistrationService, settings *services.SettingsService, uploadDir string) *AccountHandler {
	return &AccountHandler{auth: auth, registrations: registrations, settings: settings, uploadDir: uploadDir}
}

func (h *AccountHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	regs, _ := h.registrations.ListByUser(userID)

	siteName, accentColor := h.settings.GetSiteSettings()
	public.AccountDashboard(siteName, accentColor, middleware.GetDisplayName(r), regs).Render(r.Context(), w)
}

func (h *AccountHandler) ProfileForm(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	user, err := h.auth.GetUser(userID)
	if err != nil || user == nil {
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	siteName, accentColor := h.settings.GetSiteSettings()
	csrfField := middleware.CSRFTemplateField(r)
	flashes := middleware.GetFlashes(w, r, "success")
	flash := ""
	if len(flashes) > 0 {
		flash = flashes[0]
	}
	public.AccountProfile(user, siteName, accentColor, csrfField, "", flash).Render(r.Context(), w)
}

func (h *AccountHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		r.ParseForm()
	}

	userID := middleware.GetUserID(r)
	user, err := h.auth.GetUser(userID)
	if err != nil || user == nil {
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	name := r.FormValue("name")
	email := r.FormValue("email")

	// Handle avatar
	avatarPath := user.AvatarPath
	avatarFile, err := saveUpload(r, "avatar", h.uploadDir)
	if err != nil {
		siteName, accentColor := h.settings.GetSiteSettings()
		csrfField := middleware.CSRFTemplateField(r)
		public.AccountProfile(user, siteName, accentColor, csrfField, i18n.T(r.Context(), "error.upload_invalid_type"), "").Render(r.Context(), w)
		return
	}
	switch {
	case avatarFile != "":
		deleteUpload(h.uploadDir, user.AvatarPath)
		avatarPath = avatarFile
	case r.FormValue("remove_avatar") == "true":
		deleteUpload(h.uploadDir, user.AvatarPath)
		avatarPath = ""
	}

	if err := h.auth.UpdateProfile(userID, name, email, avatarPath); err != nil {
		siteName, accentColor := h.settings.GetSiteSettings()
		csrfField := middleware.CSRFTemplateField(r)
		public.AccountProfile(user, siteName, accentColor, csrfField, i18n.T(r.Context(), "error.update_failed"), "").Render(r.Context(), w)
		return
	}

	// Update session display name
	session := middleware.GetSession(r)
	if name != "" {
		session.Values["display_name"] = name
	} else {
		session.Values["display_name"] = user.Username
	}
	session.Save(r, w)

	middleware.SetFlash(w, r, "success", i18n.T(r.Context(), "flash.profile_updated"))
	http.Redirect(w, r, "/account/profile", http.StatusFound)
}

func (h *AccountHandler) PasswordForm(w http.ResponseWriter, r *http.Request) {
	siteName, accentColor := h.settings.GetSiteSettings()
	csrfField := middleware.CSRFTemplateField(r)
	flashes := middleware.GetFlashes(w, r, "success")
	flash := ""
	if len(flashes) > 0 {
		flash = flashes[0]
	}
	public.AccountPassword(siteName, accentColor, middleware.GetDisplayName(r), csrfField, "", flash).Render(r.Context(), w)
}

func (h *AccountHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	if currentPassword == "" || newPassword == "" {
		siteName, accentColor := h.settings.GetSiteSettings()
		csrfField := middleware.CSRFTemplateField(r)
		public.AccountPassword(siteName, accentColor, middleware.GetDisplayName(r), csrfField, i18n.T(r.Context(), "password.error.fields_required"), "").Render(r.Context(), w)
		return
	}

	if newPassword != confirmPassword {
		siteName, accentColor := h.settings.GetSiteSettings()
		csrfField := middleware.CSRFTemplateField(r)
		public.AccountPassword(siteName, accentColor, middleware.GetDisplayName(r), csrfField, i18n.T(r.Context(), "password.error.mismatch"), "").Render(r.Context(), w)
		return
	}

	userID := middleware.GetUserID(r)
	err := h.auth.ChangePassword(userID, currentPassword, newPassword)
	if err != nil {
		errorKey := "error.internal"
		if errors.Is(err, services.ErrInvalidCurrentPassword) {
			errorKey = "password.error.current_invalid"
		}
		siteName, accentColor := h.settings.GetSiteSettings()
		csrfField := middleware.CSRFTemplateField(r)
		public.AccountPassword(siteName, accentColor, middleware.GetDisplayName(r), csrfField, i18n.T(r.Context(), errorKey), "").Render(r.Context(), w)
		return
	}

	middleware.SetFlash(w, r, "success", i18n.T(r.Context(), "flash.password_changed"))
	http.Redirect(w, r, "/account/password", http.StatusFound)
}

func (h *AccountHandler) DeleteForm(w http.ResponseWriter, r *http.Request) {
	siteName, accentColor := h.settings.GetSiteSettings()
	csrfField := middleware.CSRFTemplateField(r)
	public.AccountDelete(siteName, accentColor, middleware.GetDisplayName(r), csrfField).Render(r.Context(), w)
}

func (h *AccountHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	// Get user to clean up avatar
	user, _ := h.auth.GetUser(userID)
	if user != nil && user.AvatarPath != "" {
		deleteUpload(h.uploadDir, user.AvatarPath)
	}

	// Anonymize registrations (keep attendance records)
	h.registrations.AnonymizeByUser(userID)

	// Delete user account
	h.auth.DeleteUser(userID)

	// Clear session
	session := middleware.GetSession(r)
	session.Values = make(map[interface{}]interface{})
	session.Options.MaxAge = -1
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusFound)
}
