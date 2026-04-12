package models

import "time"

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleManager Role = "manager"
	RoleUser    Role = "user"
)

type User struct {
	ID                   string
	Username             string
	Name                 string
	Email                string
	AvatarPath           string
	PasswordHash         string
	Role                 Role
	EmailVerified        bool
	EmailVerifyToken     string
	PasswordResetToken   string
	PasswordResetExpires *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
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
	UserID       *string
	Name         string
	Email        string
	Comment      string
	CancelToken  string
	RegisteredAt time.Time
	EventTitle   string // computed, not stored (for user dashboard)
	EventSlug    string // computed, not stored (for user dashboard)
}

type Setting struct {
	Key   string
	Value string
}

type Pagination struct {
	Page       int
	PageSize   int
	TotalItems int
	TotalPages int
}

func NewPagination(page, pageSize, totalItems int) Pagination {
	totalPages := (totalItems + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}
	return Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
}

func (p Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

func (p Pagination) HasPrev() bool {
	return p.Page > 1
}

func (p Pagination) HasNext() bool {
	return p.Page < p.TotalPages
}
