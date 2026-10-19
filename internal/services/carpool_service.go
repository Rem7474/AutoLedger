package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

// carpoolStore is the narrow slice of *database.Repository that CarpoolService actually
// needs. Consumer-defined so tests can supply a fake without a real database (mirrors the
// syncStore interface in sync_service.go).
type carpoolStore interface {
	GetVehicleOwnership(ctx context.Context, vehicleID string) (*models.VehicleOwnership, error)
	GetDriveByID(ctx context.Context, driveID, vehicleID string) (*models.Drive, error)
	GetTripGroupDrives(ctx context.Context, vehicleID, tripGroupID string) ([]models.Drive, error)
	GetTollExpensesForDrives(ctx context.Context, vehicleID string, driveIDs []string) (map[string]money.Cents, error)
	GetCarpoolTrip(ctx context.Context, id, vehicleID string) (*models.CarpoolTripWithPassengers, error)
	UpdateCarpoolTrip(ctx context.Context, trip *models.CarpoolTrip, legs []models.CarpoolLeg, passengers []models.CarpoolPassenger) error
	ListCarpoolTrips(ctx context.Context, vehicleID string) ([]models.CarpoolTripWithPassengers, error)
}

type CarpoolService struct {
	pool *pgxpool.Pool
	repo carpoolStore
}

func NewCarpoolService(pool *pgxpool.Pool, repo carpoolStore) *CarpoolService {
	return &CarpoolService{
		pool: pool,
		repo: repo,
	}
}

// Rate sources exposed to the UI so that estimates are never mistaken for measured costs.
const (
	RateSourceRecentCharges = "RECENT_CHARGES"
	RateSourceHistory       = "HISTORY"
	RateSourceMountedTires  = "MOUNTED_TIRES"
	// RateSourceUnknown marks a rate with no history behind it: the component is not priced, never replaced by a
	// built-in amount (which would be in one currency whatever the vehicle's).
	RateSourceUnknown       = "UNKNOWN"
	RateSourceIncluded      = "INCLUDED_IN_LEASE"
	EnergySourceMeasured    = "MEASURED"
	EnergySourceConsumption = "CONSUMPTION"
	EnergySourceDefault     = "DEFAULT"
)

// Fallback consumption used only when a drive has neither measured energy nor a vehicle consumption.
const (
	defaultConsumptionKwh100 = 16.5
	// Minimum distance over the insurance window for a meaningful per-km share.
	minInsuranceWindowKm = 500.0
)

type UnitRates struct {
	ElectricityPerKwh float64
	ElectricitySource string
	TiresPerKm        float64
	TiresSource       string
	MaintenancePerKm  float64
	MaintenanceSource string
	// InsurancePerKm shares the insurance premiums paid over the last 12 months (or since the first premium)
	// across the kilometers actually driven over the same window.
	InsurancePerKm       float64
	InsuranceSource      string // RECORDED_EXPENSES | INCLUDED_IN_LEASE | INSUFFICIENT_DISTANCE | NONE
	InsuranceWindowCost  *money.Cents
	InsuranceWindowKm    *float64
	DailyInsuranceCost   money.Cents
	MonthlyInsuranceCost money.Cents
}

// DefaultUnitRates returns the rates of a vehicle without history: every component is unknown, not priced.
func DefaultUnitRates() *UnitRates {
	return &UnitRates{
		ElectricitySource: RateSourceUnknown,
		TiresSource:       RateSourceUnknown,
		MaintenanceSource: RateSourceUnknown,
		InsuranceSource:   InsuranceSourceNone,
	}
}

// HasUnknownRate reports whether a priced component of a drive has no rate behind it, so its cost is missing from the
// total rather than estimated.
func (r *UnitRates) HasUnknownRate() bool {
	return r.ElectricitySource == RateSourceUnknown || r.TiresSource == RateSourceUnknown || r.MaintenanceSource == RateSourceUnknown
}

// CostComponents holds the per-km/per-kWh cost components of a drive, excluding insurance and
// tolls (insurance is a fixed daily cost allocated separately via AllocateInsuranceCosts; tolls
// are direct expenses looked up per drive).
type CostComponents struct {
	ElectricityCost money.Cents
	TiresCost       money.Cents
	MaintenanceCost money.Cents
}

