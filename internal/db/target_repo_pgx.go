package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

type PostgresTargetRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresTargetRepository(pool *pgxpool.Pool) *PostgresTargetRepository {
	return &PostgresTargetRepository{pool: pool}
}

// CreateTarget adds a record to the ydcidc_targets registry.
func (r *PostgresTargetRepository) CreateTarget(ctx context.Context, target models.YDCIDCTarget) (*models.YDCIDCTarget, error) {
	query := `
		INSERT INTO ydcidc_targets (name, occupation, state, city, cause_of_resentment)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, occupation, state, city, cause_of_resentment, created_at;
	`

	var created models.YDCIDCTarget
	err := r.pool.QueryRow(ctx, query,
		target.Name,
		target.Occupation,
		target.State,
		target.City,
		target.CauseOfResentment,
	).Scan(
		&created.ID,
		&created.Name,
		&created.Occupation,
		&created.State,
		&created.City,
		&created.CauseOfResentment,
		&created.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create target entry: %w", err)
	}

	return &created, nil
}

// ListTargets fetches registered targets with pagination.
func (r *PostgresTargetRepository) ListTargets(ctx context.Context, limit, offset int) ([]*models.YDCIDCTarget, error) {
	query := `
		SELECT id, name, occupation, state, city, cause_of_resentment, created_at
		FROM ydcidc_targets
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2;
	`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query targets: %w", err)
	}
	defer rows.Close()

	var targets []*models.YDCIDCTarget
	for rows.Next() {
		var t models.YDCIDCTarget
		err := rows.Scan(
			&t.ID,
			&t.Name,
			&t.Occupation,
			&t.State,
			&t.City,
			&t.CauseOfResentment,
			&t.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan target row: %w", err)
		}
		targets = append(targets, &t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return targets, nil
}
