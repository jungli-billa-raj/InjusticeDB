package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

type PostgresAssetRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresAssetRepository(pool *pgxpool.Pool) *PostgresAssetRepository {
	return &PostgresAssetRepository{pool: pool}
}

// AddAssets inserts multiple evidence links or media files in a batch.
func (r *PostgresAssetRepository) AddAssets(ctx context.Context, assets []models.Asset) error {
	if len(assets) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO assets (incident_id, type, url, archive_url)
		VALUES ($1, $2, $3, $4);
	`

	for _, a := range assets {
		batch.Queue(query, a.IncidentID, a.Type, a.URL, a.ArchiveURL)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < len(assets); i++ {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("failed to insert asset at index %d: %w", i, err)
		}
	}

	return nil
}

// GetByIncidentID fetches all active (non-soft-deleted) assets for an incident.
func (r *PostgresAssetRepository) GetByIncidentID(ctx context.Context, incidentID uuid.UUID) ([]*models.Asset, error) {
	query := `
		SELECT id, incident_id, type, url, archive_url, uploaded_at, deleted_at
		FROM assets
		WHERE incident_id = $1 AND deleted_at IS NULL
		ORDER BY uploaded_at ASC;
	`

	rows, err := r.pool.Query(ctx, query, incidentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query assets: %w", err)
	}
	defer rows.Close()

	var assets []*models.Asset
	for rows.Next() {
		var a models.Asset
		err := rows.Scan(
			&a.ID,
			&a.IncidentID,
			&a.Type,
			&a.URL,
			&a.ArchiveURL,
			&a.UploadedAt,
			&a.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan asset row: %w", err)
		}
		assets = append(assets, &a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return assets, nil
}

// SoftDeleteAsset marks an asset as deleted without removing the database row.
func (r *PostgresAssetRepository) SoftDeleteAsset(ctx context.Context, assetID uuid.UUID) error {
	query := `
		UPDATE assets
		SET deleted_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND deleted_at IS NULL;
	`

	res, err := r.pool.Exec(ctx, query, assetID)
	if err != nil {
		return fmt.Errorf("failed to soft delete asset: %w", err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("asset not found or already deleted")
	}

	return nil
}

// RestoreAsset un-marks a soft-deleted asset.
func (r *PostgresAssetRepository) RestoreAsset(ctx context.Context, assetID uuid.UUID) error {
	query := `
		UPDATE assets
		SET deleted_at = NULL
		WHERE id = $1 AND deleted_at IS NOT NULL;
	`

	res, err := r.pool.Exec(ctx, query, assetID)
	if err != nil {
		return fmt.Errorf("failed to restore asset: %w", err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("asset not found or not deleted")
	}

	return nil
}

// HardDeleteExpiredAssets permanently removes assets that were soft-deleted more than daysOld ago.
// Returns the list of deleted URLs so object storage (e.g., S3) can purge matching objects.
func (r *PostgresAssetRepository) HardDeleteExpiredAssets(ctx context.Context, daysOld int) ([]string, error) {
	query := `
		DELETE FROM assets
		WHERE deleted_at IS NOT NULL 
          AND deleted_at < (CURRENT_TIMESTAMP - make_interval(days => $1))
		RETURNING url;
	`

	rows, err := r.pool.Query(ctx, query, daysOld)
	if err != nil {
		return nil, fmt.Errorf("failed to hard delete expired assets: %w", err)
	}
	defer rows.Close()

	var deletedURLs []string
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return nil, fmt.Errorf("failed to scan deleted asset url: %w", err)
		}
		deletedURLs = append(deletedURLs, url)
	}

	return deletedURLs, nil
}

// UpdateArchiveURL updates the archive_url for an asset once background archiving completes.
func (r *PostgresAssetRepository) UpdateArchiveURL(ctx context.Context, assetID uuid.UUID, archiveURL string) error {
	query := `
		UPDATE assets
		SET archive_url = $1
		WHERE id = $2 AND deleted_at IS NULL;
	`

	res, err := r.pool.Exec(ctx, query, archiveURL, assetID)
	if err != nil {
		return fmt.Errorf("failed to update archive url: %w", err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("asset not found or deleted")
	}

	return nil
}
