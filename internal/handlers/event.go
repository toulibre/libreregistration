package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/toulibre/libreregistration/internal/i18n"
	"github.com/toulibre/libreregistration/internal/middleware"
	"github.com/toulibre/libreregistration/internal/models"
	"github.com/toulibre/libreregistration/internal/services"
	"github.com/toulibre/libreregistration/templates/admin"
	"github.com/toulibre/libreregistration/templates/public"
)

type EventHandler struct {
	events        *services.EventService
	registrations *services.RegistrationService
	settings      *services.SettingsService
	uploadDir     string
}

func NewEventHandler(events *services.EventService, registrations *services.RegistrationService, settings *services.SettingsService, uploadDir string) *EventHandler {
	return &EventHandler{events: events, registrations: registrations, settings: settings, uploadDir: uploadDir}
}

// Public routes

func (h *EventHandler) Home(w http.ResponseWriter, r *http.Request) {
	events, err := h.events.ListUpcoming()
	if err != nil {
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}
	siteName, accentColor := h.settings.GetSiteSettings()
	public.Home(events, siteName, accentColor).Render(r.Context(), w)
}

func (h *EventHandler) Show(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	event, err := h.events.GetBySlug(slug)
	if err != nil {
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}
	if event == nil {
		http.NotFound(w, r)
		return
	}

	var regs []models.Registration
	if event.AttendeeListPublic {
		regs, _ = h.registrations.ListByEvent(event.ID)
	}

	siteName, accentColor := h.settings.GetSiteSettings()
	csrfField := middleware.CSRFTemplateField(r)
	flashes := middleware.GetFlashes(w, r, "success")
	flash := ""
	if len(flashes) > 0 {
		flash = flashes[0]
	}

	public.Event(event, regs, csrfField, siteName, accentColor, flash, "").Render(r.Context(), w)
}

// Admin routes

func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
	events, err := h.events.ListAll()
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
	admin.Events(events, siteName, accentColor, middleware.GetDisplayName(r), csrfField, flash).Render(r.Context(), w)
}

func (h *EventHandler) NewForm(w http.ResponseWriter, r *http.Request) {
	siteName, accentColor := h.settings.GetSiteSettings()
	csrfField := middleware.CSRFTemplateField(r)
	event := &models.Event{
		AttendeeListPublic: true,
		RegistrationOpen:   true,
	}
	admin.EventForm(event, false, siteName, accentColor, middleware.GetDisplayName(r), csrfField, "").Render(r.Context(), w)
}

func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	event, err := h.parseEventForm(r)
	if err != nil {
		siteName, accentColor := h.settings.GetSiteSettings()
		csrfField := middleware.CSRFTemplateField(r)
		admin.EventForm(event, false, siteName, accentColor, middleware.GetDisplayName(r), csrfField, err.Error()).Render(r.Context(), w)
		return
	}
	event.CreatedBy = middleware.GetUserID(r)

	imgFile, err := saveUpload(r, "image", h.uploadDir)
	if err != nil {
		siteName, accentColor := h.settings.GetSiteSettings()
		csrfField := middleware.CSRFTemplateField(r)
		admin.EventForm(event, false, siteName, accentColor, middleware.GetDisplayName(r), csrfField, i18n.T(r.Context(), "error.upload_invalid_type")).Render(r.Context(), w)
		return
	}
	event.ImagePath = imgFile

	bannerFile, err := saveUpload(r, "banner", h.uploadDir)
	if err != nil {
		deleteUpload(h.uploadDir, imgFile)
		siteName, accentColor := h.settings.GetSiteSettings()
		csrfField := middleware.CSRFTemplateField(r)
		admin.EventForm(event, false, siteName, accentColor, middleware.GetDisplayName(r), csrfField, i18n.T(r.Context(), "error.upload_invalid_type")).Render(r.Context(), w)
		return
	}
	event.BannerPath = bannerFile

	if err := h.events.Create(event); err != nil {
		deleteUpload(h.uploadDir, imgFile)
		deleteUpload(h.uploadDir, bannerFile)
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	middleware.SetFlash(w, r, "success", i18n.T(r.Context(), "flash.event_created"))
	http.Redirect(w, r, "/admin/events", http.StatusFound)
}

func (h *EventHandler) EditForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	event, err := h.events.GetByID(id)
	if err != nil {
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}
	if event == nil {
		http.NotFound(w, r)
		return
	}
	siteName, accentColor := h.settings.GetSiteSettings()
	csrfField := middleware.CSRFTemplateField(r)
	admin.EventForm(event, true, siteName, accentColor, middleware.GetDisplayName(r), csrfField, "").Render(r.Context(), w)
}

