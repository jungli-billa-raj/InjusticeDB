package db

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

func TestVerificationVotingAndTallies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/injusticedb?sslmode=disable"
	}

	ctx := context.Background()

	pool, err := InitDB(ctx, dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer pool.Close()

	verificationRepo := NewPostgresVerificationRepository(pool)
	incidentRepo := NewPostgresIncidentRepository(pool)
	userRepo := NewPostgresUserRepository(pool)

	// 1. Create a Test Incident
	incident, err := incidentRepo.Create(ctx, models.CreateIncidentParams{
		Title:     "Verification Test Incident",
		FullStory: "Testing voting system and tallies",
		Severity:  5,
		State:     "Jharkhand",
		City:      "Ranchi",
	})
	if err != nil {
		t.Fatalf("Failed to create incident: %v", err)
	}

	// 2. Create 3 Test Users
	u1ID, u2ID, u3ID := uuid.New(), uuid.New(), uuid.New()

	users := []models.CreateUserParams{
		{ID: u1ID, Email: fmt.Sprintf("voter1_%s@example.com", u1ID), Name: "Voter One", AuthProvider: "google"},
		{ID: u2ID, Email: fmt.Sprintf("voter2_%s@example.com", u2ID), Name: "Voter Two", AuthProvider: "google"},
		{ID: u3ID, Email: fmt.Sprintf("voter3_%s@example.com", u3ID), Name: "Voter Three", AuthProvider: "google"},
	}

	for _, u := range users {
		_, err := userRepo.CreateOrUpdate(ctx, u)
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}
	}

	// 3. Cast Votes (2 Verifies, 1 Reject)
	err = verificationRepo.CastVote(ctx, incident.ID, u1ID, "verify")
	if err != nil {
		t.Fatalf("Failed user 1 vote: %v", err)
	}

	err = verificationRepo.CastVote(ctx, incident.ID, u2ID, "verify")
	if err != nil {
		t.Fatalf("Failed user 2 vote: %v", err)
	}

	err = verificationRepo.CastVote(ctx, incident.ID, u3ID, "reject")
	if err != nil {
		t.Fatalf("Failed user 3 vote: %v", err)
	}

	// 4. Verify Tally Count
	vCount, rCount, err := verificationRepo.GetVoteTally(ctx, incident.ID)
	if err != nil {
		t.Fatalf("Failed to get vote tally: %v", err)
	}

	if vCount != 2 {
		t.Errorf("Expected 2 verify votes, got %d", vCount)
	}

	if rCount != 1 {
		t.Errorf("Expected 1 reject vote, got %d", rCount)
	}

	// 5. User 3 changes their mind (updates vote from 'reject' to 'verify')
	err = verificationRepo.CastVote(ctx, incident.ID, u3ID, "verify")
	if err != nil {
		t.Fatalf("Failed user 3 vote update: %v", err)
	}

	vCountUpdated, rCountUpdated, err := verificationRepo.GetVoteTally(ctx, incident.ID)
	if err != nil {
		t.Fatalf("Failed to get updated vote tally: %v", err)
	}

	if vCountUpdated != 3 || rCountUpdated != 0 {
		t.Errorf("Expected 3 verifies and 0 rejects after update, got %d verify, %d reject", vCountUpdated, rCountUpdated)
	}
}
