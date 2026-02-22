package services

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/toulibre/libreregistration/internal/database"
	"github.com/toulibre/libreregistration/internal/models"
)

type AuthService struct {
	users *database.UserStore
}

func NewAuthService(users *database.UserStore) *AuthService {
	return &AuthService{users: users}
}

func (s *AuthService) Authenticate(username, password string) (*models.User, error) {
	user, err := s.users.GetByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}
	if user == nil {
		return nil, nil
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, nil
	}

	return user, nil
}

func (s *AuthService) SeedAdmin(username, password string) error {
	existing, err := s.users.GetByUsername(username)
	if err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}
	if existing != nil {
		return nil // already exists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()
	user := &models.User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: string(hash),
		Role:         models.RoleAdmin,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.users.Create(user); err != nil {
		return fmt.Errorf("create admin: %w", err)
	}

	log.Printf("Admin user '%s' created", username)
	return nil
}

func (s *AuthService) CreateUser(username, name, password string, role models.Role) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()
	user := &models.User{
		ID:           uuid.New().String(),
		Username:     username,
		Name:         name,
		PasswordHash: string(hash),
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	return s.users.Create(user)
}

func (s *AuthService) ListUsers() ([]models.User, error) {
	return s.users.List()
}

func (s *AuthService) DeleteUser(id string) error {
	return s.users.Delete(id)
}

func (s *AuthService) UserCount() (int, error) {
	return s.users.Count()
}
