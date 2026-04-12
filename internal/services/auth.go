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
		ID:            uuid.New().String(),
		Username:      username,
		PasswordHash:  string(hash),
		Role:          models.RoleAdmin,
		EmailVerified: true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.users.Create(user); err != nil {
		return fmt.Errorf("create admin: %w", err)
	}

	log.Printf("Admin user '%s' created", username)
	return nil
}

func (s *AuthService) CreateUser(username, name, email, avatarPath, password string, role models.Role) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()
	user := &models.User{
		ID:            uuid.New().String(),
		Username:      username,
		Name:          name,
		Email:         email,
		AvatarPath:    avatarPath,
		PasswordHash:  string(hash),
		Role:          role,
		EmailVerified: true, // admin-created users are verified
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	return s.users.Create(user)
}

func (s *AuthService) ListUsers() ([]models.User, error) {
	return s.users.List()
}

func (s *AuthService) GetUser(id string) (*models.User, error) {
	return s.users.GetByID(id)
}

func (s *AuthService) UpdateUser(id, username, name, email, avatarPath, password string, role models.Role) error {
	user, err := s.users.GetByID(id)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("update user: not found")
	}

	user.Username = username
	user.Name = name
	user.Email = email
	user.AvatarPath = avatarPath
	user.Role = role

	if err := s.users.Update(user); err != nil {
		return err
	}

	// Update password only if a new one was provided
	if password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}
		if err := s.users.UpdatePassword(id, string(hash)); err != nil {
			return err
		}
	}

	return nil
}

func (s *AuthService) DeleteUser(id string) error {
	return s.users.Delete(id)
}

func (s *AuthService) UserCount() (int, error) {
	return s.users.Count()
}

func (s *AuthService) ChangePassword(userID, currentPassword, newPassword string) error {
	user, err := s.users.GetByID(userID)
	if err != nil {
		return fmt.Errorf("change password: %w", err)
	}
	if user == nil {
		return fmt.Errorf("change password: user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return fmt.Errorf("change password: %w", ErrInvalidCurrentPassword)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	return s.users.UpdatePassword(userID, string(hash))
}

func (s *AuthService) VerifyEmail(token string) (*models.User, error) {
	user, err := s.users.GetByVerifyToken(token)
	if err != nil {
		return nil, fmt.Errorf("verify email: %w", err)
	}
	if user == nil {
		return nil, nil
	}
	if err := s.users.VerifyEmail(user.ID); err != nil {
		return nil, err
	}
	user.EmailVerified = true
	return user, nil
}

func (s *AuthService) GenerateVerifyToken(userID string) (string, error) {
	token := uuid.New().String()
	if err := s.users.SetVerifyToken(userID, token); err != nil {
		return "", err
	}
	return token, nil
}

func (s *AuthService) RequestPasswordReset(email string) (token string, user *models.User, err error) {
	user, err = s.users.GetByEmail(email)
	if err != nil {
		return "", nil, fmt.Errorf("request reset: %w", err)
	}
	if user == nil {
		return "", nil, nil // no user with this email, don't reveal
	}

	token = uuid.New().String()
	expires := time.Now().Add(1 * time.Hour)
	if err := s.users.SetResetToken(user.ID, token, expires); err != nil {
		return "", nil, err
	}
	return token, user, nil
}

func (s *AuthService) ResetPassword(token, newPassword string) (*models.User, error) {
	user, err := s.users.GetByResetToken(token)
	if err != nil {
		return nil, fmt.Errorf("reset password: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidResetToken
	}
	if user.PasswordResetExpires == nil || time.Now().After(*user.PasswordResetExpires) {
		return nil, ErrResetTokenExpired
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	if err := s.users.UpdatePassword(user.ID, string(hash)); err != nil {
		return nil, err
	}
	s.users.ClearResetToken(user.ID)

	// Also verify email since they proved access
	if !user.EmailVerified {
		s.users.VerifyEmail(user.ID)
	}

	return user, nil
}

var ErrInvalidCurrentPassword = fmt.Errorf("invalid current password")
var ErrUsernameTaken = fmt.Errorf("username already taken")
var ErrInvalidResetToken = fmt.Errorf("invalid reset token")
var ErrResetTokenExpired = fmt.Errorf("reset token expired")

func (s *AuthService) Register(username, name, email, password string) (*models.User, error) {
	existing, err := s.users.GetByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("register: %w", err)
	}
	if existing != nil {
		return nil, ErrUsernameTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()
	verifyToken := ""
	if email != "" {
		verifyToken = uuid.New().String()
	}
	user := &models.User{
		ID:               uuid.New().String(),
		Username:         username,
		Name:             name,
		Email:            email,
		PasswordHash:     string(hash),
		Role:             models.RoleUser,
		EmailVerified:    email == "", // no email = nothing to verify
		EmailVerifyToken: verifyToken,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.users.Create(user); err != nil {
		return nil, fmt.Errorf("register: %w", err)
	}
	return user, nil
}

func (s *AuthService) UpdateProfile(id, name, email, avatarPath string) error {
	user, err := s.users.GetByID(id)
	if err != nil {
		return fmt.Errorf("update profile: %w", err)
	}
	if user == nil {
		return fmt.Errorf("update profile: not found")
	}

	user.Name = name
	user.Email = email
	user.AvatarPath = avatarPath
	return s.users.Update(user)
}
