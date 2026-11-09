package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/money"
)

// CostEntry is one line of the cost ledger with the address of the record it comes from.
type CostEntry struct {
	IntegrationExpense
	Source   string `json:"source"`
	SourceID string `json:"source_id"`
}

// CostEntries is a page of a vehicle's cost lines for a period.
type CostEntries struct {
	VehicleID  string                       `json:"vehicle_id"`
	Currency   string                       `json:"currency"`
	Count      int                          `json:"count"`
	PaidTotal  IntegrationAmount            `json:"paid_total"`
	ByCategory []IntegrationExpenseCategory `json:"by_category"`
	// CarpoolReceived is what the passengers paid for the carpools dated in the period; 0 when the lines are
	// filtered by category or amount, since the filtered lines are then not the whole cost.
	CarpoolReceived IntegrationAmount `json:"carpool_received"`
	NetTotal        IntegrationAmount `json:"net_total"`
	// DistanceKm is the distance of the drives that began in the period, the base of a cost per kilometre.
	DistanceKm float64     `json:"distance_km"`
	Entries    []CostEntry `json:"entries"`
	NextOffset *int        `json:"next_offset"`
}

// GetCostEntries lists the recorded cost lines of a vehicle: the page, and the totals of every matching line.
func (h *TCOHandler) GetCostEntries(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	vehicleID := chi.URLParam(r, "vehicleId")
	vehicle, err := h.repo.GetVehicleByID(r.Context(), vehicleID, userID)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, apierror.New("vehicle.not_found", "Vehicle not found"))
		return
	}

	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	var categories []string
	for _, c := range query["category"] {
		categories = append(categories, strings.Split(c, ",")...)
	}
	loc := h.tcoService.Location()
	f, err := expenseFilter(IntegrationExpenseQuery{
		From: query.Get("from"), To: query.Get("to"), Categories: categories, MinAmount: query.Get("min_amount"),
		Sort: query.Get("sort"), Ascending: query.Get("order") == "asc", Limit: limit,
		Cursor: encodeExpenseCursor(max(atoiOrZero(query.Get("offset")), 0)),
	}, loc)
	if err != nil {
		if apiErr, ok := apierror.As(err); ok {
			writeAPIError(w, http.StatusBadRequest, apiErr)
			return
		}
		writeRepoError(w, r, err, "Failed to list the cost entries")
		return
	}

	entries, totals, err := h.repo.ListLedgerEntries(r.Context(), vehicle.ID, vehicle.UserID, f)
	if err != nil {
		writeRepoError(w, r, err, "Failed to list the cost entries")
		return
	}
	page := buildIntegrationExpenses(vehicle.ID, vehicle.Currency, loc, f, entries, totals)
	out := CostEntries{
		VehicleID: vehicle.ID, Currency: vehicle.Currency, Count: page.Count, PaidTotal: page.Total,
		ByCategory: page.ByCategory, Entries: make([]CostEntry, len(entries)),
	}
	for i, e := range entries {
		out.Entries[i] = CostEntry{IntegrationExpense: page.Expenses[i], Source: e.Source, SourceID: e.SourceID}
	}
	if page.NextCursor != nil {
		next := f.Offset + len(entries)
		out.NextOffset = &next
	}

	var paid, received money.Cents
	for _, t := range totals {
		paid += t.Amount
	}
	if len(f.Categories) == 0 && f.MinAmount == nil {
		if received, err = h.repo.CarpoolRevenueBetween(r.Context(), vehicle.ID, f.From, f.To); err != nil {
			writeRepoError(w, r, err, "Failed to list the cost entries")
			return
		}
	}
	if out.DistanceKm, err = h.repo.DriveDistanceBetween(r.Context(), vehicle.ID, f.From, f.To); err != nil {
		writeRepoError(w, r, err, "Failed to list the cost entries")
		return
	}
	out.CarpoolReceived = amountOf(received, vehicle.Currency)
	out.NetTotal = amountOf(paid-received, vehicle.Currency)
	writeJSON(w, http.StatusOK, out)
}

func atoiOrZero(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
