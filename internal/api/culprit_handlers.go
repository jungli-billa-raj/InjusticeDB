package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

type LinkCulpritRequest struct {
	PersonID uuid.UUID            `json:"person_id"`
	Status   models.CulpritStatus `json:"status"`
}

type UpdateCulpritStatusRequest struct {
	Status models.CulpritStatus `json:"status"`
}

// CreatePerson registers a new individual or entity in the database.
// @Summary      Create a new person/entity
// @Description  Registers a new individual or entity in the central registry.
// @Tags         people
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      models.Person  true  "Person Payload"
// @Success      201  {object}  models.Person
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /people [post]
func (s *Server) CreatePerson(w http.ResponseWriter, r *http.Request) {
	var person models.Person
	if err := json.NewDecoder(r.Body).Decode(&person); err != nil {
		Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if person.Name == "" {
		Error(w, http.StatusBadRequest, "Name is required")
		return
	}

	created, err := s.repos.Culprits.CreatePerson(r.Context(), person)
	if err != nil {
		Error(w, http.StatusInternalServerError, "Failed to create person record")
		return
	}

	JSON(w, http.StatusCreated, created)
}

// LinkCulpritToIncident links a registered person to an incident with a culprit status.
// @Summary      Link culprit to incident
// @Description  Associates a registered person to an incident under a specific status (suspect, accused, guilty, convicted).
// @Tags         incidents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string              true  "Incident UUID"
// @Param        body  body      LinkCulpritRequest  true  "Link Payload"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /incidents/{id}/culprits [post]
func (s *Server) LinkCulpritToIncident(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	incidentID, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "Invalid incident ID format")
		return
	}

	var req LinkCulpritRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.PersonID == uuid.Nil {
		Error(w, http.StatusBadRequest, "Person ID is required")
		return
	}

	if !req.Status.IsValid() {
		Error(w, http.StatusBadRequest, "Invalid culprit status value")
		return
	}

	err = s.repos.Culprits.LinkToIncident(r.Context(), incidentID, req.PersonID, req.Status)
	if err != nil {
		Error(w, http.StatusInternalServerError, "Failed to link culprit to incident")
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "Culprit linked to incident successfully"})
}

// GetCulpritsForIncident lists all linked culprits for an incident.
// @Summary      Get linked culprits for incident
// @Description  Retrieves all culprits associated with an incident along with their status details.
// @Tags         incidents
// @Produce      json
// @Param        id   path      string  true  "Incident UUID"
// @Success      200  {array}   models.IncidentCulprit
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /incidents/{id}/culprits [get]
func (s *Server) GetCulpritsForIncident(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	incidentID, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "Invalid incident ID format")
		return
	}

	culprits, err := s.repos.Culprits.GetCulpritsForIncident(r.Context(), incidentID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "Failed to retrieve culprits")
		return
	}

	JSON(w, http.StatusOK, culprits)
}

// UpdateCulpritStatus modifies the legal status of a linked culprit (Moderator/Admin only).
// @Summary      Update linked culprit status
// @Description  Updates legal status for a linked culprit on an incident. Requires Moderator or Admin role.
// @Tags         incidents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id         path      string                      true  "Incident UUID"
// @Param        person_id  path      string                      true  "Person UUID"
// @Param        body       body      UpdateCulpritStatusRequest  true  "Status Payload"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /incidents/{id}/culprits/{person_id} [patch]
func (s *Server) UpdateCulpritStatus(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value(UserRoleKey).(string)
	if role != "admin" && role != "moderator" {
		Error(w, http.StatusForbidden, "Insufficient privileges")
		return
	}

	incIDStr := chi.URLParam(r, "id")
	incidentID, err := uuid.Parse(incIDStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "Invalid incident ID format")
		return
	}

	personIDStr := chi.URLParam(r, "person_id")
	personID, err := uuid.Parse(personIDStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "Invalid person ID format")
		return
	}

	var req UpdateCulpritStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if !req.Status.IsValid() {
		Error(w, http.StatusBadRequest, "Invalid culprit status value")
		return
	}

	err = s.repos.Culprits.UpdateCulpritStatus(r.Context(), incidentID, personID, req.Status)
	if err != nil {
		Error(w, http.StatusInternalServerError, "Failed to update culprit status")
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "Culprit status updated successfully"})
}