// ComputeCostComponents applies a vehicle's unit rates to a given distance/energy, shared by the
// drive list and the carpool cost estimate so the two never drift apart.
func ComputeCostComponents(rates *UnitRates, distanceKm, kwh float64) CostComponents {
	return CostComponents{
		ElectricityCost: money.FromFloat(kwh * rates.ElectricityPerKwh),
		TiresCost:       money.FromFloat(distanceKm * rates.TiresPerKm),
		MaintenanceCost: money.FromFloat(distanceKm * rates.MaintenancePerKm),
	}
}

// DriveCost is the cost of one drive as the drive list shows it: the vehicle's unit rates applied to the drive, its
// share of the insurance and the tolls allocated to it.
type DriveCost struct {
	ElectricityCost money.Cents
	ElectricityKwh  float64
	EnergySource    string
	TiresCost       money.Cents
	MaintenanceCost money.Cents
	InsuranceCost   money.Cents
	TollsCost       money.Cents
	TotalCost       money.Cents
	// HasEstimates is true when the energy or one of the rates relies on a default assumption.
	HasEstimates bool
}

// ComputeDriveCost is the single definition of a drive's cost, shared by the drive list and the per-tag totals of
// the TCO so a total by tag always adds up the drives the list shows for that tag.
func ComputeDriveCost(rates *UnitRates, d models.Drive, insurance, tolls money.Cents) DriveCost {
	kwh, source := DriveEnergyKwh(d.Distance(), d.EnergyConsumedKwh, d.ConsumptionKwh100km)
	components := ComputeCostComponents(rates, d.Distance(), kwh)
	return DriveCost{
		ElectricityCost: components.ElectricityCost,
		ElectricityKwh:  kwh,
		EnergySource:    source,
		TiresCost:       components.TiresCost,
		MaintenanceCost: components.MaintenanceCost,
		InsuranceCost:   insurance,
		TollsCost:       tolls,
		TotalCost:       components.ElectricityCost + components.TiresCost + components.MaintenanceCost + insurance + tolls,
		HasEstimates:    source == EnergySourceDefault || rates.HasUnknownRate(),
	}
}

// DriveEnergyKwh returns the energy of a drive and how it was obtained.
func DriveEnergyKwh(distanceKm float64, energyKwh, consumptionKwh100km *float64) (float64, string) {
	switch {
	case energyKwh != nil && *energyKwh > 0:
		return *energyKwh, EnergySourceMeasured
	case consumptionKwh100km != nil && *consumptionKwh100km > 0:
		return distanceKm * *consumptionKwh100km / 100.0, EnergySourceConsumption
	case distanceKm > 0:
		return distanceKm * defaultConsumptionKwh100 / 100.0, EnergySourceDefault
	}
	return 0, EnergySourceMeasured
}

