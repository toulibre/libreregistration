package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/toulibre/libreregistration/internal/models"
)

type EventStore struct {
	db *DB
}

func NewEventStore(db *DB) *EventStore {
	return &EventStore{db: db}
}

func (s *EventStore) Create(e *models.Event) error {
	_, err := s.db.Exec(`INSERT INTO events
		(id, title, slug, description, location, event_date, registration_deadline, max_capacity,
		 attendee_list_public, registration_open, image_path, banner_path, latitude, longitude,
		 created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.Title, e.Slug, e.Description, e.Location, e.EventDate,
		e.RegistrationDeadline, e.MaxCapacity,
		e.AttendeeListPublic, e.RegistrationOpen,
		e.ImagePath, e.BannerPath, e.Latitude, e.Longitude,
		e.CreatedBy, e.CreatedAt, e.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create event: %w", err)
	}
	return nil
}

func (s *EventStore) Update(e *models.Event) error {
	_, err := s.db.Exec(`UPDATE events SET
		title = ?, slug = ?, description = ?, location = ?, event_date = ?,
		registration_deadline = ?, max_capacity = ?,
		attendee_list_public = ?, registration_open = ?,
		image_path = ?, banner_path = ?, latitude = ?, longitude = ?,
		updated_at = ?
		WHERE id = ?`,
		e.Title, e.Slug, e.Description, e.Location, e.EventDate,
		e.RegistrationDeadline, e.MaxCapacity,
		e.AttendeeListPublic, e.RegistrationOpen,
		e.ImagePath, e.BannerPath, e.Latitude, e.Longitude,
		e.UpdatedAt, e.ID,
	)
	if err != nil {
		return fmt.Errorf("update event: %w", err)
	}
	return nil
}

func (s *EventStore) GetByID(id string) (*models.Event, error) {
	return s.scanEvent(s.db.QueryRow(`SELECT e.id, e.title, e.slug, e.description, e.location, e.event_date,
		e.registration_deadline, e.max_capacity, e.attendee_list_public, e.registration_open,
		e.image_path, e.banner_path, e.latitude, e.longitude,
		e.created_by, e.created_at, e.updated_at,
		(SELECT COUNT(*) FROM registrations WHERE event_id = e.id)
		FROM events e WHERE e.id = ?`, id))
}

func (s *EventStore) GetBySlug(slug string) (*models.Event, error) {
	return s.scanEvent(s.db.QueryRow(`SELECT e.id, e.title, e.slug, e.description, e.location, e.event_date,
		e.registration_deadline, e.max_capacity, e.attendee_list_public, e.registration_open,
		e.image_path, e.banner_path, e.latitude, e.longitude,
		e.created_by, e.created_at, e.updated_at,
		(SELECT COUNT(*) FROM registrations WHERE event_id = e.id)
		FROM events e WHERE e.slug = ?`, slug))
}

func (s *EventStore) SlugExists(slug string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM events WHERE slug = ?", slug).Scan(&count)
	return count > 0, err
}

func (s *EventStore) ListUpcoming() ([]models.Event, error) {
	return s.listEvents("WHERE e.event_date >= ? AND e.registration_open = true ORDER BY e.event_date ASC", time.Now())
}

func (s *EventStore) ListUpcomingPaged(limit, offset int) ([]models.Event, error) {
	return s.listEvents(fmt.Sprintf("WHERE e.event_date >= ? AND e.registration_open = true ORDER BY e.event_date ASC LIMIT %d OFFSET %d", limit, offset), time.Now())
}

func (s *EventStore) ListAll() ([]models.Event, error) {
	return s.listEvents("ORDER BY e.event_date DESC")
}

func (s *EventStore) ListAllPaged(limit, offset int) ([]models.Event, error) {
	return s.listEvents(fmt.Sprintf("ORDER BY e.event_date DESC LIMIT %d OFFSET %d", limit, offset))
}

func (s *EventStore) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM events WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete event: %w", err)
	}
	return nil
}

func (s *EventStore) Count() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM events").Scan(&count)
	return count, err
}

func (s *EventStore) CountUpcoming() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM events WHERE event_date >= ?", time.Now()).Scan(&count)
	return count, err
}

func (s *EventStore) SetOrganizers(eventID string, userIDs []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM event_organizers WHERE event_id = ?", eventID); err != nil {
		return fmt.Errorf("clear organizers: %w", err)
	}
	for _, uid := range userIDs {
		if _, err := tx.Exec("INSERT INTO event_organizers (event_id, user_id) VALUES (?, ?)", eventID, uid); err != nil {
			return fmt.Errorf("insert organizer: %w", err)
		}
	}
	return tx.Commit()
}

func (s *EventStore) GetOrganizers(eventID string) ([]models.User, error) {
	rows, err := s.db.Query(`SELECT u.id, u.username, u.name, u.email, u.avatar_path, u.password_hash, u.role, u.email_verified, u.email_verify_token, u.created_at, u.updated_at
		FROM users u JOIN event_organizers eo ON eo.user_id = u.id
		WHERE eo.event_id = ?`, eventID)
	if err != nil {
		return nil, fmt.Errorf("get organizers: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Name, &u.Email, &u.AvatarPath, &u.PasswordHash, &u.Role, &u.EmailVerified, &u.EmailVerifyToken, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan organizer: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *EventStore) GetOrganizerIDs(eventID string) ([]string, error) {
	rows, err := s.db.Query("SELECT user_id FROM event_organizers WHERE event_id = ?", eventID)
	if err != nil {
		return nil, fmt.Errorf("get organizer IDs: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *EventStore) ListByOrganizer(userID string) ([]models.Event, error) {
	return s.listEvents(`JOIN event_organizers eo ON eo.event_id = e.id WHERE eo.user_id = ? ORDER BY e.event_date DESC`, userID)
}

func (s *EventStore) IsOrganizer(eventID, userID string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM event_organizers WHERE event_id = ? AND user_id = ?", eventID, userID).Scan(&count)
	return count > 0, err
}

func (s *EventStore) GetOrganizerEmails(eventID string) ([]string, error) {
	rows, err := s.db.Query(`SELECT u.email FROM users u
		JOIN event_organizers eo ON eo.user_id = u.id
		WHERE eo.event_id = ? AND u.email != ''`, eventID)
	if err != nil {
		return nil, fmt.Errorf("get organizer emails: %w", err)
	}
	defer rows.Close()

	var emails []string
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}
	return emails, rows.Err()
}

func (s *EventStore) listEvents(where string, args ...interface{}) ([]models.Event, error) {
	query := fmt.Sprintf(`SELECT e.id, e.title, e.slug, e.description, e.location, e.event_date,
		e.registration_deadline, e.max_capacity, e.attendee_list_public, e.registration_open,
		e.image_path, e.banner_path, e.latitude, e.longitude,
		e.created_by, e.created_at, e.updated_at,
		(SELECT COUNT(*) FROM registrations WHERE event_id = e.id)
		FROM events e %s`, where)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		e, err := s.scanEventRow(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, *e)
	}
	return events, rows.Err()
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func (s *EventStore) scanEvent(row *sql.Row) (*models.Event, error) {
	e, err := s.scanEventRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return e, err
}

func (s *EventStore) scanEventRow(row scanner) (*models.Event, error) {
	var e models.Event
	err := row.Scan(
		&e.ID, &e.Title, &e.Slug, &e.Description, &e.Location, &e.EventDate,
		&e.RegistrationDeadline, &e.MaxCapacity, &e.AttendeeListPublic, &e.RegistrationOpen,
		&e.ImagePath, &e.BannerPath, &e.Latitude, &e.Longitude,
		&e.CreatedBy, &e.CreatedAt, &e.UpdatedAt, &e.RegistrationCount,
	)
	if err != nil {
		return nil, fmt.Errorf("scan event: %w", err)
	}
	return &e, nil
}
