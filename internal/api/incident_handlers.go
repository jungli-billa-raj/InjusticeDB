package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

type UpdateVerificationStatusRequest struct {
	Status models.VerificationStatus `json:"status"`
}

type CreateRevisionRequest struct {
	Title         string               `json:"title"`
	FullStory     string               `json:"full_story"`
	Severity      int                  `json:"severity"`
	JusticeStatus models.JusticeStatus `json:"justice_status"`
	State         string               `json:"state"`
	City          string               `json:"city"`
	ChangeSummary string               `json:"change_summary"`
}

// CreateIncident handles master record and initial revision creation.
// @Summary      Create a new incident
// @Description  Creates an incident master record and seeds its initial version 1 revision snapshot.
// @Tags         incidents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      models.CreateIncidentParams  true  "Create Incident Payload"
// @Success      201  {object}  models.Incident
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /incidents [post]
func (s *Server) CreateIncident(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDKey).(uuid.UUID)
	if !ok {
		Error(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var params models.CreateIncidentParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if params.Title == "" || params.FullStory == "" || params.State == "" || params.City == "" {
		Error(w, http.StatusBadRequest, "Title, full story, state, and city are required")
		return
	}

	params.CreatedBy = &userID

	incident, err := s.repos.Incidents.Create(r.Context(), params)
	if err != nil {
		Error(w, http.StatusInternalServerError, "Failed to create incident record")
		return
	}

	JSON(w, http.StatusCreated, incident)
}

// GetIncidentByID fetches the full latest snapshot of an incident.
// @Summary      Get full latest incident details
// @Description  Retrieves the full latest incident details including current revision snapshot.
// @Tags         incidents
// @Produce      json
// @Param        id   path      string  true  "Incident UUID"
// @Success      200  {object}  models.FullLatestIncident
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /incidents/{id} [get]
func (s *Server) GetIncidentByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	incidentID, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "Invalid incident ID format")
		return
	}

	fullIncident, err := s.repos.Incidents.GetByID(r.Context(), incidentID)
	if err != nil {
		Error(w, http.StatusNotFound, "Incident not found")
		return
	}

	JSON(w, http.StatusOK, fullIncident)
}

// UpdateVerificationStatus modifies an incident's moderation state (Admin/Moderator only).
// @Summary      Update incident verification status
// @Description  Updates verification status (pending, verified, rejected, disputed). Requires Moderator or Admin role.
// @Tags         incidents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                          true  "Incident UUID"
// @Param        body  body      UpdateVerificationStatusRequest true  "Status Payload"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /incidents/{id}/verification [patch]
func (s *Server) UpdateVerificationStatus(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value(UserRoleKey).(string)
	if role != "admin" && role != "moderator" {
		Error(w, http.StatusForbidden, "Insufficient privileges")
		return
	}

	idStr := chi.URLParam(r, "id")
	incidentID, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "Invalid incident ID format")
		return
	}

	var req UpdateVerificationStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if !req.Status.IsValid() {
		Error(w, http.StatusBadRequest, "Invalid verification status value")
		return
	}

	err = s.repos.Incidents.UpdateVerificationStatus(r.Context(), incidentID, req.Status)
	if err != nil {
		Error(w, http.StatusInternalServerError, "Failed to update verification status")
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "Verification status updated successfully"})
}

// CreateIncidentRevision adds a new version snapshot to an incident.
// @Summary      Submit a new incident revision
// @Description  Creates a new revision snapshot for an incident and bumps its current version number.
// @Tags         incidents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                 true  "Incident UUID"
// @Param        body  body      CreateRevisionRequest  true  "Revision Payload"
// @Success      201  {object}  models.IncidentRevision
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /incidents/{id}/revisions [post]
func (s *Server) CreateIncidentRevision(w http.ResponseWriter, r *http.Request) {
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

	var req CreateRevisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.Title == "" || req.FullStory == "" || req.ChangeSummary == "" {
		Error(w, http.StatusBadRequest, "Title, full story, and change summary are required")
		return
	}

	revisionInput := models.IncidentRevision{
		IncidentID:    incidentID,
		Title:         req.Title,
		FullStory:     req.FullStory,
		Severity:      req.Severity,
		JusticeStatus: req.JusticeStatus,
		State:         req.State,
		City:          req.City,
		ChangeSummary: req.ChangeSummary,
		EditedBy:      &userID,
	}

	createdRevision, err := s.repos.Incidents.CreateRevision(r.Context(), revisionInput)
	if err != nil {
		Error(w, http.StatusInternalServerError, "Failed to create revision")
		return
	}

	JSON(w, http.StatusCreated, createdRevision)
}

// ListIncidentRevisions retrieves the complete edit history for an incident.
// @Summary      List incident revision history
// @Description  Retrieves all historical revisions for an incident ordered by version number.
// @Tags         incidents
// @Produce      json
// @Param        id   path      string  true  "Incident UUID"
// @Success      200  {array}   models.IncidentRevision
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /incidents/{id}/revisions [get]
func (s *Server) ListIncidentRevisions(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	incidentID, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "Invalid incident ID format")
		return
	}

	revisions, err := s.repos.Incidents.ListRevisions(r.Context(), incidentID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "Failed to list revisions")
		return
	}

	JSON(w, http.StatusOK, revisions)
}

// GetIncidentRevisionByVersion retrieves a specific historical version snapshot.
// @Summary      Get specific incident version
// @Description  Retrieves an incident snapshot for a specific version number.
// @Tags         incidents
// @Produce      json
// @Param        id       path      string  true  "Incident UUID"
// @Param        version  path      int     true  "Version Number"
// @Success      200      {object}  models.IncidentRevision
// @Failure      400      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
// @Router       /incidents/{id}/revisions/{version} [get]
func (s *Server) GetIncidentRevisionByVersion(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	incidentID, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "Invalid incident ID format")
		return
	}

	versionStr := chi.URLParam(r, "version")
	version, err := strconv.Atoi(versionStr)
	if err != nil || version < 1 {
		Error(w, http.StatusBadRequest, "Invalid version number")
		return
	}

	revision, err := s.repos.Incidents.GetRevision(r.Context(), incidentID, version)
	if err != nil {
		Error(w, http.StatusNotFound, "Revision version not found")
		return
	}

	JSON(w, http.StatusOK, revision)
}
