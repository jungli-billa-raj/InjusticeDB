package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

// setupVerificationTestDB prepares a pool and truncates related tables for isolation.
func setupVerificationTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/injusticedb?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err, "failed to connect to test database")

	_, err = pool.Exec(ctx, "TRUNCATE TABLE users, incidents, incident_verifications CASCADE;")
	require.NoError(t, err, "failed to truncate tables")

	teardown := func() {
		pool.Close()
	}

	return pool, teardown
}

func TestPostgresVerificationRepository_CastVoteAndGetTally(t *testing.T) {
	pool, teardown := setupVerificationTestDB(t)
	defer teardown()

	userRepo := NewPostgresUserRepository(pool)
	incidentRepo := NewPostgresIncidentRepository(pool)
	verificationRepo := NewPostgresVerificationRepository(pool)
	ctx := context.Background()

	// Seed Users
	u1, err := userRepo.CreateOrUpdate(ctx, models.CreateUserParams{
		Email:        "voter1@example.com",
		Name:         "Voter One",
		AuthProvider: "google",
	})
	require.NoError(t, err)

	u2, err := userRepo.CreateOrUpdate(ctx, models.CreateUserParams{
		Email:        "voter2@example.com",
		Name:         "Voter Two",
		AuthProvider: "google",
	})
	require.NoError(t, err)

	u3, err := userRepo.CreateOrUpdate(ctx, models.CreateUserParams{
		Email:        "voter3@example.com",
		Name:         "Voter Three",
		AuthProvider: "google",
	})
	require.NoError(t, err)

	// Seed Incident
	inc, err := incidentRepo.Create(ctx, models.CreateIncidentParams{
		Title:         "Verification Vote Test Incident",
		FullStory:     "Testing community verification vote tallying.",
		Severity:      6,
		JusticeStatus: models.JusticeProceeding,
		State:         "Jharkhand",
		City:          "Ranchi",
	})
	require.NoError(t, err)

	t.Run("GetVoteTally returns 0, 0 for new incident with no votes", func(t *testing.T) {
		verifyCount, rejectCount, err := verificationRepo.GetVoteTally(ctx, inc.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, verifyCount)
		assert.Equal(t, 0, rejectCount)
	})

	t.Run("CastVote records multiple votes and calculates correct tally", func(t *testing.T) {
		// User 1 votes 'verify'
		err := verificationRepo.CastVote(ctx, inc.ID, u1.ID, models.VoteVerify)
		require.NoError(t, err)

		// User 2 votes 'verify'
		err = verificationRepo.CastVote(ctx, inc.ID, u2.ID, models.VoteVerify)
		require.NoError(t, err)

		// User 3 votes 'reject'
		err = verificationRepo.CastVote(ctx, inc.ID, u3.ID, models.VoteReject)
		require.NoError(t, err)

		verifyCount, rejectCount, err := verificationRepo.GetVoteTally(ctx, inc.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, verifyCount)
		assert.Equal(t, 1, rejectCount)
	})

	t.Run("CastVote updates vote on conflict when user changes vote", func(t *testing.T) {
		// User 1 changes vote from 'verify' to 'reject'
		err := verificationRepo.CastVote(ctx, inc.ID, u1.ID, models.VoteReject)
		require.NoError(t, err)

		verifyCount, rejectCount, err := verificationRepo.GetVoteTally(ctx, inc.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, verifyCount, "Verify count should drop to 1")
		assert.Equal(t, 2, rejectCount, "Reject count should increase to 2")
	})
}
