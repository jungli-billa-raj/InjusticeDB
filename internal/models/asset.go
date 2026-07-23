package models

import (
	"time"

	"github.com/google/uuid"
)

type AssetType string

const (
	AssetImage       AssetType = "image"
	AssetVideo       AssetType = "video"
	AssetArticle     AssetType = "article"
	AssetArchiveLink AssetType = "archive_link"
)

type Asset struct {
	ID         uuid.UUID  `json:"id"`
	IncidentID uuid.UUID  `json:"incident_id"`
	Type       AssetType  `json:"type"`
	URL        string     `json:"url"`
	ArchiveURL *string    `json:"archive_url,omitempty"`
	UploadedAt time.Time  `json:"uploaded_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}
