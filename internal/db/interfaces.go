package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

// UserRepository handles authentication & user profiles
type UserRepository interface {
	CreateOrUpdate(ctx context.Context, params models.CreateUserParams) (*models.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	UpdateCredibility(ctx context.Context, id uuid.UUID, delta int) error
}

// IncidentRepository handles master records and git-style revision history
type IncidentRepository interface {
	Create(ctx context.Context, params models.CreateIncidentParams) (*models.Incident, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.FullLatestIncident, error)
	// List(ctx context.Context, filter models.IncidentFilter) ([]*models.Incident, error)
	UpdateVerificationStatus(ctx context.Context, id uuid.UUID, status models.VerificationStatus) error
	// UpdateJusticeStatus(ctx context.Context, id uuid.UUID, status models.JusticeStatus) error

	// Revision History (Version Control)
	CreateRevision(ctx context.Context, revision models.IncidentRevision) (*models.IncidentRevision, error)
	GetRevision(ctx context.Context, incidentID uuid.UUID, version int) (*models.IncidentRevision, error)
	ListRevisions(ctx context.Context, incidentID uuid.UUID) ([]*models.IncidentRevision, error)
}

// CulpritRepository handles suspects, individuals, and linking them to incidents
type CulpritRepository interface {
	CreatePerson(ctx context.Context, person models.Person) (*models.Person, error)
	LinkToIncident(ctx context.Context, incidentID uuid.UUID, personID uuid.UUID, status models.CulpritStatus) error
	GetCulpritsForIncident(ctx context.Context, incidentID uuid.UUID) ([]*models.IncidentCulprit, error)
	UpdateCulpritStatus(ctx context.Context, incidentID uuid.UUID, personID uuid.UUID, status models.CulpritStatus) error
}

// VerificationRepository handles crowd voting
type VerificationRepository interface {
	CastVote(ctx context.Context, incidentID uuid.UUID, userID uuid.UUID, vote models.VoteType) error
	GetVoteTally(ctx context.Context, incidentID uuid.UUID) (verifyCount int, rejectCount int, err error)
}

// AssetRepository handles evidence media, web archives, and soft deletion
type AssetRepository interface {
	AddAssets(ctx context.Context, assets []models.Asset) error
	GetByIncidentID(ctx context.Context, incidentID uuid.UUID) ([]*models.Asset, error)
	UpdateArchiveURL(ctx context.Context, assetID uuid.UUID, archiveURL string) error
	SoftDeleteAsset(ctx context.Context, assetID uuid.UUID) error
	RestoreAsset(ctx context.Context, assetID uuid.UUID) error
	HardDeleteExpiredAssets(ctx context.Context, daysOld int) ([]string, error)
}

// MessagingRepository handles 1-on-1 private DMs with RLS support
type MessagingRepository interface {
	GetOrCreateConversation(ctx context.Context, currentUserID, targetUserID uuid.UUID) (*models.Conversation, error)
	ListConversations(ctx context.Context, userID uuid.UUID) ([]*models.Conversation, error)
	SendMessage(ctx context.Context, conversationID, senderID uuid.UUID, content string) (*models.Message, error)
	GetMessages(ctx context.Context, requestingUserID uuid.UUID, conversationID uuid.UUID, limit, offset int) ([]*models.Message, error)
}

// CommentRepository handles incident discussion threads
type CommentRepository interface {
	CreateComment(ctx context.Context, incidentID, userID uuid.UUID, content string) (*models.Comment, error)
	ListCommentsByIncident(ctx context.Context, incidentID uuid.UUID, limit, offset int) ([]*models.Comment, error)
}

// TargetRepository handles public registry tracking
type TargetRepository interface {
	CreateTarget(ctx context.Context, target models.YDCIDCTarget) (*models.YDCIDCTarget, error)
	ListTargets(ctx context.Context, limit, offset int) ([]*models.YDCIDCTarget, error)
}
