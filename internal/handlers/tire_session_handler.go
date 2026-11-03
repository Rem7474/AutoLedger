package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/models"
)

type MountSessionPayload struct {
	Position           models.TirePosition `json:"position"`
	MountedDate        string              `json:"mounted_date"`
	MountedOdometer    float64             `json:"mounted_odometer"`
	DismountedDate     *string             `json:"dismounted_date"`
	DismountedOdometer *float64            `json:"dismounted_odometer"`
	DistanceKm         float64             `json:"distance_km"`
	Notes              *string             `json:"notes"`
	ExpectedVersion    *int64              `json:"expected_version"` // The version the client read; absent: not checked
}

func parseSessionPayload(req *MountSessionPayload) (time.Time, *time.Time, error) {
	// Historical (completed) sessions may be stored without a precise wheel position.
	// In that case the frontend sends position=STORAGE, which is accepted only when
	// a dismounted_date is provided (i.e. the session is already over).
	isHistoricalStorage := req.Position == models.TirePosStorage && req.DismountedDate != nil
	if !isMountedPosition(req.Position) && !isHistoricalStorage {
		return time.Time{}, nil, apierror.New("tire.session_position", "A fitting session needs a position FL, FR, RL or RR (or STORAGE for a finished historical session)")
	}
	if req.MountedOdometer < 0 || (req.DismountedOdometer != nil && *req.DismountedOdometer < 0) {
		return time.Time{}, nil, apierror.New("odometer.invalid", "Invalid odometer")
	}
	mountedDate, err := parseDate(req.MountedDate)
	if err != nil {
		return time.Time{}, nil, err
	}
	dismountedDate, err := parseOptionalDate(req.DismountedDate)
	if err != nil {
		return time.Time{}, nil, err
	}
	if dismountedDate != nil && dismountedDate.Before(mountedDate) {
		return time.Time{}, nil, apierror.New("tire.dismount_before_mount", "The removal date is before the fitting date")
	}
	// A day of slack covers the time zone of the client, which sends today's date.
	if limit := time.Now().UTC().Add(24 * time.Hour); mountedDate.After(limit) || (dismountedDate != nil && dismountedDate.After(limit)) {
		return time.Time{}, nil, apierror.New("tire.session_in_future", "A fitting or removal date cannot be in the future")
	}
	return mountedDate, dismountedDate, nil
}

// decodeMountSession reads a mount session of the given tire from the request body. It writes the error response
// and returns false when the payload is invalid.
func decodeMountSession(w http.ResponseWriter, r *http.Request, vehicleID, tireID string) (*models.TireMountSession, bool) {
	var req MountSessionPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, apierror.New("request.invalid_body", "Invalid request body"))
		return nil, false
	}

	mountedDate, dismountedDate, err := parseSessionPayload(&req)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return nil, false
	}

	return &models.TireMountSession{
		TireID:             tireID,
		VehicleID:          vehicleID,
		Position:           req.Position,
		MountedDate:        mountedDate,
		MountedOdometer:    req.MountedOdometer,
		DismountedDate:     dismountedDate,
		DismountedOdometer: req.DismountedOdometer,
		DistanceKm:         req.DistanceKm,
		Notes:              req.Notes,
		ExpectedVersion:    req.ExpectedVersion,
	}, true
}

func (h *TireHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	vehicleID := chi.URLParam(r, "vehicleId")
	tireID := chi.URLParam(r, "tireId")

	if v := requireVehicleAccess(w, r, h.repo, vehicleID, models.RoleEditor); v == nil {
		return
	}

	session, ok := decodeMountSession(w, r, vehicleID, tireID)
	if !ok {
		return
	}

	if err := h.repo.CreateTireMountSession(r.Context(), session); err != nil {
		writeRepoError(w, r, err, "Failed to create mount session")
		return
	}

	writeJSON(w, http.StatusCreated, session)
}

func (h *TireHandler) UpdateSession(w http.ResponseWriter, r *http.Request) {
	vehicleID := chi.URLParam(r, "vehicleId")
	tireID := chi.URLParam(r, "tireId")
	sessionID := chi.URLParam(r, "sessionId")

	if v := requireVehicleAccess(w, r, h.repo, vehicleID, models.RoleEditor); v == nil {
		return
	}

	session, ok := decodeMountSession(w, r, vehicleID, tireID)
	if !ok {
		return
	}
	session.ID = sessionID

	if err := h.repo.UpdateTireMountSession(r.Context(), session); err != nil {
		writeRepoError(w, r, err, "Failed to update mount session")
		return
	}

	writeJSON(w, http.StatusOK, session)
}

func (h *TireHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	vehicleID := chi.URLParam(r, "vehicleId")
	tireID := chi.URLParam(r, "tireId")
	sessionID := chi.URLParam(r, "sessionId")

	if v := requireVehicleAccess(w, r, h.repo, vehicleID, models.RoleEditor); v == nil {
		return
	}

	if err := h.repo.DeleteTireMountSession(r.Context(), vehicleID, sessionID, tireID); err != nil {
		writeRepoError(w, r, err, "Failed to delete mount session")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}
