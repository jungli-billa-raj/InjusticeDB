package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

func TestVerificationEndpoints_CombinedFlow(t *testing.T) {
	ts, repos, teardown := setupAPITestServer(t)
	defer teardown()

	ctx := t.Context()

	// 1. Seed Voters and an Incident
	voterOne, err := repos.Users.CreateOrUpdate(ctx, models.CreateUserParams{
		Email:        "voter_one@example.com",
		Name:         "Voter One",
		AuthProvider: "google",
	})
	require.NoError(t, err)

	voterTwo, err := repos.Users.CreateOrUpdate(ctx, models.CreateUserParams{
		Email:        "voter_two@example.com",
		Name:         "Voter Two",
		AuthProvider: "google",
	})
	require.NoError(t, err)

	tokenOne := generateTestJWT(voterOne.ID, "user")
	tokenTwo := generateTestJWT(voterTwo.ID, "user")

	inc, err := repos.Incidents.Create(ctx, models.CreateIncidentParams{
		Title:         "Public Land Encroachment",
		FullStory:     "Encroachment on public park grounds.",
		Severity:      6,
		JusticeStatus: models.JusticeProceeding,
		State:         "Jharkhand",
		City:          "Ranchi",
		CreatedBy:     &voterOne.ID,
	})
	require.NoError(t, err)

	t.Run("1. GET /api/v1/incidents/{id}/verifications/tally returns 0,0 initially", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/incidents/" + inc.ID.String() + "/verifications/tally")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var tally VoteTallyResponse
		err = json.NewDecoder(resp.Body).Decode(&tally)
		require.NoError(t, err)

		assert.Equal(t, 0, tally.VerifyCount)
		assert.Equal(t, 0, tally.RejectCount)
	})

	t.Run("2. POST /api/v1/incidents/{id}/verifications records votes from multiple users", func(t *testing.T) {
		// Voter 1 casts 'verify'
		payloadOne := CastVoteRequest{Vote: models.VoteVerify}
		bodyOne, _ := json.Marshal(payloadOne)

		reqOne, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/incidents/"+inc.ID.String()+"/verifications", bytes.NewBuffer(bodyOne))
		reqOne.Header.Set("Authorization", "Bearer "+tokenOne)
		reqOne.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		respOne, err := client.Do(reqOne)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, respOne.StatusCode)
		respOne.Body.Close()

		// Voter 2 casts 'reject'
		payloadTwo := CastVoteRequest{Vote: models.VoteReject}
		bodyTwo, _ := json.Marshal(payloadTwo)

		reqTwo, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/incidents/"+inc.ID.String()+"/verifications", bytes.NewBuffer(bodyTwo))
		reqTwo.Header.Set("Authorization", "Bearer "+tokenTwo)
		reqTwo.Header.Set("Content-Type", "application/json")

		respTwo, err := client.Do(reqTwo)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, respTwo.StatusCode)
		respTwo.Body.Close()

		// Verify tally is 1 verify, 1 reject
		respTally, err := http.Get(ts.URL + "/api/v1/incidents/" + inc.ID.String() + "/verifications/tally")
		require.NoError(t, err)
		defer respTally.Body.Close()

		var tally VoteTallyResponse
		_ = json.NewDecoder(respTally.Body).Decode(&tally)
		assert.Equal(t, 1, tally.VerifyCount)
		assert.Equal(t, 1, tally.RejectCount)
	})

	t.Run("3. POST /api/v1/incidents/{id}/verifications updates vote on conflict", func(t *testing.T) {
		// Voter 2 changes vote from 'reject' to 'verify'
		payloadChange := CastVoteRequest{Vote: models.VoteVerify}
		bodyChange, _ := json.Marshal(payloadChange)

		reqChange, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/incidents/"+inc.ID.String()+"/verifications", bytes.NewBuffer(bodyChange))
		reqChange.Header.Set("Authorization", "Bearer "+tokenTwo)
		reqChange.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		respChange, err := client.Do(reqChange)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, respChange.StatusCode)
		respChange.Body.Close()

		// Verify tally updated to 2 verify, 0 reject
		respTally, err := http.Get(ts.URL + "/api/v1/incidents/" + inc.ID.String() + "/verifications/tally")
		require.NoError(t, err)
		defer respTally.Body.Close()

		var tally VoteTallyResponse
		_ = json.NewDecoder(respTally.Body).Decode(&tally)
		assert.Equal(t, 2, tally.VerifyCount)
		assert.Equal(t, 0, tally.RejectCount)
	})
}
