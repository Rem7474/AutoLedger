package handlers

import (
	"context"
	"net/http"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/money"
	"github.com/teslacost/teslacost/internal/services"
)

// Caps of the monthly view: a request never returns more than ten years of months.
const (
	integrationMonthsDefault = 12
	integrationMonthsMax     = 120
)

var integrationMonthPattern = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// IntegrationMonthlyQuery picks the months of the series. Both bounds are YYYY-MM and optional: without them the
// series is the last twelve months, ending with the month in progress.
type IntegrationMonthlyQuery struct {
	From string
	To   string
}

// IntegrationMonth is what a vehicle cost in one calendar month of the account's reporting timezone.
type IntegrationMonth struct {
	Month      string                   `json:"month" jsonschema:"calendar month, YYYY-MM"`
	InProgress bool                     `json:"in_progress" jsonschema:"true for the month in progress, whose figures are not final"`
	Costs      IntegrationCostBreakdown `json:"costs" jsonschema:"what was spent in the month, by category; the month in progress counts the instalments that fall due later in it"`
	Scheduled  *IntegrationAmount       `json:"scheduled" jsonschema:"part of the total that falls due later in the month in progress (instalments dated after today); null when none or for a past month"`
	Distance   IntegrationDistance      `json:"distance" jsonschema:"distance driven in the month, tracked plus estimated for the untracked part"`
	CostPerKm  *IntegrationAmount       `json:"cost_per_km" jsonschema:"total cost divided by the distance, per kilometre; null while the distance is unknown"`
}

// IntegrationMonthlyCosts is a vehicle's month-by-month series, oldest first.
type IntegrationMonthlyCosts struct {
	VehicleID string `json:"vehicle_id" jsonschema:"the vehicle these figures belong to"`
	Currency  string `json:"currency" jsonschema:"currency of every amount"`
	// Months are consecutive: a month without any record is present with zero amounts.
	Months []IntegrationMonth `json:"months" jsonschema:"consecutive calendar months, oldest first; a month with no record is present with zero amounts"`
	// Total adds up the months listed, nothing else.
	Total     IntegrationCostBreakdown `json:"total" jsonschema:"the listed months added up, by category"`
	Distance  IntegrationDistance      `json:"distance" jsonschema:"distance of the listed months added up"`
	CostPerKm *IntegrationAmount       `json:"cost_per_km" jsonschema:"total cost of the listed months divided by their distance; null while the distance is unknown"`
}

func monthBound(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	if !integrationMonthPattern.MatchString(raw) {
		return nil, apierror.New("tco.contributions_invalid_month", "Month must be formatted as YYYY-MM")
	}
	t, _ := time.Parse("2006-01", raw)
	return &t, nil
}

// MonthlyCosts lists the months of a vehicle that fall in the query, with the cost of each by category.
func (h *IntegrationReadHandler) MonthlyCosts(ctx context.Context, userID, vehicleID string, q IntegrationMonthlyQuery) (*IntegrationMonthlyCosts, error) {
	from, err := monthBound(q.From)
	if err != nil {
		return nil, err
	}
	to, err := monthBound(q.To)
	if err != nil {
		return nil, err
	}
	if from != nil && to != nil && from.After(*to) {
		return nil, apierror.New("request.invalid_period", "The start of the period is after its end")
	}
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	sum, err := h.tcoService.ComputeVehicleTCO(ctx, vehicle.ID)
	if err != nil {
		return nil, err
	}
	out := buildIntegrationMonthlyCosts(vehicle.ID, vehicle.Currency, sum, from, to)
	return &out, nil
}

func buildIntegrationMonthlyCosts(vehicleID, currency string, sum *services.TCOSummary, from, to *time.Time) IntegrationMonthlyCosts {
	var picked []services.MonthlyCost
	for _, m := range sum.MonthlyCosts {
		at, err := time.Parse("2006-01", m.Month)
		if err != nil || (from != nil && at.Before(*from)) || (to != nil && at.After(*to)) {
			continue
		}
		picked = append(picked, m)
	}
	// Without any bound the series is the last twelve months. With one, it runs from or up to that month; a window
	// never exceeds the cap, and keeps the months nearest the bound that was given.
	switch {
	case from == nil && to == nil:
		picked = lastN(picked, integrationMonthsDefault)
	case from != nil && len(picked) > integrationMonthsMax:
		picked = picked[:integrationMonthsMax]
	default:
		picked = lastN(picked, integrationMonthsMax)
	}

	out := IntegrationMonthlyCosts{VehicleID: vehicleID, Currency: currency, Months: make([]IntegrationMonth, 0, len(picked))}
	var energy, tolls, maintenance, insurance, financing, tires, other, total money.Cents
	var km float64
	for _, m := range picked {
		month := IntegrationMonth{
			Month: m.Month, InProgress: m.Month == sum.CurrentMonth,
			Costs:    breakdown(currency, m.Energy, m.Tolls, m.Maintenance, m.Insurance, m.Financing, m.Tires, m.Other, m.Total),
			Distance: kilometres(m.DistanceKm),
		}
		if m.DistanceKm > 0 {
			// The cash total over the distance, as get_vehicle_costs does: the application's own monthly rate spreads
			// the tires over the kilometres they ran, which a cash series does not.
			perKm := m.Total.Float() / m.DistanceKm
			month.CostPerKm = ratePerKm(&perKm, currency)
		}
		if m.Scheduled != 0 {
			s := amountOf(m.Scheduled, currency)
			month.Scheduled = &s
		}
		out.Months = append(out.Months, month)
		energy, tolls, maintenance, insurance = energy+m.Energy, tolls+m.Tolls, maintenance+m.Maintenance, insurance+m.Insurance
		financing, tires, other, total = financing+m.Financing, tires+m.Tires, other+m.Other, total+m.Total
		km += m.DistanceKm
	}
	out.Total = breakdown(currency, energy, tolls, maintenance, insurance, financing, tires, other, total)
	out.Distance = kilometres(km)
	if km > 0 {
		perKm := total.Float() / km
		out.CostPerKm = ratePerKm(&perKm, currency)
	}
	return out
}

func lastN[T any](s []T, n int) []T {
	if len(s) > n {
		return s[len(s)-n:]
	}
	return s
}

// GetMonthlyCosts serves MonthlyCosts.
func (h *IntegrationReadHandler) GetMonthlyCosts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	out, err := h.MonthlyCosts(r.Context(), middleware.GetUserID(r.Context()), chi.URLParam(r, "vehicleId"),
		IntegrationMonthlyQuery{From: query.Get("from"), To: query.Get("to")})
	if err != nil {
		if apiErr, ok := apierror.As(err); ok {
			writeAPIError(w, http.StatusBadRequest, apiErr)
			return
		}
		writeIntegrationError(w, r, err, "Failed to compute the monthly costs")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
