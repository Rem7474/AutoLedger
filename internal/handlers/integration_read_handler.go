package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
	"github.com/teslacost/teslacost/internal/services"
)

// Caps of the read API: a response never grows with the account's history.
const (
	integrationReadMonths = 12
	integrationReadIssues = 20
)

// IntegrationReadHandler serves the read-only API behind /api/integrations/v1, for tools that report on the data
// (assistants, scripts) rather than push into it. Its views are narrower than the application's: no VIN, plate,
// coordinate or document, and the trips are the only view with addresses; amounts carry their currency, and an
// unknown figure is null rather than zero.
type IntegrationReadHandler struct {
	repo       *database.Repository
	tcoService *services.TCOService
}

func NewIntegrationReadHandler(repo *database.Repository, tcoService *services.TCOService) *IntegrationReadHandler {
	return &IntegrationReadHandler{repo: repo, tcoService: tcoService}
}

// IntegrationAmount is a monetary amount with its currency. Amount is a decimal string so no client rounds it.
type IntegrationAmount struct {
	Amount   string `json:"amount" jsonschema:"decimal string such as 12.34, never rounded or converted, so do not turn it into a rounded figure"`
	Currency string `json:"currency" jsonschema:"ISO 4217 currency code of the vehicle the amount belongs to"`
}

func amountOf(c money.Cents, currency string) IntegrationAmount {
	return IntegrationAmount{Amount: c.String(), Currency: currency}
}

// ratePerKm is a cost per kilometre, null when the distance it divides by is unknown.
func ratePerKm(v *float64, currency string) *IntegrationAmount {
	if v == nil {
		return nil
	}
	return &IntegrationAmount{Amount: strconv.FormatFloat(*v, 'f', 4, 64), Currency: currency}
}

// IntegrationDistance is a distance with its unit, always kilometres.
type IntegrationDistance struct {
	Value float64 `json:"value" jsonschema:"the distance"`
	Unit  string  `json:"unit" jsonschema:"always km"`
}

func kilometres(v float64) IntegrationDistance { return IntegrationDistance{Value: v, Unit: "km"} }

// IntegrationVehicleV1 is a vehicle of the read API. It is the integration view plus the unit of the odometer.
type IntegrationVehicleV1 struct {
	ID         string              `json:"id" jsonschema:"identifier to pass as vehicle_id to the other tools"`
	Name       string              `json:"name" jsonschema:"name the owner gave the vehicle, written by the user"`
	Make       string              `json:"make" jsonschema:"make, free text, empty when not given"`
	Model      string              `json:"model" jsonschema:"model, free text, empty when not given"`
	Powertrain string              `json:"powertrain" jsonschema:"EV, ICE, PHEV or REEV"`
	Currency   string              `json:"currency" jsonschema:"currency of every amount of this vehicle, fixed at creation"`
	Odometer   IntegrationDistance `json:"odometer" jsonschema:"latest known odometer reading"`
}

func integrationVehicleV1(v *models.Vehicle) IntegrationVehicleV1 {
	return IntegrationVehicleV1{ID: v.ID, Name: v.Name, Make: v.Make, Model: v.Model, Powertrain: v.Powertrain,
		Currency: v.Currency, Odometer: kilometres(v.CurrentOdometer)}
}

// errIntegrationVehicleNotFound is a vehicle the account cannot see, whether it does not exist or belongs to someone else.
var errIntegrationVehicleNotFound = errors.New("vehicle not found")

// writeIntegrationError answers a failed read: a vehicle the account cannot see is a 404, anything else a logged 500.
func writeIntegrationError(w http.ResponseWriter, r *http.Request, err error, fallback string) {
	if errors.Is(err, errIntegrationVehicleNotFound) {
		writeAPIError(w, http.StatusNotFound, apierror.New("vehicle.not_found", "Vehicle not found"))
		return
	}
	writeRepoError(w, r, err, fallback)
}

// vehicleOf returns a vehicle of the user's account, or errIntegrationVehicleNotFound.
func (h *IntegrationReadHandler) vehicleOf(ctx context.Context, userID, vehicleID string) (*models.Vehicle, error) {
	v, err := h.repo.GetVehicleByID(ctx, vehicleID, userID)
	if err != nil {
		return nil, errIntegrationVehicleNotFound
	}
	return v, nil
}

