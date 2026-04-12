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

const userColumns = "id, username, name, email, avatar_path, password_hash, role, email_verified, email_verify_token, password_reset_token, password_reset_expires, created_at, updated_at"

func scanUser(row interface{ Scan(...interface{}) error }) (*models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.Name, &u.Email, &u.AvatarPath, &u.PasswordHash, &u.Role, &u.EmailVerified, &u.EmailVerifyToken, &u.PasswordResetToken, &u.PasswordResetExpires, &u.CreatedAt, &u.UpdatedAt)
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
		"INSERT INTO users (id, username, name, email, avatar_path, password_hash, role, email_verified, email_verify_token, password_reset_token, password_reset_expires, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		u.ID, u.Username, u.Name, u.Email, u.AvatarPath, u.PasswordHash, u.Role, u.EmailVerified, u.EmailVerifyToken, u.PasswordResetToken, u.PasswordResetExpires, u.CreatedAt, u.UpdatedAt,
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

func (s *UserStore) Update(u *models.User) error {
	_, err := s.db.Exec(
		"UPDATE users SET username = ?, name = ?, email = ?, avatar_path = ?, role = ?, updated_at = ? WHERE id = ?",
		u.Username, u.Name, u.Email, u.AvatarPath, u.Role, time.Now(), u.ID,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (s *UserStore) GetByVerifyToken(token string) (*models.User, error) {
	u, err := scanUser(s.db.QueryRow(
		"SELECT "+userColumns+" FROM users WHERE email_verify_token = ?", token,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by verify token: %w", err)
	}
	return u, nil
}

func (s *UserStore) VerifyEmail(id string) error {
	_, err := s.db.Exec(
		"UPDATE users SET email_verified = true, email_verify_token = '', updated_at = ? WHERE id = ?",
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("verify email: %w", err)
	}
	return nil
}

func (s *UserStore) SetVerifyToken(id, token string) error {
	_, err := s.db.Exec(
		"UPDATE users SET email_verify_token = ?, updated_at = ? WHERE id = ?",
		token, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("set verify token: %w", err)
	}
	return nil
}

func (s *UserStore) GetByEmail(email string) (*models.User, error) {
	u, err := scanUser(s.db.QueryRow(
		"SELECT "+userColumns+" FROM users WHERE email = ?", email,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

func (s *UserStore) SetResetToken(id, token string, expires time.Time) error {
	_, err := s.db.Exec(
		"UPDATE users SET password_reset_token = ?, password_reset_expires = ?, updated_at = ? WHERE id = ?",
		token, expires, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("set reset token: %w", err)
	}
	return nil
}

func (s *UserStore) GetByResetToken(token string) (*models.User, error) {
	u, err := scanUser(s.db.QueryRow(
		"SELECT "+userColumns+" FROM users WHERE password_reset_token = ?", token,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by reset token: %w", err)
	}
	return u, nil
}

func (s *UserStore) ClearResetToken(id string) error {
	_, err := s.db.Exec(
		"UPDATE users SET password_reset_token = '', password_reset_expires = NULL, updated_at = ? WHERE id = ?",
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("clear reset token: %w", err)
	}
	return nil
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
