package handlers

import (
	"context"
	"net/http"

	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/services"
)

// IntegrationSummaryVehicle is the handful of figures a sensor shows for a vehicle, so a client refreshes every
// vehicle with one call instead of one call per figure and per vehicle.
type IntegrationSummaryVehicle struct {
	VehicleID  string `json:"vehicle_id" jsonschema:"the vehicle these figures belong to"`
	Name       string `json:"name" jsonschema:"name the owner gave the vehicle, user text"`
	Powertrain string `json:"powertrain" jsonschema:"EV, ICE, PHEV or REEV"`
	Currency   string `json:"currency" jsonschema:"currency of every amount of this vehicle"`

	Month     string            `json:"month" jsonschema:"the calendar month in progress in the account's reporting timezone, YYYY-MM"`
	MonthCost IntegrationAmount `json:"month_cost" jsonschema:"what the vehicle cost this month so far, with the instalments that fall due later in it; the month is not final"`
	// CostPerKm is the cash total over the last twelve months divided by the distance, as get_vehicle_costs reports it.
	CostPerKm *IntegrationAmount `json:"cost_per_km" jsonschema:"total cost per kilometre over the last twelve months; null while the distance is unknown"`

	ConsumptionKwh100km *float64 `json:"consumption_kwh_100km" jsonschema:"energy used per 100 km; null for a vehicle that cannot be plugged in, or without enough data"`
	ConsumptionL100km   *float64 `json:"consumption_l_100km" jsonschema:"fuel used per 100 km between full tanks; null for a vehicle that does not burn fuel, or without two consecutive full tanks"`

	CompletenessPct *int `json:"completeness_pct" jsonschema:"0 to 100, how much of the figures rests on recorded data; null when no score is available"`
	DataGaps        int  `json:"data_gaps" jsonschema:"number of things known to be missing from the records that make the figures less reliable; the data-gaps route lists them"`
}

// IntegrationSummary is the summary of every vehicle the token sees.
type IntegrationSummary struct {
	Vehicles []IntegrationSummaryVehicle `json:"vehicles" jsonschema:"one entry per vehicle, in the order of the vehicle list"`
}

func buildIntegrationSummaryVehicle(vehicle *models.Vehicle, sum *services.TCOSummary, stats *services.EnergyStats) IntegrationSummaryVehicle {
	costs := buildIntegrationCosts(vehicle, sum)
	out := IntegrationSummaryVehicle{
		VehicleID: vehicle.ID, Name: vehicle.Name, Powertrain: vehicle.Powertrain, Currency: vehicle.Currency,
		Month: sum.CurrentMonth, MonthCost: amountOf(0, vehicle.Currency), CostPerKm: costs.Last12Months.CostPerKm,
		CompletenessPct: costs.Completeness.ScorePct, DataGaps: len(costs.Completeness.WarningDetails),
	}
	for _, m := range sum.MonthlyCosts {
		if m.Month == sum.CurrentMonth {
			out.MonthCost = amountOf(m.Total, vehicle.Currency)
		}
	}
	var fuelSum *services.TCOSummary
	if models.PowertrainCanRefuel(vehicle.Powertrain) {
		fuelSum = sum
	}
	energy := buildIntegrationEnergy(vehicle, stats, fuelSum)
	if energy.Electric != nil {
		out.ConsumptionKwh100km = energy.Electric.Kwh100km
	}
	if energy.Fuel != nil {
		out.ConsumptionL100km = energy.Fuel.LitresPer100
	}
	return out
}

// Summary gives the figures of every vehicle the token sees, within the vehicles it is limited to.
func (h *IntegrationReadHandler) Summary(ctx context.Context, userID string) (*IntegrationSummary, error) {
	vehicles, err := h.repo.ListVehiclesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := &IntegrationSummary{Vehicles: []IntegrationSummaryVehicle{}}
	for i := range vehicles {
		v := &vehicles[i]
		if !middleware.VehicleAllowed(ctx, v.ID) {
			continue
		}
		sum, err := h.tcoService.ComputeVehicleTCO(ctx, v.ID)
		if err != nil {
			return nil, err
		}
		var stats *services.EnergyStats
		if v.CanCharge() && h.energy != nil {
			if stats, err = h.energy.Compute(ctx, v.ID); err != nil {
				return nil, err
			}
		}
		out.Vehicles = append(out.Vehicles, buildIntegrationSummaryVehicle(v, sum, stats))
	}
	return out, nil
}

// GetSummary serves Summary.
func (h *IntegrationReadHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	out, err := h.Summary(r.Context(), middleware.GetUserID(r.Context()))
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to compute the summary")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
