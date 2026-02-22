package database

import (
	"database/sql"
	"fmt"

	"github.com/toulibre/libreregistration/internal/models"
)

type SettingStore struct {
	db *DB
}

func NewSettingStore(db *DB) *SettingStore {
	return &SettingStore{db: db}
}

func (s *SettingStore) Get(key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get setting %s: %w", key, err)
	}
	return value, nil
}

func (s *SettingStore) Set(key, value string) error {
	_, err := s.db.Exec(
		"INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
		key, value,
	)
	if err != nil {
		return fmt.Errorf("set setting %s: %w", key, err)
	}
	return nil
}

func (s *SettingStore) GetAll() ([]models.Setting, error) {
	rows, err := s.db.Query("SELECT key, value FROM settings ORDER BY key")
	if err != nil {
		return nil, fmt.Errorf("list settings: %w", err)
	}
	defer rows.Close()

	var settings []models.Setting
	for rows.Next() {
		var s models.Setting
		if err := rows.Scan(&s.Key, &s.Value); err != nil {
			return nil, fmt.Errorf("scan setting: %w", err)
		}
		settings = append(settings, s)
	}
	return settings, rows.Err()
}
