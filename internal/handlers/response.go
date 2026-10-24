package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/teslacost/teslacost/internal/apierror"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

// writeAPIError answers with the English message, the stable code and the values the front end interpolates
// in its own translation of that code.
func writeAPIError(w http.ResponseWriter, status int, e *apierror.Error) {
	body := map[string]any{"error": e.Message, "code": e.Code}
	if len(e.Params) > 0 {
		body["params"] = e.Params
	}
	writeJSON(w, status, body)
}

// writeErr answers with err as an API error when it carries a code, and with its plain message otherwise.
func writeErr(w http.ResponseWriter, status int, err error) {
	if apiErr, ok := apierror.As(err); ok {
		writeAPIError(w, status, apiErr)
		return
	}
	writeError(w, status, err.Error())
}

// requestLanguage is the signed-in user's stored language ("en" or "fr", defaulting to "en"),
// used for short text built here and returned as plain strings rather than an apierror code
// (a default trip label, an auto-generated toll note): there is no catalog entry for the
// frontend to translate through.
func requestLanguage(r *http.Request, repo *database.Repository) string {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		return "en"
	}
	user, err := repo.GetUserByID(r.Context(), userID)
	if err != nil || user == nil || user.Language != "fr" {
		return "en"
	}
	return "fr"
}

// requireVehicleAccess checks that the authenticated user has access to the vehicle with at least minRole.
func requireVehicleAccess(w http.ResponseWriter, r *http.Request, repo *database.Repository, vehicleID string, minRole models.VehicleRole) *models.Vehicle {
	userID := middleware.GetUserID(r.Context())
	if !middleware.VehicleAllowed(r.Context(), vehicleID) {
		writeAPIError(w, http.StatusNotFound, apierror.New("vehicle.not_found", "Vehicle not found"))
		return nil
	}
	v, err := repo.GetVehicleByID(r.Context(), vehicleID, userID)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, apierror.New("vehicle.not_found", "Vehicle not found"))
		return nil
	}
	if minRole == models.RoleOwner && v.Role != models.RoleOwner {
		writeAPIError(w, http.StatusForbidden, apierror.New("access.owner_only", "This action is reserved to the vehicle owner"))
		return nil
	}
	if (minRole == models.RoleEditor || minRole == models.RoleOwner) && !v.Role.CanEdit() {
		writeAPIError(w, http.StatusForbidden, apierror.New("access.read_only", "Read-only access: changes are not allowed"))
		return nil
	}
	return v
}
