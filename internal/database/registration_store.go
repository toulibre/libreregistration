package database

import (
	"database/sql"
	"fmt"

	"github.com/toulibre/libreregistration/internal/models"
)

type RegistrationStore struct {
	db *DB
}

func NewRegistrationStore(db *DB) *RegistrationStore {
	return &RegistrationStore{db: db}
}

const regColumns = "id, event_id, name, email, comment, cancel_token, registered_at"

func scanReg(row interface{ Scan(...interface{}) error }) (*models.Registration, error) {
	var r models.Registration
	err := row.Scan(&r.ID, &r.EventID, &r.Name, &r.Email, &r.Comment, &r.CancelToken, &r.RegisteredAt)
	return &r, err
}

func (s *RegistrationStore) Create(r *models.Registration) error {
	_, err := s.db.Exec(
		"INSERT INTO registrations (id, event_id, name, email, comment, cancel_token, registered_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		r.ID, r.EventID, r.Name, r.Email, r.Comment, r.CancelToken, r.RegisteredAt,
	)
	if err != nil {
		return fmt.Errorf("create registration: %w", err)
	}
	return nil
}

func (s *RegistrationStore) GetByCancelToken(token string) (*models.Registration, error) {
	r, err := scanReg(s.db.QueryRow(
		"SELECT "+regColumns+" FROM registrations WHERE cancel_token = ?", token,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get registration by token: %w", err)
	}
	return r, nil
}

func (s *RegistrationStore) ListByEvent(eventID string) ([]models.Registration, error) {
	rows, err := s.db.Query(
		"SELECT "+regColumns+" FROM registrations WHERE event_id = ? ORDER BY registered_at",
		eventID,
	)
	if err != nil {
		return nil, fmt.Errorf("list registrations: %w", err)
	}
	defer rows.Close()

	var regs []models.Registration
	for rows.Next() {
		r, err := scanReg(rows)
		if err != nil {
			return nil, fmt.Errorf("scan registration: %w", err)
		}
		regs = append(regs, *r)
	}
	return regs, rows.Err()
}

func (s *RegistrationStore) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM registrations WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete registration: %w", err)
	}
	return nil
}

func (s *RegistrationStore) DeleteByToken(token string) error {
	_, err := s.db.Exec("DELETE FROM registrations WHERE cancel_token = ?", token)
	if err != nil {
		return fmt.Errorf("delete registration by token: %w", err)
	}
	return nil
}

func (s *RegistrationStore) CountByEvent(eventID string) (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM registrations WHERE event_id = ?", eventID).Scan(&count)
	return count, err
}

func (s *RegistrationStore) TotalCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM registrations").Scan(&count)
	return count, err
}
