package db

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

func TestUserLifecycleAndCredibility(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/injusticedb?sslmode=disable"
	}

	ctx := context.Background()

	pool, err := InitDB(ctx, dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresUserRepository(pool)

	testID := uuid.New()
	// Dynamic email ensures no row collisions between test runs
	testEmail := fmt.Sprintf("testuser_%s@example.com", testID.String())

	// 1. Create or Update (Insert new user)
	created, err := repo.CreateOrUpdate(ctx, models.CreateUserParams{
		ID:           testID,
		Email:        testEmail,
		Name:         "Test Citizen",
		AuthProvider: "google",
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if created.CredibilityScore != 100 {
		t.Errorf("Expected default credibility score of 100, got %d", created.CredibilityScore)
	}

	// 2. Fetch User By ID
	fetched, err := repo.GetByID(ctx, testID)
	if err != nil {
		t.Fatalf("Failed to fetch user: %v", err)
	}

	if fetched.Name != "Test Citizen" {
		t.Errorf("Expected name 'Test Citizen', got '%s'", fetched.Name)
	}

	// 3. Reward Credibility (Increase by +25)
	err = repo.UpdateCredibility(ctx, testID, 25)
	if err != nil {
		t.Fatalf("Failed to boost credibility: %v", err)
	}

	updated, err := repo.GetByID(ctx, testID)
	if err != nil {
		t.Fatalf("Failed to fetch updated user: %v", err)
	}

	if updated.CredibilityScore != 125 {
		t.Errorf("Expected credibility score of 125, got %d", updated.CredibilityScore)
	}
}
