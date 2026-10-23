package services

import (
	"context"
	"errors"
	"math"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

// GetVehicleUnitRates calculates real cost rates based on vehicle history as of now, with flagged fallbacks.
// Insurance follows the same source priority as the TCO: recorded expenses first, then vehicle settings.
func (s *CarpoolService) GetVehicleUnitRates(ctx context.Context, vehicleID string) (*UnitRates, error) {
	return s.GetVehicleUnitRatesAt(ctx, vehicleID, nil)
}

// GetVehicleUnitRatesAt calculates real cost rates referencing a specific point in time (e.g. trip date).
// Electricity is estimated using the weighted average of the last 2 charges prior to ref if spaced by <= 5 days (or the single last charge),
// falling back to the vehicle's historical average if no recent charges exist.
func (s *CarpoolService) GetVehicleUnitRatesAt(ctx context.Context, vehicleID string, refTime *time.Time) (*UnitRates, error) {
	rates, err := s.GetVehicleUnitRatesAtMany(ctx, vehicleID, []*time.Time{refTime})
	if err != nil {
		return nil, err
	}
	return rates[0], nil
}

// DriveRates returns the unit rates in force when each drive ended (when it started, for a drive with no end),
// by drive id. A drive's cost is its distance and energy priced at these rates, so that the drive list, the cost
// by tag and a carpool leg all give a drive the same cost, and a cost entered afterwards (a charge, a tire, an
// expense dated before the drive) reaches it while a later one does not.
func (s *CarpoolService) DriveRates(ctx context.Context, vehicleID string, drives []models.Drive) (map[string]*UnitRates, error) {
	byDrive := make(map[string]*UnitRates, len(drives))
	if len(drives) == 0 {
		return byDrive, nil
	}
	refs := make([]*time.Time, len(drives))
	for i := range drives {
		t := driveReference(drives[i])
		refs[i] = &t
	}
	rates, err := s.GetVehicleUnitRatesAtMany(ctx, vehicleID, refs)
	if err != nil {
		return nil, err
	}
	for i, d := range drives {
		byDrive[d.ID] = rates[i]
	}
	return byDrive, nil
}

// driveReference is the instant a drive is priced at: its end, or its start when it has none.
func driveReference(d models.Drive) time.Time {
	if !d.EndTime.IsZero() {
		return d.EndTime
	}
	return d.StartTime
}

// unitRateInputs is what the rates of one reference point are computed from.
type unitRateInputs struct {
	totalDistance   float64
	recent          []chargeSample
	pricedCost      money.Cents
	pricedKwh       float64
	mountedTireRate float64
	totalTiresCost  money.Cents
	totalMaintCost  money.Cents
	windowStart     *time.Time
	insuranceCost   money.Cents
	windowKm        float64
	annualizedMaint money.Cents
}

type chargeSample struct {
	date time.Time
	kwh  float64
	cost money.Cents
}

// GetVehicleUnitRatesAtMany computes the rates at several reference points with one round trip per kind of input,
// whatever their number: the same queries as for a single point, run for each point of an array. A nil reference
// is now, priced on the wheels fitted today.
func (s *CarpoolService) GetVehicleUnitRatesAtMany(ctx context.Context, vehicleID string, refTimes []*time.Time) ([]*UnitRates, error) {
	n := len(refTimes)
	refs := make([]time.Time, n)
	days := make([]time.Time, n)
	historical := make([]bool, n)
	for i, rt := range refTimes {
		ref := time.Now()
		if rt != nil && !rt.IsZero() {
			ref, historical[i] = *rt, true
		}
		refs[i] = ref
		// Columns of type date (purchases, ledger entries) are bounded by the calendar day of the reference, in its
		// own location, so that every date clause agrees; instants (drive starts, fittings) keep the time of day.
		days[i] = time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, time.UTC)
	}
	in := make([]unitRateInputs, n)

	// Distance driven up to each reference. The drives are read once and summed by prefix, so the work does not grow
	// with the number of references times the number of drives.
	odometer, err := s.driveDistances(ctx, vehicleID)
	if err != nil {
		return nil, err
	}
	// The drives only count the trips the vehicle was fed, so a cost accumulated over the years is spread over the
	// distance the odometer shows when that is larger.
	anchors := s.odometerAnchors(ctx, vehicleID)
	anchorsAt := func(i int) []OdometerAnchor {
		if historical[i] {
			return anchorsUpTo(anchors, refs[i])
		}
		return anchors
	}
	for i, ref := range refs {
		in[i].totalDistance = coveredDistanceKm(odometer.between(nil, ref), anchorsAt(i), time.Time{}, ref)
	}

	// 1. Electricity rate (vehicle currency/kWh): try the last 2 priced charges prior to or at ref. A free charge is a
	// known price of zero, so only a missing cost is skipped.
	if err := s.eachRow(ctx, `
		SELECT r.i, c.date, c.kwh_added, c.cost
		FROM unnest($2::timestamptz[]) WITH ORDINALITY AS r(ref, i)
		CROSS JOIN LATERAL (
			SELECT date, kwh_added,
			       COALESCE(CASE WHEN currency = (SELECT currency FROM vehicles WHERE id = $1) THEN cost ELSE cost * fx_rate END, 0) AS cost
			FROM charge_logs
			WHERE vehicle_id = $1
			  AND deleted_upstream_at IS NULL
			  AND cost IS NOT NULL
			  AND (currency = (SELECT currency FROM vehicles WHERE id = $1) OR fx_rate IS NOT NULL)
			  AND date <= r.ref
			ORDER BY date DESC
			LIMIT 2
		) c
		ORDER BY r.i, c.date DESC`, []any{vehicleID, refs}, func(rows pgx.Rows) error {
		var idx int64
		var cs chargeSample
		if err := rows.Scan(&idx, &cs.date, &cs.kwh, &cs.cost); err != nil {
			return err
		}
		if cs.kwh > 0 {
			in[idx-1].recent = append(in[idx-1].recent, cs)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// 2. Fallback to all priced charges history if no recent charge was found
	if err := s.eachRow(ctx, `
		SELECT r.i,
		       COALESCE(SUM(CASE WHEN c.currency = (SELECT currency FROM vehicles WHERE id = $1) THEN c.cost ELSE c.cost * c.fx_rate END), 0),
		       COALESCE(SUM(c.kwh_added), 0)
		FROM unnest($2::timestamptz[]) WITH ORDINALITY AS r(ref, i)
		LEFT JOIN charge_logs c ON c.vehicle_id = $1 AND c.deleted_upstream_at IS NULL AND c.cost IS NOT NULL
		  AND c.date <= r.ref AND (c.currency = (SELECT currency FROM vehicles WHERE id = $1) OR c.fx_rate IS NOT NULL)
		GROUP BY r.i`, []any{vehicleID, refs}, func(rows pgx.Rows) error {
		var idx int64
		var cost money.Cents
		var kwh float64
		if err := rows.Scan(&idx, &cost, &kwh); err != nil {
			return err
		}
		in[idx-1].pricedCost, in[idx-1].pricedKwh = cost, kwh
		return nil
	}); err != nil {
		return nil, err
	}

	// Tires rate: use the fittings at the trip date for a historical estimate, not the wheels fitted today.
	if err := s.eachRow(ctx, `
		SELECT r.i,
		       COALESCE(SUM(t.purchase_price / GREATEST(t.estimated_lifespan_km - t.initial_distance_km, 1))
		                FILTER (WHERE
		                  (NOT r.historical AND t.current_position IN ('FL', 'FR', 'RL', 'RR'))
		                  OR (r.historical AND EXISTS (
		                    SELECT 1 FROM tire_mount_sessions ms WHERE ms.tire_id = t.id
		                      AND ms.vehicle_id = $1 AND ms.position IN ('FL', 'FR', 'RL', 'RR')
		                      AND ms.mounted_date <= r.ref
		                      AND (ms.dismounted_date IS NULL OR ms.dismounted_date > r.ref)
		                  ))), 0),
		       COALESCE(SUM(t.purchase_price), 0)
		FROM unnest($2::timestamptz[], $3::date[], $4::boolean[]) WITH ORDINALITY AS r(ref, day, historical, i)
		LEFT JOIN tires t ON t.vehicle_id = $1 AND t.purchase_date <= r.day
		GROUP BY r.i`, []any{vehicleID, refs, days, historical}, func(rows pgx.Rows) error {
		var idx int64
		var rate float64
		var total money.Cents
		if err := rows.Scan(&idx, &rate, &total); err != nil {
			return err
		}
		in[idx-1].mountedTireRate, in[idx-1].totalTiresCost = rate, total
		return nil
	}); err != nil {
		return nil, err
	}

	// Maintenance and repairs from the cost ledger
	if err := s.eachRow(ctx, `
		SELECT r.i, COALESCE(SUM(l.amount_eur), 0)
		FROM unnest($2::date[]) WITH ORDINALITY AS r(day, i)
		LEFT JOIN cost_ledger l ON l.vehicle_id = $1 AND l.category IN ('MAINTENANCE', 'REPAIR') AND l.entry_date <= r.day
		GROUP BY r.i`, []any{vehicleID, days}, func(rows pgx.Rows) error {
		var idx int64
		var total money.Cents
		if err := rows.Scan(&idx, &total); err != nil {
			return err
		}
		in[idx-1].totalMaintCost = total
		return nil
	}); err != nil {
		return nil, err
	}

	// Insurance: premiums paid over the window divided by the kilometers driven over the same window
	// The ledger is a view: the premiums are read once, not once per reference.
	if err := s.eachRow(ctx, `
		WITH premiums AS MATERIALIZED (
			SELECT entry_date, amount_eur FROM cost_ledger WHERE vehicle_id = $1 AND category = 'INSURANCE'
		)
		SELECT r.i, w.start,
		       COALESCE((SELECT SUM(amount_eur) FROM premiums WHERE entry_date <= r.ref AND entry_date >= w.start), 0)
		FROM unnest($2::timestamptz[]) WITH ORDINALITY AS r(ref, i)
		CROSS JOIN LATERAL (
			SELECT GREATEST(r.ref - INTERVAL '365 days', MIN(entry_date)) AS start FROM premiums
			WHERE entry_date <= r.ref
		) w`, []any{vehicleID, refs}, func(rows pgx.Rows) error {
		var idx int64
		var start *time.Time
		var cost money.Cents
		if err := rows.Scan(&idx, &start, &cost); err != nil {
			return err
		}
		in[idx-1].windowStart, in[idx-1].insuranceCost = start, cost
		if start != nil {
			in[idx-1].windowKm = coveredDistanceKm(odometer.between(start, refs[idx-1]), anchorsAt(int(idx-1)), *start, refs[idx-1])
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// Insurance recorded as maintenance expenses, annualised
	if err := s.eachRow(ctx, `
		SELECT r.i, COALESCE(SUM(
			CASE
				WHEN m.is_recurring AND COALESCE(m.recurrence_interval_months, 0) > 0
				THEN ROUND((CASE WHEN m.currency = (SELECT currency FROM vehicles WHERE id = $1) THEN m.amount ELSE m.amount * m.fx_rate END) * 12.0 / m.recurrence_interval_months, 2)
				ELSE CASE WHEN m.currency = (SELECT currency FROM vehicles WHERE id = $1) THEN m.amount ELSE m.amount * m.fx_rate END
			END
		), 0)
		FROM unnest($2::timestamptz[]) WITH ORDINALITY AS r(ref, i)
		LEFT JOIN maintenance_expenses m ON m.vehicle_id = $1 AND m.category = 'INSURANCE'
		  AND (m.currency = (SELECT currency FROM vehicles WHERE id = $1) OR m.fx_rate IS NOT NULL)
		  AND m.date <= r.ref
		  AND (m.recurrence_end_date IS NULL OR m.recurrence_end_date >= r.ref)
		GROUP BY r.i`, []any{vehicleID, refs}, func(rows pgx.Rows) error {
		var idx int64
		var total money.Cents
		if err := rows.Scan(&idx, &total); err != nil {
			return err
		}
		in[idx-1].annualizedMaint = total
		return nil
	}); err != nil {
		return nil, err
	}

	ownership, err := s.repo.GetVehicleOwnership(ctx, vehicleID)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return nil, err
	}

	out := make([]*UnitRates, n)
	for i := range refs {
		out[i] = ratesFromInputs(&in[i], refs[i], ownership)
	}
	return out, nil
}

// driveOdometer is the distance of a vehicle's drives by start time, as running totals.
type driveOdometer struct {
	starts []time.Time
	total  []float64 // total[i] is the distance of the drives starting at or before starts[i]
}

func (s *CarpoolService) driveDistances(ctx context.Context, vehicleID string) (*driveOdometer, error) {
	o := &driveOdometer{}
	var sum float64
	err := s.eachRow(ctx, `
		SELECT start_time, COALESCE(distance_km, 0) FROM drives
		WHERE vehicle_id = $1 AND deleted_upstream_at IS NULL
		ORDER BY start_time`, []any{vehicleID}, func(rows pgx.Rows) error {
		var start time.Time
		var km float64
		if err := rows.Scan(&start, &km); err != nil {
			return err
		}
		sum += km
		o.starts, o.total = append(o.starts, start), append(o.total, sum)
		return nil
	})
	return o, err
}

// upTo is the distance of the drives that started at or before t.
func (o *driveOdometer) upTo(t time.Time) float64 {
	n := sort.Search(len(o.starts), func(i int) bool { return o.starts[i].After(t) })
	if n == 0 {
		return 0
	}
	return o.total[n-1]
}

// between is the distance of the drives that started within [from, to], from the beginning when from is nil.
func (o *driveOdometer) between(from *time.Time, to time.Time) float64 {
	km := o.upTo(to)
	if from != nil {
		n := sort.Search(len(o.starts), func(i int) bool { return !o.starts[i].Before(*from) })
		if n > 0 {
			km -= o.total[n-1]
		}
	}
	return math.Max(0, math.Round(km*1e6)/1e6)
}

// eachRow runs a query and hands every row to fill, which scans it.
func (s *CarpoolService) eachRow(ctx context.Context, query string, args []any, fill func(rows pgx.Rows) error) error {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := fill(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

// ratesFromInputs turns the inputs of one reference point into its unit rates.
func ratesFromInputs(in *unitRateInputs, ref time.Time, ownership *models.VehicleOwnership) *UnitRates {
	rates := DefaultUnitRates()

	const maxChargeAge = 30 * 24 * time.Hour
	const maxChargeInterval = 5 * 24 * time.Hour
	if recent := in.recent; len(recent) > 0 && ref.Sub(recent[0].date) <= maxChargeAge {
		if len(recent) >= 2 && recent[0].date.Sub(recent[1].date) <= maxChargeInterval {
			totalCost := recent[0].cost + recent[1].cost
			totalKwh := recent[0].kwh + recent[1].kwh
			if totalKwh > 0 {
				rates.ElectricityPerKwh = totalCost.Float() / totalKwh
				rates.ElectricitySource = RateSourceRecentCharges
			}
		} else {
			rates.ElectricityPerKwh = recent[0].cost.Float() / recent[0].kwh
			rates.ElectricitySource = RateSourceRecentCharges
		}
	}

	// Fallback to all priced charges history if no recent charge was found
	if rates.ElectricitySource == RateSourceUnknown && in.pricedKwh > 0 {
		rates.ElectricityPerKwh = in.pricedCost.Float() / in.pricedKwh
		rates.ElectricitySource = RateSourceHistory
	}

	if in.mountedTireRate > 0 {
		rates.TiresPerKm = in.mountedTireRate
		rates.TiresSource = RateSourceMountedTires
	} else if in.totalDistance > 500 && in.totalTiresCost > 0 {
		rates.TiresPerKm = in.totalTiresCost.Float() / in.totalDistance
		rates.TiresSource = RateSourceHistory
	}

	if in.totalDistance > 500 && in.totalMaintCost > 0 {
		rates.MaintenancePerKm = in.totalMaintCost.Float() / in.totalDistance
		rates.MaintenanceSource = RateSourceHistory
	}

	annualInsurance := in.annualizedMaint
	if in.insuranceCost > annualInsurance {
		annualInsurance = in.insuranceCost
	}
	if annualInsurance > 0 {
		rates.DailyInsuranceCost = money.FromFloat(annualInsurance.Float() / 365.25)
		rates.MonthlyInsuranceCost = money.FromFloat(annualInsurance.Float() / 12.0)
	}

	// Services included in a running lease contract
	if ownership != nil && ownership.InLeasePhase(ref) {
		if ownership.LeaseIncludesMaintenance {
			rates.MaintenancePerKm, rates.MaintenanceSource = 0, RateSourceIncluded
		}
		if ownership.LeaseIncludesTires {
			rates.TiresPerKm, rates.TiresSource = 0, RateSourceIncluded
		}
	}

	if ownership != nil && ownership.InLeasePhase(ref) && ownership.LeaseIncludesInsurance {
		rates.InsuranceSource = InsuranceSourceIncluded
		rates.InsurancePerKm = 0
		rates.DailyInsuranceCost = 0
		rates.MonthlyInsuranceCost = 0
	} else {
		windowCost, windowKm := in.insuranceCost, in.windowKm
		switch {
		case in.windowStart != nil && windowCost > 0 && windowKm >= minInsuranceWindowKm:
			rates.InsurancePerKm = windowCost.Float() / windowKm
			rates.InsuranceSource = InsuranceSourceRecordedExpenses
			rates.InsuranceWindowCost = &windowCost
			rates.InsuranceWindowKm = &windowKm
		case (in.windowStart != nil && windowCost > 0) || annualInsurance > 0:
			rates.InsuranceSource = InsuranceSourceInsufficientKm
			if windowCost > 0 {
				rates.InsuranceWindowCost = &windowCost
				rates.InsuranceWindowKm = &windowKm
			}
		}
	}

	rates.ElectricityPerKwh = round3(rates.ElectricityPerKwh)
	rates.TiresPerKm = round3(rates.TiresPerKm)
	rates.MaintenancePerKm = round3(rates.MaintenancePerKm)
	rates.InsurancePerKm = round3(rates.InsurancePerKm)
	return rates
}
