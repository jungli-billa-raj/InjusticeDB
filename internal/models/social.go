package models

import (
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID         uuid.UUID  `json:"id"`
	IncidentID uuid.UUID  `json:"incident_id"`
	UserID     uuid.UUID  `json:"user_id"`
	UserName   string     `json:"user_name"`
	ParentID   *uuid.UUID `json:"parent_id,omitempty"` // Pointer because top-level comments have NULL parent_id
	Content    string     `json:"content"`
	CreatedAt  time.Time  `json:"created_at"`
	Replies    []*Comment `json:"replies,omitempty"` // Nested children for the frontend tree
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
