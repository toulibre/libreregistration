package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/toulibre/libreregistration/internal/i18n"
	"github.com/toulibre/libreregistration/internal/middleware"
	"github.com/toulibre/libreregistration/internal/models"
	"github.com/toulibre/libreregistration/internal/services"
	"github.com/toulibre/libreregistration/templates/admin"
)

type AdminHandler struct {
	events        *services.EventService
	registrations *services.RegistrationService
	auth          *services.AuthService
	settings      *services.SettingsService
}

func NewAdminHandler(events *services.EventService, registrations *services.RegistrationService, auth *services.AuthService, settings *services.SettingsService) *AdminHandler {
	return &AdminHandler{events: events, registrations: registrations, auth: auth, settings: settings}
}

func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	totalEvents, _ := h.events.Count()
	upcomingEvents, _ := h.events.CountUpcoming()
	totalRegistrations, _ := h.registrations.TotalCount()

	siteName, accentColor := h.settings.GetSiteSettings()
	admin.Dashboard(siteName, accentColor, middleware.GetDisplayName(r), totalEvents, upcomingEvents, totalRegistrations).Render(r.Context(), w)
}

func (h *AdminHandler) Attendees(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	event, err := h.events.GetByID(id)
	if err != nil || event == nil {
		http.NotFound(w, r)
		return
	}

	regs, err := h.registrations.ListByEvent(id)
	if err != nil {
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
	admin.Attendees(event, regs, siteName, accentColor, middleware.GetDisplayName(r), csrfField, flash).Render(r.Context(), w)
}

func (h *AdminHandler) AttendeesCSV(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	event, err := h.events.GetByID(id)
	if err != nil || event == nil {
		http.NotFound(w, r)
		return
	}

	regs, err := h.registrations.ListByEvent(id)
	if err != nil {
		http.Error(w, i18n.T(ctx, "error.internal"), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf(i18n.T(ctx, "csv.filename_fmt"), event.Slug)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	writer := csv.NewWriter(w)
	writer.Write([]string{
		i18n.T(ctx, "csv.name"),
		i18n.T(ctx, "csv.email"),
		i18n.T(ctx, "csv.comment"),
		i18n.T(ctx, "csv.registered_at"),
	})
	for _, reg := range regs {
		writer.Write([]string{reg.Name, reg.Email, reg.Comment, i18n.FormatDateTimeCSV(ctx, reg.RegisteredAt)})
	}
	writer.Flush()
}

func (h *AdminHandler) DeleteAttendee(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "id")
	regID := chi.URLParam(r, "regID")

	if err := h.registrations.DeleteRegistration(regID); err != nil {
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	middleware.SetFlash(w, r, "success", i18n.T(r.Context(), "flash.registration_deleted"))
	http.Redirect(w, r, fmt.Sprintf("/admin/events/%s/attendees", eventID), http.StatusFound)
}

func (h *AdminHandler) Users(w http.ResponseWriter, r *http.Request) {
	users, err := h.auth.ListUsers()
	if err != nil {
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
	admin.Users(users, siteName, accentColor, middleware.GetDisplayName(r), csrfField, flash).Render(r.Context(), w)
}

func (h *AdminHandler) NewUserForm(w http.ResponseWriter, r *http.Request) {
	siteName, accentColor := h.settings.GetSiteSettings()
	csrfField := middleware.CSRFTemplateField(r)
	admin.UserForm(siteName, accentColor, middleware.GetDisplayName(r), csrfField, "").Render(r.Context(), w)
}

func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	name := r.FormValue("name")
	password := r.FormValue("password")
	role := models.Role(r.FormValue("role"))

	if username == "" || password == "" {
		siteName, accentColor := h.settings.GetSiteSettings()
		csrfField := middleware.CSRFTemplateField(r)
		admin.UserForm(siteName, accentColor, middleware.GetDisplayName(r), csrfField, i18n.T(r.Context(), "error.login_password_required")).Render(r.Context(), w)
		return
	}

	if role != models.RoleAdmin && role != models.RoleManager {
		role = models.RoleManager
	}

	if err := h.auth.CreateUser(username, name, password, role); err != nil {
		siteName, accentColor := h.settings.GetSiteSettings()
		csrfField := middleware.CSRFTemplateField(r)
		admin.UserForm(siteName, accentColor, middleware.GetDisplayName(r), csrfField, i18n.T(r.Context(), "error.creation_failed")).Render(r.Context(), w)
		return
	}

	middleware.SetFlash(w, r, "success", i18n.T(r.Context(), "flash.user_created"))
	http.Redirect(w, r, "/admin/users", http.StatusFound)
}

func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Prevent self-deletion
	if id == middleware.GetUserID(r) {
		middleware.SetFlash(w, r, "success", i18n.T(r.Context(), "flash.cannot_delete_self"))
		http.Redirect(w, r, "/admin/users", http.StatusFound)
		return
	}

	if err := h.auth.DeleteUser(id); err != nil {
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	middleware.SetFlash(w, r, "success", i18n.T(r.Context(), "flash.user_deleted"))
	http.Redirect(w, r, "/admin/users", http.StatusFound)
}

func (h *AdminHandler) Settings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settings.GetAll()
	if err != nil {
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
	admin.Settings(settings, siteName, accentColor, middleware.GetDisplayName(r), csrfField, flash).Render(r.Context(), w)
}

func (h *AdminHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	settings := make(map[string]string)
	for key, values := range r.PostForm {
		if key == "csrf_token" || key == "_method" {
			continue
		}
		if len(values) > 0 {
			settings[key] = values[0]
		}
	}

	if err := h.settings.Update(settings); err != nil {
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	middleware.SetFlash(w, r, "success", i18n.T(r.Context(), "flash.settings_updated"))
	http.Redirect(w, r, "/admin/settings", http.StatusFound)
}
