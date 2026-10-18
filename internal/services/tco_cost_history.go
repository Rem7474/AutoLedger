package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/teslacost/teslacost/internal/money"
)

// The families of recorded costs whose monthly history a page shows. Each one is the list of that page: the ledger
// rows of one source table in a set of categories, recurring entries expanded to the months they were due.
const (
	CostHistoryTolls   = "tolls"
	CostHistoryService = "service"
	CostHistoryFixed   = "fixed"
)

// ErrUnsupportedCostHistoryKind is returned for a kind that names no family of costs.
var ErrUnsupportedCostHistoryKind = errors.New("unsupported cost history kind")

type costHistorySource struct {
	table      string
	categories []string
}

var costHistorySources = map[string]costHistorySource{
	CostHistoryTolls:   {"drive_expenses", []string{LedgerToll, LedgerParking, LedgerTravelOther}},
	CostHistoryService: {"maintenance_expenses", []string{LedgerMaintenance, LedgerRepair, "ACCESSORY", "OTHER"}},
	CostHistoryFixed:   {"maintenance_expenses", []string{LedgerInsurance, LedgerSubscr, LedgerTax, LedgerFinancing}},
}

// CostHistoryMonth is what was paid in one calendar month of the reporting timezone.
type CostHistoryMonth struct {
	Month  string      `json:"month"` // YYYY-MM
	Amount money.Cents `json:"amount"`
}

// CostHistory is the monthly cash history of one family of costs.
type CostHistory struct {
	Kind         string             `json:"kind"`
	CurrentMonth string             `json:"current_month"`
	Months       []CostHistoryMonth `json:"months"`
}

// CostHistory lists, month by month and in the reporting timezone, what a vehicle paid for one family of costs. The
// months are continuous from the first record to the current one (a later recorded month extends them), a month
// without a record is a zero, so a range of N months is N calendar months.
func (s *TCOService) CostHistory(ctx context.Context, vehicleID, kind string) (*CostHistory, error) {
	src, ok := costHistorySources[kind]
	if !ok {
		return nil, ErrUnsupportedCostHistoryKind
	}
	rows, err := s.pool.Query(ctx, `
		SELECT TO_CHAR(entry_date AT TIME ZONE $2, 'YYYY-MM') AS m, SUM(amount_eur)
		FROM cost_ledger
		WHERE vehicle_id = $1 AND source_table = $3 AND category = ANY($4)
		GROUP BY m;
	`, vehicleID, s.timezone, src.table, src.categories)
	if err != nil {
		return nil, fmt.Errorf("cost history: %w", err)
	}
	defer rows.Close()

	byMonth := map[string]money.Cents{}
	for rows.Next() {
		var m string
		var amount money.Cents
		if err := rows.Scan(&m, &amount); err != nil {
			return nil, err
		}
		byMonth[m] = amount
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	current := time.Now().In(s.location()).Format("2006-01")
	return &CostHistory{Kind: kind, CurrentMonth: current, Months: continuousCostMonths(byMonth, current)}, nil
}

// continuousCostMonths turns the recorded months into every month from the first one to the latest of the last
// recorded and the current one. Without a record it is the current month alone.
func continuousCostMonths(byMonth map[string]money.Cents, current string) []CostHistoryMonth {
	if len(byMonth) == 0 {
		return []CostHistoryMonth{{Month: current}}
	}
	recorded := make([]string, 0, len(byMonth))
	for m := range byMonth {
		recorded = append(recorded, m)
	}
	sort.Strings(recorded)
	first, err := time.Parse("2006-01", recorded[0])
	if err != nil {
		return []CostHistoryMonth{{Month: current}}
	}
	last := recorded[len(recorded)-1]
	if current > last {
		last = current
	}
	var out []CostHistoryMonth
	// time.Date normalises a month past December, so the walk needs no AddDate
	for i := 0; ; i++ {
		key := time.Date(first.Year(), first.Month()+time.Month(i), 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
		if key > last {
			return out
		}
		out = append(out, CostHistoryMonth{Month: key, Amount: byMonth[key]})
	}
}
