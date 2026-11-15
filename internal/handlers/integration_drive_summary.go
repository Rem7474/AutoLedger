package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/services"
)

// IntegrationDriveSummary is what a vehicle's drives amount to over a period. It names no driver, address or drive.
type IntegrationDriveSummary struct {
	VehicleID string `json:"vehicle_id" jsonschema:"the vehicle these figures belong to"`
	Currency  string `json:"currency" jsonschema:"currency of every amount"`
	From      string `json:"from" jsonschema:"first day counted, YYYY-MM-DD (UTC); empty when the period has no start"`
	To        string `json:"to" jsonschema:"last day counted, YYYY-MM-DD (UTC); empty when the period has no end"`

	Drives                int                 `json:"drives" jsonschema:"number of drives that began in the period"`
	DrivesWithoutDistance int                 `json:"drives_without_distance" jsonschema:"drives whose distance was not recorded: they are counted in drives but add nothing to distance, so cost_per_km covers only the others"`
	Distance              IntegrationDistance `json:"distance" jsonschema:"distance of the drives that have one"`
	EnergyKwh             *float64            `json:"energy_kwh" jsonschema:"electric energy those drives used, in kWh; null for a vehicle that burns fuel only (see fuel_liters)"`
	FuelLiters            *float64            `json:"fuel_liters" jsonschema:"fuel those drives used, in litres, for a vehicle that burns fuel only (from the consumption its fill-ups gave before each drive; drives with no known consumption are not counted); null for any other vehicle and while no consumption is known"`
	Tolls                 IntegrationAmount   `json:"tolls" jsonschema:"tolls and parking attributed to those drives"`
	Cost                  IntegrationAmount   `json:"cost" jsonschema:"allocated running cost of those drives (energy, wear, insurance share and tolls at the vehicle's unit rates); an allocation, not the cash paid"`
	CostPerKm             *IntegrationAmount  `json:"cost_per_km" jsonschema:"that cost divided by the distance; null while the distance is zero"`
	UntaggedDrives        int                 `json:"untagged_drives" jsonschema:"drives that carry no tag, so are in neither the professional nor the personal share"`
	DrivesToQualify       int                 `json:"drives_to_qualify" jsonschema:"motorway-type drives with no toll entered and not reviewed as having none: their toll cost may be missing"`
	TripGroups            int                 `json:"trip_groups" jsonschema:"trip groups with at least one drive in the period"`
	Tags                  []IntegrationTag    `json:"tags" jsonschema:"the same split by tag as get_tag_breakdown, over the period only; a drive with several tags appears in each, so do not add the rows up when shared_drives is above zero"`
}

func buildIntegrationDriveSummary(vehicleID, currency string, fuelOnly bool, from, to string, sum services.DriveSummary) IntegrationDriveSummary {
	out := IntegrationDriveSummary{
		VehicleID: vehicleID, Currency: currency, From: from, To: to,
		Drives: sum.Drives, DrivesWithoutDistance: sum.DrivesWithoutDistance, Distance: kilometres(sum.DistanceKm),
		Tolls: amountOf(sum.TollsAmount, currency), Cost: amountOf(sum.Cost, currency),
		UntaggedDrives: sum.UntaggedDrives, DrivesToQualify: sum.ToQualify, TripGroups: sum.TripGroups,
		Tags: buildIntegrationTags(vehicleID, currency, fuelOnly, sum.ByTag).Tags,
	}
	out.EnergyKwh, out.FuelLiters = energyFigures(fuelOnly, sum.EnergyKwh, sum.FuelLiters)
	if sum.DistanceKm > 0 {
		out.CostPerKm = decimalAmount(&sum.CostPerKm, currency, 4)
	}
	return out
}

// DriveSummary sums a vehicle's drives over a period (From and To are days, YYYY-MM-DD, both optional).
func (h *IntegrationReadHandler) DriveSummary(ctx context.Context, userID, vehicleID, from, to string) (*IntegrationDriveSummary, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	start, err := tripBound(from, false)
	if err != nil {
		return nil, err
	}
	end, err := tripBound(to, true)
	if err != nil {
		return nil, err
	}
	sum, err := h.tcoService.DriveSummaryFor(ctx, vehicle.ID, start, end)
	if err != nil {
		return nil, err
	}
	out := buildIntegrationDriveSummary(vehicle.ID, vehicle.Currency, models.PowertrainIsFuelOnly(vehicle.Powertrain), from, to, *sum)
	return &out, nil
}

// GetDriveSummary serves DriveSummary.
func (h *IntegrationReadHandler) GetDriveSummary(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	out, err := h.DriveSummary(r.Context(), middleware.GetUserID(r.Context()), chi.URLParam(r, "vehicleId"), query.Get("from"), query.Get("to"))
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to summarize the drives")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// energyFigures is the energy of drives as a client reads it: the kWh for a vehicle that charges (never null there,
// 0 when none was measured), the litres for one that burns fuel only, the other figure being null because it does not
// apply. Litres that are not known yet are null, not 0.
func energyFigures(fuelOnly bool, kwh, litres float64) (*float64, *float64) {
	if !fuelOnly {
		return &kwh, nil
	}
	if litres <= 0 {
		return nil, nil
	}
	return nil, &litres
}