// Vehicles lists the vehicles the account can see.
func (h *IntegrationReadHandler) Vehicles(ctx context.Context, userID string) ([]IntegrationVehicleV1, error) {
	vehicles, err := h.repo.ListVehiclesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]IntegrationVehicleV1, 0, len(vehicles))
	for i := range vehicles {
		out = append(out, integrationVehicleV1(&vehicles[i]))
	}
	return out, nil
}

// ListVehicles lists the vehicles the token's account can see.
func (h *IntegrationReadHandler) ListVehicles(w http.ResponseWriter, r *http.Request) {
	out, err := h.Vehicles(r.Context(), middleware.GetUserID(r.Context()))
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to list vehicles")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// GetVehicle returns one vehicle.
func (h *IntegrationReadHandler) GetVehicle(w http.ResponseWriter, r *http.Request) {
	vehicle, err := h.vehicleOf(r.Context(), middleware.GetUserID(r.Context()), chi.URLParam(r, "vehicleId"))
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to read the vehicle")
		return
	}
	writeJSON(w, http.StatusOK, integrationVehicleV1(vehicle))
}

// IntegrationCostBreakdown is the money spent over a period, by category, in the vehicle's currency. The categories
// add up to Total (cash basis, the purchase of the vehicle excluded).
type IntegrationCostBreakdown struct {
	Energy      IntegrationAmount `json:"energy" jsonschema:"electricity charged and fuel bought"`
	Tolls       IntegrationAmount `json:"tolls" jsonschema:"tolls, parking and ferries"`
	Maintenance IntegrationAmount `json:"maintenance" jsonschema:"servicing and unplanned repairs, insurance deductibles included"`
	Insurance   IntegrationAmount `json:"insurance" jsonschema:"insurance premiums"`
	Financing   IntegrationAmount `json:"financing" jsonschema:"cash paid for the loan or lease: rents, down payment, fees and loan interest"`
	Tires       IntegrationAmount `json:"tires" jsonschema:"tires, counted in full in the month they were bought"`
	Other       IntegrationAmount `json:"other" jsonschema:"subscriptions, taxes, accessories and anything else"`
	Total       IntegrationAmount `json:"total" jsonschema:"sum of the seven categories above: the cash spent, the purchase of the vehicle excluded"`
}

// IntegrationCostPeriod is the costs and distance of a period; the month in progress counts with the instalments
// that fall due later in it. CostPerKm is null while the distance is unknown.
type IntegrationCostPeriod struct {
	// From and To are the first and last month (YYYY-MM) the period covers; null for the whole ownership.
	From      *string                  `json:"from" jsonschema:"first month covered, YYYY-MM; null for the whole ownership"`
	To        *string                  `json:"to" jsonschema:"last month covered, YYYY-MM; null for the whole ownership"`
	Costs     IntegrationCostBreakdown `json:"costs" jsonschema:"what was spent over the period, by category"`
	Distance  IntegrationDistance      `json:"distance" jsonschema:"distance driven over the period, tracked plus estimated for the untracked part"`
	CostPerKm *IntegrationAmount       `json:"cost_per_km" jsonschema:"total cost divided by the distance, per kilometre; null while the distance is unknown"`
}

// IntegrationCompleteness says how much of the figures rests on recorded data.
type IntegrationCompleteness struct {
	IsComplete     bool     `json:"is_complete" jsonschema:"true when nothing is known to be missing"`
	ScoreAvailable bool     `json:"score_available" jsonschema:"false while the vehicle has no distance and no expense, so there is nothing to score"`
	ScorePct       *int     `json:"score_pct" jsonschema:"0 to 100, how much of the figures rests on recorded data; null when no score is available"`
	Warnings       []string `json:"warnings" jsonschema:"what makes the figures less reliable, as sentences; call get_vehicle_data_gaps for the detail"`
}

// IntegrationCosts is the cost summary of a vehicle.
type IntegrationCosts struct {
	VehicleID     string                  `json:"vehicle_id" jsonschema:"the vehicle these figures belong to"`
	Currency      string                  `json:"currency" jsonschema:"currency of every amount"`
	Last12Months  IntegrationCostPeriod   `json:"last_12_months" jsonschema:"the last twelve calendar months, the month in progress included with the instalments that fall due later in it; fewer months for a vehicle with a shorter history"`
	SincePurchase IntegrationCostPeriod   `json:"since_purchase" jsonschema:"the whole ownership, from the first record"`
	Completeness  IntegrationCompleteness `json:"completeness" jsonschema:"how far these figures can be trusted"`
}

