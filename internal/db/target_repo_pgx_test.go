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

// setupTargetTestDB prepares a pool and truncates the ydcidc_targets table for clean test state.
func setupTargetTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/injusticedb?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err, "failed to connect to test database")

	_, err = pool.Exec(ctx, "TRUNCATE TABLE ydcidc_targets CASCADE;")
	require.NoError(t, err, "failed to truncate ydcidc_targets table")

	teardown := func() {
		pool.Close()
	}

	return pool, teardown
}

func TestPostgresTargetRepository_CreateTarget(t *testing.T) {
	pool, teardown := setupTargetTestDB(t)
	defer teardown()

	repo := NewPostgresTargetRepository(pool)
	ctx := context.Background()

	t.Run("Create target entry with full details successfully", func(t *testing.T) {
		occupation := "Public Official"
		state := "Jharkhand"
		city := "Ranchi"

		targetInput := models.YDCIDCTarget{
			Name:              "Jane Doe",
			Occupation:        &occupation,
			State:             &state,
			City:              &city,
			CauseOfResentment: "Misappropriation of public development funds.",
		}

		created, err := repo.CreateTarget(ctx, targetInput)
		require.NoError(t, err)
		require.NotNil(t, created)

		assert.NotEqual(t, uuid.Nil, created.ID)
		assert.Equal(t, "Jane Doe", created.Name)
		assert.Equal(t, &occupation, created.Occupation)
		assert.Equal(t, &state, created.State)
		assert.Equal(t, &city, created.City)
		assert.Equal(t, "Misappropriation of public development funds.", created.CauseOfResentment)
		assert.False(t, created.CreatedAt.IsZero())
	})

	t.Run("Create target entry with minimal details (nil optional fields)", func(t *testing.T) {
		targetInput := models.YDCIDCTarget{
			Name:              "Unidentified Official",
			CauseOfResentment: "Refusal to issue public record audit.",
		}

		created, err := repo.CreateTarget(ctx, targetInput)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, created.ID)
		assert.Equal(t, "Unidentified Official", created.Name)
		assert.Nil(t, created.Occupation)
		assert.Nil(t, created.State)
		assert.Nil(t, created.City)
	})
}

func TestPostgresTargetRepository_ListTargets(t *testing.T) {
	pool, teardown := setupTargetTestDB(t)
	defer teardown()

	repo := NewPostgresTargetRepository(pool)
	ctx := context.Background()

	t.Run("ListTargets fetches records in descending order of creation with pagination", func(t *testing.T) {
		// Seed 3 targets
		target1, err := repo.CreateTarget(ctx, models.YDCIDCTarget{
			Name:              "Target 1",
			CauseOfResentment: "Reason 1",
		})
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond) // Guarantee created_at timestamp delta

		target2, err := repo.CreateTarget(ctx, models.YDCIDCTarget{
			Name:              "Target 2",
			CauseOfResentment: "Reason 2",
		})
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)

		target3, err := repo.CreateTarget(ctx, models.YDCIDCTarget{
			Name:              "Target 3",
			CauseOfResentment: "Reason 3",
		})
		require.NoError(t, err)

		// Fetch limit 2, offset 0 (Should get Target 3, Target 2)
		page1, err := repo.ListTargets(ctx, 2, 0)
		require.NoError(t, err)
		require.Len(t, page1, 2)

		assert.Equal(t, target3.ID, page1[0].ID)
		assert.Equal(t, target2.ID, page1[1].ID)

		// Fetch limit 2, offset 2 (Should get Target 1)
		page2, err := repo.ListTargets(ctx, 2, 2)
		require.NoError(t, err)
		require.Len(t, page2, 1)

		assert.Equal(t, target1.ID, page2[0].ID)
	})

	t.Run("ListTargets returns empty slice when no targets exist or offset out of bounds", func(t *testing.T) {
		targets, err := repo.ListTargets(ctx, 10, 100)
		require.NoError(t, err)
		assert.Empty(t, targets)
	})
}
