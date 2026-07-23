package db

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

func TestCommentsAndTargets(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://app_user:app_password@localhost:5432/injusticedb?sslmode=disable"
	}

	ctx := context.Background()

	pool, err := InitDB(ctx, dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer pool.Close()

	commentRepo := NewPostgresCommentRepository(pool)
	targetRepo := NewPostgresTargetRepository(pool)
	incidentRepo := NewPostgresIncidentRepository(pool)
	userRepo := NewPostgresUserRepository(pool)

	// --- 1. Test Comment Functionality ---
	uID := uuid.New()
	user, err := userRepo.CreateOrUpdate(ctx, models.CreateUserParams{
		ID:           uID,
		Email:        fmt.Sprintf("commenter_%s@example.com", uID),
		Name:         "Commenter User",
		AuthProvider: "google",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	incident, err := incidentRepo.Create(ctx, models.CreateIncidentParams{
		Title:     "Comment Test Incident",
		FullStory: "Testing discussions on incidents",
		Severity:  4,
		State:     "Jharkhand",
		City:      "Ranchi",
	})
	if err != nil {
		t.Fatalf("Failed to create incident: %v", err)
	}

	comment, err := commentRepo.CreateComment(ctx, incident.ID, user.ID, "This is a critical update.")
	if err != nil {
		t.Fatalf("Failed to post comment: %v", err)
	}

	if comment.Content != "This is a critical update." {
		t.Errorf("Expected content 'This is a critical update.', got '%s'", comment.Content)
	}

	comments, err := commentRepo.ListCommentsByIncident(ctx, incident.ID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list comments: %v", err)
	}

	if len(comments) != 1 {
		t.Fatalf("Expected 1 comment, got %d", len(comments))
	}

	if comments[0].UserName != "Commenter User" {
		t.Errorf("Expected user name 'Commenter User', got '%s'", comments[0].UserName)
	}

	if comments[0].Content != "This is a critical update." {
		t.Errorf("Expected comment content match, got '%s'", comments[0].Content)
	}

	// --- 2. Test Target Functionality ---
	occ := "Official"
	target, err := targetRepo.CreateTarget(ctx, models.YDCIDCTarget{
		Name:              "Target Alpha",
		Occupation:        &occ,
		CauseOfResentment: "Unlawful systemic obstruction",
	})
	if err != nil {
		t.Fatalf("Failed to create target: %v", err)
	}

	targets, err := targetRepo.ListTargets(ctx, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list targets: %v", err)
	}

	if len(targets) == 0 {
		t.Fatalf("Expected at least 1 target entry")
	}

	if target.Name != "Target Alpha" {
		t.Errorf("Expected target name 'Target Alpha', got '%s'", target.Name)
	}
}
