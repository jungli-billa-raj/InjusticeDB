package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

func setupTestDB(t *testing.T) *PostgresIncidentRepository {
	t.Helper()

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/injusticedb?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := InitDB(ctx, dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return NewPostgresIncidentRepository(pool)
}

func TestIncidentLifecycleAndVersioning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	repo := setupTestDB(t)
	ctx := context.Background()

	// 1. Test Create Incident
	created, err := repo.Create(ctx, models.CreateIncidentParams{
		Title:     "Integration Test Incident",
		FullStory: "Testing full story for version 1",
		Severity:  5,
		State:     "Jharkhand",
		City:      "Ranchi",
	})
	if err != nil {
		t.Fatalf("Failed to create incident: %v", err)
	}

	if created.CurrentVersion != 1 {
		t.Errorf("Expected current_version 1, got %d", created.CurrentVersion)
	}

	// 2. Test Create Revision (Version 2)
	rev2, err := repo.CreateRevision(ctx, models.IncidentRevision{
		IncidentID:    created.ID,
		Title:         "Integration Test Incident (v2)",
		FullStory:     "Testing full story for version 2",
		ChangeSummary: "Added news citation",
	})
	if err != nil {
		t.Fatalf("Failed to create revision: %v", err)
	}

	if rev2.VersionNumber != 2 {
		t.Errorf("Expected revision version 2, got %d", rev2.VersionNumber)
	}

	// 3. Test Fetching Historical Version 1
	rev1, err := repo.GetRevision(ctx, created.ID, 1)
	if err != nil {
		t.Fatalf("Failed to fetch historical version 1: %v", err)
	}

	if rev1.Title != "Integration Test Incident" {
		t.Errorf("Expected v1 title 'Integration Test Incident', got '%s'", rev1.Title)
	}
}
