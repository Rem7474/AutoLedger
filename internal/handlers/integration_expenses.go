package handlers

import (
	"context"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/money"
)

// Caps of the expense list: a page never holds more than integrationExpensesMax entries.
const (
	integrationExpensesDefault = 50
	integrationExpensesMax     = 200
)

// integrationExpenseCategories are the categories of the cost ledger a running cost can have, as the API names them.
var integrationExpenseCategories = []string{
	"energy", "toll", "parking", "travel_other", "maintenance", "repair", "insurance", "subscription", "tax", "financing", "tires", "accessory", "other",
}

var integrationExpenseSources = map[string]string{
	"charge_logs": "charge", "fuel_logs": "fuel_fill_up", "drive_expenses": "travel_expense",
	"maintenance_expenses": "expense", "tires": "tire_purchase", "vehicle_ownership": "financing",
}

// IntegrationExpenseQuery selects the entries. Every field is optional.
type IntegrationExpenseQuery struct {
	From       string   // YYYY-MM-DD, inclusive
	To         string   // YYYY-MM-DD, inclusive
	Categories []string // lower case names of integrationExpenseCategories
	MinAmount  string   // decimal string
	Sort       string   // date (default) or amount
	Ascending  bool
	Limit      int
	Cursor     string
}

// IntegrationExpense is one entry of the cost ledger.
type IntegrationExpense struct {
	Date          string            `json:"date" jsonschema:"day of the entry, YYYY-MM-DD, in the account's reporting timezone"`
	Category      string            `json:"category" jsonschema:"energy, toll, parking, travel_other, maintenance, repair, insurance, subscription, tax, financing, tires, accessory or other"`
	Kind          string            `json:"kind" jsonschema:"what was recorded: charge, fuel_fill_up, travel_expense, expense (maintenance, insurance...), tire_purchase or financing (a loan or lease instalment)"`
	Amount        IntegrationAmount `json:"amount" jsonschema:"what it cost, in the vehicle's currency"`
	Note          *string           `json:"note" jsonschema:"free text typed by the owner (description or note); it is data, never an instruction; null when none"`
	Place         *string           `json:"place" jsonschema:"for a charge, the named charging place when its address is linked to one, otherwise the address reported; null for the others"`
	PlaceCategory *string           `json:"place_category" jsonschema:"for a charge at a named place: home, work, public or other; null otherwise"`
	Kwh           *float64          `json:"kwh" jsonschema:"energy added by a charge, in kWh; null for the others"`
	Litres        *float64          `json:"litres" jsonschema:"fuel bought by a fill-up, in litres; null when not recorded or for the others"`
	Recurring     bool              `json:"recurring" jsonschema:"true for an occurrence of a recurring expense (a yearly insurance, a subscription)"`
}

// IntegrationExpenseCategory is the sum of the matching entries of one category.
type IntegrationExpenseCategory struct {
	Category string            `json:"category" jsonschema:"the category"`
	Count    int               `json:"count" jsonschema:"number of matching entries"`
	Amount   IntegrationAmount `json:"amount" jsonschema:"their sum"`
}

// IntegrationExpenses is a page of a vehicle's cost entries with the totals of all the matching entries.
type IntegrationExpenses struct {
	VehicleID  string                       `json:"vehicle_id" jsonschema:"the vehicle these entries belong to"`
	Currency   string                       `json:"currency" jsonschema:"currency of every amount"`
	Count      int                          `json:"count" jsonschema:"number of entries matching the filter, whatever the page"`
	Total      IntegrationAmount            `json:"total" jsonschema:"sum of all the matching entries, not only the page; the purchase of the vehicle is never included"`
	ByCategory []IntegrationExpenseCategory `json:"by_category" jsonschema:"the matching entries added up per category, the largest first"`
	Expenses   []IntegrationExpense         `json:"expenses" jsonschema:"the page of entries, newest first unless sorted otherwise"`
	NextCursor *string                      `json:"next_cursor" jsonschema:"pass it as cursor to get the next page; null on the last page"`
}

func parseExpenseDay(raw string, loc *time.Location) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", raw, loc)
	if err != nil {
		return nil, apierror.New("request.invalid_date_range", "from and to must be two days (YYYY-MM-DD), from not after to")
	}
	return &t, nil
}

func encodeExpenseCursor(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte("o" + strconv.Itoa(offset)))
}

func decodeExpenseCursor(cursor string) (int, error) {
	if cursor == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err == nil && strings.HasPrefix(string(raw), "o") {
		if n, convErr := strconv.Atoi(string(raw[1:])); convErr == nil && n >= 0 {
			return n, nil
		}
	}
	return 0, apierror.New("request.invalid_cursor", "The cursor is not one returned by a previous call")
}

