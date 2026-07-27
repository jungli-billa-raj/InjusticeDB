package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

type CastVoteRequest struct {
	Vote models.VoteType `json:"vote"`
}

type VoteTallyResponse struct {
	VerifyCount int `json:"verify_count"`
	RejectCount int `json:"reject_count"`
}

// CastVote records or updates a user's crowd verification vote for an incident.
// @Summary      Cast or update verification vote
// @Description  Records a vote (verify or reject) on an incident for the authenticated user.
// @Tags         incidents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string           true  "Incident UUID"
// @Param        body  body      CastVoteRequest  true  "Vote Payload"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /incidents/{id}/verifications [post]
func (s *Server) CastVote(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(uuid.UUID)
	if !ok {
		Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	incidentID, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "Invalid incident ID format")
		return
	}

	var req CastVoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.Vote != models.VoteVerify && req.Vote != models.VoteReject {
		Error(w, http.StatusBadRequest, "Invalid vote type (must be 'verify' or 'reject')")
		return
	}

	err = s.repos.Verifications.CastVote(r.Context(), incidentID, userID, req.Vote)
	if err != nil {
		Error(w, http.StatusInternalServerError, "Failed to record vote")
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "Vote recorded successfully"})
}

// GetVoteTally retrieves the total crowd verification vote counts for an incident.
// @Summary      Get incident vote tally
// @Description  Returns total verify and reject counts for an incident.
// @Tags         incidents
// @Produce      json
// @Param        id   path      string  true  "Incident UUID"
// @Success      200  {object}  VoteTallyResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /incidents/{id}/verifications/tally [get]
func (s *Server) GetVoteTally(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	incidentID, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "Invalid incident ID format")
		return
	}

	verifyCount, rejectCount, err := s.repos.Verifications.GetVoteTally(r.Context(), incidentID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "Failed to retrieve vote tally")
		return
	}

	JSON(w, http.StatusOK, VoteTallyResponse{
		VerifyCount: verifyCount,
		RejectCount: rejectCount,
	})
}
