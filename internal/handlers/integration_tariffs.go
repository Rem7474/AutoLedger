package handlers

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
	"github.com/teslacost/teslacost/internal/services"
)

const (
	integrationEstimateMaxKwh     = 1000
	integrationEstimateMaxMinutes = 7 * 24 * 60
)

// WithTariffs gives the charge estimate the service that prices a session with a tariff plan.
func (h *IntegrationReadHandler) WithTariffs(s *services.TariffService) *IntegrationReadHandler {
	h.tariffs = s
	return h
}

// IntegrationTariffBand is a named price of a plan.
type IntegrationTariffBand struct {
	Name        string `json:"name" jsonschema:"name of the band, user text on a plan with bands; FLAT, PEAK or OFFPEAK on the others"`
	PricePerKwh string `json:"price_per_kwh" jsonschema:"decimal string in the currency of the plan, up to six decimals"`
}

// IntegrationTariffRule assigns a band to a daily interval.
type IntegrationTariffRule struct {
	Days  []int  `json:"days" jsonschema:"weekdays the interval applies to, 0 is Sunday and 6 Saturday; empty means every day"`
	Start string `json:"start" jsonschema:"HH:MM, in the account's timezone"`
	End   string `json:"end" jsonschema:"HH:MM; an interval that ends before it starts crosses midnight"`
	Band  string `json:"band" jsonschema:"name of the band charged in this interval"`
}

// IntegrationTariffPlan is a plan the owner defined.
type IntegrationTariffPlan struct {
	ID          string                  `json:"id" jsonschema:"pass it as plan_id to estimate_charge_cost"`
	Name        string                  `json:"name" jsonschema:"user text; plans of one account that share a name are the successive versions of one tariff"`
	Currency    string                  `json:"currency" jsonschema:"ISO 4217 currency code of the prices"`
	IsDefault   bool                    `json:"is_default" jsonschema:"true for the plan used when a vehicle has none of its own"`
	ValidFrom   *string                 `json:"valid_from" jsonschema:"first day the prices apply to, YYYY-MM-DD; null when there is no start"`
	ValidTo     *string                 `json:"valid_to" jsonschema:"last day the prices apply to, YYYY-MM-DD; null when there is no end"`
	Bands       []IntegrationTariffBand `json:"bands" jsonschema:"the prices of the plan; a flat plan has one"`
	Rules       []IntegrationTariffRule `json:"rules" jsonschema:"when each band applies, the first matching rule winning; empty on a flat plan"`
	DefaultBand string                  `json:"default_band" jsonschema:"the band charged when no rule matches"`
}

// IntegrationTariffPlans lists the plans of the account.
type IntegrationTariffPlans struct {
	Plans []IntegrationTariffPlan `json:"plans" jsonschema:"the plans of the account, the default one first"`
}

