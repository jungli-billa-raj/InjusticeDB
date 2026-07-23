package db

import (
	"context"
	"os"
	"testing"

	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

func TestCulpritLifecycleAndLinking(t *testing.T) {
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

	culpritRepo := NewPostgresCulpritRepository(pool)
	incidentRepo := NewPostgresIncidentRepository(pool)

	// 1. Create a Test Incident
	incident, err := incidentRepo.Create(ctx, models.CreateIncidentParams{
		Title:     "Culprit Link Test Incident",
		FullStory: "Testing linking culprits to an incident",
		Severity:  8,
		State:     "Jharkhand",
		City:      "Ranchi",
	})
	if err != nil {
		t.Fatalf("Failed to create incident: %v", err)
	}

	// 2. Create a Person (Suspect)
	org := "Test Corp"
	age := 42
	person, err := culpritRepo.CreatePerson(ctx, models.Person{
		Name:         "Jane Doe",
		Organization: &org,
		Age:          &age,
	})
	if err != nil {
		t.Fatalf("Failed to create person: %v", err)
	}

	// 3. Link Person to Incident as 'suspect'
	err = culpritRepo.LinkToIncident(ctx, incident.ID, person.ID, models.CulpritSuspect)
	if err != nil {
		t.Fatalf("Failed to link culprit to incident: %v", err)
	}

	// 4. Retrieve Culprits for Incident
	culprits, err := culpritRepo.GetCulpritsForIncident(ctx, incident.ID)
	if err != nil {
		t.Fatalf("Failed to fetch culprits: %v", err)
	}

	if len(culprits) != 1 {
		t.Fatalf("Expected 1 culprit, got %d", len(culprits))
	}

	if culprits[0].Person.Name != "Jane Doe" {
		t.Errorf("Expected person name 'Jane Doe', got '%s'", culprits[0].Person.Name)
	}

	if culprits[0].CulpritStatus != models.CulpritSuspect {
		t.Errorf("Expected status 'suspect', got '%s'", culprits[0].CulpritStatus)
	}

	// 5. Update Status to 'convicted'
	err = culpritRepo.UpdateCulpritStatus(ctx, incident.ID, person.ID, models.CulpritConvicted)
	if err != nil {
		t.Fatalf("Failed to update status to convicted: %v", err)
	}

	updatedCulprits, err := culpritRepo.GetCulpritsForIncident(ctx, incident.ID)
	if err != nil {
		t.Fatalf("Failed to fetch updated culprits: %v", err)
	}

	if updatedCulprits[0].CulpritStatus != models.CulpritConvicted {
		t.Errorf("Expected status 'convicted', got '%s'", updatedCulprits[0].CulpritStatus)
	}
}
