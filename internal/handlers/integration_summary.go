package handlers

import (
	"context"
	"math"
	"net/http"
	"sync"
	"time"

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

	// LastCharge and CostPer100km are what the metrics route of the Home Assistant integration reports for one vehicle.
	LastCharge   *IntegrationLastCharge `json:"last_charge" jsonschema:"the latest charging session; null for a vehicle that cannot be plugged in, or without any charge"`
	CostPer100km *IntegrationAmount     `json:"cost_per_100km" jsonschema:"energy cost per 100 km, the figure the dashboard shows; for a vehicle that can be plugged in the estimate from its profile stands in while no distance is tracked; null when unknown"`
	// MonthCosts is the split of MonthCost: its seven categories add up to the same total.
	MonthCosts IntegrationCostBreakdown `json:"month_costs" jsonschema:"the month in progress by category, cash spent plus the instalments that fall due later in it; the total is month_cost"`

	MonthDistance       IntegrationDistance `json:"month_distance" jsonschema:"distance driven in the month in progress, tracked plus estimated for the untracked part; 0 before the first drive of the month"`
	CarpoolRevenueMonth IntegrationAmount   `json:"carpool_revenue_month" jsonschema:"what the passengers paid for the carpools dated this month; 0 without any"`
	MonthNet            IntegrationAmount   `json:"month_net" jsonschema:"month_cost minus carpool_revenue_month: what the month costs once the passengers have paid their share; equal to month_cost without carpooling"`
	Last12MonthsCost    IntegrationAmount   `json:"last_12_months_cost" jsonschema:"cash total of the last twelve months, the figure behind cost_per_km; smaller than a year of cost while the history is shorter than twelve months"`

	// The sections below are present only when the request asks for them with include, and absent when they do not
	// apply to the vehicle (no mounted tire, not an electric vehicle, no data).
	Tires     *IntegrationSummaryTires     `json:"tires,omitempty" jsonschema:"include=tires: the wear of the mounted tires"`
	Battery   *IntegrationBatteryHealth    `json:"battery,omitempty" jsonschema:"include=battery: the state of health of the traction battery"`
	Ownership *IntegrationSummaryOwnership `json:"ownership,omitempty" jsonschema:"include=ownership: the cost of owning the vehicle, purchase included, and its value"`
	Sources   *IntegrationSummarySources   `json:"sources,omitempty" jsonschema:"include=sources: how fresh the data is"`

	CompletenessPct *int `json:"completeness_pct" jsonschema:"0 to 100, how much of the figures rests on recorded data; null when no score is available"`
	DataGaps        int  `json:"data_gaps" jsonschema:"number of things known to be missing from the records that make the figures less reliable; the data-gaps route lists them"`
}

// IntegrationLastCharge is the latest charging session of a vehicle.
type IntegrationLastCharge struct {
	Date            string             `json:"date" jsonschema:"when the session started, RFC 3339"`
	Cost            *IntegrationAmount `json:"cost" jsonschema:"what the session cost; null while its cost is not recorded, never zero for an unknown cost"`
	EnergyKwh       float64            `json:"energy_kwh" jsonschema:"energy added to the battery, in kWh"`
	DurationMinutes *int               `json:"duration_minutes" jsonschema:"length of the session; null when its end is not recorded"`
	PricePerKwh     *IntegrationAmount `json:"price_per_kwh" jsonschema:"what the session cost per kWh added; null when its cost or its energy is not recorded"`
}

// IntegrationSummary is the summary of every vehicle the token sees.
type IntegrationSummary struct {
	Vehicles []IntegrationSummaryVehicle `json:"vehicles" jsonschema:"one entry per vehicle, in the order of the vehicle list"`
}

// summaryLastCharge reads the latest charge of a vehicle as the metrics route does.
func summaryLastCharge(charge *models.ChargeLog, currency string) *IntegrationLastCharge {
	if charge == nil {
		return nil
	}
	out := &IntegrationLastCharge{Date: charge.Date.Format(time.RFC3339), EnergyKwh: charge.KwhAdded}
	if charge.Cost != nil {
		cost := amountOf(*charge.Cost, currency)
		out.Cost = &cost
		if charge.KwhAdded > 0 {
			price := charge.Cost.Float() / charge.KwhAdded
			out.PricePerKwh = decimalAmount(&price, currency, 4)
		}
	}
	if charge.EndDate != nil {
		minutes := max(int(charge.EndDate.Sub(charge.Date).Minutes()), 0)
		out.DurationMinutes = &minutes
	}
	return out
}

// summaryCostPer100km is the energy cost per 100 km of the vehicle: the electric statistics figure, the profile
// estimate while the statistics have no tracked distance, and the fuel figure of a vehicle that only burns fuel.
func summaryCostPer100km(vehicle *models.Vehicle, energy IntegrationEnergyEfficiency) *IntegrationAmount {
	if energy.Electric != nil && energy.Electric.CostPer100km != nil {
		return energy.Electric.CostPer100km
	}
	if vehicle.CanCharge() && vehicle.EstimatedKwh100km != nil && vehicle.EstimatedPricePerKwh != nil {
		estimate := math.Round(*vehicle.EstimatedKwh100km**vehicle.EstimatedPricePerKwh*100) / 100
		return decimalAmount(&estimate, vehicle.Currency, 2)
	}
	if energy.Fuel != nil {
		return energy.Fuel.CostPer100km
	}
	return nil
}

