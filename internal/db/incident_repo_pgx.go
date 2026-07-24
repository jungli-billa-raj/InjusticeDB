package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jungli-billa-raj/InjusticeDB/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresIncidentRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresIncidentRepository(pool *pgxpool.Pool) *PostgresIncidentRepository {
	return &PostgresIncidentRepository{pool: pool}
}

// Create inserts a new incident and automatically creates its Version 1 revision inside a single transaction.
func (r *PostgresIncidentRepository) Create(ctx context.Context, params models.CreateIncidentParams) (*models.Incident, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Insert Master Incident
	query := `
		INSERT INTO incidents (created_by)
		VALUES ($1)
		RETURNING id, verification_status, current_version, created_by, created_at, updated_at;
	`

	var incident models.Incident
	err = tx.QueryRow(ctx, query,
		params.CreatedBy,
	).Scan(
		&incident.ID,
		&incident.VerificationStatus,
		&incident.CurrentVersion,
		&incident.CreatedBy,
		&incident.CreatedAt,
		&incident.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert incident: %w", err)
	}

	// 2. Insert Version 1 into incident_revisions
	revQuery := `
		INSERT INTO incident_revisions (incident_id, title, full_story, severity, justice_status, state, city, version_number, change_summary, edited_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);
	`
	_, err = tx.Exec(ctx, revQuery,
		incident.ID,
		params.Title,
		params.FullStory,
		params.Severity,
		params.JusticeStatus,
		params.State,
		params.City,
		1,
		"Initial creation of the record",
		params.CreatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create initial revision: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &incident, nil
}

// GetByID fetches a single master incident record along with its latest revision text.
func (r *PostgresIncidentRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.FullLatestIncident, error) {
	query := `
		SELECT 
			i.id, 
			r.title, 
			r.full_story, 
			r.state, 
			r.city, 
			i.verification_status, 
			r.severity,
			r.justice_status, 
			i.current_version, 
			i.created_by, 
			i.created_at, 
			i.updated_at
		FROM incidents i
		JOIN incident_revisions r 
		  ON i.id = r.incident_id AND i.current_version = r.version_number
		WHERE i.id = $1;
	`

	var incident models.FullLatestIncident
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&incident.IncidentID,
		&incident.Title,
		&incident.FullStory,
		&incident.State,
		&incident.City,
		&incident.VerificationStatus,
		&incident.Severity,
		&incident.JusticeStatus,
		&incident.VersionNumber,
		&incident.CreatedBy,
		&incident.CreatedAt,
		&incident.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("incident not found")
		}
		return nil, fmt.Errorf("failed to query incident: %w", err)
	}

	return &incident, nil
}

// CreateRevision updates the current version in master incidents and adds a new revision log.
func (r *PostgresIncidentRepository) CreateRevision(ctx context.Context, rev models.IncidentRevision) (*models.IncidentRevision, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Update Master record title, full_story, and bump version count
	updateMasterQuery := `
		UPDATE incidents
		SET current_version = current_version + 1, updated_at = NOW()
		WHERE id = $1
		RETURNING current_version;
	`

	var newVersion int
	err = tx.QueryRow(ctx, updateMasterQuery, rev.IncidentID).Scan(&newVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to update master incident: %w", err)
	}

	// Insert into incident_revisions table
	insertRevQuery := `
		INSERT INTO incident_revisions (incident_id, title, full_story, severity, justice_status, state, city, version_number,  change_summary, edited_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at;
	`

	var createdRev models.IncidentRevision
	createdRev = rev
	createdRev.VersionNumber = newVersion

	err = tx.QueryRow(ctx, insertRevQuery,
		rev.IncidentID,
		rev.Title,
		rev.FullStory,
		rev.Severity,
		rev.JusticeStatus,
		rev.State,
		rev.City,
		newVersion,
		rev.ChangeSummary,
		rev.EditedBy,
	).Scan(&createdRev.ID, &createdRev.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert revision: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &createdRev, nil
}

// GetRevision fetches a specific historical version of an incident.
func (r *PostgresIncidentRepository) GetRevision(ctx context.Context, incidentID uuid.UUID, version int) (*models.IncidentRevision, error) {
	query := `
		SELECT id, incident_id, title, full_story, severity, justice_status, state, city, version_number, change_summary, edited_by, created_at
		FROM incident_revisions
		WHERE incident_id = $1 AND version_number = $2;
	`

	var rev models.IncidentRevision
	err := r.pool.QueryRow(ctx, query, incidentID, version).Scan(
		&rev.ID,
		&rev.IncidentID,
		&rev.Title,
		&rev.FullStory,
		&rev.Severity,
		&rev.JusticeStatus,
		&rev.State,
		&rev.City,
		&rev.VersionNumber,
		&rev.ChangeSummary,
		&rev.EditedBy,
		&rev.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("revision version %d not found for incident", version)
		}
		return nil, fmt.Errorf("failed to query revision: %w", err)
	}

	return &rev, nil
}

