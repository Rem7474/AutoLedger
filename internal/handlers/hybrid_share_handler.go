package handlers

import (
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

// Get returns the suggestion computed from the fill-ups and charges entered so far. It changes nothing.
func (h *HybridShareHandler) Get(w http.ResponseWriter, r *http.Request) {
	vehicleID := chi.URLParam(r, "vehicleId")
	v := requireVehicleAccess(w, r, h.repo, vehicleID, models.RoleViewer)
	if v == nil {
		return
	}
	if !v.CanRefuel() || !v.Capabilities.Charges {
		writeAPIError(w, http.StatusBadRequest, apierror.New("hybrid_share.not_hybrid", "The electric share only applies to vehicles that both charge and refuel"))
		return
	}
	fuel, err := h.repo.ListFuelLogs(r.Context(), vehicleID)
	if err != nil {
		writeRepoError(w, r, err, "Failed to list fill-ups")
		return
	}
	charges, _, err := h.repo.ListCharges(r.Context(), vehicleID, false, hybridChargePage, 0)
	if err != nil {
		writeRepoError(w, r, err, "Failed to list charges")
		return
	}
	readings, err := h.repo.ListOdometerCheckpoints(r.Context(), vehicleID)
	if err != nil {
		writeRepoError(w, r, err, "Failed to list odometer readings")
		return
	}
	ownership, err := h.repo.GetVehicleOwnership(r.Context(), vehicleID)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		writeRepoError(w, r, err, "Failed to load ownership")
		return
	}
	writeJSON(w, http.StatusOK, services.ComputeHybridShare(services.HybridShareInput{
		Fuel:            fuel,
		Charges:         charges,
		Refs:            services.BuildOdometerRefs(readings, ownership),
		ReferenceKwh100: v.EstimatedKwh100km,
		Now:             time.Now(),
	}))
}
