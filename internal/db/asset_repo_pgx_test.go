package db

import (
	"context"
	"os"
	"testing"

	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

func TestAssetLifecycleAndSoftDelete(t *testing.T) {
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

	assetRepo := NewPostgresAssetRepository(pool)
	incidentRepo := NewPostgresIncidentRepository(pool)

	// 1. Setup Parent Incident
	incident, err := incidentRepo.Create(ctx, models.CreateIncidentParams{
		Title:     "Asset Test Incident",
		FullStory: "Testing evidence asset attachment and soft deletion",
		Severity:  5,
		State:     "Jharkhand",
		City:      "Ranchi",
	})
	if err != nil {
		t.Fatalf("Failed to create test incident: %v", err)
	}

	archive := "https://web.archive.org/web/test"
	testAssets := []models.Asset{
		{
			IncidentID: incident.ID,
			Type:       models.AssetImage,
			URL:        "https://storage.example.com/photo1.jpg",
			ArchiveURL: &archive,
		},
		{
			IncidentID: incident.ID,
			Type:       models.AssetArticle,
			URL:        "https://news.example.com/article1",
		},
	}

	// 2. Test Batch Addition
	err = assetRepo.AddAssets(ctx, testAssets)
	if err != nil {
		t.Fatalf("Failed to batch insert assets: %v", err)
	}

	// 3. Test Retrieval
	retrieved, err := assetRepo.GetByIncidentID(ctx, incident.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve assets: %v", err)
	}

	if len(retrieved) != 2 {
		t.Fatalf("Expected 2 assets, got %d", len(retrieved))
	}

	targetAsset := retrieved[0]

	// 4. Test Soft Delete
	err = assetRepo.SoftDeleteAsset(ctx, targetAsset.ID)
	if err != nil {
		t.Fatalf("Failed to soft delete asset: %v", err)
	}

	// 5. Verify Soft-Deleted Asset is Filtered Out
	retrievedAfterDelete, err := assetRepo.GetByIncidentID(ctx, incident.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve assets after soft delete: %v", err)
	}

	if len(retrievedAfterDelete) != 1 {
		t.Fatalf("Expected 1 active asset remaining, got %d", len(retrievedAfterDelete))
	}

	// 6. Test Restore
	err = assetRepo.RestoreAsset(ctx, targetAsset.ID)
	if err != nil {
		t.Fatalf("Failed to restore asset: %v", err)
	}

	retrievedAfterRestore, err := assetRepo.GetByIncidentID(ctx, incident.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve assets after restore: %v", err)
	}

	if len(retrievedAfterRestore) != 2 {
		t.Fatalf("Expected 2 active assets after restore, got %d", len(retrievedAfterRestore))
	}
}
