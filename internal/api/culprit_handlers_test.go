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

func TestCulpritEndpoints_CombinedFlow(t *testing.T) {
	ts, repos, teardown := setupAPITestServer(t)
	defer teardown()

	ctx := t.Context()

	// Seed User and Incident
	user, err := repos.Users.CreateOrUpdate(ctx, models.CreateUserParams{
		Email:        "culprit_tester@example.com",
		Name:         "Culprit Tester",
		AuthProvider: "google",
	})
	require.NoError(t, err)

	userToken := generateTestJWT(user.ID, "user")
	adminToken := generateTestJWT(user.ID, "admin")

	inc, err := repos.Incidents.Create(ctx, models.CreateIncidentParams{
		Title:         "Financial Fraud Incident",
		FullStory:     "Embezzlement of public infrastructure grants.",
		Severity:      9,
		JusticeStatus: models.JusticeProceeding,
		State:         "Jharkhand",
		City:          "Ranchi",
		CreatedBy:     &user.ID,
	})
	require.NoError(t, err)

	var personID uuid.UUID

	t.Run("1. POST /api/v1/people creates person record", func(t *testing.T) {
		org := "Local Contracting Corp"
		personPayload := models.Person{
			Name:         "John Contractor",
			Organization: &org,
		}
		body, _ := json.Marshal(personPayload)

		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/people", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+userToken)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var createdPerson models.Person
		err = json.NewDecoder(resp.Body).Decode(&createdPerson)
		require.NoError(t, err)

		assert.NotEqual(t, uuid.Nil, createdPerson.ID)
		assert.Equal(t, "John Contractor", createdPerson.Name)
		personID = createdPerson.ID
	})

	t.Run("2. POST /api/v1/incidents/{id}/culprits links person to incident", func(t *testing.T) {
		linkPayload := LinkCulpritRequest{
			PersonID: personID,
			Status:   models.CulpritSuspect,
		}
		body, _ := json.Marshal(linkPayload)

		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/incidents/"+inc.ID.String()+"/culprits", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+userToken)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("3. GET /api/v1/incidents/{id}/culprits lists linked culprits publicly", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/incidents/" + inc.ID.String() + "/culprits")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var culprits []models.IncidentCulprit
		err = json.NewDecoder(resp.Body).Decode(&culprits)
		require.NoError(t, err)

		require.Len(t, culprits, 1)
		assert.Equal(t, personID, culprits[0].PersonID)
		assert.Equal(t, models.CulpritSuspect, culprits[0].CulpritStatus)
		assert.Equal(t, "John Contractor", culprits[0].Person.Name)
	})

	t.Run("4. PATCH /api/v1/incidents/{id}/culprits/{person_id} updates status with RBAC", func(t *testing.T) {
		updatePayload := UpdateCulpritStatusRequest{
			Status: models.CulpritConvicted,
		}
		body, _ := json.Marshal(updatePayload)

		// 4a. Standard user attempt should be forbidden (403)
		reqUser, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/incidents/"+inc.ID.String()+"/culprits/"+personID.String(), bytes.NewBuffer(body))
		reqUser.Header.Set("Authorization", "Bearer "+userToken)
		reqUser.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		respUser, err := client.Do(reqUser)
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, respUser.StatusCode)
		respUser.Body.Close()

		// 4b. Admin attempt should succeed (200)
		reqAdmin, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/incidents/"+inc.ID.String()+"/culprits/"+personID.String(), bytes.NewBuffer(body))
		reqAdmin.Header.Set("Authorization", "Bearer "+adminToken)
		reqAdmin.Header.Set("Content-Type", "application/json")

		respAdmin, err := client.Do(reqAdmin)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, respAdmin.StatusCode)
		respAdmin.Body.Close()

		// 4c. Verify updated status
		respVerify, err := http.Get(ts.URL + "/api/v1/incidents/" + inc.ID.String() + "/culprits")
		require.NoError(t, err)
		defer respVerify.Body.Close()

		var updatedCulprits []models.IncidentCulprit
		_ = json.NewDecoder(respVerify.Body).Decode(&updatedCulprits)
		require.Len(t, updatedCulprits, 1)
		assert.Equal(t, models.CulpritConvicted, updatedCulprits[0].CulpritStatus)
	})
}