func (h *EventHandler) Update(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	id := chi.URLParam(r, "id")
	existing, err := h.events.GetByID(id)
	if err != nil || existing == nil {
		http.NotFound(w, r)
		return
	}

	event, err := h.parseEventForm(r)
	if err != nil {
		siteName, accentColor := h.settings.GetSiteSettings()
		csrfField := middleware.CSRFTemplateField(r)
		event.ID = id
		event.ImagePath = existing.ImagePath
		event.BannerPath = existing.BannerPath
		admin.EventForm(event, true, siteName, accentColor, middleware.GetDisplayName(r), csrfField, err.Error()).Render(r.Context(), w)
		return
	}
	event.ID = id
	if event.Slug == "" {
		event.Slug = existing.Slug
	}
	event.CreatedBy = existing.CreatedBy
	event.CreatedAt = existing.CreatedAt

	// Handle image upload
	imgFile, err := saveUpload(r, "image", h.uploadDir)
	if err != nil {
		siteName, accentColor := h.settings.GetSiteSettings()
		csrfField := middleware.CSRFTemplateField(r)
		event.ImagePath = existing.ImagePath
		event.BannerPath = existing.BannerPath
		admin.EventForm(event, true, siteName, accentColor, middleware.GetDisplayName(r), csrfField, i18n.T(r.Context(), "error.upload_invalid_type")).Render(r.Context(), w)
		return
	}
	switch {
	case imgFile != "":
		deleteUpload(h.uploadDir, existing.ImagePath)
		event.ImagePath = imgFile
	case r.FormValue("remove_image") == "true":
		deleteUpload(h.uploadDir, existing.ImagePath)
		event.ImagePath = ""
	default:
		event.ImagePath = existing.ImagePath
	}

	// Handle banner upload
	bannerFile, err := saveUpload(r, "banner", h.uploadDir)
	if err != nil {
		siteName, accentColor := h.settings.GetSiteSettings()
		csrfField := middleware.CSRFTemplateField(r)
		event.BannerPath = existing.BannerPath
		admin.EventForm(event, true, siteName, accentColor, middleware.GetDisplayName(r), csrfField, i18n.T(r.Context(), "error.upload_invalid_type")).Render(r.Context(), w)
		return
	}
	switch {
	case bannerFile != "":
		deleteUpload(h.uploadDir, existing.BannerPath)
		event.BannerPath = bannerFile
	case r.FormValue("remove_banner") == "true":
		deleteUpload(h.uploadDir, existing.BannerPath)
		event.BannerPath = ""
	default:
		event.BannerPath = existing.BannerPath
	}

	if err := h.events.Update(event); err != nil {
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	middleware.SetFlash(w, r, "success", i18n.T(r.Context(), "flash.event_updated"))
	http.Redirect(w, r, "/admin/events", http.StatusFound)
}

func (h *EventHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	event, err := h.events.GetByID(id)
	if err != nil || event == nil {
		http.NotFound(w, r)
		return
	}

	if err := h.events.Delete(id); err != nil {
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}

	deleteUpload(h.uploadDir, event.ImagePath)
	deleteUpload(h.uploadDir, event.BannerPath)

	middleware.SetFlash(w, r, "success", i18n.T(r.Context(), "flash.event_deleted"))
	http.Redirect(w, r, "/admin/events", http.StatusFound)
}

func (h *EventHandler) Clone(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	suffix := i18n.T(r.Context(), "clone.suffix")
	_, err := h.events.Clone(id, middleware.GetUserID(r), suffix)
	if err != nil {
		http.Error(w, i18n.T(r.Context(), "error.internal"), http.StatusInternalServerError)
		return
	}
	middleware.SetFlash(w, r, "success", i18n.T(r.Context(), "flash.event_cloned"))
	http.Redirect(w, r, "/admin/events", http.StatusFound)
}

func (h *EventHandler) parseEventForm(r *http.Request) (*models.Event, error) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		r.ParseForm()
	}

	ctx := r.Context()
	title := r.FormValue("title")
	if title == "" {
		return &models.Event{}, errMissing(ctx, "field.title")
	}

	eventDateStr := r.FormValue("event_date")
	if eventDateStr == "" {
		return &models.Event{Title: title}, errMissing(ctx, "field.event_date")
	}
	eventDate, err := time.ParseInLocation("2006-01-02T15:04", eventDateStr, time.Local)
	if err != nil {
		return &models.Event{Title: title}, errInvalid(ctx, "field.event_date")
	}

	event := &models.Event{
		Title:              title,
		Slug:               strings.TrimSpace(r.FormValue("slug")),
		Description:        r.FormValue("description"),
		Location:           r.FormValue("location"),
		EventDate:          eventDate,
		AttendeeListPublic: r.FormValue("attendee_list_public") == "true",
		RegistrationOpen:   r.FormValue("registration_open") == "true",
	}

	if dl := r.FormValue("registration_deadline"); dl != "" {
		t, err := time.ParseInLocation("2006-01-02T15:04", dl, time.Local)
		if err != nil {
			return event, errInvalid(ctx, "field.deadline")
		}
		event.RegistrationDeadline = &t
	}

	if cap := r.FormValue("max_capacity"); cap != "" {
		n, err := strconv.Atoi(cap)
		if err != nil || n < 1 {
			return event, errInvalid(ctx, "field.capacity")
		}
		event.MaxCapacity = &n
	}

	if lat := r.FormValue("latitude"); lat != "" {
		v, err := strconv.ParseFloat(lat, 64)
		if err != nil {
			return event, errInvalid(ctx, "field.latitude")
		}
		event.Latitude = &v
	}

	if lng := r.FormValue("longitude"); lng != "" {
		v, err := strconv.ParseFloat(lng, 64)
		if err != nil {
			return event, errInvalid(ctx, "field.longitude")
		}
		event.Longitude = &v
	}

	return event, nil
}

type validationError struct {
	msg string
}

func (e *validationError) Error() string {
	return e.msg
}

func errMissing(ctx context.Context, fieldKey string) error {
	return &validationError{msg: i18n.Tf(ctx, "error.field_required_fmt", i18n.T(ctx, fieldKey))}
}

func errInvalid(ctx context.Context, fieldKey string) error {
	return &validationError{msg: i18n.Tf(ctx, "error.field_invalid_fmt", i18n.T(ctx, fieldKey))}
}
