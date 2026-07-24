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

// setupIncidentTestDB prepares a pool and truncates related tables for clean test state.
func setupIncidentTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/injusticedb?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err, "failed to connect to test database")

	// Cascade delete dependent data
	_, err = pool.Exec(ctx, "TRUNCATE TABLE users, incidents CASCADE;")
	require.NoError(t, err, "failed to truncate tables")

	teardown := func() {
		pool.Close()
	}

	return pool, teardown
}

func TestPostgresIncidentRepository_CreateAndGetByID(t *testing.T) {
	pool, teardown := setupIncidentTestDB(t)
	defer teardown()

	userRepo := NewPostgresUserRepository(pool)
	incidentRepo := NewPostgresIncidentRepository(pool)
	ctx := context.Background()

	// Seed a user to act as creator
	user, err := userRepo.CreateOrUpdate(ctx, models.CreateUserParams{
		Email:        "reporter@example.com",
		Name:         "Reporter User",
		AuthProvider: "google",
	})
	require.NoError(t, err)

	t.Run("Create incident inserts master and Version 1 revision", func(t *testing.T) {
		params := models.CreateIncidentParams{
			Title:         "Illegal Sand Mining",
			FullStory:     "Unlawful dredging reported along the riverbed.",
			Severity:      7,
			JusticeStatus: models.JusticeProceeding,
			State:         "Jharkhand",
			City:          "Ranchi",
			CreatedBy:     &user.ID,
		}

		inc, err := incidentRepo.Create(ctx, params)
		require.NoError(t, err)
		require.NotNil(t, inc)

		assert.NotEqual(t, uuid.Nil, inc.ID)
		assert.Equal(t, models.VerificationPending, inc.VerificationStatus)
		assert.Equal(t, 1, inc.CurrentVersion)
		assert.Equal(t, &user.ID, inc.CreatedBy)

		// Fetch combined FullLatestIncident via GetByID
		fullInc, err := incidentRepo.GetByID(ctx, inc.ID)
		require.NoError(t, err)
		assert.Equal(t, inc.ID, fullInc.IncidentID)
		assert.Equal(t, params.Title, fullInc.Title)
		assert.Equal(t, params.FullStory, fullInc.FullStory)
		assert.Equal(t, params.Severity, fullInc.Severity)
		assert.Equal(t, params.JusticeStatus, fullInc.JusticeStatus)
		assert.Equal(t, params.State, fullInc.State)
		assert.Equal(t, params.City, fullInc.City)
		assert.Equal(t, 1, fullInc.VersionNumber)
	})

	t.Run("GetByID returns error for non-existent incident", func(t *testing.T) {
		res, err := incidentRepo.GetByID(ctx, uuid.New())
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "incident not found")
	})
}

func TestPostgresIncidentRepository_Revisions(t *testing.T) {
	pool, teardown := setupIncidentTestDB(t)
	defer teardown()

	userRepo := NewPostgresUserRepository(pool)
	incidentRepo := NewPostgresIncidentRepository(pool)
	ctx := context.Background()

	user, err := userRepo.CreateOrUpdate(ctx, models.CreateUserParams{
		Email:        "editor@example.com",
		Name:         "Editor User",
		AuthProvider: "google",
	})
	require.NoError(t, err)

	// Create initial incident (Version 1)
	inc, err := incidentRepo.Create(ctx, models.CreateIncidentParams{
		Title:         "Initial Report Title",
		FullStory:     "Initial narrative text.",
		Severity:      4,
		JusticeStatus: models.JusticeProceeding,
		State:         "Bihar",
		City:          "Patna",
		CreatedBy:     &user.ID,
	})
	require.NoError(t, err)

	t.Run("CreateRevision bumps version number to 2", func(t *testing.T) {
		rev2 := models.IncidentRevision{
			IncidentID:    inc.ID,
			Title:         "Updated Report Title",
			FullStory:     "Updated narrative with official FIR citation.",
			Severity:      8,
			JusticeStatus: models.JusticeProceeding,
			State:         "Bihar",
			City:          "Patna",
			ChangeSummary: "Added FIR references and corrected timestamps.",
			EditedBy:      &user.ID,
		}

		createdRev, err := incidentRepo.CreateRevision(ctx, rev2)
		require.NoError(t, err)
		require.NotNil(t, createdRev)

		assert.NotEqual(t, uuid.Nil, createdRev.ID)
		assert.Equal(t, inc.ID, createdRev.IncidentID)
		assert.Equal(t, 2, createdRev.VersionNumber)
		assert.Equal(t, rev2.Title, createdRev.Title)

		// Verify GetByID now reflects Version 2
		latest, err := incidentRepo.GetByID(ctx, inc.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, latest.VersionNumber)
		assert.Equal(t, "Updated Report Title", latest.Title)
	})

	t.Run("GetRevision fetches exact snapshot history", func(t *testing.T) {
		v1, err := incidentRepo.GetRevision(ctx, inc.ID, 1)
		require.NoError(t, err)
		assert.Equal(t, "Initial Report Title", v1.Title)
		assert.Equal(t, "Initial creation of the record", v1.ChangeSummary)

		v2, err := incidentRepo.GetRevision(ctx, inc.ID, 2)
		require.NoError(t, err)
		assert.Equal(t, "Updated Report Title", v2.Title)
		assert.Equal(t, "Added FIR references and corrected timestamps.", v2.ChangeSummary)
	})

	t.Run("GetRevision returns error for missing version", func(t *testing.T) {
		rev, err := incidentRepo.GetRevision(ctx, inc.ID, 99)
		assert.Error(t, err)
		assert.Nil(t, rev)
		assert.Contains(t, err.Error(), "revision version 99 not found")
	})

	t.Run("ListRevisions returns ordered history array", func(t *testing.T) {
		revisions, err := incidentRepo.ListRevisions(ctx, inc.ID)
		require.NoError(t, err)
		require.Len(t, revisions, 2)

		assert.Equal(t, 1, revisions[0].VersionNumber)
		assert.Equal(t, 2, revisions[1].VersionNumber)
	})
}

func TestPostgresIncidentRepository_UpdateVerificationStatus(t *testing.T) {
	pool, teardown := setupIncidentTestDB(t)
	defer teardown()

	incidentRepo := NewPostgresIncidentRepository(pool)
	ctx := context.Background()

	inc, err := incidentRepo.Create(ctx, models.CreateIncidentParams{
		Title:         "Status Test Incident",
		FullStory:     "Testing status update flow.",
		Severity:      3,
		JusticeStatus: models.JusticeProceeding,
		State:         "Delhi",
		City:          "New Delhi",
	})
	require.NoError(t, err)
	assert.Equal(t, models.VerificationPending, inc.VerificationStatus)

	t.Run("Update status to verified", func(t *testing.T) {
		err := incidentRepo.UpdateVerificationStatus(ctx, inc.ID, models.VerificationVerified)
		require.NoError(t, err)

		fetched, err := incidentRepo.GetByID(ctx, inc.ID)
		require.NoError(t, err)
		assert.Equal(t, models.VerificationVerified, fetched.VerificationStatus)
	})
}
