package services

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

// PlaceGroup is the charging of one place (or of the charges linked to none) over the period.
// PricePerKwh is the recorded cost over the energy of the sessions whose cost is known, free ones included;
// the energy and the count of the sessions without a cost are reported next to it, never in its denominator.
type PlaceGroup struct {
	Key             string      `json:"key"` // place id; empty for the charges linked to no place
	Name            string      `json:"name,omitempty"`
	Category        string      `json:"category,omitempty"`
	Sessions        int         `json:"sessions"`
	KwhAdded        float64     `json:"kwh_added"`
	Cost            money.Cents `json:"cost"`
	PricedKwh       float64     `json:"priced_kwh"`
	PricePerKwh     *float64    `json:"price_per_kwh"`
	FreeSessions    int         `json:"free_sessions"`
	UnknownSessions int         `json:"unknown_sessions"`
	UnknownKwh      float64     `json:"unknown_kwh"`
}

// PlaceMonthGroup is one group's share of a month.
type PlaceMonthGroup struct {
	Key       string      `json:"key"`
	KwhAdded  float64     `json:"kwh_added"`
	Cost      money.Cents `json:"cost"`
	PricedKwh float64     `json:"priced_kwh"`
}

// PlaceMonth is the charging of one month, in total and by group.
type PlaceMonth struct {
	Month       string            `json:"month"`
	KwhAdded    float64           `json:"kwh_added"`
	Cost        money.Cents       `json:"cost"`
	PricedKwh   float64           `json:"priced_kwh"`
	PricePerKwh *float64          `json:"price_per_kwh"`
	Groups      []PlaceMonthGroup `json:"groups"`
}

// Reasons the home substitution cannot be estimated.
const (
	HomeSavingsNoHome = "no_home"
	HomeSavingsNoAway = "no_away"
)

// HomeSavings compares what charging away from home cost with the same energy at the home price.
// Only charges classified in a non-home place count as away: an unclassified charge is not assumed to be away.
// Savings is negative when charging away was cheaper than at home.
type HomeSavings struct {
	Reason           string      `json:"reason,omitempty"`
	HomePricePerKwh  *float64    `json:"home_price_per_kwh"`
	HomePricedKwh    float64     `json:"home_priced_kwh"`
	AwaySessions     int         `json:"away_sessions"`
	AwayPricedKwh    float64     `json:"away_priced_kwh"`
	AwayCost         money.Cents `json:"away_cost"`
	HypotheticalCost money.Cents `json:"hypothetical_cost"`
	Savings          money.Cents `json:"savings"`
	UnclassifiedKwh  float64     `json:"unclassified_kwh"`
}

// PlaceStatsSummary totals the period.
type PlaceStatsSummary struct {
	Sessions        int         `json:"sessions"`
	KwhAdded        float64     `json:"kwh_added"`
	Cost            money.Cents `json:"cost"`
	PricedKwh       float64     `json:"priced_kwh"`
	PricePerKwh     *float64    `json:"price_per_kwh"`
	FreeSessions    int         `json:"free_sessions"`
	UnknownSessions int         `json:"unknown_sessions"`
	UnknownKwh      float64     `json:"unknown_kwh"`
	// ZeroKwhSessions are sessions that recorded no energy; they are in none of the counts above.
	ZeroKwhSessions int `json:"zero_kwh_sessions"`
}

// ChargingPlaceStats is the charging of a vehicle by month and by place.
type ChargingPlaceStats struct {
	Months      int               `json:"months"` // 0 = whole history
	Summary     PlaceStatsSummary `json:"summary"`
	Groups      []PlaceGroup      `json:"groups"`
	Series      []PlaceMonth      `json:"series"`
	HomeSavings HomeSavings       `json:"home_savings"`
}

// placeRow is one (month, place) aggregate read from the database.
type placeRow struct {
	Month           string
	Key             string
	Name            string
	Category        string
	Sessions        int
	KwhAdded        float64
	Cost            money.Cents
	PricedKwh       float64
	FreeSessions    int
	UnknownSessions int
	UnknownKwh      float64
	ZeroKwhSessions int
}

