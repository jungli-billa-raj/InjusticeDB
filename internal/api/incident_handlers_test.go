package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

func TestIncidentEndpoints_CombinedFlow(t *testing.T) {
	ts, repos, teardown := setupAPITestServer(t)
	defer teardown()

	ctx := t.Context()

	// Seed Test User
	creator, err := repos.Users.CreateOrUpdate(ctx, models.CreateUserParams{
		Email:        "reporter@example.com",
		Name:         "Reporter User",
		AuthProvider: "google",
	})
	require.NoError(t, err)

	userToken := generateTestJWT(creator.ID, "user")
	adminToken := generateTestJWT(creator.ID, "admin")

	var createdIncidentID uuid.UUID

	t.Run("1. POST /api/v1/incidents creates master & v1 revision", func(t *testing.T) {
		payload := models.CreateIncidentParams{
			Title:         "Illegal Sand Mining Incident",
			FullStory:     "Unregulated extraction observed along riverbed without permits.",
			Severity:      8,
			JusticeStatus: models.JusticeProceeding,
			State:         "Jharkhand",
			City:          "Dhanbad",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/incidents", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+userToken)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var incident models.Incident
		err = json.NewDecoder(resp.Body).Decode(&incident)
		require.NoError(t, err)

		assert.NotEqual(t, uuid.Nil, incident.ID)
		assert.Equal(t, models.VerificationPending, incident.VerificationStatus)
		assert.Equal(t, 1, incident.CurrentVersion)

		createdIncidentID = incident.ID
	})

	t.Run("2. GET /api/v1/incidents/{id} fetches full snapshot publicly", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/incidents/" + createdIncidentID.String())
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var fullIncident models.FullLatestIncident
		err = json.NewDecoder(resp.Body).Decode(&fullIncident)
		require.NoError(t, err)

		assert.Equal(t, createdIncidentID, fullIncident.IncidentID)
		assert.Equal(t, "Illegal Sand Mining Incident", fullIncident.Title)
		assert.Equal(t, "Jharkhand", fullIncident.State)
		assert.Equal(t, 1, fullIncident.VersionNumber)
	})

	t.Run("3. PATCH /api/v1/incidents/{id}/verification updates status with RBAC", func(t *testing.T) {
		payload := UpdateVerificationStatusRequest{
			Status: models.VerificationVerified,
		}
		body, _ := json.Marshal(payload)

		// 3a. Standard user attempt should be forbidden (403)
		reqUser, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/incidents/"+createdIncidentID.String()+"/verification", bytes.NewBuffer(body))
		reqUser.Header.Set("Authorization", "Bearer "+userToken)
		reqUser.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		respUser, err := client.Do(reqUser)
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, respUser.StatusCode)
		respUser.Body.Close()

		// 3b. Admin attempt should succeed (200)
		reqAdmin, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/incidents/"+createdIncidentID.String()+"/verification", bytes.NewBuffer(body))
		reqAdmin.Header.Set("Authorization", "Bearer "+adminToken)
		reqAdmin.Header.Set("Content-Type", "application/json")

		respAdmin, err := client.Do(reqAdmin)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, respAdmin.StatusCode)
		respAdmin.Body.Close()

		// 3c. Verify status changed to 'verified'
		respVerify, err := http.Get(ts.URL + "/api/v1/incidents/" + createdIncidentID.String())
		require.NoError(t, err)
		defer respVerify.Body.Close()

		var updatedIncident models.FullLatestIncident
		_ = json.NewDecoder(respVerify.Body).Decode(&updatedIncident)
		assert.Equal(t, models.VerificationVerified, updatedIncident.VerificationStatus)
	})
}

func TestIncidentRevisionEndpoints(t *testing.T) {
	ts, repos, teardown := setupAPITestServer(t)
	defer teardown()

	ctx := t.Context()

	// Seed Editor User
	editor, err := repos.Users.CreateOrUpdate(ctx, models.CreateUserParams{
		Email:        "editor@example.com",
		Name:         "Editor User",
		AuthProvider: "google",
	})
	require.NoError(t, err)

	editorToken := generateTestJWT(editor.ID, "user")

	// Create initial incident (Version 1)
	inc, err := repos.Incidents.Create(ctx, models.CreateIncidentParams{
		Title:         "Initial Report Title",
		FullStory:     "Initial report full story text.",
		Severity:      5,
		JusticeStatus: models.JusticeProceeding,
		State:         "Jharkhand",
		City:          "Ranchi",
		CreatedBy:     &editor.ID,
	})
	require.NoError(t, err)

	t.Run("1. POST /api/v1/incidents/{id}/revisions bumps version to 2", func(t *testing.T) {
		payload := CreateRevisionRequest{
			Title:         "Updated Report Title v2",
			FullStory:     "Updated story with additional witness details.",
			Severity:      7,
			JusticeStatus: models.JusticeProceeding,
			State:         "Jharkhand",
			City:          "Ranchi",
			ChangeSummary: "Added witness testimonies and updated severity.",
		}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/incidents/"+inc.ID.String()+"/revisions", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+editorToken)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var revision models.IncidentRevision
		err = json.NewDecoder(resp.Body).Decode(&revision)
		require.NoError(t, err)

		assert.Equal(t, inc.ID, revision.IncidentID)
		assert.Equal(t, 2, revision.VersionNumber)
		assert.Equal(t, "Updated Report Title v2", revision.Title)
		assert.Equal(t, "Added witness testimonies and updated severity.", revision.ChangeSummary)
	})

	t.Run("2. GET /api/v1/incidents/{id}/revisions lists all versions", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/incidents/" + inc.ID.String() + "/revisions")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var revisions []models.IncidentRevision
		err = json.NewDecoder(resp.Body).Decode(&revisions)
		require.NoError(t, err)

		assert.Len(t, revisions, 2)
	})

	t.Run("3. GET /api/v1/incidents/{id}/revisions/1 fetches Version 1 snapshot", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/incidents/" + inc.ID.String() + "/revisions/1")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var revision models.IncidentRevision
		err = json.NewDecoder(resp.Body).Decode(&revision)
		require.NoError(t, err)

		assert.Equal(t, 1, revision.VersionNumber)
		assert.Equal(t, "Initial Report Title", revision.Title)
	})

	t.Run("4. GET /api/v1/incidents/{id}/revisions/99 returns 404 Not Found", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/incidents/" + inc.ID.String() + "/revisions/99")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