func buildIntegrationSummaryVehicle(vehicle *models.Vehicle, sum *services.TCOSummary, stats *services.EnergyStats, lastCharge *models.ChargeLog) IntegrationSummaryVehicle {
	costs := buildIntegrationCosts(vehicle, sum)
	out := IntegrationSummaryVehicle{
		VehicleID: vehicle.ID, Name: vehicle.Name, Powertrain: vehicle.Powertrain, Currency: vehicle.Currency,
		Month: sum.CurrentMonth, MonthCost: amountOf(0, vehicle.Currency), CostPerKm: costs.Last12Months.CostPerKm,
		MonthCosts:    breakdown(vehicle.Currency, 0, 0, 0, 0, 0, 0, 0, 0),
		MonthDistance: kilometres(0), CarpoolRevenueMonth: amountOf(0, vehicle.Currency), MonthNet: amountOf(0, vehicle.Currency),
		Last12MonthsCost: costs.Last12Months.Costs.Total,
		LastCharge:       summaryLastCharge(lastCharge, vehicle.Currency),
		CompletenessPct:  costs.Completeness.ScorePct, DataGaps: len(costs.Completeness.WarningDetails),
	}
	for _, m := range sum.MonthlyCosts {
		if m.Month == sum.CurrentMonth {
			out.MonthCost = amountOf(m.Total, vehicle.Currency)
			out.MonthDistance = kilometres(m.DistanceKm)
			out.CarpoolRevenueMonth = amountOf(m.CarpoolRevenue, vehicle.Currency)
			out.MonthNet = amountOf(m.NetTotal, vehicle.Currency)
			out.MonthCosts = breakdown(vehicle.Currency, m.Energy, m.Tolls, m.Maintenance, m.Insurance, m.Financing, m.Tires, m.Other, m.Total)
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
	out.CostPer100km = summaryCostPer100km(vehicle, energy)
	return out
}

// Summary gives the figures of every vehicle the token sees, within the vehicles it is limited to.
func (h *IntegrationReadHandler) Summary(ctx context.Context, userID string, include SummaryInclude) (*IntegrationSummary, error) {
	vehicles, err := h.repo.ListVehiclesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	allowed := make([]*models.Vehicle, 0, len(vehicles))
	for i := range vehicles {
		if middleware.VehicleAllowed(ctx, vehicles[i].ID) {
			allowed = append(allowed, &vehicles[i])
		}
	}
	pending := 0
	if include.Sources {
		charges, err := h.repo.ListPendingCharges(ctx, userID)
		if err != nil {
			return nil, err
		}
		pending = len(charges)
	}
	// The vehicles are independent: a few are computed at once, each writing to its own slot
	rows := make([]IntegrationSummaryVehicle, len(allowed))
	errs := make([]error, len(allowed))
	var wg sync.WaitGroup
	slots := make(chan struct{}, summaryConcurrency)
	for i := range allowed {
		slots <- struct{}{}
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			defer func() { <-slots }()
			rows[i], errs[i] = h.summaryVehicle(ctx, allowed[i], include, pending)
		}(i)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return &IntegrationSummary{Vehicles: append([]IntegrationSummaryVehicle{}, rows...)}, nil
}

// summaryConcurrency bounds the vehicles computed at once so the summary does not take every connection of the pool.
const summaryConcurrency = 4

// summaryVehicle computes the summary entry of one vehicle.
func (h *IntegrationReadHandler) summaryVehicle(ctx context.Context, v *models.Vehicle, include SummaryInclude, pending int) (IntegrationSummaryVehicle, error) {
	sum, err := h.tcoService.ComputeVehicleTCO(ctx, v.ID)
	if err != nil {
		return IntegrationSummaryVehicle{}, err
	}
	var stats *services.EnergyStats
	var lastCharge *models.ChargeLog
	if v.CanCharge() {
		if h.energy != nil {
			if stats, err = h.energy.Compute(ctx, v.ID); err != nil {
				return IntegrationSummaryVehicle{}, err
			}
		}
		charges, _, err := h.repo.ListCharges(ctx, v.ID, false, 1, 0)
		if err != nil {
			return IntegrationSummaryVehicle{}, err
		}
		if len(charges) > 0 {
			lastCharge = &charges[0]
		}
	}
	out := buildIntegrationSummaryVehicle(v, sum, stats, lastCharge)
	if err := h.addSummarySections(ctx, &out, v, sum, stats, include, pending); err != nil {
		return IntegrationSummaryVehicle{}, err
	}
	return out, nil
}

// GetSummary serves Summary.
func (h *IntegrationReadHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	include, ok := summaryInclude(w, r)
	if !ok {
		return
	}
	out, err := h.Summary(r.Context(), middleware.GetUserID(r.Context()), include)
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to compute the summary")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
