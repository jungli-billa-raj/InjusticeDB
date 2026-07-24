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

// setupTestDB initializes a pool and cleans up the users table before running tests.
func setupUserTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/injusticedb?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err, "failed to connect to test database")

	// Clean users table before test execution
	_, err = pool.Exec(ctx, "TRUNCATE TABLE users CASCADE;")
	require.NoError(t, err, "failed to truncate users table")

	teardown := func() {
		pool.Close()
	}

	return pool, teardown
}

func TestPostgresUserRepository_CreateOrUpdate(t *testing.T) {
	pool, teardown := setupUserTestDB(t)
	defer teardown()

	repo := NewPostgresUserRepository(pool)
	ctx := context.Background()

	t.Run("Create new user successfully", func(t *testing.T) {
		params := models.CreateUserParams{
			Email:        "alice@example.com",
			Name:         "Alice Dev",
			AuthProvider: "google",
		}

		user, err := repo.CreateOrUpdate(ctx, params)
		require.NoError(t, err)
		require.NotNil(t, user)

		assert.NotEqual(t, uuid.Nil, user.ID)
		assert.Equal(t, params.Email, user.Email)
		assert.Equal(t, params.Name, user.Name)
		assert.Equal(t, params.AuthProvider, user.AuthProvider)
		assert.Equal(t, models.RoleUser, user.Role)
		assert.Equal(t, 100, user.CredibilityScore)
		assert.False(t, user.CreatedAt.IsZero())
	})

	t.Run("Upsert updates existing user profile on re-login", func(t *testing.T) {
		initialParams := models.CreateUserParams{
			Email:        "bob@example.com",
			Name:         "Bob Initial",
			AuthProvider: "google",
		}

		createdUser, err := repo.CreateOrUpdate(ctx, initialParams)
		require.NoError(t, err)

		updateParams := models.CreateUserParams{
			Email:        "bob@example.com", // Same email
			Name:         "Bob Updated",
			AuthProvider: "github",
		}

		updatedUser, err := repo.CreateOrUpdate(ctx, updateParams)
		require.NoError(t, err)

		assert.Equal(t, createdUser.ID, updatedUser.ID, "UUID must remain unchanged")
		assert.Equal(t, "Bob Updated", updatedUser.Name)
		assert.Equal(t, "github", updatedUser.AuthProvider)
	})
}

func TestPostgresUserRepository_GetByID(t *testing.T) {
	pool, teardown := setupUserTestDB(t)
	defer teardown()

	repo := NewPostgresUserRepository(pool)
	ctx := context.Background()

	t.Run("Fetch existing user by ID", func(t *testing.T) {
		created, err := repo.CreateOrUpdate(ctx, models.CreateUserParams{
			Email:        "charlie@example.com",
			Name:         "Charlie",
			AuthProvider: "email",
		})
		require.NoError(t, err)

		fetched, err := repo.GetByID(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, fetched.ID)
		assert.Equal(t, created.Email, fetched.Email)
		assert.Equal(t, created.Name, fetched.Name)
	})

	t.Run("Return error when user does not exist", func(t *testing.T) {
		nonExistentID := uuid.New()
		user, err := repo.GetByID(ctx, nonExistentID)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "user not found")
	})
}

func TestPostgresUserRepository_UpdateCredibility(t *testing.T) {
	pool, teardown := setupUserTestDB(t)
	defer teardown()

	repo := NewPostgresUserRepository(pool)
	ctx := context.Background()

	user, err := repo.CreateOrUpdate(ctx, models.CreateUserParams{
		Email:        "david@example.com",
		Name:         "David",
		AuthProvider: "google",
	})
	require.NoError(t, err)
	assert.Equal(t, 100, user.CredibilityScore)

	t.Run("Increase credibility with positive delta", func(t *testing.T) {
		err := repo.UpdateCredibility(ctx, user.ID, 25)
		require.NoError(t, err)

		updated, err := repo.GetByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, 125, updated.CredibilityScore)
	})

	t.Run("Decrease credibility with negative delta", func(t *testing.T) {
		err := repo.UpdateCredibility(ctx, user.ID, -50)
		require.NoError(t, err)

		updated, err := repo.GetByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, 75, updated.CredibilityScore)
	})

	t.Run("Ensure credibility score does not drop below 0", func(t *testing.T) {
		err := repo.UpdateCredibility(ctx, user.ID, -500)
		require.NoError(t, err)

		updated, err := repo.GetByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, updated.CredibilityScore)
	})

	t.Run("Return error for non-existent user ID", func(t *testing.T) {
		err := repo.UpdateCredibility(ctx, uuid.New(), 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})
}
