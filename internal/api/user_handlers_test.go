package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jungli-billa-raj/InjusticeDB/internal/archival"
	"github.com/jungli-billa-raj/InjusticeDB/internal/db"
	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

const testJWTSecret = "test-secret-key-12345"

func setupAPITestServer(t *testing.T) (*httptest.Server, *db.Repositories, func()) {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/injusticedb?sslmode=disable"
	}

	ctx := t.Context()

	// 1. Initialize Pgxpool
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err, "failed to connect to test database pool")

	// 2. Construct Repositories matching main.go
	repos := &db.Repositories{
		Users:         db.NewPostgresUserRepository(pool),
		Incidents:     db.NewPostgresIncidentRepository(pool),
		Culprits:      db.NewPostgresCulpritRepository(pool),
		Assets:        db.NewPostgresAssetRepository(pool),
		Verifications: db.NewPostgresVerificationRepository(pool),
		Messaging:     db.NewPostgresMessagingRepository(pool),
		Comments:      db.NewPostgresCommentRepository(pool),
		Targets:       db.NewPostgresTargetRepository(pool),
	}

	cfg := Config{
		JWTSecret: testJWTSecret,
	}

	// 3. Initialize API Server with a NopArchiver (or nil if unused by user routes)
	server := NewServer(repos, archival.NewNopArchiver(), cfg)
	ts := httptest.NewServer(server)

	teardown := func() {
		ts.Close()
		pool.Close()
	}

	return ts, repos, teardown
}

// Helper function to generate JWT token for test requests
func generateTestJWT(userID uuid.UUID, role string) string {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"role":    role,
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(testJWTSecret))
	return tokenStr
}

func TestUserEndpoints_CombinedFlow(t *testing.T) {
	ts, _, teardown := setupAPITestServer(t)
	defer teardown()

	var createdUserID uuid.UUID

	t.Run("1. POST /api/v1/users/upsert creates a new user", func(t *testing.T) {
		payload := models.CreateUserParams{
			Email:        "johndoe@example.com",
			Name:         "John Doe",
			AuthProvider: "google",
		}
		body, _ := json.Marshal(payload)

		resp, err := http.Post(ts.URL+"/api/v1/users/upsert", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var user models.User
		err = json.NewDecoder(resp.Body).Decode(&user)
		require.NoError(t, err)

		assert.NotEqual(t, uuid.Nil, user.ID)
		assert.Equal(t, "johndoe@example.com", user.Email)
		assert.Equal(t, "John Doe", user.Name)
		assert.Equal(t, 100, user.CredibilityScore)

		createdUserID = user.ID
	})

	t.Run("2. GET /api/v1/users/{id} retrieves public profile", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/users/" + createdUserID.String())
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var user models.User
		err = json.NewDecoder(resp.Body).Decode(&user)
		require.NoError(t, err)

		assert.Equal(t, createdUserID, user.ID)
		assert.Equal(t, "John Doe", user.Name)
	})

	t.Run("3. PATCH /api/v1/users/{id}/credibility enforces RBAC", func(t *testing.T) {
		updatePayload := map[string]int{"delta": 15}
		body, _ := json.Marshal(updatePayload)

		// 3a. Standard 'user' token should be denied (403 Forbidden)
		userToken := generateTestJWT(createdUserID, "user")
		req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/users/"+createdUserID.String()+"/credibility", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+userToken)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		resp.Body.Close()

		// 3b. 'admin' token should succeed (200 OK)
		adminToken := generateTestJWT(createdUserID, "admin")
		reqAdmin, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/users/"+createdUserID.String()+"/credibility", bytes.NewBuffer(body))
		reqAdmin.Header.Set("Authorization", "Bearer "+adminToken)
		reqAdmin.Header.Set("Content-Type", "application/json")

		respAdmin, err := client.Do(reqAdmin)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, respAdmin.StatusCode)
		respAdmin.Body.Close()

		// 3c. Verify credibility score increased from 100 to 115
		respVerify, err := http.Get(ts.URL + "/api/v1/users/" + createdUserID.String())
		require.NoError(t, err)
		defer respVerify.Body.Close()

		var updatedUser models.User
		_ = json.NewDecoder(respVerify.Body).Decode(&updatedUser)
		assert.Equal(t, 115, updatedUser.CredibilityScore)
	})
}
