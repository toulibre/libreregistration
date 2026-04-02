package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/toulibre/libreregistration/internal/captcha"
	"github.com/toulibre/libreregistration/internal/i18n"
	"github.com/toulibre/libreregistration/internal/middleware"
	"github.com/toulibre/libreregistration/internal/models"
	"github.com/toulibre/libreregistration/internal/services"
	"github.com/toulibre/libreregistration/templates/public"
)

type RegistrationHandler struct {
	registrations *services.RegistrationService
	events        *services.EventService
	auth          *services.AuthService
	settings      *services.SettingsService
}

func NewRegistrationHandler(registrations *services.RegistrationService, events *services.EventService, auth *services.AuthService, settings *services.SettingsService) *RegistrationHandler {
	return &RegistrationHandler{registrations: registrations, events: events, auth: auth, settings: settings}
}

func (h *RegistrationHandler) Register(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	event, err := h.events.GetBySlug(slug)
	if err != nil || event == nil {
		http.NotFound(w, r)
		return
	}

	comment := strings.TrimSpace(r.FormValue("comment"))

	var name, email string
	var userID *string
	loggedIn := middleware.GetUserID(r) != ""

	if loggedIn {
		// Logged-in user: use profile data, skip anti-spam
		uid := middleware.GetUserID(r)
		userID = &uid
		user, err := h.auth.GetUser(uid)
		if err != nil || user == nil {
			http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
			return
		}
		name = user.DisplayName()
		email = user.Email
	} else {
		// Anonymous: validate name + anti-spam
		name = strings.TrimSpace(r.FormValue("name"))
		email = strings.TrimSpace(r.FormValue("email"))

		if name == "" {
			h.renderError(w, r, event, i18n.T(r.Context(), "error.name_required"))
			return
		}

		if captcha.IsHoneypotFilled(r) {
			http.Redirect(w, r, "/event/"+slug, http.StatusFound)
			return
		}

		if !captcha.Verify(w, r) {
			h.renderError(w, r, event, i18n.T(r.Context(), "error.captcha_invalid"))
			return
		}
	}

	_, err = h.registrations.Register(r.Context(), event.ID, name, email, comment, userID)
	if err != nil {
		h.renderError(w, r, event, mapRegistrationError(r.Context(), err))
		return
	}

	middleware.SetFlash(w, r, "success", i18n.T(r.Context(), "flash.registration_confirmed"))
	http.Redirect(w, r, "/event/"+slug, http.StatusFound)
}

func (h *RegistrationHandler) renderError(w http.ResponseWriter, r *http.Request, event *models.Event, errMsg string) {
	siteName, accentColor := h.settings.GetSiteSettings()
	csrfField := middleware.CSRFTemplateField(r)
	regs, _ := h.registrations.ListByEvent(event.ID)
	var captchaQuestion string
	if middleware.GetUserID(r) == "" {
		captchaQuestion = captcha.Generate(w, r).Question
	}
	w.WriteHeader(http.StatusBadRequest)
	public.Event(event, regs, csrfField, siteName, accentColor, "", errMsg, captchaQuestion, nil, "", false).Render(r.Context(), w)
}

func (h *RegistrationHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	reg, err := h.registrations.Cancel(token)
	if err != nil {
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	if reg == nil {
		http.NotFound(w, r)
		return
	}

	// Redirect to event page with cancellation message
	event, err := h.events.GetByID(reg.EventID)
	if err != nil || event == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	regs, _ := h.registrations.ListByEvent(event.ID)
	siteName, accentColor := h.settings.GetSiteSettings()
	csrfField := middleware.CSRFTemplateField(r)
	var captchaQuestion string
	if middleware.GetUserID(r) == "" {
		captchaQuestion = captcha.Generate(w, r).Question
	}
	public.Event(event, regs, csrfField, siteName, accentColor, "", i18n.T(r.Context(), "flash.registration_canceled"), captchaQuestion, nil, "", false).Render(r.Context(), w)
}

func (h *RegistrationHandler) CancelByUser(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	userID := middleware.GetUserID(r)
	if userID == "" {
		http.Redirect(w, r, "/admin/login", http.StatusFound)
		return
	}

	event, err := h.events.GetBySlug(slug)
	if err != nil || event == nil {
		http.NotFound(w, r)
		return
	}

	if err := h.registrations.CancelByUser(userID, event.ID); err != nil {
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	middleware.SetFlash(w, r, "success", i18n.T(r.Context(), "flash.registration_canceled"))
	http.Redirect(w, r, "/event/"+slug, http.StatusFound)
}

func mapRegistrationError(ctx context.Context, err error) string {
	switch {
	case errors.Is(err, services.ErrEventNotFound):
		return i18n.T(ctx, "error.event_not_found")
	case errors.Is(err, services.ErrRegistrationNotOpen):
		return i18n.T(ctx, "error.registration_not_open")
	case errors.Is(err, services.ErrRegistrationDeadlinePassed):
		return i18n.T(ctx, "error.registration_deadline_passed")
	case errors.Is(err, services.ErrRegistrationFull):
		return i18n.T(ctx, "error.registration_full")
	case errors.Is(err, services.ErrAlreadyRegistered):
		return i18n.T(ctx, "error.already_registered")
	default:
		return i18n.T(ctx, "error.internal")
	}
}
