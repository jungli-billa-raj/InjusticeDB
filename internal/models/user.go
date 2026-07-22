package models

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleUser      UserRole = "user"
	RoleModerator UserRole = "moderator"
	RoleAdmin     UserRole = "admin"
)

type User struct {
	ID               uuid.UUID `json:"id"`
	Email            string    `json:"email"`
	Name             string    `json:"name"`
	AuthProvider     string    `json:"auth_provider"`
	Role             UserRole  `json:"role"`
	CredibilityScore int       `json:"credibility_score"`
	CreatedAt        time.Time `json:"created_at"`
}

type CreateUserParams struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	AuthProvider string    `json:"auth_provider"`
}
