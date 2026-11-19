package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/services"
)

// HybridShareHandler serves the electric share suggested for a plug-in hybrid or range extender.
type HybridShareHandler struct {
	repo *database.Repository
}

func NewHybridShareHandler(repo *database.Repository) *HybridShareHandler {
	return &HybridShareHandler{repo: repo}
}

// hybridChargePage is large enough for any history: the suggestion reads every charge of the vehicle.
const hybridChargePage = 100000

// requireHybrid loads the vehicle and refuses one that does not both charge and refuel.
func (h *HybridShareHandler) requireHybrid(w http.ResponseWriter, r *http.Request, minRole models.VehicleRole) *models.Vehicle {
	vehicleID := chi.URLParam(r, "vehicleId")
	v := requireVehicleAccess(w, r, h.repo, vehicleID, minRole)
	if v == nil {
		return nil
	}
	if !v.CanRefuel() || !v.Capabilities.Charges {
		writeAPIError(w, http.StatusBadRequest, apierror.New("hybrid_share.not_hybrid", "The electric share only applies to vehicles that both charge and refuel"))
		return nil
	}
	return v
}

// suggestion computes the proposal from what the owner typed. It writes the error response and returns false when
// the history cannot be read.
func (h *HybridShareHandler) suggestion(w http.ResponseWriter, r *http.Request, v *models.Vehicle) (services.HybridShareResult, bool) {
	vehicleID := v.ID
	fuel, err := h.repo.ListFuelLogs(r.Context(), vehicleID)
	if err != nil {
		writeRepoError(w, r, err, "Failed to list fill-ups")
		return services.HybridShareResult{}, false
	}
	charges, _, err := h.repo.ListCharges(r.Context(), vehicleID, false, hybridChargePage, 0)
	if err != nil {
		writeRepoError(w, r, err, "Failed to list charges")
		return services.HybridShareResult{}, false
	}
	readings, err := h.repo.ListOdometerCheckpoints(r.Context(), vehicleID)
	if err != nil {
		writeRepoError(w, r, err, "Failed to list odometer readings")
		return services.HybridShareResult{}, false
	}
	ownership, err := h.repo.GetVehicleOwnership(r.Context(), vehicleID)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		writeRepoError(w, r, err, "Failed to load ownership")
		return services.HybridShareResult{}, false
	}
	return services.ComputeHybridShare(services.HybridShareInput{
		Fuel:            fuel,
		Charges:         charges,
		Refs:            services.BuildOdometerRefs(readings, ownership),
		ReferenceKwh100: v.EstimatedKwh100km,
		BatteryKwh:      v.BatteryUsableKwh,
		Now:             time.Now(),
	}), true
}

// hybridSettingsView is what the owner accepted, next to the suggestion.
type hybridSettingsView struct {
	ElectricShare *float64 `json:"electric_share"`
	FuelL100km    *float64 `json:"fuel_l_100km"`
	BatteryKwh    *float64 `json:"battery_usable_kwh"`
}

// hybridShareResponse is the suggestion with what is accepted today.
type hybridShareResponse struct {
	services.HybridShareResult
	Accepted hybridSettingsView `json:"accepted"`
}

func acceptedOf(v *models.Vehicle) hybridSettingsView {
	return hybridSettingsView{ElectricShare: v.HybridElectricShare, FuelL100km: v.HybridFuelL100km, BatteryKwh: v.BatteryUsableKwh}
}

// Get returns the suggestion computed from the fill-ups and charges entered so far, and what is accepted. It changes
// nothing.
func (h *HybridShareHandler) Get(w http.ResponseWriter, r *http.Request) {
	v := h.requireHybrid(w, r, models.RoleViewer)
	if v == nil {
		return
	}
	res, ok := h.suggestion(w, r, v)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, hybridShareResponse{HybridShareResult: res, Accepted: acceptedOf(v)})
}

// SaveHybridShareRequest is the owner's decision: the values are replaced as sent, null clearing one. With
// accept_suggestion the share and the fuel consumption are those of the current suggestion.
type SaveHybridShareRequest struct {
	AcceptSuggestion bool     `json:"accept_suggestion"`
	ElectricShare    *float64 `json:"electric_share"`
	FuelL100km       *float64 `json:"fuel_l_100km"`
	BatteryKwh       *float64 `json:"battery_usable_kwh"`
}

const (
	maxHybridFuelL100km = 50.0
	maxBatteryKwh       = 500.0
)

// Save stores what the owner accepts, types or clears. It never runs by itself: a suggestion stays a suggestion
// until this is called, and what is stored stays as it is until it is called again.
func (h *HybridShareHandler) Save(w http.ResponseWriter, r *http.Request) {
	v := h.requireHybrid(w, r, models.RoleEditor)
	if v == nil {
		return
	}
	var req SaveHybridShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, apierror.New("request.invalid_body", "Invalid request body"))
		return
	}
	settings := database.HybridSettings{ElectricShare: req.ElectricShare, FuelL100km: req.FuelL100km, BatteryKwh: req.BatteryKwh}
	if req.AcceptSuggestion {
		v.BatteryUsableKwh = req.BatteryKwh // the capacity correction the owner typed counts in the suggestion
		res, ok := h.suggestion(w, r, v)
		if !ok {
			return
		}
		if res.Suggested == nil {
			writeAPIError(w, http.StatusConflict, apierror.New("hybrid_share.no_suggestion", "There is not enough history to suggest an electric share"))
			return
		}
		settings.ElectricShare, settings.FuelL100km = res.Suggested, res.FuelL100kmFuelPart
	}
	if err := validateHybridSettings(settings); err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	if settings.ElectricShare == nil {
		settings.FuelL100km = nil // a consumption without its share means nothing
	}
	if err := h.repo.SetHybridSettings(r.Context(), v.ID, settings); err != nil {
		writeRepoError(w, r, err, "Failed to save the electric share")
		return
	}
	writeJSON(w, http.StatusOK, hybridSettingsView{ElectricShare: settings.ElectricShare, FuelL100km: settings.FuelL100km, BatteryKwh: settings.BatteryKwh})
}

func validateHybridSettings(s database.HybridSettings) *apierror.Error {
	if s.ElectricShare != nil && (*s.ElectricShare < 0 || *s.ElectricShare > 1) {
		return apierror.New("hybrid_share.share_range", "The electric share must be between 0 and 1")
	}
	if s.FuelL100km != nil && (*s.FuelL100km <= 0 || *s.FuelL100km > maxHybridFuelL100km) {
		return apierror.New("hybrid_share.fuel_range", "Invalid fuel consumption")
	}
	if s.BatteryKwh != nil && (*s.BatteryKwh <= 0 || *s.BatteryKwh > maxBatteryKwh) {
		return apierror.New("hybrid_share.battery_range", "Invalid battery capacity")
	}
	return nil
}
