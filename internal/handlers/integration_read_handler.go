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
// address, coordinate or document, amounts carry their currency, and an unknown figure is null rather than zero.
type IntegrationReadHandler struct {
	repo       *database.Repository
	tcoService *services.TCOService
}

func NewIntegrationReadHandler(repo *database.Repository, tcoService *services.TCOService) *IntegrationReadHandler {
	return &IntegrationReadHandler{repo: repo, tcoService: tcoService}
}

// IntegrationAmount is a monetary amount with its currency. Amount is a decimal string so no client rounds it.
type IntegrationAmount struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
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
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

func kilometres(v float64) IntegrationDistance { return IntegrationDistance{Value: v, Unit: "km"} }

// IntegrationVehicleV1 is a vehicle of the read API. It is the integration view plus the unit of the odometer.
type IntegrationVehicleV1 struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	Make       string              `json:"make"`
	Model      string              `json:"model"`
	Powertrain string              `json:"powertrain"`
	Currency   string              `json:"currency"`
	Odometer   IntegrationDistance `json:"odometer"`
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
	Energy      IntegrationAmount `json:"energy"`
	Tolls       IntegrationAmount `json:"tolls"`
	Maintenance IntegrationAmount `json:"maintenance"`
	Insurance   IntegrationAmount `json:"insurance"`
	Financing   IntegrationAmount `json:"financing"`
	Tires       IntegrationAmount `json:"tires"`
	Other       IntegrationAmount `json:"other"`
	Total       IntegrationAmount `json:"total"`
}

// IntegrationCostPeriod is the costs and distance of a period; the month in progress counts with the instalments
// that fall due later in it. CostPerKm is null while the distance is unknown.
type IntegrationCostPeriod struct {
	// From and To are the first and last month (YYYY-MM) the period covers; null for the whole ownership.
	From      *string                  `json:"from"`
	To        *string                  `json:"to"`
	Costs     IntegrationCostBreakdown `json:"costs"`
	Distance  IntegrationDistance      `json:"distance"`
	CostPerKm *IntegrationAmount       `json:"cost_per_km"`
}

// IntegrationCompleteness says how much of the figures rests on recorded data.
type IntegrationCompleteness struct {
	IsComplete     bool     `json:"is_complete"`
	ScoreAvailable bool     `json:"score_available"`
	ScorePct       *int     `json:"score_pct"`
	Warnings       []string `json:"warnings"`
}

// IntegrationCosts is the cost summary of a vehicle.
type IntegrationCosts struct {
	VehicleID     string                  `json:"vehicle_id"`
	Currency      string                  `json:"currency"`
	Last12Months  IntegrationCostPeriod   `json:"last_12_months"`
	SincePurchase IntegrationCostPeriod   `json:"since_purchase"`
	Completeness  IntegrationCompleteness `json:"completeness"`
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
	Kind     string              `json:"kind"`
	At       string              `json:"at"` // RFC 3339 instant
	Distance IntegrationDistance `json:"distance"`
}

// IntegrationDataGaps lists what is missing from a vehicle's data.
type IntegrationDataGaps struct {
	VehicleID string `json:"vehicle_id"`
	// OdometerGaps counts the stretches of odometer no drive covers, and OdometerGapDistance adds them up.
	OdometerGaps        int                 `json:"odometer_gaps"`
	OdometerGapDistance IntegrationDistance `json:"odometer_gap_distance"`
	OdometerAnomalies   int                 `json:"odometer_anomalies"`
	// Issues holds the first few, the most recent first; Truncated says more exist.
	Issues    []IntegrationDataGap `json:"issues"`
	Truncated bool                 `json:"truncated"`

	ChargesWithoutCost int                 `json:"charges_without_cost"`
	UnqualifiedDrives  int                 `json:"unqualified_drives"`
	UntrackedDistance  IntegrationDistance `json:"untracked_distance"`
	InsuranceMissing   bool                `json:"insurance_missing"`
	AcquisitionMissing bool                `json:"acquisition_missing"`
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
	out := buildIntegrationDataGaps(vehicle.ID, issues, gaps, gapKm, anomalies, sum.Completeness)
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

func buildIntegrationDataGaps(vehicleID string, issues []models.DataQualityIssue, gaps int, gapKm float64, anomalies int, comp services.TCOCompleteness) IntegrationDataGaps {
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
	}
}
