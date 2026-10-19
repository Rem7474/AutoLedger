package handlers

import (
	"context"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/services"
)

// decimalAmount is a price or a rate in the vehicle's currency (per kWh, per litre, per 100 km...), null when unknown.
func decimalAmount(v *float64, currency string, decimals int) *IntegrationAmount {
	if v == nil {
		return nil
	}
	return &IntegrationAmount{Amount: strconv.FormatFloat(*v, 'f', decimals, 64), Currency: currency}
}

// IntegrationChargeClass is what one kind of charging cost and delivered.
type IntegrationChargeClass struct {
	Class             string             `json:"class" jsonschema:"slow (domestic socket), ac (wallbox or public AC post), dc (fast charging) or unknown (no usable duration, such as a manual entry); sessions are classed by their average power"`
	Sessions          int                `json:"sessions" jsonschema:"number of charging sessions of this class"`
	KwhAdded          float64            `json:"kwh_added" jsonschema:"energy stored in the battery by these sessions"`
	EnergyCost        IntegrationAmount  `json:"energy_cost" jsonschema:"what these sessions cost; sessions without a price are not counted"`
	PricePerKwh       *IntegrationAmount `json:"price_per_kwh" jsonschema:"average price paid per kWh over the priced sessions; null when none is priced"`
	ChargeEfficiency  *float64           `json:"charge_efficiency_pct" jsonschema:"percentage of the energy drawn from the grid that reached the battery; null when the grid energy was not measured"`
	CostPerFullCharge *IntegrationAmount `json:"cost_per_full_charge" jsonschema:"what charging from 0 to 100 percent costs with this kind of charging; null when the sessions do not allow the estimate"`
	WithoutCost       int                `json:"sessions_without_cost" jsonschema:"sessions left out of the price because no cost is recorded"`
}

// IntegrationTemperature compares the consumption in cold and mild weather.
type IntegrationTemperature struct {
	ColdKwh100km *float64           `json:"cold_consumption_kwh_100km" jsonschema:"consumption below 5 degrees Celsius, in kWh per 100 km; null without enough drives"`
	MildKwh100km *float64           `json:"mild_consumption_kwh_100km" jsonschema:"consumption between 15 and 25 degrees Celsius, in kWh per 100 km; null without enough drives"`
	ExtraPercent *float64           `json:"extra_percent" jsonschema:"how much more energy the cold takes than mild weather, in percent; null when either side is unknown"`
	ExtraCost    *IntegrationAmount `json:"extra_cost_per_100km" jsonschema:"cost of that extra energy per 100 km at the average price paid per kWh; null when unknown"`
}

// IntegrationBatteryHealth is the state of health of the traction battery, last reading and its change over a year.
type IntegrationBatteryHealth struct {
	LatestPercent    float64  `json:"latest_percent" jsonschema:"latest state of health, in percent of the capacity of the new battery"`
	LatestDate       string   `json:"latest_date" jsonschema:"day of that reading, YYYY-MM-DD"`
	ChangePoints12m  *float64 `json:"change_points_12m" jsonschema:"change in percentage points over the 12 months before the latest reading, negative when the battery lost capacity; null when the history does not reach back that far or holds a single reading"`
	CapacityKwh      *float64 `json:"current_capacity_kwh" jsonschema:"usable capacity at the latest reading, in kWh; null when not reported"`
	NewCapacityKwh   *float64 `json:"max_capacity_kwh" jsonschema:"capacity of the new battery, in kWh; null when not reported"`
	ReadingsRecorded int      `json:"readings_recorded" jsonschema:"number of daily readings the history holds; only the latest and the change are returned"`
}

// IntegrationElectricEfficiency is how an electric vehicle uses and pays for its energy.
type IntegrationElectricEfficiency struct {
	Kwh100km      *float64                  `json:"consumption_kwh_100km" jsonschema:"energy used per 100 km; measured when the drives report their energy, otherwise derived from the charges and the odometer (see basis)"`
	PricePerKwh   *IntegrationAmount        `json:"price_per_kwh" jsonschema:"average price paid per kWh over the priced charges"`
	CostPer100km  *IntegrationAmount        `json:"cost_per_100km" jsonschema:"energy cost per 100 km; null while the distance is too short to be meaningful"`
	Efficiency    *float64                  `json:"charge_efficiency_pct" jsonschema:"percentage of the energy drawn from the grid that reached the battery; null when not measured"`
	FullCharge    *IntegrationAmount        `json:"cost_per_full_charge" jsonschema:"what charging from 0 to 100 percent costs on average; null when the sessions do not allow the estimate"`
	Capacity      *float64                  `json:"estimated_usable_capacity_kwh" jsonschema:"usable battery capacity estimated from the latest charging sessions, in kWh; null without enough sessions"`
	WithoutCost   int                       `json:"sessions_without_cost" jsonschema:"charging sessions whose cost is not recorded, left out of the prices"`
	ChargeMix     []IntegrationChargeClass  `json:"charge_mix" jsonschema:"charging by kind: slow, ac, dc; unknown for sessions that cannot be classed"`
	Temperature   *IntegrationTemperature   `json:"temperature" jsonschema:"effect of cold weather on the consumption; null without drives that report their outside temperature"`
	BatteryHealth *IntegrationBatteryHealth `json:"battery_health" jsonschema:"state of health of the battery; null when no reading exists"`
	Basis         map[string]string         `json:"basis" jsonschema:"for each section (consumption, cost, charging, full_charge, temperature, battery), whether its figures are measured, derived from the odometer and the charges, or unavailable"`
}

