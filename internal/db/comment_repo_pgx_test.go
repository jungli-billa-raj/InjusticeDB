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

// setupCommentTestDB prepares a pool and cleans up tables for isolation.
func setupCommentTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/injusticedb?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err, "failed to connect to test database")

	_, err = pool.Exec(ctx, "TRUNCATE TABLE users, incidents, comments CASCADE;")
	require.NoError(t, err, "failed to truncate tables")

	teardown := func() {
		pool.Close()
	}

	return pool, teardown
}

func TestPostgresCommentRepository_CreateAndListTree(t *testing.T) {
	pool, teardown := setupCommentTestDB(t)
	defer teardown()

	userRepo := NewPostgresUserRepository(pool)
	incidentRepo := NewPostgresIncidentRepository(pool)
	commentRepo := NewPostgresCommentRepository(pool)
	ctx := context.Background()

	// 1. Seed User & Incident
	userAlice, err := userRepo.CreateOrUpdate(ctx, models.CreateUserParams{
		Email:        "alice@example.com",
		Name:         "Alice",
		AuthProvider: "google",
	})
	require.NoError(t, err)

	userBob, err := userRepo.CreateOrUpdate(ctx, models.CreateUserParams{
		Email:        "bob@example.com",
		Name:         "Bob",
		AuthProvider: "google",
	})
	require.NoError(t, err)

	inc, err := incidentRepo.Create(ctx, models.CreateIncidentParams{
		Title:         "Comment Section Incident",
		FullStory:     "Story thread for comments test.",
		Severity:      5,
		JusticeStatus: models.JusticeProceeding,
		State:         "Delhi",
		City:          "New Delhi",
		CreatedBy:     &userAlice.ID,
	})
	require.NoError(t, err)

	t.Run("Create top-level comment and nested replies, then verify tree building", func(t *testing.T) {
		// Post Top-Level Comment 1 by Alice
		rootComment1, err := commentRepo.CreateComment(ctx, inc.ID, userAlice.ID, nil, "This incident needs more verifiable evidence.")
		require.NoError(t, err)
		assert.Nil(t, rootComment1.ParentID)

		// Post Reply to Comment 1 by Bob
		reply1, err := commentRepo.CreateComment(ctx, inc.ID, userBob.ID, &rootComment1.ID, "I uploaded news archives in the assets section.")
		require.NoError(t, err)
		require.NotNil(t, reply1.ParentID)
		assert.Equal(t, rootComment1.ID, *reply1.ParentID)

		// Post Nested Reply to Bob's comment by Alice
		nestedReply, err := commentRepo.CreateComment(ctx, inc.ID, userAlice.ID, &reply1.ID, "Thanks Bob, checking them now!")
		require.NoError(t, err)
		require.NotNil(t, nestedReply.ParentID)
		assert.Equal(t, reply1.ID, *nestedReply.ParentID)

		// Post Top-Level Comment 2 by Bob
		rootComment2, err := commentRepo.CreateComment(ctx, inc.ID, userBob.ID, nil, "Following this thread closely.")
		require.NoError(t, err)

		// Fetch threaded comments tree via ListCommentsByIncident
		tree, err := commentRepo.ListCommentsByIncident(ctx, inc.ID, 20, 0)
		require.NoError(t, err)

		// Assert Top-Level Root Array length
		require.Len(t, tree, 2, "There should be exactly 2 top-level root comments")

		// Root 1 assertions
		assert.Equal(t, rootComment1.ID, tree[0].ID)
		assert.Equal(t, "Alice", tree[0].UserName)
		require.Len(t, tree[0].Replies, 1, "Root comment 1 should have 1 reply")

		// Child 1 (Bob's reply) assertions
		bobReply := tree[0].Replies[0]
		assert.Equal(t, reply1.ID, bobReply.ID)
		assert.Equal(t, "Bob", bobReply.UserName)
		require.Len(t, bobReply.Replies, 1, "Bob's reply should have 1 nested reply")

		// Grandchild (Alice's nested reply) assertions
		aliceNestedReply := bobReply.Replies[0]
		assert.Equal(t, nestedReply.ID, aliceNestedReply.ID)
		assert.Equal(t, "Alice", aliceNestedReply.UserName)

		// Root 2 assertions
		assert.Equal(t, rootComment2.ID, tree[1].ID)
		assert.Equal(t, "Bob", tree[1].UserName)
		assert.Empty(t, tree[1].Replies)
	})

	t.Run("ListCommentsByIncident returns empty slice for incident with no comments", func(t *testing.T) {
		emptyInc, err := incidentRepo.Create(ctx, models.CreateIncidentParams{
			Title:         "Empty Comments Incident",
			FullStory:     "No comments here.",
			Severity:      1,
			JusticeStatus: models.JusticeProceeding,
			State:         "Goa",
			City:          "Panaji",
		})
		require.NoError(t, err)

		tree, err := commentRepo.ListCommentsByIncident(ctx, emptyInc.ID, 10, 0)
		require.NoError(t, err)
		assert.Empty(t, tree)
	})
}
