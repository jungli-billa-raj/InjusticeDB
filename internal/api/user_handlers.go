package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

type UpdateCredibilityRequest struct {
	Delta int `json:"delta"` // Can be positive (+10) or negative (-25)
}

// CreateOrUpdateUser handles user profile upserts.
// @Summary      Create or update user
// @Description  Upserts a user profile using email, name, and provider.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body      models.CreateUserParams  true  "User Upsert Payload"
// @Success      200  {object}  models.User
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /users/upsert [post]
func (s *Server) CreateOrUpdateUser(w http.ResponseWriter, r *http.Request) {
	var params models.CreateUserParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if params.Email == "" || params.Name == "" {
		Error(w, http.StatusBadRequest, "Email and name are required")
		return
	}

	user, err := s.repos.Users.CreateOrUpdate(r.Context(), params)
	if err != nil {
		Error(w, http.StatusInternalServerError, "Failed to create or update user profile")
		return
	}

	JSON(w, http.StatusOK, user)
}

// GetUserProfile handles public profile retrieval.
// @Summary      Get user profile
// @Description  Retrieves public details and credibility score for a given user UUID.
// @Tags         users
// @Produce      json
// @Param        id   path      string  true  "User UUID"
// @Success      200  {object}  models.User
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /users/{id} [get]
func (s *Server) GetUserProfile(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "Invalid user ID format")
		return
	}

	user, err := s.repos.Users.GetByID(r.Context(), userID)
	if err != nil {
		Error(w, http.StatusNotFound, "User not found")
		return
	}

	JSON(w, http.StatusOK, user)
}

// UpdateUserCredibility adjusts a user's score delta (Admin/Moderator only).
// @Summary      Update user credibility
// @Description  Adjusts a user's score up or down by a given integer delta. Requires Moderator or Admin role.
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                   true  "Target User UUID"
// @Param        body  body      UpdateCredibilityRequest true  "Credibility Delta Payload"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /users/{id}/credibility [patch]
func (s *Server) UpdateUserCredibility(w http.ResponseWriter, r *http.Request) {
	// 1. Role Enforcement (Must be admin or moderator)
	role, _ := r.Context().Value(UserRoleKey).(string)
	if role != "admin" && role != "moderator" {
		Error(w, http.StatusForbidden, "Insufficient privileges")
		return
	}

	// 2. Parse target User ID from URL
	targetIDStr := chi.URLParam(r, "id")
	targetID, err := uuid.Parse(targetIDStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "Invalid target user ID format")
		return
	}

	// 3. Decode JSON request payload
	var req UpdateCredibilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// 4. Update in Database via UserRepository
	err = s.repos.Users.UpdateCredibility(r.Context(), targetID, req.Delta)
	if err != nil {
		Error(w, http.StatusInternalServerError, "Failed to update credibility score")
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "Credibility updated successfully"})
}