// IntegrationFuelEfficiency is how a vehicle that burns fuel uses and pays for it.
type IntegrationFuelEfficiency struct {
	FillUps       int                `json:"fill_ups" jsonschema:"number of recorded fill-ups"`
	Litres        float64            `json:"litres" jsonschema:"fuel bought over the whole history, in litres"`
	PricePerLitre *IntegrationAmount `json:"price_per_litre" jsonschema:"average price paid per litre; null without a priced fill-up"`
	LitresPer100  *float64           `json:"consumption_l_100km" jsonschema:"consumption measured between full tanks, in litres per 100 km; null without two consecutive full tanks"`
	CostPer100km  *IntegrationAmount `json:"cost_per_100km" jsonschema:"fuel cost per 100 km; null while the distance is unknown"`
}

// IntegrationEnergyEfficiency is the efficiency of a vehicle's energy: the electric part, the fuel part, or both
// for a plug-in hybrid.
type IntegrationEnergyEfficiency struct {
	VehicleID  string                         `json:"vehicle_id" jsonschema:"the vehicle these figures belong to"`
	Currency   string                         `json:"currency" jsonschema:"currency of every amount"`
	Powertrain string                         `json:"powertrain" jsonschema:"EV, ICE, PHEV or REEV"`
	Electric   *IntegrationElectricEfficiency `json:"electric" jsonschema:"the electric side; null for a vehicle that cannot be plugged in"`
	Fuel       *IntegrationFuelEfficiency     `json:"fuel" jsonschema:"the fuel side; null for a vehicle that does not burn fuel"`
}

// batteryReferenceTolerance is how far from exactly a year before the latest reading the reference reading may be.
const batteryReferenceTolerance = 31 * 24 * time.Hour

func chargeClassName(class string) string { return strings.ToLower(class) }

func roundTo(v float64, decimals int) float64 {
	p := math.Pow(10, float64(decimals))
	return math.Round(v*p) / p
}

func percentOf(ratio *float64) *float64 {
	if ratio == nil {
		return nil
	}
	p := roundTo(*ratio*100, 1)
	return &p
}

func buildIntegrationBattery(readings []models.BatterySnapshot) *IntegrationBatteryHealth {
	var withHealth []models.BatterySnapshot
	for _, r := range readings {
		if r.HealthPercent != nil {
			withHealth = append(withHealth, r)
		}
	}
	if len(withHealth) == 0 {
		return nil
	}
	sort.Slice(withHealth, func(i, j int) bool { return withHealth[i].Date < withHealth[j].Date })
	latest := withHealth[len(withHealth)-1]
	out := &IntegrationBatteryHealth{
		LatestPercent: *latest.HealthPercent, LatestDate: latest.Date,
		CapacityKwh: latest.CurrentCapacityKwh, NewCapacityKwh: latest.MaxCapacityKwh, ReadingsRecorded: len(readings),
	}
	end, err := time.Parse("2006-01-02", latest.Date)
	if err != nil {
		return out
	}
	// The reference is the reading closest to a year before the latest one, accepted within a month of it: a shorter
	// history does not give a year-long change.
	target := end.AddDate(-1, 0, 0)
	var best *models.BatterySnapshot
	bestGap := batteryReferenceTolerance
	for i := range withHealth[:len(withHealth)-1] {
		day, err := time.Parse("2006-01-02", withHealth[i].Date)
		if err != nil {
			continue
		}
		if gap := time.Duration(math.Abs(float64(day.Sub(target)))); gap <= bestGap {
			best, bestGap = &withHealth[i], gap
		}
	}
	if best != nil {
		change := roundTo(*latest.HealthPercent-*best.HealthPercent, 1)
		out.ChangePoints12m = &change
	}
	return out
}