func breakdown(currency string, energy, tolls, maintenance, insurance, financing, tires, other, total money.Cents) IntegrationCostBreakdown {
	return IntegrationCostBreakdown{
		Energy: amountOf(energy, currency), Tolls: amountOf(tolls, currency), Maintenance: amountOf(maintenance, currency),
		Insurance: amountOf(insurance, currency), Financing: amountOf(financing, currency), Tires: amountOf(tires, currency),
		Other: amountOf(other, currency), Total: amountOf(total, currency),
	}
}

// Costs summarizes what a vehicle cost over its last twelve months and since it was bought.
func (h *IntegrationReadHandler) Costs(ctx context.Context, userID, vehicleID string) (*IntegrationCosts, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	sum, err := h.tcoService.ComputeVehicleTCO(ctx, vehicle.ID)
	if err != nil {
		return nil, err
	}
	out := buildIntegrationCosts(vehicle, sum)
	return &out, nil
}

// GetCosts serves Costs.
func (h *IntegrationReadHandler) GetCosts(w http.ResponseWriter, r *http.Request) {
	out, err := h.Costs(r.Context(), middleware.GetUserID(r.Context()), chi.URLParam(r, "vehicleId"))
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to compute the costs")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func buildIntegrationCosts(vehicle *models.Vehicle, sum *services.TCOSummary) IntegrationCosts {
	cur := vehicle.Currency

	months := sum.MonthlyCosts
	if len(months) > integrationReadMonths {
		months = months[len(months)-integrationReadMonths:]
	}
	var energy, tolls, maintenance, insurance, financing, tires, other, total money.Cents
	var km float64
	for _, m := range months {
		energy += m.Energy
		tolls += m.Tolls
		maintenance += m.Maintenance
		insurance += m.Insurance
		financing += m.Financing
		tires += m.Tires
		other += m.Other
		total += m.Total
		km += m.DistanceKm
	}
	last12 := IntegrationCostPeriod{
		Costs:    breakdown(cur, energy, tolls, maintenance, insurance, financing, tires, other, total),
		Distance: kilometres(km),
	}
	if len(months) > 0 {
		from, to := months[0].Month, months[len(months)-1].Month
		last12.From, last12.To = &from, &to
	}
	if km > 0 {
		perKm := total.Float() / km
		last12.CostPerKm = ratePerKm(&perKm, cur)
	}

	since := IntegrationCostPeriod{
		Costs: breakdown(cur, sum.EnergyCost, sum.TollsCost, sum.MaintenanceCost+sum.RepairCost, sum.InsuranceCost,
			sum.FinancingCost, sum.TiresCost, sum.SubscriptionCost+sum.TaxCost+sum.OtherCost, sum.TotalCost),
		Distance:  kilometres(sum.DistanceBasisKm),
		CostPerKm: ratePerKm(sum.TotalCostPerKm, cur),
	}

	comp := sum.Completeness
	completeness := IntegrationCompleteness{IsComplete: comp.IsComplete, ScoreAvailable: comp.ScoreAvailable, Warnings: comp.Warnings}
	if completeness.Warnings == nil {
		completeness.Warnings = []string{}
	}
	if comp.ScoreAvailable {
		score := comp.ScorePct
		completeness.ScorePct = &score
	}
	return IntegrationCosts{VehicleID: vehicle.ID, Currency: cur, Last12Months: last12, SincePurchase: since, Completeness: completeness}
}

// IntegrationDataGap is one place where the recorded history does not add up. It carries no drive identifier,
// address or position: when it was found and the distance are what a report needs to say what is missing.
type IntegrationDataGap struct {
	Kind     string              `json:"kind" jsonschema:"ODOMETER_GAP (odometer moved with no drive recorded), ODOMETER_REGRESSION (odometer went down) or DISTANCE_MISMATCH (drive distance differs from the odometer)"`
	At       string              `json:"at" jsonschema:"RFC 3339 instant the problem was found"`
	Distance IntegrationDistance `json:"distance" jsonschema:"distance involved"`
}

