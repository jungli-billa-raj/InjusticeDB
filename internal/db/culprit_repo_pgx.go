package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

type PostgresCulpritRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresCulpritRepository(pool *pgxpool.Pool) *PostgresCulpritRepository {
	return &PostgresCulpritRepository{pool: pool}
}

// CreatePerson registers a individual/entity into the database.
func (r *PostgresCulpritRepository) CreatePerson(ctx context.Context, person models.Person) (*models.Person, error) {
	query := `
		INSERT INTO people (name, organization, age, state, city)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, organization, age, state, city, created_at;
	`

	var created models.Person
	err := r.pool.QueryRow(ctx, query,
		person.Name,
		person.Organization,
		person.Age,
		person.State,
		person.City,
	).Scan(
		&created.ID,
		&created.Name,
		&created.Organization,
		&created.Age,
		&created.State,
		&created.City,
		&created.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create person record: %w", err)
	}

	return &created, nil
}

// LinkToIncident links a person to an incident in the junction table with an initial status.
func (r *PostgresCulpritRepository) LinkToIncident(ctx context.Context, incidentID uuid.UUID, personID uuid.UUID, status models.CulpritStatus) error {
	query := `
		INSERT INTO incident_culprits (incident_id, person_id, culprit_status)
		VALUES ($1, $2, $3)
		ON CONFLICT (incident_id, person_id) 
		DO UPDATE SET culprit_status = EXCLUDED.culprit_status;
	`

	_, err := r.pool.Exec(ctx, query, incidentID, personID, status)
	if err != nil {
		return fmt.Errorf("failed to link culprit to incident: %w", err)
	}

	return nil
}

// GetCulpritsForIncident fetches all culprits linked to a specific incident along with their person details.
func (r *PostgresCulpritRepository) GetCulpritsForIncident(ctx context.Context, incidentID uuid.UUID) ([]*models.IncidentCulprit, error) {
	query := `
		SELECT 
			ic.incident_id, ic.person_id, ic.culprit_status,
			p.id, p.name, p.organization, p.age, p.state, p.city, p.created_at
		FROM incident_culprits ic
		JOIN people p ON ic.person_id = p.id
		WHERE ic.incident_id = $1;
	`

	rows, err := r.pool.Query(ctx, query, incidentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query culprits for incident: %w", err)
	}
	defer rows.Close()

	var culprits []*models.IncidentCulprit
	for rows.Next() {
		var ic models.IncidentCulprit
		var p models.Person

		err := rows.Scan(
			&ic.IncidentID,
			&ic.PersonID,
			&ic.CulpritStatus,
			&p.ID,
			&p.Name,
			&p.Organization,
			&p.Age,
			&p.State,
			&p.City,
			&p.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan culprit row: %w", err)
		}

		ic.Person = p
		culprits = append(culprits, &ic)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return culprits, nil
}

// UpdateCulpritStatus updates the legal status of a specific suspect for an incident (e.g., from suspect to convicted).
func (r *PostgresCulpritRepository) UpdateCulpritStatus(ctx context.Context, incidentID uuid.UUID, personID uuid.UUID, status models.CulpritStatus) error {
	query := `
		UPDATE incident_culprits
		SET culprit_status = $1::culprit_status_type
		WHERE incident_id = $2 AND person_id = $3;
	`

	result, err := r.pool.Exec(ctx, query, status, incidentID, personID)
	if err != nil {
		return fmt.Errorf("failed to update culprit status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("culprit-incident link not found")
	}

	return nil
}
