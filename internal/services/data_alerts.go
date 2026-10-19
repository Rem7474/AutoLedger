package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/teslacost/teslacost/internal/money"
)

// Thresholds of the data alerts. They are relative to the vehicle's own records, so no table per currency is needed.
const (
	// alertMinComparable is the number of other records an entry is compared with: under it nothing is flagged.
	alertMinComparable = 5
	// alertRecentDays is how long an entry stays flagged. Alerts are computed at each load and cannot be dismissed,
	// so an entry stops being reported once it is older than this.
	alertRecentDays = 60
	// alertHistoryDays is the history an entry is compared with.
	alertHistoryDays = 365
	// alertAmountFactor is how many times the median of its category an expense must cost to be reported.
	alertAmountFactor = 5.0
	// alertPriceMargin is how far past the vehicle's own lowest or highest unit price a price must be.
	alertPriceMargin = 0.5
)

// Kinds of data alert.
const (
	AlertAmountOutlier = "amount_outlier"
	AlertPriceOutlier  = "price_outlier"
)

// DataAlert is one entry that looks mistyped. At most one alert of each kind is reported, the most deviant one.
type DataAlert struct {
	Kind string `json:"kind"`
	// Category is the ledger category of an amount outlier ("TOLL", "PARKING", "TRAVEL_OTHER") and the energy of a
	// price outlier ("CHARGE" or "FUEL").
	Category string    `json:"category"`
	SourceID string    `json:"source_id"`
	Date     time.Time `json:"date"`
	// Amount is the entry; Median is the median of its category (amount outliers).
	Amount money.Cents `json:"amount,omitempty"`
	Median money.Cents `json:"median,omitempty"`
	// UnitPrice is the price per kWh or per litre; Low and High bound the vehicle's other prices (price outliers).
	UnitPrice float64 `json:"unit_price,omitempty"`
	Low       float64 `json:"low,omitempty"`
	High      float64 `json:"high,omitempty"`
}

// alertEntry is one record under test: its value is an amount or a unit price.
type alertEntry struct {
	id    string
	at    time.Time
	value float64
}

// alertHit is the entry that deviates the most and what it deviates from.
type alertHit struct {
	entry         alertEntry
	median        float64
	low, high     float64
	deviationRank float64
}

func (e alertEntry) recent(now time.Time) bool {
	return !e.at.Before(now.AddDate(0, 0, -alertRecentDays))
}

// others is the values of every entry but the i-th.
func others(entries []alertEntry, i int) []float64 {
	out := make([]float64, 0, len(entries)-1)
	for j, e := range entries {
		if j != i {
			out = append(out, e.value)
		}
	}
	return out
}

// amountOutlier finds the recent expense that costs the most times the median of the others, when that is more than
// alertAmountFactor times.
func amountOutlier(entries []alertEntry, now time.Time) *alertHit {
	var best *alertHit
	for i, e := range entries {
		if !e.recent(now) || e.value <= 0 {
			continue
		}
		rest := others(entries, i)
		if len(rest) < alertMinComparable {
			continue
		}
		m := median(rest)
		if m <= 0 || e.value <= alertAmountFactor*m {
			continue
		}
		if rank := e.value / m; best == nil || rank > best.deviationRank {
			best = &alertHit{entry: e, median: m, deviationRank: rank}
		}
	}
	return best
}

// priceOutlier finds the recent unit price that lies the furthest beyond the range of the vehicle's other prices: more
// than alertPriceMargin above the highest or below the lowest.
func priceOutlier(entries []alertEntry, now time.Time) *alertHit {
	var best *alertHit
	for i, e := range entries {
		if !e.recent(now) || e.value <= 0 {
			continue
		}
		rest := others(entries, i)
		if len(rest) < alertMinComparable {
			continue
		}
		low, high := rest[0], rest[0]
		for _, v := range rest[1:] {
			low, high = math.Min(low, v), math.Max(high, v)
		}
		var rank float64
		switch {
		case high > 0 && e.value > (1+alertPriceMargin)*high:
			rank = e.value / high
		case low > 0 && e.value < (1-alertPriceMargin)*low:
			rank = low / e.value
		default:
			continue
		}
		if best == nil || rank > best.deviationRank {
			best = &alertHit{entry: e, low: low, high: high, deviationRank: rank}
		}
	}
	return best
}

