package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

// setupAssetTestDB prepares a pool and truncates related tables for a clean test state.
func setupAssetTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/injusticedb?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err, "failed to connect to test database")

	// Truncate tables to ensure clean test isolation
	_, err = pool.Exec(ctx, "TRUNCATE TABLE incidents, assets CASCADE;")
	require.NoError(t, err, "failed to truncate tables")

	teardown := func() {
		pool.Close()
	}

	return pool, teardown
}

func TestPostgresAssetRepository_BatchAndGet(t *testing.T) {
	pool, teardown := setupAssetTestDB(t)
	defer teardown()

	incidentRepo := NewPostgresIncidentRepository(pool)
	assetRepo := NewPostgresAssetRepository(pool)
	ctx := context.Background()

	// Seed parent incident
	inc, err := incidentRepo.Create(ctx, models.CreateIncidentParams{
		Title:         "Asset Test Incident",
		FullStory:     "Story detailing evidence files.",
		Severity:      5,
		JusticeStatus: models.JusticeProceeding,
		State:         "Punjab",
		City:          "Amritsar",
	})
	require.NoError(t, err)

	t.Run("AddAssets inserts batch and GetByIncidentID fetches active items", func(t *testing.T) {
		archiveVal := "https://web.archive.org/web/12345"
		assets := []models.Asset{
			{
				IncidentID: inc.ID,
				Type:       models.AssetImage,
				URL:        "https://storage.example.com/photo1.jpg",
			},
			{
				IncidentID: inc.ID,
				Type:       models.AssetArticle,
				URL:        "https://news.example.com/report1",
				ArchiveURL: &archiveVal,
			},
		}

		err := assetRepo.AddAssets(ctx, assets)
		require.NoError(t, err)

		fetched, err := assetRepo.GetByIncidentID(ctx, inc.ID)
		require.NoError(t, err)
		require.Len(t, fetched, 2)

		assert.Equal(t, models.AssetImage, fetched[0].Type)
		assert.Equal(t, "https://storage.example.com/photo1.jpg", fetched[0].URL)
		assert.Nil(t, fetched[0].DeletedAt)

		assert.Equal(t, models.AssetArticle, fetched[1].Type)
		require.NotNil(t, fetched[1].ArchiveURL)
		assert.Equal(t, archiveVal, *fetched[1].ArchiveURL)
	})

	t.Run("AddAssets with empty slice returns nil without errors", func(t *testing.T) {
		err := assetRepo.AddAssets(ctx, []models.Asset{})
		assert.NoError(t, err)
	})
}

