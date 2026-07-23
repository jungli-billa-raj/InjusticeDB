package models

import (
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID         uuid.UUID `json:"id"`
	IncidentID uuid.UUID `json:"incident_id"`
	UserID     uuid.UUID `json:"user_id"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`

	// Join fields for display
	UserName string `json:"user_name,omitempty"`
}

type YDCIDCTarget struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	Occupation        *string   `json:"occupation,omitempty"`
	State             *string   `json:"state,omitempty"`
	City              *string   `json:"city,omitempty"`
	CauseOfResentment string    `json:"cause_of_resentment"`
	CreatedAt         time.Time `json:"created_at"`
}
