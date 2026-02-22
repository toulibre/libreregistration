package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/toulibre/libreregistration/internal/config"
	"github.com/toulibre/libreregistration/internal/database"
	"github.com/toulibre/libreregistration/internal/mail"
	"github.com/toulibre/libreregistration/internal/models"
)

type RegistrationService struct {
	registrations *database.RegistrationStore
	events        *database.EventStore
	cfg           *config.Config
}

func NewRegistrationService(registrations *database.RegistrationStore, events *database.EventStore, cfg *config.Config) *RegistrationService {
	return &RegistrationService{registrations: registrations, events: events, cfg: cfg}
}

func (s *RegistrationService) Register(ctx context.Context, eventID, name, email, comment string) (*models.Registration, error) {
	// Check event exists and is open
	event, err := s.events.GetByID(eventID)
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}
	if event == nil {
		return nil, ErrEventNotFound
	}
	if !event.RegistrationOpen {
		return nil, ErrRegistrationNotOpen
	}

	// Check deadline
	if event.RegistrationDeadline != nil && time.Now().After(*event.RegistrationDeadline) {
		return nil, ErrRegistrationDeadlinePassed
	}

	// Check capacity
	if event.MaxCapacity != nil {
		count, err := s.registrations.CountByEvent(eventID)
		if err != nil {
			return nil, fmt.Errorf("count registrations: %w", err)
		}
		if count >= *event.MaxCapacity {
			return nil, ErrRegistrationFull
		}
	}

	reg := &models.Registration{
		ID:           uuid.New().String(),
		EventID:      eventID,
		Name:         name,
		Email:        email,
		Comment:      comment,
		CancelToken:  uuid.New().String(),
		RegisteredAt: time.Now(),
	}

	if err := s.registrations.Create(reg); err != nil {
		return nil, fmt.Errorf("create registration: %w", err)
	}

	// Send confirmation email if email provided and SMTP configured
	if email != "" && s.cfg.SMTPHost != "" {
		cancelURL := fmt.Sprintf("%s/cancel/%s", s.cfg.BaseURL, reg.CancelToken)
		go mail.SendConfirmation(s.cfg, ctx, email, event.Title, cancelURL)
	}

	return reg, nil
}

func (s *RegistrationService) Cancel(token string) (*models.Registration, error) {
	reg, err := s.registrations.GetByCancelToken(token)
	if err != nil {
		return nil, fmt.Errorf("get registration: %w", err)
	}
	if reg == nil {
		return nil, nil
	}

	if err := s.registrations.DeleteByToken(token); err != nil {
		return nil, fmt.Errorf("delete registration: %w", err)
	}

	return reg, nil
}

func (s *RegistrationService) ListByEvent(eventID string) ([]models.Registration, error) {
	return s.registrations.ListByEvent(eventID)
}

func (s *RegistrationService) DeleteRegistration(id string) error {
	return s.registrations.Delete(id)
}

func (s *RegistrationService) TotalCount() (int, error) {
	return s.registrations.TotalCount()
}