// EstimateCosts calculates suggested cost components for a trip.
func (s *CarpoolService) EstimateCosts(ctx context.Context, vehicleID string, driveID, tripGroupID *string, driveIDs []string, manualDistanceKm float64) (*models.CarpoolCostEstimate, error) {
	var drives []models.Drive
	var err error
	switch {
	case driveID != nil && *driveID != "":
		d, err := s.repo.GetDriveByID(ctx, *driveID, vehicleID)
		if err != nil {
			return nil, err
		}
		drives = []models.Drive{*d}
	case tripGroupID != nil && *tripGroupID != "":
		if drives, err = s.repo.GetTripGroupDrives(ctx, vehicleID, *tripGroupID); err != nil {
			return nil, err
		}
		if len(drives) == 0 {
			return nil, database.ErrNotFound
		}
	case len(driveIDs) > 0:
		for _, did := range driveIDs {
			d, err := s.repo.GetDriveByID(ctx, did, vehicleID)
			if err != nil {
				return nil, err
			}
			drives = append(drives, *d)
		}
	}

	// One leg per drive, in chronological order
	sort.Slice(drives, func(i, j int) bool { return drives[i].StartTime.Before(drives[j].StartTime) })

	// Each drive is priced at the rates in force when it ended, as in the drive list; the rates of the estimate are
	// those of the last drive. A manual distance has no drive: it is priced at today's rates.
	var driveRates map[string]*UnitRates
	var rates *UnitRates
	if len(drives) > 0 {
		if driveRates, err = s.DriveRates(ctx, vehicleID, drives); err != nil {
			return nil, err
		}
		rates = driveRates[drives[len(drives)-1].ID]
	} else if rates, err = s.GetVehicleUnitRatesAt(ctx, vehicleID, nil); err != nil {
		return nil, err
	}
	legs := make([]models.CarpoolLeg, 0, len(drives))
	energySource := EnergySourceMeasured
	costLeg := func(rates *UnitRates, distance, kwh float64) models.CarpoolLeg {
		c := ComputeCostComponents(rates, distance, kwh)
		return models.CarpoolLeg{
			DistanceKm:      math.Round(distance*100) / 100,
			ElectricityCost: c.ElectricityCost,
			TiresCost:       c.TiresCost,
			MaintenanceCost: c.MaintenanceCost,
			InsuranceCost:   money.FromFloat(distance * rates.InsurancePerKm),
		}
	}

	if len(drives) > 0 {
		ids := make([]string, len(drives))
		for i, d := range drives {
			ids[i] = d.ID
		}

		insuranceCosts, err := s.AllocateInsuranceCosts(ctx, vehicleID, drives, driveRates)
		if err != nil {
			return nil, err
		}

		// Tolls allocated to each drive (a trip group toll is split by distance)
		tolls, err := s.repo.GetTollExpensesForDrives(ctx, vehicleID, ids)
		if err != nil {
			return nil, err
		}
		for _, d := range drives {
			kwh, src := DriveEnergyKwh(d.Distance(), d.EnergyConsumedKwh, d.ConsumptionKwh100km)
			if src != EnergySourceMeasured {
				energySource = src
			}
			leg := costLeg(driveRates[d.ID], d.Distance(), kwh)
			driveID := d.ID
			leg.DriveID = &driveID
			leg.StartLabel = placeLabel(d.StartAddress)
			leg.EndLabel = placeLabel(d.EndAddress)
			leg.TollsCost = tolls[d.ID]
			leg.InsuranceCost = insuranceCosts[d.ID]

			legs = append(legs, leg)
		}
	} else {
		kwh, src := DriveEnergyKwh(manualDistanceKm, nil, nil)
		energySource = src
		legs = append(legs, costLeg(rates, manualDistanceKm, kwh))
	}

	est := &models.CarpoolCostEstimate{
		ElectricityRatePerKwh: rates.ElectricityPerKwh,
		TiresRatePerKm:        rates.TiresPerKm,
		MaintenanceRatePerKm:  rates.MaintenancePerKm,
		InsuranceRatePerKm:    rates.InsurancePerKm,
		DailyInsuranceCost:    &rates.DailyInsuranceCost,
		MonthlyInsuranceCost:  &rates.MonthlyInsuranceCost,
		InsuranceSource:       rates.InsuranceSource,
		InsuranceWindowCost:   rates.InsuranceWindowCost,
		InsuranceWindowKm:     rates.InsuranceWindowKm,
		EnergySource:          energySource,
		ElectricityRateSource: rates.ElectricitySource,
		TiresRateSource:       rates.TiresSource,
		MaintenanceRateSource: rates.MaintenanceSource,
		Legs:                  legs,
	}
	if len(drives) > 0 {
		est.StartDate = &drives[0].StartTime
	}
	for i := range legs {
		legs[i].OrderIndex = i
		legs[i].TotalCost = legs[i].Total()
		est.DistanceKm += legs[i].DistanceKm
		est.ElectricityCost += legs[i].ElectricityCost
		est.TollsCost += legs[i].TollsCost
		est.TiresCost += legs[i].TiresCost
		est.MaintenanceCost += legs[i].MaintenanceCost
		est.InsuranceCost += legs[i].InsuranceCost
	}
	est.DistanceKm = round1(est.DistanceKm)
	if len(drives) > 0 && est.DistanceKm > 0 && (rates.MonthlyInsuranceCost > 0 || rates.DailyInsuranceCost > 0) {
		est.InsuranceRatePerKm = round3(est.InsuranceCost.Float() / est.DistanceKm)
	}
	est.TotalCost = est.ElectricityCost + est.TollsCost + est.TiresCost + est.MaintenanceCost + est.InsuranceCost
	return est, nil
}

