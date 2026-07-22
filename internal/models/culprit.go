package models

import (
	"time"

	"github.com/google/uuid"
)

type CulpritStatus string

const (
	CulpritSuspect   CulpritStatus = "suspect"
	CulpritAccused   CulpritStatus = "accused"
	CulpritGuilty    CulpritStatus = "guilty"
	CulpritConvicted CulpritStatus = "convicted"
)

type Person struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Organization *string   `json:"organization,omitempty"`
	Age          *int      `json:"age,omitempty"`
	State        *string   `json:"state,omitempty"`
	City         *string   `json:"city,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type IncidentCulprit struct {
	IncidentID    uuid.UUID     `json:"incident_id"`
	PersonID      uuid.UUID     `json:"person_id"`
	CulpritStatus CulpritStatus `json:"culprit_status"`
	Person        Person        `json:"person"`
}
