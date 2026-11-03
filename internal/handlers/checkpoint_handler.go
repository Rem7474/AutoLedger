package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
)

type CheckpointHandler struct {
	repo *database.Repository
}

func NewCheckpointHandler(repo *database.Repository) *CheckpointHandler {
	return &CheckpointHandler{repo: repo}
}

type SaveCheckpointRequest struct {
	ExpectedVersion *int64  `json:"expected_version"` // The version the client read; absent: not checked
	Date            string  `json:"date"`
	Odometer        float64 `json:"odometer"`
	Notes           *string `json:"notes"`
}

// decodeCheckpointRequest reads and validates the shared payload used by Create and Update.
func decodeCheckpointRequest(r *http.Request) (time.Time, float64, *string, *int64, error) {
	var req SaveCheckpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return time.Time{}, 0, nil, nil, apierror.New("request.invalid_body", "Invalid request body")
	}

	d, err := parseDate(req.Date)
	if err != nil {
		return time.Time{}, 0, nil, nil, apierror.New("request.invalid_date", "Invalid date")
	}
	if req.Odometer < 0 || req.Odometer > 2_000_000 {
		return time.Time{}, 0, nil, nil, apierror.Newf("odometer.range_checkpoint", "Invalid odometer (must be between 0 and %.0f km)", apierror.Km(2_000_000))
	}

	var notes *string
	if req.Notes != nil && strings.TrimSpace(*req.Notes) != "" {
		n := strings.TrimSpace(*req.Notes)
		notes = &n
	}

	return d, req.Odometer, notes, req.ExpectedVersion, nil
}

// checkConsistency rejects a reading that contradicts the other manual readings and fill-ups; id is empty on creation.
func (h *CheckpointHandler) checkConsistency(w http.ResponseWriter, r *http.Request, vehicleID, id string, date time.Time, odometer float64) bool {
	points, err := h.repo.ListManualOdometerPoints(r.Context(), vehicleID)
	if err != nil {
		writeRepoError(w, r, err, "Failed to check odometer points")
		return false
	}
	if err := checkOdometerOrder(points, id, date, odometer); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return false
	}
	return true
}

func (h *CheckpointHandler) List(w http.ResponseWriter, r *http.Request) {
	vehicleID := chi.URLParam(r, "vehicleId")
	if vehicleID == "" {
		vehicleID = chi.URLParam(r, "id")
	}

	if v := requireVehicleAccess(w, r, h.repo, vehicleID, models.RoleViewer); v == nil {
		return
	}

	list, err := h.repo.ListOdometerCheckpoints(r.Context(), vehicleID)
	if err != nil {
		writeRepoError(w, r, err, "Failed to list checkpoints")
		return
	}
	if list == nil {
		list = []models.OdometerCheckpoint{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *CheckpointHandler) Create(w http.ResponseWriter, r *http.Request) {
	vehicleID := chi.URLParam(r, "vehicleId")
	if vehicleID == "" {
		vehicleID = chi.URLParam(r, "id")
	}

	if v := requireVehicleAccess(w, r, h.repo, vehicleID, models.RoleEditor); v == nil {
		return
	}

	d, odometer, notes, _, err := decodeCheckpointRequest(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	if !h.checkConsistency(w, r, vehicleID, "", d, odometer) {
		return
	}

	c := &models.OdometerCheckpoint{
		VehicleID: vehicleID,
		Date:      d,
		Odometer:  odometer,
		Notes:     notes,
	}

	if err := h.repo.CreateOdometerCheckpoint(r.Context(), c); err != nil {
		writeRepoError(w, r, err, "Failed to create checkpoint")
		return
	}

	writeJSON(w, http.StatusCreated, c)
}

func (h *CheckpointHandler) Update(w http.ResponseWriter, r *http.Request) {
	vehicleID := chi.URLParam(r, "vehicleId")
	if vehicleID == "" {
		vehicleID = chi.URLParam(r, "id")
	}
	checkpointID := chi.URLParam(r, "checkpointId")

	if v := requireVehicleAccess(w, r, h.repo, vehicleID, models.RoleEditor); v == nil {
		return
	}

	d, odometer, notes, expected, err := decodeCheckpointRequest(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	if !h.checkConsistency(w, r, vehicleID, checkpointID, d, odometer) {
		return
	}

	c := &models.OdometerCheckpoint{
		ID:              checkpointID,
		VehicleID:       vehicleID,
		Date:            d,
		Odometer:        odometer,
		Notes:           notes,
		ExpectedVersion: expected,
	}

	if err := h.repo.UpdateOdometerCheckpoint(r.Context(), c); err != nil {
		writeRepoError(w, r, err, "Failed to update checkpoint")
		return
	}

	writeJSON(w, http.StatusOK, c)
}

func (h *CheckpointHandler) Delete(w http.ResponseWriter, r *http.Request) {
	vehicleID := chi.URLParam(r, "vehicleId")
	if vehicleID == "" {
		vehicleID = chi.URLParam(r, "id")
	}
	checkpointID := chi.URLParam(r, "checkpointId")

	if v := requireVehicleAccess(w, r, h.repo, vehicleID, models.RoleEditor); v == nil {
		return
	}

	if err := h.repo.DeleteOdometerCheckpoint(r.Context(), vehicleID, checkpointID); err != nil {
		writeRepoError(w, r, err, "Failed to delete checkpoint")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}
