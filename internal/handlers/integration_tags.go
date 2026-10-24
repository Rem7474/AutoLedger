package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/services"
)

// IntegrationTag is what the drives carrying one tag cost, to split professional from personal use.
type IntegrationTag struct {
	Tag          *string             `json:"tag" jsonschema:"the tag as the owner typed it, user text; null for the drives that carry no tag"`
	Distance     IntegrationDistance `json:"distance" jsonschema:"distance of the drives with this tag"`
	EnergyKwh    float64             `json:"energy_kwh" jsonschema:"electric energy those drives used, in kWh"`
	Tolls        IntegrationAmount   `json:"tolls" jsonschema:"tolls and parking attributed to those drives"`
	Cost         IntegrationAmount   `json:"cost" jsonschema:"allocated running cost of those drives (energy, wear, insurance share and tolls at the vehicle's unit rates); an allocation, not the cash paid"`
	CostPerKm    *IntegrationAmount  `json:"cost_per_km" jsonschema:"that cost divided by the distance; null while the distance is zero"`
	SharePct     float64             `json:"distance_share_pct" jsonschema:"share of the vehicle's tracked distance, in percent"`
	SharedDrives int                 `json:"shared_drives" jsonschema:"drives that carry several tags and so count in each of them: tag rows must not be added up when this is above zero"`
}

// IntegrationTags is the split of a vehicle's drives by tag.
type IntegrationTags struct {
	VehicleID string           `json:"vehicle_id" jsonschema:"the vehicle these figures belong to"`
	Currency  string           `json:"currency" jsonschema:"currency of every amount"`
	Tags      []IntegrationTag `json:"tags" jsonschema:"one row per tag over the whole history, plus one for the untagged drives; a drive with several tags appears in each"`
}

func buildIntegrationTags(vehicleID, currency string, rows []services.TagCostBreakdown) IntegrationTags {
	out := IntegrationTags{VehicleID: vehicleID, Currency: currency, Tags: []IntegrationTag{}}
	for _, r := range rows {
		item := IntegrationTag{
			Distance: kilometres(r.DistanceKm), EnergyKwh: r.EnergyKwh, Tolls: amountOf(r.TollsAmount, currency),
			Cost: amountOf(r.Cost, currency), SharePct: r.Percentage, SharedDrives: r.SharedDrives,
		}
		if r.Tag != "" {
			tag := r.Tag
			item.Tag = &tag
		}
		if r.DistanceKm > 0 {
			item.CostPerKm = decimalAmount(&r.CostPerKm, currency, 4)
		}
		out.Tags = append(out.Tags, item)
	}
	return out
}

// Tags splits a vehicle's drives by tag.
func (h *IntegrationReadHandler) Tags(ctx context.Context, userID, vehicleID string) (*IntegrationTags, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	sum, err := h.tcoService.ComputeVehicleTCO(ctx, vehicle.ID)
	if err != nil {
		return nil, err
	}
	out := buildIntegrationTags(vehicle.ID, vehicle.Currency, sum.TagBreakdown)
	return &out, nil
}

// GetTags serves Tags.
func (h *IntegrationReadHandler) GetTags(w http.ResponseWriter, r *http.Request) {
	out, err := h.Tags(r.Context(), middleware.GetUserID(r.Context()), chi.URLParam(r, "vehicleId"))
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to compute the tag breakdown")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
