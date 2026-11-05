package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/money"
)

// IntegrationTariffPrice is the price of a band per kWh.
type IntegrationTariffPrice struct {
	Band        string            `json:"band" jsonschema:"name of the band in force, user text on a plan with bands; FLAT, PEAK or OFFPEAK on the others"`
	PricePerKwh IntegrationAmount `json:"price_per_kwh" jsonschema:"price of one kWh, in the currency of the plan, up to six decimals"`
}

// IntegrationTariffChange is the next moment the price per kWh changes.
type IntegrationTariffChange struct {
	At          string            `json:"at" jsonschema:"when the price changes, RFC 3339 in the account's timezone"`
	Band        string            `json:"band" jsonschema:"name of the band in force from then on"`
	PricePerKwh IntegrationAmount `json:"price_per_kwh" jsonschema:"price of one kWh from then on"`
}

// IntegrationCurrentTariff is the electricity price in force now and when it next changes.
type IntegrationCurrentTariff struct {
	Priced         bool                     `json:"priced" jsonschema:"false when no plan gives a price for now: then the other fields are null and unpriced_reason says why"`
	UnpricedReason *string                  `json:"unpriced_reason" jsonschema:"why there is no price; null when there is one"`
	Plan           *IntegrationChargePlan   `json:"plan" jsonschema:"the plan the price comes from; null when none applies"`
	Now            *IntegrationTariffPrice  `json:"now" jsonschema:"the band in force now and its price"`
	NextChange     *IntegrationTariffChange `json:"next_change" jsonschema:"the next time the price per kWh changes, within a week; null for a flat plan, or when the plan stops applying before the price would change"`
}

// CurrentTariff gives the price per kWh in force at the given moment with the plan assigned to the vehicle, in the
// version in force that day, or with the default plan of the account when no vehicle is named. It stores nothing.
func (h *IntegrationReadHandler) CurrentTariff(ctx context.Context, userID, vehicleID string, at time.Time) (*IntegrationCurrentTariff, error) {
	if h.tariffs == nil {
		return nil, errors.New("tariff service unavailable")
	}
	plan, err := h.estimatePlan(ctx, userID, IntegrationChargeEstimateQuery{VehicleID: vehicleID, Start: at})
	if err != nil {
		return nil, err
	}
	out := &IntegrationCurrentTariff{}
	price, ok := h.tariffs.PriceAt(plan, at)
	if !ok {
		reason := unpricedReason(h.tariffs, plan, at)
		out.UnpricedReason = &reason
		return out, nil
	}
	out.Priced = true
	out.Plan = &IntegrationChargePlan{ID: plan.ID, Name: plan.Name}
	out.Now = &IntegrationTariffPrice{Band: price.Band, PricePerKwh: ratePrice(price.Rate, plan.Currency)}
	if when, next, found := h.tariffs.NextPriceChange(plan, at); found {
		out.NextChange = &IntegrationTariffChange{At: when.In(h.tariffs.Location()).Format(time.RFC3339), Band: next.Band, PricePerKwh: ratePrice(next.Rate, plan.Currency)}
	}
	return out, nil
}

func ratePrice(rate money.Rate, currency string) IntegrationAmount {
	return IntegrationAmount{Amount: rate.String(), Currency: currency}
}

// GetCurrentTariff serves CurrentTariff; vehicle_id is optional.
func (h *IntegrationReadHandler) GetCurrentTariff(w http.ResponseWriter, r *http.Request) {
	out, err := h.CurrentTariff(r.Context(), middleware.GetUserID(r.Context()), strings.TrimSpace(r.URL.Query().Get("vehicle_id")), time.Now())
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to read the current tariff")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
