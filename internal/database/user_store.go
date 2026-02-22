package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/toulibre/libreregistration/internal/models"
)

type UserStore struct {
	db *DB
}

func NewUserStore(db *DB) *UserStore {
	return &UserStore{db: db}
}

const userColumns = "id, username, name, password_hash, role, created_at, updated_at"

func scanUser(row interface{ Scan(...interface{}) error }) (*models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.Name, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	return &u, err
}

func (s *UserStore) GetByUsername(username string) (*models.User, error) {
	u, err := scanUser(s.db.QueryRow(
		"SELECT "+userColumns+" FROM users WHERE username = ?", username,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return u, nil
}

func (s *UserStore) GetByID(id string) (*models.User, error) {
	u, err := scanUser(s.db.QueryRow(
		"SELECT "+userColumns+" FROM users WHERE id = ?", id,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func (s *UserStore) Create(u *models.User) error {
	_, err := s.db.Exec(
		"INSERT INTO users (id, username, name, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		u.ID, u.Username, u.Name, u.PasswordHash, u.Role, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (s *UserStore) List() ([]models.User, error) {
	rows, err := s.db.Query("SELECT " + userColumns + " FROM users ORDER BY created_at")
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

func (s *UserStore) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (s *UserStore) Count() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func (s *UserStore) UpdatePassword(id string, passwordHash string) error {
	_, err := s.db.Exec(
		"UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?",
		passwordHash, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}