// List fetches incidents based on filter parameters (state, city, status, etc.)
func (r *PostgresIncidentRepository) List(ctx context.Context, filter models.IncidentFilter) ([]*models.Incident, error) {
	query := `
		SELECT id, created_by, title, full_story, state, city,  
		       verification_status, justice_status, created_at, updated_at
		FROM incidents
		WHERE ($1::text IS NULL OR state = $1)
		  AND ($2::text IS NULL OR city = $2)
		  AND ($3::text IS NULL OR verification_status = $3::verification_status)
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5;
	`

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	rows, err := r.pool.Query(ctx, query, filter.State, filter.City, filter.VerificationStatus, limit, filter.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list incidents: %w", err)
	}
	defer rows.Close()

	var incidents []*models.Incident
	for rows.Next() {
		var inc models.Incident
		err := rows.Scan(
			&inc.ID,
			&inc.CreatedBy,
			&inc.Title,
			&inc.FullStory,
			&inc.State,
			&inc.City,
			&inc.VerificationStatus,
			&inc.JusticeStatus,
			&inc.CreatedAt,
			&inc.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan incident row: %w", err)
		}
		incidents = append(incidents, &inc)
	}

	return incidents, nil
}

// ListRevisions retrieves the complete version history for a given incident ordered by version.
func (r *PostgresIncidentRepository) ListRevisions(ctx context.Context, incidentID uuid.UUID) ([]*models.IncidentRevision, error) {
	query := `
		SELECT id, incident_id, version_number, title, full_story, change_summary, edited_by, created_at
		FROM incident_revisions
		WHERE incident_id = $1
		ORDER BY version_number ASC;
	`

	rows, err := r.pool.Query(ctx, query, incidentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list revisions: %w", err)
	}
	defer rows.Close()

	var revisions []*models.IncidentRevision
	for rows.Next() {
		var rev models.IncidentRevision
		err := rows.Scan(
			&rev.ID,
			&rev.IncidentID,
			&rev.VersionNumber,
			&rev.Title,
			&rev.FullStory,
			&rev.ChangeSummary,
			&rev.EditedBy,
			&rev.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan revision row: %w", err)
		}
		revisions = append(revisions, &rev)
	}

	return revisions, nil
}

// UpdateVerificationStatus updates the community verification status of an incident.
func (r *PostgresIncidentRepository) UpdateVerificationStatus(ctx context.Context, id uuid.UUID, status models.VerificationStatus) error {
	query := `
		UPDATE incidents
		SET verification_status = $1, updated_at = NOW()
		WHERE id = $2;
	`
	_, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update verification status: %w", err)
	}
	return nil
}

// UpdateJusticeStatus updates the legal/justice outcome status of an incident.
func (r *PostgresIncidentRepository) UpdateJusticeStatus(ctx context.Context, id uuid.UUID, status models.JusticeStatus) error {
	query := `
		UPDATE incidents
		SET justice_status = $1, updated_at = NOW()
		WHERE id = $2;
	`
	_, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update justice status: %w", err)
	}
	return nil
}
