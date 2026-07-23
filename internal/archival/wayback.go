package archival

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jungli-billa-raj/InjusticeDB/internal/db"
)

// Archiver defines the contract for taking web snapshots.
type Archiver interface {
	RequestSnapshotAsync(assetID uuid.UUID, targetURL string)
}

// Real Wayback Archiver (Production)

type WaybackArchiver struct {
	assetRepo db.AssetRepository
	client    *http.Client
}

func NewWaybackArchiver(assetRepo db.AssetRepository) *WaybackArchiver {
	return &WaybackArchiver{
		assetRepo: assetRepo,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (w *WaybackArchiver) RequestSnapshotAsync(assetID uuid.UUID, targetURL string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		archiveURL, err := w.triggerSnapshot(ctx, targetURL)
		if err != nil {
			log.Printf("[Wayback Error] Failed to archive %s for asset %s: %v", targetURL, assetID, err)
			return
		}

		err = w.assetRepo.UpdateArchiveURL(ctx, assetID, archiveURL)
		if err != nil {
			log.Printf("[Wayback Error] Failed to save archive_url to DB for asset %s: %v", assetID, err)
		}
	}()
}

func (w *WaybackArchiver) triggerSnapshot(ctx context.Context, targetURL string) (string, error) {
	apiURL := fmt.Sprintf("https://web.archive.org/save/%s", targetURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "InjusticeDB-Archiver/1.0")

	resp, err := w.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("wayback API returned status: %d", resp.StatusCode)
	}

	location := resp.Header.Get("Content-Location")
	if location != "" {
		return fmt.Sprintf("https://web.archive.org%s", location), nil
	}

	var wbResp struct {
		WaybackURL string `json:"wayback_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wbResp); err == nil && wbResp.WaybackURL != "" {
		return wbResp.WaybackURL, nil
	}

	return fmt.Sprintf("https://web.archive.org/web/%s/%s", time.Now().Format("20060102150405"), targetURL), nil
}

// Dry-Run Archiver (Local Dev / Offline Testing)

type NopArchiver struct{}

func NewNopArchiver() *NopArchiver {
	return &NopArchiver{}
}

func (n *NopArchiver) RequestSnapshotAsync(assetID uuid.UUID, targetURL string) {
	log.Printf("[DRY-RUN ARCHIVER] Skipping real Wayback API call for URL: %s", targetURL)
}