func priceRatio(cost money.Cents, kwh float64) *float64 {
	if kwh <= 0 {
		return nil
	}
	v := math.Round(cost.Float()/kwh*10000) / 10000
	return &v
}

// buildPlaceStats folds the aggregates into groups, a monthly series, a summary and the home substitution.
func buildPlaceStats(rows []placeRow, months int) *ChargingPlaceStats {
	out := &ChargingPlaceStats{Months: months, Groups: []PlaceGroup{}, Series: []PlaceMonth{}}
	groups := map[string]*PlaceGroup{}
	byMonth := map[string]*PlaceMonth{}

	for _, r := range rows {
		out.Summary.ZeroKwhSessions += r.ZeroKwhSessions

		g := groups[r.Key]
		if g == nil {
			g = &PlaceGroup{Key: r.Key, Name: r.Name, Category: r.Category}
			groups[r.Key] = g
		}
		g.Sessions += r.Sessions
		g.KwhAdded += r.KwhAdded
		g.Cost += r.Cost
		g.PricedKwh += r.PricedKwh
		g.FreeSessions += r.FreeSessions
		g.UnknownSessions += r.UnknownSessions
		g.UnknownKwh += r.UnknownKwh

		m := byMonth[r.Month]
		if m == nil {
			m = &PlaceMonth{Month: r.Month, Groups: []PlaceMonthGroup{}}
			byMonth[r.Month] = m
		}
		m.KwhAdded += r.KwhAdded
		m.Cost += r.Cost
		m.PricedKwh += r.PricedKwh
		m.Groups = append(m.Groups, PlaceMonthGroup{Key: r.Key, KwhAdded: round1(r.KwhAdded), Cost: r.Cost, PricedKwh: round1(r.PricedKwh)})

		s := &out.Summary
		s.Sessions += r.Sessions
		s.KwhAdded += r.KwhAdded
		s.Cost += r.Cost
		s.PricedKwh += r.PricedKwh
		s.FreeSessions += r.FreeSessions
		s.UnknownSessions += r.UnknownSessions
		s.UnknownKwh += r.UnknownKwh
	}

	for _, g := range groups {
		g.PricePerKwh = priceRatio(g.Cost, g.PricedKwh)
		g.KwhAdded, g.PricedKwh, g.UnknownKwh = round1(g.KwhAdded), round1(g.PricedKwh), round1(g.UnknownKwh)
		out.Groups = append(out.Groups, *g)
	}
	sort.Slice(out.Groups, func(i, j int) bool {
		if out.Groups[i].KwhAdded != out.Groups[j].KwhAdded {
			return out.Groups[i].KwhAdded > out.Groups[j].KwhAdded
		}
		return out.Groups[i].Key < out.Groups[j].Key
	})

	for _, m := range byMonth {
		m.PricePerKwh = priceRatio(m.Cost, m.PricedKwh)
		m.KwhAdded, m.PricedKwh = round1(m.KwhAdded), round1(m.PricedKwh)
		sort.Slice(m.Groups, func(i, j int) bool { return m.Groups[i].Key < m.Groups[j].Key })
		out.Series = append(out.Series, *m)
	}
	sort.Slice(out.Series, func(i, j int) bool { return out.Series[i].Month < out.Series[j].Month })

	s := &out.Summary
	s.PricePerKwh = priceRatio(s.Cost, s.PricedKwh)
	s.KwhAdded, s.PricedKwh, s.UnknownKwh = round1(s.KwhAdded), round1(s.PricedKwh), round1(s.UnknownKwh)

	out.HomeSavings = homeSavings(groups)
	return out
}

