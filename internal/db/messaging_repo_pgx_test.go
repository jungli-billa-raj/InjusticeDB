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

// setupMessagingTestDB prepares a pool and cleans up tables for isolation.
func setupMessagingTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/injusticedb?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err, "failed to connect to test database")

	_, err = pool.Exec(ctx, "TRUNCATE TABLE users, conversations, messages CASCADE;")
	require.NoError(t, err, "failed to truncate tables")

	teardown := func() {
		pool.Close()
	}

	return pool, teardown
}

func TestPostgresMessagingRepository_GetOrCreateConversation(t *testing.T) {
	pool, teardown := setupMessagingTestDB(t)
	defer teardown()

	userRepo := NewPostgresUserRepository(pool)
	msgRepo := NewPostgresMessagingRepository(pool)
	ctx := context.Background()

	// Seed User 1 & User 2
	u1, err := userRepo.CreateOrUpdate(ctx, models.CreateUserParams{
		Email:        "user1@example.com",
		Name:         "User One",
		AuthProvider: "google",
	})
	require.NoError(t, err)

	u2, err := userRepo.CreateOrUpdate(ctx, models.CreateUserParams{
		Email:        "user2@example.com",
		Name:         "User Two",
		AuthProvider: "google",
	})
	require.NoError(t, err)

	t.Run("Create conversation respects canonical UUID order", func(t *testing.T) {
		conv, err := msgRepo.GetOrCreateConversation(ctx, u1.ID, u2.ID)
		require.NoError(t, err)
		require.NotNil(t, conv)

		assert.NotEqual(t, uuid.Nil, conv.ID)

		// Assert user_one_id < user_two_id string comparison
		if u1.ID.String() < u2.ID.String() {
			assert.Equal(t, u1.ID, conv.UserOneID)
			assert.Equal(t, u2.ID, conv.UserTwoID)
		} else {
			assert.Equal(t, u2.ID, conv.UserOneID)
			assert.Equal(t, u1.ID, conv.UserTwoID)
		}
	})

	t.Run("GetOrCreateConversation is idempotent (returns existing ID)", func(t *testing.T) {
		conv1, err := msgRepo.GetOrCreateConversation(ctx, u1.ID, u2.ID)
		require.NoError(t, err)

		// Call with swapped user argument positions
		conv2, err := msgRepo.GetOrCreateConversation(ctx, u2.ID, u1.ID)
		require.NoError(t, err)

		assert.Equal(t, conv1.ID, conv2.ID, "Must return the exact same conversation ID")
	})
}

func TestPostgresMessagingRepository_SendMessageAndGetMessages(t *testing.T) {
	pool, teardown := setupMessagingTestDB(t)
	defer teardown()

	userRepo := NewPostgresUserRepository(pool)
	msgRepo := NewPostgresMessagingRepository(pool)
	ctx := context.Background()

	u1, err := userRepo.CreateOrUpdate(ctx, models.CreateUserParams{
		Email:        "sender@example.com",
		Name:         "Sender",
		AuthProvider: "google",
	})
	require.NoError(t, err)

	u2, err := userRepo.CreateOrUpdate(ctx, models.CreateUserParams{
		Email:        "receiver@example.com",
		Name:         "Receiver",
		AuthProvider: "google",
	})
	require.NoError(t, err)

	u3, err := userRepo.CreateOrUpdate(ctx, models.CreateUserParams{
		Email:        "eavesdropper@example.com",
		Name:         "Eavesdropper",
		AuthProvider: "google",
	})
	require.NoError(t, err)

	conv, err := msgRepo.GetOrCreateConversation(ctx, u1.ID, u2.ID)
	require.NoError(t, err)

	t.Run("Send message successfully as conversation participant", func(t *testing.T) {
		msg1, err := msgRepo.SendMessage(ctx, conv.ID, u1.ID, "Hello Receiver!")
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, msg1.ID)
		assert.Equal(t, conv.ID, msg1.ConversationID)
		assert.Equal(t, u1.ID, msg1.SenderID)
		assert.Equal(t, "Hello Receiver!", msg1.Content)
		assert.False(t, msg1.IsRead)

		msg2, err := msgRepo.SendMessage(ctx, conv.ID, u2.ID, "Hey Sender!")
		require.NoError(t, err)
		assert.Equal(t, "Hey Sender!", msg2.Content)
	})

	t.Run("GetMessages returns messages in chronological order for participant", func(t *testing.T) {
		messages, err := msgRepo.GetMessages(ctx, u1.ID, conv.ID, 10, 0)
		require.NoError(t, err)
		require.Len(t, messages, 2)

		assert.Equal(t, "Hello Receiver!", messages[0].Content)
		assert.Equal(t, "Hey Sender!", messages[1].Content)
	})

	t.Run("RLS blocks third-party non-participant from reading messages", func(t *testing.T) {
		messages, err := msgRepo.GetMessages(ctx, u3.ID, conv.ID, 10, 0)
		require.NoError(t, err)
		assert.Empty(t, messages, "Non-participant user should see zero messages due to RLS policy")
	})
}

func TestPostgresMessagingRepository_ListConversations(t *testing.T) {
	pool, teardown := setupMessagingTestDB(t)
	defer teardown()

	userRepo := NewPostgresUserRepository(pool)
	msgRepo := NewPostgresMessagingRepository(pool)
	ctx := context.Background()

	u1, err := userRepo.CreateOrUpdate(ctx, models.CreateUserParams{
		Email:        "main@example.com",
		Name:         "Main User",
		AuthProvider: "google",
	})
	require.NoError(t, err)

	u2, err := userRepo.CreateOrUpdate(ctx, models.CreateUserParams{
		Email:        "contact1@example.com",
		Name:         "Contact One",
		AuthProvider: "google",
	})
	require.NoError(t, err)

	u3, err := userRepo.CreateOrUpdate(ctx, models.CreateUserParams{
		Email:        "contact2@example.com",
		Name:         "Contact Two",
		AuthProvider: "google",
	})
	require.NoError(t, err)

	_, err = msgRepo.GetOrCreateConversation(ctx, u1.ID, u2.ID)
	require.NoError(t, err)

	_, err = msgRepo.GetOrCreateConversation(ctx, u1.ID, u3.ID)
	require.NoError(t, err)

	t.Run("ListConversations fetches active chat spaces for user", func(t *testing.T) {
		convs, err := msgRepo.ListConversations(ctx, u1.ID)
		require.NoError(t, err)
		assert.Len(t, convs, 2)
	})

	t.Run("ListConversations returns empty list for user with no conversations", func(t *testing.T) {
		lonelyUser, err := userRepo.CreateOrUpdate(ctx, models.CreateUserParams{
			Email:        "lonely@example.com",
			Name:         "Lonely User",
			AuthProvider: "google",
		})
		require.NoError(t, err)

		convs, err := msgRepo.ListConversations(ctx, lonelyUser.ID)
		require.NoError(t, err)
		assert.Empty(t, convs)
	})
}
