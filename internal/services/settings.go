package services

import (
	"github.com/toulibre/libreregistration/internal/database"
	"github.com/toulibre/libreregistration/internal/models"
)

type SettingsService struct {
	settings *database.SettingStore
}

func NewSettingsService(settings *database.SettingStore) *SettingsService {
	return &SettingsService{settings: settings}
}

func (s *SettingsService) GetSiteSettings() (siteName, accentColor string) {
	siteName, _ = s.settings.Get("site_name")
	if siteName == "" {
		siteName = "LibreRegistration"
	}
	accentColor, _ = s.settings.Get("accent_color")
	if accentColor == "" {
		accentColor = "#6d28d9"
	}
	return
}

func (s *SettingsService) GetAll() ([]models.Setting, error) {
	return s.settings.GetAll()
}

func (s *SettingsService) Update(settings map[string]string) error {
	for key, value := range settings {
		if err := s.settings.Set(key, value); err != nil {
			return err
		}
	}
	return nil
}

func (s *SettingsService) Get(key string) (string, error) {
	return s.settings.Get(key)
}