// TariffPlans lists the tariff plans of the account.
func (h *IntegrationReadHandler) TariffPlans(ctx context.Context, userID string) (*IntegrationTariffPlans, error) {
	plans, err := h.repo.ListTariffPlans(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := &IntegrationTariffPlans{Plans: make([]IntegrationTariffPlan, 0, len(plans))}
	for i := range plans {
		out.Plans = append(out.Plans, h.tariffPlanView(&plans[i]))
	}
	return out, nil
}

func (h *IntegrationReadHandler) tariffPlanView(p *models.TariffPlan) IntegrationTariffPlan {
	view := IntegrationTariffPlan{ID: p.ID, Name: p.Name, Currency: p.Currency, IsDefault: p.IsDefault, ValidFrom: blankToNil(p.ValidFrom), ValidTo: blankToNil(p.ValidTo),
		Bands: []IntegrationTariffBand{}, Rules: []IntegrationTariffRule{}}
	if h.tariffs == nil {
		return view
	}
	bands, rules, defaultBand, ok := h.tariffs.Grid(p)
	if !ok {
		return view
	}
	for _, b := range bands {
		view.Bands = append(view.Bands, IntegrationTariffBand{Name: b.Name, PricePerKwh: b.RateCents.String()})
	}
	for _, r := range rules {
		days := r.Days
		if days == nil {
			days = []int{}
		}
		view.Rules = append(view.Rules, IntegrationTariffRule{Days: days, Start: r.Start, End: r.End, Band: r.Band})
	}
	view.DefaultBand = defaultBand
	return view
}

// IntegrationChargeEstimateQuery is a charge to price: its energy, when it starts and how long it lasts, and either
// the vehicle whose plan prices it or a plan; with neither, the default plan of the account.
type IntegrationChargeEstimateQuery struct {
	VehicleID string
	PlanID    string
	Kwh       float64
	Start     time.Time
	Minutes   int
}

// IntegrationChargeBand is the part of a charge priced in one band.
type IntegrationChargeBand struct {
	Band        string  `json:"band" jsonschema:"name of the band, user text on a plan with bands"`
	Minutes     int     `json:"minutes" jsonschema:"minutes of the charge spent in this band"`
	Kwh         float64 `json:"kwh" jsonschema:"energy priced in this band, shared between the bands in proportion to the time spent in each"`
	PricePerKwh string  `json:"price_per_kwh" jsonschema:"decimal string in the currency of the plan"`
}

// IntegrationChargePlan names the plan that priced a charge.
type IntegrationChargePlan struct {
	ID   string `json:"id" jsonschema:"id of the plan, as listed by list_tariff_plans"`
	Name string `json:"name" jsonschema:"user text"`
}

// IntegrationChargeEstimate is what a charge would cost with a plan. Nothing is stored.
type IntegrationChargeEstimate struct {
	Priced             bool                    `json:"priced" jsonschema:"false when no plan prices the charge: then cost is null and unpriced_reason says why"`
	UnpricedReason     *string                 `json:"unpriced_reason" jsonschema:"why no plan priced the charge; null when it was priced"`
	Cost               *IntegrationAmount      `json:"cost" jsonschema:"the cost of the charge, in the currency of the plan"`
	Plan               *IntegrationChargePlan  `json:"plan" jsonschema:"the plan that priced the charge; null when none did"`
	Kwh                float64                 `json:"kwh" jsonschema:"the energy that was priced"`
	AveragePricePerKwh *string                 `json:"average_price_per_kwh" jsonschema:"cost divided by energy, a decimal string; null when not priced"`
	Bands              []IntegrationChargeBand `json:"bands" jsonschema:"how the charge splits between the bands of the plan, the largest share first"`
}

// EstimateCharge prices a charge that has not happened (or one the owner is thinking about) with a tariff plan, the
// way a recorded charge is priced. It reads the plans and stores nothing.
func (h *IntegrationReadHandler) EstimateCharge(ctx context.Context, userID string, q IntegrationChargeEstimateQuery) (*IntegrationChargeEstimate, error) {
	if h.tariffs == nil {
		return nil, errors.New("tariff service unavailable")
	}
	if q.Kwh <= 0 || q.Kwh > integrationEstimateMaxKwh || math.IsNaN(q.Kwh) {
		return nil, mcpUserError("energy_kwh must be above 0 and at most 1000")
	}
	if q.Minutes < 1 || q.Minutes > integrationEstimateMaxMinutes {
		return nil, mcpUserError("duration_minutes must be between 1 and 10080 (a week)")
	}
	q.PlanID = strings.TrimSpace(q.PlanID)
	if q.VehicleID != "" && q.PlanID != "" {
		return nil, mcpUserError("pass a vehicle_id or a plan_id, not both")
	}

	plan, err := h.estimatePlan(ctx, userID, q)
	if err != nil {
		return nil, err
	}
	out := &IntegrationChargeEstimate{Kwh: q.Kwh, Bands: []IntegrationChargeBand{}}
	end := q.Start.Add(time.Duration(q.Minutes) * time.Minute)
	cost, err := h.tariffs.CalculateSessionCostIfApplicable(plan, q.Start, end, q.Kwh)
	if err != nil {
		return nil, err
	}
	if cost == nil {
		reason := unpricedReason(h.tariffs, plan, q.Start)
		out.UnpricedReason = &reason
		return out, nil
	}
	pricing, err := h.tariffs.PriceSession(plan, q.Start, end, q.Kwh)
	if err != nil {
		return nil, err
	}
	amount := amountOf(*cost, plan.Currency)
	average := money.Rate(math.Round(cost.Float() / q.Kwh * 1_000_000)).String()
	out.Priced = true
	out.Cost = &amount
	out.Plan = &IntegrationChargePlan{ID: plan.ID, Name: plan.Name}
	out.AveragePricePerKwh = &average
	for _, share := range pricing.Bands {
		out.Bands = append(out.Bands, IntegrationChargeBand{Band: share.Band, Minutes: share.Minutes, Kwh: math.Round(share.Kwh*1000) / 1000, PricePerKwh: share.Rate.String()})
	}
	return out, nil
}

// estimatePlan picks the plan like the charge form does: the plan assigned to the vehicle, in the version in force on
// the day of the charge, or the plan asked for, or the default plan. A missing plan is nil, not an error.
func (h *IntegrationReadHandler) estimatePlan(ctx context.Context, userID string, q IntegrationChargeEstimateQuery) (*models.TariffPlan, error) {
	var plan *models.TariffPlan
	var err error
	switch {
	case q.VehicleID != "":
		if _, err := h.vehicleOf(ctx, userID, q.VehicleID); err != nil {
			return nil, err
		}
		plan, err = h.repo.GetVehicleTariffPlanAt(ctx, q.VehicleID, h.tariffs.DayOf(q.Start))
	case q.PlanID != "":
		plan, err = h.repo.GetTariffPlanByID(ctx, q.PlanID, userID)
		if errors.Is(err, database.ErrNotFound) {
			return nil, mcpUserError("unknown plan_id: use list_tariff_plans to get a valid one")
		}
	default:
		plan, err = h.repo.GetDefaultTariffPlan(ctx, userID)
	}
	if errors.Is(err, database.ErrNotFound) {
		return nil, nil
	}
	return plan, err
}

func unpricedReason(s *services.TariffService, plan *models.TariffPlan, start time.Time) string {
	switch {
	case plan == nil:
		return "no tariff plan applies: the account has none, or the vehicle's plan is in another currency than the vehicle"
	case !plan.AppliesOn(s.DayOf(start)):
		return "the plan does not apply on that day: its prices are valid from " + dayOrNever(plan.ValidFrom, "the start") + " to " + dayOrNever(plan.ValidTo, "no end")
	default:
		return "the plan has no complete price for that time"
	}
}

func dayOrNever(day *string, fallback string) string {
	if day == nil || *day == "" {
		return fallback
	}
	return *day
}

// GetTariffPlans serves the tariff plans of the account.
func (h *IntegrationReadHandler) GetTariffPlans(w http.ResponseWriter, r *http.Request) {
	out, err := h.TariffPlans(r.Context(), middleware.GetUserID(r.Context()))
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to list the tariff plans")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// GetChargeEstimate prices a charge from the query: energy_kwh, duration_minutes, and optionally start (RFC 3339, now
// by default), vehicle_id or plan_id.
func (h *IntegrationReadHandler) GetChargeEstimate(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	kwh, kwhErr := strconv.ParseFloat(query.Get("energy_kwh"), 64)
	minutes, minutesErr := strconv.Atoi(query.Get("duration_minutes"))
	if kwhErr != nil || minutesErr != nil {
		writeAPIError(w, http.StatusBadRequest, apierror.New("request.invalid_value", "energy_kwh (a number) and duration_minutes (a whole number) are required"))
		return
	}
	start := time.Now()
	if raw := strings.TrimSpace(query.Get("start")); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, apierror.New("request.invalid_date", "start must be an RFC 3339 date and time with its offset"))
			return
		}
		start = parsed
	}
	out, err := h.EstimateCharge(r.Context(), middleware.GetUserID(r.Context()), IntegrationChargeEstimateQuery{
		VehicleID: strings.TrimSpace(query.Get("vehicle_id")), PlanID: query.Get("plan_id"), Kwh: kwh, Start: start, Minutes: minutes})
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to estimate the charge")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