func buildIntegrationElectric(cur string, stats *services.EnergyStats) *IntegrationElectricEfficiency {
	s := stats.Summary
	out := &IntegrationElectricEfficiency{
		Kwh100km: s.ConsumptionKwh100km, PricePerKwh: decimalAmount(s.PricePerKwh, cur, 3), CostPer100km: decimalAmount(s.CostPer100km, cur, 2),
		Efficiency: percentOf(s.ChargeEfficiency), FullCharge: decimalAmount(s.CostPerFullCharge, cur, 2), Capacity: s.EstimatedCapacityKwh,
		WithoutCost: s.SessionsWithoutCost, ChargeMix: []IntegrationChargeClass{},
		BatteryHealth: buildIntegrationBattery(stats.BatteryHealth),
		Basis: map[string]string{
			"consumption": stats.Basis.Consumption, "cost": stats.Basis.Cost, "charging": stats.Basis.Charging,
			"full_charge": stats.Basis.FullCharge, "temperature": stats.Basis.Temperature, "battery": stats.Basis.Battery,
		},
	}
	for _, c := range stats.ChargeClasses {
		out.ChargeMix = append(out.ChargeMix, IntegrationChargeClass{
			Class: chargeClassName(c.Class), Sessions: c.Sessions, KwhAdded: c.KwhAdded, EnergyCost: amountOf(c.EnergyCost, cur),
			PricePerKwh: decimalAmount(c.PricePerKwh, cur, 3), ChargeEfficiency: percentOf(c.ChargeEfficiency),
			CostPerFullCharge: decimalAmount(c.CostPerFullCharge, cur, 2), WithoutCost: c.SessionsWithoutCost,
		})
	}
	t := stats.Temperature
	if t.ColdConsumptionKwh100km != nil || t.MildConsumptionKwh100km != nil {
		out.Temperature = &IntegrationTemperature{
			ColdKwh100km: t.ColdConsumptionKwh100km, MildKwh100km: t.MildConsumptionKwh100km,
			ExtraPercent: t.ExtraPercent, ExtraCost: decimalAmount(t.ExtraCostPer100km, cur, 2),
		}
	}
	return out
}

func buildIntegrationFuel(cur string, sum *services.TCOSummary) *IntegrationFuelEfficiency {
	out := &IntegrationFuelEfficiency{FillUps: sum.FuelFillUps, Litres: sum.TotalLiters, LitresPer100: sum.ConsumptionL100km}
	if sum.AvgCostPerLiter > 0 {
		out.PricePerLitre = decimalAmount(&sum.AvgCostPerLiter, cur, 3)
	}
	if sum.EnergyCostPerKm != nil && sum.FuelEnergyCost > 0 && sum.EnergyCost > 0 {
		// EnergyCost mixes fuel and electricity on a plug-in hybrid: keep the fuel share of the per-km rate.
		per100 := *sum.EnergyCostPerKm * 100 * sum.FuelEnergyCost.Float() / sum.EnergyCost.Float()
		out.CostPer100km = decimalAmount(&per100, cur, 2)
	}
	return out
}

// buildIntegrationEnergy shapes the energy statistics (nil for a vehicle that cannot charge) and the TCO summary
// (nil for one that does not burn fuel).
func buildIntegrationEnergy(vehicle *models.Vehicle, stats *services.EnergyStats, sum *services.TCOSummary) IntegrationEnergyEfficiency {
	out := IntegrationEnergyEfficiency{VehicleID: vehicle.ID, Currency: vehicle.Currency, Powertrain: vehicle.Powertrain}
	if stats != nil {
		out.Electric = buildIntegrationElectric(vehicle.Currency, stats)
	}
	if sum != nil {
		out.Fuel = buildIntegrationFuel(vehicle.Currency, sum)
	}
	return out
}

// WithEnergy gives the energy view its statistics service.
func (h *IntegrationReadHandler) WithEnergy(s *services.EnergyStatsService) *IntegrationReadHandler {
	h.energy = s
	return h
}

// EnergyEfficiency gives how a vehicle uses and pays for its energy.
func (h *IntegrationReadHandler) EnergyEfficiency(ctx context.Context, userID, vehicleID string) (*IntegrationEnergyEfficiency, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	var stats *services.EnergyStats
	if vehicle.CanCharge() && h.energy != nil {
		if stats, err = h.energy.Compute(ctx, vehicle.ID); err != nil {
			return nil, err
		}
	}
	var sum *services.TCOSummary
	if models.PowertrainCanRefuel(vehicle.Powertrain) {
		if sum, err = h.tcoService.ComputeVehicleTCO(ctx, vehicle.ID); err != nil {
			return nil, err
		}
	}
	out := buildIntegrationEnergy(vehicle, stats, sum)
	return &out, nil
}

// GetEnergyEfficiency serves EnergyEfficiency.
func (h *IntegrationReadHandler) GetEnergyEfficiency(w http.ResponseWriter, r *http.Request) {
	out, err := h.EnergyEfficiency(r.Context(), middleware.GetUserID(r.Context()), chi.URLParam(r, "vehicleId"))
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to compute the energy efficiency")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