func homeSavings(groups map[string]*PlaceGroup) HomeSavings {
	var h HomeSavings
	var homeCost money.Cents
	for _, g := range groups {
		switch g.Category {
		case models.PlaceHome:
			homeCost += g.Cost
			h.HomePricedKwh += g.PricedKwh
		case models.PlaceWork, models.PlacePublic, models.PlaceOther:
			h.AwaySessions += g.Sessions
			h.AwayPricedKwh += g.PricedKwh
			h.AwayCost += g.Cost
		default:
			h.UnclassifiedKwh += g.KwhAdded
		}
	}
	h.UnclassifiedKwh = round1(h.UnclassifiedKwh)
	h.HomePricedKwh = round1(h.HomePricedKwh)
	h.AwayPricedKwh = round1(h.AwayPricedKwh)
	if h.HomePricedKwh <= 0 {
		h.Reason = HomeSavingsNoHome
		return h
	}
	price := homeCost.Float() / h.HomePricedKwh
	h.HomePricePerKwh = priceRatio(homeCost, h.HomePricedKwh)
	if h.AwayPricedKwh <= 0 {
		h.Reason = HomeSavingsNoAway
		return h
	}
	h.HypotheticalCost = money.FromFloat(h.AwayPricedKwh * price)
	h.Savings = h.AwayCost - h.HypotheticalCost
	return h
}

// PlaceStats reads the charges of a vehicle and groups them by place. ownerID is the vehicle's owner, whose
// places classify the charges; months limits the period to the current and the previous months (0 = all).
func (s *EnergyStatsService) PlaceStats(ctx context.Context, vehicleID, ownerID string, months int, now time.Time) (*ChargingPlaceStats, error) {
	var since *time.Time
	if months > 0 {
		loc, err := time.LoadLocation(s.timezone)
		if err != nil {
			loc = time.UTC
		}
		n := now.In(loc)
		start := time.Date(n.Year(), n.Month(), 1, 0, 0, 0, 0, loc).AddDate(0, -(months - 1), 0)
		since = &start
	}

	rows, err := s.pool.Query(ctx, `
		SELECT TO_CHAR(c.date AT TIME ZONE $3, 'YYYY-MM') AS m,
		       COALESCE(COALESCE(c.place_id, a.place_id)::text, ''),
		       COALESCE(p.name, ''), COALESCE(p.category, ''),
		       COUNT(*) FILTER (WHERE c.kwh_added > 0),
		       COALESCE(SUM(c.kwh_added) FILTER (WHERE c.kwh_added > 0), 0)::float8,
		       COALESCE(SUM(l.amount_eur) FILTER (WHERE c.kwh_added > 0), 0),
		       COALESCE(SUM(c.kwh_added) FILTER (WHERE c.kwh_added > 0 AND l.source_id IS NOT NULL), 0)::float8,
		       COUNT(*) FILTER (WHERE c.kwh_added > 0 AND l.source_id IS NOT NULL AND l.amount_eur = 0),
		       COUNT(*) FILTER (WHERE c.kwh_added > 0 AND l.source_id IS NULL),
		       COALESCE(SUM(c.kwh_added) FILTER (WHERE c.kwh_added > 0 AND l.source_id IS NULL), 0)::float8,
		       COUNT(*) FILTER (WHERE c.kwh_added <= 0)
		FROM charge_logs c
		LEFT JOIN charging_place_aliases a
		       ON a.user_id = $2 AND a.alias = lower(regexp_replace(btrim(COALESCE(c.address, '')), '\s+', ' ', 'g'))
		LEFT JOIN charging_places p ON p.id = COALESCE(c.place_id, a.place_id) AND p.user_id = $2
		LEFT JOIN cost_ledger l ON l.source_table = 'charge_logs' AND l.source_id = c.id
		WHERE c.vehicle_id = $1 AND c.deleted_upstream_at IS NULL
		  AND ($4::timestamptz IS NULL OR c.date >= $4)
		GROUP BY m, COALESCE(c.place_id, a.place_id), p.name, p.category
		ORDER BY m;
	`, vehicleID, ownerID, s.timezone, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []placeRow
	for rows.Next() {
		var r placeRow
		if err := rows.Scan(&r.Month, &r.Key, &r.Name, &r.Category, &r.Sessions, &r.KwhAdded, &r.Cost, &r.PricedKwh,
			&r.FreeSessions, &r.UnknownSessions, &r.UnknownKwh, &r.ZeroKwhSessions); err != nil {
			return nil, err
		}
		data = append(data, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return buildPlaceStats(data, months), nil
}

// EmptyPlaceStats is the answer for a vehicle that cannot charge.
func EmptyPlaceStats(months int) *ChargingPlaceStats {
	return buildPlaceStats(nil, months)
}