// insuranceMonthlyCost is the fixed monthly insurance cost behind a set of rates, 0 when there is none to share
// (no premium, or insurance included in a lease).
func insuranceMonthlyCost(r *UnitRates) money.Cents {
	if r.InsuranceSource == InsuranceSourceIncluded {
		return 0
	}
	if r.MonthlyInsuranceCost > 0 {
		return r.MonthlyInsuranceCost
	}
	if r.DailyInsuranceCost > 0 {
		return money.FromFloat(r.DailyInsuranceCost.Float() * (365.25 / 12.0))
	}
	return 0
}

// AllocateInsuranceCosts shares the vehicle's insurance cost across drives, each with the rates in force when it
// ended (DriveRates). For past completed months, the fixed monthly insurance cost is allocated across all kilometers
// driven in that month, ensuring 100% of the month's premium is accounted for regardless of inactive days.
// For the current (ongoing) month, to prevent artificial cost spikes when monthly mileage is still accumulating, each
// drive uses the stable per-km reference rate (InsurancePerKm).
func (s *CarpoolService) AllocateInsuranceCosts(ctx context.Context, vehicleID string, drives []models.Drive, rates map[string]*UnitRates) (map[string]money.Cents, error) {
	costs := make(map[string]money.Cents, len(drives))
	if len(drives) == 0 {
		return costs, nil
	}

	currentMonthKey := time.Now().UTC().Format("2006-01")

	monthSet := make(map[string]struct{}, len(drives))
	drivesInMonthKm := make(map[string]float64, len(drives))
	for _, d := range drives {
		mStr := d.StartTime.UTC().Format("2006-01")
		monthSet[mStr] = struct{}{}
		drivesInMonthKm[mStr] += d.Distance()
	}

	months := make([]string, 0, len(monthSet))
	for mStr := range monthSet {
		if mStr < currentMonthKey {
			months = append(months, mStr)
		}
	}

	monthlyKmMap, err := s.getMonthlyDistances(ctx, vehicleID, months)
	if err != nil {
		return nil, err
	}

	for _, d := range drives {
		r := rates[d.ID]
		mStr := d.StartTime.UTC().Format("2006-01")
		monthlyCost := insuranceMonthlyCost(r)
		if monthlyCost <= 0 || mStr >= currentMonthKey {
			costs[d.ID] = money.FromFloat(d.Distance() * r.InsurancePerKm)
			continue
		}
		// Past completed month: consolidate on actual monthly kilometers
		totalMonthKm := math.Max(monthlyKmMap[mStr], drivesInMonthKm[mStr])
		if totalMonthKm >= 1.0 {
			costs[d.ID] = money.FromFloat(monthlyCost.Float() * (d.Distance() / totalMonthKm))
		} else {
			costs[d.ID] = money.FromFloat(d.Distance() * r.InsurancePerKm)
		}
	}
	return costs, nil
}

