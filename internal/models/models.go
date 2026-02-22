package models

import "time"

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleManager Role = "manager"
)

type User struct {
	ID           string
	Username     string
	Name         string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// DisplayName returns the name if set, otherwise the username.
func (u User) DisplayName() string {
	if u.Name != "" {
		return u.Name
	}
	return u.Username
}

type Event struct {
	ID                   string
	Title                string
	Slug                 string
	Description          string
	DescriptionHTML      string // rendered markdown, not stored
	Location             string
	EventDate            time.Time
	RegistrationDeadline *time.Time
	MaxCapacity          *int
	AttendeeListPublic   bool
	RegistrationOpen     bool
	ImagePath            string
	BannerPath           string
	Latitude             *float64
	Longitude            *float64
	CreatedBy            string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	RegistrationCount    int // computed, not stored
}

type Registration struct {
	ID           string
	EventID      string
	Name         string
	Email        string
	Comment      string
	CancelToken  string
	RegisteredAt time.Time
}

type Setting struct {
	Key   string
	Value string
}
