package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

type PostgresVerificationRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresVerificationRepository(pool *pgxpool.Pool) *PostgresVerificationRepository {
	return &PostgresVerificationRepository{pool: pool}
}

// CastVote records a user's verification or rejection vote.
// If the user has already voted, ON CONFLICT updates their existing vote.
func (r *PostgresVerificationRepository) CastVote(ctx context.Context, incidentID uuid.UUID, userID uuid.UUID, vote models.VoteType) error {
	query := `
		INSERT INTO incident_verifications (incident_id, user_id, vote)
		VALUES ($1, $2, $3::vote_type)
		ON CONFLICT (incident_id, user_id)
		DO UPDATE SET vote = EXCLUDED.vote, created_at = CURRENT_TIMESTAMP;
	`

	_, err := r.pool.Exec(ctx, query, incidentID, userID, vote)
	if err != nil {
		return fmt.Errorf("failed to cast vote: %w", err)
	}

	return nil
}

// GetVoteTally calculates the total verify and reject counts for an incident.
func (r *PostgresVerificationRepository) GetVoteTally(ctx context.Context, incidentID uuid.UUID) (verifyCount int, rejectCount int, err error) {
	query := `
		SELECT 
			COUNT(*) FILTER (WHERE vote = 'verify') AS verify_count,
			COUNT(*) FILTER (WHERE vote = 'reject') AS reject_count
		FROM incident_verifications
		WHERE incident_id = $1;
	`

	err = r.pool.QueryRow(ctx, query, incidentID).Scan(&verifyCount, &rejectCount)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to calculate vote tally: %w", err)
	}

	return verifyCount, rejectCount, nil
}