// getMonthlyDistances returns a map of month string "YYYY-MM" (UTC) to total km driven by the vehicle in that month.
func (s *CarpoolService) getMonthlyDistances(ctx context.Context, vehicleID string, months []string) (map[string]float64, error) {
	if len(months) == 0 {
		return make(map[string]float64), nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT to_char(start_time AT TIME ZONE 'UTC', 'YYYY-MM'),
		       COALESCE(SUM(distance_km), 0)
		FROM drives
		WHERE vehicle_id = $1
		  AND deleted_upstream_at IS NULL
		  AND to_char(start_time AT TIME ZONE 'UTC', 'YYYY-MM') = ANY($2)
		GROUP BY 1;
	`, vehicleID, months)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[string]float64, len(months))
	for rows.Next() {
		var mStr string
		var totalKm float64
		if err := rows.Scan(&mStr, &totalKm); err != nil {
			return nil, err
		}
		res[mStr] = totalKm
	}
	return res, rows.Err()
}

// placeLabel shortens a TeslaMate address to its first part (place or street) for stop names.
func placeLabel(address *string) *string {
	if address == nil {
		return nil
	}
	label := strings.TrimSpace(strings.Split(*address, ",")[0])
	if label == "" {
		return nil
	}
	if r := []rune(label); len(r) > 150 {
		label = string(r[:150])
	}
	return &label
}

// RecalculateTrip recalculates the costs of a carpool trip using latest vehicle unit rates and recorded tolls.
func (s *CarpoolService) RecalculateTrip(ctx context.Context, vehicleID string, tripID string) (*models.CarpoolTripWithPassengers, error) {
	trip, err := s.repo.GetCarpoolTrip(ctx, tripID, vehicleID)
	if err != nil {
		return nil, err
	}

	var driveIDs []string
	for _, l := range trip.Legs {
		if l.DriveID != nil && *l.DriveID != "" {
			driveIDs = append(driveIDs, *l.DriveID)
		}
	}

	var est *models.CarpoolCostEstimate
	if len(driveIDs) > 0 {
		est, err = s.EstimateCosts(ctx, vehicleID, trip.DriveID, trip.TripGroupID, driveIDs, 0)
		if err != nil {
			return nil, err
		}
	} else if trip.DistanceKm > 0 {
		est, err = s.EstimateCosts(ctx, vehicleID, nil, nil, nil, trip.DistanceKm)
		if err != nil {
			return nil, err
		}
	}

	if est != nil {
		if len(driveIDs) > 0 {
			estByDriveID := make(map[string]models.CarpoolLeg, len(est.Legs))
			for _, el := range est.Legs {
				if el.DriveID != nil {
					estByDriveID[*el.DriveID] = el
				}
			}
			for i := range trip.Legs {
				if trip.Legs[i].DriveID != nil {
					if el, ok := estByDriveID[*trip.Legs[i].DriveID]; ok {
						trip.Legs[i].ElectricityCost = el.ElectricityCost
						trip.Legs[i].TollsCost = el.TollsCost
						trip.Legs[i].TiresCost = el.TiresCost
						trip.Legs[i].MaintenanceCost = el.MaintenanceCost
						trip.Legs[i].InsuranceCost = el.InsuranceCost
					}
				}
			}
		} else if len(est.Legs) == 1 && len(trip.Legs) == 1 {
			trip.Legs[0].ElectricityCost = est.Legs[0].ElectricityCost
			trip.Legs[0].TiresCost = est.Legs[0].TiresCost
			trip.Legs[0].MaintenanceCost = est.Legs[0].MaintenanceCost
			trip.Legs[0].InsuranceCost = est.Legs[0].InsuranceCost
		}

		if est.StartDate != nil {
			trip.Date = *est.StartDate
		}
	}

	if err := s.repo.UpdateCarpoolTrip(ctx, &trip.CarpoolTrip, trip.Legs, trip.Passengers); err != nil {
		return nil, err
	}

	trip.DriverCostShare, trip.PassengersCostShare = AllocateCarpoolCosts(trip.Legs, trip.Passengers)
	return trip, nil
}

// RecalculateTrips recalculates multiple carpool trips for a vehicle. If tripIDs is empty, all trips are recalculated.
func (s *CarpoolService) RecalculateTrips(ctx context.Context, vehicleID string, tripIDs []string) ([]models.CarpoolTripWithPassengers, error) {
	if len(tripIDs) == 0 {
		allTrips, err := s.repo.ListCarpoolTrips(ctx, vehicleID)
		if err != nil {
			return nil, err
		}
		tripIDs = make([]string, len(allTrips))
		for i, t := range allTrips {
			tripIDs[i] = t.ID
		}
	}

	results := make([]models.CarpoolTripWithPassengers, 0, len(tripIDs))
	for _, id := range tripIDs {
		t, err := s.RecalculateTrip(ctx, vehicleID, id)
		if err != nil {
			return nil, fmt.Errorf("trip %s: %w", id, err)
		}
		results = append(results, *t)
	}
	return results, nil
}
