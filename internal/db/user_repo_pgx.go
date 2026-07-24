package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

// CreateOrUpdate creates a user if they don't exist, or updates their profile details on re-login (Upsert).
func (r *PostgresUserRepository) CreateOrUpdate(ctx context.Context, params models.CreateUserParams) (*models.User, error) {
	query := `
		INSERT INTO users (email, name, auth_provider)
		VALUES ($1, $2, $3)
		ON CONFLICT (email) DO UPDATE 
		SET name = EXCLUDED.name, auth_provider = EXCLUDED.auth_provider
		RETURNING id, email, name, auth_provider, role, credibility_score, created_at;
	`

	var user models.User
	err := r.pool.QueryRow(ctx, query,
		params.Email,
		params.Name,
		params.AuthProvider,
	).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.AuthProvider,
		&user.Role,
		&user.CredibilityScore,
		&user.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert user: %w", err)
	}

	return &user, nil
}

// GetByID retrieves a user profile by their UUID.
func (r *PostgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, email, name, auth_provider, role, credibility_score, created_at
		FROM users
		WHERE id = $1;
	`

	var user models.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.AuthProvider,
		&user.Role,
		&user.CredibilityScore,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	return &user, nil
}

// UpdateCredibility adjusts a user's credibility score by a positive or negative delta value.
func (r *PostgresUserRepository) UpdateCredibility(ctx context.Context, id uuid.UUID, delta int) error {
	query := `
		UPDATE users
		SET credibility_score = GREATEST(0, credibility_score + $1)
		WHERE id = $2;
	`

	result, err := r.pool.Exec(ctx, query, delta, id)
	if err != nil {
		return fmt.Errorf("failed to update user credibility: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}
