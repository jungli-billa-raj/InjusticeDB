package models

import (
	"time"

	"github.com/google/uuid"
)

type VerificationStatus string
type JusticeStatus string

const (
	VerificationPending  VerificationStatus = "pending"
	VerificationVerified VerificationStatus = "verified"
	VerificationRejected VerificationStatus = "rejected"
	VerificationDisputed VerificationStatus = "disputed"

	JusticeProceeding JusticeStatus = "proceeding"
	JusticeServed     JusticeStatus = "served"
	JusticeStalled    JusticeStatus = "stalled"
)

type Incident struct {
	ID                 uuid.UUID          `json:"id"`
	Title              string             `json:"title"`
	FullStory          string             `json:"full_story"`
	Severity           int                `json:"severity"`
	State              string             `json:"state"`
	City               string             `json:"city"`
	VerificationStatus VerificationStatus `json:"verification_status"`
	JusticeStatus      JusticeStatus      `json:"justice_status"`
	CurrentVersion     int                `json:"current_version"`
	CreatedBy          *uuid.UUID         `json:"created_by,omitempty"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}

type CreateIncidentParams struct {
	Title     string     `json:"title"`
	FullStory string     `json:"full_story"`
	Severity  int        `json:"severity"`
	State     string     `json:"state"`
	City      string     `json:"city"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
}

type IncidentFilter struct {
	State              string
	City               string
	VerificationStatus string
	Limit              int
	Offset             int
}

type IncidentRevision struct {
	ID            uuid.UUID  `json:"id"`
	IncidentID    uuid.UUID  `json:"incident_id"`
	VersionNumber int        `json:"version_number"`
	Title         string     `json:"title"`
	FullStory     string     `json:"full_story"`
	ChangeSummary string     `json:"change_summary"`
	EditedBy      *uuid.UUID `json:"edited_by,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}