func TestPostgresAssetRepository_UpdateArchiveURL(t *testing.T) {
	pool, teardown := setupAssetTestDB(t)
	defer teardown()

	incidentRepo := NewPostgresIncidentRepository(pool)
	assetRepo := NewPostgresAssetRepository(pool)
	ctx := context.Background()

	inc, err := incidentRepo.Create(ctx, models.CreateIncidentParams{
		Title:         "Archive URL Incident",
		FullStory:     "Story for archive testing.",
		Severity:      4,
		JusticeStatus: models.JusticeProceeding,
		State:         "Haryana",
		City:          "Gurugram",
	})
	require.NoError(t, err)

	err = assetRepo.AddAssets(ctx, []models.Asset{
		{
			IncidentID: inc.ID,
			Type:       models.AssetVideo,
			URL:        "https://storage.example.com/video1.mp4",
		},
	})
	require.NoError(t, err)

	assets, err := assetRepo.GetByIncidentID(ctx, inc.ID)
	require.NoError(t, err)
	require.Len(t, assets, 1)

	assetID := assets[0].ID

	t.Run("UpdateArchiveURL updates link successfully", func(t *testing.T) {
		archiveURL := "https://archive.is/abcde"
		err := assetRepo.UpdateArchiveURL(ctx, assetID, archiveURL)
		require.NoError(t, err)

		updatedList, err := assetRepo.GetByIncidentID(ctx, inc.ID)
		require.NoError(t, err)
		require.NotNil(t, updatedList[0].ArchiveURL)
		assert.Equal(t, archiveURL, *updatedList[0].ArchiveURL)
	})

	t.Run("UpdateArchiveURL returns error for non-existent asset", func(t *testing.T) {
		err := assetRepo.UpdateArchiveURL(ctx, uuid.New(), "https://archive.is/invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "asset not found or deleted")
	})
}

func TestPostgresAssetRepository_SoftDeleteAndRestore(t *testing.T) {
	pool, teardown := setupAssetTestDB(t)
	defer teardown()

	incidentRepo := NewPostgresIncidentRepository(pool)
	assetRepo := NewPostgresAssetRepository(pool)
	ctx := context.Background()

	inc, err := incidentRepo.Create(ctx, models.CreateIncidentParams{
		Title:         "Soft Delete Incident",
		FullStory:     "Testing soft delete lifecycle.",
		Severity:      2,
		JusticeStatus: models.JusticeProceeding,
		State:         "Karnataka",
		City:          "Bengaluru",
	})
	require.NoError(t, err)

	err = assetRepo.AddAssets(ctx, []models.Asset{
		{
			IncidentID: inc.ID,
			Type:       models.AssetImage,
			URL:        "https://storage.example.com/delete_me.png",
		},
	})
	require.NoError(t, err)

	assets, err := assetRepo.GetByIncidentID(ctx, inc.ID)
	require.NoError(t, err)
	assetID := assets[0].ID

	t.Run("SoftDeleteAsset hides item from active list", func(t *testing.T) {
		err := assetRepo.SoftDeleteAsset(ctx, assetID)
		require.NoError(t, err)

		activeAssets, err := assetRepo.GetByIncidentID(ctx, inc.ID)
		require.NoError(t, err)
		assert.Len(t, activeAssets, 0)
	})

	t.Run("SoftDeleteAsset returns error when deleting already deleted asset", func(t *testing.T) {
		err := assetRepo.SoftDeleteAsset(ctx, assetID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "asset not found or already deleted")
	})

	t.Run("RestoreAsset brings back soft deleted asset", func(t *testing.T) {
		err := assetRepo.RestoreAsset(ctx, assetID)
		require.NoError(t, err)

		activeAssets, err := assetRepo.GetByIncidentID(ctx, inc.ID)
		require.NoError(t, err)
		require.Len(t, activeAssets, 1)
		assert.Equal(t, assetID, activeAssets[0].ID)
	})
}

func TestPostgresAssetRepository_HardDeleteExpiredAssets(t *testing.T) {
	pool, teardown := setupAssetTestDB(t)
	defer teardown()

	incidentRepo := NewPostgresIncidentRepository(pool)
	assetRepo := NewPostgresAssetRepository(pool)
	ctx := context.Background()

	inc, err := incidentRepo.Create(ctx, models.CreateIncidentParams{
		Title:         "Hard Delete Incident",
		FullStory:     "Testing expired asset cleanup.",
		Severity:      6,
		JusticeStatus: models.JusticeProceeding,
		State:         "Maharashtra",
		City:          "Mumbai",
	})
	require.NoError(t, err)

	targetURL := "https://storage.example.com/expired_file.jpg"
	err = assetRepo.AddAssets(ctx, []models.Asset{
		{
			IncidentID: inc.ID,
			Type:       models.AssetImage,
			URL:        targetURL,
		},
	})
	require.NoError(t, err)

	assets, err := assetRepo.GetByIncidentID(ctx, inc.ID)
	require.NoError(t, err)
	assetID := assets[0].ID

	// Soft delete the asset
	err = assetRepo.SoftDeleteAsset(ctx, assetID)
	require.NoError(t, err)

	// Simulate an old deleted_at date (e.g., 40 days ago) directly in the database
	_, err = pool.Exec(ctx, "UPDATE assets SET deleted_at = CURRENT_TIMESTAMP - INTERVAL '40 days' WHERE id = $1", assetID)
	require.NoError(t, err)

	t.Run("HardDeleteExpiredAssets removes assets older than threshold and returns URLs", func(t *testing.T) {
		// Cleanup assets soft-deleted more than 30 days ago
		deletedURLs, err := assetRepo.HardDeleteExpiredAssets(ctx, 30)
		require.NoError(t, err)
		require.Len(t, deletedURLs, 1)
		assert.Equal(t, targetURL, deletedURLs[0])

		// Verify record is completely gone from DB
		var count int
		err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM assets WHERE id = $1", assetID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}
