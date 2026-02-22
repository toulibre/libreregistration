package services

import (
	"bytes"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yuin/goldmark"

	"github.com/toulibre/libreregistration/internal/database"
	"github.com/toulibre/libreregistration/internal/models"
	"github.com/toulibre/libreregistration/internal/slug"
)

type EventService struct {
	events *database.EventStore
	md     goldmark.Markdown
}

func NewEventService(events *database.EventStore) *EventService {
	return &EventService{
		events: events,
		md:     goldmark.New(),
	}
}

func (s *EventService) Create(e *models.Event) error {
	e.ID = uuid.New().String()
	if e.Slug == "" {
		e.Slug = slug.Generate(e.Title)
	}

	// Ensure unique slug
	base := e.Slug
	for i := 1; ; i++ {
		candidate := base
		if i > 1 {
			candidate = fmt.Sprintf("%s-%d", base, i)
		}
		exists, err := s.events.SlugExists(candidate)
		if err != nil {
			return fmt.Errorf("check slug: %w", err)
		}
		if !exists {
			e.Slug = candidate
			break
		}
	}

	now := time.Now()
	e.CreatedAt = now
	e.UpdatedAt = now

	return s.events.Create(e)
}

func (s *EventService) Update(e *models.Event) error {
	e.UpdatedAt = time.Now()
	return s.events.Update(e)
}

func (s *EventService) GetByID(id string) (*models.Event, error) {
	e, err := s.events.GetByID(id)
	if err != nil {
		return nil, err
	}
	if e != nil {
		e.DescriptionHTML = s.renderMarkdown(e.Description)
	}
	return e, nil
}

func (s *EventService) GetBySlug(slug string) (*models.Event, error) {
	e, err := s.events.GetBySlug(slug)
	if err != nil {
		return nil, err
	}
	if e != nil {
		e.DescriptionHTML = s.renderMarkdown(e.Description)
	}
	return e, nil
}

func (s *EventService) ListUpcoming() ([]models.Event, error) {
	return s.events.ListUpcoming()
}

func (s *EventService) ListAll() ([]models.Event, error) {
	return s.events.ListAll()
}

func (s *EventService) Delete(id string) error {
	return s.events.Delete(id)
}

func (s *EventService) Clone(id, userID, suffix string) (*models.Event, error) {
	original, err := s.events.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("get event for clone: %w", err)
	}
	if original == nil {
		return nil, fmt.Errorf("event not found")
	}

	clone := &models.Event{
		Title:                original.Title + " " + suffix,
		Description:          original.Description,
		Location:             original.Location,
		EventDate:            original.EventDate,
		RegistrationDeadline: original.RegistrationDeadline,
		MaxCapacity:          original.MaxCapacity,
		AttendeeListPublic:   original.AttendeeListPublic,
		RegistrationOpen:     false, // clones start closed
		ImagePath:            original.ImagePath,
		BannerPath:           original.BannerPath,
		Latitude:             original.Latitude,
		Longitude:            original.Longitude,
		CreatedBy:            userID,
	}

	if err := s.Create(clone); err != nil {
		return nil, fmt.Errorf("create clone: %w", err)
	}

	return clone, nil
}

func (s *EventService) Count() (int, error) {
	return s.events.Count()
}

func (s *EventService) CountUpcoming() (int, error) {
	return s.events.CountUpcoming()
}

func (s *EventService) renderMarkdown(source string) string {
	var buf bytes.Buffer
	if err := s.md.Convert([]byte(source), &buf); err != nil {
		return source
	}
	return buf.String()
}
