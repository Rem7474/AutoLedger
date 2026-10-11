package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
)

// ChargingPlaceHandler manages the places an account charges at and the classification of the locations its
// charges report. Places belong to the owner of the vehicle, so everyone with access sees the same classification.
type ChargingPlaceHandler struct {
	repo *database.Repository
}

func NewChargingPlaceHandler(repo *database.Repository) *ChargingPlaceHandler {
	return &ChargingPlaceHandler{repo: repo}
}

type chargingPlaceRequest struct {
	Name     string `json:"name"`
	Category string `json:"category"`
}

func (req *chargingPlaceRequest) validate() error {
	req.Name = strings.TrimSpace(req.Name)
	if n := utf8.RuneCountInString(req.Name); n < 1 || n > 100 {
		return apierror.New("charging_place.name_invalid", "The place name must be 1 to 100 characters")
	}
	if !models.ValidPlaceCategory(req.Category) {
		return apierror.New("charging_place.category_invalid", "Unknown place category")
	}
	return nil
}

func (h *ChargingPlaceHandler) writeError(w http.ResponseWriter, r *http.Request, err error, fallback string) {
	if errors.Is(err, database.ErrPlaceNameTaken) {
		writeAPIError(w, http.StatusConflict, apierror.New("charging_place.name_taken", "A place of that name already exists"))
		return
	}
	if errors.Is(err, database.ErrPlaceTextEmpty) {
		writeAPIError(w, http.StatusBadRequest, apierror.New("charging_place.text_invalid", "The location text is empty"))
		return
	}
	writeRepoError(w, r, err, fallback)
}

// List returns the places of the vehicle's owner and the location texts the vehicle's charges report.
func (h *ChargingPlaceHandler) List(w http.ResponseWriter, r *http.Request) {
	vehicleID := chi.URLParam(r, "vehicleId")
	v := requireVehicleAccess(w, r, h.repo, vehicleID, models.RoleViewer)
	if v == nil {
		return
	}
	places, err := h.repo.ListChargingPlaces(r.Context(), v.UserID)
	if err != nil {
		writeRepoError(w, r, err, "Failed to list charging places")
		return
	}
	locations, err := h.repo.ListChargingLocations(r.Context(), v.ID, v.UserID)
	if err != nil {
		writeRepoError(w, r, err, "Failed to list charging locations")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"places": places, "locations": locations})
}

func (h *ChargingPlaceHandler) Create(w http.ResponseWriter, r *http.Request) {
	v := requireVehicleAccess(w, r, h.repo, chi.URLParam(r, "vehicleId"), models.RoleEditor)
	if v == nil {
		return
	}
	var req chargingPlaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, apierror.New("request.invalid_body", "Invalid request body"))
		return
	}
	if err := req.validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	place, err := h.repo.CreateChargingPlace(r.Context(), v.UserID, req.Name, req.Category)
	if err != nil {
		h.writeError(w, r, err, "Failed to create charging place")
		return
	}
	writeJSON(w, http.StatusCreated, place)
}

func (h *ChargingPlaceHandler) Update(w http.ResponseWriter, r *http.Request) {
	v := requireVehicleAccess(w, r, h.repo, chi.URLParam(r, "vehicleId"), models.RoleEditor)
	if v == nil {
		return
	}
	var req chargingPlaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, apierror.New("request.invalid_body", "Invalid request body"))
		return
	}
	if err := req.validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if err := h.repo.UpdateChargingPlace(r.Context(), v.UserID, chi.URLParam(r, "placeId"), req.Name, req.Category); err != nil {
		h.writeError(w, r, err, "Failed to update charging place")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (h *ChargingPlaceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	v := requireVehicleAccess(w, r, h.repo, chi.URLParam(r, "vehicleId"), models.RoleEditor)
	if v == nil {
		return
	}
	if err := h.repo.DeleteChargingPlace(r.Context(), v.UserID, chi.URLParam(r, "placeId")); err != nil {
		h.writeError(w, r, err, "Failed to delete charging place")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

type chargingAliasRequest struct {
	Text    string  `json:"text"`
	PlaceID *string `json:"place_id"`
}

// SetAlias links a reported location text to a place, or unlinks it with a null place.
func (h *ChargingPlaceHandler) SetAlias(w http.ResponseWriter, r *http.Request) {
	v := requireVehicleAccess(w, r, h.repo, chi.URLParam(r, "vehicleId"), models.RoleEditor)
	if v == nil {
		return
	}
	var req chargingAliasRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, apierror.New("request.invalid_body", "Invalid request body"))
		return
	}
	if err := h.repo.SetChargingPlaceAlias(r.Context(), v.UserID, req.Text, req.PlaceID); err != nil {
		h.writeError(w, r, err, "Failed to link the location")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}