// IntegrationDataGaps lists what is missing from a vehicle's data.
type IntegrationDataGaps struct {
	VehicleID string `json:"vehicle_id" jsonschema:"the vehicle these figures belong to"`
	// OdometerGaps counts the stretches of odometer no drive covers, and OdometerGapDistance adds them up.
	OdometerGaps        int                 `json:"odometer_gaps" jsonschema:"number of stretches of odometer that no recorded drive covers"`
	OdometerGapDistance IntegrationDistance `json:"odometer_gap_distance" jsonschema:"total distance of those stretches"`
	OdometerAnomalies   int                 `json:"odometer_anomalies" jsonschema:"number of odometer readings that go down or disagree with the drives"`
	// Issues holds the first few, the most recent first; Truncated says more exist.
	Issues    []IntegrationDataGap `json:"issues" jsonschema:"the most recent gaps and anomalies, newest first, at most 20"`
	Truncated bool                 `json:"truncated" jsonschema:"true when more issues exist than the ones listed"`

	ChargesWithoutCost int                 `json:"charges_without_cost" jsonschema:"charging sessions recorded without a price, whose cost is left out of the totals"`
	UnqualifiedDrives  int                 `json:"unqualified_drives" jsonschema:"drives not yet classed as professional or personal"`
	UntrackedDistance  IntegrationDistance `json:"untracked_distance" jsonschema:"distance driven that no drive record covers, estimated from the odometer"`
	InsuranceMissing   bool                `json:"insurance_missing" jsonschema:"true when no insurance premium is recorded and the lease does not include it"`
	AcquisitionMissing bool                `json:"acquisition_missing" jsonschema:"true when the acquisition (purchase, loan or lease) is not entered or lacks details, so depreciation and financing are not fully known"`
	// AcquisitionMissingFields names what the acquisition still lacks; empty when the acquisition itself is missing.
	AcquisitionMissingFields []string `json:"acquisition_missing_fields" jsonschema:"what the acquisition still lacks for the depreciation: purchase_price, resale_value and/or holding_period; empty when the acquisition itself is not entered"`
	StartOdometerMissing     bool     `json:"start_odometer_missing" jsonschema:"true when the odometer at the start of ownership is not entered, so the distance before the first reading is not counted"`
	UnconvertedExpenses      int      `json:"unconverted_expenses" jsonschema:"expenses in a foreign currency without a conversion rate, left out of the totals"`

	// Missing is the same list the application shows under "what to complete", the most valuable first.
	Missing []IntegrationMissing `json:"missing" jsonschema:"what to complete, ranked: the missing depreciation first, then the gaps with the largest estimated amount, then the ones that cannot be estimated"`
	// ScoreDimensions are the weighted parts of the completeness score.
	ScoreDimensions []IntegrationScoreDimension `json:"score_dimensions" jsonschema:"the weighted parts the completeness score is made of"`
}

// IntegrationMissing is one thing to complete and what it is worth. EstimatedAmount is an order of magnitude drawn
// from the vehicle's own prices; it is null when it cannot be estimated, which is not the same as nothing at stake.
type IntegrationMissing struct {
	Kind            string             `json:"kind" jsonschema:"depreciation (resale value, holding period or price missing), unpriced_charges (charging sessions without a price), untracked_distance (distance no drive covers) or unconverted_expenses (foreign currency without a rate)"`
	EstimatedAmount *IntegrationAmount `json:"estimated_amount" jsonschema:"rough cost left out of the totals, estimated from the vehicle's own prices; null when it cannot be estimated (the depreciation never is)"`
	Count           int                `json:"count" jsonschema:"records behind the gap: charges or expenses; 0 when it does not apply"`
	Quantity        *float64           `json:"quantity" jsonschema:"energy or distance without a cost, in quantity_unit; null when it does not apply"`
	QuantityUnit    *string            `json:"quantity_unit" jsonschema:"kWh or km; null when there is no quantity"`
	FixIn           *string            `json:"fix_in" jsonschema:"where the owner completes it in the application: charges or expenses; null when it is the ownership details or the drives"`
}