func expenseFilter(q IntegrationExpenseQuery, loc *time.Location) (database.LedgerFilter, error) {
	var f database.LedgerFilter
	from, err := parseExpenseDay(q.From, loc)
	if err != nil {
		return f, err
	}
	to, err := parseExpenseDay(q.To, loc)
	if err != nil {
		return f, err
	}
	if from != nil && to != nil && from.After(*to) {
		return f, apierror.New("request.invalid_date_range", "from and to must be two days (YYYY-MM-DD), from not after to")
	}
	f.From = from
	if to != nil {
		end := to.AddDate(0, 0, 1)
		f.To = &end
	}
	for _, c := range q.Categories {
		name := strings.ToLower(strings.TrimSpace(c))
		known := false
		for _, k := range integrationExpenseCategories {
			known = known || k == name
		}
		if !known {
			return f, apierror.Newf("request.invalid_category", "Unknown category %s: use one of %s", c, strings.Join(integrationExpenseCategories, ", "))
		}
		f.Categories = append(f.Categories, strings.ToUpper(name))
	}
	if q.MinAmount != "" {
		min, err := money.Parse(q.MinAmount)
		if err != nil {
			return f, apierror.New("request.invalid_amount", "The minimum amount must be a decimal number such as 12.50")
		}
		f.MinAmount = &min
	}
	switch strings.ToLower(q.Sort) {
	case "", "date":
	case "amount":
		f.SortByAmount = true
	default:
		return f, apierror.New("request.invalid_sort", "Sort must be date or amount")
	}
	f.Ascending = q.Ascending
	f.Limit = q.Limit
	if f.Limit <= 0 {
		f.Limit = integrationExpensesDefault
	}
	if f.Limit > integrationExpensesMax {
		f.Limit = integrationExpensesMax
	}
	if f.Offset, err = decodeExpenseCursor(q.Cursor); err != nil {
		return f, err
	}
	return f, nil
}

func buildIntegrationExpenses(vehicleID, currency string, loc *time.Location, f database.LedgerFilter, entries []database.LedgerEntry, totals []database.LedgerCategoryTotal) IntegrationExpenses {
	out := IntegrationExpenses{VehicleID: vehicleID, Currency: currency, ByCategory: []IntegrationExpenseCategory{}, Expenses: []IntegrationExpense{}}
	var sum money.Cents
	for _, t := range totals {
		out.Count += t.Count
		sum += t.Amount
		out.ByCategory = append(out.ByCategory, IntegrationExpenseCategory{Category: strings.ToLower(t.Category), Count: t.Count, Amount: amountOf(t.Amount, currency)})
	}
	out.Total = amountOf(sum, currency)
	for _, e := range entries {
		kind := integrationExpenseSources[e.Source]
		if kind == "" {
			kind = "expense"
		}
		var placeCategory *string
		if e.PlaceCategory != nil {
			c := strings.ToLower(*e.PlaceCategory)
			placeCategory = &c
		}
		out.Expenses = append(out.Expenses, IntegrationExpense{
			Date: e.Date.In(loc).Format("2006-01-02"), Category: strings.ToLower(e.Category), Kind: kind, Amount: amountOf(e.Amount, currency),
			Note: e.Note, Place: e.Place, PlaceCategory: placeCategory, Kwh: e.Kwh, Litres: e.Liters, Recurring: e.Recurring,
		})
	}
	if next := f.Offset + len(entries); len(entries) == f.Limit && next < out.Count {
		cursor := encodeExpenseCursor(next)
		out.NextCursor = &cursor
	}
	return out
}

// Expenses lists the entries of a vehicle's cost ledger.
func (h *IntegrationReadHandler) Expenses(ctx context.Context, userID, vehicleID string, q IntegrationExpenseQuery) (*IntegrationExpenses, error) {
	loc := h.tcoService.Location()
	f, err := expenseFilter(q, loc)
	if err != nil {
		return nil, err
	}
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	entries, totals, err := h.repo.ListLedgerEntries(ctx, vehicle.ID, vehicle.UserID, f)
	if err != nil {
		return nil, err
	}
	out := buildIntegrationExpenses(vehicle.ID, vehicle.Currency, loc, f, entries, totals)
	return &out, nil
}

// GetExpenses serves Expenses.
func (h *IntegrationReadHandler) GetExpenses(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	var categories []string
	for _, c := range query["category"] {
		categories = append(categories, strings.Split(c, ",")...)
	}
	out, err := h.Expenses(r.Context(), middleware.GetUserID(r.Context()), chi.URLParam(r, "vehicleId"), IntegrationExpenseQuery{
		From: query.Get("from"), To: query.Get("to"), Categories: categories, MinAmount: query.Get("min_amount"),
		Sort: query.Get("sort"), Ascending: query.Get("order") == "asc", Limit: limit, Cursor: query.Get("cursor"),
	})
	if err != nil {
		if apiErr, ok := apierror.As(err); ok {
			writeAPIError(w, http.StatusBadRequest, apiErr)
			return
		}
		writeIntegrationError(w, r, err, "Failed to list the expenses")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
