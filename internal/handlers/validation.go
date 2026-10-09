package handlers

import (
	"errors"
	"log/slog"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/money"
)

var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

var maintenanceCategories = map[string]bool{
	"MAINTENANCE":  true,
	"REPAIR":       true,
	"INSURANCE":    true,
	"SUBSCRIPTION": true,
	"TAX":          true,
	"FINANCING":    true,
	"ACCESSORY":    true,
	"OTHER":        true,
}

var driveExpenseTypes = map[string]bool{
	"TOLL":    true,
	"PARKING": true,
	"FERRY":   true,
	"OTHER":   true,
}

// writeRepoError maps repository errors to HTTP responses without leaking internal details.
// The 500 case is logged with the request ID (via the context-aware requestIDHandler set up
// in main.go) so it can be correlated with the corresponding chi access log line.
func isAPIError(err error) bool {
	_, ok := apierror.As(err)
	return ok
}

func writeRepoError(w http.ResponseWriter, r *http.Request, err error, fallback string) {
	switch {
	case isAPIError(err):
		writeErr(w, http.StatusBadRequest, err)
	case errors.Is(err, database.ErrNotFound):
		writeAPIError(w, http.StatusNotFound, apierror.New("request.not_found", "Item not found"))
	case errors.Is(err, database.ErrForeignReference):
		writeAPIError(w, http.StatusBadRequest, apierror.New("request.invalid_reference", "Invalid reference: the drive, group or tire does not exist for this vehicle"))
	default:
		slog.ErrorContext(r.Context(), fallback, "component", "api", "error", err)
		writeAPIError(w, http.StatusInternalServerError, apierror.New("internal", fallback))
	}
}

// parseDate accepts RFC3339 timestamps and YYYY-MM-DD dates.
func parseDate(value string, locations ...*time.Location) (time.Time, error) {
	value = strings.TrimSpace(value)
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02", value, dateLocation(locations)); err == nil {
		return t, nil
	}
	return time.Time{}, apierror.Newf("request.invalid_date_value", "Invalid date: %q", value)
}

// parseOptionalDate parses an optional date; empty values yield nil.
func parseOptionalDate(value *string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	t, err := parseDate(*value)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func validateAmount(amount money.Cents, allowZero bool) error {
	if amount > money.Max {
		return apierror.New("expense.amount_invalid", "Invalid amount")
	}
	if amount < 0 || (!allowZero && amount == 0) {
		return apierror.New("expense.amount_positive", "The amount must be positive")
	}
	return nil
}

// validateQuantity validates a non-monetary positive quantity (kWh, km).
func validateQuantity(v float64, max float64) error {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > max {
		return apierror.New("request.invalid_value", "Invalid value")
	}
	return nil
}

// normalizeCurrency defaults to the vehicle's own currency and requires a conversion rate to it
// for anything else — a one-off foreign expense (a toll paid abroad), not a change of the
// vehicle's base currency, which is fixed at creation.
func normalizeCurrency(currency, baseCurrency string, fxRate *float64) (string, *float64, error) {
	cur := strings.ToUpper(strings.TrimSpace(currency))
	if cur == "" {
		cur = baseCurrency
	}
	if !currencyPattern.MatchString(cur) {
		return "", nil, apierror.Newf("expense.currency_invalid", "Invalid currency: %q", currency)
	}
	if cur == baseCurrency {
		return cur, nil, nil
	}
	if fxRate == nil || math.IsNaN(*fxRate) || math.IsInf(*fxRate, 0) || *fxRate <= 0 {
		return "", nil, apierror.Newf("expense.fx_required", "A conversion rate to %s is required for an expense in %s", baseCurrency, cur)
	}
	return cur, fxRate, nil
}
