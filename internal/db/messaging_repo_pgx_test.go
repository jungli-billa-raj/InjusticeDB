package db

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

func TestMessagingAndRLSEnforcement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dbURL := os.Getenv("NON_SU_DB_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/injusticedb?sslmode=disable"
	}

	ctx := context.Background()

	pool, err := InitDB(ctx, dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer pool.Close()

	msgRepo := NewPostgresMessagingRepository(pool)
	userRepo := NewPostgresUserRepository(pool)

	// 1. Create 3 Test Users (User A, User B, and User C [Unrelated third party])
	uAID, uBID, uCID := uuid.New(), uuid.New(), uuid.New()

	users := []models.CreateUserParams{
		{ID: uAID, Email: fmt.Sprintf("usera_%s@example.com", uAID), Name: "User A", AuthProvider: "google"},
		{ID: uBID, Email: fmt.Sprintf("userb_%s@example.com", uBID), Name: "User B", AuthProvider: "google"},
		{ID: uCID, Email: fmt.Sprintf("userc_%s@example.com", uCID), Name: "User C", AuthProvider: "google"},
	}

	for _, u := range users {
		_, err := userRepo.CreateOrUpdate(ctx, u)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
	}

	// 2. User A opens a conversation with User B
	conv, err := msgRepo.GetOrCreateConversation(ctx, uAID, uBID)
	if err != nil {
		t.Fatalf("Failed to create conversation: %v", err)
	}

	// 3. User A sends a message to User B
	sentMsg, err := msgRepo.SendMessage(ctx, conv.ID, uAID, "Hey B, let's talk privately.")
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	if sentMsg.Content != "Hey B, let's talk privately." {
		t.Errorf("Unexpected message content: %s", sentMsg.Content)
	}

	// 4. User B reads messages (Should succeed)
	bMessages, err := msgRepo.GetMessages(ctx, uBID, conv.ID, 10, 0)
	if err != nil {
		t.Fatalf("User B failed to read messages: %v", err)
	}

	if len(bMessages) != 1 {
		t.Fatalf("Expected 1 message for User B, got %d", len(bMessages))
	}

	// 5. TEST RLS: User C tries to read messages from User A and B's conversation (Should return 0 messages!)
	cMessages, err := msgRepo.GetMessages(ctx, uCID, conv.ID, 10, 0)
	if err != nil {
		t.Fatalf("Query failed for User C: %v", err)
	}

	if len(cMessages) != 0 {
		t.Errorf("SECURITY RISK: User C was able to read private messages of User A and B! RLS policy failed.")
	}
}
