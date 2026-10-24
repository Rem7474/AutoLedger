package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/services"
)

// integrationMonthEntriesMax caps the entries listed for one category of one month.
const integrationMonthEntriesMax = 100

// IntegrationMonthEntry is one record behind a category total. It carries no identifier.
type IntegrationMonthEntry struct {
	Date           string            `json:"date" jsonschema:"RFC 3339 instant of the record"`
	Kind           string            `json:"kind" jsonschema:"charge (a charging session) or expense (a maintenance, insurance or other expense entry)"`
	Label          *string           `json:"label" jsonschema:"the expense description or the place of the charge as the owner recorded it, user text; null when none"`
	Amount         IntegrationAmount `json:"amount" jsonschema:"what the record contributes to the month"`
	Recurring      bool              `json:"recurring" jsonschema:"true when the expense repeats"`
	IntervalMonths *int              `json:"interval_months" jsonschema:"months between two occurrences of a recurring expense; null otherwise"`
}

// IntegrationMonthEstimate is the part of an energy total that is estimated rather than recorded.
type IntegrationMonthEstimate struct {
	Amount    IntegrationAmount   `json:"amount" jsonschema:"energy cost added for distance driven that no record covers"`
	Distance  IntegrationDistance `json:"distance" jsonschema:"that distance"`
	EnergyKwh float64             `json:"energy_kwh" jsonschema:"the energy the estimate stands for, in kWh"`
}

// IntegrationMonthBreakdown explains one category of one month: the records that add up to the figure of
// get_monthly_costs.
type IntegrationMonthBreakdown struct {
	VehicleID string                    `json:"vehicle_id" jsonschema:"the vehicle these figures belong to"`
	Currency  string                    `json:"currency" jsonschema:"currency of every amount"`
	Month     string                    `json:"month" jsonschema:"calendar month, YYYY-MM"`
	Category  string                    `json:"category" jsonschema:"energy or insurance"`
	Total     IntegrationAmount         `json:"total" jsonschema:"the month's figure for this category, as get_monthly_costs gives it: the records plus the estimate"`
	Entries   []IntegrationMonthEntry   `json:"entries" jsonschema:"the records behind the total, oldest first"`
	Matching  int                       `json:"matching" jsonschema:"how many records the month holds in this category, listed or not"`
	Truncated bool                      `json:"truncated" jsonschema:"true when more records exist than the ones listed"`
	Estimate  *IntegrationMonthEstimate `json:"estimate" jsonschema:"part of the total that is estimated rather than recorded; null when none"`
}

func buildIntegrationMonthBreakdown(vehicleID, currency string, c services.MonthlyContributions) IntegrationMonthBreakdown {
	out := IntegrationMonthBreakdown{
		VehicleID: vehicleID, Currency: currency, Month: c.Month, Category: strings.ToLower(c.Category),
		Total: amountOf(c.Total, currency), Entries: []IntegrationMonthEntry{}, Matching: len(c.Items), Truncated: len(c.Items) > integrationMonthEntriesMax,
	}
	for i, it := range c.Items {
		if i >= integrationMonthEntriesMax {
			break
		}
		entry := IntegrationMonthEntry{
			Date: it.Date.UTC().Format(time.RFC3339), Kind: "expense", Label: noteOf(&it.Label),
			Amount: amountOf(it.Amount, currency), Recurring: it.Recurring,
		}
		if it.SourceTable == "charge_logs" {
			// The label of a charge is its address, which may only be a position.
			entry.Kind, entry.Label = "charge", publicAddress(&it.Label)
		}
		if it.IntervalMonths > 0 {
			months := it.IntervalMonths
			entry.IntervalMonths = &months
		}
		out.Entries = append(out.Entries, entry)
	}
	if c.Estimate != nil {
		out.Estimate = &IntegrationMonthEstimate{Amount: amountOf(c.Estimate.Amount, currency), Distance: kilometres(c.Estimate.Km), EnergyKwh: c.Estimate.Kwh}
	}
	return out
}

// MonthBreakdown lists the records behind the energy or the insurance of one month.
func (h *IntegrationReadHandler) MonthBreakdown(ctx context.Context, userID, vehicleID, month, category string) (*IntegrationMonthBreakdown, error) {
	category = strings.ToUpper(strings.TrimSpace(category))
	if err := services.ValidateContributionQuery(month, category); err != nil {
		return nil, apierror.New("request.invalid_value", "Invalid value")
	}
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	contributions, err := h.tcoService.MonthlyCostContributions(ctx, vehicle.ID, month, category)
	if err != nil {
		return nil, err
	}
	out := buildIntegrationMonthBreakdown(vehicle.ID, vehicle.Currency, *contributions)
	return &out, nil
}

// GetMonthBreakdown serves MonthBreakdown.
func (h *IntegrationReadHandler) GetMonthBreakdown(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	out, err := h.MonthBreakdown(r.Context(), middleware.GetUserID(r.Context()), chi.URLParam(r, "vehicleId"), query.Get("month"), query.Get("category"))
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to list the records of the month")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