// DataAlerts reports the entries of a vehicle that look mistyped: an expense far above the usual of its category, and an
// energy price far outside the vehicle's own range. Nothing is stored; the answer follows the records.
func (s *TCOService) DataAlerts(ctx context.Context, vehicleID string) ([]DataAlert, error) {
	now := time.Now()
	since := now.AddDate(0, 0, -alertHistoryDays)
	alerts := []DataAlert{}

	// A trip-group expense is a whole trip's cost, not comparable with a single one.
	rows, err := s.pool.Query(ctx, `
		SELECT l.category, l.source_id::text, l.entry_date, l.amount_eur
		FROM cost_ledger l
		LEFT JOIN drive_expenses e ON l.source_table = 'drive_expenses' AND e.id = l.source_id
		WHERE l.vehicle_id = $1 AND l.category IN ('TOLL', 'PARKING', 'TRAVEL_OTHER')
		  AND l.entry_date >= $2 AND l.amount_eur > 0 AND e.trip_group_id IS NULL;
	`, vehicleID, since)
	if err != nil {
		return nil, fmt.Errorf("expenses: %w", err)
	}
	byCategory := map[string][]alertEntry{}
	for rows.Next() {
		var category string
		var e alertEntry
		var amount money.Cents
		if err := rows.Scan(&category, &e.id, &e.at, &amount); err != nil {
			rows.Close()
			return nil, err
		}
		e.value = amount.Float()
		byCategory[category] = append(byCategory[category], e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var worst *DataAlert
	var worstRank float64
	for category, entries := range byCategory {
		if hit := amountOutlier(entries, now); hit != nil && (worst == nil || hit.deviationRank > worstRank) {
			worstRank = hit.deviationRank
			worst = &DataAlert{Kind: AlertAmountOutlier, Category: category, SourceID: hit.entry.id, Date: hit.entry.at,
				Amount: money.FromFloat(hit.entry.value), Median: money.FromFloat(hit.median)}
		}
	}
	if worst != nil {
		alerts = append(alerts, *worst)
	}

	// Unit prices, in the vehicle's currency only: a charge in another currency has no comparable price. A fuel price
	// is compared within one fuel type.
	prices := map[string][]alertEntry{}
	rows, err = s.pool.Query(ctx, `
		SELECT 'CHARGE', c.id::text, c.date, c.cost / c.kwh_added
		FROM charge_logs c JOIN vehicles v ON v.id = c.vehicle_id
		WHERE c.vehicle_id = $1 AND c.date >= $2 AND c.deleted_upstream_at IS NULL
		  AND c.currency = v.currency AND c.cost > 0 AND c.kwh_added > 0
		UNION ALL
		SELECT 'FUEL' || COALESCE(f.fuel_type, ''), f.id::text, f.date, f.amount / f.liters
		FROM fuel_logs f
		WHERE f.vehicle_id = $1 AND f.date >= $2 AND f.liters > 0;
	`, vehicleID, since)
	if err != nil {
		return nil, fmt.Errorf("unit prices: %w", err)
	}
	for rows.Next() {
		var group string
		var e alertEntry
		if err := rows.Scan(&group, &e.id, &e.at, &e.value); err != nil {
			rows.Close()
			return nil, err
		}
		prices[group] = append(prices[group], e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var priciest *DataAlert
	worstRank = 0
	for group, entries := range prices {
		if hit := priceOutlier(entries, now); hit != nil && (priciest == nil || hit.deviationRank > worstRank) {
			worstRank = hit.deviationRank
			category := "FUEL"
			if group == "CHARGE" {
				category = "CHARGE"
			}
			priciest = &DataAlert{Kind: AlertPriceOutlier, Category: category, SourceID: hit.entry.id, Date: hit.entry.at,
				UnitPrice: round3(hit.entry.value), Low: round3(hit.low), High: round3(hit.high)}
		}
	}
	if priciest != nil {
		alerts = append(alerts, *priciest)
	}
	return alerts, nil
}
