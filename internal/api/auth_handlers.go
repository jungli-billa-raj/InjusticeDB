package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
	"google.golang.org/api/idtoken"
)

type GoogleAuthRequest struct {
	IDToken string `json:"id_token"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

// HandleGoogleLogin processes Google OAuth ID tokens and returns an application JWT
func (s *Server) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	var req GoogleAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.IDToken == "" {
		Error(w, http.StatusBadRequest, "Missing Google id_token")
		return
	}

	// 1. Verify Google ID Token using official Google API validator
	// Note: You can pass your GOOGLE_CLIENT_ID here to validate audience matching
	payload, err := idtoken.Validate(r.Context(), req.IDToken, s.cfg.GoogleClientID)
	if err != nil {
		Error(w, http.StatusUnauthorized, fmt.Sprintf("Invalid Google token: %v", err))
		return
	}

	// 2. Extract Claims from verified Google token payload
	email, ok := payload.Claims["email"].(string)
	if !ok || email == "" {
		Error(w, http.StatusBadRequest, "Email not provided in Google token")
		return
	}

	name, _ := payload.Claims["name"].(string)
	if name == "" {
		name = "Anonymous User"
	}

	// 3. Upsert user in database via UserRepository
	user, err := s.repos.Users.CreateOrUpdate(r.Context(), models.CreateUserParams{
		Email:        email,
		Name:         name,
		AuthProvider: "google",
	})
	if err != nil {
		Error(w, http.StatusInternalServerError, "Failed to register or retrieve user")
		return
	}

	// 4. Generate signed JWT for the authenticated session
	jwtToken, err := s.generateJWT(user.ID, string(user.Role))
	if err != nil {
		Error(w, http.StatusInternalServerError, "Failed to generate authentication token")
		return
	}

	// 5. Send token and user record back to client
	JSON(w, http.StatusOK, AuthResponse{
		Token: jwtToken,
		User:  user,
	})
}

// Helper to generate signed JWTs
func (s *Server) generateJWT(userID uuid.UUID, role string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"role":    role,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(), // 7 day expiration
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}