// IntegrationScoreDimension is one weighted part of the completeness score.
type IntegrationScoreDimension struct {
	Key        string  `json:"key" jsonschema:"stable identifier of the dimension"`
	Label      string  `json:"label" jsonschema:"English name of the dimension"`
	ScorePct   int     `json:"score_pct" jsonschema:"0 to 100, how complete this part is"`
	Weight     float64 `json:"weight" jsonschema:"weight of this part in the overall score"`
	Applicable bool    `json:"applicable" jsonschema:"false when there is nothing to evaluate for this vehicle; the part then carries no weight"`
}

// DataGaps says what a vehicle's history is missing, so a report can state how far its figures can be trusted.
func (h *IntegrationReadHandler) DataGaps(ctx context.Context, userID, vehicleID string) (*IntegrationDataGaps, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	issues, err := h.repo.ListDataQualityIssues(ctx, vehicle.ID, integrationReadIssues+1)
	if err != nil {
		return nil, err
	}
	gaps, gapKm, anomalies, err := h.repo.DataQualitySummary(ctx, vehicle.ID)
	if err != nil {
		return nil, err
	}
	sum, err := h.tcoService.ComputeVehicleTCO(ctx, vehicle.ID)
	if err != nil {
		return nil, err
	}
	out := buildIntegrationDataGaps(vehicle.ID, vehicle.Currency, issues, gaps, gapKm, anomalies, sum.Completeness)
	return &out, nil
}

// GetDataGaps serves DataGaps.
func (h *IntegrationReadHandler) GetDataGaps(w http.ResponseWriter, r *http.Request) {
	out, err := h.DataGaps(r.Context(), middleware.GetUserID(r.Context()), chi.URLParam(r, "vehicleId"))
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to check the data")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func buildIntegrationDataGaps(vehicleID, currency string, issues []models.DataQualityIssue, gaps int, gapKm float64, anomalies int, comp services.TCOCompleteness) IntegrationDataGaps {
	truncated := len(issues) > integrationReadIssues
	if truncated {
		issues = issues[:integrationReadIssues]
	}
	out := make([]IntegrationDataGap, 0, len(issues))
	for _, i := range issues {
		out = append(out, IntegrationDataGap{Kind: i.Type, At: i.Date.UTC().Format(time.RFC3339), Distance: kilometres(i.Km)})
	}
	return IntegrationDataGaps{
		VehicleID: vehicleID, OdometerGaps: gaps, OdometerGapDistance: kilometres(gapKm), OdometerAnomalies: anomalies,
		Issues: out, Truncated: truncated,
		ChargesWithoutCost: comp.ChargesWithoutCost, UnqualifiedDrives: comp.UnqualifiedDrives,
		UntrackedDistance: kilometres(comp.UntrackedDistanceKm),
		InsuranceMissing:  comp.InsuranceMissing, AcquisitionMissing: comp.AcquisitionMissing,
		AcquisitionMissingFields: nonNil(comp.AcquisitionMissingFields), StartOdometerMissing: comp.StartOdometerMissing,
		UnconvertedExpenses: comp.UnconvertedExpenses,
		Missing:             integrationMissing(comp.Gaps, currency), ScoreDimensions: integrationScoreDimensions(comp.Dimensions),
	}
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// gapQuantityUnit is the unit of the quantity a kind of gap carries.
var gapQuantityUnit = map[string]string{"unpriced_charges": "kWh", "untracked_distance": "km"}

func integrationMissing(gaps []services.CompletenessGap, currency string) []IntegrationMissing {
	out := make([]IntegrationMissing, 0, len(gaps))
	for _, g := range gaps {
		m := IntegrationMissing{Kind: g.Key, Count: g.Count}
		if g.Amount != nil {
			a := amountOf(*g.Amount, currency)
			m.EstimatedAmount = &a
		}
		if unit, ok := gapQuantityUnit[g.Key]; ok {
			q := g.Quantity
			m.Quantity, m.QuantityUnit = &q, &unit
		}
		if g.Target != "" {
			target := g.Target
			m.FixIn = &target
		}
		out = append(out, m)
	}
	return out
}

func integrationScoreDimensions(dims []services.CompletenessDimension) []IntegrationScoreDimension {
	out := make([]IntegrationScoreDimension, 0, len(dims))
	for _, d := range dims {
		out = append(out, IntegrationScoreDimension{Key: d.Key, Label: d.Label, ScorePct: d.ScorePct, Weight: d.Weight, Applicable: d.Applicable})
	}
	return out
}
