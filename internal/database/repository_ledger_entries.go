package database

import (
	"context"
	"time"

	"github.com/teslacost/teslacost/internal/money"
)

// LedgerEntry is one line of the cost ledger with what is needed to tell it from the others: the amount is in the
// vehicle's currency, as in every total.
type LedgerEntry struct {
	Date     time.Time
	Category string // ENERGY, TOLL, PARKING, TRAVEL_OTHER, MAINTENANCE, REPAIR, INSURANCE, SUBSCRIPTION, TAX, FINANCING, TIRES...
	Source   string // charge_logs, fuel_logs, drive_expenses, maintenance_expenses, tires, vehicle_ownership
	Amount   money.Cents
	Note     *string // free text the owner typed (description, notes, tire brand)
	// Place is where a charge happened: the named place when the address is linked to one, else the reported address.
	Place         *string
	PlaceCategory *string
	Kwh           *float64
	Liters        *float64
	Recurring     bool
}

// LedgerFilter selects and orders the entries. Categories empty means every category except the acquisition,
// which is not a running cost.
type LedgerFilter struct {
	From, To     *time.Time // From inclusive, To exclusive
	Categories   []string
	MinAmount    *money.Cents
	SortByAmount bool
	Ascending    bool
	Limit        int
	Offset       int
}

// LedgerCategoryTotal is the sum of the filtered entries of one category.
type LedgerCategoryTotal struct {
	Category string
	Count    int
	Amount   money.Cents
}

const ledgerFilterSQL = `
	l.vehicle_id = $1
	AND ($2::timestamptz IS NULL OR l.entry_date >= $2)
	AND ($3::timestamptz IS NULL OR l.entry_date < $3)
	AND (CASE WHEN cardinality($4::text[]) = 0 THEN l.category <> 'ACQUISITION' ELSE l.category = ANY($4) END)
	AND ($5::numeric IS NULL OR l.amount_eur >= $5)`

// ListLedgerEntries lists the entries of a vehicle's cost ledger matching the filter, and the totals of all the
// matching entries by category (the page limit does not apply to them).
func (r *Repository) ListLedgerEntries(ctx context.Context, vehicleID, ownerID string, f LedgerFilter) ([]LedgerEntry, []LedgerCategoryTotal, error) {
	cats := f.Categories
	if cats == nil {
		cats = []string{}
	}
	var minAmount *float64
	if f.MinAmount != nil {
		v := f.MinAmount.Float()
		minAmount = &v
	}

	order := "l.entry_date"
	if f.SortByAmount {
		order = "l.amount_eur"
	}
	dir := "DESC"
	if f.Ascending {
		dir = "ASC"
	}
	rows, err := r.pool.Query(ctx, `
		SELECT l.entry_date, l.category, l.source_table, l.amount_eur,
		       CASE l.source_table
		            WHEN 'maintenance_expenses' THEN NULLIF(btrim(m.description), '')
		            WHEN 'drive_expenses' THEN NULLIF(btrim(e.notes), '')
		            WHEN 'fuel_logs' THEN NULLIF(btrim(fl.notes), '')
		            WHEN 'charge_logs' THEN NULLIF(btrim(c.notes), '')
		            WHEN 'tires' THEN NULLIF(btrim(t.brand || ' ' || t.model), '')
		       END,
		       COALESCE(p.name, NULLIF(btrim(c.address), '')), p.category,
		       c.kwh_added::float8, fl.liters::float8, COALESCE(m.is_recurring, false)
		FROM cost_ledger l
		LEFT JOIN maintenance_expenses m ON l.source_table = 'maintenance_expenses' AND m.id = l.source_id
		LEFT JOIN drive_expenses e ON l.source_table = 'drive_expenses' AND e.id = l.source_id
		LEFT JOIN fuel_logs fl ON l.source_table = 'fuel_logs' AND fl.id = l.source_id
		LEFT JOIN charge_logs c ON l.source_table = 'charge_logs' AND c.id = l.source_id
		LEFT JOIN tires t ON l.source_table = 'tires' AND t.id = l.source_id
		LEFT JOIN charging_place_aliases a ON a.user_id = $6 AND a.alias = `+placeAliasSQL("COALESCE(c.address, '')")+`
		LEFT JOIN charging_places p ON p.id = COALESCE(c.place_id, a.place_id) AND p.user_id = $6
		WHERE `+ledgerFilterSQL+`
		ORDER BY `+order+` `+dir+`, l.source_id, l.entry_date
		LIMIT $7 OFFSET $8;
	`, vehicleID, f.From, f.To, cats, minAmount, ownerID, f.Limit, f.Offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var entries []LedgerEntry
	for rows.Next() {
		var e LedgerEntry
		if err := rows.Scan(&e.Date, &e.Category, &e.Source, &e.Amount, &e.Note, &e.Place, &e.PlaceCategory, &e.Kwh, &e.Liters, &e.Recurring); err != nil {
			return nil, nil, err
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	totals, err := r.pool.Query(ctx, `
		SELECT l.category, COUNT(*), COALESCE(SUM(l.amount_eur), 0)
		FROM cost_ledger l
		WHERE `+ledgerFilterSQL+`
		GROUP BY l.category
		ORDER BY SUM(l.amount_eur) DESC, l.category;
	`, vehicleID, f.From, f.To, cats, minAmount)
	if err != nil {
		return nil, nil, err
	}
	defer totals.Close()
	var byCategory []LedgerCategoryTotal
	for totals.Next() {
		var t LedgerCategoryTotal
		if err := totals.Scan(&t.Category, &t.Count, &t.Amount); err != nil {
			return nil, nil, err
		}
		byCategory = append(byCategory, t)
	}
	return entries, byCategory, totals.Err()
}
